# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

openai-mokku is an OpenAI API mock server written in Go. It echoes back user messages for testing purposes and supports both streaming and non-streaming responses. The server is instrumented with OpenTelemetry for distributed tracing.

## Build & Run Commands

```bash
# Build the application
go build

# Run locally
go run .

# Run with Docker Compose (includes Jaeger for tracing)
docker compose up --build

# Docker build only
docker build .

# Regenerate ogen API code (after modifying openapi.yml)
go generate ./...
```

## API Endpoints

All endpoints are prefixed with `/v1`:
- `GET /v1/models` - List available models
- `GET /v1/models/{model}` - Get model details
- `POST /v1/chat/completions` - Chat completions (streaming supported via `stream: true`)
- `POST /v1/completions` - Text completions
- `POST /v1/embeddings` - Embeddings (handler not yet implemented)

## Architecture

### Code Generation with ogen
- `openapi.yml` - OpenAPI spec defining the API schema
- `ogen.yml` - ogen generator configuration
- `api/` - Auto-generated code by ogen (do not edit manually)

### Core Files
- `main.go` - Entry point, server setup, OpenTelemetry initialization
- `handler.go` - `MockHandler` implementing `api.Handler` interface (non-streaming endpoints)
- `streaming.go` - `StreamingHandler` wrapper for SSE streaming support on chat completions

### Request Flow
1. HTTP requests go to `StreamingHandler`
2. Streaming chat completion requests (`stream: true`) are handled directly in `streaming.go`
3. All other requests are passed through to the ogen-generated server

## Environment Variables

- `OTEL_EXPORTER_OTLP_ENDPOINT` - OpenTelemetry OTLP endpoint (default: `jaeger:4317`)

## Development Guidelines

- Go version: 1.25.4
- Comments and documentation must be in English
