# Implementation Plan: IoT Backend v1.0

## Overview

Refactor existing Go backend and add Python ML microservice with Grafana dashboard for IoT sensor monitoring and automated window control.

## Architecture Summary

**MQTT Topics**:
- Input: `sensor/{device_id}/temperature`, `sensor/{device_id}/humidity`, `sensor/{device_id}/audio`
- Inference: `ml/inference/request/{device_id}` (Go → Python)
- Control: `window/{device_id}/control` (Python → ESP32 & Go)

**Data Flow**:
ESP32 → Go Backend (store to ClickHouse) → Detect changes → Publish inference request → Python ML Service (PyTorch) → Publish control action → ESP32 & Go Backend (logging)

**Key Decisions**:
- Audio: Buffered WAV/PCM recordings sent periodically
- Window Control: Continuous (0-100%)
- Predictions: Event-based (on significant changes)
- Scale: 2-10 devices

---

## Phase 1: Refactor Go Backend for Multi-Topic MQTT

### Tasks

1. ✅ **Update Data Models** (`internal/models/`)
   - ✅ Create separate structs for Temperature, Humidity, Audio
   - ✅ Update WindowAction for continuous position (0-100%)
   - ✅ Add Device model
   - ✅ Add InferenceRequest and InferenceResponse models

2. ✅ **Refactor MQTT Client** (`internal/mqtt/`)
   - ✅ Update `client.go` to support multiple topic subscriptions
   - ✅ Create `multi_topic.go` with topic-specific handlers
   - ✅ Add handler for `temperature`, `humidity`, `audio` topics
   - ✅ Add publisher for `ml/inference/request/{device_id}`
   - ✅ Add subscriber for `window/+/control` for logging

3. ✅ **Create Sensor Aggregator** (`internal/aggregator/`)
   - ✅ New package: `sensor_buffer.go`
   - ✅ Buffer sensor data per device
   - ✅ Implement change detection logic (thresholds)
   - ✅ Aggregate and prepare inference request payload

4. ✅ **Update ClickHouse Schema** (`internal/database/`)
   - ✅ Create `schema.go` with all table definitions
   - ✅ Add separate tables: `sensor_temperature`, `sensor_humidity`, `sensor_audio`
   - ✅ Update `window_actions` for continuous position
   - ✅ Add `device_registry` table
   - ✅ Add `ml_predictions` table
   - ✅ Update `clickhouse.go` with new save methods

5. ✅ **Update Configuration** (`pkg/config/`)
   - ✅ Add multi-topic configuration
   - ✅ Add change detection thresholds
   - ✅ Add inference request topic config

6. ✅ **Update Main Application** (`cmd/server/main.go`)
   - ✅ Initialize multi-topic MQTT subscriptions
   - ✅ Set up sensor aggregator
   - ✅ Wire up inference request publisher
   - ✅ Add window control subscriber for logging

### Files to Modify/Create
```
backend/internal/models/
  ├── sensor.go (update)
  ├── audio.go (new)
  └── device.go (new)

backend/internal/mqtt/
  ├── client.go (update)
  └── multi_topic.go (new)

backend/internal/aggregator/
  └── sensor_buffer.go (new)

backend/internal/database/
  ├── clickhouse.go (update)
  └── schema.go (new)

backend/pkg/config/
  └── config.go (update)

backend/cmd/server/
  └── main.go (update)
```

---

## Phase 1.5: Refactor ML Inference Logic + Channel-Based Architecture

### Overview

Refactor the inference trigger logic and architecture to:
1. Use sound volume (dB) instead of full audio data for ML inference
2. Handle sensors arriving separately (first inference requires all 3, then use most recent values)
3. Implement channel-based communication between MQTT layer and services
4. Separate API/transport layer (MQTT) from business logic (services)

### Tasks

1. ✅ **Update Documentation**
   - ✅ Update SPEC.md with new ML features (sound volume instead of audio features)
   - ✅ Update inference request format to use `sound_volume` field
   - ✅ Update prediction trigger logic documentation
   - ✅ Update database schema in spec (sound_volume instead of audio_hash)

2. **Create Audio Processing Module** (`internal/aggregator/audio_processor.go`)
   - Implement `ExtractSoundVolume(audioData []byte, sampleRate int) float64`
   - Parse 16-bit PCM samples from audio data
   - Calculate RMS: sqrt(mean(samples²))
   - Convert to dB: 20*log10(RMS/32768.0)
   - Handle edge cases (silence, invalid data)

