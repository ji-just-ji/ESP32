# ML Service Implementation Plan

## Overview

This document tracks the implementation progress for migrating the ML Service from PyTorch to XGBoost and from YAML-based config to .env-based config.

**Status**: 🟡 In Progress
**Started**: 2025-10-31
**Target Completion**: TBD

---

## Phase 0: Documentation ✅

### ✅ Task 0.1: Create SPEC.md
**Status**: Complete
**Completed**: 2025-10-31

Created comprehensive specification document covering:
- Service architecture and responsibilities
- XGBoost model specifications
- MQTT communication protocol
- ClickHouse training pipeline
- Configuration reference
- Component specifications

**Files Modified**:
- `ml_service/SPEC.md` (created)

---

### ✅ Task 0.2: Create PLAN.md
**Status**: Complete
**Completed**: 2025-10-31

Created this implementation plan document with detailed checklist.

**Files Modified**:
- `ml_service/PLAN.md` (created)

---

## Phase 1: Configuration & Dependencies

### ✅ Task 1.1: Update root .env.config
**Status**: Complete
**Completed**: 2025-10-31

Add ML service configuration to root `.env.config` file:

**New Environment Variables to Add**:
```bash
# ML Service Configuration
ML_SERVICE_CLIENT_ID=ml-service
ML_SERVICE_MODEL_PATH=./ml_service/models/window_regressor.json
ML_SERVICE_MODEL_VERSION=v1.0.0
ML_SERVICE_OUTPUT_MIN=0.0
ML_SERVICE_OUTPUT_MAX=100.0

# Inference Configuration
ML_MIN_CONFIDENCE=0.0
ML_PERCENTILE_LOW=0.1
ML_PERCENTILE_HIGH=0.9

# Training Configuration
ML_TRAINING_MIN_SAMPLES=100
ML_TRAINING_TEST_SPLIT=0.2
ML_TRAINING_AUTO_TRAIN=true
ML_TRAINING_LOOKBACK_DAYS=30

# XGBoost Hyperparameters
XGBOOST_MAX_DEPTH=6
XGBOOST_LEARNING_RATE=0.1
XGBOOST_N_ESTIMATORS=100
XGBOOST_SUBSAMPLE=0.8

# Logging (if not already present)
LOG_LEVEL=INFO
LOG_FORMAT=json
```

**Files to Modify**:
- `.env.config` (append new variables)

**Verification**:
- [ ] All variables added
- [ ] No syntax errors
- [ ] Documented in SPEC.md

---

### ✅ Task 1.2: Update pyproject.toml
**Status**: Complete
**Completed**: 2025-10-31

Replace PyTorch dependencies with XGBoost and add environment loading.

**Changes**:
- Remove: `torch>=2.0.0`
- Remove: `pyyaml>=6.0` (no longer needed)
- Add: `xgboost>=2.0.0`
- Add: `clickhouse-connect>=0.7.0`
- Add: `python-dotenv>=1.0.0`
- Keep: `paho-mqtt`, `numpy`, `python-json-logger`

**Files to Modify**:
- `ml_service/pyproject.toml`

**Verification**:
- [ ] Run `uv sync` to verify dependencies resolve
- [ ] No conflicting versions
- [ ] All packages install successfully

---

## Phase 2: Core Model Components

### ✅ Task 2.1: Create src/config.py
**Status**: Complete
**Completed**: 2025-10-31

Create configuration module to load and parse `.env.config` from project root.

**Implementation**:
- Load `.env.config` from `../env.config` (one level up)
- Create `MLServiceConfig` dataclass with all settings
- Validate required fields
- Provide helper methods for accessing config

**Structure**:
```python
@dataclass
class MQTTConfig:
    broker: str
    client_id: str
    inference_topic: str
    window_control_topic: str
    qos: int = 1
    keepalive: int = 60
    reconnect_delay: int = 5

@dataclass
class ModelConfig:
    path: str
    version: str
    output_min: float
    output_max: float

@dataclass
class InferenceConfig:
    min_confidence: float
    percentile_low: float
    percentile_high: float

@dataclass
class TrainingConfig:
    min_samples: int
    test_split: float
    auto_train: bool
    lookback_days: int

@dataclass
class XGBoostConfig:
    max_depth: int
    learning_rate: float
    n_estimators: int
    subsample: float

@dataclass
class ClickHouseConfig:
    addr: str
    database: str
    user: str
    password: str

@dataclass
class LoggingConfig:
    level: str
    format: str

@dataclass
class MLServiceConfig:
    mqtt: MQTTConfig
    model: ModelConfig
    inference: InferenceConfig
    training: TrainingConfig
    xgboost: XGBoostConfig
    clickhouse: ClickHouseConfig
    logging: LoggingConfig
```

