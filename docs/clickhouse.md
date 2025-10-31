# ClickHouse Schema Deep Dive

Based on the schema in `mqtt_backbone/internal/database/schema.go` and the specification in `mqtt_backbone/SPEC.md`, this document provides a detailed explanation of the ClickHouse table design.

---

## **Group 1: MergeTree Engine (Time-Series Tables)**

These 6 tables use the standard **MergeTree** engine for append-only time-series data:

### **1. sensor_temperature** (schema.go:7-15)
```sql
CREATE TABLE IF NOT EXISTS sensor_temperature (
    timestamp DateTime64(3),
    device_id String,
    value Float64
) ENGINE = MergeTree()
ORDER BY (device_id, timestamp)
PARTITION BY toYYYYMM(timestamp)
```

**Why MergeTree?**
- **Append-only workload**: Temperature readings are immutable once recorded
- **High write throughput**: ESP32 devices continuously stream temperature data
- **No updates/deletes**: Historical sensor data is never modified
- **Efficient compression**: Time-series data compresses extremely well in columnar format

**Why ORDER BY (device_id, timestamp)?**
- **Query pattern optimization**: Most queries filter by specific device and time range
- **Data locality**: All readings from the same device are stored together physically
- **Range scan efficiency**: Sequential reads for time-range queries per device
- **Example query**: `SELECT * FROM sensor_temperature WHERE device_id = 'sensor-001' AND timestamp > now() - INTERVAL 1 HOUR`

**Why PARTITION BY toYYYYMM(timestamp)?**
- **Data lifecycle management**: Easy to drop old partitions (e.g., data older than 12 months)
- **Query performance**: ClickHouse can skip entire partitions when time filters don't match
- **Parallel processing**: Each partition can be processed independently
- **Storage optimization**: Older partitions can be moved to cold storage

---

### **2. sensor_humidity** (schema.go:17-26)
```sql
CREATE TABLE IF NOT EXISTS sensor_humidity (
    timestamp DateTime64(3),
    device_id String,
    value Float64
) ENGINE = MergeTree()
ORDER BY (device_id, timestamp)
PARTITION BY toYYYYMM(timestamp)
```

**Design rationale**: Identical to `sensor_temperature` because:
- Same append-only access pattern
- Same query patterns (device + time range)
- Same data lifecycle requirements
- Separate table allows independent schema evolution and optimized compression per sensor type

---

### **3. sensor_audio** (schema.go:28-42)
```sql
CREATE TABLE IF NOT EXISTS sensor_audio (
    timestamp DateTime64(3),
    device_id String,
    sample_rate UInt32,
    duration Float64,
    format String,
    audio_hash String,
    sound_volume Float64,
    features String
) ENGINE = MergeTree()
ORDER BY (device_id, timestamp)
PARTITION BY toYYYYMM(timestamp)
```

**Why MergeTree?**
- **Metadata storage**: Stores audio metadata (sample_rate, duration, format, hash) but **NOT raw audio bytes**
- **High cardinality**: `audio_hash` provides unique identification without storing raw data
- **JSON features**: `features` column stores extracted audio features as JSON string

**Key design choice from spec**:
- Audio data is **processed in real-time** to extract sound volume (dB)
- Only metadata is persisted to ClickHouse
- This dramatically reduces storage requirements while preserving analytical value

**Why ORDER BY (device_id, timestamp)?**
- Same query pattern as other sensors: "Show me audio events for device X in time range Y"
- Enables correlation with temperature/humidity data

---

### **4. window_actions** (schema.go:44-57)
```sql
CREATE TABLE IF NOT EXISTS window_actions (
    timestamp DateTime64(3),
    device_id String,
    position Float64,
    confidence Float64,
    temperature Float64,
    humidity Float64,
    sound_volume Float64
) ENGINE = MergeTree()
ORDER BY (device_id, timestamp)
PARTITION BY toYYYYMM(timestamp)
```

