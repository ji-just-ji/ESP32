package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"iot-backend/internal/models"
)

// Publisher handles MQTT publishing from channels
type Publisher struct {
	client mqtt.Client

	// Input channel (read by publisher, written by inference service)
	InferenceReqChan chan *models.InferenceRequest

	// Topic pattern
	inferenceReqTopic string // e.g., "ml/inference/request/{device_id}"
}

// PublisherConfig holds configuration for MQTT publisher
type PublisherConfig struct {
	InferenceReqTopic string // e.g., "ml/inference/request/{device_id}"
}

// NewPublisher creates a new MQTT publisher with channels
func NewPublisher(
	client mqtt.Client,
	config PublisherConfig,
	inferenceReqChan chan *models.InferenceRequest,
) *Publisher {
	return &Publisher{
		client:            client,
		InferenceReqChan:  inferenceReqChan,
		inferenceReqTopic: config.InferenceReqTopic,
	}
}

// Start begins publishing inference requests from the channel
// Runs until context is cancelled or channel is closed
func (p *Publisher) Start(ctx context.Context) {
	log.Println("MQTT Publisher: Starting...")

	for {
		select {
		case <-ctx.Done():
			log.Println("MQTT Publisher: Context cancelled, shutting down...")
			return

		case req, ok := <-p.InferenceReqChan:
			if !ok {
				// Channel closed
				log.Println("MQTT Publisher: Inference request channel closed, shutting down...")
				return
			}

			// Publish the inference request
			if err := p.publishInferenceRequest(req); err != nil {
				log.Printf("Error publishing inference request: %v", err)
			}
		}
	}
}

// publishInferenceRequest publishes an inference request to the ML service
func (p *Publisher) publishInferenceRequest(req *models.InferenceRequest) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal inference request: %w", err)
	}

	// Replace {device_id} placeholder with actual device ID
	topic := formatTopic(p.inferenceReqTopic, req.DeviceID)

	token := p.client.Publish(topic, 1, false, payload)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish inference request: %w", token.Error())
	}

	log.Printf("Published inference request for device %s to topic: %s", req.DeviceID, topic)
	return nil
}

// formatTopic replaces {device_id} placeholder with actual device ID
func formatTopic(topicPattern, deviceID string) string {
	return strings.ReplaceAll(topicPattern, "{device_id}", deviceID)
}
