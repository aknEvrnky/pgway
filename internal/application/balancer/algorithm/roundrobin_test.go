package algorithm

import (
	"sync"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var rrTestPool = &domain.Pool{
	Title: "test pool",
	Proxies: []*domain.Proxy{
		{"1", "http", "127.0.0.1", 8080, nil},
		{"2", "https", "127.0.0.1", 8081, nil},
		{"3", "http", "127.0.0.1", 8082, nil},
		{"4", "socks5", "127.0.0.1", 8083, nil},
	},
}

func TestNewRoundRobin(t *testing.T) {
	for _, tt := range []struct {
		name        string
		pool        *domain.Pool
		expectedErr error
	}{
		{"Round-Robin non-empty pool", rrTestPool, nil},
		{"Round-Robin with empty pool", nil, domain.ErrNoPool},
		{"Round-Robin with empty proxies", &domain.Pool{Proxies: nil}, domain.ErrNoProxy},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rr, err := NewRoundRobin(tt.pool)

			if tt.expectedErr == nil {
				require.NoError(t, err)

				assert.IsType(t, &RoundRobin{}, rr)
				return
			}

			require.Nil(t, rr)
			assert.ErrorIs(t, err, tt.expectedErr)
		})

	}
}

func TestRoundRobin_Next(t *testing.T) {
	rr, err := NewRoundRobin(rrTestPool)
	require.NoError(t, err)

	for i, expected := range rrTestPool.Proxies {
		proxy, err := rr.Next()
		require.NoError(t, err)
		assert.Equal(t, expected.Id, proxy.Id, "call %d", i+1)
	}

	proxy, err := rr.Next()
	require.NoError(t, err)
	assert.Equal(t, rrTestPool.Proxies[0].Id, proxy.Id, "wrap-around")
}

func TestRoundRobin_Next_Concurrent(t *testing.T) {
	rr, err := NewRoundRobin(rrTestPool)
	require.NoError(t, err)

	var wg sync.WaitGroup
	results := make(chan string, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			proxy, err := rr.Next()
			require.NoError(t, err)
			results <- proxy.Id
		}()
	}

	wg.Wait()
	close(results)

	// expect 100 results, without panic or race condition
	assert.Len(t, results, 100)
}