**Files to Create**:
- `ml_service/src/config.py`

**Verification**:
- [ ] Successfully loads `.env.config` from root
- [ ] All required fields parsed
- [ ] Type validation works
- [ ] Missing required fields raise clear errors

---

### ✅ Task 2.2: Rewrite src/model_loader.py
**Status**: Complete
**Completed**: 2025-10-31

Replace PyTorch model loading with XGBoost.

**Changes**:
- Remove: `WindowRegressorModel` class (PyTorch)
- Remove: `torch` imports
- Add: `xgboost` imports
- Implement: XGBoost `Booster` loading from JSON
- Keep: Metadata loading logic (percentiles)

**Key Methods**:
- `load()`: Load XGBoost model from JSON + metadata from .meta file
- `get_model()`: Return `xgb.Booster` instance
- `get_metadata()`: Return metadata dict
- `get_percentiles()`: Return percentile dict
- `_validate_metadata()`: Validate metadata structure

**Model Format**:
- Model file: `window_regressor.json` (XGBoost JSON format)
- Metadata file: `window_regressor.json.meta` (JSON with percentiles)

**Files to Modify**:
- `ml_service/src/model_loader.py`

**Verification**:
- [ ] Loads XGBoost JSON model successfully
- [ ] Loads metadata from .meta file
- [ ] Validates percentile structure
- [ ] Raises clear errors on missing files
- [ ] No torch dependencies remain

---

### ✅ Task 2.3: Update src/predictor.py
**Status**: Complete
**Completed**: 2025-10-31

Replace PyTorch inference with XGBoost.

**Changes**:
- Remove: `torch` imports and tensor operations
- Update: Use `xgb.DMatrix` for input
- Update: Use `booster.predict()` for inference
- Keep: Feature normalization logic
- Keep: Confidence calculation
- Keep: Output denormalization

**Key Changes**:
```python
# OLD (PyTorch):
input_tensor = torch.from_numpy(features).float().unsqueeze(0)
output = self.model(input_tensor)
position = output.item()

# NEW (XGBoost):
dmatrix = xgb.DMatrix(features.reshape(1, -1))
output = self.model.predict(dmatrix)
position = float(output[0])
```

**Files to Modify**:
- `ml_service/src/predictor.py`

**Verification**:
- [ ] Inference runs without torch
- [ ] Output matches expected format
- [ ] Confidence scoring works
- [ ] All tests pass

---

### ✅ Task 2.4: Keep src/feature_processor.py unchanged
**Status**: Complete (No changes needed)
**Completed**: 2025-10-31

No changes needed - already uses numpy.

**Verification**:
- [ ] Confirmed compatible with XGBoost
- [ ] No torch dependencies

---

## Phase 3: Training Pipeline

### ✅ Task 3.1: Create scripts/train_from_clickhouse.py
**Status**: Complete
**Completed**: 2025-10-31

Create training script that loads data from ClickHouse and trains XGBoost model.

**Implementation Steps**:
1. Load config from root `.env.config`
2. Connect to ClickHouse using credentials
3. Query `window_actions` table (last N days)
4. Extract features and target
5. Validate and clean data
6. Check minimum sample requirement
7. Split train/test
8. Compute percentiles from training set
9. Normalize features
10. Train XGBoost with hyperparameters from config
11. Evaluate on test set
12. Save model as JSON
13. Save metadata with percentiles and metrics

**ClickHouse Query**:
```sql
SELECT
    timestamp,
    device_id,
    temperature,
    humidity,
    sound_volume,
    position,
    confidence
FROM window_actions
WHERE timestamp >= now() - INTERVAL {lookback_days} DAY
ORDER BY timestamp DESC
```

**Fallback Logic**:
- If < `ML_TRAINING_MIN_SAMPLES`: Use synthetic data
- If ClickHouse unavailable: Use synthetic data
- Log clear warnings when using fallback

**Files to Create**:
- `ml_service/scripts/train_from_clickhouse.py`

**Verification**:
- [ ] Connects to ClickHouse successfully
- [ ] Queries data correctly
- [ ] Computes percentiles accurately
- [ ] Trains XGBoost model
- [ ] Saves model and metadata
- [ ] Handles insufficient data gracefully
- [ ] Handles ClickHouse connection errors

---

### ✅ Task 3.2: Update scripts/create_initial_model.py
**Status**: Complete
**Completed**: 2025-10-31

Replace PyTorch synthetic model with XGBoost.

