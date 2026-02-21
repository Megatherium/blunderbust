# Blunderbuss

Blunderbuss is a TUI-driven launcher for AI coding harnesses, integrated with Beads for issue tracking. Select tickets, configure harnesses, and launch development sessions in organized tmux windows.

## Overview

Blunderbuss provides a streamlined workflow:
- Browse tickets from your Beads/Dolt issue database
- Choose harness configurations (which tool, model, agent to use)
- Launch development sessions in new tmux windows
- Monitor running sessions from the TUI

Think of it as a mission control for AI-assisted development work.

## Prerequisites

- **tmux** (required): All sessions are launched in tmux windows
- **Go 1.25+**: For building the binary
- **Beads project**: A project initialized with Beads and a Dolt database in `.beads/`

## Quick Start

```bash
# Clone and build
git clone https://github.com/megatherium/blunderbuss.git
cd blunderbuss
make build

# Start a tmux session (required)
tmux

# Run blunderbuss in your beads project directory
cd /path/to/your/beads/project
../blunderbuss/blunderbuss

# Use the TUI to select a ticket and launch
```

## Building

### Embedded Dolt Mode (Default)

The default mode connects to a local Dolt database stored in `.beads/dolt/`.
**This mode requires CGO** due to the github.com/dolthub/driver dependency.

```bash
make build
```

This builds the binary as `blunderbuss` in the current directory.

### Server Mode (No CGO Required)

If you only use server mode connections (remote Dolt sql-server), you can build without CGO:

```bash
CGO_ENABLED=0 go build -o blunderbuss ./cmd/blunderbuss
```

## Running

**Important**: Blunderbuss must run inside a tmux session.

```bash
# Start tmux if not already running
tmux

# Run with default config (looks for ./config.yaml)
./blunderbuss

# Run with custom config path
./blunderbuss --config /path/to/config.yaml

# Dry run mode (prints commands without executing)
./blunderbuss --dry-run

# Debug mode (verbose logging to stderr)
./blunderbuss --debug

# Demo mode (uses fake data instead of real beads database)
./blunderbuss --demo
```

## Usage Flow

1. **Select a ticket**: Browse open tickets from your Beads database
2. **Choose harness**: Select which tool to use (opencode, claude-code, etc.)
3. **Pick model**: Select the AI model to use
4. **Select agent**: Choose the agent mode (coder, task, researcher, etc.)
5. **Confirm**: Review the rendered command and prompt
6. **Launch**: A new tmux window is created with your development session

## Configuration

Configuration is loaded from a YAML file. By default, blunderbuss looks for `./config.yaml`.

Use `--config` to specify a custom path. See `config.example.yaml` for a complete example.

### Config File Structure

```yaml
harnesses:
  - name: opencode
    command_template: "opencode --model {{.Model}} --agent {{.Agent}}"
    prompt_template: "Work on ticket {{.TicketID}}: {{.TicketTitle}}\n\n{{.TicketDescription}}"
    models:
      - claude-sonnet-4-20250514
      - o3-mini
    agents:
      - coder
      - task
      - debugger
    env:
      OPENCODE_LOG_LEVEL: "info"

defaults:
  harness: opencode
  model: claude-sonnet-4-20250514
  agent: coder
```

### Template Context

Both `command_template` and `prompt_template` are rendered with Go's `text/template` syntax. Available fields:

- Ticket: `TicketID`, `TicketTitle`, `TicketDescription`, `TicketStatus`, `TicketPriority`, `TicketIssueType`, `TicketAssignee`
- Harness: `HarnessName`, `Model`, `Agent`
- Environment: `RepoPath`, `Branch`, `WorkDir`, `User`, `Hostname`
- Runtime: `DryRun`, `Debug`, `Timestamp`
- Prompt: `Prompt` (in `command_template` only - contains the rendered prompt text from `prompt_template`)

Example:
```yaml
command_template: "opencode --model {{.Model}} --agent {{.Agent}} --repo {{.RepoPath}}"
```

Using the rendered prompt in command templates:
```yaml
command_template: "ai-agent --prompt \"{{.Prompt}}\""
prompt_template: "Work on {{.TicketID}}: {{.TicketTitle}}"
```

## Beads Database Connection

