# ML Service Specification

## Overview

The ML Service is a Python microservice that provides machine learning inference for automated window control decisions in the ESP32 IoT ecosystem. It receives sensor data (temperature, humidity, sound volume) via MQTT, runs predictions using an XGBoost regression model, and publishes window position commands.

### Key Features

- **XGBoost-based Regression**: Lightweight, fast gradient boosting model
- **ClickHouse Training Pipeline**: Automatically trains on historical window actuation data
- **Percentile-based Normalization**: Robust feature scaling using 0.1 and 0.9 percentiles
- **MQTT Communication**: Reliable QoS 1 messaging with automatic reconnection
- **Confidence Scoring**: Only publishes predictions above configurable threshold
- **Auto-training**: Automatically trains model on startup if no model exists
- **Environment-based Configuration**: All configuration via root `.env.config`

---

## Architecture

### System Context

```
┌─────────────────────────────────────────────────────────────┐
│                    ESP32 IoT Ecosystem                       │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                  Go Backend (mqtt_backbone)                  │
│  • Collects sensor data (temp, humidity, audio)             │
│  • Extracts sound volume (dB) from audio                    │
│  • Persists to ClickHouse                                   │
│  • Triggers inference requests                              │
└────────────────────┬────────────────────────────────────────┘
                     ↓ MQTT: ml/inference/request
┌─────────────────────────────────────────────────────────────┐
│                    ML Service (This Service)                 │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ MQTT Client                                         │   │
│  │  - Subscribe: MQTT_TOPIC_INFERENCE_REQ              │   │
│  │  - Publish: MQTT_TOPIC_WINDOW_CONTROL               │   │
│  └────────┬────────────────────────────────────────────┘   │
│           ↓                                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Feature Processor                                   │   │
│  │  - Normalize using percentiles                      │   │
│  │  - Validate input ranges                            │   │
│  └────────┬────────────────────────────────────────────┘   │
│           ↓                                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Model Loader                                        │   │
│  │  - Load XGBoost Booster                             │   │
│  │  - Load percentile metadata                         │   │
│  └────────┬────────────────────────────────────────────┘   │
│           ↓                                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Predictor                                           │   │
│  │  - Run XGBoost inference                            │   │
│  │  - Calculate confidence score                       │   │
│  │  - Denormalize output                               │   │
│  └────────┬────────────────────────────────────────────┘   │
│           ↓                                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ MQTT Client                                         │   │
│  │  - Publish window control command                   │   │
│  └─────────────────────────────────────────────────────┘   │
└────────────────────┬────────────────────────────────────────┘
                     ↓ MQTT: window/control
┌─────────────────────────────────────────────────────────────┐
│              Go Backend + ESP32 Devices                      │
│  • Log window action to ClickHouse                          │
│  • Forward to ESP32 for motor control                       │
└─────────────────────────────────────────────────────────────┘
```

### Training Pipeline

```
┌─────────────────────────────────────────────────────────────┐
│                      ClickHouse Database                     │
│  Table: window_actions                                       │
│    - timestamp, device_id, position                         │
│    - temperature, humidity, sound_volume                    │
└────────────────────┬────────────────────────────────────────┘
                     ↓ Query historical data
┌─────────────────────────────────────────────────────────────┐
│               Training Script (train_from_clickhouse.py)     │
│  1. Query window_actions (last N days)                      │
│  2. Extract features: [temp, humidity, sound_volume]        │
│  3. Extract target: position                                │
│  4. Compute percentiles (p10, p90) for normalization        │
│  5. Normalize features                                      │
│  6. Train XGBoost regressor                                 │
│  7. Save model + metadata                                   │
└────────────────────┬────────────────────────────────────────┘
                     ↓ Save to disk
┌─────────────────────────────────────────────────────────────┐
│                    Model Artifacts                           │
│  • models/window_regressor.json (XGBoost model)             │
│  • models/window_regressor.json.meta (percentiles)          │
└─────────────────────────────────────────────────────────────┘
```

---

## Component Specifications

### 1. MQTT Client (`src/mqtt_client.py`)

**Responsibilities:**
- Connect to MQTT broker using credentials from `.env.config`
- Subscribe to inference request topic
- Parse and validate incoming messages
- Publish window control commands
- Handle reconnection and errors

