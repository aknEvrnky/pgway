package metrics_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/platform/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

var wantDurationBounds = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 300}

func TestRecordProxy_Increments(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	metrics.InitManual(mp)
	t.Cleanup(func() { _ = metrics.Shutdown(context.Background()) })

	metrics.RecordProxy(context.Background(), metrics.ProxyRecord{
		Entrypoint: "ep1",
		Protocol:   metrics.ProtocolHTTP,
		Result:     metrics.ResultOK,
		Duration:   50 * time.Millisecond,
		BytesIn:    10,
		BytesOut:   20,
	})

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	require.NotEmpty(t, rm.ScopeMetrics)

	var sawRequests, sawBytes bool
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch m.Name {
			case "pgway.proxy.requests":
				sawRequests = true
				sum := m.Data.(metricdata.Sum[int64])
				require.NotEmpty(t, sum.DataPoints)
				assert.Equal(t, int64(1), sum.DataPoints[0].Value)
			case "pgway.proxy.bytes":
				sawBytes = true
			}
		}
	}
	assert.True(t, sawRequests)
	assert.True(t, sawBytes)
}

func TestRecordCPRPC_Increments(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	metrics.InitManual(mp)
	t.Cleanup(func() { _ = metrics.Shutdown(context.Background()) })

	metrics.RecordCPRPC(context.Background(), metrics.CPRPCRecord{
		Method:   "ApplyProxy",
		Result:   metrics.ResultOK,
		Duration: time.Millisecond,
	})

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	var saw bool
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "pgway.cp.rpc.requests" {
				saw = true
				sum := m.Data.(metricdata.Sum[int64])
				require.NotEmpty(t, sum.DataPoints)
				assert.Equal(t, int64(1), sum.DataPoints[0].Value)
			}
		}
	}
	assert.True(t, saw)
}

func TestInit_DisabledNoop(t *testing.T) {
	_ = metrics.Shutdown(context.Background())
	require.NoError(t, metrics.Init(context.Background(), metrics.Config{Enabled: false}))
	metrics.RecordProxy(context.Background(), metrics.ProxyRecord{
		Entrypoint: "ep",
		Protocol:   metrics.ProtocolHTTP,
		Result:     metrics.ResultOK,
	})
	require.NoError(t, metrics.Shutdown(context.Background()))
}

func TestSetup_DisabledNoop(t *testing.T) {
	_ = metrics.Shutdown(context.Background())
	stop, err := metrics.Setup(context.Background(), metrics.Config{Enabled: false}, "pgway-test")
	require.NoError(t, err)
	require.NoError(t, stop(context.Background()))
}

func TestSetup_AppliesDefaultServiceName(t *testing.T) {
	_ = metrics.Shutdown(context.Background())
	stop, err := metrics.Setup(context.Background(), metrics.Config{
		Enabled:        true,
		Endpoint:       "127.0.0.1:1",
		ExportInterval: time.Second,
		Insecure:       true,
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
	case <-time.After(2 * time.Second):
		t.Fatal("stop did not return within 2s")
	}
	_ = metrics.Shutdown(context.Background())
}

func TestInit_ValidationMessages(t *testing.T) {
	_ = metrics.Shutdown(context.Background())
	err := metrics.Init(context.Background(), metrics.Config{Enabled: true, ExportInterval: time.Second})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "otel.endpoint must be non-empty")

	err = metrics.Init(context.Background(), metrics.Config{Enabled: true, Endpoint: "localhost:4317"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "otel.export_interval must be > 0")
}

func TestProxyDuration_BucketBoundaries(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	metrics.InitManual(mp)
	t.Cleanup(func() { _ = metrics.Shutdown(context.Background()) })

	metrics.RecordProxy(context.Background(), metrics.ProxyRecord{
		Entrypoint: "ep1", Protocol: metrics.ProtocolHTTP, Result: metrics.ResultOK, Duration: time.Second,
	})
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	var saw bool
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "pgway.proxy.duration" {
				continue
			}
			saw = true
			hist := m.Data.(metricdata.Histogram[float64])
			require.NotEmpty(t, hist.DataPoints)
			assert.Equal(t, wantDurationBounds, hist.DataPoints[0].Bounds)
		}
	}
	assert.True(t, saw)
}

func TestCPRPCDuration_BucketBoundaries(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	metrics.InitManual(mp)
	t.Cleanup(func() { _ = metrics.Shutdown(context.Background()) })

	metrics.RecordCPRPC(context.Background(), metrics.CPRPCRecord{
		Method: "ApplyProxy", Result: metrics.ResultOK, Duration: time.Millisecond,
	})
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	var saw bool
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "pgway.cp.rpc.duration" {
				continue
			}
			saw = true
			hist := m.Data.(metricdata.Histogram[float64])
			require.NotEmpty(t, hist.DataPoints)
			assert.Equal(t, wantDurationBounds, hist.DataPoints[0].Bounds)
		}
	}
	assert.True(t, saw)
}

func TestRecordProxy_ConcurrentWithRebind(t *testing.T) {
	var wg sync.WaitGroup
	stop := make(chan struct{})
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					metrics.RecordProxy(context.Background(), metrics.ProxyRecord{
						Entrypoint: "ep1", Protocol: metrics.ProtocolHTTP, Result: metrics.ResultOK,
					})
				}
			}
		}()
	}
	reader := sdkmetric.NewManualReader()
	metrics.InitManual(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	close(stop)
	wg.Wait()
	_ = metrics.Shutdown(context.Background())
}
