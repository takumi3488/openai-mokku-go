# openai-mokku

An OpenAI API mock server written in Go. It echoes back user messages for testing purposes and supports both streaming and non-streaming responses.

## Features

- OpenAI API compatible endpoints
- Streaming support (Server-Sent Events)
- OpenTelemetry instrumentation for distributed tracing
- Code generation with [ogen](https://github.com/ogen-go/ogen)
- Docker support with Jaeger integration

## Quick Start

### Using Docker Compose (Recommended)

```bash
docker compose up --build
```

This starts the mock server on port 8080 and Jaeger UI on port 16686.

### Local Development

```bash
go run .
```

## API Endpoints

All endpoints are prefixed with `/v1`:

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/v1/models` | List available models |
| GET | `/v1/models/{model}` | Retrieve model details |
| POST | `/v1/chat/completions` | Chat completions (streaming supported) |
| POST | `/v1/completions` | Text completions |
| POST | `/v1/embeddings` | Embeddings |

## Usage Examples

### Chat Completion

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### Streaming Chat Completion

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": true
  }'
```

### List Models

```bash
curl http://localhost:8080/v1/models
```

## Error Simulation

You can simulate API errors by using special model names.

### 402 Credit Error (insufficient_quota)

Use model name `credit-error` to simulate a quota exceeded error:

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "credit-error",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

Response:
```json
{
  "error": {
    "message": "You exceeded your current quota, please check your plan and billing details.",
    "type": "insufficient_quota",
    "param": null,
    "code": "insufficient_quota"
  }
}
```

This works for both `/v1/chat/completions` and `/v1/completions` endpoints.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry OTLP endpoint | `jaeger:4317` |

## Development

### Build

```bash
go build
```

### Regenerate API Code

After modifying `openapi.yml`:

```bash
go generate ./...
```

### Docker Build

```bash
docker build .
```

## Architecture

```
openai-mokku/
├── api/              # Auto-generated ogen code (do not edit)
├── main.go           # Entry point, server setup, OpenTelemetry init
├── handler.go        # MockHandler for non-streaming endpoints
├── streaming.go      # StreamingHandler for SSE streaming
├── openapi.yml       # OpenAPI specification
└── ogen.yml          # ogen generator configuration
```

### Request Flow

1. HTTP requests are received by `StreamingHandler`
2. Streaming chat completion requests (`stream: true`) are handled directly in `streaming.go`
3. All other requests are passed through to the ogen-generated server

## License

MIT