**Changes**:
- Remove: PyTorch model creation
- Remove: `WindowRegressorModel` usage
- Add: XGBoost training on synthetic data
- Keep: Synthetic data generation
- Keep: Percentile computation
- Update: Save as XGBoost JSON format

**Implementation**:
```python
# Generate synthetic data
X_train, y_train = generate_synthetic_data(n_samples=1000)

# Compute percentiles
percentiles = compute_percentiles(X_train)

# Normalize features
X_train_norm = normalize(X_train, percentiles)

# Train XGBoost
model = xgb.XGBRegressor(
    max_depth=6,
    learning_rate=0.1,
    n_estimators=100,
    objective='reg:squarederror'
)
model.fit(X_train_norm, y_train)

# Save model
model.get_booster().save_model('models/window_regressor.json')

# Save metadata
save_metadata('models/window_regressor.json.meta', percentiles)
```

**Files to Modify**:
- `ml_service/scripts/create_initial_model.py`

**Verification**:
- [ ] Generates synthetic data
- [ ] Trains XGBoost model
- [ ] Saves in correct format
- [ ] Model loadable by model_loader.py
- [ ] No torch dependencies

---

## Phase 4: Application Integration

### ✅ Task 4.1: Update src/main.py
**Status**: Complete
**Completed**: 2025-10-31

Replace YAML config loading with .env config and add auto-training.

**Changes**:
- Remove: YAML config loading
- Remove: `yaml` imports
- Add: Import `config.py` module
- Add: Load `.env.config` using python-dotenv
- Add: Auto-training logic on startup
- Update: Use config module for all settings
- Keep: MQTT orchestration
- Keep: Graceful shutdown

**Auto-training Logic**:
```python
# Check if model exists
if not os.path.exists(config.model.path):
    logger.info("Model not found, checking auto-train setting...")

    if config.training.auto_train:
        logger.info("Auto-train enabled, attempting to train from ClickHouse...")
        try:
            # Try training from ClickHouse
            train_from_clickhouse()
        except Exception as e:
            logger.warning(f"ClickHouse training failed: {e}")
            logger.info("Falling back to synthetic data...")
            create_initial_model()
    else:
        logger.error("Model not found and auto-train disabled. Exiting.")
        sys.exit(1)
```

**Files to Modify**:
- `ml_service/src/main.py`

**Verification**:
- [ ] Loads config from .env.config
- [ ] Auto-training works when enabled
- [ ] Falls back to synthetic data correctly
- [ ] Service starts successfully with model
- [ ] No YAML dependencies remain

---

### ✅ Task 4.2: Update src/mqtt_client.py
**Status**: Complete
**Completed**: 2025-10-31

Replace config dict parameters with config module.

**Changes**:
- Update constructor to accept `MQTTConfig` object
- Remove hard-coded config dict parsing
- Use config object for all MQTT settings
- Keep all MQTT logic unchanged

**Example**:
```python
# OLD:
def __init__(self, broker: str, port: int, client_id: str, ...):
    self.broker = broker
    ...

# NEW:
def __init__(self, mqtt_config: MQTTConfig):
    self.broker = mqtt_config.broker
    self.port = mqtt_config.port
    ...
```

**Files to Modify**:
- `ml_service/src/mqtt_client.py`

**Verification**:
- [ ] Accepts config object
- [ ] All MQTT operations work
- [ ] Topics configured correctly
- [ ] Reconnection works

---

### ✅ Task 4.3: Remove config.yaml
**Status**: Complete
**Completed**: 2025-10-31

Delete the old YAML configuration file.

**Files to Delete**:
- `ml_service/config.yaml`

**Verification**:
- [ ] File deleted
- [ ] No references to config.yaml in code
- [ ] Service runs without config.yaml

---

## Phase 5: Deployment & Documentation

### ✅ Task 5.1: Update Dockerfile
**Status**: Complete
**Completed**: 2025-10-31

Update Docker build to use .env.config and XGBoost.

**Changes**:
- Copy `.env.config` from root directory
- Remove `config.yaml` copy
- Verify XGBoost installs correctly with uv
- Keep existing structure

**Dockerfile updates**:
```dockerfile
# Copy .env.config from root
COPY ../.env.config ./

# Remove this line:
# COPY config.yaml ./

# Keep existing uv and dependency installation
```

**Files to Modify**:
- `ml_service/Dockerfile`

**Verification**:
- [ ] Docker build succeeds
- [ ] .env.config copied correctly
- [ ] XGBoost installed
- [ ] Service starts in container
- [ ] MQTT connection works from container

---

