import numpy as np
import paho.mqtt.client as mqtt
from config import BIT_DEPTH, ENDIANNESS, MAX_AMPLITUDE, MQTT_BROKER, MQTT_PORT, NORMALIZE, TOPIC_REF, TOPIC_ERROR, TOPIC_SPEAKER, CHUNK_SIZE
from anc_system import AdaptiveANC

# Create ANC processor
anc = AdaptiveANC()

# Buffers for reference and error chunks
ref_chunk_buffer = np.zeros(CHUNK_SIZE, dtype=np.int16)
error_chunk_buffer = np.zeros(CHUNK_SIZE, dtype=np.int16)


def on_message(client, userdata, msg):
    try:
        global ref_chunk_buffer, error_chunk_buffer

        if msg.topic == TOPIC_REF:
            raw_bytes = msg.payload

            dtype = np.int16 if BIT_DEPTH == 16 else np.int32
            audio_array = np.frombuffer(raw_bytes, dtype=dtype)
            if ENDIANNESS == 'big':
                audio_array = audio_array.byteswap()
            if NORMALIZE:
                audio_array = audio_array.astype(np.float32) / MAX_AMPLITUDE

            ref_chunk_buffer = audio_array  # store reference chunk

        elif msg.topic == TOPIC_ERROR:
            raw_bytes = msg.payload

            dtype = np.int16 if BIT_DEPTH == 16 else np.int32
            audio_array = np.frombuffer(raw_bytes, dtype=dtype)
            if ENDIANNESS == 'big':
                audio_array = audio_array.byteswap()
            if NORMALIZE:
                audio_array = audio_array.astype(np.float32) / MAX_AMPLITUDE

            error_chunk_buffer = audio_array  # store error chunk

            # Process with AdaptiveANC
            anti_noise_chunk = anc.process_chunk(ref_chunk_buffer, error_chunk_buffer)

            # Convert back to int16 before publishing
            if NORMALIZE:
                anti_noise_chunk = (anti_noise_chunk * MAX_AMPLITUDE).astype(np.int16)

            client.publish(TOPIC_SPEAKER, anti_noise_chunk.tobytes())

    except Exception as e:
        print(f"Error processing chunk: {e}")


def main():
    client = mqtt.Client()
    client.on_message = on_message

    client.connect(MQTT_BROKER, MQTT_PORT, 60)
    client.subscribe([(TOPIC_REF, 0), (TOPIC_ERROR, 0)])
    print(f"Subscribed to {TOPIC_REF} and {TOPIC_ERROR}. Waiting for audio chunks...")

    client.loop_forever()

if __name__ == "__main__":
    main()
