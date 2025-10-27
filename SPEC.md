# ESP32 IoT System Specification

## Project Overview

An intelligent IoT system for automated environmental control using ESP32 microcontrollers, MQTT messaging, machine learning inference, and real-time data visualization. The system collects environmental data (temperature, humidity, sound levels) from multiple ESP32 devices, processes this data through a Go-based backend service, and uses a PyTorch ML model to make automated window control decisions.

### Key Features

- **Multi-Device Support**: 2-10 ESP32 devices per deployment
- **Real-Time Processing**: Sub-second sensor data collection and processing
- **ML-Powered Decisions**: PyTorch regression model for intelligent window positioning
- **Sound-Based Intelligence**: Audio processing to extract ambient sound volume (dB)
- **Historical Analytics**: ClickHouse time-series database with Grafana visualization
- **Scalable Architecture**: Channel-based Go backend with clear separation of concerns

---

## System Architecture

### High-Level System Diagram

```
┌──────────────────────────────────────────────────────────────────┐
│                    ESP32 IoT Ecosystem                           │
└──────────────────────────────────────────────────────────────────┘

┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│  ESP32 Device 1 │      │  ESP32 Device 2 │      │  ESP32 Device N │
│                 │      │                 │      │                 │
│ • Temperature   │      │ • Temperature   │      │ • Temperature   │
│ • Humidity      │      │ • Humidity      │      │ • Humidity      │
│ • Microphone    │      │ • Microphone    │      │ • Microphone    │
│ • Window Motor  │      │ • Window Motor  │      │ • Window Motor  │
└────────┬────────┘      └────────┬────────┘      └────────┬────────┘
         │                        │                        │
         └────────────────────────┼────────────────────────┘
                                  │ MQTT (WiFi)
                                  ↓
         ┌────────────────────────────────────────────┐
         │         Mosquitto MQTT Broker              │
         │         (Message Bus)                      │
         └────────┬────────────────────────┬──────────┘
                  ↓                        ↓
    ┌─────────────────────────┐  ┌─────────────────────────┐
    │   Go Backend Service    │  │  Python ML Service      │
    │   (mqtt_backbone)       │  │  (ml_service)           │
    │                         │  │                         │
    │ • MQTT Subscriber       │  │ • MQTT Subscriber       │
    │ • Sensor Processing     │  │ • PyTorch Model         │
    │ • Volume Extraction     │  │ • Inference Engine      │
    │ • Data Persistence      │  │ • Window Control        │
    │ • Inference Triggering  │  │                         │
    │ • MQTT Publisher        │  │                         │
    └────────┬────────────────┘  └─────────────────────────┘
             ↓
    ┌─────────────────────────┐
    │   ClickHouse Database   │
    │   (Time-Series Store)   │
    │                         │
    │ • Sensor Readings       │
    │ • Window Actions        │
    │ • ML Predictions        │
    │ • Device Registry       │
    └────────┬────────────────┘
             ↓
    ┌─────────────────────────┐
    │   Grafana Dashboard     │
    │   (Visualization)       │
    │                         │
    │ • Real-time Monitoring  │
    │ • Historical Analytics  │
    │ • System Metrics        │
    └─────────────────────────┘
```

---

## System Components

### 1. ESP32 Devices (Edge Layer)

**Hardware:**
- ESP32 microcontroller with WiFi
- DHT22 or similar temperature/humidity sensor
- I2S MEMS microphone for audio capture
- Servo or stepper motor for window control

**Firmware Responsibilities:**
- Sensor data collection (temperature, humidity, audio)
- Audio buffering and compression
- MQTT client for bi-directional communication
- Window motor control based on received commands
- WiFi connection management

**MQTT Topics (Published):**
- `sensor/{device_id}/temperature` - Raw temperature value (float)
- `sensor/{device_id}/humidity` - Raw humidity value (float)
- `sensor/{device_id}/audio` - Base64-encoded WAV audio (JSON)

**MQTT Topics (Subscribed):**
- `window/{device_id}/control` - Window position commands (0-100%)

**Reference:** See `esp32_firmware/` directory for implementation

---

### 2. Go Backend Service (mqtt_backbone)

**Architecture:** Channel-based with layered separation of concerns

