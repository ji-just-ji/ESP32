"""
Model loader for XGBoost models with metadata.

Handles loading of trained models and associated normalization metadata.
"""

import xgboost as xgb
import json
import logging
from pathlib import Path
from typing import Dict, Tuple, Any

logger = logging.getLogger(__name__)


class ModelLoader:
    """Loads XGBoost models and associated metadata."""

    def __init__(self, model_path: str):
        """
        Initialize model loader.

        Args:
            model_path: Path to the .json model file (XGBoost JSON format)
        """
        self.model_path = Path(model_path)
        self.metadata_path = Path(str(model_path) + '.meta')

        logger.info(f"ModelLoader initialized")
        logger.info(f"Model path: {self.model_path}")
        logger.info(f"Metadata path: {self.metadata_path}")

        self.model: xgb.Booster = None
        self.metadata: Dict[str, Any] = None

    def load(self) -> Tuple[xgb.Booster, Dict[str, Any]]:
        """
        Load model and metadata.

        Returns:
            (model, metadata) tuple where metadata contains:
                - version: Model version string
                - percentiles: Dict of feature percentiles
                - training_info: Training metadata (optional)

        Raises:
            FileNotFoundError: If model or metadata file not found
            ValueError: If metadata is invalid
        """
        # Load XGBoost model
        if not self.model_path.exists():
            raise FileNotFoundError(f"Model file not found: {self.model_path}")

        logger.info(f"Loading XGBoost model from {self.model_path}")

        try:
            # Create Booster and load from JSON
            self.model = xgb.Booster()
            self.model.load_model(str(self.model_path))

            logger.info(f"XGBoost model loaded successfully")

        except Exception as e:
            logger.error(f"Failed to load XGBoost model: {e}")
            raise

        # Load metadata from separate file
        if not self.metadata_path.exists():
            raise FileNotFoundError(
                f"Metadata file not found: {self.metadata_path}. "
                "Expected JSON file with percentile information."
            )

        logger.info(f"Loading metadata from {self.metadata_path}")

        try:
            with open(self.metadata_path, 'r') as f:
                self.metadata = json.load(f)
        except Exception as e:
            logger.error(f"Failed to load metadata: {e}")
            raise

        # Validate metadata
        self._validate_metadata(self.metadata)

        logger.info(f"Model version: {self.metadata.get('version', 'unknown')}")
        logger.info(f"Percentiles: {self.metadata['percentiles']}")

        return self.model, self.metadata

    def _validate_metadata(self, metadata: Dict[str, Any]) -> None:
        """
        Validate metadata structure.

        Args:
            metadata: Metadata dictionary

        Raises:
            ValueError: If metadata is invalid
        """
        required_fields = ['percentiles']
        for field in required_fields:
            if field not in metadata:
                raise ValueError(f"Missing required metadata field: {field}")

        # Validate percentiles
        percentiles = metadata['percentiles']
        required_features = ['temperature', 'humidity', 'sound_volume']

        for feature in required_features:
            if feature not in percentiles:
                raise ValueError(f"Missing percentile data for feature: {feature}")

            p_data = percentiles[feature]
            if not isinstance(p_data, (list, tuple)) or len(p_data) != 2:
                raise ValueError(
                    f"Invalid percentile format for {feature}. "
                    "Expected [p10, p90] list/tuple."
                )

            p_low, p_high = p_data
            if p_low >= p_high:
                raise ValueError(
                    f"Invalid percentile range for {feature}: "
                    f"p10={p_low} must be < p90={p_high}"
                )

        logger.info("Metadata validation passed")

    def get_model(self) -> xgb.Booster:
        """Get the loaded model."""
        if self.model is None:
            raise RuntimeError("Model not loaded. Call load() first.")
        return self.model

    def get_metadata(self) -> Dict[str, Any]:
        """Get the loaded metadata."""
        if self.metadata is None:
            raise RuntimeError("Metadata not loaded. Call load() first.")
        return self.metadata

    def get_percentiles(self) -> Dict[str, Tuple[float, float]]:
        """
        Get percentiles as a dictionary of tuples.

        Returns:
            Dict mapping feature names to (p10, p90) tuples
        """
        if self.metadata is None:
            raise RuntimeError("Metadata not loaded. Call load() first.")

        percentiles = {}
        for feature, values in self.metadata['percentiles'].items():
            percentiles[feature] = tuple(values)

        return percentiles
