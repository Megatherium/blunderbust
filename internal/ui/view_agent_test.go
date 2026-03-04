package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/megatherium/blunderbust/internal/domain"
)

func TestRenderAgentOutput_AgentNotFound(t *testing.T) {
	cfg := AgentConfig{
		Agent:  nil,
		Width:  80,
		Height: 24,
		Theme:  MatrixTheme,
	}

	s := RenderAgentOutput(cfg)
	assert.Contains(t, s, "Agent not found")
}

func TestRenderAgentOutput_AgentRunning(t *testing.T) {
	agent := &RunningAgent{
		Info: &domain.AgentInfo{
			Name:       "test-agent",
			Status:     domain.AgentRunning,
			WindowName: "test-window",
		},
		LastOutput: "Running output...",
	}

	cfg := AgentConfig{
		Agent:  agent,
		Width:  80,
		Height: 24,
		Theme:  MatrixTheme,
	}

	s := RenderAgentOutput(cfg)
	assert.Contains(t, s, "test-agent")
	assert.Contains(t, s, "Running")
	assert.Contains(t, s, "test-window")
	assert.Contains(t, s, "Running output...")
}

func TestRenderAgentOutput_AgentCompleted(t *testing.T) {
	agent := &RunningAgent{
		Info: &domain.AgentInfo{
			Name:       "test-agent",
			Status:     domain.AgentCompleted,
			WindowName: "test-window",
		},
		LastOutput: "",
	}

	cfg := AgentConfig{
		Agent:  agent,
		Width:  80,
		Height: 24,
		Theme:  MatrixTheme,
	}

	s := RenderAgentOutput(cfg)
	assert.Contains(t, s, "test-agent")
	assert.Contains(t, s, "Completed")
	assert.Contains(t, s, "No output available")
}

func TestRenderAgentOutput_AgentFailed(t *testing.T) {
	agent := &RunningAgent{
		Info: &domain.AgentInfo{
			Name:       "test-agent",
			Status:     domain.AgentFailed,
			WindowName: "test-window",
		},
		LastOutput: "Error occurred",
	}

	cfg := AgentConfig{
		Agent:  agent,
		Width:  80,
		Height: 24,
		Theme:  MatrixTheme,
	}

	s := RenderAgentOutput(cfg)
	assert.Contains(t, s, "test-agent")
	assert.Contains(t, s, "Failed")
	assert.Contains(t, s, "Error occurred")
}

func TestRenderAgentOutput_NoOutput(t *testing.T) {
	agent := &RunningAgent{
		Info: &domain.AgentInfo{
			Name:       "test-agent",
			Status:     domain.AgentCompleted,
			WindowName: "test-window",
		},
		LastOutput: "",
	}

	cfg := AgentConfig{
		Agent:  agent,
		Width:  80,
		Height: 24,
		Theme:  MatrixTheme,
	}

	s := RenderAgentOutput(cfg)
	assert.Contains(t, s, "No output available")
}

func TestRenderAgentOutput_ReturnToMatrixHint(t *testing.T) {
	agent := &RunningAgent{
		Info: &domain.AgentInfo{
			Name:       "test-agent",
			Status:     domain.AgentRunning,
			WindowName: "test-window",
		},
		LastOutput: "",
	}

	cfg := AgentConfig{
		Agent:  agent,
		Width:  80,
		Height: 24,
		Theme:  MatrixTheme,
	}

	s := RenderAgentOutput(cfg)
	assert.Contains(t, s, "Press Enter to return to matrix")
}

func TestGetAgentStatus_Running(t *testing.T) {
	statusStr, statusColor := getAgentStatus(domain.AgentRunning)
	assert.Equal(t, "Running", statusStr)
	assert.NotNil(t, statusColor)
}

func TestGetAgentStatus_Completed(t *testing.T) {
	statusStr, statusColor := getAgentStatus(domain.AgentCompleted)
	assert.Equal(t, "Completed", statusStr)
	assert.NotNil(t, statusColor)
}

func TestGetAgentStatus_Failed(t *testing.T) {
	statusStr, statusColor := getAgentStatus(domain.AgentFailed)
	assert.Equal(t, "Failed", statusStr)
	assert.NotNil(t, statusColor)
}

func TestGetAgentStatus_Unknown(t *testing.T) {
	statusStr, statusColor := getAgentStatus(domain.AgentStatus(999))
	assert.Equal(t, "Unknown", statusStr)
	assert.NotNil(t, statusColor)
}

func TestGetAgentOutputContent_WithLastOutput(t *testing.T) {
	agent := &RunningAgent{
		Info: &domain.AgentInfo{
			Status: domain.AgentCompleted,
		},
		LastOutput: "some output",
	}

	content := getAgentOutputContent(agent)
	assert.Equal(t, "some output", content)
}

func TestGetAgentOutputContent_RunningWaiting(t *testing.T) {
	agent := &RunningAgent{
		Info: &domain.AgentInfo{
			Status: domain.AgentRunning,
		},
		LastOutput: "",
	}

	content := getAgentOutputContent(agent)
	assert.Equal(t, "Waiting for output...", content)
}

func TestGetAgentOutputContent_NoOutput(t *testing.T) {
	agent := &RunningAgent{
		Info: &domain.AgentInfo{
			Status: domain.AgentCompleted,
		},
		LastOutput: "",
	}

	content := getAgentOutputContent(agent)
	assert.Equal(t, "No output available", content)
}
