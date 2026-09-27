package cmd

import (
	"github.com/aknEvrnky/pgway/internal/ports"
)

// Client is what pgctl needs from a control plane connection.
type Client interface {
	ports.ControlPlane
	ports.UserManager
	ports.AuthManager
	ports.AgentManager
	Close() error
}

// ConnectFunc dials the control plane at addr with the resolved bearer token.
type ConnectFunc func(addr, token string) (Client, error)

// Deps carries the connected client to commands. It is populated by the root
// command's PersistentPreRunE, after flags are parsed.
type Deps struct {
	Client Client
	// Token is the bearer token read from TokenPath for this invocation.
	Token string
	// TokenPath is the resolved token file; login/init write and logout
	// removes it.
	TokenPath string
}
