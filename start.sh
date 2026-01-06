#!/bin/bash
set -e

echo "Starting DX Unified Server..."

# Build if needed
if [ ! -f "./dx-unified" ]; then
    echo "Building..."
    go build -o dx-unified ./cmd/server
fi

# Run
./dx-unified