3. **Update Data Models** (`internal/models/sensor.go`)
   - Modify `InferenceRequest`:
     - Remove: `AudioData string`, `AudioMetadata AudioMetadata`
     - Add: `SoundVolume float64`
   - Modify `WindowAction`:
     - Replace: `AudioHash string` with `SoundVolume float64`

4. **Create Services Layer** (New package: `internal/services/`)

   **SensorService** (`sensor_service.go`):
   - Struct with channels: `TempChan`, `HumidityChan`, `AudioChan`
   - Dependencies: database client, inference service
   - Methods:
     - `Start(ctx context.Context)` - goroutine listening to sensor channels
     - `processTemperature()` - save to DB, forward to inference service
     - `processHumidity()` - save to DB, forward to inference service
     - `processAudio()` - extract volume, save metadata to DB, forward to inference service

   **InferenceService** (`inference_service.go`):
   - Struct with output channel: `InferenceReqChan`
   - Dependencies: sensor aggregator
   - Buffering logic:
     - Track latest temp, humidity, volume per device
     - Track `HasCompletedFirstInference` per device
     - First inference: require all 3 sensors
     - Subsequent: always use most recent values
   - Methods:
     - `Start(ctx context.Context)` - goroutine for processing
     - `UpdateTemperature()` - check threshold, maybe trigger
     - `UpdateHumidity()` - check threshold, maybe trigger
     - `UpdateVolume()` - always triggers inference
     - `triggerInference()` - validate, create request, write to channel

5. **Refactor MQTT Layer** (`internal/mqtt/`)

   **New: subscriber.go**:
   - `Subscriber` struct with output channels
   - Subscribe to all sensor topics
   - Parse messages, write to channels
   - No business logic, pure transport

   **New: publisher.go**:
   - `Publisher` struct with input channels
   - `Start(ctx context.Context)` - goroutine reading from `InferenceReqChan`
   - Publish to MQTT when requests arrive
   - No business logic, pure transport

   **Update: client.go**:
   - Simplify to low-level MQTT connection management
   - Remove handler callbacks
   - Keep connection, reconnection logic

6. **Update Sensor Aggregator** (`internal/aggregator/sensor_buffer.go`)
   - Add fields: `LastSoundVolume *float64`, `HasCompletedFirstInference bool`
   - Remove: `onInferenceNeeded` callback (services layer handles this now)
   - Add method: `IsReadyForInference() bool` - checks if all 3 sensors present
   - Simplify to state tracking only

7. **Update Database Layer** (`internal/database/`)
   - Update `schema.go`: `window_actions` table with `sound_volume Float64`
   - Update `clickhouse.go`:
     - Modify `SaveWindowAction()` to use sound_volume
     - Update insert queries

8. **Update Main Application** (`cmd/server/main.go`)
   - Create all channels (temp, humidity, audio, window control, inference request)
   - Initialize services with channels and dependencies
   - Wire: MQTT subscriber → channels → sensor service → inference service → channels → MQTT publisher
   - Start all goroutines (subscriber, services, publisher)
   - Implement graceful shutdown with context cancellation

### New Architecture Flow

```
ESP32 Sensors
    ↓ MQTT
┌─────────────────────────────────────┐
│ MQTT Subscriber (mqtt/subscriber.go)│
│  - Parse messages                   │
│  - Write to channels                │
└────────┬────────────────────────────┘
         ↓ (channels)
┌─────────────────────────────────────┐
│ Sensor Service (services/)          │
│  - Read from sensor channels        │
│  - Save to ClickHouse               │
│  - Extract volume from audio        │
│  - Forward to InferenceService      │
└────────┬────────────────────────────┘
         ↓
┌─────────────────────────────────────┐
│ Inference Service (services/)       │
│  - Buffer sensor data per device    │
│  - Apply trigger logic              │
│  - Generate InferenceRequest        │
│  - Write to channel                 │
└────────┬────────────────────────────┘
         ↓ (channel)
┌─────────────────────────────────────┐
│ MQTT Publisher (mqtt/publisher.go)  │
│  - Read from inference channel      │
│  - Publish to MQTT                  │
└─────────────────────────────────────┘
         ↓ MQTT
    Python ML Service
```

### Files to Create

```
backend/internal/services/
  ├── sensor_service.go (new)
  └── inference_service.go (new)

backend/internal/mqtt/
  ├── subscriber.go (new)
  └── publisher.go (new)

backend/internal/aggregator/
  └── audio_processor.go (new)
```

### Files to Modify

```
backend/PLAN.md (this file)
backend/internal/models/sensor.go
backend/internal/aggregator/sensor_buffer.go
backend/internal/database/schema.go
backend/internal/database/clickhouse.go
backend/internal/mqtt/client.go
backend/cmd/server/main.go
```

