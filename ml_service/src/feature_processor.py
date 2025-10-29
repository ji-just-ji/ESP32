"""
Feature preprocessing for ML inference.

Handles input normalization using percentiles computed from training data.
The percentiles are stored as model metadata and loaded at runtime.
"""

import numpy as np
from typing import Dict, Tuple
import logging

logger = logging.getLogger(__name__)


class FeatureProcessor:
    """Processes and normalizes input features for model inference."""

    def __init__(self, percentiles: Dict[str, Tuple[float, float]]):
        """
        Initialize the feature processor with percentile ranges.

        Args:
            percentiles: Dict mapping feature names to (p10, p90) tuples
                        e.g., {'temperature': (10.5, 34.8), 'humidity': (22.0, 78.5), ...}
        """
        self.percentiles = percentiles
        self.feature_names = ['temperature', 'humidity', 'sound_volume']

        # Validate percentiles
        for feature in self.feature_names:
            if feature not in percentiles:
                raise ValueError(f"Missing percentile range for feature: {feature}")

            p_low, p_high = percentiles[feature]
            if p_low >= p_high:
                raise ValueError(
                    f"Invalid percentile range for {feature}: "
                    f"p10={p_low} must be < p90={p_high}"
                )

        logger.info(f"FeatureProcessor initialized with percentiles: {percentiles}")

    def normalize(self, features: Dict[str, float]) -> np.ndarray:
        """
        Normalize input features using percentile-based scaling.

        Maps [p10, p90] -> [0, 1] with clipping for out-of-range values.

        Args:
            features: Dict with keys 'temperature', 'humidity', 'sound_volume'

        Returns:
            numpy array of normalized features [temp, humidity, volume]

        Raises:
            ValueError: If required features are missing
        """
        # Validate input
        for feature in self.feature_names:
            if feature not in features:
                raise ValueError(f"Missing required feature: {feature}")

        normalized = []

        for feature in self.feature_names:
            value = features[feature]
            p_low, p_high = self.percentiles[feature]

            # Normalize: (value - p10) / (p90 - p10)
            # Then clip to [0, 1]
            normalized_value = (value - p_low) / (p_high - p_low)
            normalized_value = np.clip(normalized_value, 0.0, 1.0)

            normalized.append(normalized_value)

            # Log warning if value is outside percentile range
            if value < p_low or value > p_high:
                logger.warning(
                    f"{feature}={value:.2f} outside training range "
                    f"[{p_low:.2f}, {p_high:.2f}]"
                )

        return np.array(normalized, dtype=np.float32)

    def denormalize_output(self, normalized_value: float,
                          output_min: float = 0.0,
                          output_max: float = 100.0) -> float:
        """
        Denormalize model output to target range.

        Args:
            normalized_value: Model output (typically in [0, 1])
            output_min: Minimum value of output range
            output_max: Maximum value of output range

        Returns:
            Denormalized value in [output_min, output_max]
        """
        # Scale from [0, 1] to [output_min, output_max]
        denormalized = normalized_value * (output_max - output_min) + output_min

        # Clip to valid range
        return np.clip(denormalized, output_min, output_max)

    def validate_input(self, features: Dict[str, float]) -> Tuple[bool, str]:
        """
        Validate input features for sanity.

        Args:
            features: Input feature dictionary

        Returns:
            (is_valid, error_message) tuple
        """
        # Check all required features present
        for feature in self.feature_names:
            if feature not in features:
                return False, f"Missing required feature: {feature}"

        # Check for NaN or infinite values
        for feature, value in features.items():
            if not np.isfinite(value):
                return False, f"Invalid value for {feature}: {value}"

        # Basic range checks (very permissive, just sanity)
        if features['temperature'] < -50 or features['temperature'] > 60:
            return False, f"Temperature {features['temperature']} out of reasonable range"

        if features['humidity'] < 0 or features['humidity'] > 100:
            return False, f"Humidity {features['humidity']} out of valid range [0, 100]"

        if features['sound_volume'] < 0 or features['sound_volume'] > 120:
            return False, f"Sound volume {features['sound_volume']} out of reasonable range"

        return True, ""
