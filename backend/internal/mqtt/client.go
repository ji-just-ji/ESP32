package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"iot-backend/internal/models"
)

// MessageHandlers contains callback functions for different message types
type MessageHandlers struct {
	OnTemperature      func(*models.TemperatureReading)
	OnHumidity         func(*models.HumidityReading)
	OnAudio            func(*models.AudioRecording)
	OnInferenceRequest func(*models.InferenceRequest)
	OnWindowControl    func(*models.InferenceResponse)
}

type Client struct {
	client   mqtt.Client
	handlers MessageHandlers

	// Topic patterns
	temperatureTopic    string
	humidityTopic       string
	audioTopic          string
	inferenceReqTopic   string
	windowControlTopic  string
}

// ClientConfig holds MQTT client configuration
type ClientConfig struct {
	Broker               string
	ClientID             string
	Username             string
	Password             string
	TemperatureTopic     string // e.g., "sensor/+/temperature"
	HumidityTopic        string // e.g., "sensor/+/humidity"
	AudioTopic           string // e.g., "sensor/+/audio"
	InferenceReqTopic    string // e.g., "ml/inference/request/{device_id}"
	WindowControlTopic   string // e.g., "window/+/control"
}

// NewClient creates a new MQTT client with multi-topic support
func NewClient(config ClientConfig) (*Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.Broker)
	opts.SetClientID(config.ClientID)
	opts.SetUsername(config.Username)
	opts.SetPassword(config.Password)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.SetOnConnectHandler(connectHandler)
	opts.SetConnectionLostHandler(connectLostHandler)
	opts.SetAutoReconnect(true)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)

	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	log.Println("Connected to MQTT broker:", config.Broker)

	return &Client{
		client:              client,
		temperatureTopic:    config.TemperatureTopic,
		humidityTopic:       config.HumidityTopic,
		audioTopic:          config.AudioTopic,
		inferenceReqTopic:   config.InferenceReqTopic,
		windowControlTopic:  config.WindowControlTopic,
	}, nil
}

// SetHandlers sets the message handlers for different topics
func (c *Client) SetHandlers(handlers MessageHandlers) {
	c.handlers = handlers
}

// SubscribeAll subscribes to all configured sensor topics
func (c *Client) SubscribeAll() error {
	// Subscribe to temperature topic
	if c.temperatureTopic != "" {
		if err := c.subscribeToTopic(c.temperatureTopic, c.handleTemperature); err != nil {
			return fmt.Errorf("failed to subscribe to temperature topic: %w", err)
		}
		log.Printf("Subscribed to temperature topic: %s", c.temperatureTopic)
	}

	// Subscribe to humidity topic
	if c.humidityTopic != "" {
		if err := c.subscribeToTopic(c.humidityTopic, c.handleHumidity); err != nil {
			return fmt.Errorf("failed to subscribe to humidity topic: %w", err)
		}
		log.Printf("Subscribed to humidity topic: %s", c.humidityTopic)
	}

	// Subscribe to audio topic
	if c.audioTopic != "" {
		if err := c.subscribeToTopic(c.audioTopic, c.handleAudio); err != nil {
			return fmt.Errorf("failed to subscribe to audio topic: %w", err)
		}
		log.Printf("Subscribed to audio topic: %s", c.audioTopic)
	}

	// Subscribe to window control topic for logging
	if c.windowControlTopic != "" {
		if err := c.subscribeToTopic(c.windowControlTopic, c.handleWindowControl); err != nil {
			return fmt.Errorf("failed to subscribe to window control topic: %w", err)
		}
		log.Printf("Subscribed to window control topic: %s", c.windowControlTopic)
	}

	return nil
}

// subscribeToTopic is a helper function to subscribe to a topic with a handler
func (c *Client) subscribeToTopic(topic string, handler mqtt.MessageHandler) error {
	token := c.client.Subscribe(topic, 1, handler)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

// PublishInferenceRequest publishes an inference request to the ML service
func (c *Client) PublishInferenceRequest(req *models.InferenceRequest) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal inference request: %w", err)
	}

	// Replace {device_id} placeholder with actual device ID
	topic := formatTopic(c.inferenceReqTopic, req.DeviceID)

	token := c.client.Publish(topic, 1, false, payload)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish inference request: %w", token.Error())
	}

	log.Printf("Published inference request for device %s to topic: %s", req.DeviceID, topic)
	return nil
}

// Close closes the MQTT client connection
func (c *Client) Close() {
	c.client.Disconnect(250)
	log.Println("MQTT client disconnected")
}

// Helper function to format topic with device ID
func formatTopic(topicPattern, deviceID string) string {
	// Simple string replacement for {device_id} placeholder
	topic := topicPattern
	if deviceID != "" {
		// Replace {device_id} with actual device ID
		for i := 0; i < len(topic); i++ {
			if topic[i:] == "{device_id}" {
				topic = topic[:i] + deviceID + topic[i+11:]
				break
			}
			if i+11 < len(topic) && topic[i:i+11] == "{device_id}" {
				topic = topic[:i] + deviceID + topic[i+11:]
				break
			}
		}
	}
	return topic
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message from topic: %s\n", msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("MQTT client connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("MQTT connection lost: %v", err)
}
