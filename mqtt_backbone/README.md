# IoT Backend Service

A Go backend service that processes continuous sensor data (audio streams, temperature, humidity) from ESP32 devices via separate MQTT channels. Data is stored in ClickHouse for time-series analysis and visualization via Grafana. The system works in conjunction with a Python ML microservice that uses PyTorch to make automated window control decisions.

## System Overview

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

## Responsibilities

The Go Backend Service:
- Subscribes to all sensor MQTT topics (`sensor/+/temperature`, `sensor/+/humidity`, `sensor/+/audio`)
- Stores all incoming sensor data to ClickHouse
- Aggregates sensor data per device
- Detects significant changes (event-based triggering)
- Publishes inference requests to Python ML service via MQTT
- Subscribes to window control responses for logging
- Monitors device health
- Provides metrics and logging

## Prerequisites

- **Go 1.21+** - [Install Go](https://golang.org/doc/install)
- **ClickHouse** - [Install ClickHouse](https://clickhouse.com/docs/en/install)
- **MQTT Broker** (Mosquitto) - [Install Mosquitto](https://mosquitto.org/download/)
- **Python ML Service** - See the ML service documentation for setup

## Installation

1. Navigate to the backend directory:
```bash
cd backend
```

2. Install Go dependencies:
```bash
go mod download
```

3. Copy the example environment file:
```bash
cp .env.example .env
```

4. Update `.env` with your configuration

## Configuration

Configuration is managed via environment variables or YAML config file:

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

## Running the Service

### Development

```bash
go run cmd/server/main.go
```

### Production Build

```bash
go build -o iot-backend cmd/server/main.go
./iot-backend
```

## MQTT Topic Structure

### Sensor Data Topics (ESP32 → Go Backend)

**Temperature**: `sensor/{device_id}/temperature`
```json
{
  "value": 25.5,
  "timestamp": "2025-10-24T12:00:00Z"
}
```

**Humidity**: `sensor/{device_id}/humidity`
```json
{
  "value": 60.0,
  "timestamp": "2025-10-24T12:00:00Z"
}
```

**Audio**: `sensor/{device_id}/audio`
```json
{
  "data": "base64_encoded_wav",
  "sample_rate": 16000,
  "duration": 2.0,
  "timestamp": "2025-10-24T12:00:00Z"
}
```

### ML Inference Topics

**Request**: `ml/inference/request/{device_id}` (Go Backend → Python Service)
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

**Response**: `window/{device_id}/control` (Python Service → ESP32 & Go Backend)
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

## Data Models

### Temperature Reading
```go
type TemperatureReading struct {
    Timestamp time.Time
    DeviceID  string
    Value     float64  // Celsius
}
```

### Humidity Reading
```go
type HumidityReading struct {
    Timestamp time.Time
    DeviceID  string
    Value     float64  // Percentage 0-100
}
```

### Audio Recording
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

### Window Action (Continuous Control)
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

### Device Registry
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

## Database Schema (ClickHouse)

### sensor_temperature
```sql
CREATE TABLE sensor_temperature (
    timestamp DateTime64(3),
    device_id String,
    value Float64
) ENGINE = MergeTree()
ORDER BY (device_id, timestamp)
PARTITION BY toYYYYMM(timestamp)
```

### sensor_humidity
```sql
CREATE TABLE sensor_humidity (
    timestamp DateTime64(3),
    device_id String,
    value Float64
) ENGINE = MergeTree()
ORDER BY (device_id, timestamp)
PARTITION BY toYYYYMM(timestamp)
```

### sensor_audio
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

### window_actions
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

### device_registry
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

### ml_predictions
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

## Multi-Device Support

The system supports 2-10 devices initially and is designed for horizontal scaling:

- Each device has a unique `device_id`
- Devices are registered automatically on first connection
- Health monitoring via MQTT keep-alive
- Per-device configuration support
- Inactive device detection and alerts

## Project Structure

```
backend/
├── cmd/
│   └── server/          # Main application entry point
├── internal/
│   ├── mqtt/            # MQTT client implementation
│   ├── database/        # ClickHouse database client
│   ├── models/          # Data models
│   └── handlers/        # Message handlers and processing
├── pkg/
│   └── config/          # Configuration management
├── .env.example         # Example environment variables
├── go.mod               # Go module definition
├── spec.md              # System specification
└── README.md
```

## Technology Stack

- **Go 1.21+**
- **eclipse/paho.mqtt.golang** - MQTT client library
- **ClickHouse/clickhouse-go/v2** - ClickHouse driver
- **godotenv** - Environment configuration

## Testing MQTT

### Publishing sensor data:

**Temperature:**
```bash
mosquitto_pub -h localhost -t "sensor/sensor-001/temperature" -m '{
  "value": 25.5,
  "timestamp": "2025-10-24T12:00:00Z"
}'
```

**Humidity:**
```bash
mosquitto_pub -h localhost -t "sensor/sensor-001/humidity" -m '{
  "value": 60.0,
  "timestamp": "2025-10-24T12:00:00Z"
}'
```

### Subscribing to window control actions:
```bash
mosquitto_sub -h localhost -t "window/+/control"
```

### Subscribing to ML inference requests:
```bash
mosquitto_sub -h localhost -t "ml/inference/request/#"
```

## Change Detection & Event Triggering

The Go backend detects significant changes in sensor data to trigger ML inference:

- **Temperature threshold**: 0.5°C (configurable)
- **Humidity threshold**: 2.0% (configurable)
- **Audio**: Always triggers inference when new recording received

## Deployment

The full system is deployed using Docker Compose with the following services:

- ClickHouse (time-series database)
- Mosquitto (MQTT broker)
- Go Backend Service (this service)
- Python ML Service (PyTorch inference)
- Grafana (visualization)

Refer to the main project `docker-compose.yml` for complete deployment configuration.

## Monitoring

The service logs all operations including:
- MQTT connections and messages
- Sensor data ingestion rates
- Database write operations
- Event detection and triggering
- ML inference request/response cycles
- Device health status
- Error conditions and retries

## Related Services

- **Python ML Service**: Performs PyTorch-based inference for window control decisions
- **Grafana**: Provides real-time dashboards for sensor data and system metrics
- **ESP32 Devices**: Source of sensor data and target for window control commands

See `spec.md` for complete system architecture and specifications.