### Success Criteria

- ✅ Sound volume (dB) extracted from audio data
- ✅ First inference requires all 3 sensor types (temp, humidity, volume)
- ✅ Subsequent inferences always use most recent values
- ✅ MQTT layer is pure transport (no business logic)
- ✅ Services layer handles all business logic
- ✅ Go channels provide clean separation between layers
- ✅ Graceful shutdown with context cancellation
- ✅ All existing functionality preserved

---

## Phase 2: Create Python ML Microservice

### Tasks

1. **Project Structure**
   ```
   ml-service/
   ├── Dockerfile
   ├── requirements.txt
   ├── config.yaml
   ├── src/
   │   ├── __init__.py
   │   ├── main.py
   │   ├── mqtt_client.py
   │   ├── audio_processor.py
   │   ├── model_loader.py
   │   └── predictor.py
   ├── models/
   │   └── window_regressor.pth
   └── tests/
       └── test_predictor.py
   ```

2. **MQTT Client** (`mqtt_client.py`)
   - Subscribe to `ml/inference/request/#`
   - Publish to `window/{device_id}/control`
   - Handle connection/reconnection
   - QoS 1 for reliable delivery

3. **Audio Processor** (`audio_processor.py`)
   - Decode base64 audio data
   - Extract features using librosa:
     - MFCC
     - Spectral centroid
     - Spectral rolloff
     - RMS energy
     - Zero-crossing rate
   - Optional: Save raw audio files

4. **Model Loader** (`model_loader.py`)
   - Load PyTorch model from .pth file
   - Validate model on startup
   - Support model versioning
   - Handle model loading errors gracefully

5. **Predictor** (`predictor.py`)
   - Combine audio features with temp/humidity
   - Run model inference
   - Calculate confidence score
   - Return window position (0-100%)

6. **Main Application** (`main.py`)
   - Load configuration
   - Initialize MQTT client
   - Set up message handlers
   - Orchestrate prediction pipeline
   - Logging and error handling

7. **Configuration** (`config.yaml`)
   - MQTT broker settings
   - Model path and version
   - Audio processing parameters
   - Feature extraction settings
   - Storage options

8. **Dependencies** (`requirements.txt`)
   ```
   paho-mqtt>=1.6.1
   torch>=2.0.0
   librosa>=0.10.0
   numpy>=1.24.0
   soundfile>=0.12.0
   pyyaml>=6.0
   ```

9. **Dockerfile**
   - Python 3.9+ base image
   - Install dependencies
   - Copy source code and models
   - Entry point: `python src/main.py`

### Files to Create
```
ml-service/
├── Dockerfile
├── requirements.txt
├── config.yaml
├── src/
│   ├── __init__.py
│   ├── main.py
│   ├── mqtt_client.py
│   ├── audio_processor.py
│   ├── model_loader.py
│   └── predictor.py
├── models/
│   └── .gitkeep
└── tests/
    └── test_predictor.py
```

---

## Phase 3: Integration & Testing

### Tasks

1. **Update Docker Compose**
   - Add Python ML service
   - Configure service dependencies
   - Add volume for ML models
   - Ensure all services on same network

2. **End-to-End Testing**
   - Create test MQTT publishers for each sensor type
   - Simulate ESP32 devices
   - Test complete data flow
   - Verify ClickHouse data storage
   - Verify window control outputs

3. **Integration Tests**
   - Test MQTT communication between services
   - Test inference request/response flow
   - Test error handling
   - Test reconnection logic

4. **Test Scripts** (`scripts/`)
   - `test_temp_publisher.py` - Simulate temperature data
   - `test_humidity_publisher.py` - Simulate humidity data
   - `test_audio_publisher.py` - Simulate audio recordings
   - `test_multi_device.py` - Simulate multiple devices
   - `monitor_window_control.py` - Monitor control outputs

### Files to Modify/Create
```
backend/docker-compose.yml (update)
backend/scripts/
  ├── test_temp_publisher.py (new)
  ├── test_humidity_publisher.py (new)
  ├── test_audio_publisher.py (new)
  ├── test_multi_device.py (new)
  └── monitor_window_control.py (new)
```

---

## Phase 4: Grafana Dashboard Setup

### Tasks

1. **Update Docker Compose**
   - Add Grafana service (port 3000)
   - Add volume for Grafana data
   - Add volume for dashboard provisioning

2. **ClickHouse Datasource**
   - Create datasource configuration file
   - Configure connection to ClickHouse
   - Set up provisioning

