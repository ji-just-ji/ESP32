# ESP32 IoT System - Grafana Dashboards

This directory contains Grafana dashboard configurations and provisioning files for real-time monitoring and analytics of the ESP32 IoT system.

## 📁 Directory Structure

```
grafana/
├── README.md                          # This file
├── provisioning/
│   ├── datasources/
│   │   └── clickhouse.yml            # ClickHouse datasource configuration
│   └── dashboards/
│       └── default.yml                # Dashboard provisioning configuration
└── dashboards/
    ├── system-overview.json           # Dashboard 1: System Overview
    ├── device-detail.json             # Dashboard 2: Device Detail
    ├── ml-metrics.json                # Dashboard 3: ML Metrics
    └── system-health.json             # Dashboard 4: System Health
```

## 🚀 Quick Start

### 1. Start the Stack

From the `mqtt_backbone` directory, start all services including Grafana:

```bash
cd mqtt_backbone
docker-compose up -d
```

This will start:
- ClickHouse (port 9000, 8123)
- Mosquitto MQTT Broker (port 1883)
- Grafana (port 3000)

### 2. Access Grafana

Open your browser and navigate to:

```
http://localhost:3000
```

**Default Credentials:**
- Username: `admin`
- Password: `admin`

You will be prompted to change the password on first login.

### 3. View Dashboards

Once logged in, navigate to **Dashboards** → **ESP32 IoT System** folder to see all available dashboards.

## 📊 Available Dashboards

### 1. System Overview

**Refresh Rate:** 5 seconds
**Default Time Range:** Last 6 hours

**Panels:**
- Active Devices Counter
- Latest Temperature/Humidity/Window Position (all devices)
- Temperature, Humidity Trends (time-series)
- Sound Volume Trends (time-series)
- Recent Window Actions (table)
- Message Throughput (messages/minute)

**Use Case:** Real-time monitoring of the entire system at a glance.

---

### 2. Device Detail

**Refresh Rate:** 10 seconds
**Default Time Range:** Last 24 hours

**Features:**
- **Device Selector:** Dropdown to select specific device
- Device Information table
- Temperature History (time-series)
- Humidity History (time-series)
- Sound Volume History (time-series)
- Window Position History (time-series)
- ML Prediction Confidence Timeline
- Window Position Distribution (pie chart)
- Recent Predictions (detailed table)

**Use Case:** Deep-dive analytics for a specific ESP32 device.

---

### 3. ML Metrics

**Refresh Rate:** 30 seconds
**Default Time Range:** Last 24 hours

**Panels:**
- Prediction Frequency by Device (bar chart)
- Confidence Score Distribution (histogram)
- Prediction Rate Over Time (predictions/min)
- Confidence Score Over Time
- Feature Correlations:
  - Temperature vs Window Position
  - Humidity vs Window Position
  - Sound Volume vs Window Position
- Average Confidence Score
- Total Predictions
- Minimum Confidence
- Active Devices (making predictions)

**Use Case:** Monitor ML model performance, confidence, and feature relationships.

---

### 4. System Health

**Refresh Rate:** 10 seconds
**Default Time Range:** Last 6 hours

**Panels:**
- Device Status Table (online/offline)
- Device Status Distribution (pie chart)
- Healthy Devices Counter (last 5 min)
- MQTT Message Rates (messages/sec)
- Data Ingestion Rate (rows/hour)
- Record Counts (Temperature, Humidity, Audio, Window Actions)
- Total Devices Registered
- ML Predictions Count
- Device Last-Seen Status (detailed table with color coding)

**Use Case:** System health monitoring, identifying offline devices, message throughput.

---

## 🔧 Configuration

### ClickHouse Datasource

The ClickHouse datasource is automatically provisioned with the following settings:

- **Name:** ClickHouse
- **Type:** grafana-clickhouse-datasource
- **URL:** http://clickhouse:9000
- **Database:** iot
- **Protocol:** native
- **Username:** default
- **Password:** (empty)

**Configuration File:** `provisioning/datasources/clickhouse.yml`

### Dashboard Provisioning

Dashboards are automatically loaded from the `dashboards/` directory on Grafana startup.

**Configuration File:** `provisioning/dashboards/default.yml`

**Settings:**
- Organization: Default (orgId: 1)
- Folder: "ESP32 IoT System"
- Update Interval: 10 seconds
- Allow UI Updates: Yes (editable)

---

## 📈 ClickHouse Schema

The dashboards query the following ClickHouse tables:

| Table Name | Description | Key Columns |
|------------|-------------|-------------|
| `sensor_temperature` | Temperature readings | timestamp, device_id, value |
| `sensor_humidity` | Humidity readings | timestamp, device_id, value |
| `sensor_audio` | Audio metadata | timestamp, device_id, sample_rate, duration, features |
| `window_actions` | Window control actions | timestamp, device_id, position, confidence, temperature, humidity, sound_volume |
| `device_registry` | Device metadata | device_id, name, location, last_seen, is_active |
| `ml_predictions` | ML performance metrics | timestamp, device_id, prediction, confidence, inference_time_ms |

**Note:** Sound volume data is extracted from audio and stored in `window_actions.sound_volume` field (in dB).

---

## 🎨 Customization

### Editing Dashboards

