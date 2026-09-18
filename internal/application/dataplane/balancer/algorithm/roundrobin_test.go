package algorithm

import (
	"sync"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var rrProxies = []*domain.Proxy{
	{Id: "1", Protocol: "http", Host: "127.0.0.1", Port: 8080},
	{Id: "2", Protocol: "https", Host: "127.0.0.1", Port: 8081},
	{Id: "3", Protocol: "http", Host: "127.0.0.1", Port: 8082},
	{Id: "4", Protocol: "socks5", Host: "127.0.0.1", Port: 8083},
}

var rrTestPool = func() *domain.Pool {
	p := &domain.Pool{Title: "test pool"}
	p.LoadResolvedProxies(rrProxies)
	return p
}()

func TestNewRoundRobin(t *testing.T) {
	emptyPool := &domain.Pool{}
	emptyPool.LoadResolvedProxies(nil)

	for _, tt := range []struct {
		name        string
		pool        *domain.Pool
		expectedErr error
	}{
		{"Round-Robin non-empty pool", rrTestPool, nil},
		{"Round-Robin with empty pool", nil, domain.ErrNoPool},
		{"Round-Robin with empty proxies", emptyPool, domain.ErrNoProxy},
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

	for i, expected := range rrProxies {
		proxy, err := rr.Next()
		require.NoError(t, err)
		assert.Equal(t, expected.Id, proxy.Id, "call %d", i+1)
	}

	proxy, err := rr.Next()
	require.NoError(t, err)
	assert.Equal(t, rrProxies[0].Id, proxy.Id, "wrap-around")
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

	assert.Len(t, results, 100)
}
