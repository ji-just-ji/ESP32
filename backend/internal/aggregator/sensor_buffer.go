package aggregator

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"math"
	"sync"
	"time"

	"iot-backend/internal/models"
)

// ChangeThresholds defines thresholds for detecting significant changes
type ChangeThresholds struct {
	TemperatureDelta float64 // Celsius
	HumidityDelta    float64 // Percentage
	AudioAlwaysTrigger bool  // If true, any audio triggers inference
}

// DeviceState holds the latest sensor readings for a device
type DeviceState struct {
	DeviceID           string
	LastTemperature    *models.TemperatureReading
	LastHumidity       *models.HumidityReading
	LastAudio          *models.AudioRecording
	LastInferenceTime  time.Time
	mu                 sync.RWMutex
}

// SensorAggregator buffers and aggregates sensor data per device
type SensorAggregator struct {
	devices    map[string]*DeviceState
	thresholds ChangeThresholds
	mu         sync.RWMutex

	// Callback for triggering inference
	onInferenceNeeded func(*models.InferenceRequest)
}

// NewSensorAggregator creates a new sensor aggregator
func NewSensorAggregator(thresholds ChangeThresholds) *SensorAggregator {
	return &SensorAggregator{
		devices:    make(map[string]*DeviceState),
		thresholds: thresholds,
	}
}

// SetInferenceCallback sets the callback function for inference requests
func (sa *SensorAggregator) SetInferenceCallback(callback func(*models.InferenceRequest)) {
	sa.onInferenceNeeded = callback
}

// getOrCreateDevice gets or creates a device state
func (sa *SensorAggregator) getOrCreateDevice(deviceID string) *DeviceState {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	if device, exists := sa.devices[deviceID]; exists {
		return device
	}

	device := &DeviceState{
		DeviceID: deviceID,
	}
	sa.devices[deviceID] = device
	return device
}

// UpdateTemperature updates temperature reading and checks for significant changes
func (sa *SensorAggregator) UpdateTemperature(reading *models.TemperatureReading) {
	device := sa.getOrCreateDevice(reading.DeviceID)

	device.mu.Lock()
	previousTemp := device.LastTemperature
	device.LastTemperature = reading
	device.mu.Unlock()

	// Check if temperature change is significant
	if previousTemp != nil {
		delta := math.Abs(reading.Value - previousTemp.Value)
		if delta >= sa.thresholds.TemperatureDelta {
			log.Printf("Significant temperature change detected for %s: %.2f°C (delta: %.2f°C)",
				reading.DeviceID, reading.Value, delta)
			sa.triggerInference(device)
		}
	}
}

// UpdateHumidity updates humidity reading and checks for significant changes
func (sa *SensorAggregator) UpdateHumidity(reading *models.HumidityReading) {
	device := sa.getOrCreateDevice(reading.DeviceID)

	device.mu.Lock()
	previousHumidity := device.LastHumidity
	device.LastHumidity = reading
	device.mu.Unlock()

	// Check if humidity change is significant
	if previousHumidity != nil {
		delta := math.Abs(reading.Value - previousHumidity.Value)
		if delta >= sa.thresholds.HumidityDelta {
			log.Printf("Significant humidity change detected for %s: %.2f%% (delta: %.2f%%)",
				reading.DeviceID, reading.Value, delta)
			sa.triggerInference(device)
		}
	}
}

// UpdateAudio updates audio recording and triggers inference if configured
func (sa *SensorAggregator) UpdateAudio(recording *models.AudioRecording) {
	device := sa.getOrCreateDevice(recording.DeviceID)

	device.mu.Lock()
	device.LastAudio = recording
	device.mu.Unlock()

	// Audio always triggers inference if configured
	if sa.thresholds.AudioAlwaysTrigger {
		log.Printf("Audio received for %s, triggering inference", recording.DeviceID)
		sa.triggerInference(device)
	}
}

// triggerInference creates and sends an inference request
func (sa *SensorAggregator) triggerInference(device *DeviceState) {
	if sa.onInferenceNeeded == nil {
		log.Printf("No inference callback set, skipping inference for %s", device.DeviceID)
		return
	}

	device.mu.RLock()
	defer device.mu.RUnlock()

	// Check if we have all required data
	if device.LastTemperature == nil || device.LastHumidity == nil || device.LastAudio == nil {
		log.Printf("Incomplete sensor data for %s, skipping inference (temp=%v, humidity=%v, audio=%v)",
			device.DeviceID,
			device.LastTemperature != nil,
			device.LastHumidity != nil,
			device.LastAudio != nil)
		return
	}

	// Rate limiting: Don't trigger too frequently (e.g., max once per 5 seconds)
	if time.Since(device.LastInferenceTime) < 5*time.Second {
		log.Printf("Rate limiting inference for %s (last inference was %.1fs ago)",
			device.DeviceID, time.Since(device.LastInferenceTime).Seconds())
		return
	}

	// Create inference request
	request := &models.InferenceRequest{
		DeviceID:    device.DeviceID,
		Timestamp:   time.Now(),
		Temperature: device.LastTemperature.Value,
		Humidity:    device.LastHumidity.Value,
		AudioData:   device.LastAudio.DataBase64,
		AudioMetadata: models.AudioMetadata{
			SampleRate: device.LastAudio.SampleRate,
			Duration:   device.LastAudio.Duration,
		},
	}

	log.Printf("Triggering inference for %s (temp=%.2f°C, humidity=%.2f%%, audio=%.2fs)",
		device.DeviceID, request.Temperature, request.Humidity, request.AudioMetadata.Duration)

	// Update last inference time
	device.LastInferenceTime = time.Now()

	// Call the callback
	sa.onInferenceNeeded(request)
}

// GetDeviceState returns the current state of a device
func (sa *SensorAggregator) GetDeviceState(deviceID string) *DeviceState {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.devices[deviceID]
}

// GetAllDevices returns all device IDs
func (sa *SensorAggregator) GetAllDevices() []string {
	sa.mu.RLock()
	defer sa.mu.RUnlock()

	devices := make([]string, 0, len(sa.devices))
	for deviceID := range sa.devices {
		devices = append(devices, deviceID)
	}
	return devices
}

// ComputeAudioHash computes SHA256 hash of audio data for reference
func ComputeAudioHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