**Topics:**
- **Subscribe**: `MQTT_TOPIC_INFERENCE_REQ` (default: `ml/inference/request`)
- **Publish**: `MQTT_TOPIC_WINDOW_CONTROL` (default: `window/control`)

**Message Formats:**

*Inference Request (Received):*
```json
{
  "device_id": "sensor-001",
  "timestamp": "2025-10-31T12:00:00Z",
  "temperature": 25.5,
  "humidity": 60.0,
  "sound_volume": 65.5
}
```

*Window Control (Published):*
```json
{
  "device_id": "sensor-001",
  "timestamp": "2025-10-31T12:00:01Z",
  "position": 75.5,
  "confidence": 0.92,
  "features_used": {
    "temperature": 25.5,
    "humidity": 60.0,
    "sound_volume": 65.5
  }
}
```

**Configuration (from .env.config):**
- `MQTT_BROKER`: Broker URL (e.g., `tcp://localhost:1883`)
- `ML_SERVICE_CLIENT_ID`: MQTT client identifier
- `MQTT_TOPIC_INFERENCE_REQ`: Inference request topic
- `MQTT_TOPIC_WINDOW_CONTROL`: Window control topic

---

### 2. Configuration Module (`src/config.py`)

**Responsibilities:**
- Load `.env.config` from project root
- Parse and validate all configuration
- Provide typed configuration object

**Configuration Groups:**

**MQTT Settings:**
- Broker URL, client ID
- Inference request topic, window control topic
- QoS, keepalive, reconnect delay

**Model Settings:**
- Model path, version
- Output range (min/max)

**Inference Settings:**
- Minimum confidence threshold
- Percentile ranges (low/high)

**Training Settings:**
- Minimum samples required
- Test split ratio
- Auto-train on startup flag

**XGBoost Hyperparameters:**
- max_depth, learning_rate, n_estimators, subsample

**ClickHouse Settings:**
- Address, database, username, password

**Logging:**
- Log level, log format

---

### 3. Model Loader (`src/model_loader.py`)

**Responsibilities:**
- Load XGBoost model from JSON file
- Load percentile metadata from separate file
- Validate model and metadata structure

**Model Format:**
- Primary: `models/window_regressor.json` (XGBoost JSON format)
- Metadata: `models/window_regressor.json.meta` (JSON)

**Metadata Structure:**
```json
{
  "version": "v1.0.0",
  "model_type": "XGBoost",
  "percentiles": {
    "temperature": [10.5, 34.8],
    "humidity": [22.0, 78.5],
    "sound_volume": [45.2, 75.6]
  },
  "training_info": {
    "date": "2025-10-31T12:00:00Z",
    "samples": 5000,
    "rmse": 5.2
  }
}
```

**Methods:**
- `load()`: Load model and metadata
- `get_model()`: Return XGBoost Booster
- `get_metadata()`: Return metadata dict
- `get_percentiles()`: Return percentile dict

---

### 4. Feature Processor (`src/feature_processor.py`)

**Responsibilities:**
- Normalize input features using percentiles
- Denormalize model output
- Validate input ranges

**Normalization:**
- Method: Percentile-based scaling
- Formula: `(value - p10) / (p90 - p10)`, clipped to [0, 1]
- Percentiles computed from training data

**Features:**
- `temperature`: Celsius (typical range: 10-35°C)
- `humidity`: Percentage (range: 0-100%)
- `sound_volume`: Decibels (typical range: 40-80 dB)

**Output:**
- Normalized: [0, 1]
- Denormalized: [0, 100] (window position %)

---

### 5. Predictor (`src/predictor.py`)

**Responsibilities:**
- Run XGBoost inference
- Calculate confidence scores
- Apply confidence threshold filtering

**Inference Pipeline:**
1. Validate input features
2. Normalize features using percentiles
3. Create `xgb.DMatrix` from normalized features
4. Run `booster.predict()`
5. Calculate confidence score
6. Denormalize output to [0, 100]
7. Return prediction and publish decision

**Confidence Calculation:**
- Simple heuristic based on distance from boundaries
- Predictions near middle (50%) more confident than extremes (0%, 100%)
- Formula: `confidence = 0.5 + min(position, 1.0 - position)`
- Future: Implement proper uncertainty estimation (quantile regression, ensemble)

