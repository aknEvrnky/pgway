package server

import (
	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/adapters/grpc/interceptor"
	"github.com/aknEvrnky/pgway/internal/ports"
	"google.golang.org/grpc"
)

// New builds the control plane gRPC server: auth interceptors installed and
// every service registered. Single wiring point shared by the binaries and
// the integration test server.
func New(
	cp ports.ControlPlane,
	resolver ports.ProxyResolver,
	users ports.UserManager,
	authManager ports.AuthManager,
	authenticator ports.TokenAuthenticator,
) *grpc.Server {
	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptor.UnaryAuth(authenticator)),
		grpc.ChainStreamInterceptor(interceptor.StreamAuth(authenticator)),
	)

	RegisterControlPlane(s, NewControlPlaneServer(cp, resolver))
	RegisterAuth(s, NewAuthServer(users, authManager))

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
