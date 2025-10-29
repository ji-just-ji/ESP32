# MQTT Backbone Service Implementation Plan

## Overview

This document describes the implementation of CQRS-based inference triggering for the MQTT Backbone Go service. This replaces the previous event-driven (sensor arrival) approach with a time-based polling system that uses statistical analysis to determine when to trigger ML inference.

---

## Current Status: ✅ COMPLETE

**Version:** v2.0 - CQRS-Based Inference Triggering
**Completed:** 2025-10-29

---

## Architecture Change: Event-Driven → CQRS Polling

### Before (v1.5): Event-Driven Architecture
```
MQTT Sensor Data → Sensor Service → Inference Service (in-memory state)
                                    ↓
                              Check thresholds immediately
                                    ↓
                              Trigger if threshold exceeded
                                    ↓
                              Inference Request → MQTT
```

**Limitations:**
- Coupled to sensor polling rates (variable)
- Simple threshold-based triggering
- No historical context
- Immediate processing may miss patterns

### After (v2.0): CQRS Polling Architecture
```
MQTT Sensor Data → Sensor Service → ClickHouse (write model)
                                          ↓
                                    [Time passes]
                                          ↓
Inference Service (polling ticker) → Query ClickHouse (read model)
                                          ↓
                              Compare current window to last inference
                                          ↓
                              Calculate Z-scores vs historical baseline
                                          ↓
                              Trigger if Z-score > threshold
                                          ↓
                              Inference Request → MQTT
```

**Benefits:**
- Decoupled from sensor polling rates
- Statistical rigor (Z-score based)
- Historical context (baseline from N days)
- Configurable time windows
- Predictable inference frequency

---

## Implementation Details

### 1. Database Schema Updates

**Added `sound_volume` field to `sensor_audio` table:**
```sql
CREATE TABLE sensor_audio (
    timestamp DateTime64(3),
    device_id String,
    sample_rate UInt32,
    duration Float64,
    format String,
    audio_hash String,
    sound_volume Float64,  -- NEW: Extracted dB value
    features String
) ENGINE = MergeTree()
ORDER BY (device_id, timestamp)
```

**Added `inference_history` table:**
```sql
CREATE TABLE inference_history (
    timestamp DateTime64(3),
    device_id String,
    trigger_reason String,
    temp_z_score Float64,
    humidity_z_score Float64,
    volume_z_score Float64
) ENGINE = MergeTree()
ORDER BY (device_id, timestamp)
```

**Purpose:**
- `sound_volume`: Store extracted volume for querying (CQRS read model)
- `inference_history`: Track when inferences were triggered and why

### 2. Configuration Parameters

**New CQRS Configuration (pkg/config/config.go):**
```go
// CQRS Inference Configuration
InferencePollingIntervalSeconds int     // How often to poll ClickHouse (default: 60)
InferenceDataWindowSeconds      int     // Time window for current data (default: 120)
InferenceHistoricalBaselineDays int     // Days for std dev calculation (default: 7)
InferenceZScoreThreshold        float64 // Trigger threshold (default: 1.5)
```

**Environment Variables:**
- `INFERENCE_POLLING_INTERVAL_SECONDS` - Polling frequency
- `INFERENCE_DATA_WINDOW_SECONDS` - Data aggregation window
- `INFERENCE_HISTORICAL_BASELINE_DAYS` - Historical baseline period
- `INFERENCE_Z_SCORE_THRESHOLD` - Z-score threshold for triggering

### 3. ClickHouse Query Methods

**New query methods in `internal/database/clickhouse.go`:**

1. **`SaveInferenceHistory`** - Record when inference was triggered
2. **`GetLastInferenceTimestamp`** - Get timestamp of last inference per device
3. **`GetCurrentWindowAggregates`** - Get mean(temp, humidity, volume) for current window
4. **`GetLastInferenceWindowAggregates`** - Get mean values from last inference window
5. **`GetHistoricalBaselineStats`** - Get std dev over historical period (N days)

**Query Pattern:**
```sql
-- Current window (e.g., last 2 minutes)
SELECT avg(value) FROM sensor_temperature
WHERE device_id = ? AND timestamp >= NOW() - INTERVAL 120 SECOND

-- Historical baseline (e.g., last 7 days)
SELECT stddevPop(value) FROM sensor_temperature
WHERE device_id = ? AND timestamp >= NOW() - INTERVAL 7 DAY
```

### 4. Inference Service Refactor

**Complete rewrite of `internal/services/inference_service.go`:**

