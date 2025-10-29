"""
Model loader for PyTorch models with metadata.

Handles loading of trained models and associated normalization metadata.
"""

import torch
import torch.nn as nn
import json
import logging
from pathlib import Path
from typing import Dict, Tuple, Any

logger = logging.getLogger(__name__)


class WindowRegressorModel(nn.Module):
    """
    Simple neural network for window position prediction.

    Architecture: 3 inputs -> hidden layers -> 1 output (sigmoid)
    Input: [temperature, humidity, sound_volume] (normalized)
    Output: window position in [0, 1] (will be scaled to 0-100%)
    """

    def __init__(self, input_size: int = 3, hidden_sizes: list = [16, 8]):
        super(WindowRegressorModel, self).__init__()

        layers = []
        prev_size = input_size

        # Build hidden layers
        for hidden_size in hidden_sizes:
            layers.append(nn.Linear(prev_size, hidden_size))
            layers.append(nn.ReLU())
            layers.append(nn.Dropout(0.2))
            prev_size = hidden_size

        # Output layer with sigmoid to constrain to [0, 1]
        layers.append(nn.Linear(prev_size, 1))
        layers.append(nn.Sigmoid())

        self.network = nn.Sequential(*layers)

    def forward(self, x):
        return self.network(x)


class ModelLoader:
    """Loads PyTorch models and associated metadata."""

    def __init__(self, model_path: str, device: str = None):
        """
        Initialize model loader.

        Args:
            model_path: Path to the .pth model file
            device: Device to load model on ('cpu', 'cuda', 'mps', or None for auto)
        """
        self.model_path = Path(model_path)
        self.metadata_path = self.model_path.with_suffix('.json')

        # Auto-detect device if not specified
        if device is None:
            if torch.cuda.is_available():
                self.device = torch.device('cuda')
            elif torch.backends.mps.is_available():
                self.device = torch.device('mps')
            else:
                self.device = torch.device('cpu')
        else:
            self.device = torch.device(device)

        logger.info(f"ModelLoader initialized with device: {self.device}")

        self.model = None
        self.metadata = None

    def load(self) -> Tuple[nn.Module, Dict[str, Any]]:
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
        # Load model
        if not self.model_path.exists():
            raise FileNotFoundError(f"Model file not found: {self.model_path}")

        logger.info(f"Loading model from {self.model_path}")

        try:
            # Load state dict
            checkpoint = torch.load(self.model_path, map_location=self.device)

            # Handle different checkpoint formats
            if isinstance(checkpoint, dict) and 'model_state_dict' in checkpoint:
                # Checkpoint contains both state dict and metadata
                state_dict = checkpoint['model_state_dict']

                # Extract metadata if embedded in checkpoint
                if 'metadata' in checkpoint:
                    self.metadata = checkpoint['metadata']
            else:
                # Checkpoint is just the state dict
                state_dict = checkpoint

            # Instantiate model
            # TODO: Make architecture configurable
            self.model = WindowRegressorModel()
            self.model.load_state_dict(state_dict)
            self.model.to(self.device)
            self.model.eval()

            logger.info(f"Model loaded successfully on {self.device}")

        except Exception as e:
            logger.error(f"Failed to load model: {e}")
            raise

        # Load metadata from separate JSON file if not embedded
        if self.metadata is None:
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

    def get_model(self) -> nn.Module:
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
