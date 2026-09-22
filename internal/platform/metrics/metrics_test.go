package metrics_test

import (
	"context"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/platform/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

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
