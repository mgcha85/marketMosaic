#!/bin/bash

echo "Stopping DX Unified Server..."
pkill -f dx-unified || true
echo "Stopped."
