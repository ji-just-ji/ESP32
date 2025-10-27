# Grafana Dashboard Queries Reference

This document contains all ClickHouse queries used in the Grafana dashboards for easy reference and customization.

---

## Table of Contents

1. [System Overview Queries](#system-overview-queries)
2. [Device Detail Queries](#device-detail-queries)
3. [ML Metrics Queries](#ml-metrics-queries)
4. [System Health Queries](#system-health-queries)
5. [Custom Query Examples](#custom-query-examples)

---

## System Overview Queries

### Active Devices Count
```sql
SELECT count(DISTINCT device_id) as active_devices
FROM device_registry
WHERE is_active = true
```

### Latest Temperature (All Devices)
```sql
SELECT device_id, argMax(value, timestamp) as temperature
FROM sensor_temperature
WHERE timestamp >= now() - INTERVAL 5 MINUTE
GROUP BY device_id
```

### Latest Humidity (All Devices)
```sql
SELECT device_id, argMax(value, timestamp) as humidity
FROM sensor_humidity
WHERE timestamp >= now() - INTERVAL 5 MINUTE
GROUP BY device_id
```

### Current Window Position (All Devices)
```sql
SELECT device_id, argMax(position, timestamp) as window_position
FROM window_actions
WHERE timestamp >= now() - INTERVAL 5 MINUTE
GROUP BY device_id
```

### Temperature Trends
```sql
SELECT timestamp, device_id, value
FROM sensor_temperature
WHERE $__timeFilter(timestamp)
ORDER BY timestamp
```

### Humidity Trends
```sql
SELECT timestamp, device_id, value
FROM sensor_humidity
WHERE $__timeFilter(timestamp)
ORDER BY timestamp
```

### Sound Volume Trends
```sql
SELECT timestamp, device_id, sound_volume
FROM window_actions
WHERE $__timeFilter(timestamp)
ORDER BY timestamp
```

### Recent Window Actions
```sql
SELECT timestamp, device_id, position, confidence,
       temperature, humidity, sound_volume
FROM window_actions
WHERE timestamp >= now() - INTERVAL 1 HOUR
ORDER BY timestamp DESC
LIMIT 20
```

### Message Throughput (Temperature)
```sql
SELECT toStartOfMinute(timestamp) as time,
       'temperature' as metric,
       count(*) as rate
FROM sensor_temperature
WHERE $__timeFilter(timestamp)
GROUP BY time
ORDER BY time
```

### Message Throughput (Humidity)
```sql
SELECT toStartOfMinute(timestamp) as time,
       'humidity' as metric,
       count(*) as rate
FROM sensor_humidity
WHERE $__timeFilter(timestamp)
GROUP BY time
ORDER BY time
```

### Message Throughput (Audio)
```sql
SELECT toStartOfMinute(timestamp) as time,
       'audio' as metric,
       count(*) as rate
FROM sensor_audio
WHERE $__timeFilter(timestamp)
GROUP BY time
ORDER BY time
```

---

## Device Detail Queries

### Device Information
```sql
SELECT device_id, name, location, registered_at, last_seen, is_active
FROM device_registry
WHERE device_id = '$device'
```

### Temperature History (Single Device)
```sql
SELECT timestamp, value as temperature
FROM sensor_temperature
WHERE device_id = '$device'
  AND $__timeFilter(timestamp)
ORDER BY timestamp
```

### Humidity History (Single Device)
```sql
SELECT timestamp, value as humidity
FROM sensor_humidity
WHERE device_id = '$device'
  AND $__timeFilter(timestamp)
ORDER BY timestamp
```

### Sound Volume History (Single Device)
```sql
SELECT timestamp, sound_volume
FROM window_actions
WHERE device_id = '$device'
  AND $__timeFilter(timestamp)
ORDER BY timestamp
```

### Window Position History (Single Device)
```sql
SELECT timestamp, position as window_position
FROM window_actions
WHERE device_id = '$device'
  AND $__timeFilter(timestamp)
ORDER BY timestamp
```

### ML Confidence Timeline
```sql
SELECT timestamp, confidence
FROM window_actions
WHERE device_id = '$device'
  AND $__timeFilter(timestamp)
ORDER BY timestamp
```

### Window Position Distribution
```sql
SELECT
  CASE
    WHEN position < 25 THEN 'Closed (0-25%)'
    WHEN position < 50 THEN 'Partially Open (25-50%)'
    WHEN position < 75 THEN 'Mostly Open (50-75%)'
    ELSE 'Fully Open (75-100%)'
  END as position_range,
  count(*) as count
FROM window_actions
WHERE device_id = '$device'
  AND $__timeFilter(timestamp)
GROUP BY position_range
ORDER BY position_range
```

### Recent Predictions (Detailed)
```sql
SELECT timestamp, position, confidence,
       temperature, humidity, sound_volume
FROM window_actions
WHERE device_id = '$device'
  AND $__timeFilter(timestamp)
ORDER BY timestamp DESC
LIMIT 50
```

---

## ML Metrics Queries

### Prediction Frequency by Device
```sql
SELECT device_id, count(*) as prediction_count
FROM window_actions
WHERE $__timeFilter(timestamp)
GROUP BY device_id
ORDER BY prediction_count DESC
```

### Confidence Score Distribution
```sql
SELECT
  CASE
    WHEN confidence < 0.5 THEN '0-50%'
    WHEN confidence < 0.7 THEN '50-70%'
    WHEN confidence < 0.85 THEN '70-85%'
    WHEN confidence < 0.95 THEN '85-95%'
    ELSE '95-100%'
  END as confidence_range,
  count(*) as count
FROM window_actions
WHERE $__timeFilter(timestamp)
GROUP BY confidence_range
ORDER BY confidence_range
```

### Prediction Rate Over Time
```sql
SELECT toStartOfMinute(timestamp) as time,
       device_id,
       count(*) as predictions_per_minute
FROM window_actions
WHERE $__timeFilter(timestamp)
GROUP BY time, device_id
ORDER BY time
```

### Confidence Score Over Time
```sql
SELECT timestamp, device_id, confidence
FROM window_actions
WHERE $__timeFilter(timestamp)
ORDER BY timestamp
```

### Temperature vs Window Position (Correlation)
```sql
SELECT temperature, position
FROM window_actions
WHERE $__timeFilter(timestamp)
```

### Humidity vs Window Position (Correlation)
```sql
SELECT humidity, position
FROM window_actions
WHERE $__timeFilter(timestamp)
```

### Sound Volume vs Window Position (Correlation)
```sql
SELECT sound_volume, position
FROM window_actions
WHERE $__timeFilter(timestamp)
```

### Average Confidence Score
```sql
SELECT avg(confidence) as avg_confidence
FROM window_actions
WHERE $__timeFilter(timestamp)
```

### Total Predictions
```sql
SELECT count(*) as total_predictions
FROM window_actions
WHERE $__timeFilter(timestamp)
```

### Minimum Confidence
```sql
SELECT min(confidence) as min_confidence
FROM window_actions
WHERE $__timeFilter(timestamp)
```

### Active Devices (Making Predictions)
```sql
SELECT count(DISTINCT device_id) as active_devices
FROM window_actions
WHERE $__timeFilter(timestamp)
```

---

## System Health Queries

### Device Status
```sql
SELECT device_id, name, location, is_active, last_seen,
  CASE
    WHEN last_seen > now() - INTERVAL 5 MINUTE THEN 'Healthy'
    WHEN last_seen > now() - INTERVAL 15 MINUTE THEN 'Warning'
    ELSE 'Critical'
  END as status
FROM device_registry
ORDER BY is_active DESC, last_seen DESC
```

### Device Status Distribution
```sql
SELECT
  CASE
    WHEN is_active = true THEN 'Active'
    ELSE 'Inactive'
  END as status,
  count(*) as count
FROM device_registry
GROUP BY status
```

### Healthy Devices (Last 5 min)
```sql
SELECT count(*) as online_devices
FROM device_registry
WHERE is_active = true
  AND last_seen > now() - INTERVAL 5 MINUTE
```

### MQTT Message Rates (Temperature)
```sql
SELECT toStartOfMinute(timestamp) as time,
       'temperature' as metric,
       count(*) / 60 as rate
FROM sensor_temperature
WHERE $__timeFilter(timestamp)
GROUP BY time
ORDER BY time
```

### MQTT Message Rates (Humidity)
```sql
SELECT toStartOfMinute(timestamp) as time,
       'humidity' as metric,
       count(*) / 60 as rate
FROM sensor_humidity
WHERE $__timeFilter(timestamp)
GROUP BY time
ORDER BY time
```

### MQTT Message Rates (Audio)
```sql
SELECT toStartOfMinute(timestamp) as time,
       'audio' as metric,
       count(*) / 60 as rate
FROM sensor_audio
WHERE $__timeFilter(timestamp)
GROUP BY time
ORDER BY time
```

### MQTT Message Rates (Window Actions)
```sql
SELECT toStartOfMinute(timestamp) as time,
       'window_actions' as metric,
       count(*) / 60 as rate
FROM window_actions
WHERE $__timeFilter(timestamp)
GROUP BY time
ORDER BY time
```

### Data Ingestion Rate (Temperature)
```sql
SELECT toStartOfHour(timestamp) as time,
       'temperature' as table_name,
       count(*) as row_count
FROM sensor_temperature
WHERE $__timeFilter(timestamp)
GROUP BY time
ORDER BY time
```

### Data Ingestion Rate (Humidity)
```sql
SELECT toStartOfHour(timestamp) as time,
       'humidity' as table_name,
       count(*) as row_count
FROM sensor_humidity
WHERE $__timeFilter(timestamp)
GROUP BY time
ORDER BY time
```

### Data Ingestion Rate (Audio)
```sql
SELECT toStartOfHour(timestamp) as time,
       'audio' as table_name,
       count(*) as row_count
FROM sensor_audio
WHERE $__timeFilter(timestamp)
GROUP BY time
ORDER BY time
```

### Data Ingestion Rate (Window Actions)
```sql
SELECT toStartOfHour(timestamp) as time,
       'window_actions' as table_name,
       count(*) as row_count
FROM window_actions
WHERE $__timeFilter(timestamp)
GROUP BY time
ORDER BY time
```

### Record Counts by Table
```sql
-- Temperature
SELECT count(*) FROM sensor_temperature WHERE $__timeFilter(timestamp)

-- Humidity
SELECT count(*) FROM sensor_humidity WHERE $__timeFilter(timestamp)

-- Audio
SELECT count(*) FROM sensor_audio WHERE $__timeFilter(timestamp)

-- Window Actions
SELECT count(*) FROM window_actions WHERE $__timeFilter(timestamp)

-- Devices
SELECT count(*) FROM device_registry

-- ML Predictions (if available)
SELECT count(*) FROM ml_predictions WHERE $__timeFilter(timestamp)
```

### Device Last-Seen Status (Detailed)
```sql
SELECT
  device_id,
  name,
  location,
  last_seen,
  round(dateDiff('second', last_seen, now()) / 60, 2) as minutes_since_last_seen,
  is_active
FROM device_registry
ORDER BY last_seen DESC
```

---

## Custom Query Examples

### Average Temperature by Hour
```sql
SELECT
  toStartOfHour(timestamp) as hour,
  device_id,
  avg(value) as avg_temperature,
  min(value) as min_temperature,
  max(value) as max_temperature
FROM sensor_temperature
WHERE $__timeFilter(timestamp)
GROUP BY hour, device_id
ORDER BY hour, device_id
```

### Sensor Reading Gaps (Missing Data Detection)
```sql
SELECT
  device_id,
  max(timestamp) as last_reading,
  dateDiff('minute', max(timestamp), now()) as minutes_since_last_reading
FROM sensor_temperature
GROUP BY device_id
HAVING minutes_since_last_reading > 10
ORDER BY minutes_since_last_reading DESC
```

### Window Position Changes Per Day
```sql
SELECT
  toDate(timestamp) as date,
  device_id,
  count(*) as position_changes,
  avg(confidence) as avg_confidence
FROM window_actions
WHERE $__timeFilter(timestamp)
GROUP BY date, device_id
ORDER BY date DESC, device_id
```

### Sound Volume Statistics
```sql
SELECT
  device_id,
  avg(sound_volume) as avg_volume,
  min(sound_volume) as min_volume,
  max(sound_volume) as max_volume,
  stddevPop(sound_volume) as std_dev
FROM window_actions
WHERE $__timeFilter(timestamp)
GROUP BY device_id
ORDER BY device_id
```

### Confidence Score Trends by Device
```sql
SELECT
  toStartOfDay(timestamp) as day,
  device_id,
  avg(confidence) as avg_confidence,
  min(confidence) as min_confidence,
  count(*) as predictions
FROM window_actions
WHERE $__timeFilter(timestamp)
GROUP BY day, device_id
ORDER BY day DESC, device_id
```

### Hourly Data Completeness Check
```sql
SELECT
  toStartOfHour(timestamp) as hour,
  count(DISTINCT device_id) as active_devices,
  count(*) as total_readings
FROM sensor_temperature
WHERE $__timeFilter(timestamp)
GROUP BY hour
ORDER BY hour DESC
```

### ML Model Performance Over Time
```sql
SELECT
  toStartOfDay(timestamp) as day,
  count(*) as total_predictions,
  avg(confidence) as avg_confidence,
  countIf(confidence >= 0.8) / count(*) as high_confidence_ratio
FROM window_actions
WHERE $__timeFilter(timestamp)
GROUP BY day
ORDER BY day DESC
```

---

## Query Optimization Tips

### Use Time Filters
Always include `$__timeFilter(timestamp)` or explicit time ranges:
```sql
WHERE timestamp >= now() - INTERVAL 1 HOUR
```

### Limit Results
Use `LIMIT` for tables and large result sets:
```sql
ORDER BY timestamp DESC LIMIT 100
```

### Aggregate When Possible
Use aggregation functions to reduce data volume:
```sql
SELECT toStartOfMinute(timestamp) as time, avg(value) as avg_temp
FROM sensor_temperature
WHERE $__timeFilter(timestamp)
GROUP BY time
```

### Index-Friendly Queries
ClickHouse tables are ordered by `(device_id, timestamp)`:
- Filter by device_id first when possible
- Always include timestamp filters
- Use `ORDER BY` matching the table's primary key

---

## Grafana Variables

### $device (Device Selector)
Populated from:
```sql
SELECT DISTINCT device_id
FROM device_registry
ORDER BY device_id
```

Used in Device Detail dashboard for per-device filtering.

### $__timeFilter(column)
Grafana built-in variable that converts dashboard time range to SQL:
```sql
-- Expands to something like:
timestamp >= toDateTime('2025-10-27 00:00:00')
  AND timestamp <= toDateTime('2025-10-27 23:59:59')
```

---

## Testing Queries

You can test these queries directly in ClickHouse:

```bash
# Access ClickHouse CLI
docker exec -it iot-clickhouse clickhouse-client

# Switch to iot database
USE iot;

# Run any query (replace $__timeFilter with actual time range)
SELECT timestamp, device_id, value
FROM sensor_temperature
WHERE timestamp >= now() - INTERVAL 1 HOUR
ORDER BY timestamp
LIMIT 10;
```

---

**Reference Version:** 1.0
**Last Updated:** 2025-10-27
**Phase:** 5 - Grafana Dashboards
