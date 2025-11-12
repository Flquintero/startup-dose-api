# Go Proxy API

A minimal, production-ready Go HTTP service that proxies requests to JSONPlaceholder.

## Quick Start

```bash
go run ./cmd/server
```

Then in another terminal:

```bash
curl -i http://localhost:8080/posts/1
```

## Endpoints

- **GET /posts/1** - Fetches post #1 from JSONPlaceholder and returns it as JSON
- **GET /healthz** - Health check endpoint, returns `{"ok": true}`

## Features

- Standard library only (no external dependencies)
- HTTP client with 5-second timeout
- Graceful shutdown on SIGINT/SIGTERM
- CORS enabled for GET requests from any origin
- Request logging (method, path, status, latency)
- Consistent JSON responses with proper Content-Type headers
- Panic recovery middleware
- Proper error handling with 502 Bad Gateway responses

## Building

```bash
make build
```

Binary will be at `bin/server`.

## Docker

Build and run in Docker:

```bash
make docker-run
```

Or manually:

```bash
docker build -t go-proxy:latest .
docker run -p 8080:8080 go-proxy:latest
```

## Project Structure

```
.
├── cmd/server/main.go          # Server entry point
├── internal/http/
│   ├── handler.go              # HTTP handlers
│   ├── middleware.go           # Request middleware (logging, CORS, recovery)
│   ├── router.go               # Route setup
│   └── client.go               # Configured HTTP client
├── go.mod                       # Go module definition
├── Dockerfile                   # Container image
├── Makefile                     # Build automation
└── README.md                    # This file
```

## Error Responses

When upstream returns non-200 or errors occur, responses follow this format:

```json
{
  "error": "bad_gateway",
  "message": "Failed to reach upstream service"
}
```

Status code: 502 Bad Gateway
