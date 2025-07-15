# Tasks Platform Agent Guide

## Build/Test Commands
- **Test:** `go test ./...` (run all tests)
- **Test single:** `go test ./path/to/package -run TestName`
- **Build:** `go build ./cmd/...`  
- **Run:** `go run ./cmd/main.go`
- **Lint:** `go vet ./...` and `gofmt -s -w .`

## Architecture
- **Entry point:** `cmd/main.go` - HTTP server with Redis-backed task queue
- **Core components:** `internal/api/server.go` (Gin web server), `internal/config/config.go` (env config)
- **Database:** Redis for task queue (using asynq library)
- **Key dependencies:** Gin HTTP framework, asynq task queue, envconfig

## Code Style
- **Imports:** Standard library first, then third-party, then local packages
- **Naming:** camelCase for vars/funcs, PascalCase for exported types
- **Error handling:** Return errors, don't panic
- **Struct tags:** JSON tags for API types, envconfig for config
- **Types:** Use `json.RawMessage` for dynamic payloads
- **Validation:** Gin binding tags for required fields

## Project Structure
- `cmd/` - application entry points
- `internal/` - private app code (api, config)
- `pkg/` - public packages (dto for data transfer objects)
