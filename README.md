# ESP32 Smart Home IoT System

An intelligent IoT system for automated smart home control using ESP32 microcontrollers, machine learning, and real-time environmental monitoring.

## Overview

This project provides a comprehensive smart home automation solution that monitors ambient conditions (temperature, humidity, and noise levels) and automatically controls home systems for optimal comfort and efficiency.

### Smart Home Features

1. **Active Noise Cancellation (ANC)** - Intelligent noise monitoring and mitigation
2. **Automated Window Control** - ML-driven window positioning based on environmental conditions
3. **Smart Blinds Automation** - Automated blind control for optimal lighting and temperature
4. **Air Conditioning Control** - Intelligent climate control based on ambient temperature and humidity

All automation decisions are driven by real-time sensor data and machine learning models that learn optimal settings based on environmental conditions.

## System Architecture

### High-Level Overview

```
┌───────────────────────────────────────────────────────────────┐
│                   ESP32 Smart Home System                     │
└───────────────────────────────────────────────────────────────┘

┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  ESP32 Sensors  │     │  ESP32 Sensors  │     │  ESP32 Sensors  │
│  Room 1         │     │  Room 2         │     │  Room N         │
│                 │     │                 │     │                 │
│ • Temperature   │     │ • Temperature   │     │ • Temperature   │
│ • Humidity      │     │ • Humidity      │     │ • Humidity      │
│ • Microphone    │     │ • Microphone    │     │ • Microphone    │
│ • Actuators     │     │ • Actuators     │     │ • Actuators     │
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │ WiFi + MQTT
                                 ↓
         ┌───────────────────────────────────────────┐
         │      Mosquitto MQTT Broker                │
         │      (Message Bus)                        │
         └───────┬───────────────────────┬───────────┘
                 │                       │
     ┌───────────┴────────┐     ┌────────┴──────────┐
     ↓                    ↓     ↓                    ↓
┌──────────────┐   ┌────────────────┐   ┌──────────────────┐
│ Go Backend   │   │ Python ML      │   │ ClickHouse DB    │
│ Service      │   │ Service        │   │ (Time-Series)    │
│              │   │                │   │                  │
│ • Data       │   │ • PyTorch      │   │ • Sensor History │
│   Processing │   │   Inference    │   │ • Analytics      │
│ • Sound      │   │ • Decision     │   │ • Device Logs    │
│   Analysis   │   │   Making       │   │                  │
└──────┬───────┘   └────────────────┘   └────────┬─────────┘
       │                                          │
       └──────────────────────────────────────────┘
                            ↓
                   ┌─────────────────┐
                   │ Grafana         │
                   │ Dashboards      │
                   │                 │
                   │ • Real-time     │
                   │   Monitoring    │
                   │ • Analytics     │
                   │ • Alerts        │
                   └─────────────────┘
```

### How It Works

1. **Sensing**: ESP32 devices continuously monitor:
   - **Temperature** (DHT22/BME280 sensors)
   - **Humidity** (DHT22/BME280 sensors)
   - **Ambient Noise** (I2S MEMS microphone)

2. **Communication**: Sensor data flows through MQTT topics to the backend services

3. **Processing**:
   - Go backend extracts sound volume (dB) from audio streams
   - Data is persisted to ClickHouse for historical analysis
   - Smart triggering logic decides when to request ML inference

4. **Intelligence**: Python ML service uses PyTorch to predict optimal settings:
   - Window position (0-100%)
   - Blind position (planned)
   - AC settings (planned)

5. **Actuation**: Control commands are sent back to ESP32 devices to:
   - Adjust window positions via servo/stepper motors
   - Control blinds (planned)
   - Adjust air conditioning (planned)

6. **Visualization**: Grafana provides real-time dashboards for monitoring and analytics

## Key Features

### Multi-Device Support
- 2-10 ESP32 devices per deployment
- Auto-registration and health monitoring
- Per-device configuration and control
- Scalable architecture for future expansion

### Sound-Based Intelligence
- Audio capture and volume extraction (dB)
- RMS-based sound level calculation
- No raw audio storage (privacy-first)
- Used for noise-aware automation decisions

### Machine Learning Control
- PyTorch-based regression models
- Multi-factor decision making (temp, humidity, noise)
- Confidence scoring for reliable predictions
- Real-time inference (<2 seconds)

### Real-Time Analytics
- Historical data storage in ClickHouse
- Grafana dashboards for visualization
- System health monitoring
- ML performance tracking

## Technology Stack

| Layer | Technology |
|-------|-----------|
| **Edge Devices** | ESP32, C++, Arduino/ESP-IDF |
| **Sensors** | DHT22/BME280, I2S MEMS Microphone |
| **Actuators** | Servo Motors, Stepper Motors |
| **Communication** | MQTT (Mosquitto), WiFi |
| **Backend** | Go 1.21+, Channels |
| **ML Service** | Python 3.9+, PyTorch |
| **Database** | ClickHouse (Time-Series) |
| **Visualization** | Grafana |
| **Deployment** | Docker, Docker Compose |

