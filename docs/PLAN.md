# ESP32 IoT System Implementation Plan

## Project Overview

Complete implementation plan for an intelligent IoT system featuring ESP32 sensors, MQTT messaging, Go-based backend processing, PyTorch ML inference, and real-time Grafana visualization.

---

## Project Parts

### Python ML Service

**Goal:** Implement PyTorch ML microservice for window control predictions

**Tasks:**

#### 2.1 Project Structure
- [ ] Create `ml_service/` directory structure
- [ ] Set up and manage Python virtual environment with `uv`
- [ ] Create `pyproject.toml` with dependencies
- [ ] Create `config.yaml` for configuration

#### 2.2 MQTT Client
- [ ] Implement MQTT subscriber for `ml/inference/request/#`
- [ ] Implement MQTT publisher for `window/{device_id}/control`
- [ ] Add connection/reconnection logic
- [ ] QoS 1 for reliable delivery

#### 2.3 Model Infrastructure
- [ ] Create model loader with versioning support
- [ ] Design input preprocessing pipeline
- [ ] Implement predictor with confidence scoring
  - [ ] Choose model with incremental training ability
- [ ] Add model validation on startup

#### 2.4 Feature Processing
- [ ] Parse inference request (temp, humidity, sound_volume)
- [ ] Normalize/scale input features
- [ ] Log and ignore requests with missing or invalid data

#### 2.5 Inference Pipeline
- [ ] Load PyTorch model
- [ ] Run inference on sensor data
- [ ] Calculate window position (0-100%)
- [ ] Generate confidence score
- [ ] Publish control message

#### 2.6 Deployment
- [ ] Create Dockerfile
- [ ] Add to docker-compose.yml
- [ ] Environment configuration
- [ ] Logging and error handling

**Components:**
```
ml_service/
├── Dockerfile
├── requirements.txt
├── config.yaml
├── src/
│   ├── main.py              # Entry point
│   ├── mqtt_client.py       # MQTT communication
│   ├── model_loader.py      # PyTorch model loading
│   ├── predictor.py         # Inference logic
│   └── feature_processor.py # Input preprocessing
├── models/
│   └── window_regressor.pth # Trained model
└── tests/
    └── test_predictor.py
```

**Dependencies:**
- paho-mqtt >= 1.6.1
- torch >= 2.0.0
- numpy >= 1.24.0
- pyyaml >= 6.0

**Duration:** 4-5 days (In Progress)

**Reference:** `mqtt_backbone/PLAN.md` - Phase 2

---

### Phase 3: ESP32 Firmware Development 📋 PLANNED

**Goal:** Develop ESP32 firmware for sensor reading and motor control

**Tasks:**

#### 3.1 Sensor Integration
- [ ] Temperature/humidity sensor driver (DHT22/BME280)
- [ ] I2S microphone driver for audio capture
- [ ] Sensor reading loop with configurable intervals

#### 3.2 Audio Processing
- [ ] Audio buffer management (2-second buffers)
- [ ] PCM data formatting
- [ ] Base64 encoding for MQTT
- [ ] Memory-efficient buffering

#### 3.3 MQTT Client
- [ ] WiFi connection management
- [ ] MQTT client library integration
- [ ] Publish sensor data to topics
- [ ] Subscribe to window control commands
- [ ] Reconnection logic

#### 3.4 Motor Control
- [ ] Servo/stepper motor driver
- [ ] Position control (0-100% mapping)
- [ ] Safety limits and calibration
- [ ] Smooth movement profiles

#### 3.5 Configuration
- [ ] WiFi credentials via web portal
- [ ] Device ID configuration
- [ ] MQTT broker settings
- [ ] OTA (Over-The-Air) updates

**Components:**
```
esp32_firmware/
├── src/
│   ├── main.cpp             # Main loop
│   ├── sensors.cpp          # Sensor drivers
│   ├── audio.cpp            # Audio capture
│   ├── mqtt_client.cpp      # MQTT communication
│   ├── motor_control.cpp    # Window motor
│   └── wifi_manager.cpp     # WiFi setup
├── include/
│   └── config.h             # Configuration
└── platformio.ini           # Build config
```

**Hardware Requirements:**
- ESP32 DevKit
- DHT22 or BME280 sensor
- I2S MEMS microphone (e.g., INMP441)
- Servo motor (SG90) or stepper motor
- Power supply (5V/2A)

**Duration:** 5-7 days

---

### Phase 4: Integration & Testing 📋 PLANNED

**Goal:** End-to-end testing with all components

**Tasks:**

#### 4.1 Component Integration
- [ ] Update docker-compose.yml with all services
- [ ] Configure service dependencies
- [ ] Set up Docker networks
- [ ] Volume management for persistence

