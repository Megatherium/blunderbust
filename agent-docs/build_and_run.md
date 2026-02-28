# Build and Run Reference

**MANDATORY READ BEFORE:**
- Building the project (make/build commands)
- Running bdb locally
- Configuring CI/CD pipelines
- Adding new CLI flags or exit codes

---

## Building Blunderbust

Blunderbust supports two build configurations:

**Server-only build (default, ~20-30MB)**:
```bash
make build
```

**Full build with embedded support (~93MB)**:
```bash
make build-full
```

**Development builds with debug symbols**:
```bash
make debug          # Server-only
make debug-full     # Full build
```

**Installation**:
```bash
make install        # Server-only to GOPATH/bin
make install-full   # Full build to GOPATH/bin
```

**Which build to use**:
- Default build: Always use when testing or developing (faster builds)
- Full build: Only when testing embedded mode functionality
- CI/CD: Default build unless embedded mode is explicitly tested

---

## Running Blunderbust

**Critical**: Blunderbust must run inside tmux to create new windows.

```bash
# Start tmux if not already running
tmux

# Run with default config
bdb

# Run with custom config
bdb --config /path/to/config.yaml

# Dry run mode (useful for testing without launching)
bdb --dry-run

# Debug mode (verbose logging)
bdb --debug

# Demo mode (uses fake data)
bdb --demo
```

---

## Testing

```bash
# Run all tests with coverage
make test

# Run tests without coverage
go test -v ./...

# Run tests for a specific package
go test -v ./internal/config

# Run with race detector
go test -race ./...
```

---

## Common Development Tasks

**Testing config rendering**:
```bash
# Use dry-run to see rendered commands without launching
bdb --dry-run --debug
```

**Testing TUI with fake data**:
```bash
# Use demo mode to test UI without database
bdb --demo
```

**Debugging connection issues**:
```bash
# Enable debug logging to see database connection details
bdb --debug --beads-dir ./.beads
```

---

## Exit Codes

When implementing CLI behavior:
- `0` - Success
- `1` - General error
- `2` - Config error (file not found, parse error, validation error)
- `3` - No tmux (running outside tmux)

---

## Key Files to Understand

- `cmd/blunderbust/main.go` - CLI entrypoint, flag parsing, composition root
- `internal/config/yaml.go` - YAML config loading
- `internal/config/render.go` - Template rendering for commands/prompts
- `internal/domain/template_context.go` - Available fields for templates
- `internal/discovery/models.go` - Model discovery from models.dev
- `internal/ui/` - TUI implementation with Bubbletea
- `internal/data/dolt/` - Beads/Dolt database access

---

## Model Discovery

Blunderbust supports dynamic model discovery from `models.dev/api.json`.

**Commands**:
- `blunderbust update-models`: Manually refresh the local model cache.

**Config Patterns**:
- `provider:<id>`: Expands to all active models for that provider.
- `discover:active`: Expands to all models from all providers that have their required environment variables set.
