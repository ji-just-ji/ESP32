package mqtt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"iot-backend/internal/models"
)

// Subscriber handles MQTT subscriptions and writes messages to channels
type Subscriber struct {
	client mqtt.Client

	// Output channels (written by subscriber, read by services)
	TempChan          chan *models.TemperatureReading
	HumidityChan      chan *models.HumidityReading
	AudioChan         chan *models.AudioRecording
	WindowControlChan chan *models.InferenceResponse

	// Topic patterns
	temperatureTopic   string
	humidityTopic      string
	audioTopic         string
	windowControlTopic string
}

// SubscriberConfig holds configuration for MQTT subscriber
type SubscriberConfig struct {
	TemperatureTopic   string // e.g., "sensor/+/temperature"
	HumidityTopic      string // e.g., "sensor/+/humidity"
	AudioTopic         string // e.g., "sensor/+/audio"
	WindowControlTopic string // e.g., "window/+/control"
}

// NewSubscriber creates a new MQTT subscriber with channels
func NewSubscriber(
	client mqtt.Client,
	config SubscriberConfig,
	tempChan chan *models.TemperatureReading,
	humidityChan chan *models.HumidityReading,
	audioChan chan *models.AudioRecording,
	windowControlChan chan *models.InferenceResponse,
) *Subscriber {
	return &Subscriber{
		client:             client,
		TempChan:           tempChan,
		HumidityChan:       humidityChan,
		AudioChan:          audioChan,
		WindowControlChan:  windowControlChan,
		temperatureTopic:   config.TemperatureTopic,
		humidityTopic:      config.HumidityTopic,
		audioTopic:         config.AudioTopic,
		windowControlTopic: config.WindowControlTopic,
	}
}

// SubscribeAll subscribes to all configured sensor topics
func (s *Subscriber) SubscribeAll() error {
	// Subscribe to temperature topic
	if s.temperatureTopic != "" {
		if err := s.subscribeToTopic(s.temperatureTopic, s.handleTemperature); err != nil {
			return fmt.Errorf("failed to subscribe to temperature topic: %w", err)
		}
		log.Printf("Subscribed to temperature topic: %s", s.temperatureTopic)
	}

	// Subscribe to humidity topic
	if s.humidityTopic != "" {
		if err := s.subscribeToTopic(s.humidityTopic, s.handleHumidity); err != nil {
			return fmt.Errorf("failed to subscribe to humidity topic: %w", err)
		}
		log.Printf("Subscribed to humidity topic: %s", s.humidityTopic)
	}

	// Subscribe to audio topic
	if s.audioTopic != "" {
		if err := s.subscribeToTopic(s.audioTopic, s.handleAudio); err != nil {
			return fmt.Errorf("failed to subscribe to audio topic: %w", err)
		}
		log.Printf("Subscribed to audio topic: %s", s.audioTopic)
	}

	// Subscribe to window control topic for logging
	if s.windowControlTopic != "" {
		if err := s.subscribeToTopic(s.windowControlTopic, s.handleWindowControl); err != nil {
			return fmt.Errorf("failed to subscribe to window control topic: %w", err)
		}
		log.Printf("Subscribed to window control topic: %s", s.windowControlTopic)
	}

	return nil
}

// subscribeToTopic is a helper function to subscribe to a topic with a handler
func (s *Subscriber) subscribeToTopic(topic string, handler mqtt.MessageHandler) error {
	token := s.client.Subscribe(topic, 1, handler)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

// handleTemperature processes temperature sensor messages and writes to channel
func (s *Subscriber) handleTemperature(client mqtt.Client, msg mqtt.Message) {
	// Parse raw float value from payload
	var value float64
	if _, err := fmt.Sscanf(string(msg.Payload()), "%f", &value); err != nil {
		log.Printf("Error parsing temperature value: %v", err)
		return
	}

	// Extract device ID from topic (sensor/{device_id}/temperature)
	deviceID := extractDeviceID(msg.Topic())
	if deviceID == "" {
		log.Printf("Could not extract device ID from topic: %s", msg.Topic())
		return
	}

	// Generate timestamp server-side
	timestamp := time.Now()

	reading := &models.TemperatureReading{
		Timestamp: timestamp,
		DeviceID:  deviceID,
		Value:     value,
	}

	log.Printf("Received temperature from %s: %.2f°C", deviceID, value)

	// Write to channel (non-blocking with timeout)
	select {
	case s.TempChan <- reading:
		// Successfully sent
	case <-time.After(1 * time.Second):
		log.Printf("Warning: Temperature channel full, dropping message from %s", deviceID)
	}
}

// handleHumidity processes humidity sensor messages and writes to channel
func (s *Subscriber) handleHumidity(client mqtt.Client, msg mqtt.Message) {
	// Parse raw float value from payload
	var value float64
	if _, err := fmt.Sscanf(string(msg.Payload()), "%f", &value); err != nil {
		log.Printf("Error parsing humidity value: %v", err)
		return
	}

	// Extract device ID from topic (sensor/{device_id}/humidity)
	deviceID := extractDeviceID(msg.Topic())
	if deviceID == "" {
		log.Printf("Could not extract device ID from topic: %s", msg.Topic())
		return
	}

	// Generate timestamp server-side
	timestamp := time.Now()

	reading := &models.HumidityReading{
		Timestamp: timestamp,
		DeviceID:  deviceID,
		Value:     value,
	}

	log.Printf("Received humidity from %s: %.2f%%", deviceID, value)

	// Write to channel (non-blocking with timeout)
	select {
	case s.HumidityChan <- reading:
		// Successfully sent
	case <-time.After(1 * time.Second):
		log.Printf("Warning: Humidity channel full, dropping message from %s", deviceID)
	}
}

// handleAudio processes audio sensor messages and writes to channel
func (s *Subscriber) handleAudio(client mqtt.Client, msg mqtt.Message) {
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

	// Generate timestamp server-side
	timestamp := time.Now()

	// payload.Data is already decoded from base64 by json.Unmarshal
	recording := &models.AudioRecording{
		Timestamp:  timestamp,
		DeviceID:   deviceID,
		Data:       payload.Data,
		DataBase64: base64.StdEncoding.EncodeToString(payload.Data),
		SampleRate: payload.SampleRate,
		Duration:   payload.Duration,
		Format:     "wav", // Default format
	}

	log.Printf("Received audio from %s: %.2fs @ %dHz", deviceID, payload.Duration, payload.SampleRate)

	// Write to channel (non-blocking with timeout)
	select {
	case s.AudioChan <- recording:
		// Successfully sent
	case <-time.After(2 * time.Second): // Longer timeout for audio
		log.Printf("Warning: Audio channel full, dropping message from %s", deviceID)
	}
}

// handleWindowControl processes window control responses from ML service and writes to channel
func (s *Subscriber) handleWindowControl(client mqtt.Client, msg mqtt.Message) {
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

	// Write to channel (non-blocking with timeout)
	select {
	case s.WindowControlChan <- &response:
		// Successfully sent
	case <-time.After(1 * time.Second):
		log.Printf("Warning: Window control channel full, dropping message for %s", response.DeviceID)
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
