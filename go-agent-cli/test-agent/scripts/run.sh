#!/bin/bash
set -e

echo "Starting test-agent..."

if [ ! -f "./config/config.yaml" ]; then
    echo "Error: config/config.yaml not found"
    exit 1
fi

go run main.go
