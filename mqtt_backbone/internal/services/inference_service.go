package services

import (
	"context"
	"log"
	"math"
	"sync"
	"time"

	"iot-backend/internal/database"
	"iot-backend/internal/models"
)

// InferenceService manages ML inference triggering using CQRS pattern
// Instead of event-driven triggering, it polls ClickHouse periodically
// and uses statistical analysis (Z-scores) to determine when to trigger inference
type InferenceService struct {
	db *database.ClickHouseDB

	// Configuration
	pollingInterval time.Duration
	dataWindow      time.Duration
	baselineDays    int
	zScoreThreshold float64

	// Output channel for inference requests
	InferenceReqChan chan *models.InferenceRequest

	// Internal state
	mu             sync.RWMutex
	trackedDevices map[string]bool // Devices we've seen
}

// InferenceServiceConfig holds configuration for inference service
type InferenceServiceConfig struct {
	PollingIntervalSeconds int     // How often to check for changes
	DataWindowSeconds      int     // Time window for querying current data
	HistoricalBaselineDays int     // Days of historical data for std dev
	ZScoreThreshold        float64 // Threshold for triggering
	ChannelSize            int     // Size of inference request channel
}

// DefaultInferenceServiceConfig returns default configuration
func DefaultInferenceServiceConfig() InferenceServiceConfig {
	return InferenceServiceConfig{
		PollingIntervalSeconds: 60,
		DataWindowSeconds:      120,
		HistoricalBaselineDays: 7,
		ZScoreThreshold:        1.5,
		ChannelSize:            50,
	}
}

// NewInferenceService creates a new CQRS-based inference service
func NewInferenceService(db *database.ClickHouseDB, config InferenceServiceConfig) *InferenceService {
	return &InferenceService{
		db:               db,
		pollingInterval:  time.Duration(config.PollingIntervalSeconds) * time.Second,
		dataWindow:       time.Duration(config.DataWindowSeconds) * time.Second,
		baselineDays:     config.HistoricalBaselineDays,
		zScoreThreshold:  config.ZScoreThreshold,
		InferenceReqChan: make(chan *models.InferenceRequest, config.ChannelSize),
		trackedDevices:   make(map[string]bool),
	}
}

// Start begins the polling loop
func (is *InferenceService) Start(ctx context.Context) {
	log.Println("InferenceService: Starting CQRS polling loop...")
	log.Printf("InferenceService: Polling every %v, data window=%v, baseline=%d days, Z-threshold=%.2f",
		is.pollingInterval, is.dataWindow, is.baselineDays, is.zScoreThreshold)

	ticker := time.NewTicker(is.pollingInterval)
	defer ticker.Stop()

	// Initial poll
	is.pollAllDevices(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("InferenceService: Shutting down...")
			close(is.InferenceReqChan)
			log.Println("InferenceService: Shutdown complete")
			return
		case <-ticker.C:
			is.pollAllDevices(ctx)
		}
	}
}

// pollAllDevices checks all known devices for inference triggers
func (is *InferenceService) pollAllDevices(ctx context.Context) {
	is.mu.RLock()
	devices := make([]string, 0, len(is.trackedDevices))
	for deviceID := range is.trackedDevices {
		devices = append(devices, deviceID)
	}
	is.mu.RUnlock()

	if len(devices) == 0 {
		// Try to discover devices from device registry
		// For now, we'll just wait for devices to appear through other means
		return
	}

	log.Printf("InferenceService: Polling %d devices", len(devices))

	for _, deviceID := range devices {
		if ctx.Err() != nil {
			return // Context cancelled
		}
		is.checkDevice(deviceID)
	}
}

