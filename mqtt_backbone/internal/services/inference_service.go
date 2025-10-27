package services

import (
	"context"
	"log"
	"math"
	"sync"
	"time"

	"iot-backend/internal/models"
)

// DeviceInferenceState tracks the inference state for a single device
type DeviceInferenceState struct {
	DeviceID                  string
	LastTemperature           *models.TemperatureReading
	LastHumidity              *models.HumidityReading
	LastSoundVolume           *float64
	LastSoundVolumeTimestamp  time.Time
	HasCompletedFirstInference bool
	LastInferenceTime         time.Time
	mu                        sync.RWMutex
}

// IsReadyForInference checks if device has all required sensors for inference
func (state *DeviceInferenceState) IsReadyForInference() bool {
	state.mu.RLock()
	defer state.mu.RUnlock()

	return state.LastTemperature != nil &&
		state.LastHumidity != nil &&
		state.LastSoundVolume != nil
}

// InferenceService manages ML inference triggering based on sensor data
type InferenceService struct {
	devices map[string]*DeviceInferenceState
	mu      sync.RWMutex

	// Thresholds for triggering inference
	temperatureThreshold float64 // °C
	humidityThreshold    float64 // %
	rateLimitDuration    time.Duration

	// Output channel for inference requests
	InferenceReqChan chan *models.InferenceRequest
}

// InferenceServiceConfig holds configuration for inference service
type InferenceServiceConfig struct {
	TemperatureThreshold float64       // °C change to trigger inference
	HumidityThreshold    float64       // % change to trigger inference
	RateLimitDuration    time.Duration // Minimum time between inferences per device
	ChannelSize          int           // Size of inference request channel
}

// DefaultInferenceServiceConfig returns default configuration
func DefaultInferenceServiceConfig() InferenceServiceConfig {
	return InferenceServiceConfig{
		TemperatureThreshold: 0.5,             // 0.5°C
		HumidityThreshold:    2.0,             // 2%
		RateLimitDuration:    5 * time.Second, // Max once per 5 seconds
		ChannelSize:          50,
	}
}

// NewInferenceService creates a new inference service
func NewInferenceService(config InferenceServiceConfig) *InferenceService {
	return &InferenceService{
		devices:              make(map[string]*DeviceInferenceState),
		temperatureThreshold: config.TemperatureThreshold,
		humidityThreshold:    config.HumidityThreshold,
		rateLimitDuration:    config.RateLimitDuration,
		InferenceReqChan:     make(chan *models.InferenceRequest, config.ChannelSize),
	}
}

// Start begins the inference service (currently passive, but can add background tasks)
func (is *InferenceService) Start(ctx context.Context) {
	log.Println("InferenceService: Starting...")

	// This service is mostly passive (responds to updates)
	// but we keep the Start method for consistency and future background tasks

	<-ctx.Done()
	log.Println("InferenceService: Shutting down...")

	// Close the output channel
	close(is.InferenceReqChan)

	log.Println("InferenceService: Shutdown complete")
}

// getOrCreateDevice gets or creates a device inference state
func (is *InferenceService) getOrCreateDevice(deviceID string) *DeviceInferenceState {
	is.mu.Lock()
	defer is.mu.Unlock()

	if state, exists := is.devices[deviceID]; exists {
		return state
	}

	state := &DeviceInferenceState{
		DeviceID: deviceID,
	}
	is.devices[deviceID] = state
	return state
}

// UpdateTemperature updates temperature reading and checks for inference trigger
func (is *InferenceService) UpdateTemperature(reading *models.TemperatureReading) {
	state := is.getOrCreateDevice(reading.DeviceID)

	state.mu.Lock()
	previousTemp := state.LastTemperature
	state.LastTemperature = reading
	state.mu.Unlock()

	// Check if temperature change is significant
	shouldTrigger := false
	if state.HasCompletedFirstInference && previousTemp != nil {
		delta := math.Abs(reading.Value - previousTemp.Value)
		if delta >= is.temperatureThreshold {
			log.Printf("Significant temperature change for %s: %.2f°C (delta: %.2f°C)",
				reading.DeviceID, reading.Value, delta)
			shouldTrigger = true
		}
	}

	if shouldTrigger {
		is.triggerInference(state)
	}
}

