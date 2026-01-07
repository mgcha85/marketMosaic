#!/bin/bash
set -e

echo "Starting Market Mosaic (Full Stack)..."

# Ensure podman-compose is available or alias it
if ! command -v podman-compose &> /dev/null; then
    if command -v docker-compose &> /dev/null; then
        COMPOSE_CMD="docker-compose"
    else
        echo "Error: podman-compose or docker-compose not found."
        exit 1
    fi
else
    COMPOSE_CMD="podman-compose"
fi

$COMPOSE_CMD up -d --build
echo "Services started. Backend at localhost:8080, Frontend at localhost:5173 (if local) or configured port."
