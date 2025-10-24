package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"iot-backend/internal/aggregator"
	"iot-backend/internal/database"
	"iot-backend/internal/models"
	"iot-backend/internal/mqtt"
	"iot-backend/pkg/config"
)

func main() {
	log.Println("Starting IoT Backend Service v2.0...")

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

	// Initialize sensor aggregator with change detection thresholds
	thresholds := aggregator.ChangeThresholds{
		TemperatureDelta:   cfg.TemperatureThreshold,
		HumidityDelta:      cfg.HumidityThreshold,
		AudioAlwaysTrigger: cfg.AudioAlwaysTrigger,
	}
	sensorAggregator := aggregator.NewSensorAggregator(thresholds)

	// Initialize MQTT client with multi-topic configuration
	mqttConfig := mqtt.ClientConfig{
		Broker:             cfg.MQTTBroker,
		ClientID:           cfg.MQTTClientID,
		Username:           cfg.MQTTUsername,
		Password:           cfg.MQTTPassword,
		TemperatureTopic:   cfg.MQTTTopicTemperature,
		HumidityTopic:      cfg.MQTTTopicHumidity,
		AudioTopic:         cfg.MQTTTopicAudio,
		InferenceReqTopic:  cfg.MQTTTopicInferenceReq,
		WindowControlTopic: cfg.MQTTTopicWindowControl,
	}

	mqttClient, err := mqtt.NewClient(mqttConfig)
	if err != nil {
		log.Fatalf("Failed to initialize MQTT client: %v", err)
	}
	defer mqttClient.Close()

	// Set up inference callback - publishes inference requests to ML service
	sensorAggregator.SetInferenceCallback(func(req *models.InferenceRequest) {
		if err := mqttClient.PublishInferenceRequest(req); err != nil {
			log.Printf("Error publishing inference request: %v", err)
		}
	})

	// Set up MQTT message handlers
	handlers := mqtt.MessageHandlers{
		OnTemperature: func(reading *models.TemperatureReading) {
			handleTemperature(reading, db, sensorAggregator)
		},
		OnHumidity: func(reading *models.HumidityReading) {
			handleHumidity(reading, db, sensorAggregator)
		},
		OnAudio: func(recording *models.AudioRecording) {
			handleAudio(recording, db, sensorAggregator)
		},
		OnWindowControl: func(response *models.InferenceResponse) {
			handleWindowControl(response, db)
		},
	}

	mqttClient.SetHandlers(handlers)

	// Subscribe to all topics
	if err := mqttClient.SubscribeAll(); err != nil {
		log.Fatalf("Failed to subscribe to MQTT topics: %v", err)
	}

	log.Println("IoT Backend Service v2.0 is running. Press Ctrl+C to exit.")
	log.Printf("Change detection thresholds: Temp=%.2f°C, Humidity=%.2f%%, AudioTrigger=%v",
		cfg.TemperatureThreshold, cfg.HumidityThreshold, cfg.AudioAlwaysTrigger)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")
}

// handleTemperature processes temperature sensor data
func handleTemperature(reading *models.TemperatureReading, db *database.ClickHouseDB, agg *aggregator.SensorAggregator) {
	// Save to database
	if err := db.SaveTemperature(reading); err != nil {
		log.Printf("Error saving temperature: %v", err)
		return
	}

	// Update aggregator (triggers inference if threshold exceeded)
	agg.UpdateTemperature(reading)

	// Auto-register device on first seen
	registerDevice(reading.DeviceID, db)
}

// handleHumidity processes humidity sensor data
func handleHumidity(reading *models.HumidityReading, db *database.ClickHouseDB, agg *aggregator.SensorAggregator) {
	// Save to database
	if err := db.SaveHumidity(reading); err != nil {
		log.Printf("Error saving humidity: %v", err)
		return
	}

	// Update aggregator (triggers inference if threshold exceeded)
	agg.UpdateHumidity(reading)

	// Auto-register device on first seen
	registerDevice(reading.DeviceID, db)
}

// handleAudio processes audio sensor data
func handleAudio(recording *models.AudioRecording, db *database.ClickHouseDB, agg *aggregator.SensorAggregator) {
	// Compute audio hash for reference
	audioHash := aggregator.ComputeAudioHash(recording.Data)

	// Save audio metadata to database (not the raw data)
	if err := db.SaveAudio(recording, audioHash); err != nil {
		log.Printf("Error saving audio metadata: %v", err)
		return
	}

	// Update aggregator (triggers inference if configured)
	agg.UpdateAudio(recording)

	// Auto-register device on first seen
	registerDevice(recording.DeviceID, db)
}

// handleWindowControl logs window control responses from ML service
func handleWindowControl(response *models.InferenceResponse, db *database.ClickHouseDB) {
	log.Printf("Window control received: Device=%s, Position=%.2f%%, Confidence=%.2f",
		response.DeviceID, response.Position, response.Confidence)

	// Create window action record
	windowAction := &models.WindowAction{
		Timestamp:   response.Timestamp,
		DeviceID:    response.DeviceID,
		Position:    response.Position,
		Confidence:  response.Confidence,
		Temperature: 0.0, // These would be populated from features_used if available
		Humidity:    0.0,
		AudioHash:   "",
	}

	// Extract features from response if available
	if temp, ok := response.FeaturesUsed["temperature"].(float64); ok {
		windowAction.Temperature = temp
	}
	if humidity, ok := response.FeaturesUsed["humidity"].(float64); ok {
		windowAction.Humidity = humidity
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

// registerDevice auto-registers a device on first message
func registerDevice(deviceID string, db *database.ClickHouseDB) {
	device := &models.Device{
		DeviceID:     deviceID,
		Name:         deviceID,
		Location:     "Unknown",
		RegisteredAt: time.Now(),
		LastSeen:     time.Now(),
		IsActive:     true,
		Config:       make(map[string]interface{}),
	}

	// Best effort - don't fail if registration fails
	if err := db.UpsertDevice(device); err != nil {
		log.Printf("Error registering device %s: %v", deviceID, err)
	}
}
