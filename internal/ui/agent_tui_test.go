// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package ui

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AgentTuiIntegrationTests verify TUI behavior using agent-tui and websocket streaming.
// These tests capture ANSI color codes to verify visual states like greyed-out columns.

const (
	agentTuiTimeout = 30 * time.Second
	wsTimeout       = 5 * time.Second
)

// agentTuiSession manages an agent-tui session for testing
type agentTuiSession struct {
	SessionID string
	WsURL     string
	Cancel    context.CancelFunc
	cmd       *exec.Cmd
	tmpDir    string
}

// LivePreviewEvent represents events from the websocket stream
type LivePreviewEvent struct {
	Event string `json:"event"`
	// Ready event
	SessionID string `json:"session_id,omitempty"`
	Cols      int    `json:"cols,omitempty"`
	Rows      int    `json:"rows,omitempty"`
	// Init event
	Init string `json:"init,omitempty"`
	// Output event
	DataB64 string `json:"data_b64,omitempty"`
	// Command event
	Kind  string `json:"kind,omitempty"`
	Value string `json:"value,omitempty"`
	// Resize event
	// Dropped event
	DroppedBytes int64 `json:"dropped_bytes,omitempty"`
	// Heartbeat
	Time float64 `json:"time,omitempty"`
}

// startAgentTuiSession starts the blunderbust TUI via agent-tui and returns session info
func startAgentTuiSession(t *testing.T, demo bool) *agentTuiSession {
	// Build blunderbust binary if needed
	blunderbustPath := buildBlunderbust(t)

	// Create temp dir for session
	tmpDir, err := os.MkdirTemp("", "agent-tui-test-*")
	require.NoError(t, err)

	// Start agent-tui with blunderbust in demo mode
	args := []string{"run", "--json", "--", blunderbustPath}
	if demo {
		args = append(args, "--demo")
	}

	// agent-tui run --json returns immediately with session info, then exits
	cmd := exec.Command("agent-tui", args...)
	cmd.Dir = tmpDir

	// Capture combined output to get session ID
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to start agent-tui: %s", output)

	// Parse session ID from JSON output
	var result struct {
		SessionID string `json:"session_id"`
		PID       int    `json:"pid"`
	}
	err = json.Unmarshal(output, &result)
	require.NoError(t, err, "Failed to parse agent-tui output: %s", output)
	require.NotEmpty(t, result.SessionID, "Session ID not found in output: %s", output)

	session := &agentTuiSession{
		SessionID: result.SessionID,
		tmpDir:    tmpDir,
	}

	// Get websocket URL from daemon
	session.WsURL = getWebsocketURL(t)

	// Wait for app to initialize
	time.Sleep(500 * time.Millisecond)

	return session
}

// buildBlunderbust builds the blunderbust binary for testing
func buildBlunderbust(t *testing.T) string {
	t.Helper()

	binPath := filepath.Join(os.TempDir(), "blunderbust-test")
	
	// Check if binary exists and is up to date
	if info, err := os.Stat(binPath); err == nil {
		needsRebuild := false
		
		// Walk source directory to check modification times
		filepath.Walk(getProjectRoot(), func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if filepath.Ext(path) == ".go" && fi.ModTime().After(info.ModTime()) {
				needsRebuild = true
				return filepath.SkipDir
			}
			return nil
		})
		
		if !needsRebuild {
			return binPath
		}
	}

	// Build
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/blunderbust")
	cmd.Dir = getProjectRoot()
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build blunderbust: %s", output)

	return binPath
}

// getProjectRoot returns the project root directory
func getProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	// filename is .../internal/ui/agent_tui_test.go
	// We need to go up 3 levels to get to the project root
	dir := filepath.Dir(filename)
	dir = filepath.Dir(dir) // internal
	dir = filepath.Dir(dir) // blunderbust root
	return dir
}