## Quick Start

### Prerequisites

- Docker & Docker Compose
- (Optional) ESP32 development environment (PlatformIO/Arduino IDE)
- (Optional) Go 1.21+ for local development
- (Optional) Python 3.9+ for ML service development

### 1. Clone the Repository

```bash
git clone <repository-url>
cd ESP32
```

### 2. Start the Backend Services

```bash
cd mqtt_backbone
docker-compose up -d
```

This starts:
- Mosquitto MQTT Broker (port 1883)
- ClickHouse Database (ports 9000, 8123)
- Go Backend Service
- Python ML Service
- Grafana (port 3000)

### 3. Access Grafana

Open your browser to [http://localhost:3000](http://localhost:3000)

- **Username**: `admin`
- **Password**: `admin`

View dashboards under **ESP32 IoT System** folder.

### 4. Connect ESP32 Devices

Flash the ESP32 firmware (see `esp32_firmware/`) or use MQTT simulators for testing:

```bash
# Publish test sensor data
mosquitto_pub -h localhost -t "sensor/sensor-001/temperature" -m '{"value": 25.5}'
mosquitto_pub -h localhost -t "sensor/sensor-001/humidity" -m '{"value": 60.0}'

# Subscribe to window control commands
mosquitto_sub -h localhost -t "window/#"
```

## Project Structure

```
ESP32/
├── README.md                    # This file
├── SPEC.md                      # Detailed system specification
├── PLAN.md                      # Implementation plan & status
├── mqtt_backbone/               # Go backend service
│   ├── cmd/                     # Main application
│   ├── internal/                # Core business logic
│   │   ├── mqtt/                # MQTT transport layer
│   │   ├── services/            # Sensor & inference services
│   │   └── database/            # ClickHouse persistence
│   ├── docker-compose.yml       # Service orchestration
│   └── README.md                # Backend documentation
├── ml_service/                  # Python ML service
│   ├── src/                     # ML inference code
│   ├── models/                  # PyTorch models
│   ├── scripts/                 # Training & utilities
│   └── README.md                # ML service documentation
├── grafana/                     # Visualization
│   ├── dashboards/              # Pre-built dashboards
│   ├── provisioning/            # Auto-configuration
│   └── README.md                # Grafana setup guide
└── esp32_firmware/              # ESP32 firmware (planned)
    ├── src/                     # Firmware code
    └── README.md                # Firmware documentation
```

## Current Status

### Completed Components

| Component | Status | Description |
|-----------|--------|-------------|
| **Infrastructure** | ✅ Complete | Docker Compose, MQTT broker, ClickHouse |
| **Go Backend** | ✅ Complete | Channel-based architecture, sensor processing |
| **ML Service** | ✅ Complete | PyTorch inference, MQTT integration |
| **Grafana** | ✅ Complete | 4 dashboards with real-time monitoring |
| **Documentation** | ✅ Complete | Comprehensive specs and guides |

### In Progress

| Component | Status | Description |
|-----------|--------|-------------|
| **ESP32 Firmware** | 📋 Planned | Sensor drivers, motor control, MQTT client |
| **Integration Testing** | 📋 Planned | End-to-end testing with hardware |
| **Blinds Control** | 📋 Planned | Automated blind positioning |
| **AC Control** | 📋 Planned | Smart air conditioning automation |
| **ANC Features** | 📋 Planned | Active noise cancellation logic |

See [PLAN.md](PLAN.md) for detailed implementation timeline.

## MQTT Topics Reference

### Sensor Data (ESP32 → Backend)

```
sensor/{device_id}/temperature    - Temperature readings (°C)
sensor/{device_id}/humidity       - Humidity readings (%)
sensor/{device_id}/audio          - Audio samples (Base64 WAV)
```

### ML Inference (Backend ↔ ML Service)

```
ml/inference/request/{device_id}  - Inference requests
window/{device_id}/control        - Window control commands
```

### Future Topics (Planned)

```
blind/{device_id}/control         - Blind control commands
ac/{device_id}/control            - Air conditioning commands
anc/{device_id}/control           - ANC system commands
```

## Configuration

### Environment Variables

Key configuration options (see component READMEs for full details):

**MQTT:**
- `MQTT_BROKER`: Broker address (default: `tcp://mosquitto:1883`)

**ClickHouse:**
- `CLICKHOUSE_ADDR`: Database address (default: `clickhouse:9000`)
- `CLICKHOUSE_DB`: Database name (default: `iot`)

**ML Service:**
- `MODEL_PATH`: Path to PyTorch model (default: `/app/models/window_regressor.pth`)

**Thresholds:**
- `TEMPERATURE_THRESHOLD`: Trigger threshold in °C (default: `0.5`)
- `HUMIDITY_THRESHOLD`: Trigger threshold in % (default: `2.0`)

## Data Flow Example

1. **ESP32** reads temperature (25.5°C), humidity (60%), captures audio
2. **ESP32** publishes to MQTT topics: `sensor/room-1/temperature`, etc.
3. **Go Backend** receives data, extracts sound volume (65.5 dB)
4. **Go Backend** stores data in ClickHouse for history
5. **Go Backend** detects significant change, publishes inference request
6. **ML Service** receives request, runs PyTorch model
7. **ML Service** predicts window position (75%), confidence (0.92)
8. **ML Service** publishes control command to `window/room-1/control`
9. **ESP32** receives command, adjusts window motor to 75% open
10. **Grafana** displays real-time updates on dashboards

## Smart Home Use Cases

### Scenario 1: Hot Summer Day
- **Input**: High temperature (30°C), low humidity (40%), moderate noise (60 dB)
- **Action**: ML model opens windows to 80% for natural ventilation
- **Future**: Could also trigger AC if temperature exceeds threshold

### Scenario 2: Noisy Street
- **Input**: Normal temperature (22°C), high noise (85 dB)
- **Action**: ML model closes windows to 20% to reduce noise
- **Future**: ANC system activates for remaining indoor noise

### Scenario 3: Rainy Weather
- **Input**: High humidity (80%), normal temperature, low noise
- **Action**: ML model closes windows to 10% to prevent rain entry
- **Future**: Closes blinds to prevent water damage

### Scenario 4: Sleep Time
- **Input**: Low temperature (20°C), low noise (45 dB)
- **Action**: Partial window opening (40%) for fresh air
- **Future**: AC maintains comfortable temperature, blinds close completely

## Monitoring & Analytics

### Grafana Dashboards

Access at [http://localhost:3000](http://localhost:3000)

1. **System Overview** - All devices at a glance
2. **Device Detail** - Per-device deep dive
3. **ML Metrics** - Model performance and confidence
4. **System Health** - Service status and device monitoring

### ClickHouse Queries

Connect to ClickHouse for custom analytics:

```bash
docker exec -it iot-clickhouse clickhouse-client

USE iot;
SELECT device_id, avg(value) as avg_temp
FROM sensor_temperature
WHERE timestamp > now() - INTERVAL 1 HOUR
GROUP BY device_id;
```

## Development

### Running Components Locally

**Go Backend:**
```bash
cd mqtt_backbone
go run cmd/server/main.go
```

**Python ML Service:**
```bash
cd ml_service
uv run python -m src.main
```

### Testing with MQTT

```bash
# Subscribe to all topics
mosquitto_sub -h localhost -t "#" -v

# Publish test data
mosquitto_pub -h localhost -t "sensor/test/temperature" -m '{"value": 23.0}'
```

## Documentation

- [SPEC.md](SPEC.md) - Detailed system architecture and specifications
- [PLAN.md](PLAN.md) - Implementation plan and roadmap
- [mqtt_backbone/README.md](mqtt_backbone/README.md) - Go backend documentation
- [ml_service/README.md](ml_service/README.md) - ML service documentation
- [grafana/README.md](grafana/README.md) - Grafana setup and dashboards

## Roadmap

### Phase 1: Core Infrastructure (✅ Complete)
- MQTT backbone with Go backend
- ML inference service
- Database and visualization

### Phase 2: ESP32 Integration (📋 Planned)
- Firmware development
- Sensor integration
- Motor control implementation

### Phase 3: Extended Automation (📋 Planned)
- Smart blinds control
- Air conditioning automation
- Multi-room coordination

### Phase 4: Advanced Features (📋 Future)
- Active noise cancellation
- Predictive control (weather API integration)
- Mobile app for remote control
- Voice assistant integration

## Troubleshooting

### Services won't start
```bash
docker-compose down -v
docker-compose up -d
docker-compose logs -f
```

### No data in Grafana
1. Check if backend services are running: `docker ps`
2. Verify MQTT messages: `mosquitto_sub -h localhost -t "#"`
3. Check ClickHouse has data: `docker exec -it iot-clickhouse clickhouse-client`

### ESP32 won't connect
1. Verify WiFi credentials in firmware
2. Check MQTT broker is accessible from ESP32 network
3. Monitor MQTT broker logs: `docker logs iot-mosquitto`

## Contributing

Contributions are welcome! Please read the [SPEC.md](SPEC.md) to understand the architecture before making changes.

## License

[Specify your license here]

## Support

For issues, questions, or feature requests:
1. Check the troubleshooting section above
2. Review component-specific README files
3. Check system logs: `docker-compose logs`
4. Open an issue on the repository

## Acknowledgments

Built with:
- ESP32 microcontrollers
- Go, Python, PyTorch
- MQTT, ClickHouse, Grafana
- Docker and open-source software

---

**Current Version:** v1.5
**Last Updated:** 2025-10-27
**Project Status:** Core infrastructure complete, ESP32 integration in progress
