#!/usr/bin/env python3
"""
Create an initial XGBoost model for window control.

This script creates a simple trained model with metadata for testing purposes.
In production, this would be replaced with a properly trained model on real data.
"""

import xgboost as xgb
import json
import numpy as np
from pathlib import Path
import sys

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

from src.config import load_config


def generate_synthetic_data(n_samples: int = 1000):
    """
    Generate synthetic training data to compute percentiles.

    In production, this would be replaced with real sensor data.

    Returns:
        Tuple of (X, y) where X is features and y is target
    """
    np.random.seed(42)

    # Generate realistic ranges
    temperature = np.random.normal(22, 7, n_samples)  # Mean 22°C, std 7°C
    humidity = np.random.beta(4, 4, n_samples) * 100  # Beta dist for 0-100%
    sound_volume = np.random.gamma(2, 10, n_samples) + 35  # Gamma dist, min 35 dB

    # Clip to reasonable ranges
    temperature = np.clip(temperature, 5, 40)
    humidity = np.clip(humidity, 10, 95)
    sound_volume = np.clip(sound_volume, 30, 85)

    # Create features array
    X = np.column_stack([temperature, humidity, sound_volume])

    # Generate synthetic target (window position)
    # Simple rule-based logic:
    # - Higher temperature -> open more
    # - Higher humidity -> close more
    # - Higher sound volume -> close more (for noise isolation)

    # Normalize features for rule
    temp_norm = (temperature - 5) / (40 - 5)
    humidity_norm = (humidity - 10) / (95 - 10)
    volume_norm = (sound_volume - 30) / (85 - 30)

    # Combine: open window when hot, close when humid or noisy
    y = (temp_norm * 0.5 - humidity_norm * 0.3 - volume_norm * 0.2 + 0.4) * 100
    y = np.clip(y, 0, 100)

    # Add some noise
    y += np.random.normal(0, 5, n_samples)
    y = np.clip(y, 0, 100)

    return X, y


def compute_percentiles(X: np.ndarray, p_low: float = 0.1, p_high: float = 0.9):
    """
    Compute percentiles for normalization.

    Args:
        X: Feature array (N, 3)
        p_low: Lower percentile (default 0.1 = 10th percentile)
        p_high: Upper percentile (default 0.9 = 90th percentile)

    Returns:
        Dictionary of percentile tuples
    """
    feature_names = ['temperature', 'humidity', 'sound_volume']
    percentiles = {}

    for i, name in enumerate(feature_names):
        p10 = float(np.percentile(X[:, i], p_low * 100))
        p90 = float(np.percentile(X[:, i], p_high * 100))
        percentiles[name] = [p10, p90]
        print(f"{name}: p{p_low*100:.0f}={p10:.2f}, p{p_high*100:.0f}={p90:.2f}")

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
        Normalized target array
    """
    return (y - output_min) / (output_max - output_min)


def create_metadata(percentiles: dict, n_samples: int, rmse: float, version: str):
    """
    Create model metadata.

    Args:
        percentiles: Feature percentiles
        n_samples: Number of training samples
        rmse: RMSE on training data
        version: Model version string

    Returns:
        Metadata dictionary
    """
    metadata = {
        'version': version,
        'model_type': 'XGBoost',
        'percentiles': percentiles,
        'training_info': {
            'date': '2025-10-31T00:00:00Z',
            'samples': n_samples,
            'rmse': float(rmse),
            'source': 'synthetic_data',
            'notes': 'Initial dummy model for testing. Replace with trained model from real data.'
        }
    }

    return metadata


def save_model(model: xgb.XGBRegressor, metadata: dict, output_path: Path):
    """
    Save model and metadata.

    Args:
        model: Trained XGBoost model
        metadata: Metadata dictionary
        output_path: Path to save model
    """
    output_path.parent.mkdir(parents=True, exist_ok=True)

    # Save model as JSON
    model.get_booster().save_model(str(output_path))
    print(f"Model saved to: {output_path}")

    # Save metadata
    metadata_path = Path(str(output_path) + '.meta')
    with open(metadata_path, 'w') as f:
        json.dump(metadata, f, indent=2)
    print(f"Metadata saved to: {metadata_path}")


def main():
    """Main entry point."""
    print("=" * 60)
    print("Creating Initial XGBoost Model (Synthetic Data)")
    print("=" * 60)

    # Load configuration
    print("\n1. Loading configuration from .env.config...")
    try:
        config = load_config()
    except Exception as e:
        print(f"ERROR: Failed to load configuration: {e}")
        sys.exit(1)

    # Generate synthetic training data
    print("\n2. Generating synthetic training data...")
    n_samples = 1000
    X, y = generate_synthetic_data(n_samples=n_samples)
    print(f"Generated {n_samples} samples")

    # Compute percentiles
    print("\n3. Computing percentiles...")
    percentiles = compute_percentiles(
        X,
        config.inference.percentile_low,
        config.inference.percentile_high
    )

    # Normalize features and target
    print("\n4. Normalizing features and target...")
    X_norm = normalize_features(X, percentiles)
    y_norm = normalize_target(y, config.model.output_min, config.model.output_max)

    # Create and train XGBoost model
    print("\n5. Training XGBoost model...")
    model = xgb.XGBRegressor(
        max_depth=config.xgboost.max_depth,
        learning_rate=config.xgboost.learning_rate,
        n_estimators=config.xgboost.n_estimators,
        subsample=config.xgboost.subsample,
        objective='reg:squarederror',
        random_state=42
    )

    model.fit(X_norm, y_norm, verbose=False)
    print("Training complete")

    # Test model with sample input
    print("\n6. Testing model with sample input...")
    sample_input = np.array([[0.5, 0.5, 0.5]])  # Middle of normalized range
    sample_output = model.predict(sample_input)
    position_pct = sample_output[0] * 100
    print(f"Sample prediction: {sample_output[0]:.4f} (normalized)")
    print(f"Window position: {position_pct:.2f}%")

    # Calculate RMSE
    y_pred = model.predict(X_norm)
    rmse = np.sqrt(np.mean((y_pred - y_norm) ** 2))
    print(f"Training RMSE: {rmse:.4f}")

    # Create metadata
    print("\n7. Creating metadata...")
    metadata = create_metadata(percentiles, n_samples, rmse, config.model.version)

    # Save model
    print("\n8. Saving model and metadata...")
    output_path = Path(config.model.path)
    save_model(model, metadata, output_path)

    print("\n" + "=" * 60)
    print("Model creation complete!")
    print("=" * 60)
    print("\nNext steps:")
    print("1. Replace this dummy model with a trained model on real data")
    print("   Run: uv run python scripts/train_from_clickhouse.py")
    print("2. Start the ML service:")
    print("   Run: uv run python -m src.main")
    print("3. Update percentiles based on actual sensor data")


if __name__ == '__main__':
    main()
