package models

import "time"

// TemperatureReading represents temperature sensor data
type TemperatureReading struct {
	Timestamp time.Time `json:"timestamp"`
	DeviceID  string    `json:"device_id"`
	Value     float64   `json:"value"` // Celsius
}

// HumidityReading represents humidity sensor data
type HumidityReading struct {
	Timestamp time.Time `json:"timestamp"`
	DeviceID  string    `json:"device_id"`
	Value     float64   `json:"value"` // Percentage 0-100
}

// WindowAction represents the ML model decision for continuous window control
type WindowAction struct {
	Timestamp   time.Time `json:"timestamp"`
	DeviceID    string    `json:"device_id"`
	Position    float64   `json:"position"`     // 0-100% window position
	Confidence  float64   `json:"confidence"`   // 0-1 confidence score
	Temperature float64   `json:"temperature"`  // Input feature
	Humidity    float64   `json:"humidity"`     // Input feature
	SoundVolume float64   `json:"sound_volume"` // Input feature (dB)
}

// InferenceRequest represents the request sent to Python ML service
type InferenceRequest struct {
	DeviceID    string    `json:"device_id"`
	Timestamp   time.Time `json:"timestamp"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	SoundVolume float64   `json:"sound_volume"` // dB level
}

// InferenceResponse represents the response from Python ML service
type InferenceResponse struct {
	DeviceID     string                 `json:"device_id"`
	Timestamp    time.Time              `json:"timestamp"`
	Position     float64                `json:"position"`    // 0-100%
	Confidence   float64                `json:"confidence"`  // 0-1
	FeaturesUsed map[string]interface{} `json:"features_used"`
}