**Key Changes:**
- Removed in-memory state (`DeviceInferenceState`)
- Removed event-driven methods (`UpdateTemperature`, `UpdateHumidity`, `UpdateVolume`)
- Removed threshold-based triggering
- Added polling loop with `time.Ticker`
- Added Z-score calculation logic
- Added device tracking (`RegisterDevice`)

**Core Algorithm:**
```go
For each device (every N seconds):
    1. Get current window aggregates (mean temp, humidity, volume)
    2. Get last inference timestamp
    3. Get last inference window aggregates
    4. Get historical baseline (std dev over M days)
    5. Calculate Z-scores:
       Z_temp = (current_temp - last_temp) / baseline_std_temp
       Z_humidity = (current_humidity - last_humidity) / baseline_std_humidity
       Z_volume = (current_volume - last_volume) / baseline_std_volume
    6. If ANY |Z-score| > threshold:
       - Trigger inference
       - Save to inference_history
       - Send InferenceRequest to MQTT
```

**Z-Score Calculation:**
```go
Z = (mean(current_window) - mean(last_inference_window)) / std_dev(historical_baseline)
```

**Trigger Conditions:**
- First inference: Always trigger (no baseline)
- Subsequent: Trigger if |Z_temp| > threshold OR |Z_humidity| > threshold OR |Z_volume| > threshold

### 5. Sensor Service Updates

**Changes to `internal/services/sensor_service.go`:**

1. **Audio Processing:**
   - Now saves `sound_volume` to database via updated `SaveAudio(recording, audioHash, volume)`
   - Removed direct forwarding to inference service

2. **Temperature/Humidity Processing:**
   - Removed forwarding to inference service
   - Only saves to database

3. **Device Registration:**
   - Added `inferenceService.RegisterDevice(deviceID)` call
   - Ensures inference service tracks all active devices

**Rationale:** Sensor service is now purely a write-side handler (CQRS write model)

### 6. Main Application Updates

**Changes to `cmd/server/main.go`:**

1. **Inference Service Initialization:**
```go
// v1.5 (old)
inferenceConfig := services.InferenceServiceConfig{
    TemperatureThreshold: 0.5,
    HumidityThreshold:    2.0,
    RateLimitDuration:    5 * time.Second,
}
inferenceService := services.NewInferenceService(inferenceConfig)

// v2.0 (new)
inferenceConfig := services.InferenceServiceConfig{
    PollingIntervalSeconds: cfg.InferencePollingIntervalSeconds,
    DataWindowSeconds:      cfg.InferenceDataWindowSeconds,
    HistoricalBaselineDays: cfg.InferenceHistoricalBaselineDays,
    ZScoreThreshold:        cfg.InferenceZScoreThreshold,
}
inferenceService := services.NewInferenceService(db, inferenceConfig)
```

2. **Version Update:** v1.5 → v2.0

---

## Testing & Validation

### Test Scenarios

1. **First Inference:**
   - Device sends first batch of sensor data
   - Inference triggers immediately (no baseline)
   - Verify `inference_history` record created

2. **Z-Score Triggering:**
   - Generate stable sensor data for baseline period
   - Introduce significant change (>1.5 std dev)
   - Verify inference triggers
   - Verify Z-scores logged correctly

3. **No Trigger:**
   - Generate stable sensor data
   - Make small changes (<1.5 std dev)
   - Verify no inference triggered

4. **Multi-Device:**
   - Test with 2-3 devices simultaneously
   - Verify independent tracking per device
   - Verify correct device isolation

5. **Configuration Changes:**
   - Test different polling intervals
   - Test different Z-score thresholds
   - Test different baseline periods

### Validation Queries

```sql
-- Check inference history
SELECT * FROM inference_history ORDER BY timestamp DESC LIMIT 10;

-- Check Z-scores over time
SELECT
    timestamp,
    device_id,
    temp_z_score,
    humidity_z_score,
    volume_z_score
FROM inference_history
WHERE device_id = 'device_01'
ORDER BY timestamp DESC;

-- Check sensor data availability
SELECT
    COUNT(*) as count,
    AVG(value) as avg_value,
    STDDEV(value) as std_dev
FROM sensor_temperature
WHERE device_id = 'device_01'
AND timestamp >= NOW() - INTERVAL 7 DAY;
```

---

## Performance Considerations

### Database Load

**Queries per polling interval (per device):**
- 1x `GetLastInferenceTimestamp`
- 1x `GetCurrentWindowAggregates` (3 sensor tables)
- 1x `GetLastInferenceWindowAggregates` (3 sensor tables)
- 1x `GetHistoricalBaselineStats` (3 sensor tables)

