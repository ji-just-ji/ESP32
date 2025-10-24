package mqtt

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"iot-backend/internal/models"
)

// handleTemperature processes temperature sensor messages
func (c *Client) handleTemperature(client mqtt.Client, msg mqtt.Message) {
	var payload struct {
		Value     float64 `json:"value"`
		Timestamp string  `json:"timestamp"`
	}

	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		log.Printf("Error unmarshaling temperature data: %v", err)
		return
	}

	// Extract device ID from topic (sensor/{device_id}/temperature)
	deviceID := extractDeviceID(msg.Topic())
	if deviceID == "" {
		log.Printf("Could not extract device ID from topic: %s", msg.Topic())
		return
	}

	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339, payload.Timestamp)
	if err != nil {
		timestamp = time.Now()
	}

	reading := &models.TemperatureReading{
		Timestamp: timestamp,
		DeviceID:  deviceID,
		Value:     payload.Value,
	}

	log.Printf("Received temperature from %s: %.2f°C", deviceID, payload.Value)

	if c.handlers.OnTemperature != nil {
		c.handlers.OnTemperature(reading)
	}
}

// handleHumidity processes humidity sensor messages
func (c *Client) handleHumidity(client mqtt.Client, msg mqtt.Message) {
	var payload struct {
		Value     float64 `json:"value"`
		Timestamp string  `json:"timestamp"`
	}

	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		log.Printf("Error unmarshaling humidity data: %v", err)
		return
	}

	// Extract device ID from topic (sensor/{device_id}/humidity)
	deviceID := extractDeviceID(msg.Topic())
	if deviceID == "" {
		log.Printf("Could not extract device ID from topic: %s", msg.Topic())
		return
	}

	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339, payload.Timestamp)
	if err != nil {
		timestamp = time.Now()
	}

	reading := &models.HumidityReading{
		Timestamp: timestamp,
		DeviceID:  deviceID,
		Value:     payload.Value,
	}

	log.Printf("Received humidity from %s: %.2f%%", deviceID, payload.Value)

	if c.handlers.OnHumidity != nil {
		c.handlers.OnHumidity(reading)
	}
}

// handleAudio processes audio sensor messages
func (c *Client) handleAudio(client mqtt.Client, msg mqtt.Message) {
	var payload models.AudioPayload

	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		log.Printf("Error unmarshaling audio data: %v", err)
		return
	}

	// Extract device ID from topic (sensor/{device_id}/audio)
	deviceID := extractDeviceID(msg.Topic())
	if deviceID == "" {
		log.Printf("Could not extract device ID from topic: %s", msg.Topic())
		return
	}

	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339, payload.Timestamp)
	if err != nil {
		timestamp = time.Now()
	}

	// Decode base64 audio data
	audioData, err := base64.StdEncoding.DecodeString(payload.Data)
	if err != nil {
		log.Printf("Error decoding audio data: %v", err)
		return
	}

	recording := &models.AudioRecording{
		Timestamp:  timestamp,
		DeviceID:   deviceID,
		Data:       audioData,
		DataBase64: payload.Data,
		SampleRate: payload.SampleRate,
		Duration:   payload.Duration,
		Format:     "wav", // Default format
	}

	log.Printf("Received audio from %s: %.2fs @ %dHz", deviceID, payload.Duration, payload.SampleRate)

	if c.handlers.OnAudio != nil {
		c.handlers.OnAudio(recording)
	}
}

// handleWindowControl processes window control responses from ML service
func (c *Client) handleWindowControl(client mqtt.Client, msg mqtt.Message) {
	var response models.InferenceResponse

	if err := json.Unmarshal(msg.Payload(), &response); err != nil {
		log.Printf("Error unmarshaling window control response: %v", err)
		return
	}

	// Extract device ID from topic if not in payload
	if response.DeviceID == "" {
		response.DeviceID = extractDeviceID(msg.Topic())
	}

	log.Printf("Received window control for %s: position=%.2f%%, confidence=%.2f",
		response.DeviceID, response.Position, response.Confidence)

	if c.handlers.OnWindowControl != nil {
		c.handlers.OnWindowControl(&response)
	}
}

// extractDeviceID extracts device ID from MQTT topic
// Example: "sensor/sensor-001/temperature" -> "sensor-001"
// Example: "window/sensor-001/control" -> "sensor-001"
func extractDeviceID(topic string) string {
	parts := strings.Split(topic, "/")
	if len(parts) >= 2 {
		// For topics like sensor/{device_id}/temperature or window/{device_id}/control
		// The device_id is the second part
		return parts[1]
	}
	return ""
}
