"""
MQTT client for ML service communication.

Handles subscribing to inference requests and publishing window control commands.
"""

import paho.mqtt.client as mqtt
import json
import logging
import time
from typing import Callable, Dict, Optional
from threading import Event

from .config import MQTTConfig

logger = logging.getLogger(__name__)


class MQTTClient:
    """MQTT client wrapper for ML service."""

    def __init__(self, mqtt_config: MQTTConfig):
        """
        Initialize MQTT client.

        Args:
            mqtt_config: MQTTConfig object with all MQTT settings
        """
        self.config = mqtt_config

        self.client = mqtt.Client(client_id=mqtt_config.client_id)
        self.client.on_connect = self._on_connect
        self.client.on_disconnect = self._on_disconnect
        self.client.on_message = self._on_message

        self.inference_callback: Optional[Callable] = None

        self.connected = Event()
        self.should_reconnect = True

        logger.info(
            f"MQTT client initialized: broker={mqtt_config.broker}, "
            f"client_id={mqtt_config.client_id}, qos={mqtt_config.qos}"
        )

    def set_inference_callback(self, callback: Callable[[Dict], None]) -> None:
        """
        Set callback for handling inference requests.

        Args:
            callback: Function that takes inference request dict as input
        """
        self.inference_callback = callback
        logger.info("Inference callback registered")

    def connect(self) -> None:
        """Connect to MQTT broker and subscribe to inference topic."""
        logger.info(f"Connecting to MQTT broker at {self.config.broker}")

        # Parse broker URL (tcp://host:port)
        broker_url = self.config.broker
        if broker_url.startswith('tcp://'):
            broker_url = broker_url[6:]  # Remove 'tcp://' prefix

        # Split host and port
        if ':' in broker_url:
            host, port_str = broker_url.split(':')
            port = int(port_str)
        else:
            host = broker_url
            port = 1883  # Default MQTT port

        try:
            self.client.connect(host, port, self.config.keepalive)
            self.client.loop_start()

            # Wait for connection (with timeout)
            if not self.connected.wait(timeout=10):
                logger.error("Connection timeout")
                raise ConnectionError("Failed to connect to MQTT broker")

            logger.info("Connected to MQTT broker")

        except Exception as e:
            logger.error(f"Failed to connect: {e}")
            raise

    def _on_connect(self, client, userdata, flags, rc):
        """Callback for when client connects to broker."""
        if rc == 0:
            logger.info("Connected to MQTT broker successfully")
            self.connected.set()

            # Subscribe to inference request topic
            logger.info(f"Subscribing to {self.config.inference_topic}")
            client.subscribe(self.config.inference_topic, qos=self.config.qos)
        else:
            logger.error(f"Connection failed with code {rc}")
            self.connected.clear()

    def _on_disconnect(self, client, userdata, rc):
        """Callback for when client disconnects from broker."""
        self.connected.clear()

        if rc != 0:
            logger.warning(f"Unexpected disconnection (rc={rc})")

            if self.should_reconnect:
                logger.info(f"Attempting to reconnect in {self.config.reconnect_delay}s...")
                time.sleep(self.config.reconnect_delay)
                try:
                    client.reconnect()
                except Exception as e:
                    logger.error(f"Reconnection failed: {e}")
        else:
            logger.info("Disconnected from MQTT broker")

    def _on_message(self, client, userdata, msg):
        """Callback for when a message is received."""
        try:
            topic = msg.topic
            payload = msg.payload.decode('utf-8')

            logger.debug(f"Received message on {topic}: {payload[:100]}...")

            # Parse JSON payload
            try:
                data = json.loads(payload)
            except json.JSONDecodeError as e:
                logger.error(f"Failed to parse JSON payload: {e}")
                return

            # Handle inference requests
            if mqtt.topic_matches_sub(self.config.inference_topic, topic):
                self._handle_inference_request(data)
            else:
                logger.warning(f"Received message on unexpected topic: {topic}")

        except Exception as e:
            logger.error(f"Error processing message: {e}", exc_info=True)

    def _handle_inference_request(self, request: Dict) -> None:
        """
        Handle incoming inference request.

        Args:
            request: Inference request dictionary
        """
        try:
            # Validate required fields
            required_fields = ['device_id', 'temperature', 'humidity', 'sound_volume']
            missing_fields = [f for f in required_fields if f not in request]

            if missing_fields:
                logger.error(f"Invalid inference request, missing: {missing_fields}")
                return

            logger.info(f"Processing inference request for device: {request['device_id']}")

            # Call registered callback
            if self.inference_callback:
                self.inference_callback(request)
            else:
                logger.warning("No inference callback registered")

        except Exception as e:
            logger.error(f"Error handling inference request: {e}", exc_info=True)

    def publish_window_control(self, device_id: str, prediction: Dict) -> None:
        """
        Publish window control command.

        Args:
            device_id: Device ID to publish to
            prediction: Prediction dictionary with position, confidence, etc.
        """
        try:
            # Build topic from template
            topic = self.config.window_control_topic.replace('{device_id}', device_id)

            # Serialize to JSON
            payload = json.dumps(prediction)

            # Publish
            result = self.client.publish(topic, payload, qos=self.config.qos)

            if result.rc == mqtt.MQTT_ERR_SUCCESS:
                logger.info(
                    f"Published window control to {topic}: "
                    f"position={prediction['position']:.2f}%"
                )
            else:
                logger.error(f"Failed to publish to {topic}: rc={result.rc}")

        except Exception as e:
            logger.error(f"Error publishing window control: {e}", exc_info=True)

    def disconnect(self) -> None:
        """Disconnect from MQTT broker."""
        logger.info("Disconnecting from MQTT broker")
        self.should_reconnect = False
        self.client.loop_stop()
        self.client.disconnect()
        self.connected.clear()

    def is_connected(self) -> bool:
        """Check if client is connected."""
        return self.connected.is_set()