**Total:** ~8 queries per device per polling interval

**Optimization Strategies:**
1. Increase polling interval (60s → 120s)
2. Add query caching for historical baseline (slow-changing)
3. Use materialized views for aggregates
4. Batch device queries

### Memory Usage

- No in-memory state per device (unlike v1.5)
- Minimal: Only tracked device IDs map
- Scales linearly with device count

---

## Migration from v1.5 to v2.0

### Breaking Changes

1. **Configuration:**
   - Old: `TEMPERATURE_THRESHOLD`, `HUMIDITY_THRESHOLD`
   - New: `INFERENCE_POLLING_INTERVAL_SECONDS`, `INFERENCE_DATA_WINDOW_SECONDS`, etc.

2. **Inference Behavior:**
   - Old: Immediate triggering on sensor arrival
   - New: Time-based polling with statistical analysis

3. **Database Schema:**
   - New: `sound_volume` column in `sensor_audio`
   - New: `inference_history` table

### Migration Steps

1. **Update Configuration:**
   - Add new CQRS config parameters to `.env`
   - Can keep old params for backward compatibility (not used)

2. **Update Database:**
   - ClickHouse will auto-create new tables on startup
   - `sound_volume` field defaults to 0 for old records

3. **Deploy:**
   - Stop v1.5 service
   - Deploy v2.0 binary
   - Service will auto-initialize schema

4. **Warm-up Period:**
   - First N days will build historical baseline
   - Early inferences may trigger frequently (expected)

---

## Monitoring & Observability

### Key Metrics

1. **Inference Frequency:**
   - Count of inference triggers per device per hour
   - Query: `SELECT COUNT(*) FROM inference_history WHERE timestamp >= NOW() - INTERVAL 1 HOUR`

2. **Z-Score Distribution:**
   - Distribution of Z-scores across all triggers
   - Helps tune threshold

3. **Polling Performance:**
   - Query execution time for aggregates
   - Should be <100ms per device

4. **Device Coverage:**
   - Number of tracked devices
   - Log: "InferenceService: Polling N devices"

### Logging

**Inference Service Logs:**
```
InferenceService: Starting CQRS polling loop...
InferenceService: Polling every 1m0s, data window=2m0s, baseline=7 days, Z-threshold=1.50
InferenceService: Polling 3 devices
InferenceService: Device device_01 Z-scores: temp=0.23, humidity=0.45, volume=2.14
InferenceService: Triggering inference for device_01 (reason: volume_zscore)
```

---

## Future Enhancements

### Potential Improvements

1. **Adaptive Thresholds:**
   - Per-device Z-score thresholds
   - Time-of-day adjustments

2. **Composite Scoring:**
   - Weighted combination of Z-scores
   - Machine learning for trigger prediction

3. **Query Optimization:**
   - Materialized views for aggregates
   - Cached historical baselines (updated daily)

4. **Multi-Window Analysis:**
   - Compare multiple time windows
   - Trend detection (rising, falling, stable)

5. **Device Discovery:**
   - Auto-discover devices from `device_registry`
   - No manual registration needed

---

## References

### Related Documentation
- **Project Specification:** `/SPEC.md` (root)
- **Project Plan:** `/PLAN.md` (root)
- **Backend Specification:** `mqtt_backbone/SPEC.md`

### ClickHouse Documentation
- [Aggregate Functions](https://clickhouse.com/docs/en/sql-reference/aggregate-functions/)
- [DateTime Functions](https://clickhouse.com/docs/en/sql-reference/functions/date-time-functions/)
- [Window Functions](https://clickhouse.com/docs/en/sql-reference/window-functions/)

---

## Revision History

| Version | Date       | Changes                                    |
|---------|------------|-------------------------------------------|
| 2.0     | 2025-10-29 | Complete CQRS-based inference triggering  |
| 1.5     | 2025-10-27 | Channel-based architecture (deprecated)   |
| 1.0     | 2025-10-24 | Initial implementation (deprecated)       |

---

## Summary

The v2.0 CQRS-based inference triggering system provides:

✅ **Decoupling** - Inference independent of sensor polling rates
✅ **Statistical Rigor** - Z-score based triggering with historical context
✅ **Configurability** - All parameters externalized via environment variables
✅ **Scalability** - Database-driven approach scales with device count
✅ **Observability** - Full history of inference triggers and Z-scores
✅ **Maintainability** - Clear separation between write and read models

This architecture provides a robust foundation for ML-driven IoT decision-making with predictable performance characteristics and transparent triggering logic.
