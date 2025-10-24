import socket
import wave
import numpy as np

UDP_IP = "0.0.0.0"
UDP_PORT = 12345
SAMPLE_RATE = 16000  # same as ESP32
CHANNELS = 1
FORMAT = np.int16

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.bind((UDP_IP, UDP_PORT))

print("Listening for audio packets...")

frames = []

try:
    while True:
        data, addr = sock.recvfrom(1024)
        frames.append(np.frombuffer(data, dtype=FORMAT))
except KeyboardInterrupt:
    pass

audio = np.concatenate(frames)
with wave.open("output.wav", "wb") as wf:
    wf.setnchannels(CHANNELS)
    wf.setsampwidth(2)
    wf.setframerate(SAMPLE_RATE)
    wf.writeframes(audio.tobytes())

print("Saved output.wav")
