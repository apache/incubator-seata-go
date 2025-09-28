#!/bin/bash
set -e

echo "Building test-agent..."

rm -rf ./bin
mkdir -p ./bin

go build -o ./bin/test-agent ./main.go

echo "Build completed successfully!"
echo "Binary location: ./bin/test-agent"
