# IoT Backend System Specification v2.0

## Project Overview

An IoT backend system that processes continuous sensor data (audio streams, temperature, humidity) from ESP32 devices via separate MQTT channels. Data is stored in ClickHouse, and a Python ML microservice uses PyTorch to make automated window control decisions (0-100% position) via MQTT. Grafana provides real-time visualization.

## System Architecture

```
ESP32 Devices (2-10)
    ↓ (separate MQTT topics)
    ├─ sensor/{device_id}/temperature
    ├─ sensor/{device_id}/humidity
    └─ sensor/{device_id}/audio
              ↓
        Go Backend Service
              ↓
    ┌─────────┴─────────┐
    ↓                   ↓
ClickHouse DB    ml/inference/request/{device_id}
    ↓                   ↓
Grafana         Python ML Service
Dashboard       (PyTorch Model)
                        ↓
              window/{device_id}/control
                        ↓
                ┌───────┴────────┐
                ↓                ↓
          Go Backend      ESP32 Window
          (logging)       Controller
```

## Requirements & Specifications

### 1. MQTT Topic Structure

#### Sensor Data Topics (ESP32 → Go Backend)

**Topic Pattern**: `sensor/{device_id}/{sensor_type}`

- `sensor/{device_id}/temperature`
  - Payload: `{"value": 25.5, "timestamp": "2025-10-24T12:00:00Z"}`
  - Unit: Celsius
  - Frequency: As needed

- `sensor/{device_id}/humidity`
  - Payload: `{"value": 60.0, "timestamp": "2025-10-24T12:00:00Z"}`
  - Unit: Percentage (0-100)
  - Frequency: As needed

- `sensor/{device_id}/audio`
  - Payload: `{"data": "base64_encoded_wav", "sample_rate": 16000, "duration": 2.0, "timestamp": "2025-10-24T12:00:00Z"}`
  - Format: WAV/PCM (buffered recordings)
  - Transmission: Periodic buffered batches
  - Encoding: Base64 for binary data

#### ML Inference Topics

**Request Topic**: `ml/inference/request/{device_id}` (Go Backend → Python Service)
```json
{
  "device_id": "sensor-001",
  "timestamp": "2025-10-24T12:00:00Z",
  "temperature": 25.5,
  "humidity": 60.0,
  "audio_data": "base64_encoded_wav",
  "audio_metadata": {
    "sample_rate": 16000,
    "duration": 2.0
  }
}
```

**Response Topic**: `window/{device_id}/control` (Python Service → ESP32 & Go Backend)
```json
{
  "device_id": "sensor-001",
  "timestamp": "2025-10-24T12:00:01Z",
  "position": 75.5,
  "confidence": 0.92,
  "features_used": {
    "temperature": 25.5,
    "humidity": 60.0,
    "audio_features": ["mfcc_mean", "spectral_centroid", "rms"]
  }
}
```

### 2. Data Models

#### Temperature Reading
```go
type TemperatureReading struct {
    Timestamp time.Time
    DeviceID  string
    Value     float64  // Celsius
}
```

#### Humidity Reading
```go
type HumidityReading struct {
    Timestamp time.Time
    DeviceID  string
    Value     float64  // Percentage 0-100
}
```

#### Audio Recording
```go
type AudioRecording struct {
    Timestamp  time.Time
    DeviceID   string
    Data       []byte    // Raw audio bytes
    SampleRate int       // e.g., 16000 Hz
    Duration   float64   // seconds
    Format     string    // "wav", "pcm"
}
```

#### Window Action (Continuous Control)
```go
type WindowAction struct {
    Timestamp   time.Time
    DeviceID    string
    Position    float64   // 0-100%
    Confidence  float64   // 0-1
    Temperature float64   // Input feature
    Humidity    float64   // Input feature
    AudioHash   string    // Reference to audio data
}
```

#### Device Registry
```go
type Device struct {
    DeviceID      string
    Name          string
    Location      string
    RegisteredAt  time.Time
    LastSeen      time.Time
    IsActive      bool
    Config        map[string]interface{}
}
```

### 3. ML Model Specifications

**Model Type**: PyTorch Regression Model
**Input Features**:
- Audio features: MFCC coefficients, spectral features, RMS, zero-crossing rate
- Temperature: Float (Celsius)
- Humidity: Float (Percentage)

**Output**: Continuous value 0-100 representing window position percentage
**Confidence Score**: Model uncertainty/confidence metric

**Feature Extraction** (Audio):
- MFCC (Mel-frequency cepstral coefficients)
- Spectral centroid
- Spectral rolloff
- RMS energy
- Zero-crossing rate
- Additional features as needed

**Audio Processing**:
- Storage: Configurable (can be enabled/disabled)
- Preprocessing: TBD (noise reduction, normalization)
- Sample Rate: 16000 Hz (configurable)

**Prediction Trigger**: Event-based
- Significant change in temperature (threshold: TBD)
- Significant change in humidity (threshold: TBD)
- New audio recording received
- Configurable change detection logic

### 4. Database Schema (ClickHouse)

