package algorithm

import (
	"sync"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func staticPool(members []domain.PoolMember, proxies []*domain.Proxy) *domain.Pool {
	p := &domain.Pool{
		Id:      "pool-1",
		Type:    domain.PoolTypeStatic,
		Members: members,
	}
	p.LoadResolvedProxies(proxies)
	return p
}

func TestNewWeightedRejectsDynamic(t *testing.T) {
	p := &domain.Pool{Id: "pool-1", Type: domain.PoolTypeDynamic}
	p.LoadResolvedProxies([]*domain.Proxy{{Id: "p1"}})
	_, err := NewWeighted(p)
	require.ErrorContains(t, err, "requires static pool")
}

func TestNewWeightedRejectsWeightLessThanOne(t *testing.T) {
	p := staticPool(
		[]domain.PoolMember{{ProxyId: "p1", Weight: 0}},
		[]*domain.Proxy{{Id: "p1"}},
	)
	_, err := NewWeighted(p)
	require.ErrorContains(t, err, "weight must be >= 1")
}

func TestWeightedDistribution(t *testing.T) {
	p := staticPool(
		[]domain.PoolMember{
			{ProxyId: "a", Weight: 5},
			{ProxyId: "b", Weight: 1},
			{ProxyId: "c", Weight: 1},
		},
		[]*domain.Proxy{{Id: "a"}, {Id: "b"}, {Id: "c"}},
	)
	w, err := NewWeighted(p)
	require.NoError(t, err)

	counts := map[string]int{}
	const n = 700
	for i := 0; i < n; i++ {
		proxy, err := w.Next()
		require.NoError(t, err)
		counts[proxy.Id]++
	}

	assert.Equal(t, 500, counts["a"])
	assert.Equal(t, 100, counts["b"])
	assert.Equal(t, 100, counts["c"])
}

func TestWeightedEqualWeights(t *testing.T) {
	p := staticPool(
		[]domain.PoolMember{
			{ProxyId: "a", Weight: 1},
			{ProxyId: "b", Weight: 1},
		},
		[]*domain.Proxy{{Id: "a"}, {Id: "b"}},
	)
	w, err := NewWeighted(p)
	require.NoError(t, err)

	counts := map[string]int{}
	for i := 0; i < 100; i++ {
		proxy, err := w.Next()
		require.NoError(t, err)
		counts[proxy.Id]++
	}
	assert.Equal(t, 50, counts["a"])
	assert.Equal(t, 50, counts["b"])
}

func TestWeightedSingleProxy(t *testing.T) {
	p := staticPool(
		[]domain.PoolMember{{ProxyId: "only", Weight: 3}},
		[]*domain.Proxy{{Id: "only"}},
	)
	w, err := NewWeighted(p)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		proxy, err := w.Next()
		require.NoError(t, err)
		assert.Equal(t, "only", proxy.Id)
	}
}

func TestWeightedConcurrentNext(t *testing.T) {
	p := staticPool(
		[]domain.PoolMember{
			{ProxyId: "a", Weight: 2},
			{ProxyId: "b", Weight: 1},
		},
		[]*domain.Proxy{{Id: "a"}, {Id: "b"}},
	)
	w, err := NewWeighted(p)
	require.NoError(t, err)

	const goroutines = 8
	const each = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < each; i++ {
				_, err := w.Next()
				assert.NoError(t, err)
			}
		}()
	}
	wg.Wait()
}
