# Quick Start: Cleanup Deprecated Code

## TL;DR - Run This Command

```bash
cd /Users/bytedance/code/ESP32/mqtt_backbone
bash scripts/cleanup_deprecated.sh
```

This will remove:
- ✅ `internal/aggregator/sensor_buffer.go` (old callback-based aggregator)
- ✅ `internal/mqtt/multi_topic.go.deprecated` (old handler-based MQTT)

---

## What Was Already Cleaned

✅ **Completed automatically:**
1. Removed `AudioMetadata` struct from `models/sensor.go`
2. Removed `SensorReading` legacy struct from `models/sensor.go`
3. Updated `SPEC.md` to v1.5 with new architecture diagrams
4. Updated `PLAN.md` with Phase 1.5 details

---

## Manual Cleanup (Alternative to Script)

If you prefer manual deletion:

```bash
cd /Users/bytedance/code/ESP32/mqtt_backbone

# Remove deprecated files
rm internal/aggregator/sensor_buffer.go
rm internal/mqtt/multi_topic.go.deprecated
```

---

## Verify Everything Works

```bash
# Should compile cleanly
go build ./cmd/server

# Run the server (if you have MQTT broker + ClickHouse running)
./server
```

---

## What's Left (v1.5 Architecture)

Your codebase now has:

**MQTT Layer (Pure Transport):**
- `internal/mqtt/client.go` - Connection management
- `internal/mqtt/subscriber.go` - Subscribe & write to channels
- `internal/mqtt/publisher.go` - Read from channels & publish

**Services Layer (Business Logic):**
- `internal/services/sensor_service.go` - Process sensors, extract volume, persist
- `internal/services/inference_service.go` - Smart triggering, rate limiting

**Audio Processing:**
- `internal/aggregator/audio_processor.go` - Extract sound volume (dB)

**Database:**
- `internal/database/clickhouse.go` - Persistence (updated with sound_volume)
- `internal/database/schema.go` - Updated schema

**Models:**
- `internal/models/sensor.go` - Clean models (no deprecated structs)
- `internal/models/audio.go` - Audio recording model
- `internal/models/device.go` - Device registry

---

## Next Steps

1. ✅ Run cleanup script
2. ✅ Verify build: `go build ./cmd/server`
3. ✅ Review `CLEANUP_SUMMARY.md` for details
4. ✅ Move to Phase 2: Python ML Microservice

---

## Need Help?

- **Architecture details:** Read `SPEC.md` (updated to v1.5)
- **Cleanup details:** Read `CLEANUP_SUMMARY.md`
- **Implementation plan:** Read `PLAN.md` (Phase 1.5 section)