3. **Dashboard Templates**
   - **System Overview** (`overview.json`)
     - Time-series: Temperature by device
     - Time-series: Humidity by device
     - Gauge: Current window positions
     - Stat: Active devices count
     - Table: Recent predictions

   - **Device Detail** (`device-detail.json`)
     - Variable: device_id selector
     - Time-series: Temperature history
     - Time-series: Humidity history
     - Time-series: Window position history
     - Bar chart: Audio activity

   - **ML Metrics** (`ml-metrics.json`)
     - Time-series: Prediction frequency
     - Histogram: Confidence distribution
     - Time-series: Inference latency
     - Table: Model performance

   - **System Health** (`system-health.json`)
     - Stat: Message throughput
     - Time-series: Error rates
     - Table: Device last seen
     - Heatmap: Device activity

4. **Provisioning Configuration**
   - Auto-provision datasource on startup
   - Auto-load dashboards on startup
   - Set default home dashboard

### Files to Create
```
grafana/
├── provisioning/
│   ├── datasources/
│   │   └── clickhouse.yml
│   └── dashboards/
│       └── default.yml
└── dashboards/
    ├── overview.json
    ├── device-detail.json
    ├── ml-metrics.json
    └── system-health.json
```

---

## Phase 5: Device Management & Health Monitoring

### Tasks

1. **Device Registry** (Go Backend)
   - Auto-register devices on first message
   - Update last_seen timestamp
   - Track device health status
   - Store device configurations

2. **Health Monitoring**
   - Periodic health checks
   - Detect inactive devices
   - MQTT connection monitoring
   - Database connection monitoring

3. **Metrics Collection**
   - Message processing rate
   - Inference latency
   - Database write latency
   - Error rates by type

4. **Logging Enhancements**
   - Structured JSON logging
   - Log levels (DEBUG, INFO, WARN, ERROR)
   - Request tracing (correlation IDs)

### Files to Modify/Create
```
backend/internal/device/
  └── registry.go (new)

backend/internal/health/
  └── monitor.go (new)

backend/internal/metrics/
  └── collector.go (new)
```

---

## Phase 6: Documentation & Deployment

### Tasks

1. **Update Documentation**
   - Update README.md with new architecture
   - Update QUICKSTART.md with all services
   - Add ML service deployment guide
   - Add model training guide
   - Add troubleshooting guide

2. **Environment Configuration**
   - Update `.env.example` with all new variables
   - Document all configuration options
   - Add validation for required configs

3. **Deployment Guides**
   - Local development setup
   - Docker deployment
   - Production considerations
   - Scaling guidelines

4. **API Documentation**
   - Document MQTT topic structure
   - Document message formats
   - Document database schema
   - Add sequence diagrams

### Files to Modify/Create
```
backend/README.md (update)
backend/QUICKSTART.md (update)
backend/.env.example (update)
backend/docs/
  ├── DEPLOYMENT.md (new)
  ├── MODEL_TRAINING.md (new)
  ├── TROUBLESHOOTING.md (new)
  └── API.md (new)
```

---

## Timeline Estimate

| Phase | Duration | Dependencies |
|-------|----------|--------------|
| Phase 1: Go Backend Refactor | 3-4 days | None |
| Phase 2: Python ML Service | 4-5 days | None (parallel) |
| Phase 3: Integration & Testing | 2-3 days | Phase 1, 2 |
| Phase 4: Grafana Dashboard | 2-3 days | Phase 1, 3 |
| Phase 5: Device Management | 2-3 days | Phase 1 |
| Phase 6: Documentation | 2-3 days | All phases |

**Total: 15-21 days**

---

## Success Criteria

- [ ] Multi-topic MQTT working for all sensor types
- [ ] ClickHouse storing data in separate tables
- [ ] Python ML service receiving inference requests
- [ ] PyTorch model making predictions
- [ ] Window control commands published and logged
- [ ] Grafana dashboards showing real-time data
- [ ] Multiple devices supported (tested with 2-3)
- [ ] End-to-end data flow working
- [ ] All services in Docker Compose
- [ ] Documentation complete and accurate

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| PyTorch model format incompatible | High | Create model export script, test early |
| Audio processing too slow | Medium | Optimize feature extraction, add caching |
| MQTT message size limits | Medium | Chunk large audio, adjust broker config |
| ClickHouse schema changes | Low | Version migrations, test with sample data |
| Service coordination complexity | Medium | Clear interfaces, extensive testing |

---

## Next Steps

1. ✅ Finalize specification
2. ✅ Review and approve plan
3. ⏳ Start Phase 1: Go Backend refactor
4. ⏳ Start Phase 2: Python ML service (parallel)
5. Continue through remaining phases
