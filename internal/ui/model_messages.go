package ui

import "github.com/megatherium/blunderbust/internal/domain"

type ticketsLoadedMsg []domain.Ticket

type errMsg struct{ err error }

type warningMsg struct{ err error }

type registryLoadedMsg struct{}

type launchResultMsg struct {
	res *domain.LaunchResult
	err error
}

type modalContentMsg string

type worktreesDiscoveredMsg struct {
	nodes []domain.SidebarNode
	err   error
}

// Agent-related messages
type AgentStatusMsg struct {
	AgentID string
	Status  domain.AgentStatus
}

type AgentClearedMsg struct {
	AgentID string
}

type AllStoppedAgentsClearedMsg struct{
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

func (e errMsg) Error() string { return e.err.Error() }