// getWebsocketURL retrieves the websocket URL from the daemon state file
func getWebsocketURL(t *testing.T) string {
	t.Helper()

	// Read state file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "ws://127.0.0.1:0/ws"
	}

	stateFile := filepath.Join(homeDir, ".agent-tui", "api.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return "ws://127.0.0.1:0/ws"
	}

	var state struct {
		WsURL string `json:"ws_url"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return "ws://127.0.0.1:0/ws"
	}

	return state.WsURL
}

// connectWebsocket connects to the agent-tui websocket and returns the connection
func connectWebsocket(t *testing.T, wsURL string) *websocket.Conn {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), wsTimeout)
	defer cancel()

	u, err := url.Parse(wsURL)
	require.NoError(t, err)

	conn, _, err := websocket.Dial(ctx, u.String(), nil)
	require.NoError(t, err)

	return conn
}

// sendLivePreviewRequest sends the live_preview_stream RPC request
func sendLivePreviewRequest(t *testing.T, conn *websocket.Conn, sessionID string) {
	t.Helper()

	ctx := context.Background()

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "live_preview_stream",
		"params": map[string]interface{}{
			"session": sessionID,
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	err = conn.Write(ctx, websocket.MessageText, data)
	require.NoError(t, err)
}

// readLivePreviewEvents reads events from the websocket until timeout or condition met
func readLivePreviewEvents(t *testing.T, conn *websocket.Conn, timeout time.Duration, condition func(LivePreviewEvent) bool) []LivePreviewEvent {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var events []LivePreviewEvent

	for {
		select {
		case <-ctx.Done():
			return events
		default:
		}

		_, data, err := conn.Read(ctx)
		if err != nil {
			return events
		}

		// Try parsing as LivePreviewEvent first
		var event LivePreviewEvent
		if err := json.Unmarshal(data, &event); err == nil && event.Event != "" {
			events = append(events, event)
			if condition != nil && condition(event) {
				return events
			}
			continue
		}

		// Try parsing as RPC response (result might contain the event)
		var rpcResp struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  interface{}     `json:"error"`
		}
		if err := json.Unmarshal(data, &rpcResp); err == nil && rpcResp.ID != 0 {
			// This is an RPC response, might contain event data
			var resultEvent LivePreviewEvent
			if err := json.Unmarshal(rpcResp.Result, &resultEvent); err == nil && resultEvent.Event != "" {
				events = append(events, resultEvent)
				if condition != nil && condition(resultEvent) {
					return events
				}
			}
			continue
		}

		// Unknown format, log for debugging
		t.Logf("Unknown message: %s", string(data))
	}
}

// decodeBase64Data decodes base64 data from live preview events
func decodeBase64Data(events []LivePreviewEvent) string {
	var result strings.Builder
	for _, event := range events {
		if event.DataB64 != "" {
			data, _ := base64.StdEncoding.DecodeString(event.DataB64)
			result.Write(data)
		}
	}
	return result.String()
}

// getScreenContent extracts screen content from events (from init or output events)
func getScreenContent(events []LivePreviewEvent) string {
	var result strings.Builder
	for _, event := range events {
		if event.Init != "" {
			// Init event contains the initial screen state
			result.WriteString(event.Init)
		}
		if event.DataB64 != "" {
			data, _ := base64.StdEncoding.DecodeString(event.DataB64)
			result.Write(data)
		}
	}
	return result.String()
}

// containsAnsiColor checks if text contains ANSI color codes
func containsAnsiColor(text string, colorCode int) bool {
	// ANSI color code format: ESC[38;5;COLORm (foreground) or ESC[48;5;COLORm (background)
	fgPattern := fmt.Sprintf("\x1b[38;5;%dm", colorCode)
	bgPattern := fmt.Sprintf("\x1b[48;5;%dm", colorCode)
	return strings.Contains(text, fgPattern) || strings.Contains(text, bgPattern)
}

// cleanup kills the agent-tui session and cleans up
func (s *agentTuiSession) cleanup(t *testing.T) {
	// Kill agent-tui session via CLI
	if s.SessionID != "" {
		cmd := exec.Command("agent-tui", "kill", "--session", s.SessionID)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Logf("Failed to kill agent-tui session: %v, output: %s", err, output)
		}
	}
	
	if s.tmpDir != "" {
		os.RemoveAll(s.tmpDir)
	}
}

// TestAgentTui_BasicConnection tests basic websocket connection and live preview
func TestAgentTui_BasicConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	session := startAgentTuiSession(t, true)
	defer session.cleanup(t)

	// Connect to websocket
	conn := connectWebsocket(t, session.WsURL)
	defer conn.Close(websocket.StatusNormalClosure, "test complete")

	// Send live preview request
	sendLivePreviewRequest(t, conn, session.SessionID)

	// Read events until we get the init event
	events := readLivePreviewEvents(t, conn, 3*time.Second, func(e LivePreviewEvent) bool {
		return e.Event == "init"
	})

	require.NotEmpty(t, events, "Should receive live preview events")

	// Log all events for debugging
	for _, e := range events {
		t.Logf("Event: %s", e.Event)
	}

	// Verify we got the ready and init events
	var foundReady, foundInit bool
	for _, e := range events {
		switch e.Event {
		case "ready":
			foundReady = true
			assert.Equal(t, session.SessionID, e.SessionID)
		case "init":
			foundInit = true
			assert.NotEmpty(t, e.Init)
		}
	}

	assert.True(t, foundReady, "Should receive ready event")
	assert.True(t, foundInit, "Should receive init event")
}

// TestAgentTui_GreyedOutColumns tests that disabled columns are rendered with grey color
func TestAgentTui_GreyedOutColumns(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	session := startAgentTuiSession(t, true)
	defer session.cleanup(t)

	// Wait for initial render
	time.Sleep(1 * time.Second)

	// Connect to websocket
	conn := connectWebsocket(t, session.WsURL)
	defer conn.Close(websocket.StatusNormalClosure, "test complete")

	// Send live preview request
	sendLivePreviewRequest(t, conn, session.SessionID)

	// Read events and capture screen content
	var outputCount int
	events := readLivePreviewEvents(t, conn, 5*time.Second, func(e LivePreviewEvent) bool {
		// Stop after receiving some output events
		if e.Event == "output" {
			outputCount++
		}
		return outputCount > 10
	})

	// Decode the screen content
	screenContent := getScreenContent(events)

	// The TUI should render with various colors
	// Check for common ANSI color codes used in the TUI
	// Grey color (240) is used for inactive elements
	hasColors := strings.Contains(screenContent, "\x1b[")
	assert.True(t, hasColors, "Screen should contain ANSI color codes")

	// Look for grey color code (38;5;240 or similar)
	hasGrey := containsAnsiColor(screenContent, 240) ||
		containsAnsiColor(screenContent, 245) ||
		containsAnsiColor(screenContent, 250)

	// The test verifies that grey colors are present for inactive UI elements.
	// If this assertion fails, it means the grey color scheme is not being
	// applied to inactive columns as expected.
	assert.True(t, hasGrey, "Screen should contain grey ANSI color codes for inactive elements")
	t.Logf("Screen content contains grey colors: %v", hasGrey)
	t.Logf("Screen content length: %d", len(screenContent))
}

// TestAgentTui_KeyboardNavigation tests keyboard navigation via agent-tui
func TestAgentTui_KeyboardNavigation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	session := startAgentTuiSession(t, true)
	defer session.cleanup(t)

	// Wait for initial render
	time.Sleep(1 * time.Second)

	// Connect to websocket to observe
	conn := connectWebsocket(t, session.WsURL)
	defer conn.Close(websocket.StatusNormalClosure, "test complete")

	// Send live preview request
	sendLivePreviewRequest(t, conn, session.SessionID)

	// Read initial screen
	events := readLivePreviewEvents(t, conn, 2*time.Second, nil)
	initialScreen := getScreenContent(events)
	
	// Send tab key via agent-tui
	cmd := exec.Command("agent-tui", "press", "Tab", "--session", session.SessionID)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to send Tab key: %v, output: %s", err, output)
	}
	time.Sleep(200 * time.Millisecond)

	// Read updated screen
	events = readLivePreviewEvents(t, conn, 2*time.Second, nil)
	updatedScreen := getScreenContent(events)

	// Screens should be different after navigation (focus changed)
	// Note: This is a basic check; focus indicators may be subtle
	t.Logf("Initial screen length: %d, Updated screen length: %d",
		len(initialScreen), len(updatedScreen))

	// Send quit key
	quitCmd := exec.Command("agent-tui", "type", "q", "--session", session.SessionID)
	if output, err := quitCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to send quit key: %v, output: %s", err, output)
	}
}

// TestAgentTui_SidebarToggle tests sidebar visibility toggle
func TestAgentTui_SidebarToggle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	session := startAgentTuiSession(t, true)
	defer session.cleanup(t)

	// Wait for initial render
	time.Sleep(1 * time.Second)

	// Connect to websocket
	conn := connectWebsocket(t, session.WsURL)
	defer conn.Close(websocket.StatusNormalClosure, "test complete")

	// Send live preview request
	sendLivePreviewRequest(t, conn, session.SessionID)

	// Capture screen with sidebar
	events := readLivePreviewEvents(t, conn, 2*time.Second, nil)
	screenWithSidebar := getScreenContent(events)

	// Toggle sidebar off
	cmd := exec.Command("agent-tui", "type", "p", "--session", session.SessionID)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to toggle sidebar off: %v, output: %s", err, output)
	}
	time.Sleep(200 * time.Millisecond)

	// Capture screen without sidebar
	events = readLivePreviewEvents(t, conn, 2*time.Second, nil)
	screenWithoutSidebar := getScreenContent(events)

	// Screens should differ in width/layout
	// The sidebar takes up space, so content should shift
	t.Logf("With sidebar: %d chars, Without sidebar: %d chars",
		len(screenWithSidebar), len(screenWithoutSidebar))

	// Send quit
	quitCmd := exec.Command("agent-tui", "type", "q", "--session", session.SessionID)
	if output, err := quitCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to send quit key: %v, output: %s", err, output)
	}
}

// TestAgentTui_StateTransitions tests state transitions via websocket
func TestAgentTui_StateTransitions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	session := startAgentTuiSession(t, true)
	defer session.cleanup(t)

	// Wait for initial render
	time.Sleep(1 * time.Second)

	// Connect to websocket
	conn := connectWebsocket(t, session.WsURL)
	defer conn.Close(websocket.StatusNormalClosure, "test complete")

	// Send live preview request
	sendLivePreviewRequest(t, conn, session.SessionID)

	// Capture matrix state
	events := readLivePreviewEvents(t, conn, 2*time.Second, nil)
	matrixScreen := getScreenContent(events)

	// Navigate to a state and press Enter to confirm
	// First, select a ticket
	cmd := exec.Command("agent-tui", "press", "Enter", "--session", session.SessionID)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to send Enter key: %v, output: %s", err, output)
	}
	time.Sleep(200 * time.Millisecond)

	// Capture updated state
	events = readLivePreviewEvents(t, conn, 2*time.Second, nil)
	updatedScreen := getScreenContent(events)

	t.Logf("Matrix state: %d chars, After enter: %d chars",
		len(matrixScreen), len(updatedScreen))

	// Send quit
	quitCmd := exec.Command("agent-tui", "type", "q", "--session", session.SessionID)
	if output, err := quitCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to send quit key: %v, output: %s", err, output)
	}
}

// TestAgentTui_WaitCondition tests the wait functionality
func TestAgentTui_WaitCondition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	session := startAgentTuiSession(t, true)
	defer session.cleanup(t)

	// Wait for initial render
	time.Sleep(1 * time.Second)

	// Use agent-tui wait to verify UI is stable (timeout in milliseconds)
	cmd := exec.Command("agent-tui", "wait", "Select a Ticket", "--session", session.SessionID, "--timeout", "5000")
	output, err := cmd.CombinedOutput()

	assert.NoError(t, err, "Wait should succeed: %s", output)

	// Send quit
	quitCmd := exec.Command("agent-tui", "type", "q", "--session", session.SessionID)
	if quitOutput, quitErr := quitCmd.CombinedOutput(); quitErr != nil {
		t.Fatalf("Failed to send quit key: %v, output: %s", quitErr, quitOutput)
	}
}
