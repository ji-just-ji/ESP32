package mqtt

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Client manages the MQTT connection (low-level connection management only)
// For subscribing and publishing, use Subscriber and Publisher respectively
type Client struct {
	client mqtt.Client
	config ClientConfig
}

// ClientConfig holds MQTT client configuration
type ClientConfig struct {
	Broker   string
	ClientID string
	Username string
	Password string
}

// NewClient creates a new MQTT client connection
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

	log.Println("MQTT Client: Connected to broker:", config.Broker)

	return &Client{
		client: client,
		config: config,
	}, nil
}

// GetNativeClient returns the underlying paho MQTT client
// This is used by Subscriber and Publisher
func (c *Client) GetNativeClient() mqtt.Client {
	return c.client
}

// IsConnected returns whether the client is currently connected
func (c *Client) IsConnected() bool {
	return c.client.IsConnected()
}

// Close closes the MQTT client connection
func (c *Client) Close() {
	c.client.Disconnect(250)
	log.Println("MQTT Client: Disconnected")
}

// Connection event handlers
var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Printf("MQTT: Received message from topic: %s", msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("MQTT: Connection established")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("MQTT: Connection lost: %v", err)
}
