package aggregator

import (
	"encoding/binary"
	"log"
	"math"
)

// AudioConfig holds configuration for audio processing
type AudioConfig struct {
	BitsPerSample int     // Typically 16 for 16-bit PCM
	ReferenceLevel float64 // Reference level for dB calculation (default: 32768.0 for 16-bit)
	MinimumRMS    float64 // Minimum RMS to avoid log(0), represents silence threshold
}

// DefaultAudioConfig returns default audio processing configuration
func DefaultAudioConfig() AudioConfig {
	return AudioConfig{
		BitsPerSample:  16,
		ReferenceLevel: 32768.0, // Maximum value for signed 16-bit audio
		MinimumRMS:     1.0,     // Prevents log(0) and extremely low values
	}
}

// ExtractSoundVolume extracts sound volume in dB from audio data
// Assumes 16-bit PCM little-endian format (standard for WAV files)
func ExtractSoundVolume(audioData []byte, sampleRate int) float64 {
	config := DefaultAudioConfig()
	return ExtractSoundVolumeWithConfig(audioData, sampleRate, config)
}

// ExtractSoundVolumeWithConfig extracts sound volume with custom configuration
func ExtractSoundVolumeWithConfig(audioData []byte, sampleRate int, config AudioConfig) float64 {
	if len(audioData) == 0 {
		log.Printf("Warning: Empty audio data received, returning silence level")
		return calculateDecibels(config.MinimumRMS, config.ReferenceLevel)
	}

	// For 16-bit PCM, each sample is 2 bytes
	bytesPerSample := config.BitsPerSample / 8
	if len(audioData)%bytesPerSample != 0 {
		log.Printf("Warning: Audio data length (%d) not aligned to sample size (%d bytes), truncating",
			len(audioData), bytesPerSample)
	}

	// Parse samples and calculate RMS
	rms := calculateRMS16Bit(audioData)

	// Apply minimum threshold to avoid log(0)
	if rms < config.MinimumRMS {
		rms = config.MinimumRMS
	}

	// Convert to decibels
	db := calculateDecibels(rms, config.ReferenceLevel)

	log.Printf("Audio processing: samples=%d, RMS=%.2f, volume=%.2f dB",
		len(audioData)/bytesPerSample, rms, db)

	return db
}

// calculateRMS16Bit calculates RMS from 16-bit PCM audio data
// Assumes little-endian format (standard for WAV files on most platforms)
func calculateRMS16Bit(audioData []byte) float64 {
	if len(audioData) < 2 {
		return 0.0
	}

	var sumSquares float64
	sampleCount := len(audioData) / 2

	for i := 0; i < len(audioData)-1; i += 2 {
		// Read 16-bit little-endian signed integer
		sample := int16(binary.LittleEndian.Uint16(audioData[i : i+2]))

		// Convert to float and accumulate squared values
		floatSample := float64(sample)
		sumSquares += floatSample * floatSample
	}

	// Calculate mean of squares
	meanSquares := sumSquares / float64(sampleCount)

	// Return square root (RMS)
	return math.Sqrt(meanSquares)
}

// calculateDecibels converts RMS value to decibels
// Formula: dB = 20 * log10(RMS / reference)
func calculateDecibels(rms float64, reference float64) float64 {
	if rms <= 0 || reference <= 0 {
		return -60.0 // Return a very low dB value for silence
	}

	ratio := rms / reference
	db := 20.0 * math.Log10(ratio)

	// Typical range for 16-bit audio: -60 dB (quiet) to 0 dB (maximum)
	// Clamp to reasonable bounds
	if db < -80.0 {
		db = -80.0 // Lower bound for practical silence
	}
	if db > 0.0 {
		db = 0.0 // Upper bound (clipping would occur beyond this)
	}

	return db
}

// AnalyzeAudioQuality provides basic audio quality metrics
type AudioQualityMetrics struct {
	RMS            float64 // RMS value
	VolumeDB       float64 // Volume in decibels
	PeakAmplitude  int16   // Peak sample value
	IsClipping     bool    // True if clipping detected
	IsSilent       bool    // True if audio is essentially silent
	SampleCount    int     // Number of samples
}

// AnalyzeAudio provides detailed audio analysis
func AnalyzeAudio(audioData []byte, sampleRate int) AudioQualityMetrics {
	config := DefaultAudioConfig()
	metrics := AudioQualityMetrics{
		SampleCount: len(audioData) / 2,
	}

	if len(audioData) < 2 {
		metrics.IsSilent = true
		metrics.VolumeDB = -80.0
		return metrics
	}

	var sumSquares float64
	var peakAmp int16 = 0
	clippingThreshold := int16(32000) // Close to max value of 32767

	for i := 0; i < len(audioData)-1; i += 2 {
		sample := int16(binary.LittleEndian.Uint16(audioData[i : i+2]))

		// Track peak amplitude
		absSample := sample
		if absSample < 0 {
			absSample = -absSample
		}
		if absSample > peakAmp {
			peakAmp = absSample
		}

		// Check for clipping
		if absSample > clippingThreshold {
			metrics.IsClipping = true
		}

		// Accumulate for RMS
		floatSample := float64(sample)
		sumSquares += floatSample * floatSample
	}

	// Calculate RMS
	meanSquares := sumSquares / float64(metrics.SampleCount)
	metrics.RMS = math.Sqrt(meanSquares)
	metrics.PeakAmplitude = peakAmp

	// Check for silence (RMS below threshold)
	if metrics.RMS < config.MinimumRMS {
		metrics.IsSilent = true
		metrics.RMS = config.MinimumRMS
	}

	// Calculate dB
	metrics.VolumeDB = calculateDecibels(metrics.RMS, config.ReferenceLevel)

	return metrics
}
