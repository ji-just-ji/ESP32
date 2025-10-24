package ml

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"iot-backend/internal/models"
)

// Model represents a simple linear regression model
type Model struct {
	Coefficients map[string]float64 `json:"coefficients"`
	Intercept    float64            `json:"intercept"`
	Threshold    float64            `json:"threshold"` // Threshold to decide open vs close
}

// Predictor handles ML predictions
type Predictor struct {
	model *Model
}

// NewPredictor creates a new predictor by loading the model from file
func NewPredictor(modelPath string) (*Predictor, error) {
	data, err := os.ReadFile(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read model file: %w", err)
	}

	var model Model
	if err := json.Unmarshal(data, &model); err != nil {
		return nil, fmt.Errorf("failed to unmarshal model: %w", err)
	}

	log.Printf("Loaded model from %s with threshold: %.2f", modelPath, model.Threshold)

	return &Predictor{model: &model}, nil
}

// Predict takes sensor readings and predicts whether to open or close the window
func (p *Predictor) Predict(reading *models.SensorReading) string {
	// Calculate prediction score using linear regression
	score := p.model.Intercept

	if coef, ok := p.model.Coefficients["temperature"]; ok {
		score += coef * reading.Temperature
	}
	if coef, ok := p.model.Coefficients["humidity"]; ok {
		score += coef * reading.Humidity
	}
	if coef, ok := p.model.Coefficients["sound"]; ok {
		score += coef * reading.Sound
	}

	log.Printf("Prediction score: %.4f (threshold: %.2f)", score, p.model.Threshold)

	// Decide action based on threshold
	if score >= p.model.Threshold {
		return models.ActionOpen
	}
	return models.ActionClose
}

// CreateSampleModel creates a sample model file for demonstration
// Call this if no model file exists
func CreateSampleModel(path string) error {
	// Sample model: Open window if temperature is high or humidity is high
	// Close window if sound is too loud (noisy outside)
	model := Model{
		Coefficients: map[string]float64{
			"temperature": 0.3,   // Higher temp -> open window
			"humidity":    -0.2,  // Higher humidity -> close window
			"sound":       -0.15, // Louder noise -> close window
		},
		Intercept: 0.0,
		Threshold: 5.0, // Threshold for decision boundary
	}

	data, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal model: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write model file: %w", err)
	}

	log.Printf("Created sample model at %s", path)
	return nil
}
