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
