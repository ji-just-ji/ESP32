# -------------------------
# MQTT Configuration
# -------------------------
MQTT_BROKER = "localhost"       # Replace with your MQTT broker address
MQTT_PORT = 1883                  # Standard MQTT port

# MQTT Topics
TOPIC_REF = "esp32/audio_ref"          # Reference mic (noise, other mics)
TOPIC_ERROR = "esp32/audio_error"      # Error mic (signal+noise at speaker, user mic)
TOPIC_SPEAKER = "esp32/audio_processed"  # Output anti-noise

# -------------------------
# ANC Configuration
# -------------------------
SAMPLE_RATE = 16000        # Hz
CHUNK_SIZE = 256           # Number of samples per audio chunk
FILTER_LENGTH = 2048        # Adaptive filter length in samples
MU = 0.0005                 # Step size for LMS adaptive filter
LATENCY_SAMPLES = 0      # Compensates for network + processing delay (~20 ms)

# -------------------------
# Audio Format Standardization
# -------------------------
BIT_DEPTH = 16             # bits per sample
CHANNELS = 1               # Mono audio (1 channel)
ENDIANNESS = 'little'       # Byte order: 'little' or 'big'
NORMALIZE = True            # Whether to normalize audio samples to [-1,1] float32
MAX_AMPLITUDE = 32767       # Maximum amplitude for int16 PCM (used if NORMALIZE=False)

# -------------------------
# Debug / Optional
# -------------------------
ENABLE_DEBUG_PLOTS = False
