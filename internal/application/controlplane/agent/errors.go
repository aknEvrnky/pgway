package agent

import "errors"

var (
	ErrAgentNotFound = errors.New("agent not found")
	ErrAgentRequired = errors.New("agent principal required")
	ErrTokenRequired = errors.New("bearer token required in call context")
)
