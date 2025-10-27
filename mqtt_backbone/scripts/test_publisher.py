#!/usr/bin/env python3
"""
Test MQTT publisher for simulating multi-topic sensor data
Install dependencies: pip install paho-mqtt
"""

import json
import time
import random
import base64
import paho.mqtt.client as mqtt

# MQTT Configuration
BROKER = "localhost"
PORT = 1883
CLIENT_ID = "test-publisher"
DEVICE_ID = "sensor-001"

def on_connect(client, userdata, flags, rc):
    if rc == 0:
        print(f"Connected to MQTT Broker at {BROKER}:{PORT}")
    else:
        print(f"Failed to connect, return code {rc}")

def generate_temperature():
    """Generate random temperature value"""
    return round(random.uniform(15.0, 35.0), 2)

def generate_humidity():
    """Generate random humidity value"""
    return round(random.uniform(30.0, 90.0), 2)

def generate_audio():
    """Generate fake audio data (base64 encoded)"""
    # Simulate 2 seconds of audio at 16kHz (simplified)
    fake_audio = bytes([random.randint(0, 255) for _ in range(1000)])
    return {
        "data": base64.b64encode(fake_audio).decode('utf-8'),
        "sample_rate": 16000,
        "duration": 2.0
    }

def main():
    # Create MQTT client
    client = mqtt.Client(CLIENT_ID)
    client.on_connect = on_connect

    try:
        # Connect to broker
        client.connect(BROKER, PORT, 60)
        client.loop_start()

        print(f"Publishing sensor data for device: {DEVICE_ID}")
        print("Press Ctrl+C to stop\n")

        counter = 0
        # Publish data every 2 seconds
        while True:
            # Publish temperature (raw value)
            temp = generate_temperature()
            temp_topic = f"sensor/{DEVICE_ID}/temperature"
            client.publish(temp_topic, str(temp), qos=1)
            print(f"Published temp: {temp}°C to {temp_topic}")

            # Publish humidity (raw value)
            humidity = generate_humidity()
            humidity_topic = f"sensor/{DEVICE_ID}/humidity"
            client.publish(humidity_topic, str(humidity), qos=1)
            print(f"Published humidity: {humidity}% to {humidity_topic}")

            # Publish audio every 5th iteration (10 seconds)
            counter += 1
            if counter % 5 == 0:
                audio = generate_audio()
                audio_topic = f"sensor/{DEVICE_ID}/audio"
                audio_payload = json.dumps(audio)
                client.publish(audio_topic, audio_payload, qos=1)
                print(f"Published audio: {audio['duration']}s @ {audio['sample_rate']}Hz to {audio_topic}")

            print()
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
