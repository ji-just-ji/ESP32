# IoT Backend Service

A Golang backend service for IoT environmental monitoring and automated window control.

## Features

- **MQTT Integration**: Subscribes to sensor data and publishes window control actions
- **Real-time Processing**: Continuous monitoring of temperature, humidity, and sound levels
- **ClickHouse Storage**: Efficient time-series data storage for sensor readings and actions
- **ML-based Decision Making**: Uses a regression model to determine optimal window control

## Architecture

```
MQTT Broker (Sensor Data)
    ↓
Backend Service
    ├─> ClickHouse (Data Storage)
    ├─> ML Model (Prediction)
    └─> MQTT Broker (Control Actions)
```

## Prerequisites

1. **Go 1.21+** - [Install Go](https://golang.org/doc/install)
2. **ClickHouse** - [Install ClickHouse](https://clickhouse.com/docs/en/install)
3. **MQTT Broker** (e.g., Mosquitto) - [Install Mosquitto](https://mosquitto.org/download/)

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

Edit the `.env` file with your settings:

- **MQTT Settings**: Configure broker address, credentials, and topics
- **ClickHouse Settings**: Database connection details
- **Model Path**: Path to your regression model file

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

## Data Models

### Sensor Reading (Input)

```json
{
  "timestamp": "2025-10-24T12:00:00Z",
  "device_id": "sensor-001",
  "temperature": 25.5,
  "humidity": 60.0,
  "sound": 45.2
}
```

### Window Action (Output)

```json
{
  "timestamp": "2025-10-24T12:00:01Z",
  "device_id": "sensor-001",
  "action": "open",
  "temperature": 25.5,
  "humidity": 60.0,
  "sound": 45.2
}
```

## ML Model

The service uses a linear regression model stored in JSON format:

```json
{
  "coefficients": {
    "temperature": 0.3,
    "humidity": -0.2,
    "sound": -0.15
  },
  "intercept": 0.0,
  "threshold": 5.0
}
```

A sample model will be created automatically on first run if none exists.

### Training Your Own Model

To train a custom model, you can:

1. Collect historical sensor data from ClickHouse
2. Train a regression model using Python/scikit-learn
3. Export the model coefficients to the JSON format shown above
4. Update `MODEL_PATH` in `.env` to point to your model

## Database Schema

### sensor_readings table

- `timestamp`: DateTime64(3)
- `device_id`: String
- `temperature`: Float64
- `humidity`: Float64
- `sound`: Float64

### window_actions table

- `timestamp`: DateTime64(3)
- `device_id`: String
- `action`: String
- `temperature`: Float64
- `humidity`: Float64
- `sound`: Float64

## Project Structure

```
backend/
├── cmd/
│   └── server/          # Main application entry point
├── internal/
│   ├── mqtt/            # MQTT client implementation
│   ├── database/        # ClickHouse database client
│   ├── models/          # Data models
│   └── ml/              # ML predictor
├── pkg/
│   └── config/          # Configuration management
├── model/               # ML model files
├── .env.example         # Example environment variables
├── go.mod               # Go module definition
└── README.md
```

## Testing MQTT

### Publishing sensor data:

```bash
mosquitto_pub -h localhost -t "sensor/data" -m '{
  "device_id": "sensor-001",
  "temperature": 28.5,
  "humidity": 65.0,
  "sound": 42.3
}'
```

### Subscribing to window actions:

```bash
mosquitto_sub -h localhost -t "window/action"
```

## Monitoring

The service logs all operations including:
- MQTT connections and messages
- Database operations
- ML predictions and decisions
- Window control actions

## License

MIT