// checkDevice checks a single device and triggers inference if needed
func (is *InferenceService) checkDevice(deviceID string) {
	// Get last inference timestamp
	lastInferenceTime, err := is.db.GetLastInferenceTimestamp(deviceID)
	if err != nil {
		log.Printf("InferenceService: Error getting last inference time for %s: %v", deviceID, err)
		return
	}

	// Get current window aggregates
	currentAgg, err := is.db.GetCurrentWindowAggregates(deviceID, int(is.dataWindow.Seconds()))
	if err != nil {
		log.Printf("InferenceService: Error getting current aggregates for %s: %v", deviceID, err)
		return
	}

	if !currentAgg.HasData {
		log.Printf("InferenceService: No current data for %s, skipping", deviceID)
		return
	}

	// If no previous inference, trigger immediately
	if lastInferenceTime.IsZero() {
		log.Printf("InferenceService: First inference for %s, triggering immediately", deviceID)
		is.triggerInference(deviceID, currentAgg, 0, 0, 0, "first_inference")
		return
	}

	// Get last inference window aggregates
	lastAgg, err := is.db.GetLastInferenceWindowAggregates(deviceID, lastInferenceTime, int(is.dataWindow.Seconds()))
	if err != nil {
		log.Printf("InferenceService: Error getting last inference aggregates for %s: %v", deviceID, err)
		return
	}

	if !lastAgg.HasData {
		log.Printf("InferenceService: No last inference data for %s, triggering", deviceID)
		is.triggerInference(deviceID, currentAgg, 0, 0, 0, "missing_last_data")
		return
	}

	// Get historical baseline statistics
	baseline, err := is.db.GetHistoricalBaselineStats(deviceID, is.baselineDays)
	if err != nil {
		log.Printf("InferenceService: Error getting baseline stats for %s: %v", deviceID, err)
		return
	}

	// Calculate Z-scores for each sensor type
	tempZScore := is.calculateZScore(currentAgg.Temperature, lastAgg.Temperature, baseline.Temperature)
	humidityZScore := is.calculateZScore(currentAgg.Humidity, lastAgg.Humidity, baseline.Humidity)
	volumeZScore := is.calculateZScore(currentAgg.SoundVolume, lastAgg.SoundVolume, baseline.SoundVolume)

	log.Printf("InferenceService: Device %s Z-scores: temp=%.2f, humidity=%.2f, volume=%.2f",
		deviceID, tempZScore, humidityZScore, volumeZScore)

	// Check if any Z-score exceeds threshold
	shouldTrigger := false
	triggerReason := ""

	if math.Abs(tempZScore) >= is.zScoreThreshold {
		shouldTrigger = true
		triggerReason = "temperature_zscore"
	}
	if math.Abs(humidityZScore) >= is.zScoreThreshold {
		shouldTrigger = true
		if triggerReason != "" {
			triggerReason += ",humidity_zscore"
		} else {
			triggerReason = "humidity_zscore"
		}
	}
	if math.Abs(volumeZScore) >= is.zScoreThreshold {
		shouldTrigger = true
		if triggerReason != "" {
			triggerReason += ",volume_zscore"
		} else {
			triggerReason = "volume_zscore"
		}
	}

	if shouldTrigger {
		log.Printf("InferenceService: Triggering inference for %s (reason: %s)", deviceID, triggerReason)
		is.triggerInference(deviceID, currentAgg, tempZScore, humidityZScore, volumeZScore, triggerReason)
	}
}

// calculateZScore computes normalized Z-score
// Z = (current - last) / historical_std_dev
func (is *InferenceService) calculateZScore(current, last, stdDev float64) float64 {
	if stdDev == 0 {
		// Avoid division by zero - if no variance, no significant change
		return 0
	}
	return (current - last) / stdDev
}

// triggerInference creates and sends an inference request
func (is *InferenceService) triggerInference(deviceID string, agg *database.SensorAggregates, tempZ, humidityZ, volumeZ float64, reason string) {
	// Save inference history
	err := is.db.SaveInferenceHistory(deviceID, reason, tempZ, humidityZ, volumeZ)
	if err != nil {
		log.Printf("InferenceService: Error saving inference history for %s: %v", deviceID, err)
	}

	// Create inference request
	request := &models.InferenceRequest{
		DeviceID:    deviceID,
		Timestamp:   time.Now(),
		Temperature: agg.Temperature,
		Humidity:    agg.Humidity,
		SoundVolume: agg.SoundVolume,
	}

	// Send request to channel (non-blocking with timeout)
	select {
	case is.InferenceReqChan <- request:
		log.Printf("InferenceService: Inference request sent for %s (temp=%.2f°C, humidity=%.2f%%, volume=%.2f dB)",
			deviceID, request.Temperature, request.Humidity, request.SoundVolume)
	case <-time.After(1 * time.Second):
		log.Printf("InferenceService: Warning - Inference request channel full, dropping request for %s", deviceID)
	}
}

// RegisterDevice adds a device to the tracking list
func (is *InferenceService) RegisterDevice(deviceID string) {
	is.mu.Lock()
	defer is.mu.Unlock()

	if !is.trackedDevices[deviceID] {
		is.trackedDevices[deviceID] = true
		log.Printf("InferenceService: Now tracking device %s", deviceID)
	}
}

// GetTrackedDevices returns all tracked device IDs
func (is *InferenceService) GetTrackedDevices() []string {
	is.mu.RLock()
	defer is.mu.RUnlock()

	devices := make([]string, 0, len(is.trackedDevices))
	for deviceID := range is.trackedDevices {
		devices = append(devices, deviceID)
	}
	return devices
}
