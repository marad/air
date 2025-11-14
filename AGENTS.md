# Agent Guidelines for AIR (AI Requester)

## Build & Run
- Build: `make build` or `go build -o air main.go`
- Run: `./air <prompt_template.md>`
- Test: `go test ./...` (run all tests) or `go test -run TestName ./path` (single test)
- Format: `gofmt -w .` (format all Go files)
- Vet: `go vet ./...` (static analysis)

## Code Style
- **Language**: Go 1.21+, module name: `consistency`
- **Imports**: Standard lib first, then external packages (cloud.google.com, google.golang.org, github.com)
- **Formatting**: Use `gofmt` (tabs for indentation, standard Go formatting)
- **Error Handling**: Always wrap errors with context using `fmt.Errorf("context: %w", err)`
- **Naming**: camelCase for unexported, PascalCase for exported; descriptive names (e.g., `callVertexAI`, `projectID`)
- **Comments**: Use complete sentences; document exported functions/types
- **Dependencies**: Main: Google Cloud AI Platform, protobuf; uses Vertex AI Gemini models

## Project Structure
- Single binary CLI tool that sends prompts to Vertex AI (Gemini)
- Supports markdown prompt templates with YAML frontmatter configuration
- Environment: Reads `.env` for `GOOGLE_CLOUD_PROJECT` and `GOOGLE_CLOUD_LOCATION`
- Config: Temperature, topP, maxTokens, safety settings, response schema via YAML
