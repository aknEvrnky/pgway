package agentruntime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// State holds CP-issued credentials persisted on the DP host.
type State struct {
	AgentID    string `json:"agent_id"`
	AgentToken string `json:"agent_token"`
}

// LoadState reads agent credentials from path. Returns os.ErrNotExist if missing.
func LoadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse agent state %s: %w", path, err)
	}
	if s.AgentID == "" || s.AgentToken == "" {
		return nil, fmt.Errorf("agent state %s: agent_id and agent_token are required", path)
	}
	return &s, nil
}

// SaveState writes credentials with dir 0700 and file 0600.
func SaveState(path string, s *State) error {
	if s == nil || s.AgentID == "" || s.AgentToken == "" {
		return fmt.Errorf("agent state: agent_id and agent_token are required")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create agent state dir %s: %w", dir, err)
	}
	// MkdirAll does not tighten perms on an existing directory.
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("chmod agent state dir %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write agent state temp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("commit agent state: %w", err)
	}
	return nil
}

// LockPath returns the flock path sibling to the state file.
func LockPath(statePath string) string {
	return filepath.Join(filepath.Dir(statePath), "agent.lock")
}
