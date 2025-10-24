package models

import "time"

// Device represents an IoT device in the system
type Device struct {
	DeviceID     string                 `json:"device_id"`
	Name         string                 `json:"name"`
	Location     string                 `json:"location"`
	RegisteredAt time.Time              `json:"registered_at"`
	LastSeen     time.Time              `json:"last_seen"`
	IsActive     bool                   `json:"is_active"`
	Config       map[string]interface{} `json:"config"`
}

// MLPrediction represents ML model prediction metadata for logging
type MLPrediction struct {
	Timestamp        time.Time `json:"timestamp"`
	DeviceID         string    `json:"device_id"`
	Prediction       float64   `json:"prediction"`        // Window position 0-100
	Confidence       float64   `json:"confidence"`        // 0-1
	InferenceTimeMs  float64   `json:"inference_time_ms"` // Inference latency
	ModelVersion     string    `json:"model_version"`
}
