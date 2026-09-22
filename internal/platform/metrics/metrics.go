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
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
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

var (
	mu       sync.Mutex
	provider *sdkmetric.MeterProvider

	proxyRequests metric.Int64Counter
	proxyDuration metric.Float64Histogram
	proxyBytes    metric.Int64Counter
	proxyActive   metric.Int64UpDownCounter
	cpRPCRequests metric.Int64Counter
	cpRPCDuration metric.Float64Histogram
)

func init() {
	bindInstruments(otel.GetMeterProvider().Meter("pgway"))
}

func bindInstruments(m metric.Meter) {
	proxyRequests, _ = m.Int64Counter(
		"pgway.proxy.requests",
		metric.WithDescription("Proxy requests finished"),
		metric.WithUnit("{request}"),
	)
	proxyDuration, _ = m.Float64Histogram(
		"pgway.proxy.duration",
		metric.WithDescription("Proxy request duration"),
		metric.WithUnit("s"),
	)
	proxyBytes, _ = m.Int64Counter(
		"pgway.proxy.bytes",
		metric.WithDescription("Proxy bytes transferred"),
		metric.WithUnit("By"),
	)
	proxyActive, _ = m.Int64UpDownCounter(
		"pgway.proxy.active",
		metric.WithDescription("Active proxy connections"),
		metric.WithUnit("{connection}"),
	)
	cpRPCRequests, _ = m.Int64Counter(
		"pgway.cp.rpc.requests",
		metric.WithDescription("Control-plane unary RPCs finished"),
		metric.WithUnit("{request}"),
	)
	cpRPCDuration, _ = m.Float64Histogram(
		"pgway.cp.rpc.duration",
		metric.WithDescription("Control-plane unary RPC duration"),
		metric.WithUnit("s"),
	)
}

// Init starts the MeterProvider and OTLP/gRPC exporter when enabled.
// Disabled is a no-op (instruments stay on the global noop/default provider).
func Init(ctx context.Context, cfg Config) error {
	if !cfg.Enabled {
		return nil
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return fmt.Errorf("otel endpoint is empty")
	}
	if cfg.ExportInterval <= 0 {
		return fmt.Errorf("otel export interval must be > 0")
	}
	serviceName := strings.TrimSpace(cfg.ServiceName)
	if serviceName == "" {
		serviceName = "pgway"
	}
	version := strings.TrimSpace(cfg.Version)
	if version == "" {
		version = "dev"
	}

	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("otlp metric exporter: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
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
	defer mu.Unlock()
	if provider != nil {
		_ = provider.Shutdown(ctx)
	}
	provider = mp
	otel.SetMeterProvider(mp)
	bindInstruments(mp.Meter("pgway"))

	if err := runtime.Start(runtime.WithMeterProvider(mp)); err != nil {
		return fmt.Errorf("runtime metrics: %w", err)
	}
	return nil
}

// InitManual wires instruments to a test MeterProvider (ManualReader).
// Not for production; tests call this instead of Init.
func InitManual(mp *sdkmetric.MeterProvider) {
	mu.Lock()
	defer mu.Unlock()
	provider = mp
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
	if proxyRequests == nil {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("entrypoint", rec.Entrypoint),
		attribute.String("result", rec.Result),
		attribute.String("protocol", rec.Protocol),
	}
	opt := metric.WithAttributes(attrs...)
	proxyRequests.Add(ctx, 1, opt)
	proxyDuration.Record(ctx, rec.Duration.Seconds(), opt)
	if rec.BytesIn > 0 {
		proxyBytes.Add(ctx, rec.BytesIn, metric.WithAttributes(
			attribute.String("entrypoint", rec.Entrypoint),
			attribute.String("direction", DirectionIn),
			attribute.String("protocol", rec.Protocol),
		))
	}
	if rec.BytesOut > 0 {
		proxyBytes.Add(ctx, rec.BytesOut, metric.WithAttributes(
			attribute.String("entrypoint", rec.Entrypoint),
			attribute.String("direction", DirectionOut),
			attribute.String("protocol", rec.Protocol),
		))
	}
}

// ActiveProxyDelta adjusts the active connection gauge (+1 / -1).
func ActiveProxyDelta(ctx context.Context, protocol string, delta int64) {
	if proxyActive == nil {
		return
	}
	proxyActive.Add(ctx, delta, metric.WithAttributes(
		attribute.String("protocol", protocol),
	))
}

// RecordCPRPC records one finished unary CP RPC.
func RecordCPRPC(ctx context.Context, rec CPRPCRecord) {
	if cpRPCRequests == nil {
		return
	}
	opt := metric.WithAttributes(
		attribute.String("method", rec.Method),
		attribute.String("result", rec.Result),
	)
	cpRPCRequests.Add(ctx, 1, opt)
	cpRPCDuration.Record(ctx, rec.Duration.Seconds(), opt)
}