**Responsibilities:**
- **MQTT Transport Layer**: Subscribe to sensor data, publish inference requests
- **Sensor Processing**: Extract sound volume (dB) from audio data
- **Data Persistence**: Store all sensor readings to ClickHouse
- **Inference Coordination**: Smart triggering based on sensor changes
- **Device Management**: Auto-registration and health monitoring

**Key Features:**
- **Sound Volume Extraction**: Converts audio to dB using RMS calculation
- **Smart Triggering**: First inference requires all 3 sensors, then uses latest values
- **Rate Limiting**: Maximum one inference per 5 seconds per device
- **Concurrent Processing**: Go channels enable parallel data flow

**Technology Stack:**
- Go 1.21+
- paho.mqtt.golang (MQTT client)
- clickhouse-go/v2 (database driver)

**Reference:** See `mqtt_backbone/SPEC.md` for detailed architecture

---

### 3. Python ML Service (ml_service)

**Purpose:** Run PyTorch machine learning model for window position prediction

**Responsibilities:**
- Subscribe to inference request topics from Go backend
- Load and execute PyTorch regression model
- Predict optimal window position (0-100%) based on:
  - Temperature (°C)
  - Humidity (%)
  - Sound Volume (dB)
- Publish window control commands to ESP32 devices
- Track model performance and confidence scores

**Model Specifications:**
- **Type**: PyTorch regression model
- **Input**: 3 features (temp, humidity, sound volume)
- **Output**: Continuous value 0-100 (window position %)
- **Confidence**: Model uncertainty metric (0-1)

**Technology Stack:**
- Python 3.9+
- PyTorch
- paho-mqtt
- numpy

**Reference:** See `ml_service/` directory for implementation

---

### 4. Infrastructure Components

#### ClickHouse Database
- **Purpose**: Time-series data storage
- **Tables**:
  - `sensor_temperature` - Temperature readings
  - `sensor_humidity` - Humidity readings
  - `sensor_audio` - Audio metadata (hash, duration, sample rate)
  - `window_actions` - ML-driven window positions
  - `device_registry` - Device tracking
  - `ml_predictions` - Model performance metrics

#### Mosquitto MQTT Broker
- **Purpose**: Message bus for all components
- **Features**:
  - QoS 1 for reliable delivery
  - Topic-based routing
  - Wildcard subscriptions

#### Grafana
- **Purpose**: Real-time visualization and monitoring
- **Dashboards**:
  - System Overview (all devices)
  - Device Detail (per-device analytics)
  - ML Metrics (model performance)
  - Audio Analysis (sound levels over time)

---

## Communication Protocols

### MQTT Topic Structure

#### Sensor Data (ESP32 → Backend)
```
sensor/{device_id}/temperature   → "25.5"
sensor/{device_id}/humidity      → "60.0"
sensor/{device_id}/audio         → {"data": "base64...", "sample_rate": 16000, "duration": 2.0}
```

#### ML Inference (Backend → ML Service)
```
ml/inference/request/{device_id} → {
  "device_id": "sensor-001",
  "timestamp": "2025-10-24T12:00:00Z",
  "temperature": 25.5,
  "humidity": 60.0,
  "sound_volume": 65.5
}
```

#### Window Control (ML Service → ESP32 & Backend)
```
window/{device_id}/control → {
  "device_id": "sensor-001",
  "timestamp": "2025-10-24T12:00:01Z",
  "position": 75.5,
  "confidence": 0.92,
  "features_used": {
    "temperature": 25.5,
    "humidity": 60.0,
    "sound_volume": 65.5
  }
}
```

---

## Data Flow

### Normal Operation Sequence

1. **Sensor Reading** (ESP32)
   - ESP32 reads temperature, humidity, captures audio
   - Publishes to respective MQTT topics

2. **Data Ingestion** (Go Backend)
   - MQTT subscriber receives sensor data
   - Writes to processing channels

3. **Processing** (Go Backend)
   - Sensor service extracts sound volume from audio
   - Persists all data to ClickHouse
   - Forwards to inference service

4. **Inference Trigger** (Go Backend)
   - Inference service checks if thresholds met
   - Buffers latest values (temp, humidity, volume)
   - Publishes inference request if triggered

5. **ML Prediction** (Python Service)
   - Receives inference request
   - Runs PyTorch model
   - Publishes window control command

6. **Actuation** (ESP32 + Backend)
   - ESP32 receives window control, moves motor
   - Backend logs action to ClickHouse

