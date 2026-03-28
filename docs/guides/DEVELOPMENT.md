# Development Guide

Guide for developers working on mroki.

## Setup

### Prerequisites

- **Go 1.21+**
- **Docker & Docker Compose**
- **Node.js 18+** (for hub development)
- **PostgreSQL 15+** (via Docker is fine)
- **Make** (optional, for convenience)

### Clone Repository

```bash
git clone https://github.com/pedrobarco/mroki.git
cd mroki
```

### Install Dependencies

```bash
# Go dependencies (downloaded automatically on first build/run)
go mod download

# For hub development
cd cmd/mroki-hub
npm install
```

## Project Structure

```
mroki/
├── cmd/                    # Executables
│   ├── mroki-agent/       # Agent binary
│   ├── mroki-api/         # API binary
│   ├── mroki-hub/         # Web UI (Vue.js)
│   └── caddy-mroki/       # Caddy module main
├── internal/              # Private application code
│   ├── application/       # CQRS commands and queries
│   ├── domain/            # Business logic and domain models
│   ├── infrastructure/    # External concerns (database, etc.)
│   └── interfaces/        # HTTP handlers and middleware
├── pkg/                   # Public libraries
│   ├── proxy/             # Core proxy logic
│   ├── diff/              # JSON diffing
│   ├── client/            # API client
│   ├── logger/            # Structured logging
│   └── caddymodule/       # Caddy integration
├── docs/                  # Documentation
└── build/                 # Build configurations
```

## Development Workflow

### Running Tests

```bash
# Run all tests
go test ./...

# Run with race detector
go test -race ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/domain/...
go test ./pkg/proxy/...
```

### Running Components Locally

**Terminal 1: PostgreSQL**
```bash
docker compose -f build/dev/compose.yaml up
```

**Terminal 2: API**
```bash
cd cmd/mroki-api
go run .
```

**Terminal 3: Agent**
```bash
cd cmd/mroki-agent
go run .
```

**Terminal 4: Hub** (when implemented)
```bash
cd cmd/mroki-hub
npm run dev
```

### Building Binaries

```bash
# Build agent
go build -o mroki-agent ./cmd/mroki-agent

# Build API
go build -o mroki-api ./cmd/mroki-api

# Build Caddy with mroki module
cd cmd/caddy-mroki
go build -o caddy-mroki .

# Build all
go build ./...
```

## Code Style

### Go Code

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Use `golint` for linting
- Write tests for all new code
- Document exported functions

```bash
# Format code
gofmt -w .

# Run linter
golangci-lint run
```

### Vue/TypeScript Code

- Follow Vue 3 Composition API style
- **TypeScript is required** - All Vue components must use `lang="ts"`
- Use ESLint for linting
- Use Prettier for formatting
- **Use CSS variables for theming** (see Theming section below)

```bash
cd web/mroki-hub
pnpm lint
pnpm format
```

### TypeScript Requirement

**All Vue components MUST use TypeScript.** This is enforced by ESLint.

```vue
<!-- ✅ Correct: TypeScript with lang="ts" -->
<script setup lang="ts">
import { ref } from 'vue'

const count = ref<number>(0)
const increment = (): void => {
  count.value++
}
</script>

<!-- ❌ Incorrect: Plain JavaScript (ESLint error) -->
<script setup>
const count = ref(0)
// Error: The 'lang' attribute of '<script>' is missing
</script>
```

**Why this matters:**
- Ensures type safety across all components
- Catches errors at compile time
- Provides better IDE support and autocomplete
- Maintains consistency across the codebase

### Theming Convention