**Confidence Threshold:**
- Configurable via `ML_MIN_CONFIDENCE`
- Default: 0.0 (always publish)
- Predictions below threshold are logged but not published

---

### 6. Training Pipeline (`scripts/train_from_clickhouse.py`)

**Responsibilities:**
- Connect to ClickHouse database
- Query historical window actuation data
- Compute feature percentiles
- Train XGBoost model
- Save model and metadata

**Data Source:**
- Table: `window_actions`
- Features: `temperature`, `humidity`, `sound_volume`
- Target: `position` (0-100%)

**Training Process:**
1. Query last N days of data from ClickHouse
2. Filter valid samples (non-null, in valid ranges)
3. Check minimum sample requirement (default: 100)
4. Split into train/test sets
5. Compute percentiles (p10, p90) from training set
6. Normalize features
7. Train XGBoost with hyperparameters from config
8. Evaluate on test set (RMSE)
9. Save model as JSON
10. Save metadata with percentiles and metrics

**Fallback Strategy:**
- If insufficient data (<100 samples): Use synthetic data
- If ClickHouse unavailable: Use synthetic data
- Synthetic data generator: `create_initial_model.py`

**Hyperparameters (from .env.config):**
- `XGBOOST_MAX_DEPTH`: Tree depth (default: 6)
- `XGBOOST_LEARNING_RATE`: Learning rate (default: 0.1)
- `XGBOOST_N_ESTIMATORS`: Number of trees (default: 100)
- `XGBOOST_SUBSAMPLE`: Row sampling ratio (default: 0.8)

---

### 7. Main Application (`src/main.py`)

**Responsibilities:**
- Load configuration from `.env.config`
- Initialize logging
- Check for model existence
- Auto-train if no model exists (when enabled)
- Initialize MQTT client
- Load model and predictor
- Start inference loop
- Handle graceful shutdown

**Startup Sequence:**
1. Load `.env.config` from project root
2. Setup logging (JSON or text format)
3. Check if model file exists
4. If not exists and auto-train enabled:
   - Attempt to train from ClickHouse
   - Fallback to synthetic data if needed
5. Load model and metadata
6. Initialize feature processor with percentiles
7. Initialize predictor
8. Connect to MQTT broker
9. Subscribe to inference request topic
10. Enter main loop (handle MQTT messages)

**Shutdown:**
- Graceful shutdown on SIGINT/SIGTERM
- Disconnect MQTT client
- Log final statistics

---

## Data Models

### InferenceRequest
```python
{
    "device_id": str,          # Device identifier
    "timestamp": str,          # ISO 8601 timestamp
    "temperature": float,      # Celsius
    "humidity": float,         # Percentage (0-100)
    "sound_volume": float      # Decibels
}
```

### WindowControl
```python
{
    "device_id": str,          # Device identifier
    "timestamp": str,          # ISO 8601 timestamp
    "position": float,         # Window position (0-100%)
    "confidence": float,       # Confidence score (0-1)
    "features_used": {
        "temperature": float,
        "humidity": float,
        "sound_volume": float
    }
}
```

---

## Configuration Reference

All configuration loaded from root `.env.config` file.

### Required Environment Variables

```bash
# MQTT Configuration (existing)
MQTT_BROKER=tcp://localhost:1883
MQTT_TOPIC_INFERENCE_REQ=ml/inference/request
MQTT_TOPIC_WINDOW_CONTROL=window/control

# ClickHouse Configuration (existing)
CLICKHOUSE_ADDR=localhost:9000
CLICKHOUSE_DB=iot
CLICKHOUSE_USER=default
CLICKHOUSE_PASS=

# ML Service Configuration (new)
ML_SERVICE_CLIENT_ID=ml-service
ML_SERVICE_MODEL_PATH=./ml_service/models/window_regressor.json
ML_SERVICE_MODEL_VERSION=v1.0.0
ML_SERVICE_OUTPUT_MIN=0.0
ML_SERVICE_OUTPUT_MAX=100.0

# Inference Configuration (new)
ML_MIN_CONFIDENCE=0.0
ML_PERCENTILE_LOW=0.1
ML_PERCENTILE_HIGH=0.9

# Training Configuration (new)
ML_TRAINING_MIN_SAMPLES=100
ML_TRAINING_TEST_SPLIT=0.2
ML_TRAINING_AUTO_TRAIN=true

# XGBoost Hyperparameters (new)
XGBOOST_MAX_DEPTH=6
XGBOOST_LEARNING_RATE=0.1
XGBOOST_N_ESTIMATORS=100
XGBOOST_SUBSAMPLE=0.8

# Logging (existing)
LOG_LEVEL=INFO
LOG_FORMAT=json
```

