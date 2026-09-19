package server

import (
	"context"
	"time"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/adapters/grpc/interceptor"
	"github.com/aknEvrnky/pgway/internal/ports"
	"google.golang.org/grpc"
)

// AgentServerConfig carries TTLs used only at the transport edge (status
// derivation, default registration TTL, Register response expiry).
type AgentServerConfig struct {
	HeartbeatThreshold   time.Duration
	AgentTokenTTL        time.Duration
	RegistrationTokenTTL time.Duration
}

// New builds the control plane gRPC server: auth and rate-limit interceptors
// installed and every service registered. Single wiring point shared by the
// binaries and the integration test server.
//
// shutdown is canceled when the process is stopping; ChangeService.Watch
// streams exit so GracefulStop can finish.
func New(
	cp ports.ControlPlane,
	resolver ports.ProxyResolver,
	users ports.UserManager,
	authManager ports.AuthManager,
	authenticator ports.TokenAuthenticator,
	agents ports.AgentManager,
	events ports.EventSubscriberPort,
	shutdown context.Context,
	agentCfg AgentServerConfig,
	ka KeepaliveConfig,
	rl RateLimitConfig,
) *grpc.Server {
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			interceptor.UnaryAuth(authenticator),
			interceptor.UnaryRateLimit(rl.toInterceptor()),
		),
		grpc.ChainStreamInterceptor(interceptor.StreamAuth(authenticator)),
	}
	opts = append(opts, keepaliveServerOptions(ka)...)

	s := grpc.NewServer(opts...)

	RegisterControlPlane(s, NewControlPlaneServer(cp, resolver))
	RegisterAuth(s, NewAuthServer(users, authManager))
	RegisterAgent(s, NewAgentServer(
		agents,
		agentCfg.HeartbeatThreshold,
		agentCfg.AgentTokenTTL,
		agentCfg.RegistrationTokenTTL,
	))
	if events != nil {
		RegisterChange(s, NewChangeServer(events, shutdown))
	}

	return s
}

func RegisterControlPlane(s grpc.ServiceRegistrar, srv *ControlPlaneServer) {
	controlplanev1.RegisterProxyServiceServer(s, srv)
	controlplanev1.RegisterPoolServiceServer(s, srv)
	controlplanev1.RegisterBalancerServiceServer(s, srv)
	controlplanev1.RegisterRouterServiceServer(s, srv)
	controlplanev1.RegisterFlowServiceServer(s, srv)
	controlplanev1.RegisterEntrypointServiceServer(s, srv)
}

func RegisterAuth(s grpc.ServiceRegistrar, srv *AuthServer) {
	controlplanev1.RegisterUserServiceServer(s, srv)
	controlplanev1.RegisterAuthServiceServer(s, srv)
}

func RegisterAgent(s grpc.ServiceRegistrar, srv *AgentServer) {
	controlplanev1.RegisterAgentServiceServer(s, srv)
}
