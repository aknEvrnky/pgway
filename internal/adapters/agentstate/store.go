package agentstate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aknEvrnky/pgway/internal/ports"
)

// Store persists agent credentials on the local filesystem.
type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Load(_ context.Context) (*ports.AgentHostCredentials, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}

	var raw struct {
		AgentID    string `json:"agent_id"`
		AgentToken string `json:"agent_token"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse agent state %s: %w", s.path, err)
	}
	if raw.AgentID == "" || raw.AgentToken == "" {
		return nil, fmt.Errorf("agent state %s: agent_id and agent_token are required", s.path)
	}
	return &ports.AgentHostCredentials{AgentID: raw.AgentID, AgentToken: raw.AgentToken}, nil
}

func (s *Store) Save(_ context.Context, creds *ports.AgentHostCredentials) error {
	if creds == nil || creds.AgentID == "" || creds.AgentToken == "" {
		return fmt.Errorf("agent state: agent_id and agent_token are required")
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create agent state dir %s: %w", dir, err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("chmod agent state dir %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(map[string]string{
		"agent_id":    creds.AgentID,
		"agent_token": creds.AgentToken,
	}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write agent state temp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("commit agent state: %w", err)
	}
	return nil
}

// LockPath returns the flock path sibling to the state file.
func LockPath(statePath string) string {
	return filepath.Join(filepath.Dir(statePath), "agent.lock")
}
