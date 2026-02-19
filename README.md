# Blunderbuss

Launch development harnesses (OpenCode, Claude, etc.) in tmux windows with context from Beads issues.

## Overview

Blunderbuss provides a TUI-driven workflow for:
- Selecting tickets from your Beads/Dolt issue database
- Choosing harness configurations (which tool, model, agent)
- Launching development sessions in organized tmux windows

## Building

Requires Go 1.22 or later.

```bash
# Build the binary
make build

# Or with go directly
go build -o blunderbuss ./cmd/blunderbuss
```

## Running

```bash
# Run with default config
./blunderbuss

# Run with custom config
./blunderbuss --config /path/to/config.yaml

# Dry run (print commands without executing)
./blunderbuss --dry-run

# Debug mode
./blunderbuss --debug
```

## Development

```bash
# Run linter
make lint

# Run tests
make test

# Clean build artifacts
make clean
```

## Configuration

Configuration is loaded from a YAML file (default: `./config.yaml`).
See the example configuration for harness definitions.

## License

GPL-3.0 License - See LICENSE file for details.