// UpdateHumidity updates humidity reading and checks for inference trigger
func (is *InferenceService) UpdateHumidity(reading *models.HumidityReading) {
	state := is.getOrCreateDevice(reading.DeviceID)

	state.mu.Lock()
	previousHumidity := state.LastHumidity
	state.LastHumidity = reading
	state.mu.Unlock()

	// Check if humidity change is significant
	shouldTrigger := false
	if state.HasCompletedFirstInference && previousHumidity != nil {
		delta := math.Abs(reading.Value - previousHumidity.Value)
		if delta >= is.humidityThreshold {
			log.Printf("Significant humidity change for %s: %.2f%% (delta: %.2f%%)",
				reading.DeviceID, reading.Value, delta)
			shouldTrigger = true
		}
	}

	if shouldTrigger {
		is.triggerInference(state)
	}
}

// UpdateVolume updates sound volume and always triggers inference
func (is *InferenceService) UpdateVolume(deviceID string, volume float64, timestamp time.Time) {
	state := is.getOrCreateDevice(deviceID)

	state.mu.Lock()
	state.LastSoundVolume = &volume
	state.LastSoundVolumeTimestamp = timestamp
	state.mu.Unlock()

	log.Printf("Sound volume updated for %s: %.2f dB (always triggers)", deviceID, volume)

	// Volume always triggers inference
	is.triggerInference(state)
}

// triggerInference creates and sends an inference request if conditions are met
func (is *InferenceService) triggerInference(state *DeviceInferenceState) {
	state.mu.Lock()
	defer state.mu.Unlock()

	// Check if we have all required sensors
	if !state.HasCompletedFirstInference {
		// First inference: require all 3 sensors
		if state.LastTemperature == nil || state.LastHumidity == nil || state.LastSoundVolume == nil {
			log.Printf("Incomplete sensor data for %s (first inference), skipping (temp=%v, humidity=%v, volume=%v)",
				state.DeviceID,
				state.LastTemperature != nil,
				state.LastHumidity != nil,
				state.LastSoundVolume != nil)
			return
		}
	} else {
		// Subsequent inferences: should always have data, but check anyway
		if state.LastTemperature == nil || state.LastHumidity == nil || state.LastSoundVolume == nil {
			log.Printf("Warning: Missing sensor data for %s after first inference, skipping",
				state.DeviceID)
			return
		}
	}

	// Rate limiting: Don't trigger too frequently
	if time.Since(state.LastInferenceTime) < is.rateLimitDuration {
		log.Printf("Rate limiting inference for %s (last inference was %.1fs ago)",
			state.DeviceID, time.Since(state.LastInferenceTime).Seconds())
		return
	}

	// Create inference request
	request := &models.InferenceRequest{
		DeviceID:    state.DeviceID,
		Timestamp:   time.Now(),
		Temperature: state.LastTemperature.Value,
		Humidity:    state.LastHumidity.Value,
		SoundVolume: *state.LastSoundVolume,
	}

	log.Printf("Triggering inference for %s (temp=%.2f°C, humidity=%.2f%%, volume=%.2f dB)",
		state.DeviceID, request.Temperature, request.Humidity, request.SoundVolume)

	// Update last inference time
	state.LastInferenceTime = time.Now()

	// Mark first inference as completed
	if !state.HasCompletedFirstInference {
		state.HasCompletedFirstInference = true
		log.Printf("Completed first inference for %s", state.DeviceID)
	}

	// Send request to channel (non-blocking with timeout)
	select {
	case is.InferenceReqChan <- request:
		log.Printf("Inference request sent for %s", state.DeviceID)
	case <-time.After(1 * time.Second):
		log.Printf("Warning: Inference request channel full, dropping request for %s", state.DeviceID)
	}
}

// GetDeviceState returns the current inference state for a device (for debugging/monitoring)
func (is *InferenceService) GetDeviceState(deviceID string) *DeviceInferenceState {
	is.mu.RLock()
	defer is.mu.RUnlock()
	return is.devices[deviceID]
}

// GetAllDevices returns all device IDs being tracked
func (is *InferenceService) GetAllDevices() []string {
	is.mu.RLock()
	defer is.mu.RUnlock()

	devices := make([]string, 0, len(is.devices))
	for deviceID := range is.devices {
		devices = append(devices, deviceID)
	}
	return devices
}