mroki-hub uses **CSS variables** for theming following [shadcn-vue conventions](https://www.shadcn-vue.com/docs/theming.html).

**Always use semantic color tokens:**

```vue
<!-- ✅ Correct: CSS variables -->
<div class="bg-background text-foreground">
  <button class="bg-primary text-primary-foreground">Action</button>
  <p class="text-muted-foreground">Secondary text</p>
</div>

<!-- ❌ Incorrect: Hardcoded colors -->
<div class="bg-white text-gray-900">
  <button class="bg-blue-600 text-white">Action</button>
  <p class="text-gray-500">Secondary text</p>
</div>
```

**Common color tokens:**
- `bg-background` / `text-foreground` - Main colors
- `bg-primary` / `text-primary-foreground` - Primary actions
- `text-muted-foreground` - Secondary text
- `bg-card` / `text-card-foreground` - Cards
- `bg-destructive` / `text-destructive-foreground` - Destructive actions
- `border` - Borders
- `ring` - Focus rings

See `web/mroki-hub/README.md` for complete theming documentation.

### Pre-commit Hooks

The project uses `pre-commit` to automatically run linters and formatters before commits:

```bash
# Install pre-commit hooks (first time only)
pre-commit install

# Run hooks manually on all files
pre-commit run --all-files

# Run hooks on staged files
pre-commit run
```

**Hooks configured:**
- Go: `go mod tidy`, `golangci-lint`
- Vue/TypeScript: `prettier`, `eslint` (for mroki-hub)
- YAML/JSON validation

## Testing Guidelines

### Unit Tests

- Test file naming: `*_test.go`
- Table-driven tests preferred
- Use mocks for external dependencies
- Test edge cases and error paths

**Example:**
```go
func TestProxy_ServeHTTP(t *testing.T) {
    tests := []struct {
        name           string
        liveResponse   int
        shadowResponse int
        wantStatus     int
    }{
        {"both success", 200, 200, 200},
        {"live fails", 500, 200, 500},
        {"shadow fails", 200, 500, 200},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

### Integration Tests

- Test realistic scenarios
- Use Docker for dependencies
- Clean up resources after tests

## Database Development

### Schema Changes

Currently using ent for schema definition and auto-migration.

**File Locations:**
- Schema: `ent/schema/` (e.g., `request.go`, `response.go`, `diff.go`, `gate.go`)
- Generated code: `ent/` (auto-generated by ent)

**To modify schema:**

1. Edit the relevant schema file in `ent/schema/`
2. Regenerate ent code:
   ```bash
   go generate ./ent/...
   ```
3. Update domain models, mappers, and repositories if needed

### Running Migrations

Currently applied automatically on startup. Future versions will use migration tools.

## Debugging

### Enable Debug Logging

```bash
# Agent
MROKI_APP_LOG_LEVEL=debug go run ./cmd/mroki-agent

# API
MROKI_APP_LOG_LEVEL=debug go run ./cmd/mroki-api
```

### Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug agent
cd cmd/mroki-agent
dlv debug

# Set breakpoints
(dlv) break main.main
(dlv) continue
```

### VS Code Configuration

**`.vscode/launch.json`:**
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Agent",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/mroki-agent"
    },
    {
      "name": "Debug API",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/mroki-api"
    }
  ]
}
```

## Contributing

1. **Fork** the repository
2. **Create** a feature branch: `git checkout -b feature/my-feature`
3. **Write** code and tests
4. **Commit** with clear messages: `git commit -m "feat: add sampling rate support"`
5. **Push** to your fork: `git push origin feature/my-feature`
6. **Open** a Pull Request

### Commit Message Format

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding tests
- `refactor`: Code refactoring
- `chore`: Maintenance tasks

**Examples:**
```
feat(agent): add retry logic with exponential backoff
fix(api): handle nil pointer in gate creation
docs: update API contracts documentation
test(proxy): add test for timeout handling
```

## Release Process

(To be defined - placeholder for future)

1. Update version in appropriate files
2. Update CHANGELOG
3. Create git tag
4. Build binaries
5. Create GitHub release
6. Publish to package registries

## Useful Commands

```bash
# Run all tests with verbose output
go test -v ./...

# Run tests for specific package
go test ./internal/domain/traffictesting -v

# Benchmark tests
go test -bench=. ./pkg/proxy

# Check for race conditions
go test -race ./...

# Generate mocks (using mockgen)
go generate ./...

# Clean build cache
go clean -cache

# Update dependencies
go get -u ./...
go mod tidy
```

## Related Documentation

- [Quick Start Guide](QUICK_START.md)
- [Architecture Overview](../architecture/OVERVIEW.md)
- [Deployment Guide](DEPLOYMENT.md)