### ✅ Task 5.2: Update README.md
**Status**: Complete
**Completed**: 2025-10-31

Update documentation to reflect new configuration and XGBoost.

**Sections to Update**:
1. **Overview**: Mention XGBoost instead of PyTorch
2. **Configuration**: Remove config.yaml, document .env.config
3. **Quick Start**: Update setup instructions
4. **Model Training**: Document new training pipeline
5. **Dependencies**: List XGBoost, remove torch

**New Content**:
- How to configure via .env.config
- How to train from ClickHouse data
- XGBoost model format explanation
- uv usage for all operations

**Files to Modify**:
- `ml_service/README.md`

**Verification**:
- [ ] All sections updated
- [ ] No config.yaml references
- [ ] Instructions accurate and complete
- [ ] Examples work as documented

---

## Phase 6: Testing & Validation

### ⬜ Task 6.1: Manual Testing
**Status**: Pending

Test the complete service end-to-end.

**Test Cases**:
1. **Config Loading**: Verify .env.config loads correctly
2. **Auto-training**: Test training from ClickHouse
3. **Synthetic Fallback**: Test synthetic data training
4. **MQTT Communication**: Test inference request → window control
5. **Model Loading**: Test XGBoost model loads
6. **Inference**: Test predictions are reasonable
7. **Confidence Filtering**: Test confidence threshold

**Test Commands**:
```bash
# 1. Start service
uv run python -m src.main

# 2. Send test inference request
mosquitto_pub -h localhost -t "ml/inference/request" \
  -m '{"device_id":"test-001","timestamp":"2025-10-31T12:00:00Z","temperature":25.5,"humidity":60.0,"sound_volume":65.5}'

# 3. Monitor window control output
mosquitto_sub -h localhost -t "window/control" -v
```

**Verification**:
- [ ] Service starts without errors
- [ ] MQTT connection established
- [ ] Inference requests processed
- [ ] Window control commands published
- [ ] Logs show expected behavior

---

### ⬜ Task 6.2: Unit Tests (Optional)
**Status**: Pending

Update existing tests to work with XGBoost.

**Tests to Update/Create**:
- `tests/test_config.py`: Test config loading
- `tests/test_model_loader.py`: Test XGBoost loading
- `tests/test_predictor.py`: Test XGBoost inference
- `tests/test_feature_processor.py`: (should work as-is)

**Verification**:
- [ ] All tests pass
- [ ] No torch dependencies in tests

---

## Summary

### Completion Status

| Phase | Tasks | Completed | Percentage |
|-------|-------|-----------|------------|
| Phase 0: Documentation | 2 | 2 | 100% |
| Phase 1: Config & Dependencies | 2 | 2 | 100% |
| Phase 2: Core Components | 4 | 4 | 100% |
| Phase 3: Training Pipeline | 2 | 2 | 100% |
| Phase 4: Integration | 3 | 3 | 100% |
| Phase 5: Deployment & Docs | 2 | 2 | 100% |
| Phase 6: Testing | 2 | 0 | 0% |
| **TOTAL** | **17** | **15** | **88%** |

---

### Files Modified Summary

**Created**:
- ✅ `ml_service/SPEC.md`
- ✅ `ml_service/PLAN.md`
- ⬜ `ml_service/src/config.py`
- ⬜ `ml_service/scripts/train_from_clickhouse.py`

**Modified**:
- ⬜ `.env.config`
- ⬜ `ml_service/pyproject.toml`
- ⬜ `ml_service/src/model_loader.py`
- ⬜ `ml_service/src/predictor.py`
- ⬜ `ml_service/scripts/create_initial_model.py`
- ⬜ `ml_service/src/main.py`
- ⬜ `ml_service/src/mqtt_client.py`
- ⬜ `ml_service/Dockerfile`
- ⬜ `ml_service/README.md`

**Deleted**:
- ⬜ `ml_service/config.yaml`

---

### Next Steps

1. ✅ Complete Phase 0 (Documentation)
2. ⬜ Start Phase 1: Update .env.config and dependencies
3. ⬜ Continue through phases sequentially
4. ⬜ Update this PLAN.md after completing each task

---

## Notes & Decisions

### Design Decisions
- Using XGBoost JSON format for portability
- Percentile metadata stored in separate .meta file
- Auto-training enabled by default
- Fallback to synthetic data when insufficient historical data
- All configuration centralized in root .env.config

### Challenges Encountered
(To be filled in during implementation)

### Deviations from Original Plan
(To be filled in if implementation differs from plan)

---

**Last Updated**: 2025-10-31
**Updated By**: Claude Code
