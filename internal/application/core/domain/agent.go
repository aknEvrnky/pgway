package domain

import (
	"fmt"
	"time"
)

type AgentStatus string

const (
	AgentStatusActive       AgentStatus = "active"
	AgentStatusPassive      AgentStatus = "passive"
	AgentStatusDisconnected AgentStatus = "disconnected"
)

func (s AgentStatus) IsValid() bool {
	switch s {
	case AgentStatusActive, AgentStatusPassive, AgentStatusDisconnected:
		return true
	default:
		return false
	}
}

type Agent struct {
	Timestamps
	Id            string            `json:"id"`
	Hostname      string            `json:"hostname"`
	Version       string            `json:"version"`
	Labels        map[string]string `json:"labels,omitempty"`
	LastHeartbeat *time.Time        `json:"last_heartbeat,omitempty"`
}

func (a *Agent) Validate() error {
	if a.Id == "" {
		return fmt.Errorf("agent ID is required")
	}

	return nil
}

func (a *Agent) Status(now time.Time, threshold time.Duration) AgentStatus {
	if a.LastHeartbeat == nil {
		return AgentStatusPassive
	}

	if now.Sub(*a.LastHeartbeat) > threshold {
		return AgentStatusDisconnected
	}

	return AgentStatusActive
}
