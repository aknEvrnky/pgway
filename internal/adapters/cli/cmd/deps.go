package cmd

import (
	"github.com/aknEvrnky/pgway/internal/ports"
)

// Client is what pgctl needs from a control plane connection.
type Client interface {
	ports.ControlPlane
	ports.UserManager
	ports.AuthManager
	Close() error
}

// ConnectFunc dials the control plane with the resolved bearer token.
type ConnectFunc func(token string) (Client, error)

// Deps carries the connected client to commands. It is populated by the root
// command's PersistentPreRunE, after flags are parsed.
type Deps struct {
	Client Client
	// Token is the resolved bearer token used for this invocation.
	Token string
}