#### sensor_temperature
```sql
CREATE TABLE sensor_temperature (
    timestamp DateTime64(3),
    device_id String,
    value Float64
) ENGINE = MergeTree()
ORDER BY (device_id, timestamp)
PARTITION BY toYYYYMM(timestamp)
```

#### sensor_humidity
```sql
CREATE TABLE sensor_humidity (
    timestamp DateTime64(3),
    device_id String,
    value Float64
) ENGINE = MergeTree()
ORDER BY (device_id, timestamp)
PARTITION BY toYYYYMM(timestamp)
```

#### sensor_audio
```sql
CREATE TABLE sensor_audio (
    timestamp DateTime64(3),
    device_id String,
    sample_rate UInt32,
    duration Float64,
    format String,
    audio_hash String,
    features String  -- JSON
) ENGINE = MergeTree()
ORDER BY (device_id, timestamp)
PARTITION BY toYYYYMM(timestamp)
```

#### window_actions
```sql
CREATE TABLE window_actions (
    timestamp DateTime64(3),
    device_id String,
    position Float64,        -- 0-100
    confidence Float64,      -- 0-1
    temperature Float64,
    humidity Float64,
    audio_hash String
) ENGINE = MergeTree()
ORDER BY (device_id, timestamp)
PARTITION BY toYYYYMM(timestamp)
```

#### device_registry
```sql
CREATE TABLE device_registry (
    device_id String,
    name String,
    location String,
    registered_at DateTime64(3),
    last_seen DateTime64(3),
    is_active Bool,
    config String  -- JSON
) ENGINE = ReplacingMergeTree(last_seen)
ORDER BY device_id
```

#### ml_predictions
```sql
CREATE TABLE ml_predictions (
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

### 5. Service Components

#### Go Backend Service

**Responsibilities**:
- Subscribe to all sensor MQTT topics (`sensor/+/temperature`, `sensor/+/humidity`, `sensor/+/audio`)
- Store all incoming sensor data to ClickHouse
- Aggregate sensor data per device
- Detect significant changes (event-based triggering)
- Publish inference requests to Python ML service
- Subscribe to window control responses for logging
- Device health monitoring
- Metrics and logging

**Configuration**:
```yaml
mqtt:
  broker: tcp://localhost:1883
  topics:
    temperature: sensor/+/temperature
    humidity: sensor/+/humidity
    audio: sensor/+/audio
    inference_request: ml/inference/request/{device_id}
    window_control: window/+/control

clickhouse:
  addr: localhost:9000
  database: iot

change_detection:
  temperature_threshold: 0.5  # Celsius
  humidity_threshold: 2.0     # Percentage
  audio_always_trigger: true
```

#### Python ML Microservice

**Responsibilities**:
- Subscribe to `ml/inference/request/#` MQTT topic
- Receive aggregated sensor data including audio
- Extract audio features using librosa
- Load and run PyTorch model inference
- Publish predictions to `window/{device_id}/control`
- Optional: Store raw audio files
- Model versioning and management
- Performance metrics

**Configuration**:
```yaml
mqtt:
  broker: tcp://localhost:1883
  topics:
    inference_request: ml/inference/request/#
    window_control: window/{device_id}/control

model:
  path: ./models/window_regressor.pth
  version: v1.0.0
  device: cpu  # or cuda

audio:
  sample_rate: 16000
  store_raw: false  # Configurable
  storage_path: ./audio_data/
  features:
    - mfcc
    - spectral_centroid
    - spectral_rolloff
    - rms
    - zero_crossing_rate
```

**Dependencies**:
- paho-mqtt
- torch
- librosa
- numpy
- soundfile

### 6. Grafana Dashboard

**Datasource**: ClickHouse

**Dashboards**:

1. **System Overview**
   - Real-time sensor readings (all devices)
   - Current window positions
   - System health indicators
   - Message throughput

2. **Device Detail View**
   - Historical temperature trends
   - Historical humidity trends
   - Audio activity timeline
   - Window position history
   - ML prediction accuracy

3. **ML Metrics**
   - Prediction frequency
   - Confidence score distribution
   - Inference latency
   - Model performance over time

4. **Audio Analysis**
   - Audio feature visualization
   - Spectral analysis
   - RMS levels over time

### 7. Deployment Configuration

**Docker Compose Services**:
- ClickHouse
- Mosquitto (MQTT Broker)
- Go Backend Service
- Python ML Service
- Grafana

**Networking**: All services on same Docker network

**Volumes**:
- ClickHouse data persistence
- Mosquitto data/logs
- ML model files
- Grafana dashboards
- Optional: Audio file storage

### 8. Multi-Device Support

**Scale**: 2-10 devices initially, designed for horizontal scaling

**Device Management**:
- Each device has unique device_id
- Device registration on first connection
- Health checks via MQTT keep-alive
- Per-device configuration support
- Inactive device detection

## Technology Stack

### Go Backend
- Go 1.21+
- eclipse/paho.mqtt.golang
- ClickHouse/clickhouse-go/v2
- godotenv

### Python ML Service
- Python 3.9+
- PyTorch
- librosa
- paho-mqtt
- numpy, scipy

### Infrastructure
- ClickHouse (time-series database)
- Mosquitto (MQTT broker)
- Grafana (visualization)
- Docker & Docker Compose
