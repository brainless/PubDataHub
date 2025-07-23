# Backend CLAUDE.md

This file provides guidance to Claude Code when working with the Go backend in this repository.

## Project Overview

This is the Go backend for PubDataHub that provides RESTful API endpoints for the frontend application.

## Technology Stack

- **Language**: Go 1.21+
- **Framework**: Gin (github.com/gin-gonic/gin)
- **Type Generation**: tygo (github.com/gzuidhof/tygo)

## Project Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go              # Main server entry point
├── internal/
│   ├── api/
│   │   └── handlers/            # HTTP request handlers
│   │       └── home.go          # Home directory endpoint
│   └── types/                   # Go struct definitions
│       └── api.go               # API response types
├── scripts/
│   └── generate-types.sh        # TypeScript type generation script
├── go.mod                       # Go module definition
├── go.sum                       # Go dependency checksums
├── tygo.yaml                    # TypeScript generation config
└── api-types.ts                 # Generated TypeScript types
```

## Development Commands

### Start the server
```bash
go run cmd/server/main.go
```
Server runs on port 8080 by default (configurable via PORT environment variable).

### Generate TypeScript types
```bash
./scripts/generate-types.sh
```
This generates TypeScript interfaces from Go structs for frontend consumption.

### Install dependencies
```bash
go mod tidy
```

### Test endpoints
```bash
# Test home endpoint
curl http://localhost:8080/api/home
```

## API Endpoints

### GET /api/home
Returns the user's home directory path.

**Response:**
```json
{
  "homePath": "/home/username"
}
```

**Error Response:**
```json
{
  "error": "Failed to get home directory",
  "message": "Unable to determine user home directory"
}
```

## Development Guidelines

1. **Add new endpoints**: Create handlers in `internal/api/handlers/`
2. **Define response types**: Add structs to `internal/types/api.go`
3. **Generate types**: Run `./scripts/generate-types.sh` after adding new types
4. **CORS**: Already configured for frontend integration
5. **Error handling**: Use consistent error response format from `types.ErrorResponse`

## Build and Deployment

### Build binary
```bash
go build -o bin/server cmd/server/main.go
```

### Run binary
```bash
./bin/server
```

## TypeScript Integration

- TypeScript types are automatically generated from Go structs using tygo
- Generated types are saved to `api-types.ts`
- Frontend can import these types for type-safe API communication
- Re-run type generation after modifying structs in `internal/types/`

## Notes

- Server includes basic CORS configuration for browser-based frontend
- JSON response format is used throughout
- Error responses follow consistent structure
- Logging is configured via Gin's default middleware