package interceptor

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/platform/tracing"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
)

// UnaryTracing starts a span for each unary CP RPC (outermost interceptor).
func UnaryTracing() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		ctx, span := tracing.Tracer().Start(ctx, tracing.SpanCPRPC)
		defer span.End()

		resp, err := handler(ctx, req)
		span.SetAttributes(
			attribute.String("method", shortMethod(info.FullMethod)),
			attribute.String("result", resultFromGRPCError(err)),
		)
		return resp, err
	}
}
