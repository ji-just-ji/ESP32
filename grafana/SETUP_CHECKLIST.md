# Grafana Setup Checklist

---

## Files

### Directory Structure
```
grafana/
   provisioning/
      datasources/
         clickhouse.yml              # ClickHouse datasource configuration
      dashboards/
         default.yml                  # Dashboard provisioning
   dashboards/
      system-overview.json            # Dashboard 1: System Overview
      device-detail.json              # Dashboard 2: Device Detail
      ml-metrics.json                 # Dashboard 3: ML Metrics
      system-health.json              # Dashboard 4: System Health
   README.md                          # Documentation
   SETUP_CHECKLIST.md                 # This file
```

---

## Configuration Changes

### Docker Compose (mqtt_backbone/docker-compose.yml)

**Added Grafana service:**
```yaml
grafana:
  image: grafana/grafana:latest
  container_name: iot-grafana
  ports:
    - "3000:3000"
  environment:
    - GF_SECURITY_ADMIN_USER=admin
    - GF_SECURITY_ADMIN_PASSWORD=admin
    - GF_INSTALL_PLUGINS=grafana-clickhouse-datasource
  volumes:
    - ../grafana/provisioning:/etc/grafana/provisioning
    - ../grafana/dashboards:/var/lib/grafana/dashboards
    - grafana_data:/var/lib/grafana
  depends_on:
    - clickhouse
```

**Added volume:**
```yaml
grafana_data:
  driver: local
```

---

## Dashboard Specifications

### Dashboard 1: System Overview
- **UID:** `esp32-system-overview`
- **Refresh:** 5 seconds
- **Time Range:** Last 6 hours
- **Panels:** 9 panels
  - Active Devices counter
  - Latest sensor readings (temp, humidity, window position)
  - Temperature/Humidity trends
  - Sound Volume trends
  - Recent Window Actions table
  - Message throughput graph

### Dashboard 2: Device Detail
- **UID:** `esp32-device-detail`
- **Refresh:** 10 seconds
- **Time Range:** Last 24 hours
- **Panels:** 8 panels
- **Features:** Device selector variable
  - Device information
  - Temperature/Humidity/Sound Volume history
  - Window Position history
  - ML Confidence timeline
  - Position distribution pie chart
  - Recent predictions table

### Dashboard 3: ML Metrics
- **UID:** `esp32-ml-metrics`
- **Refresh:** 30 seconds
- **Time Range:** Last 24 hours
- **Panels:** 11 panels
  - Prediction frequency by device
  - Confidence score distribution
  - Prediction rate over time
  - Feature correlations (temp, humidity, sound vs position)
  - Performance statistics

### Dashboard 4: System Health
- **UID:** `esp32-system-health`
- **Refresh:** 10 seconds
- **Time Range:** Last 6 hours
- **Panels:** 12 panels
  - Device status table with health indicators
  - Device status distribution
  - MQTT message rates
  - Data ingestion rates
  - Record counts per table
  - Last-seen monitoring with color coding

---

## Data Sources

### ClickHouse Tables Queried

All dashboards query the following ClickHouse tables:

1. **sensor_temperature**
   - Device temperature readings
   - Columns: timestamp, device_id, value

2. **sensor_humidity**
   - Device humidity readings
   - Columns: timestamp, device_id, value

3. **sensor_audio**
   - Audio metadata (not raw audio)
   - Columns: timestamp, device_id, sample_rate, duration, features

4. **window_actions**
   - Window control actions from ML service
   - Columns: timestamp, device_id, position, confidence, temperature, humidity, **sound_volume**
   - ⭐ **Sound volume (dB) is stored here**

5. **device_registry**
   - Device metadata and health
   - Columns: device_id, name, location, last_seen, is_active

6. **ml_predictions**
   - ML performance metrics (if available)
   - Columns: timestamp, device_id, prediction, confidence, inference_time_ms

---

## Key Features Implemented

### Auto-Provisioning
- Datasource automatically configured on startup
- Dashboards automatically loaded
- No manual configuration required

### Sound Volume Integration
- Sound volume data displayed in dB units
- Queried from `window_actions.sound_volume` field
- Correlations with other features in ML Metrics dashboard

### Multi-Device Support
- Device selector variable in Device Detail dashboard
- Per-device filtering in all relevant panels
- System-wide aggregations in overview dashboards

### Real-Time Monitoring
- Auto-refresh configured per dashboard
- Live updates for sensor data
- Health monitoring with color-coded alerts

### Health Indicators
- Device online/offline status
- Last-seen timestamps with color coding
  - Green: < 5 minutes
  - Yellow: 5-15 minutes
  - Red: > 15 minutes
- Message throughput monitoring

---

## Testing Checklist

When you deploy, verify the following:

### Pre-Deployment
- [ ] All JSON files are valid (✅ Already validated)
- [ ] docker-compose.yml syntax is correct
- [ ] Directory structure is correct (✅ Verified)
- [ ] File permissions are correct

### Post-Deployment
- [ ] Start services: `cd mqtt_backbone && docker-compose up -d`
- [ ] Verify Grafana starts: `docker logs iot-grafana`
- [ ] Access Grafana: http://localhost:3000 (admin/admin)
- [ ] Verify ClickHouse datasource connects
- [ ] Check all 4 dashboards load
- [ ] Verify panels show "No Data" initially (expected without data)
- [ ] Run test publisher to generate data
- [ ] Verify panels populate with data
- [ ] Test device selector in Device Detail dashboard
- [ ] Verify auto-refresh works
- [ ] Check time range selectors

---

## Quick Start Commands

```bash
# Start the stack
cd mqtt_backbone
docker-compose up -d

# Check Grafana logs
docker logs -f iot-grafana

# Generate test data
python scripts/test_publisher.py

# Access Grafana
open http://localhost:3000
# Login: admin / admin
```

---

## File Validation Results

```
grafana/dashboards/device-detail.json     - Valid JSON (803 lines)
grafana/dashboards/ml-metrics.json        - Valid JSON (1007 lines)
grafana/dashboards/system-health.json     - Valid JSON (1115 lines)
grafana/dashboards/system-overview.json   - Valid JSON (872 lines)
grafana/provisioning/datasources/clickhouse.yml
grafana/provisioning/dashboards/default.yml
```

---

## Integration Points

### ClickHouse Connection
- Protocol: Native (port 9000)
- Database: iot
- User: default
- Password: (empty)

### Volume Mounts
- Provisioning: `../grafana/provisioning:/etc/grafana/provisioning`
- Dashboards: `../grafana/dashboards:/var/lib/grafana/dashboards`
- Data persistence: `grafana_data:/var/lib/grafana`

### Network
- Docker network: iot-network
- Services can communicate via service names
- ClickHouse accessible at: `http://clickhouse:9000`

---

## Next Steps (Phase 6+)

After deploying Grafana, continue with:

1. **Phase 3:** ESP32 Firmware Development
   - Sensor drivers
   - MQTT client
   - Motor control

2. **Phase 4:** Integration & Testing
   - End-to-end testing
   - Multi-device scenarios
   - Performance benchmarks

3. **Phase 6:** Device Management
   - Advanced health monitoring
   - Metrics export
   - Alerting

4. **Phase 7:** Documentation & Production
   - Deployment guides
   - Security hardening
   - Backup procedures

---

## Contact & Support

For issues:
1. Check grafana/README.md troubleshooting section
2. Review Grafana logs: `docker logs iot-grafana`
3. Verify ClickHouse connection: http://localhost:3000/datasources
4. Check SPEC.md and PLAN.md for system architecture

