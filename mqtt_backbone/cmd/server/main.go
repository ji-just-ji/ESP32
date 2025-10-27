package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"iot-backend/internal/database"
	"iot-backend/internal/models"
	"iot-backend/internal/mqtt"
	"iot-backend/internal/services"
	"iot-backend/pkg/config"
)

func main() {
	log.Println("Starting IoT Backend Service v1.5 (Channel-Based Architecture)...")

	// Load configuration
	cfg := config.Load()

	// Initialize ClickHouse database
	db, err := database.NewClickHouseDB(
		cfg.ClickHouseAddr,
		cfg.ClickHouseDB,
		cfg.ClickHouseUser,
		cfg.ClickHousePass,
	)
	if err != nil {
		log.Fatalf("Failed to initialize ClickHouse: %v", err)
	}
	defer db.Close()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// === Channel Creation ===
	// These channels connect MQTT layer with services layer
	log.Println("Creating communication channels...")

	// Sensor data channels (MQTT → Services)
	tempChan := make(chan *models.TemperatureReading, 100)
	humidityChan := make(chan *models.HumidityReading, 100)
	audioChan := make(chan *models.AudioRecording, 50)
	windowControlChan := make(chan *models.InferenceResponse, 50)

	// Inference request channel (Services → MQTT)
	inferenceReqChan := make(chan *models.InferenceRequest, 50)

	// === Initialize MQTT Client ===
	log.Println("Connecting to MQTT broker...")
	mqttConfig := mqtt.ClientConfig{
		Broker:   cfg.MQTTBroker,
		ClientID: cfg.MQTTClientID,
		Username: cfg.MQTTUsername,
		Password: cfg.MQTTPassword,
	}

	mqttClient, err := mqtt.NewClient(mqttConfig)
	if err != nil {
		log.Fatalf("Failed to initialize MQTT client: %v", err)
	}
	defer mqttClient.Close()

	// === Initialize MQTT Subscriber ===
	log.Println("Setting up MQTT subscriber...")
	subscriberConfig := mqtt.SubscriberConfig{
		TemperatureTopic:   cfg.MQTTTopicTemperature,
		HumidityTopic:      cfg.MQTTTopicHumidity,
		AudioTopic:         cfg.MQTTTopicAudio,
		WindowControlTopic: cfg.MQTTTopicWindowControl,
	}

	subscriber := mqtt.NewSubscriber(
		mqttClient.GetNativeClient(),
		subscriberConfig,
		tempChan,
		humidityChan,
		audioChan,
		windowControlChan,
	)

	// Subscribe to all topics
	if err := subscriber.SubscribeAll(); err != nil {
		log.Fatalf("Failed to subscribe to MQTT topics: %v", err)
	}

	// === Initialize MQTT Publisher ===
	log.Println("Setting up MQTT publisher...")
	publisherConfig := mqtt.PublisherConfig{
		InferenceReqTopic: cfg.MQTTTopicInferenceReq,
	}

	publisher := mqtt.NewPublisher(
		mqttClient.GetNativeClient(),
		publisherConfig,
		inferenceReqChan,
	)

	// Start publisher goroutine
	go publisher.Start(ctx)

	// === Initialize Inference Service ===
	log.Println("Initializing inference service...")
	inferenceConfig := services.InferenceServiceConfig{
		TemperatureThreshold: cfg.TemperatureThreshold,
		HumidityThreshold:    cfg.HumidityThreshold,
		RateLimitDuration:    5 * time.Second,
		ChannelSize:          50,
	}

	inferenceService := services.NewInferenceService(inferenceConfig)

	// Connect inference service output to publisher input
	// (They share the same channel)
	inferenceService.InferenceReqChan = inferenceReqChan

	// Start inference service
	go inferenceService.Start(ctx)

	// === Initialize Sensor Service ===
	log.Println("Initializing sensor service...")
	sensorConfig := services.DefaultSensorServiceConfig()

	sensorService := services.NewSensorService(db, inferenceService, sensorConfig)

	// Connect sensor service inputs to subscriber outputs
	sensorService.TempChan = tempChan
	sensorService.HumidityChan = humidityChan
	sensorService.AudioChan = audioChan

	// Start sensor service
	go sensorService.Start(ctx)

	// === Initialize Window Control Service ===
	// This service handles window control responses from ML service
	go handleWindowControlLoop(ctx, db, windowControlChan)

	// === Log startup info ===
	log.Println("=== IoT Backend Service v1.5 is running ===")
	log.Printf("Architecture: Channel-based with separated layers")
	log.Printf("Change detection thresholds: Temp=%.2f°C, Humidity=%.2f%%",
		cfg.TemperatureThreshold, cfg.HumidityThreshold)
	log.Printf("Inference trigger: Volume always triggers, temp/humidity on threshold")
	log.Printf("MQTT Topics:")
	log.Printf("  - Temperature:    %s", cfg.MQTTTopicTemperature)
	log.Printf("  - Humidity:       %s", cfg.MQTTTopicHumidity)
	log.Printf("  - Audio:          %s", cfg.MQTTTopicAudio)
	log.Printf("  - Inference Req:  %s", cfg.MQTTTopicInferenceReq)
	log.Printf("  - Window Control: %s", cfg.MQTTTopicWindowControl)
	log.Println("Press Ctrl+C to exit...")

	// === Wait for interrupt signal ===
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// === Graceful shutdown ===
	log.Println("Shutdown signal received, stopping services...")
	cancel() // Cancel context to stop all goroutines

	// Give services time to finish processing
	time.Sleep(2 * time.Second)

	log.Println("Shutdown complete. Goodbye!")
}

