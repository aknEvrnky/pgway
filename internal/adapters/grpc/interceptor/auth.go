package interceptor

import (
	"context"
	"strings"

	"github.com/aknEvrnky/pgway/internal/application/auth"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// exemptMethods can be called without a token. Only RPCs that hand out
// credentials in the first place belong here.
var exemptMethods = map[string]struct{}{
	"/pgway.controlplane.v1.AuthService/Login":     {},
	"/pgway.controlplane.v1.AuthService/InitAdmin": {},
	"/pgway.controlplane.v1.AgentService/Register": {},
}

// agentAllowedMethods is the allow-list for agent principals. Everything else
// is PermissionDenied. Users are unrestricted by this map (handler-level
// authz still applies).
var agentAllowedMethods = map[string]struct{}{
	"/pgway.controlplane.v1.AgentService/Heartbeat":  {},
	"/pgway.controlplane.v1.AgentService/Deregister": {},
	"/pgway.controlplane.v1.ChangeService/Watch":     {},

	"/pgway.controlplane.v1.ProxyService/GetProxy":            {},
	"/pgway.controlplane.v1.ProxyService/ListProxies":         {},
	"/pgway.controlplane.v1.ProxyService/GetProxiesByIds":     {},
	"/pgway.controlplane.v1.ProxyService/FindProxiesByLabels": {},

	"/pgway.controlplane.v1.PoolService/GetPool":   {},
	"/pgway.controlplane.v1.PoolService/ListPools": {},

	"/pgway.controlplane.v1.BalancerService/GetBalancer":   {},
	"/pgway.controlplane.v1.BalancerService/ListBalancers": {},

	"/pgway.controlplane.v1.RouterService/GetRouter":   {},
	"/pgway.controlplane.v1.RouterService/ListRouters": {},

	"/pgway.controlplane.v1.FlowService/GetFlow":   {},
	"/pgway.controlplane.v1.FlowService/ListFlows": {},

	"/pgway.controlplane.v1.EntrypointService/GetEntrypoint":   {},
	"/pgway.controlplane.v1.EntrypointService/ListEntrypoints": {},
}

func isExempt(fullMethod string) bool {
	_, ok := exemptMethods[fullMethod]
	return ok
}

func isAgentAllowed(fullMethod string) bool {
	_, ok := agentAllowedMethods[fullMethod]
	return ok
}

// UnaryAuth validates the bearer token on every unary RPC and injects the
// authenticated principal (and raw token) into the handler context.
func UnaryAuth(authenticator ports.TokenAuthenticator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if isExempt(info.FullMethod) {
			return handler(ctx, req)
		}

		ctx, err := authenticate(ctx, authenticator, info.FullMethod)
		if err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

// StreamAuth is the streaming counterpart of UnaryAuth.
func StreamAuth(authenticator ports.TokenAuthenticator) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if isExempt(info.FullMethod) {
			return handler(srv, ss)
		}

		ctx, err := authenticate(ss.Context(), authenticator, info.FullMethod)
		if err != nil {
			return err
		}

		return handler(srv, &wrappedStream{ServerStream: ss, ctx: ctx})
	}
}

func authenticate(ctx context.Context, authenticator ports.TokenAuthenticator, fullMethod string) (context.Context, error) {
	token, err := bearerToken(ctx)
	if err != nil {
		return nil, err
	}

	principal, err := authenticator.Authenticate(ctx, token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	if principal.Kind() == domain.PrincipalKindAgent && !isAgentAllowed(fullMethod) {
		return nil, status.Error(codes.PermissionDenied, "agent not permitted for this method")
	}

	ctx = auth.ContextWithPrincipal(ctx, principal)
	ctx = auth.ContextWithToken(ctx, token)

	return ctx, nil
}

func bearerToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization token")
	}

	token, found := strings.CutPrefix(values[0], "Bearer ")
	if !found || token == "" {
		return "", status.Error(codes.Unauthenticated, "authorization must be a bearer token")
	}

	return token, nil
}

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *wrappedStream) Context() context.Context {
	return s.ctx
}
