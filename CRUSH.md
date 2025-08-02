# PubDataHub Development Guide

## Build/Test/Lint Commands
- **Build**: `make build` or `go build -o pubdatahub cmd/main.go`
- **Test all**: `make test` or `go test -race -coverprofile=coverage.out ./...`
- **Test single**: `go test -race ./internal/command -run TestParserFunction`
- **Lint**: `make lint` (runs gofmt, go vet)
- **Format**: `make format` or `gofmt -s -w .`
- **Security**: `make security` (govulncheck, gosec)
- **Quick check**: `make quick-check` (fast pre-commit checks)
- **Full CI**: `make ci-check` (complete CI simulation)

## Code Style Guidelines
- **Package naming**: lowercase, single word (e.g., `hackernews`, `datasource`)
- **Imports**: standard library first, then third-party, then local packages
- **Error handling**: wrap errors with context using `fmt.Errorf("message: %w", err)`
- **Types**: use descriptive names, constants for enums (e.g., `JobState`, `JobType`)
- **Interfaces**: end with -er suffix when possible (e.g., `DataSource`)
- **Structs**: PascalCase for exported, camelCase for unexported
- **Functions**: PascalCase for exported, camelCase for unexported
- **Constants**: use const blocks with descriptive names (e.g., `JobStateQueued`)
- **File organization**: group related functionality, use internal/ for private packages
- **Testing**: use testify/assert, table-driven tests, _test.go suffix
- **Context**: always pass context.Context as first parameter for operations
- **Logging**: use logrus for structured logging
- **Configuration**: use viper for config management, struct tags for mapping
- **Database**: use sqlite3 with proper error handling and transactions
- **CLI**: use cobra for command structure, follow help/version patterns
- **Concurrency**: use proper synchronization, avoid data races
- **Dependencies**: prefer standard library, minimize external dependencies