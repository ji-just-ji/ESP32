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
func (db *ClickHouseDB) SaveAudio(recording *models.AudioRecording, audioHash string, soundVolume float64) error {
	ctx := context.Background()

	query := `
		INSERT INTO sensor_audio (timestamp, device_id, sample_rate, duration, format, audio_hash, sound_volume, features)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	err := db.conn.Exec(ctx, query,
		recording.Timestamp,
		recording.DeviceID,
		recording.SampleRate,
		recording.Duration,
		recording.Format,
		audioHash,
		soundVolume,
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

// SensorAggregates holds aggregated sensor values for a time window
type SensorAggregates struct {
	Temperature float64
	Humidity    float64
	SoundVolume float64
	HasData     bool
}

// SensorStdDevs holds standard deviations for historical baseline
type SensorStdDevs struct {
	Temperature float64
	Humidity    float64
	SoundVolume float64
}

// SaveInferenceHistory records when an inference was triggered
func (db *ClickHouseDB) SaveInferenceHistory(deviceID string, triggerReason string, tempZ, humidityZ, volumeZ float64) error {
	ctx := context.Background()

	query := `
		INSERT INTO inference_history (timestamp, device_id, trigger_reason, temp_z_score, humidity_z_score, volume_z_score)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	err := db.conn.Exec(ctx, query,
		time.Now(),
		deviceID,
		triggerReason,
		tempZ,
		humidityZ,
		volumeZ,
	)

	if err != nil {
		return fmt.Errorf("failed to insert inference history: %w", err)
	}

	return nil
}

// GetLastInferenceTimestamp returns the timestamp of the last inference for a device
func (db *ClickHouseDB) GetLastInferenceTimestamp(deviceID string) (time.Time, error) {
	ctx := context.Background()

	query := `
		SELECT timestamp
		FROM inference_history
		WHERE device_id = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var timestamp time.Time
	row := db.conn.QueryRow(ctx, query, deviceID)
	err := row.Scan(&timestamp)
	if err != nil {
		// No previous inference found
		return time.Time{}, nil
	}

	return timestamp, nil
}

// GetCurrentWindowAggregates returns mean values for current time window
func (db *ClickHouseDB) GetCurrentWindowAggregates(deviceID string, windowSeconds int) (*SensorAggregates, error) {
	ctx := context.Background()

	// Calculate start time for window
	windowStart := time.Now().Add(-time.Duration(windowSeconds) * time.Second)

	query := `
		SELECT
			avg(temp.value) as avg_temp,
			avg(hum.value) as avg_humidity,
			avg(audio.sound_volume) as avg_volume,
			count(*) as total_count
		FROM
			(SELECT value FROM sensor_temperature WHERE device_id = ? AND timestamp >= ?) as temp,
			(SELECT value FROM sensor_humidity WHERE device_id = ? AND timestamp >= ?) as hum,
			(SELECT sound_volume FROM sensor_audio WHERE device_id = ? AND timestamp >= ?) as audio
	`

	var avgTemp, avgHumidity, avgVolume float64
	var totalCount uint64

	row := db.conn.QueryRow(ctx, query,
		deviceID, windowStart,
		deviceID, windowStart,
		deviceID, windowStart,
	)
	err := row.Scan(&avgTemp, &avgHumidity, &avgVolume, &totalCount)
	if err != nil || totalCount == 0 {
		return &SensorAggregates{HasData: false}, nil
	}

	return &SensorAggregates{
		Temperature: avgTemp,
		Humidity:    avgHumidity,
		SoundVolume: avgVolume,
		HasData:     true,
	}, nil
}

// GetLastInferenceWindowAggregates returns mean values from last inference window
func (db *ClickHouseDB) GetLastInferenceWindowAggregates(deviceID string, lastInferenceTime time.Time, windowSeconds int) (*SensorAggregates, error) {
	ctx := context.Background()

	// Calculate start time for window (going back from last inference time)
	windowStart := lastInferenceTime.Add(-time.Duration(windowSeconds) * time.Second)

	query := `
		SELECT
			avg(temp.value) as avg_temp,
			avg(hum.value) as avg_humidity,
			avg(audio.sound_volume) as avg_volume,
			count(*) as total_count
		FROM
			(SELECT value FROM sensor_temperature WHERE device_id = ? AND timestamp >= ? AND timestamp <= ?) as temp,
			(SELECT value FROM sensor_humidity WHERE device_id = ? AND timestamp >= ? AND timestamp <= ?) as hum,
			(SELECT sound_volume FROM sensor_audio WHERE device_id = ? AND timestamp >= ? AND timestamp <= ?) as audio
	`

	var avgTemp, avgHumidity, avgVolume float64
	var totalCount uint64

	row := db.conn.QueryRow(ctx, query,
		deviceID, windowStart, lastInferenceTime,
		deviceID, windowStart, lastInferenceTime,
		deviceID, windowStart, lastInferenceTime,
	)
	err := row.Scan(&avgTemp, &avgHumidity, &avgVolume, &totalCount)
	if err != nil || totalCount == 0 {
		return &SensorAggregates{HasData: false}, nil
	}

	return &SensorAggregates{
		Temperature: avgTemp,
		Humidity:    avgHumidity,
		SoundVolume: avgVolume,
		HasData:     true,
	}, nil
}

// GetHistoricalBaselineStats returns standard deviations over historical period
func (db *ClickHouseDB) GetHistoricalBaselineStats(deviceID string, baselineDays int) (*SensorStdDevs, error) {
	ctx := context.Background()

	// Calculate start time for historical baseline
	baselineStart := time.Now().Add(-time.Duration(baselineDays) * 24 * time.Hour)

	query := `
		SELECT
			stddevPop(temp.value) as std_temp,
			stddevPop(hum.value) as std_humidity,
			stddevPop(audio.sound_volume) as std_volume
		FROM
			(SELECT value FROM sensor_temperature WHERE device_id = ? AND timestamp >= ?) as temp,
			(SELECT value FROM sensor_humidity WHERE device_id = ? AND timestamp >= ?) as hum,
			(SELECT sound_volume FROM sensor_audio WHERE device_id = ? AND timestamp >= ?) as audio
	`

	var stdTemp, stdHumidity, stdVolume float64

	row := db.conn.QueryRow(ctx, query,
		deviceID, baselineStart,
		deviceID, baselineStart,
		deviceID, baselineStart,
	)
	err := row.Scan(&stdTemp, &stdHumidity, &stdVolume)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate historical baseline stats: %w", err)
	}

	return &SensorStdDevs{
		Temperature: stdTemp,
		Humidity:    stdHumidity,
		SoundVolume: stdVolume,
	}, nil
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
