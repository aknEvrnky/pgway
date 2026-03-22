package api

import (
	"fmt"
	"sync"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

type ResourceCache struct {
	entrypoints sync.Map
	flows       sync.Map
	routers     sync.Map
}

func NewResourceCache() *ResourceCache {
	return &ResourceCache{}
}

func (c *ResourceCache) AllEntrypoints() []*domain.Entrypoint {
	var result []*domain.Entrypoint

	c.entrypoints.Range(func(key, value any) bool {
		result = append(result, value.(*domain.Entrypoint))
		return true
	})

	return result
}

func (c *ResourceCache) GetEntrypoint(id string) (*domain.Entrypoint, error) {
	val, ok := c.entrypoints.Load(id)
	if !ok {
		return nil, fmt.Errorf("entrypoint %q not found", id)
	}

	return val.(*domain.Entrypoint), nil
}

func (c *ResourceCache) SetEntrypoint(ep *domain.Entrypoint) {
	c.entrypoints.Store(ep.Id, ep)
}

func (c *ResourceCache) DeleteEntrypoint(id string) {
	c.entrypoints.Delete(id)
}

func (c *ResourceCache) ClearEntrypoints() {
	c.entrypoints.Clear()
}

func (c *ResourceCache) GetFlow(id string) (*domain.Flow, error) {
	val, ok := c.flows.Load(id)
	if !ok {
		return nil, fmt.Errorf("flow %q not found", id)
	}

	return val.(*domain.Flow), nil
}

func (c *ResourceCache) SetFlow(f *domain.Flow) {
	c.flows.Store(f.Id, f)
}

func (c *ResourceCache) DeleteFlow(id string) {
	c.flows.Delete(id)
}

func (c *ResourceCache) ClearFlows() {
	c.flows.Clear()
}

func (c *ResourceCache) GetRouter(id string) (*domain.Router, error) {
	val, ok := c.routers.Load(id)
	if !ok {
		return nil, fmt.Errorf("router %q not found", id)
	}

	return val.(*domain.Router), nil
}

func (c *ResourceCache) SetRouter(id string, ep *domain.Router) {
	c.routers.Store(id, ep)
}

func (c *ResourceCache) DeleteRouter(id string) {
	c.routers.Delete(id)
}

func (c *ResourceCache) ClearRouters() {
	c.routers.Clear()
}

func (c *ResourceCache) Reload(entrypoints []*domain.Entrypoint, flows []*domain.Flow, routers []*domain.Router) {
	c.ClearEntrypoints()
	for _, entrypoint := range entrypoints {
		c.SetEntrypoint(entrypoint)
	}

	c.ClearFlows()
	for _, flow := range flows {
		c.SetFlow(flow)
	}

	c.ClearRouters()
	for _, router := range routers {
		c.SetRouter(router.Id, router)
	}
}