#### 4.2 Test Infrastructure
- [ ] Create MQTT simulator scripts
- [ ] Implement test data generators
- [ ] Mock ESP32 devices for testing

#### 4.3 End-to-End Testing
- [ ] Sensor data flow (ESP32 → Backend → Database)
- [ ] ML inference pipeline (Backend → ML → Backend)
- [ ] Window control loop (ML → ESP32)
- [ ] Multi-device scenarios (2-3 devices)

#### 4.4 Performance Testing
- [ ] MQTT message throughput
- [ ] Database write performance
- [ ] ML inference latency
- [ ] System stability (24-hour test)

#### 4.5 Test Scripts
```
scripts/
├── test_mqtt_simulator.py      # Simulate ESP32 devices
├── test_multi_device.py        # Multi-device scenarios
├── test_inference_load.py      # ML service load testing
├── monitor_system.py           # System health monitoring
└── validate_data_flow.py       # End-to-end validation
```

**Duration:** 2-3 days

**Reference:** `mqtt_backbone/PLAN.md` - Phase 3

---

### Phase 5: Grafana Dashboards ✅ COMPLETE

**Goal:** Real-time visualization and monitoring

**Tasks:**

#### 5.1 Infrastructure
- [x] Add Grafana to docker-compose.yml
- [x] Configure ClickHouse datasource
- [x] Set up dashboard provisioning

#### 5.2 System Overview Dashboard
- [x] Real-time sensor readings (all devices)
- [x] Current window positions (gauge)
- [x] Active devices counter
- [x] Message throughput metrics
- [x] Recent predictions table

#### 5.3 Device Detail Dashboard
- [x] Device selector variable
- [x] Temperature history (time-series)
- [x] Humidity history (time-series)
- [x] Sound volume levels (time-series)
- [x] Window position history
- [x] ML prediction timeline

#### 5.4 ML Metrics Dashboard
- [x] Prediction frequency by device
- [x] Confidence score distribution (histogram)
- [x] Inference latency (time-series)
- [x] Model accuracy metrics
- [x] Feature correlations

#### 5.5 System Health Dashboard
- [x] Service status indicators
- [x] MQTT message rates
- [x] Database query performance
- [x] Error rates and logs
- [x] Device last-seen timestamps

**Components:**
```
grafana/
├── provisioning/
│   ├── datasources/
│   │   └── clickhouse.yml
│   └── dashboards/
│       └── default.yml
├── dashboards/
│   ├── system-overview.json
│   ├── device-detail.json
│   ├── ml-metrics.json
│   └── system-health.json
├── README.md
├── SETUP_CHECKLIST.md
└── QUERIES.md
```

**Duration:** 2-3 days (Completed)

**Key Achievements:**
- ✅ 4 production-ready dashboards with 40+ panels
- ✅ Auto-provisioning for zero-config setup
- ✅ Sound volume visualization from window_actions table
- ✅ Multi-device support with device selector
- ✅ Real-time monitoring with auto-refresh
- ✅ Health indicators with color-coded alerts
- ✅ Comprehensive documentation and query reference

**Reference:** `grafana/README.md`, `grafana/SETUP_CHECKLIST.md`

---

### Phase 6: Device Management & Monitoring 📋 PLANNED

**Goal:** Production-ready device management and health monitoring

**Tasks:**

#### 6.1 Device Registry (Go Backend)
- [ ] Auto-registration on first message
- [ ] Device metadata storage
- [ ] Last-seen tracking
- [ ] Active/inactive status
- [ ] Per-device configuration

#### 6.2 Health Monitoring
- [ ] Service health checks
- [ ] MQTT connection monitoring
- [ ] Database connection monitoring
- [ ] Detect inactive devices
- [ ] Alert on failures

#### 6.3 Metrics & Observability
- [ ] Prometheus metrics export
- [ ] Message processing rates
- [ ] Inference latency tracking
- [ ] Database performance metrics
- [ ] Error rate tracking

#### 6.4 Logging
- [ ] Structured JSON logging
- [ ] Log levels (DEBUG, INFO, WARN, ERROR)
- [ ] Request correlation IDs
- [ ] Log aggregation (optional)

**Duration:** 2-3 days

**Reference:** `mqtt_backbone/PLAN.md` - Phase 5

---

### Phase 7: Documentation & Production Readiness 📋 PLANNED

**Goal:** Complete documentation and production deployment guides

**Tasks:**

#### 7.1 Documentation
- [ ] Project README with quick start
- [ ] Architecture documentation
- [ ] API/MQTT topic reference
- [ ] Deployment guides
- [ ] Troubleshooting guide
- [ ] Model training guide

