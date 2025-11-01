"""
ML Service main application.

Orchestrates MQTT communication, model loading, and inference.
"""

import sys
import signal
import logging
import subprocess
from pathlib import Path
from typing import Dict, Optional
from pythonjsonlogger import jsonlogger

from .config import load_config
from .mqtt_client import MQTTClient
from .model_loader import ModelLoader
from .predictor import Predictor


class MLService:
    """Main ML service application."""

    def __init__(self):
        """Initialize ML service."""
        # Load configuration
        try:
            self.config = load_config()
        except Exception as e:
            print(f"ERROR: Failed to load configuration: {e}")
            sys.exit(1)

        self._setup_logging()

        self.mqtt_client: Optional[MQTTClient] = None
        self.predictor: Optional[Predictor] = None
        self.running = False

        logger.info("ML Service initialized")

    def _setup_logging(self) -> None:
        """Setup logging based on configuration."""
        log_level = self.config.logging.level
        log_format = self.config.logging.format

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

    def _check_and_train_model(self) -> None:
        """
        Check if model exists, and train if necessary.

        This method implements the auto-training logic:
        1. Check if model file exists
        2. If not and auto-train enabled, try training from ClickHouse
        3. If ClickHouse training fails, fallback to synthetic data
        4. If auto-train disabled and no model, exit with error
        """
        model_path = Path(self.config.model.path)

        if model_path.exists():
            logger.info(f"Model found at {model_path}")
            return

        logger.warning(f"Model not found at {model_path}")

        if not self.config.training.auto_train:
            logger.error("Auto-train is disabled. Cannot proceed without model.")
            logger.error("Please train a model manually:")
            logger.error("  Option 1: uv run python scripts/train_from_clickhouse.py")
            logger.error("  Option 2: uv run python scripts/create_initial_model.py")
            sys.exit(1)

        logger.info("Auto-train is enabled. Attempting to train model...")

        # Try training from ClickHouse first
        try:
            logger.info("Attempting to train from ClickHouse data...")
            self._run_training_script('scripts/train_from_clickhouse.py')
            logger.info("Successfully trained model from ClickHouse")
            return
        except Exception as e:
            logger.warning(f"ClickHouse training failed: {e}")
            logger.info("Falling back to synthetic data...")

        # Fallback to synthetic data
        try:
            logger.info("Training model with synthetic data...")
            self._run_training_script('scripts/create_initial_model.py')
            logger.info("Successfully created initial model with synthetic data")
        except Exception as e:
            logger.error(f"Failed to create initial model: {e}")
            logger.error("Cannot proceed without a model. Exiting.")
            sys.exit(1)

    def _run_training_script(self, script_path: str) -> None:
        """
        Run a training script using uv.

        Args:
            script_path: Relative path to training script

        Raises:
            RuntimeError: If training script fails
        """
        # Get absolute path to script
        ml_service_dir = Path(__file__).parent.parent
        script_full_path = ml_service_dir / script_path

        if not script_full_path.exists():
            raise FileNotFoundError(f"Training script not found: {script_full_path}")

        # Run script with uv
        cmd = ['uv', 'run', 'python', str(script_full_path)]
        logger.info(f"Running: {' '.join(cmd)}")

        result = subprocess.run(
            cmd,
            cwd=str(ml_service_dir),
            capture_output=True,
            text=True
        )

        if result.returncode != 0:
            logger.error(f"Training script failed with code {result.returncode}")
            logger.error(f"stdout: {result.stdout}")
            logger.error(f"stderr: {result.stderr}")
            raise RuntimeError(f"Training script failed: {result.stderr}")

        logger.info(f"Training script output:\n{result.stdout}")

    def start(self) -> None:
        """Start the ML service."""
        try:
            logger.info("=" * 60)
            logger.info("Starting ML Service")
            logger.info("=" * 60)

            # Check model and train if necessary
            logger.info("Checking for model...")
            self._check_and_train_model()

            # Load model
            logger.info("Loading ML model...")
            model_path = self.config.model.path
            model_loader = ModelLoader(model_path)
            model, metadata = model_loader.load()

            logger.info(f"Model loaded: version={metadata.get('version', 'unknown')}")
            logger.info(f"Model type: {metadata.get('model_type', 'unknown')}")

            # Initialize predictor
            logger.info("Initializing predictor...")
            self.predictor = Predictor(
                model_loader=model_loader,
                output_min=self.config.model.output_min,
                output_max=self.config.model.output_max,
                min_confidence=self.config.inference.min_confidence
            )

            # Log model info
            model_info = self.predictor.get_model_info()
            logger.info(f"Model info: {model_info}")

            # Initialize MQTT client
            logger.info("Initializing MQTT client...")
            self.mqtt_client = MQTTClient(self.config.mqtt)

            # Set inference callback
            self.mqtt_client.set_inference_callback(self._handle_inference_request)

            # Connect to MQTT broker
            logger.info("Connecting to MQTT broker...")
            self.mqtt_client.connect()

            logger.info("=" * 60)
            logger.info("ML Service is running")
            logger.info(f"Subscribed to: {self.config.mqtt.inference_topic}")
            logger.info(f"Publishing to: {self.config.mqtt.window_control_topic}")
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
    try:
        service = MLService()
        service.start()
    except KeyboardInterrupt:
        logger.info("Keyboard interrupt received")
    except Exception as e:
        logger.error(f"Fatal error: {e}", exc_info=True)
        sys.exit(1)


if __name__ == '__main__':
    main()
