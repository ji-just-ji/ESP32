#!/usr/bin/env python3
"""
Test MQTT subscriber for monitoring window actions
Install dependencies: pip install paho-mqtt
"""

import json
import paho.mqtt.client as mqtt

# MQTT Configuration
BROKER = "localhost"
PORT = 1883
TOPIC = "window/action"
CLIENT_ID = "test-subscriber"

def on_connect(client, userdata, flags, rc):
    if rc == 0:
        print(f"Connected to MQTT Broker at {BROKER}:{PORT}")
        client.subscribe(TOPIC)
        print(f"Subscribed to topic: {TOPIC}\n")
    else:
        print(f"Failed to connect, return code {rc}")

def on_message(client, userdata, msg):
    """Handle incoming messages"""
    try:
        data = json.loads(msg.payload.decode())
        print(f"[{data['timestamp']}] Device: {data['device_id']}")
        print(f"  Action: {data['action'].upper()}")
        print(f"  Conditions: Temp={data['temperature']}°C, "
              f"Humidity={data['humidity']}%, "
              f"Sound={data['sound']}dB")
        print("-" * 60)
    except Exception as e:
        print(f"Error processing message: {e}")
        print(f"Raw payload: {msg.payload.decode()}")

def main():
    # Create MQTT client
    client = mqtt.Client(CLIENT_ID)
    client.on_connect = on_connect
    client.on_message = on_message

    try:
        # Connect to broker
        client.connect(BROKER, PORT, 60)

        print("Monitoring window actions...")
        print("Press Ctrl+C to stop\n")

        # Keep listening
        client.loop_forever()

    except KeyboardInterrupt:
        print("\nStopping subscriber...")
    except Exception as e:
        print(f"Error: {e}")
    finally:
        client.disconnect()
        print("Disconnected from broker")

if __name__ == "__main__":
    main()
