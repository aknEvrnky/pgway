package tracing

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "pgway"

// Span names (low-cardinality instrumentation surface).
const (
	SpanProxyRequest  = "pgway.proxy.request"
	SpanProxyUpstream = "pgway.proxy.upstream"
	SpanCPRPC         = "pgway.cp.rpc"
)

// Config configures OTLP trace export.
// Enabled should be true only when both otel.enabled and otel.traces_enabled
// are set (master switch + opt-in); callers in cmd wire that conjunction.
type Config struct {
	Enabled     bool
	Endpoint    string
	Insecure    bool
	ServiceName string
	Version     string
	SampleRatio float64
}

var (
	mu       sync.RWMutex
	provider *sdktrace.TracerProvider
)

// withDefaults trims ServiceName/Version and applies fallbacks so callers
// never emit empty resource attributes.
func (c Config) withDefaults(defaultServiceName string) Config {
	c.ServiceName = strings.TrimSpace(c.ServiceName)
	if c.ServiceName == "" {
		c.ServiceName = defaultServiceName
	}
	c.Version = strings.TrimSpace(c.Version)
	if c.Version == "" {
		c.Version = "dev"
	}
	return c
}

// Init starts the TracerProvider and OTLP/gRPC exporter when enabled.
// Disabled is a no-op (global tracer stays noop/default).
func Init(ctx context.Context, cfg Config) error {
	if !cfg.Enabled {
		return nil
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return fmt.Errorf("otel.endpoint must be non-empty")
	}
	if cfg.SampleRatio < 0 || cfg.SampleRatio > 1 {
		return fmt.Errorf("otel.trace_sample_ratio must be between 0 and 1")
	}
	cfg = cfg.withDefaults("pgway")

	exporterOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
	}
	if cfg.Insecure {
		exporterOpts = append(exporterOpts, otlptracegrpc.WithInsecure())
	}
	exporter, err := otlptracegrpc.New(ctx, exporterOpts...)
	if err != nil {
		return fmt.Errorf("otlp trace exporter: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.Version),
		),
	)
	if err != nil {
		return fmt.Errorf("otel resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))),
	)

	mu.Lock()
	if provider != nil {
		_ = provider.Shutdown(ctx)
	}
	provider = tp
	mu.Unlock()
	otel.SetTracerProvider(tp)
	return nil
}

// Setup normalizes cfg, starts tracing via Init, and returns Shutdown as stop.
func Setup(ctx context.Context, cfg Config, defaultServiceName string) (stop func(context.Context) error, err error) {
	cfg = cfg.withDefaults(defaultServiceName)
	if err := Init(ctx, cfg); err != nil {
		return nil, err
	}
	return Shutdown, nil
}

// InitManual installs a test TracerProvider. Not for production.
func InitManual(tp *sdktrace.TracerProvider) {
	mu.Lock()
	provider = tp
	mu.Unlock()
	otel.SetTracerProvider(tp)
}

// Shutdown flushes and stops the TracerProvider. Safe when Init was disabled.
func Shutdown(ctx context.Context) error {
	mu.Lock()
	tp := provider
	provider = nil
	mu.Unlock()
	if tp == nil {
		return nil
	}
	return tp.Shutdown(ctx)
}

// Tracer returns the shared pgway tracer (noop-safe when tracing is off).
func Tracer() trace.Tracer {
	return otel.Tracer(tracerName)
}
