package algorithm

import (
	"sync"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lbTestPool(ids ...string) *domain.Pool {
	members := make([]domain.PoolMember, len(ids))
	proxies := make([]*domain.Proxy, len(ids))
	for i, id := range ids {
		members[i] = domain.PoolMember{ProxyId: id, Weight: 1}
		proxies[i] = &domain.Proxy{Id: id}
	}
	p := &domain.Pool{
		Id:      "pool-1",
		Type:    domain.PoolTypeStatic,
		Members: members,
	}
	p.LoadResolvedProxies(proxies)
	return p
}

func TestNewLeastBytes(t *testing.T) {
	_, err := NewLeastBytes(nil, time.Minute)
	require.ErrorIs(t, err, domain.ErrNoPool)

	empty := &domain.Pool{Id: "p", Type: domain.PoolTypeStatic}
	_, err = NewLeastBytes(empty, time.Minute)
	require.Error(t, err)

	lb, err := NewLeastBytes(lbTestPool("a", "b"), 0)
	require.NoError(t, err)
	assert.Equal(t, time.Minute, lb.resetInterval)
}

func TestLeastBytes_SelectsMinimum(t *testing.T) {
	lb, err := NewLeastBytes(lbTestPool("a", "b", "c"), time.Hour)
	require.NoError(t, err)

	lb.Release(domain.BalancerResult{ProxyId: "a", Bytes: 100})
	lb.Release(domain.BalancerResult{ProxyId: "b", Bytes: 10})
	lb.Release(domain.BalancerResult{ProxyId: "c", Bytes: 50})

	got, err := lb.Next()
	require.NoError(t, err)
	assert.Equal(t, "b", got.Id)
}

func TestLeastBytes_TieBreaksLowestIndex(t *testing.T) {
	lb, err := NewLeastBytes(lbTestPool("a", "b", "c"), time.Hour)
	require.NoError(t, err)

	got, err := lb.Next()
	require.NoError(t, err)
	assert.Equal(t, "a", got.Id)
}

func TestLeastBytes_ReleaseIgnoresNonPositiveAndUnknown(t *testing.T) {
	lb, err := NewLeastBytes(lbTestPool("a", "b"), time.Hour)
	require.NoError(t, err)

	lb.Release(domain.BalancerResult{ProxyId: "a", Bytes: 0})
	lb.Release(domain.BalancerResult{ProxyId: "a", Bytes: -5})
	lb.Release(domain.BalancerResult{ProxyId: "ghost", Bytes: 999})

	got, err := lb.Next()
	require.NoError(t, err)
	assert.Equal(t, "a", got.Id)

	lb.Release(domain.BalancerResult{ProxyId: "a", Bytes: 1})
	got, err = lb.Next()
	require.NoError(t, err)
	assert.Equal(t, "b", got.Id)
}

func TestLeastBytes_LazyReset(t *testing.T) {
	lb, err := NewLeastBytes(lbTestPool("a", "b"), time.Hour)
	require.NoError(t, err)

	lb.Release(domain.BalancerResult{ProxyId: "a", Bytes: 1000})
	lb.lastReset = time.Now().Add(-2 * time.Hour)

	got, err := lb.Next()
	require.NoError(t, err)
	assert.Equal(t, "a", got.Id)
	assert.Empty(t, lb.counters)
}

func TestLeastBytes_Concurrent(t *testing.T) {
	lb, err := NewLeastBytes(lbTestPool("a", "b", "c"), time.Hour)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = lb.Next()
		}()
		go func(i int) {
			defer wg.Done()
			ids := []string{"a", "b", "c"}
			lb.Release(domain.BalancerResult{ProxyId: ids[i%3], Bytes: int64(i + 1)})
		}(i)
	}
	wg.Wait()

	got, err := lb.Next()
	require.NoError(t, err)
	require.NotNil(t, got)
}
