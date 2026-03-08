package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/megatherium/blunderbust/internal/domain"
)

func TestRenderMainContent_MatrixState(t *testing.T) {
	cfg := MainContentConfig{
		State:        ViewStateMatrix,
		Loading:      true,
		AnimState:    AnimationState{},
		CurrentTheme: MatrixTheme,
	}

	s := RenderMainContent(cfg)
	assert.NotEmpty(t, s)
}

func TestRenderMainContent_ConfirmState(t *testing.T) {
	cfg := MainContentConfig{
		State: ViewStateConfirm,
		Selection: domain.Selection{
			Ticket:  domain.Ticket{ID: "T1", Title: "Test Ticket"},
			Harness: domain.Harness{Name: "Test Harness"},
		},
		CurrentTheme: MatrixTheme,
	}

	s := RenderMainContent(cfg)
	assert.NotEmpty(t, s)
}

func TestRenderMainContent_ErrorState(t *testing.T) {
	cfg := MainContentConfig{
		State:        ViewStateError,
		Err:          assert.AnError,
		CurrentTheme: MatrixTheme,
	}

	s := RenderMainContent(cfg)
	assert.NotEmpty(t, s)
}

func TestRenderMainContent_FilePicker(t *testing.T) {
	cfg := MainContentConfig{
		State:          ViewStateMatrix,
		ShowFilePicker: true,
		CurrentTheme:   MatrixTheme,
	}

	s := RenderMainContent(cfg)
	assert.NotEmpty(t, s)
	assert.Contains(t, s, "Add Project - Select Directory")
}

func TestRenderMainContent_AddProjectModal(t *testing.T) {
	cfg := MainContentConfig{
		State:               ViewStateMatrix,
		ShowAddProjectModal: true,
		PendingProjectPath:  "/path/to/project",
		CurrentTheme:        MatrixTheme,
	}

	s := RenderMainContent(cfg)
	assert.NotEmpty(t, s)
	assert.Contains(t, s, "Add Project?")
}

func TestRenderMainContent_AgentOutput(t *testing.T) {
	cfg := MainContentConfig{
		State:          ViewStateMatrix,
		ViewingAgentID: "agent-1",
		Agent: &RunningAgent{
			Info: &domain.AgentInfo{
				Name:       "Test Agent",
				Status:     domain.AgentRunning,
				WindowName: "test-window",
			},
		},
		CurrentTheme: MatrixTheme,
	}

	s := RenderMainContent(cfg)
	assert.NotEmpty(t, s)
	assert.Contains(t, s, "Test Agent")
}

func TestRenderModalOverlay_ShowsModal(t *testing.T) {
	cfg := MainContentConfig{
		ShowModal:    true,
		ModalContent: "Test Modal Content",
		Width:        80,
		Height:       24,
	}

	content := "Original Content"
	s := renderModalOverlay(content, cfg)

	assert.NotEmpty(t, s)
	assert.Contains(t, s, "Test Modal Content")
}

func TestRenderModalOverlay_NoModal(t *testing.T) {
	cfg := MainContentConfig{
		ShowModal:    false,
		ModalContent: "Test Modal Content",
		Width:        80,
		Height:       24,
	}

	content := "Original Content"
	s := renderModalOverlay(content, cfg)

	// When modal is not shown, content should be returned as-is
	assert.Equal(t, content, s)
}

func TestRenderModalOverlay_CenteredAndSized(t *testing.T) {
	cfg := MainContentConfig{
		ShowModal:    true,
		ModalContent: "Test Content",
		Width:        100,
		Height:       50,
	}

	content := "Background"
	s := renderModalOverlay(content, cfg)

	// Modal should be rendered (not empty)
	assert.NotEmpty(t, s)
	// Should contain the modal content
	assert.Contains(t, s, "Test Content")
}

func TestRenderWarnings_WithWarnings(t *testing.T) {
	warnings := []string{
		"Warning 1",
		"Warning 2",
	}

	content := "Original Content"
	s := renderWarnings(content, warnings)

	assert.Contains(t, s, "Original Content")
	assert.Contains(t, s, "⚠")
	assert.Contains(t, s, "Warning 1")
	assert.Contains(t, s, "Warning 2")
}

func TestRenderWarnings_NoWarnings(t *testing.T) {
	warnings := []string{}

	content := "Original Content"
	s := renderWarnings(content, warnings)

	// Content should be unchanged
	assert.Equal(t, content, s)
}

func TestRenderWarnings_EmptyWarnings(t *testing.T) {
	var warnings []string // nil slice

	content := "Original Content"
	s := renderWarnings(content, warnings)

	// Content should be unchanged
	assert.Equal(t, content, s)
}

func TestRenderWarnings_PreservesContent(t *testing.T) {
	warnings := []string{"Test Warning"}

	content := "Multi\nLine\nContent"
	s := renderWarnings(content, warnings)

	// Original content should be preserved
	assert.Contains(t, s, "Multi")
	assert.Contains(t, s, "Line")
	assert.Contains(t, s, "Content")
	// Warning should be appended
	assert.Contains(t, s, "⚠ Test Warning")
}
