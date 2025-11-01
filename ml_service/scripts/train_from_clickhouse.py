#!/usr/bin/env python3
"""
Train XGBoost model from ClickHouse window_actions data.

This script queries historical window actuation data from ClickHouse,
computes feature percentiles, trains an XGBoost model, and saves the
model with metadata.
"""

import sys
import json
import numpy as np
import xgboost as xgb
import clickhouse_connect
from pathlib import Path
from datetime import datetime
from sklearn.model_selection import train_test_split
from sklearn.metrics import mean_squared_error, r2_score

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

from src.config import load_config


def query_training_data(client, lookback_days: int):
    """
    Query training data from ClickHouse window_actions table.

    Args:
        client: ClickHouse client
        lookback_days: Number of days to look back

    Returns:
        Tuple of (features, target) as numpy arrays
    """
    print(f"Querying window_actions table (last {lookback_days} days)...")

    query = f"""
    SELECT
        temperature,
        humidity,
        sound_volume,
        position
    FROM window_actions
    WHERE timestamp >= now() - INTERVAL {lookback_days} DAY
      AND temperature IS NOT NULL
      AND humidity IS NOT NULL
      AND sound_volume IS NOT NULL
      AND position IS NOT NULL
    ORDER BY timestamp DESC
    """

    result = client.query(query)
    data = result.result_rows

    if len(data) == 0:
        print("No data found in window_actions table")
        return None, None

    # Convert to numpy arrays
    data_array = np.array(data, dtype=np.float64)

    # Split features and target
    X = data_array[:, :3]  # temperature, humidity, sound_volume
    y = data_array[:, 3]   # position

    print(f"Retrieved {len(X)} samples")
    print(f"Feature ranges:")
    print(f"  Temperature: [{X[:, 0].min():.2f}, {X[:, 0].max():.2f}]")
    print(f"  Humidity: [{X[:, 1].min():.2f}, {X[:, 1].max():.2f}]")
    print(f"  Sound Volume: [{X[:, 2].min():.2f}, {X[:, 2].max():.2f}]")
    print(f"  Position: [{y.min():.2f}, {y.max():.2f}]")

    return X, y


def compute_percentiles(X: np.ndarray, p_low: float = 0.1, p_high: float = 0.9):
    """
    Compute percentiles for feature normalization.

    Args:
        X: Feature array (N, 3)
        p_low: Lower percentile (default 0.1 = 10th percentile)
        p_high: Upper percentile (default 0.9 = 90th percentile)

    Returns:
        Dictionary of percentile tuples
    """
    feature_names = ['temperature', 'humidity', 'sound_volume']
    percentiles = {}

    print(f"\nComputing percentiles (p{p_low*100:.0f}, p{p_high*100:.0f}):")

    for i, name in enumerate(feature_names):
        p10 = float(np.percentile(X[:, i], p_low * 100))
        p90 = float(np.percentile(X[:, i], p_high * 100))
        percentiles[name] = [p10, p90]
        print(f"  {name}: [{p10:.2f}, {p90:.2f}]")

    return percentiles


def normalize_features(X: np.ndarray, percentiles: dict):
    """
    Normalize features using percentiles.

    Args:
        X: Feature array (N, 3)
        percentiles: Dictionary of percentile tuples

    Returns:
        Normalized feature array
    """
    X_norm = np.zeros_like(X)
    feature_names = ['temperature', 'humidity', 'sound_volume']

    for i, name in enumerate(feature_names):
        p_low, p_high = percentiles[name]
        X_norm[:, i] = (X[:, i] - p_low) / (p_high - p_low)
        X_norm[:, i] = np.clip(X_norm[:, i], 0.0, 1.0)

    return X_norm


def normalize_target(y: np.ndarray, output_min: float, output_max: float):
    """
    Normalize target to [0, 1] range.

    Args:
        y: Target array (position in [output_min, output_max])
        output_min: Minimum output value
        output_max: Maximum output value

    Returns:
        Normalized target array in [0, 1]
    """
    return (y - output_min) / (output_max - output_min)


def train_xgboost(X_train, y_train, X_test, y_test, xgb_config):
    """
    Train XGBoost model.

    Args:
        X_train: Training features (normalized)
        y_train: Training target (normalized)
        X_test: Test features (normalized)
        y_test: Test target (normalized)
        xgb_config: XGBoost configuration

    Returns:
        Trained XGBoost Booster
    """
    print("\nTraining XGBoost model...")
    print(f"Hyperparameters:")
    print(f"  max_depth: {xgb_config.max_depth}")
    print(f"  learning_rate: {xgb_config.learning_rate}")
    print(f"  n_estimators: {xgb_config.n_estimators}")
    print(f"  subsample: {xgb_config.subsample}")

    # Create XGBoost regressor
    model = xgb.XGBRegressor(
        max_depth=xgb_config.max_depth,
        learning_rate=xgb_config.learning_rate,
        n_estimators=xgb_config.n_estimators,
        subsample=xgb_config.subsample,
        objective='reg:squarederror',
        random_state=42
    )

    # Train model
    model.fit(
        X_train, y_train,
        eval_set=[(X_test, y_test)],
        verbose=False
    )

    # Evaluate on test set
    y_pred = model.predict(X_test)
    rmse = np.sqrt(mean_squared_error(y_test, y_pred))
    r2 = r2_score(y_test, y_pred)

    print(f"\nModel Performance:")
    print(f"  Test RMSE: {rmse:.4f} (normalized)")
    print(f"  Test R²: {r2:.4f}")

    return model