// handleWindowControlLoop processes window control responses from ML service
func handleWindowControlLoop(ctx context.Context, db *database.ClickHouseDB, windowControlChan chan *models.InferenceResponse) {
	log.Println("WindowControlService: Starting...")

	for {
		select {
		case <-ctx.Done():
			log.Println("WindowControlService: Shutting down...")
			return

		case response, ok := <-windowControlChan:
			if !ok {
				log.Println("WindowControlService: Channel closed, shutting down...")
				return
			}

			handleWindowControl(response, db)
		}
	}
}

// handleWindowControl logs and saves window control responses from ML service
func handleWindowControl(response *models.InferenceResponse, db *database.ClickHouseDB) {
	log.Printf("Window control received: Device=%s, Position=%.2f%%, Confidence=%.2f",
		response.DeviceID, response.Position, response.Confidence)

	// Create window action record
	windowAction := &models.WindowAction{
		Timestamp:   response.Timestamp,
		DeviceID:    response.DeviceID,
		Position:    response.Position,
		Confidence:  response.Confidence,
		Temperature: 0.0,
		Humidity:    0.0,
		SoundVolume: 0.0,
	}

	// Extract features from response if available
	if temp, ok := response.FeaturesUsed["temperature"].(float64); ok {
		windowAction.Temperature = temp
	}
	if humidity, ok := response.FeaturesUsed["humidity"].(float64); ok {
		windowAction.Humidity = humidity
	}
	if volume, ok := response.FeaturesUsed["sound_volume"].(float64); ok {
		windowAction.SoundVolume = volume
	}

	// Save window action to database
	if err := db.SaveWindowAction(windowAction); err != nil {
		log.Printf("Error saving window action: %v", err)
		return
	}

	// Save ML prediction metadata
	mlPrediction := &models.MLPrediction{
		Timestamp:    response.Timestamp,
		DeviceID:     response.DeviceID,
		Prediction:   response.Position,
		Confidence:   response.Confidence,
		ModelVersion: "v1.0.0", // Could be extracted from response if available
	}

	if err := db.SaveMLPrediction(mlPrediction); err != nil {
		log.Printf("Error saving ML prediction: %v", err)
	}
}
