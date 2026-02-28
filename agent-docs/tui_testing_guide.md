# TUI Testing Guide

**MANDATORY READ BEFORE:**
- Modifying or testing TUI components (`internal/ui/`)
- Writing or debugging `agent-tui` tests
- Making visual changes that require color/focus state verification

---

## 🤖 Interacting with this TUI (For AI Agents)

This project features a Terminal User Interface (TUI). To programmatically drive, test, or interact with this application, you should use the **`agent-tui`** CLI tool.

`agent-tui` allows you to "see" the terminal screen and interact with specific UI elements reliably without guessing ANSI escape sequences.

### Core Workflow
Always follow this observe-act-verify loop:

0. **Start the daemon** `agent-tui daemon start` - It's usually running but it doesn't hurt to do it once per session
1. **Start the App:** `agent-tui run --json "<your-app-command>"` (use `--json` flag for structured output)
2. **Observe**
   - static: `agent-tui screenshot`
   - live: `agent-tui live` (gives websocket address, use `live_preview_stream` method for real-time stream)
3. **Act:**
   - Type text: `agent-tui type "my input"`
   - Send Keystrokes: `agent-tui press Enter` or `agent-tui press ArrowRight`
4. **Verify/Wait:** `agent-tui wait "Success Message"`
5. **Cleanup:** `agent-tui kill`

### Screenshot vs Live (Websocket) - Critical Difference

**`agent-tui screenshot`**: Returns plain text only (no color/ANSI codes)
- Good for: Checking text content, verifying UI state by string matching
- Bad for: Testing visual attributes like greyed-out columns, colors, styles

**`agent-tui live` (websocket)**: Returns base64-encoded ANSI escape sequences
- Good for: Testing colors, styles, greyed-out states, visual attributes
- Use method: `live_preview_stream` via websocket connection
- Decoded data contains full ANSI color codes (e.g., `\x1b[38;5;240m` for grey)

**When to use each:**
- Simple state checks → screenshot
- Color/style verification → live/websocket
- Performance testing → screenshot (faster)
- Full visual regression → live/websocket

### Concrete Usage
1. `agent-tui run -- sh -c "bd dolt start ; cd /home/sloth/Documents/projects/blunderbust/ ; ./bdb"` -- Chain multiple commands via `sh -c`
2. `agent-tui screenshot --format json` -- Get session_id + screenshot text

### TUI Automated Testing

This project has two layers of TUI tests:

**1. Unit Tests with teatest** (`internal/ui/*_test.go`)
- Uses `github.com/charmbracelet/x/exp/teatest`
- Tests keyboard navigation, state transitions, focus management
- Runs in-process without external dependencies
- Fast and reliable for regression testing

**Example teatest pattern:**
```go
func TestKeyboardNavigation(t *testing.T) {
    m := NewUIModel(app, harnesses)
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))

    // Send key
    tm.Send(tea.KeyMsg{Type: tea.KeyTab})

    // Verify state
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return strings.Contains(string(bts), "expected")
    })
}
```

**2. Integration Tests with agent-tui** (`internal/ui/agent_tui_test.go`)
- Uses real `agent-tui` + websocket streaming
- Captures ANSI color codes for visual state testing
- Tests actual binary with real PTY
- Slower but tests real rendering

**Example agent-tui pattern:**
```go
func TestVisualState(t *testing.T) {
    // Start app via agent-tui
    session := startAgentTuiSession(t, true)

    // Connect websocket for live preview
    conn := connectWebsocket(t, session.WsURL)
    sendLivePreviewRequest(t, conn, session.SessionID)

    // Capture screen with ANSI codes
    events := readLivePreviewEvents(t, conn, 5*time.Second, nil)
    screen := getScreenContent(events)

    // Verify colors (e.g., grey = 240)
    assert.True(t, containsAnsiColor(screen, 240))
}
```

**Running tests:**
```bash
# All UI tests (both teatest and agent-tui)
go test -v ./internal/ui/...

# Only fast unit tests (teatest)
go test -v ./internal/ui/... -run TestTeatest

# Only integration tests (agent-tui)
go test -v ./internal/ui/... -run TestAgentTui

# Skip slow integration tests
go test -v ./internal/ui/... -short
```
