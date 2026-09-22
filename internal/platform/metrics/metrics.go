package metrics

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// Result / protocol attribute values (low cardinality).
const (
	ResultOK       = "ok"
	ResultError    = "error"
	ResultTimeout  = "timeout"
	ResultRejected = "rejected"

	ProtocolHTTP    = "http"
	ProtocolConnect = "connect"

	DirectionIn  = "in"
	DirectionOut = "out"
)

// Config configures OTLP metrics export.
type Config struct {
	Enabled        bool
	Endpoint       string
	Insecure       bool
	ServiceName    string
	ExportInterval time.Duration
	Version        string
}

// ProxyRecord is one finished proxy attempt.
type ProxyRecord struct {
	Entrypoint string
	Protocol   string
	Result     string
	Duration   time.Duration
	BytesIn    int64
	BytesOut   int64
}

// CPRPCRecord is one finished unary control-plane RPC.
type CPRPCRecord struct {
	Method   string
	Result   string
	Duration time.Duration
}

type instruments struct {
	proxyRequests metric.Int64Counter
	proxyDuration metric.Float64Histogram
	proxyBytes    metric.Int64Counter
	proxyActive   metric.Int64UpDownCounter
	cpRPCRequests metric.Int64Counter
	cpRPCDuration metric.Float64Histogram
}

var (
	mu       sync.RWMutex
	provider *sdkmetric.MeterProvider
	instr    instruments
)

var proxyDurationBounds = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 300}

func init() {
	bindInstruments(otel.GetMeterProvider().Meter("pgway"))
}

func bindInstruments(m metric.Meter) {
	var next instruments
	next.proxyRequests, _ = m.Int64Counter(
		"pgway.proxy.requests",
		metric.WithDescription("Proxy requests finished"),
		metric.WithUnit("{request}"),
	)
	next.proxyDuration, _ = m.Float64Histogram(
		"pgway.proxy.duration",
		metric.WithDescription("Proxy request duration"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(proxyDurationBounds...),
	)
	next.proxyBytes, _ = m.Int64Counter(
		"pgway.proxy.bytes",
		metric.WithDescription("Proxy bytes transferred"),
		metric.WithUnit("By"),
	)
	next.proxyActive, _ = m.Int64UpDownCounter(
		"pgway.proxy.active",
		metric.WithDescription("Active proxy connections"),
		metric.WithUnit("{connection}"),
	)
	next.cpRPCRequests, _ = m.Int64Counter(
		"pgway.cp.rpc.requests",
		metric.WithDescription("Control-plane unary RPCs finished"),
		metric.WithUnit("{request}"),
	)
	next.cpRPCDuration, _ = m.Float64Histogram(
		"pgway.cp.rpc.duration",
		metric.WithDescription("Control-plane unary RPC duration"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(proxyDurationBounds...),
	)
	mu.Lock()
	instr = next
	mu.Unlock()
}

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

// Init starts the MeterProvider and OTLP/gRPC exporter when enabled.
// Disabled is a no-op (instruments stay on the global noop/default provider).
func Init(ctx context.Context, cfg Config) error {
	if !cfg.Enabled {
		return nil
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return fmt.Errorf("otel.endpoint must be non-empty")
	}
	if cfg.ExportInterval <= 0 {
		return fmt.Errorf("otel.export_interval must be > 0")
	}
	cfg = cfg.withDefaults("pgway")

	exporterOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(endpoint),
	}
	if cfg.Insecure {
		exporterOpts = append(exporterOpts, otlpmetricgrpc.WithInsecure())
	}
	exporter, err := otlpmetricgrpc.New(ctx, exporterOpts...)
	if err != nil {
		return fmt.Errorf("otlp metric exporter: %w", err)
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

	reader := sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(cfg.ExportInterval))
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(res),
	)

	mu.Lock()
	if provider != nil {
		_ = provider.Shutdown(ctx)
	}
	provider = mp
	mu.Unlock()
	otel.SetMeterProvider(mp)
	bindInstruments(mp.Meter("pgway"))

	// runtime.Start cannot be stopped; its goroutine survives Shutdown and later records are dropped by the shut-down provider.
	if err := runtime.Start(runtime.WithMeterProvider(mp)); err != nil {
		return fmt.Errorf("runtime metrics: %w", err)
	}
	return nil
}

// Setup normalizes cfg (service name and version fallbacks), starts metrics
// via Init, and returns a stop function that flushes via Shutdown with the
// context supplied by the caller at stop time. On Init error the returned
// stop function is nil.
func Setup(ctx context.Context, cfg Config, defaultServiceName string) (stop func(context.Context) error, err error) {
	cfg = cfg.withDefaults(defaultServiceName)
	if err := Init(ctx, cfg); err != nil {
		return nil, err
	}
	return Shutdown, nil
}

// InitManual wires instruments to a test MeterProvider (ManualReader).
// Not for production; tests call this instead of Init.
func InitManual(mp *sdkmetric.MeterProvider) {
	mu.Lock()
	provider = mp
	mu.Unlock()
	otel.SetMeterProvider(mp)
	bindInstruments(mp.Meter("pgway"))
}

// Shutdown flushes and stops the MeterProvider. Safe when Init was disabled.
func Shutdown(ctx context.Context) error {
	mu.Lock()
	mp := provider
	provider = nil
	mu.Unlock()
	if mp == nil {
		return nil
	}
	return mp.Shutdown(ctx)
}

// RecordProxy records one finished proxy attempt.
func RecordProxy(ctx context.Context, rec ProxyRecord) {
	mu.RLock()
	i := instr
	mu.RUnlock()
	if i.proxyRequests == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("entrypoint", rec.Entrypoint),
		attribute.String("result", rec.Result),
		attribute.String("protocol", rec.Protocol),
	}
	opt := metric.WithAttributes(attrs...)
	i.proxyRequests.Add(ctx, 1, opt)
	i.proxyDuration.Record(ctx, rec.Duration.Seconds(), opt)
	if rec.BytesIn > 0 {
		i.proxyBytes.Add(ctx, rec.BytesIn, metric.WithAttributes(
			attribute.String("entrypoint", rec.Entrypoint),
			attribute.String("direction", DirectionIn),
			attribute.String("protocol", rec.Protocol),
		))
	}
	if rec.BytesOut > 0 {
		i.proxyBytes.Add(ctx, rec.BytesOut, metric.WithAttributes(
			attribute.String("entrypoint", rec.Entrypoint),
			attribute.String("direction", DirectionOut),
			attribute.String("protocol", rec.Protocol),
		))
	}
}

// ActiveProxyDelta adjusts the active connection gauge (+1 / -1).
func ActiveProxyDelta(ctx context.Context, protocol string, delta int64) {
	mu.RLock()
	i := instr
	mu.RUnlock()
	if i.proxyActive == nil {
		return
	}
	i.proxyActive.Add(ctx, delta, metric.WithAttributes(
		attribute.String("protocol", protocol),
	))
}

// RecordCPRPC records one finished unary CP RPC.
func RecordCPRPC(ctx context.Context, rec CPRPCRecord) {
	mu.RLock()
	i := instr
	mu.RUnlock()
	if i.cpRPCRequests == nil {
		return
	}
	opt := metric.WithAttributes(
		attribute.String("method", rec.Method),
		attribute.String("result", rec.Result),
	)
	i.cpRPCRequests.Add(ctx, 1, opt)
	i.cpRPCDuration.Record(ctx, rec.Duration.Seconds(), opt)
}
