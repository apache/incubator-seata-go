<!--
  ~ Licensed to the Apache Software Foundation (ASF) under one or more
  ~ contributor license agreements.  See the NOTICE file distributed with
  ~ this work for additional information regarding copyright ownership.
  ~ The ASF licenses this file to You under the Apache License, Version 2.0
  ~ (the "License"); you may not use this file except in compliance with
  ~ the License.  You may obtain a copy of the License at
  ~
  ~     http://www.apache.org/licenses/LICENSE-2.0
  ~
  ~ Unless required by applicable law or agreed to in writing, software
  ~ distributed under the License is distributed on an "AS IS" BASIS,
  ~ WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  ~ See the License for the specific language governing permissions and
  ~ limitations under the License.
-->

# test-provider

A provider mode Go agent generated on 2025-09-26 21:33:48.

## Description

This is a Go-based agent that provides skill-based functionality through HTTP APIs.

## Features

- RESTful API endpoints
- Configurable skills system
- Health check endpoint
- Agent card endpoint
- Docker support
- Structured logging

## Quick Start

1. Install dependencies:
   go mod tidy

2. Run the agent:
   go run main.go

3. Test the agent:
   curl http://localhost:8080/health
   curl http://localhost:8080/agent-card
   curl -X POST http://localhost:8080/skills/example -H "Content-Type: application/json" -d '{"input": "Hello World"}'

## Configuration

Edit config/config.yaml to customize the agent settings.

## Docker

docker-compose up --build

## API Endpoints

- GET /health - Health check
- GET /agent-card - Agent capabilities
- POST /skills/{skill-name} - Execute a skill

## Skills

- example: Processes input text and returns analysis

## Development

- scripts/build.sh - Build the project
- scripts/run.sh - Run in development mode