**Why MergeTree?**
- **Action log**: Records every ML-driven window control decision
- **Immutable audit trail**: Historical actions should never be modified
- **Analytical queries**: Used for ML model evaluation and dashboard visualization

**Unique columns**:
- `position`: Continuous control (0-100%) instead of binary open/closed
- `confidence`: ML model's confidence score (0-1)
- `temperature`, `humidity`, `sound_volume`: **Input features are denormalized** for easy analysis

**Why denormalize input features?**
- **Self-contained analysis**: No need to JOIN with sensor tables to analyze model behavior
- **Performance**: Grafana dashboards can query this single table
- **Traceability**: Exact sensor values that triggered each action are preserved

---

### **5. ml_predictions** (schema.go:73-85)
```sql
CREATE TABLE IF NOT EXISTS ml_predictions (
    timestamp DateTime64(3),
    device_id String,
    prediction Float64,
    confidence Float64,
    inference_time_ms Float64,
    model_version String
) ENGINE = MergeTree()
ORDER BY (device_id, timestamp)
PARTITION BY toYYYYMM(timestamp)
```

**Why MergeTree?**
- **Performance monitoring**: Tracks ML service health and latency
- **Model versioning**: `model_version` enables A/B testing and rollback analysis
- **Append-only**: Prediction logs are immutable

**Why ORDER BY (device_id, timestamp)?**
- **Per-device monitoring**: "Show me prediction latency for sensor-001 over the last week"
- **Time-series analysis**: Detect performance degradation over time
- **Comparison queries**: Compare prediction distribution across devices

**Key analytical queries this enables**:
- Model performance per device
- Inference latency trends
- Confidence score distribution
- Model version comparison

---

### **6. inference_history** (schema.go:87-99)
```sql
CREATE TABLE IF NOT EXISTS inference_history (
    timestamp DateTime64(3),
    device_id String,
    trigger_reason String,
    temp_z_score Float64,
    humidity_z_score Float64,
    volume_z_score Float64
) ENGINE = MergeTree()
ORDER BY (device_id, timestamp)
PARTITION BY toYYYYMM(timestamp)
```

**Why this table exists**:
- **Trigger audit log**: Records *why* each inference was triggered
- **Statistical analysis**: Stores Z-scores for temperature, humidity, and volume changes
- **Rate limiting verification**: Helps debug if inference triggering logic is working correctly

**Unique columns**:
- `trigger_reason`: Explains what sensor change triggered inference (e.g., "volume_change", "temp_threshold")
- `temp_z_score`, `humidity_z_score`, `volume_z_score`: Statistical significance of changes

**Why MergeTree?**
- Same reasoning as other time-series tables
- Append-only audit log
- Query pattern: "Show me what triggered inferences for device X"

---

## **Group 2: ReplacingMergeTree Engine (Mutable Registry)**

### **7. device_registry** (schema.go:59-71)
```sql
CREATE TABLE IF NOT EXISTS device_registry (
    device_id String,
    name String,
    location String,
    registered_at DateTime64(3),
    last_seen DateTime64(3),
    is_active Bool,
    config String
) ENGINE = ReplacingMergeTree(last_seen)
ORDER BY device_id
```

**Why ReplacingMergeTree instead of MergeTree?**
- **Mutable state**: Device information changes over time (last_seen, is_active, config)
- **Latest state queries**: "What is the current state of device-001?" requires getting the most recent record
- **Automatic deduplication**: Multiple updates to the same device_id are automatically merged

**How ReplacingMergeTree works**:
1. When a device updates, a **new row** is inserted with the same `device_id`
2. During merges, ClickHouse keeps the row with the highest `last_seen` value
3. The version column `(last_seen)` determines which record is "latest"