7. **Visualization** (Grafana)
   - Queries ClickHouse for real-time display
   - Shows sensor trends, window positions, ML metrics

---

## Deployment Architecture

### Docker Compose Stack

```yaml
services:
  mosquitto:
    image: eclipse-mosquitto
    ports: ["1883:1883"]

  clickhouse:
    image: clickhouse/clickhouse-server
    ports: ["9000:9000", "8123:8123"]

  mqtt_backbone:
    build: ./mqtt_backbone
    depends_on: [mosquitto, clickhouse]

  ml_service:
    build: ./ml_service
    depends_on: [mosquitto]

  grafana:
    image: grafana/grafana
    ports: ["3000:3000"]
    depends_on: [clickhouse]
```

### Network Architecture
- All services on same Docker network
- ESP32 devices connect via WiFi to Mosquitto
- Grafana accessible at `http://localhost:3000`

---

## Technology Stack Summary

| Component | Technologies |
|-----------|-------------|
| **ESP32 Firmware** | C++, Arduino/ESP-IDF, MQTT, WiFi |
| **Go Backend** | Go 1.21+, paho.mqtt.golang, clickhouse-go |
| **ML Service** | Python 3.9+, PyTorch, paho-mqtt |
| **Database** | ClickHouse (time-series) |
| **Message Bus** | Mosquitto MQTT Broker |
| **Visualization** | Grafana |
| **Deployment** | Docker, Docker Compose |

---

## Configuration

### Environment Variables

**mqtt_backbone:**
```bash
MQTT_BROKER=tcp://mosquitto:1883
CLICKHOUSE_ADDR=clickhouse:9000
CLICKHOUSE_DB=iot
TEMPERATURE_THRESHOLD=0.5
HUMIDITY_THRESHOLD=2.0
```

**ml_service:**
```bash
MQTT_BROKER=tcp://mosquitto:1883
MODEL_PATH=/models/window_regressor.pth
MODEL_VERSION=v1.0.0
```

---

## Multi-Device Support

**Scale:** 2-10 devices initially, designed for horizontal scaling

**Device Identification:**
- Unique `device_id` per ESP32 (e.g., "sensor-001")
- Device auto-registration on first message
- Per-device state tracking
- Independent inference per device

**Load Characteristics:**
- ~10 sensor readings/second per device
- ~1 inference request per 5 seconds per device
- Audio: 2-second buffers at ~30KB each

---

## Key Architecture Principles

### 1. Sound-Based Intelligence
- Audio converted to sound volume (dB) at ingestion
- No raw audio storage (metadata only)
- ML model uses volume as input feature
- Formula: `20 * log10(RMS / 32768.0)`

### 2. Smart Inference Triggering
- **First inference**: Requires all 3 sensors available
- **Subsequent**: Always uses most recent values
- **Triggers**: Volume always triggers; temp/humidity on threshold
- **Rate limiting**: Max 1 per 5 seconds per device

### 3. Separation of Concerns
- **Edge**: Sensing and actuation (ESP32)
- **Transport**: Message routing (MQTT)
- **Processing**: Data transformation (Go backend)
- **Intelligence**: Decision making (Python ML)
- **Storage**: Historical data (ClickHouse)
- **Visualization**: Monitoring (Grafana)

### 4. Channel-Based Architecture (Go Backend)
- MQTT layer: Pure transport
- Services layer: Business logic
- Database layer: Persistence
- Go channels: Decoupled communication

---

## Component References

For detailed specifications of each component:

- **Go Backend**: `mqtt_backbone/SPEC.md`
- **Python ML Service**: `ml_service/README.md` (to be created)
- **ESP32 Firmware**: `esp32_firmware/README.md` (to be created)
- **Deployment**: `docker-compose.yml`
- **Implementation Plan**: `mqtt_backbone/PLAN.md`

---

## Version History

- **v1.5** (Current): Channel-based architecture, sound volume inference
- **v1.0**: Initial implementation with callback-based architecture

---

## Future Enhancements

- [ ] Multi-room support with room-level aggregation
- [ ] Mobile app for manual control and monitoring
- [ ] Advanced audio features (speech detection, noise classification)
- [ ] Adaptive learning (model retraining based on user feedback)
- [ ] Energy optimization (predict heating/cooling costs)
- [ ] Weather API integration for predictive control
