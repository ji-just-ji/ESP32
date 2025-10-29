"""
ML Service main application.

Orchestrates MQTT communication, model loading, and inference.
"""

import sys
import signal
import logging
import yaml
from pathlib import Path
from typing import Dict, Optional
from pythonjsonlogger import jsonlogger

from .mqtt_client import MQTTClient
from .model_loader import ModelLoader
from .predictor import Predictor


class MLService:
    """Main ML service application."""

    def __init__(self, config_path: str):
        """
        Initialize ML service.

        Args:
            config_path: Path to config.yaml file
        """
        self.config = self._load_config(config_path)
        self._setup_logging()

        self.mqtt_client: Optional[MQTTClient] = None
        self.predictor: Optional[Predictor] = None
        self.running = False

        logger.info("ML Service initialized")
        logger.info(f"Configuration loaded from {config_path}")

    def _load_config(self, config_path: str) -> Dict:
        """Load configuration from YAML file."""
        config_file = Path(config_path)

        if not config_file.exists():
            raise FileNotFoundError(f"Config file not found: {config_path}")

        with open(config_file, 'r') as f:
            config = yaml.safe_load(f)

        return config

    def _setup_logging(self) -> None:
        """Setup logging based on configuration."""
        log_level = self.config['logging']['level']
        log_format = self.config['logging']['format']

        # Get root logger
        root_logger = logging.getLogger()
        root_logger.setLevel(log_level)

        # Remove existing handlers
        root_logger.handlers.clear()

        # Create handler
        handler = logging.StreamHandler(sys.stdout)

        # Set formatter based on config
        if log_format == 'json':
            formatter = jsonlogger.JsonFormatter(
                '%(asctime)s %(name)s %(levelname)s %(message)s'
            )
        else:
            formatter = logging.Formatter(
                '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
            )

        handler.setFormatter(formatter)
        root_logger.addHandler(handler)

        # Set level for paho.mqtt to WARNING to reduce noise
        logging.getLogger('paho.mqtt').setLevel(logging.WARNING)

    def start(self) -> None:
        """Start the ML service."""
        try:
            logger.info("=" * 60)
            logger.info("Starting ML Service")
            logger.info("=" * 60)

            # Load model
            logger.info("Loading ML model...")
            model_path = self.config['model']['path']
            model_loader = ModelLoader(model_path)
            model, metadata = model_loader.load()

            logger.info(f"Model loaded: version={metadata.get('version', 'unknown')}")

            # Initialize predictor
            logger.info("Initializing predictor...")
            output_min = self.config['model']['output_range']['min']
            output_max = self.config['model']['output_range']['max']
            min_confidence = self.config['inference']['min_confidence']

            self.predictor = Predictor(
                model_loader=model_loader,
                output_min=output_min,
                output_max=output_max,
                min_confidence=min_confidence
            )

            # Log model info
            model_info = self.predictor.get_model_info()
            logger.info(f"Model info: {model_info}")

            # Initialize MQTT client
            logger.info("Initializing MQTT client...")
            mqtt_config = self.config['mqtt']

            self.mqtt_client = MQTTClient(
                broker=mqtt_config['broker'],
                port=mqtt_config['port'],
                client_id=mqtt_config['client_id'],
                qos=mqtt_config['qos'],
                keepalive=mqtt_config['keepalive'],
                reconnect_delay=mqtt_config['reconnect_delay']
            )

            # Set inference callback
            self.mqtt_client.set_inference_callback(self._handle_inference_request)

            # Connect to MQTT broker
            logger.info("Connecting to MQTT broker...")
            self.mqtt_client.connect(
                inference_topic=mqtt_config['topics']['inference_request'],
                window_control_topic_template=mqtt_config['topics']['window_control']
            )

            logger.info("=" * 60)
            logger.info("ML Service is running")
            logger.info(f"Subscribed to: {mqtt_config['topics']['inference_request']}")
            logger.info(f"Publishing to: {mqtt_config['topics']['window_control']}")
            logger.info("=" * 60)

            self.running = True

            # Setup signal handlers for graceful shutdown
            signal.signal(signal.SIGINT, self._signal_handler)
            signal.signal(signal.SIGTERM, self._signal_handler)

            # Keep the service running
            while self.running:
                import time
                time.sleep(1)

        except Exception as e:
            logger.error(f"Failed to start ML service: {e}", exc_info=True)
            raise

    def _handle_inference_request(self, request: Dict) -> None:
        """
        Handle incoming inference request.

        Args:
            request: Inference request dictionary
        """
        try:
            device_id = request['device_id']

            logger.info(
                f"Inference request from {device_id}: "
                f"temp={request['temperature']:.2f}, "
                f"humidity={request['humidity']:.2f}, "
                f"volume={request['sound_volume']:.2f}"
            )

            # Run prediction
            prediction, should_publish = self.predictor.predict(request)

            # Publish if confidence threshold met
            if should_publish:
                self.mqtt_client.publish_window_control(device_id, prediction)
                logger.info(
                    f"Window control published for {device_id}: "
                    f"position={prediction['position']:.2f}%, "
                    f"confidence={prediction['confidence']:.3f}"
                )
            else:
                logger.warning(
                    f"Prediction for {device_id} not published "
                    f"(confidence {prediction['confidence']:.3f} below threshold)"
                )

        except Exception as e:
            logger.error(f"Error handling inference request: {e}", exc_info=True)

    def _signal_handler(self, signum, frame) -> None:
        """Handle shutdown signals."""
        signal_name = signal.Signals(signum).name
        logger.info(f"Received signal {signal_name}, shutting down gracefully...")
        self.stop()

    def stop(self) -> None:
        """Stop the ML service."""
        logger.info("Stopping ML service...")
        self.running = False

        if self.mqtt_client:
            self.mqtt_client.disconnect()

        logger.info("ML service stopped")


# Module-level logger
logger = logging.getLogger(__name__)


def main():
    """Main entry point."""
    # Default config path
    config_path = Path(__file__).parent.parent / 'config.yaml'

    # Allow override via environment variable
    import os
    config_path = os.getenv('ML_SERVICE_CONFIG', str(config_path))

    try:
        service = MLService(config_path)
        service.start()
    except KeyboardInterrupt:
        logger.info("Keyboard interrupt received")
    except Exception as e:
        logger.error(f"Fatal error: {e}", exc_info=True)
        sys.exit(1)


if __name__ == '__main__':
    main()
