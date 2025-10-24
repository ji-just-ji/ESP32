package models

import "time"

// AudioRecording represents audio sensor data
type AudioRecording struct {
	Timestamp  time.Time `json:"timestamp"`
	DeviceID   string    `json:"device_id"`
	Data       []byte    `json:"-"`           // Raw audio bytes (not serialized in JSON)
	DataBase64 string    `json:"data"`        // Base64 encoded for MQTT transmission
	SampleRate int       `json:"sample_rate"` // e.g., 16000 Hz
	Duration   float64   `json:"duration"`    // seconds
	Format     string    `json:"format"`      // "wav", "pcm"
}

// AudioPayload represents the incoming audio MQTT message structure
type AudioPayload struct {
	Data       string  `json:"data"`        // Base64 encoded WAV/PCM
	SampleRate int     `json:"sample_rate"`
	Duration   float64 `json:"duration"`
	Timestamp  string  `json:"timestamp"`
}