**Why ORDER BY device_id (not device_id, last_seen)?**
- **Single primary key**: Device registry is keyed by unique device_id
- **Point lookups**: Queries are "SELECT * FROM device_registry WHERE device_id = ?"
- **No time-range queries**: Unlike sensor tables, we only care about current device state

**Why PARTITION BY is omitted**:
- **Small dataset**: Only 2-10 devices initially
- **No time-based retention**: Device records are kept indefinitely
- **Single partition is optimal**: Partitioning overhead would hurt more than help

**Important caveat**:
- `ReplacingMergeTree` deduplication happens during **background merges**, not immediately
- Queries should use `FINAL` modifier to get deduplicated results: `SELECT * FROM device_registry FINAL WHERE device_id = ?`
- Or use `GROUP BY device_id` with `argMax(last_seen)` for explicit latest-record selection

---

## **Design Patterns & Rationale Summary**

### **1. Engine Selection Philosophy**

| Pattern | Engine | Tables |
|---------|--------|--------|
| Immutable time-series | `MergeTree()` | sensor_temperature, sensor_humidity, sensor_audio, window_actions, ml_predictions, inference_history |
| Mutable registry (latest-state) | `ReplacingMergeTree(version_col)` | device_registry |

### **2. ORDER BY Strategy**

All time-series tables use `(device_id, timestamp)`:
- **Primary access pattern**: Filter by device, then by time range
- **Data locality**: Device data clustered together for efficient scans
- **Compression**: Similar values from the same device compress better
- **Multi-device queries**: Still efficient with ClickHouse's parallel processing

Device registry uses `device_id` only:
- **Point lookups**: No time component in queries
- **Unique key**: One current record per device

### **3. PARTITION BY Strategy**

All time-series tables use `toYYYYMM(timestamp)`:
- **Monthly granularity**: Good balance between number of partitions and partition size
- **Data retention**: Easy to implement "drop data older than N months"
- **Query optimization**: Partition pruning on time-range queries
- **Operational efficiency**: Manageable number of partitions (12 per year)

Why monthly instead of daily?
- **IoT use case**: 2-10 devices with moderate data volume
- **Avoids partition explosion**: Daily partitioning would create 365+ partitions/year
- **Query patterns**: Most analytics are monthly/weekly aggregations (Grafana dashboards)

### **4. Key Design Choices from Spec**

**Audio processing**:
- Raw audio is **processed but not stored** ’ dramatically reduces storage
- Sound volume (dB) extracted in real-time ’ becomes input feature for ML
- Only metadata stored in `sensor_audio` table

**Denormalization for analytics**:
- `window_actions` duplicates sensor values ’ eliminates JOINs in Grafana
- Trade-off: Increased storage for much better query performance

**Statistical tracking**:
- `inference_history` table tracks Z-scores ’ enables debugging trigger logic
- Separate from `ml_predictions` ’ separates concerns (triggering vs. model output)

---

## **Performance Characteristics**

**Write performance** (all tables):
- ClickHouse buffers writes in memory
- Asynchronous background merges
- Columnar storage ’ excellent compression ratios (10:1 typical for sensor data)

**Read performance**:
- Time-series tables: O(log N) device lookup + sequential time scan
- Partition pruning: Skips entire months when not in query range
- Parallel processing: Each device can be processed independently
- Device registry: O(log N) point lookup, O(1) with `device_id` index

**Storage optimization**:
- Delta encoding: Sequential timestamps compress extremely well
- LZ4 compression: Default, very fast with good ratios
- Monthly partitions: Old data can be moved to cold storage
- No raw audio: Metadata-only approach saves TB of storage

---

## **Summary**

This schema design is **optimized for IoT time-series workloads** with:

 High-frequency sensor writes
 Device-centric time-range queries
 Analytical aggregations (Grafana)
 Minimal storage footprint
 Simple data lifecycle management

The combination of MergeTree for immutable time-series data and ReplacingMergeTree for mutable device state provides an efficient, scalable foundation for the IoT backend system.
