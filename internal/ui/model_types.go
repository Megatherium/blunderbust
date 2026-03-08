package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"

	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/megatherium/blunderbust/internal/exec/tmux"
	"github.com/megatherium/blunderbust/internal/ui/filepicker"
)

var (
	ThemeFooterBg = lipgloss.AdaptiveColor{Light: "62", Dark: "62"}
	ThemeFooterFg = lipgloss.AdaptiveColor{Light: "230", Dark: "230"}
	ThemeActive   = lipgloss.AdaptiveColor{Light: "205", Dark: "205"}
	ThemeInactive = lipgloss.AdaptiveColor{Light: "240", Dark: "240"}
	ThemeWarning  = lipgloss.AdaptiveColor{Light: "214", Dark: "214"}
)

var docStyle = lipgloss.NewStyle().Padding(1, 2)

const (
	footerHeight     = 1
	filterHeight     = 3
	statusBarHeight  = 1 // bubbles list status bar consumes 1 line
	borderHeight     = 2 // thick border consumes 2 lines (top + bottom)
	minWindowWidth   = 60
	minWindowHeight  = 10
	sidebarBaseWidth = 25
	verticalMargins  = 2 // docStyle has Margin(1, 2) - 1 top + 1 bottom

	// Auto-refresh constants
	ticketPollingInterval    = 3 * time.Second
	refreshIndicatorDuration = 3 * time.Second
	animationTickInterval    = 500 * time.Millisecond
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
	ViewStateLoading
	ViewStateFilePicker
	ViewStateAddProjectModal
	ViewStateAgentOutput
	ViewStateConfirm
	ViewStateError
)

// UIModel represents the complete state of the TUI application.
//
// View State Machine (ViewState enum):
//
//   - ViewStateLoading: Loading animation (initial startup)
//   - ViewStateFilePicker: File picker overlay for adding projects
//   - ViewStateAddProjectModal: "Add project?" confirmation modal
//   - ViewStateAgentOutput: Agent output view (viewingAgentID identifies which)
//   - ViewStateMatrix: Main matrix view (ticket/harness/model/agent columns)
//   - ViewStateConfirm: Launch confirmation
//   - ViewStateError: Error display with retry options
//
// Note: showModal is a separate overlay system used for error/info messages
// and is composited on top of the main content.
//
// Valid State Transitions (Add Project flow):
//
//	Sidebar (focus=FocusSidebar) + 'a' key → OpenFilePickerMsg
//	→ state = ViewStateFilePicker
//	→ File picker active, 'a' to select dir, 'esc' to cancel
//	→ Select dir with .beads → ShowAddProjectModalMsg
//	→ state = ViewStateAddProjectModal
//	→ 'y' or Enter → addProjectConfirmedMsg → project added
//	→ 'n' or Esc → addProjectCancelledMsg → back to file picker
//
// Valid State Transitions (Agent view):
//
//	Matrix view + select agent in sidebar → state = ViewStateAgentOutput
//	→ Agent output view
//	→ Enter or Back → state = ViewStateMatrix → back to matrix
//
// Column Disable Logic:
//
//	modelColumnDisabled = true when harness has no models
//	agentColumnDisabled = true when harness has no agents
//	These are set by handleModelSkip() and handleAgentSkip()
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

	layout LayoutDimensions

	err          error
	warnings     []string
	launchResult *domain.LaunchResult

	showModal    bool
	modalContent string

	showSidebar      bool
	selectedWorktree string

	// Agent tracking
	agents         map[string]*RunningAgent // Keyed by agent ID
	viewingAgentID string                   // Which agent output is displayed ("" = show matrix)
	hoveredAgentID string                   // Agent currently hovered in sidebar ("" = no hover)

	// Column disable state - set based on harness configuration
	modelColumnDisabled bool // true when harness has no models
	agentColumnDisabled bool // true when harness has no agents

	// Caches for list views to avoid re-rendering on every tick
	cachesDirty      bool // true when caches need rebuilding
	ticketViewCache  string
	harnessViewCache string
	modelViewCache   string
	agentViewCache   string

	// Animation state
	animState AnimationState

	// Current theme for visual styling
	currentTheme *ThemePalette

	// Ticket auto-refresh tracking
	lastTicketUpdate      time.Time
	refreshedRecently     bool
	refreshAnimationFrame int

	// Error recovery
	retryStore data.TicketStore // Store preserved for retry/start operations after errors

	// File picker for adding projects
	filepicker         filepicker.Model
	pendingProjectPath string
}

// RunningAgent tracks a launched agent session
type RunningAgent struct {
	Info       *domain.AgentInfo
	Capture    *tmux.OutputCapture
	LastOutput string
}
