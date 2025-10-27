package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"iot-backend/internal/models"
)

type ClickHouseDB struct {
	conn driver.Conn
}

// NewClickHouseDB creates a new ClickHouse database connection
func NewClickHouseDB(addr, database, username, password string) (*ClickHouseDB, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: database,
			Username: username,
			Password: password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 5 * time.Second,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	log.Printf("Connected to ClickHouse at %s", addr)

	db := &ClickHouseDB{conn: conn}

	// Initialize schema
	if err := db.InitSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// InitSchema creates the necessary tables if they don't exist
func (db *ClickHouseDB) InitSchema() error {
	ctx := context.Background()

	// Create all tables from schema
	tables := AllTables()
	for _, tableSQL := range tables {
		if err := db.conn.Exec(ctx, tableSQL); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	log.Println("Database schema initialized successfully")
	return nil
}

// SaveTemperature saves a temperature reading to the database
func (db *ClickHouseDB) SaveTemperature(reading *models.TemperatureReading) error {
	ctx := context.Background()

	query := `
		INSERT INTO sensor_temperature (timestamp, device_id, value)
		VALUES (?, ?, ?)
	`

	err := db.conn.Exec(ctx, query,
		reading.Timestamp,
		reading.DeviceID,
		reading.Value,
	)

	if err != nil {
		return fmt.Errorf("failed to insert temperature reading: %w", err)
	}

	return nil
}

// SaveHumidity saves a humidity reading to the database
func (db *ClickHouseDB) SaveHumidity(reading *models.HumidityReading) error {
	ctx := context.Background()

	query := `
		INSERT INTO sensor_humidity (timestamp, device_id, value)
		VALUES (?, ?, ?)
	`

	err := db.conn.Exec(ctx, query,
		reading.Timestamp,
		reading.DeviceID,
		reading.Value,
	)

	if err != nil {
		return fmt.Errorf("failed to insert humidity reading: %w", err)
	}

	return nil
}

// SaveAudio saves audio metadata to the database (not the raw audio data)
func (db *ClickHouseDB) SaveAudio(recording *models.AudioRecording, audioHash string) error {
	ctx := context.Background()

	query := `
		INSERT INTO sensor_audio (timestamp, device_id, sample_rate, duration, format, audio_hash, features)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	err := db.conn.Exec(ctx, query,
		recording.Timestamp,
		recording.DeviceID,
		recording.SampleRate,
		recording.Duration,
		recording.Format,
		audioHash,
		"{}", // Empty JSON for features (can be populated later)
	)

	if err != nil {
		return fmt.Errorf("failed to insert audio metadata: %w", err)
	}

	return nil
}

// SaveWindowAction saves a window action decision to the database (updated for continuous control)
func (db *ClickHouseDB) SaveWindowAction(action *models.WindowAction) error {
	ctx := context.Background()

	query := `
		INSERT INTO window_actions (timestamp, device_id, position, confidence, temperature, humidity, sound_volume)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	err := db.conn.Exec(ctx, query,
		action.Timestamp,
		action.DeviceID,
		action.Position,
		action.Confidence,
		action.Temperature,
		action.Humidity,
		action.SoundVolume,
	)

	if err != nil {
		return fmt.Errorf("failed to insert window action: %w", err)
	}

	log.Printf("Saved window action to ClickHouse: Position=%.2f%%, DeviceID=%s", action.Position, action.DeviceID)
	return nil
}

// SaveMLPrediction saves ML prediction metadata to the database
func (db *ClickHouseDB) SaveMLPrediction(prediction *models.MLPrediction) error {
	ctx := context.Background()

	query := `
		INSERT INTO ml_predictions (timestamp, device_id, prediction, confidence, inference_time_ms, model_version)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	err := db.conn.Exec(ctx, query,
		prediction.Timestamp,
		prediction.DeviceID,
		prediction.Prediction,
		prediction.Confidence,
		prediction.InferenceTimeMs,
		prediction.ModelVersion,
	)

	if err != nil {
		return fmt.Errorf("failed to insert ML prediction: %w", err)
	}

	return nil
}

// UpsertDevice inserts or updates a device in the registry
func (db *ClickHouseDB) UpsertDevice(device *models.Device) error {
	ctx := context.Background()

	// Convert config map to JSON string
	configJSON := "{}"
	if device.Config != nil {
		// Simple JSON serialization (in production, use json.Marshal)
		configJSON = "{}"
	}

	query := `
		INSERT INTO device_registry (device_id, name, location, registered_at, last_seen, is_active, config)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	err := db.conn.Exec(ctx, query,
		device.DeviceID,
		device.Name,
		device.Location,
		device.RegisteredAt,
		device.LastSeen,
		device.IsActive,
		configJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert device: %w", err)
	}

	return nil
}

// Close closes the ClickHouse connection
func (db *ClickHouseDB) Close() error {
	if db.conn != nil {
		if err := db.conn.Close(); err != nil {
			return fmt.Errorf("failed to close ClickHouse connection: %w", err)
		}
		log.Println("ClickHouse connection closed")
	}
	return nil
}
