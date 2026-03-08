package ui

import (
	"time"

	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/domain"
)

type ticketsLoadedMsg []domain.Ticket

type errMsg struct{ err error }

type warningMsg struct{ err error }

type registryLoadedMsg struct{}

type launchResultMsg struct {
	res  *domain.LaunchResult
	spec *domain.LaunchSpec
	err  error
}

type modalContentMsg string

// addProjectConfirmedMsg is emitted when user confirms adding a project.
type addProjectConfirmedMsg struct {
	path string
}

// addProjectCancelledMsg is emitted when user cancels adding a project.
type addProjectCancelledMsg struct{}

// ShowAddProjectModalMsg is emitted to show the add project confirmation modal.
type ShowAddProjectModalMsg struct {
	path string
}

type worktreesDiscoveredMsg struct {
	nodes []domain.SidebarNode
	err   error
}

type runningAgentsLoadedMsg struct {
	agents []domain.PersistedRunningAgent
	err    error
}

// Agent-related messages
type AgentStatusMsg struct {
	AgentID string
	Status  domain.AgentStatus
}

type AgentHoveredMsg struct {
	AgentID string
}

type AgentHoverEndedMsg struct{}

type AgentClearedMsg struct {
	AgentID string
}

type AllStoppedAgentsClearedMsg struct {
	ClearedIDs []string
}

// Agent tick and output messages
type agentTickMsg struct {
	agentID string
}

type agentOutputMsg struct {
	agentID string
	content string
}

// Auto-refresh messages
type ticketUpdateCheckMsg struct{}

type ticketUpdateCheckNeededMsg struct{}

type ticketsAutoRefreshedMsg struct {
	dbUpdatedAt time.Time
}

type clearRefreshIndicatorMsg struct{}

type refreshAnimationTickMsg struct{}

type serverStartedMsg struct {
	store data.TicketStore
}

func (e errMsg) Error() string { return e.err.Error() }
