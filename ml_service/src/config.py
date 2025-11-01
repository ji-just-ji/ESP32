"""
Configuration module for ML service.

Loads environment variables from root .env.config file.
"""

import os
from dataclasses import dataclass
from pathlib import Path
from typing import Optional
from dotenv import load_dotenv


@dataclass
class MQTTConfig:
    """MQTT configuration."""
    broker: str
    client_id: str
    inference_topic: str
    window_control_topic: str
    qos: int = 1
    keepalive: int = 60
    reconnect_delay: int = 5


@dataclass
class ModelConfig:
    """Model configuration."""
    path: str
    version: str
    output_min: float
    output_max: float


@dataclass
class InferenceConfig:
    """Inference configuration."""
    min_confidence: float
    percentile_low: float
    percentile_high: float


@dataclass
class TrainingConfig:
    """Training configuration."""
    min_samples: int
    test_split: float
    auto_train: bool
    lookback_days: int


@dataclass
class XGBoostConfig:
    """XGBoost hyperparameters."""
    max_depth: int
    learning_rate: float
    n_estimators: int
    subsample: float


@dataclass
class ClickHouseConfig:
    """ClickHouse database configuration."""
    addr: str
    database: str
    user: str
    password: str


@dataclass
class LoggingConfig:
    """Logging configuration."""
    level: str
    format: str


@dataclass
class MLServiceConfig:
    """Complete ML service configuration."""
    mqtt: MQTTConfig
    model: ModelConfig
    inference: InferenceConfig
    training: TrainingConfig
    xgboost: XGBoostConfig
    clickhouse: ClickHouseConfig
    logging: LoggingConfig


def load_config() -> MLServiceConfig:
    """
    Load configuration from root .env.config file.

    Returns:
        MLServiceConfig object with all configuration

    Raises:
        ValueError: If required environment variables are missing
        FileNotFoundError: If .env.config file not found
    """
    # Find .env.config in project root (one level up from ml_service/)
    config_path = Path(__file__).parent.parent.parent / '.env.config'

    if not config_path.exists():
        raise FileNotFoundError(
            f".env.config not found at {config_path}. "
            "Please create it in the project root directory."
        )

    # Load environment variables
    load_dotenv(config_path)

    # Parse MQTT config
    mqtt_config = MQTTConfig(
        broker=_get_env('MQTT_BROKER'),
        client_id=_get_env('ML_SERVICE_CLIENT_ID', default='ml-service'),
        inference_topic=_get_env('MQTT_TOPIC_INFERENCE_REQ'),
        window_control_topic=_get_env('MQTT_TOPIC_WINDOW_CONTROL'),
        qos=int(_get_env('MQTT_QOS', default='1')),
        keepalive=int(_get_env('MQTT_KEEPALIVE', default='60')),
        reconnect_delay=int(_get_env('MQTT_RECONNECT_DELAY', default='5'))
    )

    # Parse model config
    model_config = ModelConfig(
        path=_get_env('ML_SERVICE_MODEL_PATH'),
        version=_get_env('ML_SERVICE_MODEL_VERSION', default='v1.0.0'),
        output_min=float(_get_env('ML_SERVICE_OUTPUT_MIN', default='0.0')),
        output_max=float(_get_env('ML_SERVICE_OUTPUT_MAX', default='100.0'))
    )

    # Parse inference config
    inference_config = InferenceConfig(
        min_confidence=float(_get_env('ML_MIN_CONFIDENCE', default='0.0')),
        percentile_low=float(_get_env('ML_PERCENTILE_LOW', default='0.1')),
        percentile_high=float(_get_env('ML_PERCENTILE_HIGH', default='0.9'))
    )

    # Parse training config
    training_config = TrainingConfig(
        min_samples=int(_get_env('ML_TRAINING_MIN_SAMPLES', default='100')),
        test_split=float(_get_env('ML_TRAINING_TEST_SPLIT', default='0.2')),
        auto_train=_get_env('ML_TRAINING_AUTO_TRAIN', default='true').lower() == 'true',
        lookback_days=int(_get_env('ML_TRAINING_LOOKBACK_DAYS', default='30'))
    )

    # Parse XGBoost config
    xgboost_config = XGBoostConfig(
        max_depth=int(_get_env('XGBOOST_MAX_DEPTH', default='6')),
        learning_rate=float(_get_env('XGBOOST_LEARNING_RATE', default='0.1')),
        n_estimators=int(_get_env('XGBOOST_N_ESTIMATORS', default='100')),
        subsample=float(_get_env('XGBOOST_SUBSAMPLE', default='0.8'))
    )

    # Parse ClickHouse config
    clickhouse_config = ClickHouseConfig(
        addr=_get_env('CLICKHOUSE_ADDR'),
        database=_get_env('CLICKHOUSE_DB'),
        user=_get_env('CLICKHOUSE_USER', default='default'),
        password=_get_env('CLICKHOUSE_PASS', default='')
    )

    # Parse logging config
    logging_config = LoggingConfig(
        level=_get_env('LOG_LEVEL', default='INFO'),
        format=_get_env('LOG_FORMAT', default='json')
    )

    return MLServiceConfig(
        mqtt=mqtt_config,
        model=model_config,
        inference=inference_config,
        training=training_config,
        xgboost=xgboost_config,
        clickhouse=clickhouse_config,
        logging=logging_config
    )


def _get_env(key: str, default: Optional[str] = None) -> str:
    """
    Get environment variable with optional default.

    Args:
        key: Environment variable name
        default: Default value if not found

    Returns:
        Environment variable value

    Raises:
        ValueError: If variable not found and no default provided
    """
    value = os.getenv(key, default)

    if value is None:
        raise ValueError(
            f"Required environment variable '{key}' not found in .env.config"
        )

    return value


# Global config instance (lazy loaded)
_config: Optional[MLServiceConfig] = None


def get_config() -> MLServiceConfig:
    """
    Get global configuration instance (singleton pattern).

    Returns:
        MLServiceConfig instance
    """
    global _config

    if _config is None:
        _config = load_config()

    return _config
