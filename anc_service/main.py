import socket
import numpy as np
import sounddevice as sd

UDP_IP = "0.0.0.0"   # Listen on all available network interfaces
UDP_PORT = 12345      # Same port as used in ESP32 code

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.bind((UDP_IP, UDP_PORT))

print(f"Listening for UDP packets on port {UDP_PORT}...")

while True:
    data, addr = sock.recvfrom(1024)  # buffer size in bytes
    print(f"Received {len(data)} bytes from {addr}: {data.decode(errors='ignore')}")
