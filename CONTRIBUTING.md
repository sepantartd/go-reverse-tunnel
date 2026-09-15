# Contributing to go-reverse-tunnel

First off, thank you for considering contributing to `go-reverse-tunnel`!

## Getting Started

1. **Fork the repository** on GitHub.
2. **Clone your fork** locally:
   ```bash
   git clone [https://github.com/sepantartd/go-reverse-tunnel.git](https://github.com/sepantartd/go-reverse-tunnel.git)
   cd go-reverse-tunnel
   ```

## Prerequisites

- **Go 1.21+** installed on your system.

## Development Workflow

### 1. Code Formatting
All Go source code must adhere to standard formatting. Run:
```bash
gofmt -w .
```

### 2. Static Analysis & Linting
Ensure no static analysis flaws exist:
```bash
go vet ./...
```

### 3. Running Unit Tests
All packages must pass unit tests before submitting a Pull Request:
```bash
go test -count=1 ./pkg/config ./pkg/protocol ./pkg/tunnel
```

### 4. Building Project Binaries
To build both server and client binaries locally:
```bash
# Build Server
go build -o bin/server ./cmd/server

# Build Client
go build -o bin/client ./cmd/client
```

## Commit Message Conventions

We follow Conventional Commits standard:
- `feat:` A new feature
- `fix:` A bug fix
- `docs:` Documentation only changes
- `refactor:` Code change that neither fixes a bug nor adds a feature
- `test:` Adding missing tests or correcting existing tests
- `ci:` Changes to CI configuration files and scripts
- `chore:` Maintenance tasks

Example: `feat(tunnel): add support for custom heartbeat interval`

## Pull Request Guidelines

1. Create a descriptive branch name (e.g., `feature/socks5-auth` or `fix/hmac-nonce`).
2. Ensure all tests pass (`go test ./...`) and code is formatted (`gofmt -w .`).
3. Keep pull requests focused on a single change or bug fix.
4. Avoid breaking changes without prior discussion.
