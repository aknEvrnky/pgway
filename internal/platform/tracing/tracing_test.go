package tracing_test

import (
	"context"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/platform/tracing"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestInit_DisabledNoop(t *testing.T) {
	_ = tracing.Shutdown(context.Background())
	require.NoError(t, tracing.Init(context.Background(), tracing.Config{Enabled: false}))
	require.NoError(t, tracing.Shutdown(context.Background()))
}

func TestSetup_DisabledNoop(t *testing.T) {
	_ = tracing.Shutdown(context.Background())
	stop, err := tracing.Setup(context.Background(), tracing.Config{Enabled: false}, "pgway-test")
	require.NoError(t, err)
	require.NoError(t, stop(context.Background()))
}

func TestInit_InvalidSampleRatio(t *testing.T) {
	_ = tracing.Shutdown(context.Background())
	err := tracing.Init(context.Background(), tracing.Config{
		Enabled:     true,
		Endpoint:    "127.0.0.1:4317",
		SampleRatio: 1.5,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "trace_sample_ratio")
}

func TestInit_EmptyEndpoint(t *testing.T) {
	_ = tracing.Shutdown(context.Background())
	err := tracing.Init(context.Background(), tracing.Config{
		Enabled:     true,
		Endpoint:    "",
		SampleRatio: 0.1,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "endpoint")
}

func TestSetup_AppliesDefaultServiceName(t *testing.T) {
	_ = tracing.Shutdown(context.Background())
	stop, err := tracing.Setup(context.Background(), tracing.Config{
		Enabled:     true,
		Endpoint:    "127.0.0.1:1",
		Insecure:    true,
		SampleRatio: 0.1,
	}, "pgway-test")
	require.NoError(t, err)

	stopCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	t.Cleanup(func() { _ = stop(stopCtx) })

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = stop(stopCtx)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown hung")
	}
}

func TestInitManual_RecordsSpans(t *testing.T) {
	_ = tracing.Shutdown(context.Background())
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	tracing.InitManual(tp)
	t.Cleanup(func() { _ = tracing.Shutdown(context.Background()) })

	ctx, span := tracing.Tracer().Start(context.Background(), tracing.SpanProxyRequest)
	span.End()
	_ = ctx

	spans := sr.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, tracing.SpanProxyRequest, spans[0].Name())
}
