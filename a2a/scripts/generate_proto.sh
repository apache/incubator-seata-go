#!/bin/bash
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# Generate Protocol Buffers code for A2A Protocol
set -e

PROTO_DIR="api/proto/v1"
OUT_DIR="pkg/proto/v1"
GOOGLEAPIS_DIR="third_party/googleapis"
PROTOBUF_DIR="third_party/protobuf/src"

echo "Generating Go code from Protocol Buffers..."

# Create output directory if it doesn't exist
mkdir -p "$OUT_DIR"

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc is not installed"
    echo "Please install Protocol Buffers compiler:"
    echo "  - macOS: brew install protobuf"
    echo "  - Ubuntu: apt-get install protobuf-compiler"
    echo "  - Or download from: https://github.com/protocolbuffers/protobuf/releases"
    exit 1
fi

# Check if protoc-gen-go is installed
if ! command -v protoc-gen-go &> /dev/null; then
    echo "Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

# Check if protoc-gen-go-grpc is installed
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Check if third_party directories exist
if [ ! -d "$GOOGLEAPIS_DIR" ]; then
    echo "Error: $GOOGLEAPIS_DIR not found"
    echo "Please run: cd third_party && git clone https://github.com/googleapis/googleapis.git"
    exit 1
fi

if [ ! -d "$PROTOBUF_DIR" ]; then
    echo "Error: $PROTOBUF_DIR not found"
    echo "Please run: cd third_party && git clone https://github.com/protocolbuffers/protobuf.git"
    exit 1
fi

# Generate Go code with proper include paths
protoc --proto_path="$PROTO_DIR" \
    --proto_path="$GOOGLEAPIS_DIR" \
    --proto_path="$PROTOBUF_DIR" \
    --go_out="$OUT_DIR" --go_opt=paths=source_relative \
    --go-grpc_out="$OUT_DIR" --go-grpc_opt=paths=source_relative \
    "$PROTO_DIR"/*.proto

echo "Protocol Buffers code generation completed successfully!"
echo "Generated files in: $OUT_DIR"