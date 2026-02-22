# Agent Instructions
## Issue Tracking

This project uses **bd (beads)** for issue tracking.
Run `bd prime` for workflow context, or install hooks (`bd hooks install`) for auto-injection.

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started.

## Quick Reference

```bash
bd ready              # Find available work
bd create "Title" --type task --priority 2` # Create issue
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```
For full workflow details: `bd prime`

## Landing the Plane (Session Completion)

**When ending a work session** before sayind "done" or "complete", you MUST complete ALL steps below. 
Work is NOT complete until `git push` succeeds.
Push is not allowed until the work is REVIEWED

**MANDATORY WORKFLOW:**
Phase 1:
  1. **File issues for remaining work** - Create issues for anything that needs follow-up
  2. **Run quality gates** (if code changed) - Tests, linters, builds
  3. **Run CODE REVIEW & REFINEMENT PROTOCOL** - See `bd prime` for details
Phase 2 (after SOMEONE ELSE has reviewed it):
  4. **Update issue status** - Close finished work, update in-progress items
  5. **PUSH TO REMOTE** - This is MANDATORY:
    ```bash
    git pull --rebase
    bd sync  # <-- RUN THIS BEFORE git add!
    git add (careful with using -A, the user sometimes leaves untracked crap lying around) && git commit ...
    git push
    git status  # MUST show "up to date with origin"
    ```
  6. **Clean up** - Clear stashes, prune remote branches
  7. **Verify** - All changes committed AND pushed
  8. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
- **ALL bd operations BEFORE any git operations** - bd sync first, then git add/commit/push
- Failure to follow this order creates double commits (one for code, one for .beads/issues.jsonl)
- MODIFY ticket, bd SYNC, git STAGE/PUSH - else you will be creating extra commits 

## Project Structure

### internal/data/dolt/ - Beads Database Access

The `dolt` package implements `data.TicketStore` for reading tickets from Beads/Dolt databases.

**Key files:**
- `metadata.go` - Parses `.beads/metadata.json` to determine connection mode
- `embedded.go` - Embedded Dolt driver (requires CGO, single-connection)
- `server.go` - MySQL driver for Dolt server connections
- `store.go` - Main `Store` type implementing `TicketStore`
- `schema.go` - Schema verification utilities

**Connection modes:**
- **Embedded**: Default, uses `github.com/dolthub/driver`, local `.beads/dolt/` directory
- **Server**: Activated by `dolt_mode: server` in metadata.json, uses MySQL protocol

**Usage:**
```go
store, err := dolt.NewStore(ctx, domain.AppOptions{BeadsDir: ".beads"})
if err != nil {
    // Handle with actionable error message
}
defer store.Close()

tickets, err := store.ListTickets(ctx, data.TicketFilter{
    Status: "open",
    Limit: 10,
})
```

**Error handling:** All errors include context. Common patterns:
- Missing metadata.json → "Is this a beads project? Run 'bd init'"
- Missing dolt directory → "The beads database may not be initialized"
- Connection failures → Check server running / database corrupted
- Schema failures → "Try running 'bd init' to repair"

## Execution hints

You can use the timeout command (and should) if you want to start the TUI but guarantee a return to shell

## Modern tooling

All kinds of modern replacements for standard shell tools are available: rg, fd, sd, choose, hck
The interface is nicer for humans. You pick whatever feels right for you.

## File Editing Strategy

- **Use the Right Tool for the Job**: For any non-trivial file modifications, you **must** use the advanced editing tools provided by the MCP server.
  - **Simple Edits**: Use `sed` or `write_file` only for simple, unambiguous, single-line changes or whole-file creation.
  - **Complex Edits**: For multi-line changes, refactoring, or context-aware modifications, use `edit_file` (or equivalent diff-based tool) to minimize regression risks.

## Commit Messages

- **Conventional Commits**: All commit messages **must** adhere to the Conventional Commits specification.
  - **Format**: `<type>[optional scope]: <description>`
  - **Example**: `feat(harvester): implement reverse-scroll logic for Gemini`
  - **Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `perf`
- **Beads extra**: Add a line like "Affected ticket(s): bb-foo", can be multiple with e.g. review tickets
- **WARNING**: Forgetting the ticket reference line is a commit message format violation. Double-check before committing.

## Blunderbust-Specific Instructions

### Building Blunderbust

```bash
# Standard build (requires CGO for embedded Dolt mode)
make build

# Server mode only (no CGO required)
CGO_ENABLED=0 go build -o blunderbust ./cmd/blunderbust

# Development build with debug info
go build -gcflags="all=-N -l" -o blunderbust ./cmd/blunderbust
```

### Running Blunderbust

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

### Testing

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

### Common Development Tasks

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

### Exit Codes

When implementing CLI behavior:
- `0` - Success
- `1` - General error
- `2` - Config error (file not found, parse error, validation error)
- `3` - No tmux (running outside tmux)

### Key Files to Understand

- `cmd/blunderbust/main.go` - CLI entrypoint, flag parsing, composition root
- `internal/config/yaml.go` - YAML config loading
- `internal/config/render.go` - Template rendering for commands/prompts
- `internal/domain/template_context.go` - Available fields for templates
- `internal/discovery/models.go` - Model discovery from models.dev
- `internal/ui/` - TUI implementation with Bubbletea
- `internal/data/dolt/` - Beads/Dolt database access

### Model Discovery

Blunderbust supports dynamic model discovery from `models.dev/api.json`.

**Commands**:
- `blunderbust update-models`: Manually refresh the local model cache.

**Config Patterns**:
- `provider:<id>`: Expands to all active models for that provider.
- `discover:active`: Expands to all models from all providers that have their required environment variables set.

## Documentation

- **New Features**: When implementing new features, **must** update documentation:
  - User-facing features: Update README.md with usage examples
  - Behavioral changes: Update AGENTS.md to inform agents
  - Always keep both files in sync

