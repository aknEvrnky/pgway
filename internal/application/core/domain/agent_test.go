package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAgentStatus_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		isValid bool
	}{
		{
			name:    "it returns true on active status",
			status:  "active",
			isValid: true,
		},
		{
			name:    "it returns true on passive status",
			status:  "passive",
			isValid: true,
		},
		{
			name:    "it returns true on disconnected status",
			status:  "disconnected",
			isValid: true,
		},
		{
			name:    "it returns false on invalid status",
			status:  "some_random",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := AgentStatus(tt.status)
			assert.Equal(t, tt.isValid, status.IsValid())
		})
	}
}

func TestAgent_Status(t *testing.T) {
	threshold := 5 * time.Minute
	now := time.Now()
	oneMinAgo := now.Add(-1 * time.Minute)
	tenMinAgo := now.Add(-10 * time.Minute)

	tests := []struct {
		name           string
		lastHeartbeat  *time.Time
		expectedStatus AgentStatus
	}{
		{
			name:           "it returns passive if no heartbeat",
			lastHeartbeat:  nil,
			expectedStatus: AgentStatusPassive,
		},
		{
			name:           "it returns disconnected if threshold passed",
			lastHeartbeat:  &tenMinAgo,
			expectedStatus: AgentStatusDisconnected,
		},
		{
			name:           "it returns active",
			lastHeartbeat:  &oneMinAgo,
			expectedStatus: AgentStatusActive,
		},
		{
			name: "it returns active on threshold",
			lastHeartbeat: func() *time.Time {
				th := now.Add(-threshold)
				return &th
			}(),
			expectedStatus: AgentStatusActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := Agent{
				LastHeartbeat: tt.lastHeartbeat,
			}

			status := agent.Status(now, threshold)
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

func TestAgent_Validate(t *testing.T) {
	tests := []struct {
		name        string
		agent       Agent
		expectedErr string
	}{
		{
			name: "it is valid if only id is filled",
			agent: Agent{
				Id: "agent-1",
			},
			expectedErr: "",
		},
		{
			name: "it is valid if all fields are filled",
			agent: Agent{
				Id:       "agent-1",
				Hostname: "boutiqe-file",
				Version:  "v0.1.2",
				Labels: map[string]string{
					"foo": "bar",
				},
				LastHeartbeat: func() *time.Time {
					hb := time.Now().Add(-1 * time.Minute)
					return &hb
				}(),
				Timestamps: Timestamps{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			expectedErr: "",
		},
		{
			name: "it is not valid if id is missing",
			agent: Agent{
				Id: "",
			},
			expectedErr: "agent ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.agent.Validate()

			if tt.expectedErr == "" {
				assert.NoError(t, err, "agent must be error-free")
				return
			}

			assert.ErrorContains(t, err, tt.expectedErr)
		})
	}
}