def save_model(model: xgb.XGBRegressor, percentiles: dict, rmse: float, r2: float,
               n_samples: int, output_path: Path, version: str):
    """
    Save XGBoost model and metadata.

    Args:
        model: Trained XGBoost model
        percentiles: Feature percentiles
        rmse: Test RMSE
        r2: Test R² score
        n_samples: Number of training samples
        output_path: Path to save model
        version: Model version string
    """
    output_path.parent.mkdir(parents=True, exist_ok=True)

    # Save model as JSON
    model_file = output_path
    model.get_booster().save_model(str(model_file))
    print(f"\nModel saved to: {model_file}")

    # Save metadata
    metadata = {
        'version': version,
        'model_type': 'XGBoost',
        'percentiles': percentiles,
        'training_info': {
            'date': datetime.utcnow().isoformat() + 'Z',
            'samples': n_samples,
            'rmse': float(rmse),
            'r2': float(r2),
            'source': 'clickhouse_window_actions'
        }
    }

    metadata_file = Path(str(model_file) + '.meta')
    with open(metadata_file, 'w') as f:
        json.dump(metadata, f, indent=2)
    print(f"Metadata saved to: {metadata_file}")


def main():
    """Main training pipeline."""
    print("=" * 60)
    print("XGBoost Training from ClickHouse")
    print("=" * 60)

    # Load configuration
    print("\n1. Loading configuration from .env.config...")
    try:
        config = load_config()
    except Exception as e:
        print(f"ERROR: Failed to load configuration: {e}")
        sys.exit(1)

    # Connect to ClickHouse
    print(f"\n2. Connecting to ClickHouse at {config.clickhouse.addr}...")
    try:
        # Parse address (host:port)
        host_parts = config.clickhouse.addr.split(':')
        if len(host_parts) == 2:
            host, port = host_parts
            port = int(port)
        else:
            host = config.clickhouse.addr
            port = 9000

        client = clickhouse_connect.get_client(
            host=host,
            port=port,
            username=config.clickhouse.user,
            password=config.clickhouse.password,
            database=config.clickhouse.database
        )
        print("Connected successfully")
    except Exception as e:
        print(f"ERROR: Failed to connect to ClickHouse: {e}")
        print("Tip: Ensure ClickHouse is running and accessible")
        sys.exit(1)

    # Query training data
    print(f"\n3. Querying training data...")
    X, y = query_training_data(client, config.training.lookback_days)

    if X is None or len(X) < config.training.min_samples:
        print(f"\nERROR: Insufficient training data")
        print(f"  Required: {config.training.min_samples} samples")
        print(f"  Found: {len(X) if X is not None else 0} samples")
        print("\nFalling back to synthetic data training...")
        print("Run: uv run python scripts/create_initial_model.py")
        sys.exit(1)

    # Compute percentiles
    print(f"\n4. Computing feature percentiles...")
    percentiles = compute_percentiles(
        X,
        config.inference.percentile_low,
        config.inference.percentile_high
    )

    # Normalize features and target
    print(f"\n5. Normalizing features and target...")
    X_norm = normalize_features(X, percentiles)
    y_norm = normalize_target(y, config.model.output_min, config.model.output_max)

    # Split train/test
    print(f"\n6. Splitting train/test sets (test_size={config.training.test_split})...")
    X_train, X_test, y_train, y_test = train_test_split(
        X_norm, y_norm,
        test_size=config.training.test_split,
        random_state=42
    )
    print(f"  Training samples: {len(X_train)}")
    print(f"  Test samples: {len(X_test)}")

    # Train model
    print(f"\n7. Training XGBoost model...")
    model = train_xgboost(X_train, y_train, X_test, y_test, config.xgboost)

    # Evaluate
    y_pred = model.predict(X_test)
    rmse = np.sqrt(mean_squared_error(y_test, y_pred))
    r2 = r2_score(y_test, y_pred)

    # Save model
    print(f"\n8. Saving model and metadata...")
    model_path = Path(config.model.path)
    save_model(model, percentiles, rmse, r2, len(X), model_path, config.model.version)

    print("\n" + "=" * 60)
    print("Training Complete!")
    print("=" * 60)
    print(f"\nModel Location: {model_path}")
    print(f"Metadata Location: {str(model_path) + '.meta'}")
    print(f"\nNext Steps:")
    print(f"1. Start the ML service: uv run python -m src.main")
    print(f"2. Send inference requests via MQTT")


if __name__ == '__main__':
    main()
