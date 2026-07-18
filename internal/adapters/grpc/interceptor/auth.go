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

// exemptMethods can be called without a token. Only the two RPCs that hand
// out credentials in the first place belong here.
var exemptMethods = map[string]struct{}{
	"/pgway.controlplane.v1.AuthService/Login":     {},
	"/pgway.controlplane.v1.AuthService/InitAdmin": {},
}

func isExempt(fullMethod string) bool {
	_, ok := exemptMethods[fullMethod]
	return ok
}

// UnaryAuth validates the bearer token on every unary RPC and injects the
// authenticated principal (and raw token) into the handler context.
func UnaryAuth(authenticator ports.TokenAuthenticator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if isExempt(info.FullMethod) {
			return handler(ctx, req)
		}

		ctx, err := authenticate(ctx, authenticator)
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

		ctx, err := authenticate(ss.Context(), authenticator)
		if err != nil {
			return err
		}

		return handler(srv, &wrappedStream{ServerStream: ss, ctx: ctx})
	}
}

func authenticate(ctx context.Context, authenticator ports.TokenAuthenticator) (context.Context, error) {
	token, err := bearerToken(ctx)
	if err != nil {
		return nil, err
	}

	principal, err := authenticator.Authenticate(ctx, token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	// Agent principals stay locked out until the per-method policy map lands
	// (T6, #44): without it an authenticated agent would reach every RPC,
	// including writes. Authentication succeeded, so this is PermissionDenied,
	// not Unauthenticated.
	if principal.Kind() == domain.PrincipalKindAgent {
		return nil, status.Error(codes.PermissionDenied, "agent access not yet enabled")
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
