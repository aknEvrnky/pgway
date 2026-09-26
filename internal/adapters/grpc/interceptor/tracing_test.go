package interceptor_test

import (
	"context"
	"testing"

	"github.com/aknEvrnky/pgway/internal/adapters/grpc/interceptor"
	"github.com/aknEvrnky/pgway/internal/platform/metrics"
	"github.com/aknEvrnky/pgway/internal/platform/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryTracing_RecordsOK(t *testing.T) {
	_ = tracing.Shutdown(context.Background())
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracing.InitManual(tp)
	t.Cleanup(func() { _ = tracing.Shutdown(context.Background()) })

	ix := interceptor.UnaryTracing()
	_, err := ix(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: "/pgway.controlplane.v1.ProxyService/ApplyProxy",
	}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	require.NoError(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, tracing.SpanCPRPC, spans[0].Name())

	attrs := map[string]string{}
	for _, kv := range spans[0].Attributes() {
		attrs[string(kv.Key)] = kv.Value.AsString()
	}
	assert.Equal(t, "ApplyProxy", attrs["method"])
	assert.Equal(t, metrics.ResultOK, attrs["result"])
	_, hasURL := attrs["url"]
	assert.False(t, hasURL)
}

func TestUnaryTracing_RecordsError(t *testing.T) {
	_ = tracing.Shutdown(context.Background())
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracing.InitManual(tp)
	t.Cleanup(func() { _ = tracing.Shutdown(context.Background()) })

	ix := interceptor.UnaryTracing()
	_, err := ix(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: "/pgway.controlplane.v1.ProxyService/ApplyProxy",
	}, func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.Internal, "boom")
	})
	require.Error(t, err)

	spans := sr.Ended()
	require.Len(t, spans, 1)
	attrs := map[string]string{}
	for _, kv := range spans[0].Attributes() {
		attrs[string(kv.Key)] = kv.Value.AsString()
	}
	assert.Equal(t, metrics.ResultError, attrs["result"])
}
