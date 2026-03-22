package api

import (
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceCache_Entrypoint(t *testing.T) {
	var testCache = NewResourceCache()

	ep := &domain.Entrypoint{
		Id:       "test-ep-1",
		Title:    "main entrypoint",
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     8396,
	}

	testCache.SetEntrypoint(ep)

	cacheResult, err := testCache.GetEntrypoint("test-ep-1")

	require.NoError(t, err)
	assert.Equal(t, ep, cacheResult)

	testCache.DeleteEntrypoint("test-ep-1")
	cacheResult, err = testCache.GetEntrypoint("test-ep-1")

	require.Nil(t, cacheResult)
	assert.Error(t, err)
}

func TestResourceCache_AllEntrypoints(t *testing.T) {
	var testCache = NewResourceCache()

	ep1 := &domain.Entrypoint{
		Id:       "test-ep-1",
		Title:    "main entrypoint",
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     8396,
	}

	ep2 := &domain.Entrypoint{
		Id:       "test-ep-2",
		Title:    "backup entrypoint",
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     8803,
	}

	testCache.SetEntrypoint(ep1)
	testCache.SetEntrypoint(ep2)

	results := testCache.AllEntrypoints()

	assert.Equal(t, 2, len(results))
	assert.Contains(t, results, ep1)
	assert.Contains(t, results, ep2)

}

func TestResourceCache_Flow(t *testing.T) {
	var testCache = NewResourceCache()

	flow := &domain.Flow{
		Id:       "test-flow-1",
		RouterId: "main-router",
	}

	testCache.SetFlow(flow)

	cacheResult, err := testCache.GetFlow("test-flow-1")

	require.NoError(t, err)
	assert.Equal(t, flow, cacheResult)

	testCache.DeleteFlow("test-flow-1")
	cacheResult, err = testCache.GetFlow("test-flow-1")

	require.Nil(t, cacheResult)
	assert.Error(t, err)
}

func TestResourceCache_Router(t *testing.T) {
	var testCache = NewResourceCache()

	router := &domain.Router{
		Id:    "test-router-1",
		Title: "Main router",
	}

	testCache.SetRouter(router.Id, router)

	cacheResult, err := testCache.GetRouter("test-router-1")

	require.NoError(t, err)
	assert.Equal(t, router, cacheResult)

	testCache.DeleteRouter("test-router-1")
	cacheResult, err = testCache.GetRouter("test-router-1")

	require.Nil(t, cacheResult)
	assert.Error(t, err)
}

func TestResourceCache_Reload(t *testing.T) {
	var testCache = NewResourceCache()

	entrypoints := []*domain.Entrypoint{
		&domain.Entrypoint{
			Id:       "test-ep-1",
			Title:    "main entrypoint",
			Protocol: "http",
			Host:     "127.0.0.1",
			Port:     8396,
		},
	}

	flows := []*domain.Flow{
		&domain.Flow{
			Id:       "test-flow-1",
			RouterId: "test-router-1",
		},
	}

	routers := []*domain.Router{
		&domain.Router{
			Id:    "test-router-1",
			Title: "Main router",
		},
	}

	staleEntrypoint := &domain.Entrypoint{
		Id:       "test-ep-2",
		Title:    "secondary entrypoint",
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     8399,
	}

	testCache.SetEntrypoint(staleEntrypoint)

	testCache.Reload(entrypoints, flows, routers)

	// assert stale entrypoint is not in the cache anymore
	res, err := testCache.GetEntrypoint("test-ep-2")
	assert.Nil(t, res)
	assert.Error(t, err)

	// assert regular cache exists
	ep, err := testCache.GetEntrypoint("test-ep-1")
	require.NoError(t, err)
	f, err := testCache.GetFlow("test-flow-1")
	require.NoError(t, err)
	r, err := testCache.GetRouter("test-router-1")
	require.NoError(t, err)

	assert.Equal(t, entrypoints[0], ep)
	assert.Equal(t, flows[0], f)
	assert.Equal(t, routers[0], r)
}
