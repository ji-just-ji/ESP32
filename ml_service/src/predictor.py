"""
ML predictor for window control inference.

Combines feature processing and model inference to predict window positions.
"""

import torch
import torch.nn as nn
import numpy as np
import logging
from typing import Dict, Tuple
from datetime import datetime

from .feature_processor import FeatureProcessor
from .model_loader import ModelLoader

logger = logging.getLogger(__name__)


class Predictor:
    """Handles ML inference for window position prediction."""

    def __init__(self, model_loader: ModelLoader, output_min: float = 0.0,
                 output_max: float = 100.0, min_confidence: float = 0.0):
        """
        Initialize predictor.

        Args:
            model_loader: Loaded ModelLoader instance
            output_min: Minimum output value (default 0%)
            output_max: Maximum output value (default 100%)
            min_confidence: Minimum confidence threshold to publish predictions
        """
        self.model = model_loader.get_model()
        self.device = next(self.model.parameters()).device
        self.metadata = model_loader.get_metadata()

        # Initialize feature processor with percentiles
        percentiles = model_loader.get_percentiles()
        self.feature_processor = FeatureProcessor(percentiles)

        self.output_min = output_min
        self.output_max = output_max
        self.min_confidence = min_confidence

        logger.info(
            f"Predictor initialized: output_range=[{output_min}, {output_max}], "
            f"min_confidence={min_confidence}"
        )

    def predict(self, inference_request: Dict) -> Tuple[Dict, bool]:
        """
        Make window position prediction from inference request.

        Args:
            inference_request: Dict containing:
                - device_id: str
                - timestamp: str (ISO format)
                - temperature: float
                - humidity: float
                - sound_volume: float

        Returns:
            (prediction_dict, should_publish) tuple where:
                - prediction_dict contains the response to publish
                - should_publish is True if confidence >= min_confidence

        Raises:
            ValueError: If input validation fails
        """
        try:
            # Extract features
            features = {
                'temperature': inference_request['temperature'],
                'humidity': inference_request['humidity'],
                'sound_volume': inference_request['sound_volume']
            }

            device_id = inference_request['device_id']
            timestamp = inference_request.get(
                'timestamp', datetime.utcnow().isoformat() + 'Z')

            logger.debug(
                f"Processing inference for device {device_id}: {features}")

            # Validate input
            is_valid, error_msg = self.feature_processor.validate_input(
                features)
            if not is_valid:
                logger.error(
                    f"Input validation failed for {device_id}: {error_msg}")
                raise ValueError(error_msg)

            # Normalize features
            normalized_features = self.feature_processor.normalize(features)

            # Run inference
            position, confidence = self._run_inference(normalized_features)

            # Denormalize output
            position = self.feature_processor.denormalize_output(
                position, self.output_min, self.output_max
            )

            # Build response
            prediction = {
                'device_id': device_id,
                'timestamp': datetime.utcnow().isoformat() + 'Z',
                'position': float(position),
                'confidence': float(confidence),
                'features_used': {
                    'temperature': features['temperature'],
                    'humidity': features['humidity'],
                    'sound_volume': features['sound_volume']
                }
            }

            # Determine if should publish based on confidence
            should_publish = confidence >= self.min_confidence

            if should_publish:
                logger.info(
                    f"Prediction for {device_id}: position={position:.2f}%, "
                    f"confidence={confidence:.3f}"
                )
            else:
                logger.warning(
                    f"Prediction for {device_id} below confidence threshold: "
                    f"{confidence:.3f} < {self.min_confidence}"
                )

            return prediction, should_publish

        except KeyError as e:
            error_msg = f"Missing required field in inference request: {e}"
            logger.error(error_msg)
            raise ValueError(error_msg)
        except Exception as e:
            logger.error(f"Prediction failed: {e}", exc_info=True)
            raise

    def _run_inference(self, normalized_features: np.ndarray) -> Tuple[float, float]:
        """
        Run model inference and calculate confidence.

        Args:
            normalized_features: Normalized feature array [3,]

        Returns:
            (position, confidence) tuple where:
                - position: Predicted window position in [0, 1]
                - confidence: Confidence score in [0, 1]
        """
        # Convert to tensor
        input_tensor = torch.from_numpy(
            normalized_features).float().unsqueeze(0)
        input_tensor = input_tensor.to(self.device)

        # Run inference
        with torch.no_grad():
            output = self.model(input_tensor)
            position = output.item()

        # Calculate confidence
        # For now, use a simple heuristic based on distance from boundaries
        # More sophisticated: ensemble variance, dropout-based uncertainty, etc.
        confidence = self._calculate_confidence(position)

        return position, confidence

    def _calculate_confidence(self, position: float) -> float:
        """
        Calculate confidence score for prediction.

        Simple heuristic: Predictions near middle (0.5) are more confident
        than predictions at extremes (0 or 1).

        Args:
            position: Predicted position in [0, 1]

        Returns:
            Confidence score in [0, 1]
        """
        # Distance from boundaries (0 and 1)
        dist_from_boundary = min(position, 1.0 - position)

        # Map to confidence: 0.0 -> 0.5, 0.5 -> 1.0
        # Linear scaling: confidence = 0.5 + dist_from_boundary
        confidence = 0.5 + dist_from_boundary

        # Clamp to [0, 1]
        confidence = max(0.0, min(1.0, confidence))

        return confidence

    def get_model_info(self) -> Dict:
        """
        Get model information for logging/monitoring.

        Returns:
            Dict with model metadata
        """
        return {
            'version': self.metadata.get('version', 'unknown'),
            'device': str(self.device),
            'percentiles': self.metadata['percentiles'],
            'output_range': [self.output_min, self.output_max],
            'min_confidence': self.min_confidence
        }
