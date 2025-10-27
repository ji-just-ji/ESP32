package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// MQTT Configuration
	MQTTBroker             string
	MQTTClientID           string
	MQTTUsername           string
	MQTTPassword           string

	// Multi-topic MQTT configuration
	MQTTTopicTemperature   string
	MQTTTopicHumidity      string
	MQTTTopicAudio         string
	MQTTTopicInferenceReq  string
	MQTTTopicWindowControl string

	// Legacy topics (for backward compatibility)
	MQTTTopicSensor        string
	MQTTTopicAction        string

	// ClickHouse Configuration
	ClickHouseAddr         string
	ClickHouseDB           string
	ClickHouseUser         string
	ClickHousePass         string

	// ML Model Configuration
	ModelPath              string

	// Change Detection Thresholds
	TemperatureThreshold   float64
	HumidityThreshold      float64
	AudioAlwaysTrigger     bool
}

func Load() *Config {
	// Load .env file if it exists
	_ = godotenv.Load()

	return &Config{
		// MQTT Configuration
		MQTTBroker:             getEnv("MQTT_BROKER", "tcp://localhost:1883"),
		MQTTClientID:           getEnv("MQTT_CLIENT_ID", "iot-backend"),
		MQTTUsername:           getEnv("MQTT_USERNAME", ""),
		MQTTPassword:           getEnv("MQTT_PASSWORD", ""),

		// Multi-topic MQTT configuration
		MQTTTopicTemperature:   getEnv("MQTT_TOPIC_TEMPERATURE", "sensor/+/temperature"),
		MQTTTopicHumidity:      getEnv("MQTT_TOPIC_HUMIDITY", "sensor/+/humidity"),
		MQTTTopicAudio:         getEnv("MQTT_TOPIC_AUDIO", "sensor/+/audio"),
		MQTTTopicInferenceReq:  getEnv("MQTT_TOPIC_INFERENCE_REQ", "ml/inference/request/{device_id}"),
		MQTTTopicWindowControl: getEnv("MQTT_TOPIC_WINDOW_CONTROL", "window/+/control"),

		// Legacy topics
		MQTTTopicSensor:        getEnv("MQTT_TOPIC_SENSOR", "sensor/data"),
		MQTTTopicAction:        getEnv("MQTT_TOPIC_ACTION", "window/action"),

		// ClickHouse Configuration
		ClickHouseAddr:         getEnv("CLICKHOUSE_ADDR", "localhost:9000"),
		ClickHouseDB:           getEnv("CLICKHOUSE_DB", "iot"),
		ClickHouseUser:         getEnv("CLICKHOUSE_USER", "default"),
		ClickHousePass:         getEnv("CLICKHOUSE_PASS", ""),

		// ML Model Configuration
		ModelPath:              getEnv("MODEL_PATH", "./model/regression_model.json"),

		// Change Detection Thresholds
		TemperatureThreshold:   getEnvFloat("TEMPERATURE_THRESHOLD", 0.5),
		HumidityThreshold:      getEnvFloat("HUMIDITY_THRESHOLD", 2.0),
		AudioAlwaysTrigger:     getEnvBool("AUDIO_ALWAYS_TRIGGER", true),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvFloat(key string, defaultValue float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		log.Printf("Warning: failed to parse %s as float, using default: %v", key, err)
		return defaultValue
	}
	return floatValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("Warning: failed to parse %s as bool, using default: %v", key, err)
		return defaultValue
	}
	return boolValue
}
