package services

import (
	"context"
	"log"
	"time"

	"iot-backend/internal/aggregator"
	"iot-backend/internal/database"
	"iot-backend/internal/models"
)

// SensorService handles sensor data processing, persistence, and forwarding
type SensorService struct {
	db               *database.ClickHouseDB
	inferenceService *InferenceService

	// Input channels from MQTT subscribers
	TempChan     chan *models.TemperatureReading
	HumidityChan chan *models.HumidityReading
	AudioChan    chan *models.AudioRecording

	// Audio processor for volume extraction
	audioProcessor AudioProcessor
}

// AudioProcessor interface for extracting volume from audio
type AudioProcessor interface {
	ExtractVolume(audioData []byte, sampleRate int) float64
}

// defaultAudioProcessor implements AudioProcessor using the aggregator package
type defaultAudioProcessor struct{}

func (p *defaultAudioProcessor) ExtractVolume(audioData []byte, sampleRate int) float64 {
	return aggregator.ExtractSoundVolume(audioData, sampleRate)
}

// SensorServiceConfig holds configuration for sensor service
type SensorServiceConfig struct {
	TempChannelSize     int
	HumidityChannelSize int
	AudioChannelSize    int
}

// DefaultSensorServiceConfig returns default configuration
func DefaultSensorServiceConfig() SensorServiceConfig {
	return SensorServiceConfig{
		TempChannelSize:     100,
		HumidityChannelSize: 100,
		AudioChannelSize:    50, // Smaller since audio is larger
	}
}

// NewSensorService creates a new sensor service
func NewSensorService(
	db *database.ClickHouseDB,
	inferenceService *InferenceService,
	config SensorServiceConfig,
) *SensorService {
	return &SensorService{
		db:               db,
		inferenceService: inferenceService,
		TempChan:         make(chan *models.TemperatureReading, config.TempChannelSize),
		HumidityChan:     make(chan *models.HumidityReading, config.HumidityChannelSize),
		AudioChan:        make(chan *models.AudioRecording, config.AudioChannelSize),
		audioProcessor:   &defaultAudioProcessor{},
	}
}

// Start begins processing sensor data from channels
// Runs until context is cancelled
func (s *SensorService) Start(ctx context.Context) {
	log.Println("SensorService: Starting...")

	// Start goroutines for each sensor type
	go s.processTemperatureLoop(ctx)
	go s.processHumidityLoop(ctx)
	go s.processAudioLoop(ctx)

	log.Println("SensorService: All processing loops started")

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("SensorService: Shutting down...")

	// Close all channels
	close(s.TempChan)
	close(s.HumidityChan)
	close(s.AudioChan)

	log.Println("SensorService: Shutdown complete")
}

// processTemperatureLoop continuously processes temperature readings
func (s *SensorService) processTemperatureLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case reading, ok := <-s.TempChan:
			if !ok {
				return
			}
			s.processTemperature(reading)
		}
	}
}

// processHumidityLoop continuously processes humidity readings
func (s *SensorService) processHumidityLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case reading, ok := <-s.HumidityChan:
			if !ok {
				return
			}
			s.processHumidity(reading)
		}
	}
}

// processAudioLoop continuously processes audio recordings
func (s *SensorService) processAudioLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case recording, ok := <-s.AudioChan:
			if !ok {
				return
			}
			s.processAudio(recording)
		}
	}
}

// processTemperature handles a single temperature reading
func (s *SensorService) processTemperature(reading *models.TemperatureReading) {
	// Save to database
	if err := s.db.SaveTemperature(reading); err != nil {
		log.Printf("Error saving temperature: %v", err)
		return
	}

	log.Printf("Saved temperature: device=%s, value=%.2f°C", reading.DeviceID, reading.Value)

	// Auto-register device
	s.registerDevice(reading.DeviceID)
}

// processHumidity handles a single humidity reading
func (s *SensorService) processHumidity(reading *models.HumidityReading) {
	// Save to database
	if err := s.db.SaveHumidity(reading); err != nil {
		log.Printf("Error saving humidity: %v", err)
		return
	}

	log.Printf("Saved humidity: device=%s, value=%.2f%%", reading.DeviceID, reading.Value)

	// Auto-register device
	s.registerDevice(reading.DeviceID)
}

// processAudio handles a single audio recording
func (s *SensorService) processAudio(recording *models.AudioRecording) {
	// Extract sound volume from audio data
	volume := s.audioProcessor.ExtractVolume(recording.Data, recording.SampleRate)

	log.Printf("Extracted volume: device=%s, volume=%.2f dB, duration=%.2fs",
		recording.DeviceID, volume, recording.Duration)

	// Compute audio hash for reference
	audioHash := aggregator.ComputeAudioHash(recording.Data)

	// Save audio metadata to database (not the raw data)
	if err := s.db.SaveAudio(recording, audioHash, volume); err != nil {
		log.Printf("Error saving audio metadata: %v", err)
		return
	}

	log.Printf("Saved audio metadata: device=%s, hash=%s, volume=%.2f dB", recording.DeviceID, audioHash[:8], volume)

	// Auto-register device
	s.registerDevice(recording.DeviceID)
}

// registerDevice auto-registers a device on first message
func (s *SensorService) registerDevice(deviceID string) {
	device := &models.Device{
		DeviceID:     deviceID,
		Name:         deviceID,
		Location:     "Unknown",
		RegisteredAt: time.Now(),
		LastSeen:     time.Now(),
		IsActive:     true,
		Config:       make(map[string]interface{}),
	}

	// Best effort - don't fail if registration fails
	if err := s.db.UpsertDevice(device); err != nil {
		log.Printf("Error registering device %s: %v", deviceID, err)
	}

	// Register device with inference service for tracking
	if s.inferenceService != nil {
		s.inferenceService.RegisterDevice(deviceID)
	}
}
