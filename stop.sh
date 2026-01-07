#!/bin/bash

echo "Stopping Market Mosaic..."

if ! command -v podman-compose &> /dev/null; then
    COMPOSE_CMD="docker-compose"
else
    COMPOSE_CMD="podman-compose"
fi

$COMPOSE_CMD down
echo "Stopped."
