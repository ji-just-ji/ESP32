# ML Service

PyTorch-based machine learning microservice for window control predictions in the ESP32 IoT system.

## Overview

The ML service receives inference requests from the Go backend via MQTT, runs predictions using a PyTorch neural network, and publishes window control commands back via MQTT.

### Key Features

- **Percentile-based Normalization**: Uses 0.1 and 0.9 percentiles from training data for robust feature scaling
- **PyTorch Inference**: Simple neural network (3 inputs → 1 output) for window position prediction
- **MQTT Communication**: Reliable QoS 1 messaging with automatic reconnection
- **Confidence Scoring**: Only publishes predictions above configurable confidence threshold
- **Docker Ready**: Containerized deployment with `uv` for fast dependency management

## Architecture

```
MQTT Inference Request
        ↓
   MQTT Client
        ↓
   Feature Processor (Normalize with percentiles)
        ↓
   Model Loader (PyTorch)
        ↓
   Predictor (Run inference + confidence)
        ↓
   MQTT Client (Publish window control)
```

## Quick Start

### 1. Generate Initial Model

Before running the service, generate an initial model with metadata:

```bash
cd ml_service
uv run scripts/create_initial_model.py
```

This creates:
- `models/window_regressor.pth` - PyTorch model state dict
- `models/window_regressor.json` - Metadata with percentiles
- `models/window_regressor_checkpoint.pth` - Combined checkpoint (optional)

### 2. Install Dependencies

Using `uv` (recommended):

```bash
uv sync
```

Or with pip:

```bash
pip install -e .
```

### 3. Run Locally

```bash
uv run python -m src.main
```

Or with Docker:

```bash
docker build -t ml-service .
docker run --rm --network iot-network ml-service
```

### 4. Run with Docker Compose

From the `mqtt_backbone` directory:

```bash
cd ../mqtt_backbone
docker-compose up ml_service
```

## Configuration

Edit `config.yaml`:

```yaml
mqtt:
  broker: "mosquitto"  # MQTT broker hostname
  port: 1883
  topics:
    inference_request: "ml/inference/request/#"
    window_control: "window/{device_id}/control"

model:
  path: "/app/models/window_regressor.pth"
  version: "v1.0.0"

inference:
  percentile_low: 0.1   # Lower percentile for normalization
  percentile_high: 0.9  # Upper percentile for normalization
  min_confidence: 0.0   # Minimum confidence to publish (0 = always)

logging:
  level: "INFO"
  format: "json"  # or "text"
```

## MQTT Topics

### Subscribed Topics

- `ml/inference/request/#` - Inference requests from Go backend

**Payload**:
```json
{
  "device_id": "sensor-001",
  "timestamp": "2025-10-27T12:00:00Z",
  "temperature": 25.5,
  "humidity": 60.0,
  "sound_volume": 65.5
}
```

### Published Topics

- `window/{device_id}/control` - Window control commands

**Payload**:
```json
{
  "device_id": "sensor-001",
  "timestamp": "2025-10-27T12:00:01Z",
  "position": 75.5,
  "confidence": 0.92,
  "features_used": {
    "temperature": 25.5,
    "humidity": 60.0,
    "sound_volume": 65.5
  }
}
```

## Model Training

The included model is a **dummy model for testing**. To use in production:

1. **Collect Training Data**: Gather real sensor data and corresponding window positions
2. **Compute Percentiles**: Calculate 0.1 and 0.9 percentiles for each feature
3. **Train Model**: Train the `WindowRegressorModel` on your data
4. **Save with Metadata**: Save model state dict + metadata JSON with percentiles

Example training script structure:

```python
# 1. Load training data
temperature, humidity, sound_volume, window_position = load_data()

# 2. Compute percentiles
percentiles = {
    'temperature': [np.percentile(temperature, 10), np.percentile(temperature, 90)],
    'humidity': [np.percentile(humidity, 10), np.percentile(humidity, 90)],
    'sound_volume': [np.percentile(sound_volume, 10), np.percentile(sound_volume, 90)]
}

# 3. Train model (normalize using percentiles)
model = WindowRegressorModel()
# ... training loop ...

# 4. Save model + metadata
torch.save(model.state_dict(), 'window_regressor.pth')
with open('window_regressor.json', 'w') as f:
    json.dump({'version': 'v1.0.0', 'percentiles': percentiles}, f)
```

## Development

### Project Structure

```
ml_service/
├── src/
│   ├── __init__.py
│   ├── main.py              # Entry point & orchestration
│   ├── mqtt_client.py       # MQTT communication
│   ├── model_loader.py      # PyTorch model loading
│   ├── predictor.py         # Inference logic
│   └── feature_processor.py # Feature normalization
├── models/
│   ├── .gitkeep
│   ├── window_regressor.pth  # Model weights (generated)
│   └── window_regressor.json # Metadata (generated)
├── scripts/
│   └── create_initial_model.py
├── tests/
│   └── __init__.py
├── config.yaml
├── pyproject.toml
├── Dockerfile
└── README.md
```

### Testing

```bash
# Run tests (when implemented)
uv run pytest

# Test MQTT connection
mosquitto_sub -h localhost -t "window/#" -v

# Publish test inference request
mosquitto_pub -h localhost -t "ml/inference/request/sensor-001" \
  -m '{"device_id":"sensor-001","timestamp":"2025-10-27T12:00:00Z","temperature":25.5,"humidity":60.0,"sound_volume":65.5}'
```

### Environment Variables

- `ML_SERVICE_CONFIG` - Path to config.yaml (default: `./config.yaml`)

## Troubleshooting

### Model not found

```
FileNotFoundError: Model file not found: /app/models/window_regressor.pth
```

**Solution**: Run `uv run scripts/create_initial_model.py` to generate the initial model.

### MQTT connection refused

```
ConnectionError: Failed to connect to MQTT broker
```

**Solution**: Ensure Mosquitto is running and accessible at the configured broker address.

### Percentile warnings

```
WARNING - temperature=40.00 outside training range [10.50, 34.80]
```

**Solution**: This is expected for values outside the 0.1-0.9 percentile range. The feature processor clips values to [0, 1] automatically. Update model training data if this occurs frequently.

## Performance

- **Inference Latency**: < 10ms per prediction (CPU)
- **Memory Usage**: ~200MB (including PyTorch)
- **Throughput**: 100+ inferences/second

## Future Improvements

- [ ] Implement proper model training pipeline
- [ ] Add model versioning and A/B testing
- [ ] Implement uncertainty estimation (MC Dropout, ensembles)
- [ ] Add online learning / incremental training
- [ ] Metrics export (Prometheus)
- [ ] Add comprehensive unit tests
- [ ] Support GPU inference
- [ ] Model performance monitoring

## References

- **Project Spec**: `../SPEC.md`
- **Implementation Plan**: `../PLAN.md`
- **Go Backend**: `../mqtt_backbone/`
