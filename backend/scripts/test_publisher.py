#!/usr/bin/env python3
"""
Test MQTT publisher for simulating sensor data
Install dependencies: pip install paho-mqtt
"""

import json
import time
import random
from datetime import datetime
import paho.mqtt.client as mqtt

# MQTT Configuration
BROKER = "localhost"
PORT = 1883
TOPIC = "sensor/data"
CLIENT_ID = "test-publisher"

def on_connect(client, userdata, flags, rc):
    if rc == 0:
        print(f"Connected to MQTT Broker at {BROKER}:{PORT}")
    else:
        print(f"Failed to connect, return code {rc}")

def generate_sensor_data(device_id="sensor-001"):
    """Generate random sensor data"""
    return {
        "timestamp": datetime.utcnow().isoformat() + "Z",
        "device_id": device_id,
        "temperature": round(random.uniform(15.0, 35.0), 2),
        "humidity": round(random.uniform(30.0, 90.0), 2),
        "sound": round(random.uniform(30.0, 80.0), 2)
    }

def main():
    # Create MQTT client
    client = mqtt.Client(CLIENT_ID)
    client.on_connect = on_connect

    try:
        # Connect to broker
        client.connect(BROKER, PORT, 60)
        client.loop_start()

        print(f"Publishing sensor data to topic: {TOPIC}")
        print("Press Ctrl+C to stop\n")

        # Publish data every 2 seconds
        while True:
            data = generate_sensor_data()
            payload = json.dumps(data)

            result = client.publish(TOPIC, payload, qos=1)

            if result.rc == mqtt.MQTT_ERR_SUCCESS:
                print(f"Published: Temp={data['temperature']}°C, "
                      f"Humidity={data['humidity']}%, "
                      f"Sound={data['sound']}dB")
            else:
                print(f"Failed to publish message")

            time.sleep(2)

    except KeyboardInterrupt:
        print("\nStopping publisher...")
    except Exception as e:
        print(f"Error: {e}")
    finally:
        client.loop_stop()
        client.disconnect()
        print("Disconnected from broker")

if __name__ == "__main__":
    main()