---

## Model Specifications

### XGBoost Regressor

**Type**: Gradient Boosted Decision Trees (Regression)

**Input Features** (3):
1. `temperature`: Normalized temperature in [0, 1]
2. `humidity`: Normalized humidity in [0, 1]
3. `sound_volume`: Normalized sound volume in [0, 1]

**Output**:
- Raw prediction: [0, 1] (normalized)
- Final output: [0, 100] (window position %)

**Objective**: `reg:squarederror` (minimize RMSE)

**Training Parameters**:
- `max_depth`: Maximum tree depth (prevents overfitting)
- `learning_rate`: Step size shrinkage (smaller = more conservative)
- `n_estimators`: Number of boosting rounds
- `subsample`: Row sampling ratio (prevents overfitting)

**Model Files**:
- `models/window_regressor.json`: XGBoost model in JSON format
- `models/window_regressor.json.meta`: Metadata with percentiles

---

## Error Handling

### Model Loading Errors
- **Model file not found**: Trigger auto-training if enabled, else fail
- **Invalid model format**: Log error and exit
- **Metadata missing**: Cannot proceed (percentiles required)

### MQTT Errors
- **Connection refused**: Retry with exponential backoff
- **Message parse error**: Log warning, skip message
- **Publish failure**: Log error, continue processing

### Inference Errors
- **Invalid input**: Log error, skip inference
- **Out-of-range values**: Log warning, clip to valid range
- **Model prediction error**: Log error, skip prediction

### Training Errors
- **ClickHouse unavailable**: Fallback to synthetic data
- **Insufficient data**: Fallback to synthetic data
- **Training failure**: Log error, use existing model if available

---

## Performance Characteristics

**Inference Latency**: < 5ms per prediction (CPU)
**Memory Usage**: ~100MB (XGBoost model + dependencies)
**Throughput**: 1000+ inferences/second (CPU)
**Model Size**: ~50KB (XGBoost JSON)

---

## Testing Strategy

### Unit Tests
- `test_feature_processor.py`: Normalization/denormalization
- `test_model_loader.py`: Model loading with mocks
- `test_predictor.py`: Inference logic
- `test_config.py`: Configuration parsing

### Integration Tests
- `test_mqtt_integration.py`: End-to-end MQTT flow
- `test_training_pipeline.py`: Training with mock ClickHouse

### Manual Testing
- Use `mosquitto_pub` to send test inference requests
- Use `mosquitto_sub` to verify window control responses

---

## Deployment

### Docker

**Image**: Python 3.11 slim
**Package Manager**: `uv` for fast dependency installation
**Volumes**: Model directory for persistence

**Dockerfile**:
- Copy root `.env.config`
- Install dependencies via `uv`
- Copy source code
- Expose no ports (MQTT client only)
- Health check: Python import test

### Docker Compose

```yaml
ml_service:
  build: ./ml_service
  depends_on:
    - mosquitto
    - clickhouse
  volumes:
    - ./ml_service/models:/app/models
  env_file:
    - .env.config
  restart: unless-stopped
```

---

## Future Enhancements

- [ ] Proper uncertainty estimation (quantile regression, SHAP values)
- [ ] Model versioning and A/B testing
- [ ] Online learning / incremental training
- [ ] Hyperparameter tuning (Optuna)
- [ ] Feature importance analysis
- [ ] Metrics export (Prometheus)
- [ ] Model performance monitoring dashboard
- [ ] Multi-model ensemble
- [ ] GPU acceleration support
- [ ] Automated retraining pipeline (cron job)

---

## References

- **Project Spec**: `../docs/SPEC.md`
- **Go Backend**: `../mqtt_backbone/SPEC.md`
- **ClickHouse Schema**: `../docs/clickhouse.md`
- **XGBoost Documentation**: https://xgboost.readthedocs.io/
