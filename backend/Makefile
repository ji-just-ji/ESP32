.PHONY: build run test clean deps docker-up docker-down help

# Build the application
build:
	@echo "Building IoT Backend..."
	go build -o bin/iot-backend cmd/server/main.go

# Run the application
run:
	@echo "Running IoT Backend..."
	go run cmd/server/main.go

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf model/*.json

# Start Docker services (ClickHouse + Mosquitto)
docker-up:
	@echo "Starting Docker services..."
	docker-compose up -d

# Stop Docker services
docker-down:
	@echo "Stopping Docker services..."
	docker-compose down

# Show help
help:
	@echo "Available targets:"
	@echo "  build       - Build the application"
	@echo "  run         - Run the application"
	@echo "  deps        - Download and tidy dependencies"
	@echo "  test        - Run tests"
	@echo "  clean       - Clean build artifacts"
	@echo "  docker-up   - Start Docker services (ClickHouse + Mosquitto)"
	@echo "  docker-down - Stop Docker services"
	@echo "  help        - Show this help message"
