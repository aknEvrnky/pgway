package client

import (
	"context"
	"testing"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubAgentService struct {
	controlplanev1.AgentServiceClient
	registerErr error
}

func (s *stubAgentService) Register(context.Context, *controlplanev1.RegisterRequest, ...grpc.CallOption) (*controlplanev1.RegisterResponse, error) {
	return nil, s.registerErr
}

func TestRegisterMapsUnauthenticated(t *testing.T) {
	c := &Client{}
	c.agent = &stubAgentService{registerErr: status.Error(codes.Unauthenticated, "registration token rejected")}

	_, _, err := c.Register(context.Background(), "tok", domain.Agent{Id: "edge-1"})

	require.Error(t, err)
	assert.ErrorIs(t, err, ports.ErrAgentUnauthenticated)
}

func TestRegisterLeavesTransientErrorsUntouched(t *testing.T) {
	c := &Client{}
	c.agent = &stubAgentService{registerErr: status.Error(codes.Unavailable, "cp starting")}

	_, _, err := c.Register(context.Background(), "tok", domain.Agent{Id: "edge-1"})

	require.Error(t, err)
	assert.NotErrorIs(t, err, ports.ErrAgentUnauthenticated)
}
