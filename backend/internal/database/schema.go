package database

// SQL schemas for all ClickHouse tables

const (
	// SensorTemperatureTableSQL creates the sensor_temperature table
	SensorTemperatureTableSQL = `
		CREATE TABLE IF NOT EXISTS sensor_temperature (
			timestamp DateTime64(3),
			device_id String,
			value Float64
		) ENGINE = MergeTree()
		ORDER BY (device_id, timestamp)
		PARTITION BY toYYYYMM(timestamp)
	`

	// SensorHumidityTableSQL creates the sensor_humidity table
	SensorHumidityTableSQL = `
		CREATE TABLE IF NOT EXISTS sensor_humidity (
			timestamp DateTime64(3),
			device_id String,
			value Float64
		) ENGINE = MergeTree()
		ORDER BY (device_id, timestamp)
		PARTITION BY toYYYYMM(timestamp)
	`

	// SensorAudioTableSQL creates the sensor_audio table
	SensorAudioTableSQL = `
		CREATE TABLE IF NOT EXISTS sensor_audio (
			timestamp DateTime64(3),
			device_id String,
			sample_rate UInt32,
			duration Float64,
			format String,
			audio_hash String,
			features String
		) ENGINE = MergeTree()
		ORDER BY (device_id, timestamp)
		PARTITION BY toYYYYMM(timestamp)
	`

	// WindowActionsTableSQL creates the window_actions table (updated for continuous control)
	WindowActionsTableSQL = `
		CREATE TABLE IF NOT EXISTS window_actions (
			timestamp DateTime64(3),
			device_id String,
			position Float64,
			confidence Float64,
			temperature Float64,
			humidity Float64,
			audio_hash String
		) ENGINE = MergeTree()
		ORDER BY (device_id, timestamp)
		PARTITION BY toYYYYMM(timestamp)
	`

	// DeviceRegistryTableSQL creates the device_registry table
	DeviceRegistryTableSQL = `
		CREATE TABLE IF NOT EXISTS device_registry (
			device_id String,
			name String,
			location String,
			registered_at DateTime64(3),
			last_seen DateTime64(3),
			is_active Bool,
			config String
		) ENGINE = ReplacingMergeTree(last_seen)
		ORDER BY device_id
	`

	// MLPredictionsTableSQL creates the ml_predictions table
	MLPredictionsTableSQL = `
		CREATE TABLE IF NOT EXISTS ml_predictions (
			timestamp DateTime64(3),
			device_id String,
			prediction Float64,
			confidence Float64,
			inference_time_ms Float64,
			model_version String
		) ENGINE = MergeTree()
		ORDER BY (device_id, timestamp)
		PARTITION BY toYYYYMM(timestamp)
	`

	// Legacy sensor_readings table (kept for backward compatibility)
	SensorReadingsTableSQL = `
		CREATE TABLE IF NOT EXISTS sensor_readings (
			timestamp DateTime64(3),
			device_id String,
			temperature Float64,
			humidity Float64,
			sound Float64
		) ENGINE = MergeTree()
		ORDER BY (device_id, timestamp)
		PARTITION BY toYYYYMM(timestamp)
	`
)

// AllTables returns all table creation SQL statements
func AllTables() []string {
	return []string{
		SensorTemperatureTableSQL,
		SensorHumidityTableSQL,
		SensorAudioTableSQL,
		WindowActionsTableSQL,
		DeviceRegistryTableSQL,
		MLPredictionsTableSQL,
		SensorReadingsTableSQL, // Legacy table
	}
}
