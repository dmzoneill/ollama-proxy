# Contributing to Ollama Compute Proxy

Thank you for your interest in contributing to the Ollama Compute Proxy project! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Environment](#development-environment)
- [Code Style](#code-style)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Reporting Issues](#reporting-issues)

## Code of Conduct

Please be respectful and constructive in all interactions. We're here to build great software together.

## Getting Started

### Prerequisites

- Go 1.24 or later
- Git
- Make (optional, but recommended)

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/ollama-proxy.git
   cd ollama-proxy
   ```

3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/daoneill/ollama-proxy.git
   ```

## Development Environment

### Install Dependencies

```bash
go mod download
```

### Build the Project

```bash
go build -o bin/ollama-proxy ./cmd/proxy
```

Or with version information:

```bash
go build -ldflags "-X main.Version=dev -X main.GitCommit=$(git rev-parse HEAD) -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o bin/ollama-proxy ./cmd/proxy
```

### Run Locally

```bash
./bin/ollama-proxy --config config/config.yaml
```

Or directly with `go run`:

```bash
go run ./cmd/proxy --config config/config.yaml
```

### Run with Docker

```bash
docker-compose up --build
```

## Code Style

### Go Code Standards

- Follow standard Go conventions and idioms
- Use `gofmt` for formatting (enforced by CI)
- Use `goimports` for import organization
- Keep functions small and focused
- Add comments for exported types and functions

### Running Linters

We use `golangci-lint` for code quality checks:

```bash
golangci-lint run
```

Install golangci-lint:

```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Windows
choco install golangci-lint
```

### Code Organization

- Place new packages in `pkg/`
- Keep `cmd/` for executable entry points only
- Add tests in `*_test.go` files alongside source
- Use table-driven tests where appropriate

## Testing

### Running Tests

Run all tests:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -html=coverage.out -o coverage.html
```

Run tests for a specific package:

```bash
go test ./pkg/router/...
```

### Writing Tests

- Aim for 90%+ test coverage on new code
- Write unit tests for all public functions
- Use table-driven tests for multiple test cases
- Mock external dependencies
- Test error paths and edge cases

Example test structure:

```go
func TestRouteRequest(t *testing.T) {
    tests := []struct {
        name        string
        annotations *backends.Annotations
        want        string
        wantErr     bool
    }{
        {
            name: "latency critical request",
            annotations: &backends.Annotations{
                LatencyCritical: true,
            },
            want:    "fast-backend",
            wantErr: false,
        },
        // Add more test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Benchmarks

Add benchmarks for performance-critical code:

```go
func BenchmarkRouteRequest(b *testing.B) {
    // Setup
    for i := 0; i < b.N; i++ {
        // Benchmark code
    }
}
```

Run benchmarks:

```bash
go test -bench=. -benchmem ./pkg/router/
```

## Submitting Changes

### Before Submitting

1. **Update from upstream:**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run tests:**
   ```bash
   go test ./...
   ```

3. **Run linters:**
   ```bash
   golangci-lint run
   ```

4. **Check formatting:**
   ```bash
   gofmt -s -w .
   goimports -w .
   ```

### Commit Messages

Write clear, descriptive commit messages:

```
Short summary (50 chars or less)

More detailed explanation if needed. Wrap at 72 characters.
Explain the problem this commit solves and why you chose
this particular solution.

- Bullet points are okay
- Use present tense ("Add feature" not "Added feature")
- Reference issues: "Fixes #123" or "Relates to #456"
```

Examples:
- `Add circuit breaker to prevent cascading failures`
- `Fix race condition in thermal monitor`
- `Improve error messages in routing decisions`

### Pull Request Process

1. **Create a feature branch:**
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make your changes and commit:**
   ```bash
   git add .
   git commit -m "Add my feature"
   ```

3. **Push to your fork:**
   ```bash
   git push origin feature/my-feature
   ```

4. **Create a Pull Request on GitHub**

5. **PR Description should include:**
   - What problem does this solve?
   - How did you solve it?
   - Any breaking changes?
   - Test coverage added/modified?
   - Screenshots (if UI changes)

6. **PR Checklist:**
   - [ ] Tests pass locally
   - [ ] Linters pass
   - [ ] Added tests for new code
   - [ ] Updated documentation
   - [ ] No breaking changes (or documented)
   - [ ] Commit messages are clear

### Review Process

- A maintainer will review your PR
- Address any feedback or requested changes
- Once approved, a maintainer will merge your PR

## Reporting Issues

### Bug Reports

Include:
- **Description:** Clear description of the bug
- **Steps to Reproduce:** Detailed steps to reproduce
- **Expected Behavior:** What should happen
- **Actual Behavior:** What actually happens
- **Environment:** OS, Go version, config details
- **Logs:** Relevant log output (use `--log-level=debug`)

### Feature Requests

Include:
- **Use Case:** Why is this feature needed?
- **Proposed Solution:** How should it work?
- **Alternatives:** Other approaches considered?
- **Additional Context:** Any other relevant details

## Development Tips

### Debugging

Enable debug logging:

```bash
./bin/ollama-proxy --config config/config.yaml --log-level=debug
```

Use pprof for profiling:

```bash
# CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile

# Heap profile
go tool pprof http://localhost:6060/debug/pprof/heap
```

### Project Structure

```
ollama-proxy/
├── cmd/                    # Executable entry points
│   └── proxy/             # Main proxy server
├── pkg/                    # Reusable packages
│   ├── backends/          # Backend implementations
│   ├── router/            # Request routing logic
│   ├── thermal/           # Thermal monitoring
│   ├── efficiency/        # Efficiency modes
│   ├── metrics/           # Prometheus metrics
│   ├── validation/        # Input validation
│   ├── auth/              # Authentication
│   ├── ratelimit/         # Rate limiting
│   └── ...
├── api/                    # API definitions (protobuf)
├── config/                 # Configuration files
├── docs/                   # Documentation
└── tests/                  # Integration tests
```

### Useful Commands

```bash
# Build
go build ./cmd/proxy

# Test
go test ./...

# Test with coverage
go test -coverprofile=coverage.out ./...

# Lint
golangci-lint run

# Format
gofmt -s -w .
goimports -w .

# Update dependencies
go mod tidy

# Check for security issues
gosec ./...

# View module graph
go mod graph

# Clean build cache
go clean -cache
```

## Getting Help

- **Documentation:** Check the [docs/](docs/) directory
- **Discussions:** GitHub Discussions
- **Issues:** GitHub Issues

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.

---

Thank you for contributing to Ollama Compute Proxy! 🚀
