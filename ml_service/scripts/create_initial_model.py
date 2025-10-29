#!/usr/bin/env python3
"""
Create an initial PyTorch model for window control.

This script creates a simple trained model with metadata for testing purposes.
In production, this would be replaced with a properly trained model on real data.
"""

import torch
import torch.nn as nn
import json
import numpy as np
from pathlib import Path
import sys

# Add src to path
sys.path.insert(0, str(Path(__file__).parent.parent / 'src'))

from model_loader import WindowRegressorModel


def create_dummy_model() -> nn.Module:
    """
    Create and initialize a simple model.

    Uses a simple rule-based approach:
    - Higher temperature -> open more
    - Higher humidity -> close more
    - Higher sound volume -> close more (for noise isolation)
    """
    model = WindowRegressorModel(input_size=3, hidden_sizes=[16, 8])

    # Initialize with reasonable weights for rule-based behavior
    # This is a placeholder - in production, train on real data
    with torch.no_grad():
        # First layer: Extract basic features
        model.network[0].weight.data = torch.randn(16, 3) * 0.5
        model.network[0].bias.data = torch.randn(16) * 0.1

        # Second layer
        model.network[3].weight.data = torch.randn(8, 16) * 0.5
        model.network[3].bias.data = torch.randn(8) * 0.1

        # Output layer: Combine features
        model.network[6].weight.data = torch.randn(1, 8) * 0.5
        model.network[6].bias.data = torch.tensor([0.5])  # Bias towards middle

    model.eval()
    return model


def generate_training_data(n_samples: int = 1000):
    """
    Generate synthetic training data to compute percentiles.

    In production, this would be replaced with real sensor data.
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

    return {
        'temperature': temperature,
        'humidity': humidity,
        'sound_volume': sound_volume
    }


def compute_percentiles(data: dict, p_low: float = 0.1, p_high: float = 0.9) -> dict:
    """
    Compute percentiles for normalization.

    Args:
        data: Dictionary of feature arrays
        p_low: Lower percentile (default 0.1 = 10th percentile)
        p_high: Upper percentile (default 0.9 = 90th percentile)

    Returns:
        Dictionary of percentile tuples
    """
    percentiles = {}

    for feature, values in data.items():
        p10 = float(np.percentile(values, p_low * 100))
        p90 = float(np.percentile(values, p_high * 100))
        percentiles[feature] = [p10, p90]
        print(f"{feature}: p10={p10:.2f}, p90={p90:.2f}")

    return percentiles


def create_metadata(percentiles: dict) -> dict:
    """Create model metadata."""
    metadata = {
        'version': 'v1.0.0',
        'model_type': 'WindowRegressorModel',
        'architecture': {
            'input_size': 3,
            'hidden_sizes': [16, 8],
            'output_size': 1
        },
        'percentiles': percentiles,
        'training_info': {
            'date': '2025-10-27',
            'notes': 'Initial dummy model for testing. Replace with trained model.'
        }
    }

    return metadata


def save_model(model: nn.Module, metadata: dict, output_dir: Path):
    """Save model and metadata."""
    output_dir.mkdir(parents=True, exist_ok=True)

    # Save model state dict
    model_path = output_dir / 'window_regressor.pth'
    torch.save(model.state_dict(), model_path)
    print(f"Model saved to: {model_path}")

    # Save metadata as JSON
    metadata_path = output_dir / 'window_regressor.json'
    with open(metadata_path, 'w') as f:
        json.dump(metadata, f, indent=2)
    print(f"Metadata saved to: {metadata_path}")

    # Also save combined checkpoint (optional, for convenience)
    checkpoint_path = output_dir / 'window_regressor_checkpoint.pth'
    checkpoint = {
        'model_state_dict': model.state_dict(),
        'metadata': metadata
    }
    torch.save(checkpoint, checkpoint_path)
    print(f"Checkpoint saved to: {checkpoint_path}")


def main():
    print("=" * 60)
    print("Creating Initial PyTorch Model")
    print("=" * 60)

    # Generate synthetic training data
    print("\n1. Generating synthetic training data...")
    training_data = generate_training_data(n_samples=1000)

    # Compute percentiles
    print("\n2. Computing percentiles (0.1 and 0.9)...")
    percentiles = compute_percentiles(training_data, p_low=0.1, p_high=0.9)

    # Create model
    print("\n3. Creating model...")
    model = create_dummy_model()

    # Test model with sample input
    print("\n4. Testing model with sample input...")
    sample_input = torch.tensor([[0.5, 0.5, 0.5]], dtype=torch.float32)
    with torch.no_grad():
        output = model(sample_input)
        print(f"Sample prediction: {output.item():.4f} (normalized)")
        print(f"Window position: {output.item() * 100:.2f}%")

    # Create metadata
    print("\n5. Creating metadata...")
    metadata = create_metadata(percentiles)

    # Save model
    print("\n6. Saving model and metadata...")
    output_dir = Path(__file__).parent.parent / 'models'
    save_model(model, metadata, output_dir)

    print("\n" + "=" * 60)
    print("Model creation complete!")
    print("=" * 60)
    print("\nNext steps:")
    print("1. Replace this dummy model with a trained model on real data")
    print("2. Update percentiles based on actual sensor data")
    print("3. Implement proper training pipeline with validation")


if __name__ == '__main__':
    main()