Blunderbuss reads ticket data from a Beads/Dolt database. The connection mode is determined by `.beads/metadata.json`:

### Embedded Mode (Local Database)

Default when `dolt_mode` is not set to `server`:

```json
{
  "database": "dolt",
  "backend": "dolt",
  "dolt_database": "beads_bb"
}
```

Requires CGO for the embedded Dolt driver.

### Server Mode (Remote Database)

Activated when `dolt_mode: server` or server connection fields are present:

```json
{
  "database": "dolt",
  "backend": "dolt",
  "dolt_mode": "server",
  "dolt_database": "beads_fo",
  "dolt_server_host": "10.11.0.1",
  "dolt_server_port": 13307,
  "dolt_server_user": "mysql-root"
}
```

For server mode with authentication, set the password via environment variable:

```bash
export BEADS_DOLT_PASSWORD="your-password"
./blunderbuss
```

You can also override the connection using the `--dsn` flag:

```bash
./blunderbuss --dsn "user:password@tcp(host:port)/database"
```

## Command-Line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--config` | Path to config file | `./config.yaml` |
| `--beads-dir` | Path to beads directory | `./.beads` |
| `--dry-run` | Print commands without executing | `false` |
| `--debug` | Enable debug logging | `false` |
| `--demo` | Use fake data instead of real database | `false` |
| `--dsn` | DSN for Dolt server mode (overrides metadata) | - |
| `--version` | Print version and exit | - |
| `--help` | Show help message | - |

## Dry-Run Mode

Use `--dry-run` to preview what will be executed without actually launching any tmux sessions. This is useful for:

- Debugging template rendering
- Verifying config setup
- Understanding the command that will be run

In dry-run mode, the confirm screen shows a `[DRY RUN]` badge, and the result screen displays the command that would have been executed.

## Troubleshooting

### "tmux: command not found"

**Solution**: Install tmux
```bash
# Ubuntu/Debian
sudo apt-get install tmux

# macOS
brew install tmux

# Fedora/CentOS
sudo dnf install tmux
```

### "Not running inside tmux"

Blunderbuss requires tmux to create new windows. Start a tmux session first:
```bash
tmux
./blunderbuss
```

### "failed to load config: file not found"

**Solution**: Create a config file or specify the correct path
```bash
# Copy the example config
cp config.example.yaml config.yaml

# Or specify a custom path
./blunderbuss --config /path/to/config.yaml
```

### "failed to load config: parse error"

**Solution**: Validate YAML syntax
```bash
# Check for YAML syntax errors
yamllint config.yaml

# Or use Python
python3 -c "import yaml; yaml.safe_load(open('config.yaml'))"
```

### "Is this a beads project?"

Blunderbuss expects a `.beads/` directory with a Dolt database.

**Solution**: Initialize Beads in your project
```bash
bd init
```

### "The beads database may not be initialized"

The `.beads/dolt/` directory is missing.

**Solution**: Check your Beads setup
```bash
ls .beads/dolt/
# If empty or missing, try:
bd init
```

### CGO build errors

If you see "cannot load such file" or linker errors with the Dolt driver:

**Solution**: Ensure CGO is enabled and required dependencies are installed

```bash
# Install gcc (required for CGO)
# Ubuntu/Debian
sudo apt-get install build-essential

# macOS (install Xcode command line tools)
xcode-select --install

# Then build with CGO enabled
make build
```

Or build without CGO if using server mode only:
```bash
CGO_ENABLED=0 go build -o blunderbuss ./cmd/blunderbuss
```

### TUI display issues

If the TUI appears garbled or misaligned:

**Solution**: Check terminal compatibility and run in tmux
```bash
# Ensure your terminal supports 256 colors
export TERM=xterm-256color

# Run inside tmux
tmux
./blunderbuss
```

## Development

```bash
# Run linter
make lint

# Run tests
make test

# Format code
make fmt

# Run static analysis
make vet

# Clean build artifacts
make clean

# Install binary to GOPATH/bin
make install
```

## Future

Planned features:
- **quickdraw**: Instant launch with default harness/model/agent
- **blitzdraw**: Rapid-fire launching of multiple tickets
- **x-draw**: Cross-session management and monitoring

## License

GPL-3.0 License - See LICENSE file for details.
