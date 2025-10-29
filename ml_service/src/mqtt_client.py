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

logger = logging.getLogger(__name__)


class MQTTClient:
    """MQTT client wrapper for ML service."""

    def __init__(self, broker: str, port: int, client_id: str,
                 qos: int = 1, keepalive: int = 60, reconnect_delay: int = 5):
        """
        Initialize MQTT client.

        Args:
            broker: MQTT broker hostname
            port: MQTT broker port
            client_id: Unique client ID
            qos: Quality of Service level (0, 1, or 2)
            keepalive: Keepalive interval in seconds
            reconnect_delay: Delay between reconnection attempts
        """
        self.broker = broker
        self.port = port
        self.client_id = client_id
        self.qos = qos
        self.keepalive = keepalive
        self.reconnect_delay = reconnect_delay

        self.client = mqtt.Client(client_id=client_id)
        self.client.on_connect = self._on_connect
        self.client.on_disconnect = self._on_disconnect
        self.client.on_message = self._on_message

        self.inference_callback: Optional[Callable] = None
        self.inference_topic: Optional[str] = None
        self.window_control_topic_template: Optional[str] = None

        self.connected = Event()
        self.should_reconnect = True

        logger.info(
            f"MQTT client initialized: broker={broker}:{port}, "
            f"client_id={client_id}, qos={qos}"
        )

    def set_inference_callback(self, callback: Callable[[Dict], None]) -> None:
        """
        Set callback for handling inference requests.

        Args:
            callback: Function that takes inference request dict as input
        """
        self.inference_callback = callback
        logger.info("Inference callback registered")

    def connect(self, inference_topic: str, window_control_topic_template: str) -> None:
        """
        Connect to MQTT broker and subscribe to inference topic.

        Args:
            inference_topic: Topic pattern for inference requests (e.g., 'ml/inference/request/#')
            window_control_topic_template: Template for window control (e.g., 'window/{device_id}/control')
        """
        self.inference_topic = inference_topic
        self.window_control_topic_template = window_control_topic_template

        logger.info(f"Connecting to MQTT broker at {self.broker}:{self.port}")

        try:
            self.client.connect(self.broker, self.port, self.keepalive)
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
            if self.inference_topic:
                logger.info(f"Subscribing to {self.inference_topic}")
                client.subscribe(self.inference_topic, qos=self.qos)
        else:
            logger.error(f"Connection failed with code {rc}")
            self.connected.clear()

    def _on_disconnect(self, client, userdata, rc):
        """Callback for when client disconnects from broker."""
        self.connected.clear()

        if rc != 0:
            logger.warning(f"Unexpected disconnection (rc={rc})")

            if self.should_reconnect:
                logger.info(f"Attempting to reconnect in {self.reconnect_delay}s...")
                time.sleep(self.reconnect_delay)
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
            if self.inference_topic and mqtt.topic_matches_sub(self.inference_topic, topic):
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
            topic = self.window_control_topic_template.replace('{device_id}', device_id)

            # Serialize to JSON
            payload = json.dumps(prediction)

            # Publish
            result = self.client.publish(topic, payload, qos=self.qos)

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
