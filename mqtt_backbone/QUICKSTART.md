# Quick Start Guide

## Step 1: Install Prerequisites

### macOS
```bash
# Install Go
brew install go

# Install Docker (for ClickHouse and Mosquitto)
brew install --cask docker

# Start Docker Desktop
open -a Docker
```

### Linux
```bash
# Install Go
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
```

## Step 2: Install Dependencies

```bash
cd backend

# Download Go dependencies
make deps
```

## Step 3: Start Infrastructure

```bash
# Start ClickHouse and Mosquitto using Docker
make docker-up

# Wait a few seconds for services to initialize
sleep 5
```

## Step 4: Configure Environment

```bash
# Copy environment template
cp .env.example .env

# Edit if needed (default values work with Docker setup)
# nano .env
```

## Step 5: Run the Backend

```bash
# Run the backend service
make run
```

You should see output like:
```
Starting IoT Backend Service...
Loaded model from ./model/regression_model.json with threshold: 5.00
Connected to ClickHouse at localhost:9000
Database schema initialized successfully
Connected to MQTT broker: tcp://localhost:1883
Subscribed to topic: sensor/data
IoT Backend Service is running. Press Ctrl+C to exit.
```

## Step 6: Test the System

Open two new terminal windows:

### Terminal 2 - Monitor Window Actions
```bash
cd backend
python3 scripts/test_subscriber.py
```

### Terminal 3 - Publish Sensor Data
```bash
cd backend
# Install paho-mqtt if needed
pip3 install paho-mqtt

python3 scripts/test_publisher.py
```

You should now see:
- Terminal 1 (backend): Processing logs
- Terminal 2 (subscriber): Window actions being published
- Terminal 3 (publisher): Sensor data being sent

## Step 7: Query Data in ClickHouse

```bash
# Connect to ClickHouse
docker exec -it iot-clickhouse clickhouse-client

# Query sensor readings
SELECT * FROM iot.sensor_readings ORDER BY timestamp DESC LIMIT 10;

# Query window actions
SELECT * FROM iot.window_actions ORDER BY timestamp DESC LIMIT 10;

# Exit
exit;
```

## Stopping Services

```bash
# Stop the backend (Ctrl+C in Terminal 1)

# Stop Docker services
make docker-down
```

## Troubleshooting

### Port Already in Use
If ports 1883 or 9000 are already in use:
```bash
# Check what's using the port
lsof -i :1883
lsof -i :9000

# Kill the process or change ports in docker-compose.yml
```

### Go Not Found
Make sure Go is in your PATH:
```bash
go version
```

### Docker Not Running
```bash
# Check Docker status
docker ps

# Start Docker Desktop (macOS)
open -a Docker
```

### Connection Refused
Wait a bit longer for services to start, then try again:
```bash
# Check if services are healthy
docker-compose ps
```

## Next Steps

1. **Customize the ML Model**: Edit `model/regression_model.json` with your own coefficients
2. **Connect Real Sensors**: Update your ESP32 to publish to the MQTT topic
3. **Add More Features**: Temperature forecasting, anomaly detection, etc.
4. **Deploy to Production**: Use Docker Compose or Kubernetes

## Useful Commands

```bash
make build       # Build the binary
make run         # Run the service
make test        # Run tests
make clean       # Clean build artifacts
make docker-up   # Start Docker services
make docker-down # Stop Docker services
make help        # Show all commands
```
