package ui

import (
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestRenderMatrix_SmallHeightGuard(t *testing.T) {
	cfg := MatrixConfig{
		Width:  80,
		Height: 4, // filterHeight is 3, so guard triggers for height < 5
	}

	s := RenderMatrix(cfg)
	assert.Equal(t, "Initializing...", s)

	cfg.Height = 5
	s = RenderMatrix(cfg)
	assert.NotEqual(t, "Initializing...", s)
}

func TestRenderMatrix_ColumnWithFocus(t *testing.T) {
	cfg := MatrixConfig{
		Width:        100,
		Height:       20,
		ShowSidebar:  false,
		TWidth:       20,
		HWidth:       20,
		MWidth:       20,
		AWidth:       20,
		Focus:        FocusTickets,
		AnimState:    AnimationState{StartTime: time.Now()},
		Theme:        MatrixTheme,
		TicketView:   "ticket1\nticket2",
		HarnessView:  "harness1",
		ModelView:    "model1",
		AgentView:    "agent1",
		TicketTitle:  "Tickets",
		HarnessTitle: "Harnesses",
		ModelTitle:   "Models",
		AgentTitle:   "Agents",
	}

	s := RenderMatrix(cfg)

	// Should contain the focus indicator ▶ for focused column
	assert.Contains(t, s, "▶")
	// Should contain column titles
	assert.Contains(t, s, "Tickets")
	assert.Contains(t, s, "Harnesses")
	assert.Contains(t, s, "Models")
	assert.Contains(t, s, "Agents")
}

func TestRenderMatrix_ColumnWithoutFocus(t *testing.T) {
	cfg := MatrixConfig{
		Width:        100,
		Height:       20,
		ShowSidebar:  false,
		TWidth:       20,
		HWidth:       20,
		MWidth:       20,
		AWidth:       20,
		Focus:        FocusHarness, // Harness is focused, not tickets
		AnimState:    AnimationState{StartTime: time.Now()},
		Theme:        MatrixTheme,
		TicketView:   "ticket1",
		HarnessView:  "harness1",
		ModelView:    "model1",
		AgentView:    "agent1",
		TicketTitle:  "Tickets",
		HarnessTitle: "Harnesses",
		ModelTitle:   "Models",
		AgentTitle:   "Agents",
	}

	s := RenderMatrix(cfg)

	// Tickets column should NOT have focus indicator, Harness should
	// We check by looking at the structure - non-focused items use "  " prefix
	// This is harder to test directly, but we verify it renders without error
	assert.NotEmpty(t, s)
}

func TestRenderMatrix_ModelColumnDisabled(t *testing.T) {
	cfg := MatrixConfig{
		Width:               100,
		Height:              20,
		ShowSidebar:         false,
		TWidth:              20,
		HWidth:              20,
		MWidth:              20,
		AWidth:              20,
		ModelColumnDisabled: true,
		Focus:               FocusTickets,
		AnimState:           AnimationState{StartTime: time.Now()},
		Theme:               MatrixTheme,
		TicketView:          "ticket1",
		HarnessView:         "harness1",
		ModelView:           "model1",
		AgentView:           "agent1",
		TicketTitle:         "Tickets",
		HarnessTitle:        "Harnesses",
		ModelTitle:          "Models",
		AgentTitle:          "Agents",
	}

	s := RenderMatrix(cfg)

	// Disabled model column should show N/A message
	assert.Contains(t, s, "N/A")
	assert.Contains(t, s, "No models")
}

func TestRenderMatrix_AgentColumnDisabled(t *testing.T) {
	cfg := MatrixConfig{
		Width:               100,
		Height:              20,
		ShowSidebar:         false,
		TWidth:              20,
		HWidth:              20,
		MWidth:              20,
		AWidth:              20,
		AgentColumnDisabled: true,
		Focus:               FocusTickets,
		AnimState:           AnimationState{StartTime: time.Now()},
		Theme:               MatrixTheme,
		TicketView:          "ticket1",
		HarnessView:         "harness1",
		ModelView:           "model1",
		AgentView:           "agent1",
		TicketTitle:         "Tickets",
		HarnessTitle:        "Harnesses",
		ModelTitle:          "Models",
		AgentTitle:          "Agents",
	}

	s := RenderMatrix(cfg)

	// Disabled agent column should show N/A message
	assert.Contains(t, s, "N/A")
	assert.Contains(t, s, "No agents")
}

func TestRenderMatrix_WithSidebar(t *testing.T) {
	cfg := MatrixConfig{
		Width:        120,
		Height:       20,
		ShowSidebar:  true,
		SidebarWidth: 25,
		TWidth:       20,
		HWidth:       15,
		MWidth:       20,
		AWidth:       20,
		Focus:        FocusSidebar,
		AnimState:    AnimationState{StartTime: time.Now()},
		Theme:        MatrixTheme,
		TicketView:   "ticket1",
		HarnessView:  "harness1",
		ModelView:    "model1",
		AgentView:    "agent1",
		SidebarView:  "sidebar content",
		TicketTitle:  "Tickets",
		HarnessTitle: "Harnesses",
		ModelTitle:   "Models",
		AgentTitle:   "Agents",
	}

	s := RenderMatrix(cfg)

	// Should render with sidebar
	assert.NotEmpty(t, s)
	assert.Contains(t, s, "sidebar content")
}

func TestRenderMatrix_WithoutSidebar(t *testing.T) {
	cfg := MatrixConfig{
		Width:        100,
		Height:       20,
		ShowSidebar:  false,
		TWidth:       20,
		HWidth:       20,
		MWidth:       20,
		AWidth:       20,
		Focus:        FocusTickets,
		AnimState:    AnimationState{StartTime: time.Now()},
		Theme:        MatrixTheme,
		TicketView:   "ticket1",
		HarnessView:  "harness1",
		ModelView:    "model1",
		AgentView:    "agent1",
		SidebarView:  "sidebar content",
		TicketTitle:  "Tickets",
		HarnessTitle: "Harnesses",
		ModelTitle:   "Models",
		AgentTitle:   "Agents",
	}

	s := RenderMatrix(cfg)

	// Should render without sidebar content visible
	assert.NotEmpty(t, s)
	// Sidebar view content should NOT appear when sidebar is disabled
	assert.NotContains(t, s, "sidebar content")
}

func TestRenderMatrix_FilterBox(t *testing.T) {
	cfg := MatrixConfig{
		Width:        100,
		Height:       20,
		ShowSidebar:  false,
		TWidth:       20,
		HWidth:       20,
		MWidth:       20,
		AWidth:       20,
		Focus:        FocusTickets,
		AnimState:    AnimationState{StartTime: time.Now()},
		Theme:        MatrixTheme,
		TicketView:   "ticket1",
		HarnessView:  "harness1",
		ModelView:    "model1",
		AgentView:    "agent1",
		TicketTitle:  "Tickets",
		HarnessTitle: "Harnesses",
		ModelTitle:   "Models",
		AgentTitle:   "Agents",
	}

	s := RenderMatrix(cfg)

	// Should contain filter box
	assert.Contains(t, s, "Filters:")
	assert.Contains(t, s, "Press / to search")
}

func TestGetActiveColor_WithFlash(t *testing.T) {
	animState := AnimationState{
		LockInActive:    true,
		LockInTarget:    FocusTickets,
		LockInIntensity: 0.8, // Above threshold
	}
	theme := MatrixTheme

	color := getActiveColor(animState, FocusTickets, theme)
	assert.Equal(t, FlashColor, color)
}

func TestGetActiveColor_WithoutFlash(t *testing.T) {
	animState := AnimationState{
		LockInActive:    false,
		PulsePhase:      0.5,
		ColorCycleIndex: 0,
	}
	theme := MatrixTheme

	color := getActiveColor(animState, FocusTickets, theme)
	// Should return a cycling color, not FlashColor
	assert.NotEqual(t, FlashColor, color)
}

func TestCreateActiveBorder(t *testing.T) {
	listHeight := 20
	activeColor := lipgloss.Color("#ff0000")
	glowColor := lipgloss.Color("#00ff00")

	borderFunc := createActiveBorder(listHeight, activeColor, glowColor)
	style := borderFunc(10)

	// Should return a valid style
	assert.NotNil(t, style)
}

func TestCreateInactiveBorder(t *testing.T) {
	listHeight := 20

	borderFunc := createInactiveBorder(listHeight)
	style := borderFunc(10)

	// Should return a valid style
	assert.NotNil(t, style)
}

func TestRenderMatrixColumn_Focused(t *testing.T) {
	activeBorder := createActiveBorder(20, lipgloss.Color("#ff0000"), lipgloss.Color("#00ff00"))
	inactiveBorder := createInactiveBorder(20)
	capView := func(view string, w int) string { return view }
	faintCapView := func(view string, w int) string { return view }
	theme := MatrixTheme

	s := renderMatrixColumn(
		"item1\nitem2",
		20,
		true, // focused
		"Column Title",
		theme,
		activeBorder,
		inactiveBorder,
		capView,
		faintCapView,
	)

	assert.NotEmpty(t, s)
	// Should contain focus indicator and title
	assert.Contains(t, s, "Column Title")
}

func TestRenderMatrixColumn_NotFocused(t *testing.T) {
	activeBorder := createActiveBorder(20, lipgloss.Color("#ff0000"), lipgloss.Color("#00ff00"))
	inactiveBorder := createInactiveBorder(20)
	capView := func(view string, w int) string { return view }
	faintCapView := func(view string, w int) string { return view }
	theme := MatrixTheme

	s := renderMatrixColumn(
		"item1\nitem2",
		20,
		false, // not focused
		"Column Title",
		theme,
		activeBorder,
		inactiveBorder,
		capView,
		faintCapView,
	)

	assert.NotEmpty(t, s)
	assert.Contains(t, s, "Column Title")
}

func TestApplySidebarBorder(t *testing.T) {
	cfg := MatrixConfig{
		Width:        120,
		Height:       20,
		SidebarWidth: 25,
		Focus:        FocusSidebar,
		SidebarView:  "sidebar content",
	}
	activeColor := lipgloss.Color("#ff0000")
	rightPanelBox := "right panel"

	s := applySidebarBorder(cfg, rightPanelBox, activeColor)

	assert.NotEmpty(t, s)
	assert.Contains(t, s, "sidebar content")
}
