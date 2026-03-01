package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"

	"github.com/megatherium/blunderbust/internal/exec/tmux"

	"github.com/megatherium/blunderbust/internal/domain"
)

var (
	ThemeFooterBg = lipgloss.AdaptiveColor{Light: "62", Dark: "62"}
	ThemeFooterFg = lipgloss.AdaptiveColor{Light: "230", Dark: "230"}
	ThemeActive   = lipgloss.AdaptiveColor{Light: "205", Dark: "205"}
	ThemeInactive = lipgloss.AdaptiveColor{Light: "240", Dark: "240"}
	ThemeWarning  = lipgloss.AdaptiveColor{Light: "214", Dark: "214"}
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

const (
	footerHeight     = 1
	filterHeight     = 3
	minWindowWidth   = 60
	minWindowHeight  = 10
	sidebarBaseWidth = 25
	verticalMargins  = 2 // docStyle has Margin(1, 2) - 1 top + 1 bottom
)

type FocusColumn int

const (
	FocusSidebar FocusColumn = iota
	FocusTickets
	FocusHarness
	FocusModel
	FocusAgent
)

type ViewState int

const (
	ViewStateMatrix ViewState = iota
	ViewStateConfirm
	ViewStateError
)

type UIModel struct {
	app       *App
	state     ViewState
	focus     FocusColumn
	selection domain.Selection

	sidebar     SidebarModel
	ticketList  list.Model
	harnessList list.Model
	modelList   list.Model
	agentList   list.Model

	help help.Model
	keys KeyMap

	harnesses []domain.Harness

	width  int
	height int

	err          error
	warnings     []string
	launchResult *domain.LaunchResult
	loading      bool

	showModal    bool
	modalContent string

	showSidebar      bool
	sidebarWidth     int
	tWidth           int
	hWidth           int
	mWidth           int
	aWidth           int
	selectedWorktree string

	// Agent tracking
	agents         map[string]*RunningAgent // Keyed by agent ID
	viewingAgentID string                   // Which agent output is displayed ("" = show matrix)

	// Column disable state - set based on harness configuration
	modelColumnDisabled bool // true when harness has no models
	agentColumnDisabled bool // true when harness has no agents

	// Animation state
	animState AnimationState

	// Ticket auto-refresh tracking
	lastTicketUpdate      time.Time
	refreshedRecently    bool
	refreshAnimationFrame int
}

// RunningAgent tracks a launched agent session
type RunningAgent struct {
	Info       *domain.AgentInfo
	Capture    *tmux.OutputCapture
	LastOutput string
}
