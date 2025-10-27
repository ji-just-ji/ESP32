#!/bin/bash
# Cleanup script to remove deprecated v1.0 architecture files
# Run this from the mqtt_backbone directory: bash scripts/cleanup_deprecated.sh

set -e

echo "=== Cleaning up deprecated files from v1.0 architecture ==="
echo ""

# Files to remove
DEPRECATED_FILES=(
    "internal/aggregator/sensor_buffer.go"
    "internal/mqtt/multi_topic.go.deprecated"
)

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "Error: Please run this script from the mqtt_backbone directory"
    echo "Usage: bash scripts/cleanup_deprecated.sh"
    exit 1
fi

echo "The following deprecated files will be removed:"
for file in "${DEPRECATED_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo "  ✓ $file (exists)"
    else
        echo "  ✗ $file (not found)"
    fi
done

echo ""
read -p "Do you want to proceed? (y/N): " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cleanup cancelled."
    exit 0
fi

echo ""
echo "Removing deprecated files..."

for file in "${DEPRECATED_FILES[@]}"; do
    if [ -f "$file" ]; then
        rm "$file"
        echo "  ✓ Removed: $file"
    else
        echo "  ✗ Not found: $file"
    fi
done

echo ""
echo "=== Cleanup complete! ==="
echo ""
echo "Summary of removed components:"
echo "  • internal/aggregator/sensor_buffer.go - Old callback-based aggregator"
echo "  • internal/mqtt/multi_topic.go.deprecated - Old handler-based MQTT layer"
echo ""
echo "New v1.5 architecture components:"
echo "  • internal/services/sensor_service.go - Channel-based sensor processing"
echo "  • internal/services/inference_service.go - Smart inference triggering"
echo "  • internal/mqtt/subscriber.go - Pure transport subscriber"
echo "  • internal/mqtt/publisher.go - Pure transport publisher"
echo ""
echo "Next steps:"
echo "  1. Run: go build ./cmd/server"
echo "  2. Run tests if available"
echo "  3. Commit the changes"
echo ""