#### 7.2 Configuration Management
- [ ] `.env.example` files for all services
- [ ] Configuration validation
- [ ] Environment-specific configs (dev/prod)

#### 7.3 Deployment
- [ ] Docker Compose production profile
- [ ] Kubernetes manifests (optional)
- [ ] Scaling guidelines
- [ ] Backup/restore procedures

#### 7.4 Security
- [ ] MQTT authentication/authorization
- [ ] Database access controls
- [ ] Network security policies
- [ ] Secrets management

**Duration:** 2-3 days

**Reference:** `mqtt_backbone/PLAN.md` - Phase 6

---

## Project Timeline

### Overall Schedule

| Phase | Status | Duration | Dependencies | Timeline |
|-------|--------|----------|--------------|----------|
| Phase 0: Infrastructure | ✅ Complete | 1-2 days | None | Week 1 |
| Phase 1: Go Backend v1.0 | ✅ Complete | 3-4 days | Phase 0 | Week 1 |
| Phase 1.5: Backend Refactor | ✅ Complete | 2-3 days | Phase 1 | Week 2 |
| Phase 2: Python ML Service | 📋 Planned | 4-5 days | Phase 1.5 | Week 2-3 |
| Phase 3: ESP32 Firmware | 📋 Planned | 5-7 days | None (parallel) | Week 2-3 |
| Phase 4: Integration Testing | 📋 Planned | 2-3 days | Phase 2, 3 | Week 3 |
| Phase 5: Grafana Dashboards | ✅ Complete | 2-3 days | Phase 1.5 (can run parallel) | Week 4 |
| Phase 6: Device Management | 📋 Planned | 2-3 days | Phase 4 | Week 4 |
| Phase 7: Documentation | 📋 Planned | 2-3 days | All phases | Week 4-5 |

**Total Estimated Duration:** 4-5 weeks

**Parallel Tracks:**
- Backend (Phase 1-2) and ESP32 (Phase 3) can be developed simultaneously
- Testing can begin incrementally as components are completed

---

## Current Status Summary

### ✅ Completed Components

**Infrastructure:**
- Docker Compose setup
- Mosquitto MQTT broker
- ClickHouse database

**Go Backend (mqtt_backbone):**
- Channel-based architecture ✅
- Sound volume extraction ✅
- Smart inference triggering ✅
- Multi-topic MQTT support ✅
- ClickHouse persistence ✅
- Device auto-registration ✅
- Graceful shutdown ✅

**Documentation:**
- Project-level SPEC.md ✅
- Backend SPEC.md (detailed) ✅
- Backend PLAN.md ✅
- Cleanup documentation ✅

**Grafana Dashboards (Phase 5):**
- 4 production-ready dashboards ✅
- Auto-provisioning configuration ✅
- ClickHouse datasource integration ✅
- Sound volume visualization ✅
- Multi-device support ✅
- Health monitoring panels ✅
- Comprehensive documentation ✅

### 📋 Planned

**Python ML Service (Phase 2):**
- Project structure definition
- MQTT client design
- Model infrastructure planning

**ESP32 Firmware (Phase 3):**
- Sensor drivers
- Audio capture
- MQTT client
- Motor control

**Integration & Testing (Phase 4)**
**Device Management (Phase 6)**
**Documentation (Phase 7)**

---

## Integration Points

### Component Communication

```
┌──────────────┐    MQTT     ┌──────────────┐    Channels   ┌──────────────┐
│   ESP32      ├────────────>│ Go Backend   ├──────────────>│  ClickHouse  │
│   Devices    │<────────────┤  (mqtt_      │               │   Database   │
└──────────────┘    MQTT     │   backbone)  │               └──────┬───────┘
                             └──────┬───────┘                      │
                                    │ MQTT                         │ SQL
                                    ↓                              ↓
                             ┌──────────────┐               ┌──────────────┐
                             │   Python     │               │   Grafana    │
                             │  ML Service  │               │  Dashboards  │
                             └──────────────┘               └──────────────┘
```

### Data Flow Checkpoints

1. **ESP32 → MQTT Broker**
   - Topic: `sensor/{device_id}/{type}`
   - Payload: Raw sensor values
   - Frequency: Variable (on-change or periodic)

2. **Go Backend Subscriber → Services**
   - Channel: Go typed channels
   - Processing: Volume extraction, persistence
   - Validation: Data integrity checks

3. **Go Backend Publisher → MQTT Broker**
   - Topic: `ml/inference/request/{device_id}`
   - Payload: {temp, humidity, volume}
   - Trigger: Smart buffering logic