1. Open Grafana at http://localhost:3000
2. Navigate to the dashboard you want to edit
3. Click the **gear icon** (⚙️) in the top right → **Settings**
4. Make your changes
5. Click **Save Dashboard**

Changes are persisted in the Grafana database (Docker volume).

### Exporting Dashboards

To export a dashboard as JSON:

1. Open the dashboard
2. Click the **share icon** (📤) → **Export**
3. Click **Save to file**
4. Replace the corresponding JSON file in `grafana/dashboards/`

### Adding New Panels

1. Open a dashboard
2. Click **Add** → **Visualization**
3. Select **ClickHouse** as the datasource
4. Write your SQL query
5. Configure visualization options
6. Click **Apply**

---

## 🔍 Example Queries

### Latest Temperature by Device

```sql
SELECT device_id, argMax(value, timestamp) as temperature
FROM sensor_temperature
WHERE timestamp >= now() - INTERVAL 5 MINUTE
GROUP BY device_id
```

### Sound Volume Over Time

```sql
SELECT timestamp, device_id, sound_volume
FROM window_actions
WHERE $__timeFilter(timestamp)
ORDER BY timestamp
```

### Device Last Seen

```sql
SELECT device_id, name, location, last_seen,
  round(dateDiff('second', last_seen, now()) / 60, 2) as minutes_since_last_seen
FROM device_registry
ORDER BY last_seen DESC
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

---

## 🐛 Troubleshooting

### Dashboards Not Loading

1. **Check Grafana logs:**
   ```bash
   docker logs iot-grafana
   ```

2. **Verify ClickHouse datasource:**
   - Navigate to **Configuration** → **Data Sources**
   - Click on **ClickHouse**
   - Click **Test** to verify connection

3. **Check ClickHouse is running:**
   ```bash
   docker ps | grep clickhouse
   docker logs iot-clickhouse
   ```

### No Data in Dashboards

1. **Verify data exists in ClickHouse:**
   ```bash
   docker exec -it iot-clickhouse clickhouse-client
   ```

   Then run:
   ```sql
   USE iot;
   SHOW TABLES;
   SELECT count(*) FROM sensor_temperature;
   SELECT count(*) FROM device_registry;
   ```

2. **Check time range:** Ensure the dashboard time range includes periods when data was collected.

3. **Verify MQTT backend is running:**
   ```bash
   docker ps | grep mqtt_backbone
   ```

### ClickHouse Plugin Not Installed

If you see "Plugin not found" errors:

1. **Check plugin installation:**
   ```bash
   docker exec iot-grafana grafana-cli plugins ls
   ```

2. **Manually install plugin:**
   ```bash
   docker exec iot-grafana grafana-cli plugins install grafana-clickhouse-datasource
   docker restart iot-grafana
   ```

### Permission Denied Errors

If you see permission errors with provisioning:

1. **Check file permissions:**
   ```bash
   chmod -R 755 grafana/
   ```

2. **Check Docker volume mounts:**
   ```bash
   docker inspect iot-grafana | grep -A 10 Mounts
   ```

---

## 📝 Best Practices

### Query Performance

1. **Use time filters:** Always include `$__timeFilter(timestamp)` in time-series queries
2. **Limit results:** Use `LIMIT` clause for table panels
3. **Aggregate data:** Use `toStartOfMinute()` or `toStartOfHour()` for high-frequency data
4. **Index usage:** Queries are optimized for `(device_id, timestamp)` ordering

### Dashboard Design

1. **Refresh rates:**
   - Overview: 5-10 seconds
   - Detail: 10-30 seconds
   - Analytics: 30-60 seconds

2. **Time ranges:**
   - Real-time monitoring: Last 6 hours
   - Daily analysis: Last 24 hours
   - Historical: Last 7 days

3. **Color coding:**
   - Green: Normal/healthy
   - Yellow: Warning
   - Red: Critical/offline

---

## 🔗 Related Documentation

- [Project Specification](../SPEC.md)
- [Implementation Plan](../PLAN.md)
- [Backend Documentation](../mqtt_backbone/README.md)
- [ClickHouse Schema](../mqtt_backbone/internal/database/schema.go)

---

## 📋 Phase 5 Checklist

- [x] Create Grafana directory structure
- [x] Configure docker-compose.yml with Grafana service
- [x] Create ClickHouse datasource provisioning
- [x] Create dashboard provisioning configuration
- [x] Build Dashboard 1: System Overview
- [x] Build Dashboard 2: Device Detail
- [x] Build Dashboard 3: ML Metrics
- [x] Build Dashboard 4: System Health
- [x] Create README documentation

---

## 🎯 Next Steps

1. **Start the system:**
   ```bash
   cd mqtt_backbone
   docker-compose up -d
   ```

2. **Generate test data:**
   ```bash
   python scripts/test_publisher.py
   ```

3. **Access Grafana:**
   - URL: http://localhost:3000
   - Login: admin/admin
   - Navigate to dashboards

4. **Connect ESP32 devices** (Phase 3) or use MQTT simulators for testing

---

## 📞 Support

For issues or questions:
- Check troubleshooting section above
- Review project documentation in parent directory
- Check ClickHouse and Grafana logs
- Verify MQTT data flow with `test_subscriber.py`

---

**Version:** 1.0
**Last Updated:** 2025-10-27
**Phase:** 5 - Grafana Dashboards ✅
