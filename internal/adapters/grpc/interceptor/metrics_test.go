package interceptor_test

import (
	"context"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/adapters/grpc/interceptor"
	"github.com/aknEvrnky/pgway/internal/platform/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryMetrics_RecordsOK(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	metrics.InitManual(mp)
	t.Cleanup(func() { _ = metrics.Shutdown(context.Background()) })

	ix := interceptor.UnaryMetrics()
	_, err := ix(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: "/pgway.controlplane.v1.ProxyService/ApplyProxy",
	}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	var saw bool
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "pgway.cp.rpc.requests" {
				continue
			}
			saw = true
			sum := m.Data.(metricdata.Sum[int64])
			require.NotEmpty(t, sum.DataPoints)
			assert.Equal(t, int64(1), sum.DataPoints[0].Value)
		}
	}
	assert.True(t, saw)
}

func TestUnaryMetrics_RecordsError(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	metrics.InitManual(mp)
	t.Cleanup(func() { _ = metrics.Shutdown(context.Background()) })

	ix := interceptor.UnaryMetrics()
	_, err := ix(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: "/pgway.controlplane.v1.ProxyService/ApplyProxy",
	}, func(ctx context.Context, req any) (any, error) {
		time.Sleep(time.Millisecond)
		return nil, status.Error(codes.Internal, "boom")
	})
	require.Error(t, err)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "pgway.cp.rpc.requests" {
				continue
			}
			sum := m.Data.(metricdata.Sum[int64])
			for _, dp := range sum.DataPoints {
				for _, kv := range dp.Attributes.ToSlice() {
					if string(kv.Key) == "result" && kv.Value.AsString() == metrics.ResultError {
						found = true
					}
				}
			}
		}
	}
	assert.True(t, found)
}