4. **Python ML → MQTT Broker**
   - Topic: `window/{device_id}/control`
   - Payload: {position, confidence}
   - Response: <1 second

5. **Go Backend → ClickHouse**
   - All sensor readings
   - Window actions
   - ML predictions
   - Device registry

6. **Grafana → ClickHouse**
   - Real-time queries
   - Historical analytics
   - Aggregations

---

## Success Criteria

### Phase Completion Criteria

**Phase 2 (ML Service):**
- [ ] Receives inference requests via MQTT
- [ ] Loads PyTorch model successfully
- [ ] Predicts window position (0-100%)
- [ ] Publishes control messages
- [ ] Handles errors gracefully
- [ ] Runs in Docker container

**Phase 3 (ESP32 Firmware):**
- [ ] Reads temperature/humidity
- [ ] Captures audio samples
- [ ] Publishes to MQTT topics
- [ ] Receives window control commands
- [ ] Controls motor position
- [ ] Reconnects on WiFi loss

**Phase 4 (Integration):**
- [ ] End-to-end data flow validated
- [ ] Multi-device testing (2-3 devices)
- [ ] 24-hour stability test passed
- [ ] Performance benchmarks met
- [ ] All docker services running

**Phase 5 (Grafana):**
- [x] All dashboards functional
- [x] Real-time data display
- [x] Historical queries working
- [x] Alerting configured (optional - health indicators)

**Phase 6 (Device Management):**
- [ ] Device auto-registration works
- [ ] Health monitoring active
- [ ] Metrics exported
- [ ] Inactive device detection

**Phase 7 (Documentation):**
- [ ] All README files complete
- [ ] Deployment guides tested
- [ ] API documentation current
- [ ] Troubleshooting guide available

### Overall System Success

- ✅ Multi-device support (2-10 devices)
- ⏳ ML-driven window control operational
- ✅ Real-time visualization in Grafana
- 📋 Data persistence in ClickHouse
- 📋 System stability >99% uptime
- 📋 Inference latency <2 seconds
- 📋 Complete documentation

---

## Risk Management

### Technical Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| PyTorch model compatibility | High | Medium | Test model export/import early, use standard formats |
| ESP32 memory constraints | High | Medium | Optimize audio buffering, use streaming where possible |
| MQTT message size limits | Medium | Low | Audio already chunked, monitor payload sizes |
| Network reliability (ESP32) | Medium | Medium | Implement reconnection logic, message queuing |
| ClickHouse performance | Medium | Low | Partition by time, index properly, monitor queries |
| Audio processing overhead | Medium | Medium | Profile and optimize RMS calculation, consider hardware |
| Multi-device scaling | Low | Low | Architecture designed for horizontal scaling |

### Mitigation Strategies

1. **Early Testing**: Test integration points early and often
2. **Incremental Development**: Complete and test each phase before moving forward
3. **Parallel Development**: ESP32 and ML service can be developed simultaneously
4. **Simulation**: Use MQTT simulators before hardware is ready
5. **Monitoring**: Implement observability from the start
6. **Documentation**: Keep documentation in sync with implementation

---

## Next Steps

### Immediate Actions (Week 2-3)

1. **Complete Phase 2 (ML Service)**
   - [ ] Set up Python project structure
   - [ ] Implement MQTT client
   - [ ] Create model loader and predictor
   - [ ] Build and deploy Docker container
   - [ ] Test with Go backend

2. **Start Phase 3 (ESP32 Firmware)**
   - [ ] Set up development environment
   - [ ] Implement sensor drivers
   - [ ] Test MQTT publishing
   - [ ] Basic motor control

3. **Prepare Phase 4 (Integration)**
   - [ ] Create test scripts
   - [ ] Set up test infrastructure
   - [ ] Define test scenarios

### Mid-Term Goals (Week 3-4)

- Complete ESP32 firmware development
- Integration testing with 2-3 devices
- Grafana dashboard implementation
- Device management features

### Long-Term Goals (Week 4-5)

- Production-ready deployment
- Complete documentation
- Performance optimization
- Security hardening

---

## References

- **Project Specification**: `SPEC.md` (project root)
- **Backend Specification**: `mqtt_backbone/SPEC.md`
- **Backend Implementation Plan**: `mqtt_backbone/PLAN.md`
- **Cleanup Documentation**: `mqtt_backbone/CLEANUP_SUMMARY.md`

---

## Revision History

| Version | Date | Changes |
|---------|------|---------|
| 1.5 | 2025-10-27 | Added project-level plan, Phase 1.5 complete |
| 1.0 | 2025-10-24 | Initial plan creation |
