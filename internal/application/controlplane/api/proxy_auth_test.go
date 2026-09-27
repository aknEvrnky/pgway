package api

import (
	"context"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/aknEvrnky/pgway/internal/schema"
	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memProxyRepo struct {
	byID map[string]*domain.Proxy
}

func (m *memProxyRepo) Save(_ context.Context, p *domain.Proxy) error {
	if m.byID == nil {
		m.byID = map[string]*domain.Proxy{}
	}
	cp := *p
	if p.Auth != nil {
		a := *p.Auth
		cp.Auth = &a
	}
	m.byID[p.Id] = &cp
	return nil
}

func (m *memProxyRepo) Find(_ context.Context, id string) (*domain.Proxy, error) {
	p, ok := m.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *p
	if p.Auth != nil {
		a := *p.Auth
		cp.Auth = &a
	}
	return &cp, nil
}

func (m *memProxyRepo) Delete(context.Context, string) error { return nil }
func (m *memProxyRepo) List(context.Context, domain.ListParams, domain.ProxyFilter) (domain.ListResult[domain.Proxy], error) {
	return domain.ListResult[domain.Proxy]{}, nil
}
func (m *memProxyRepo) GetByIds(context.Context, []string) ([]*domain.Proxy, error) {
	return nil, nil
}
func (m *memProxyRepo) FindByLabels(context.Context, map[string]string) ([]*domain.Proxy, error) {
	return nil, nil
}

type nopPublisher struct{}

func (nopPublisher) Publish(context.Context, ports.ChangeEvent) error { return nil }

var _ ports.ProxyRepositoryPort = (*memProxyRepo)(nil)

func TestApplyProxyV1_PreservesAuthWhenOmitted(t *testing.T) {
	repo := &memProxyRepo{byID: map[string]*domain.Proxy{
		"edge": {
			Id:       "edge",
			Protocol: domain.ProtocolHTTP,
			Host:     "10.0.0.1",
			Port:     8080,
			Auth:     &domain.BasicAuth{User: "u", Pass: "secret"},
		},
	}}
	svc := NewService(repo, nil, nil, nil, nil, nil, nopPublisher{})

	got, err := svc.ApplyProxyV1(context.Background(), schema.Metadata{Name: "edge"}, proxyv1.ProxySpecV1{
		Protocol: "http",
		Host:     "10.0.0.2",
		Port:     8081,
		// Auth omitted — must keep existing credentials
	})
	require.NoError(t, err)
	require.NotNil(t, got.Auth)
	assert.Equal(t, "u", got.Auth.User)
	assert.Equal(t, "secret", got.Auth.Pass)
	assert.Equal(t, "10.0.0.2", got.Host)
	assert.Equal(t, uint16(8081), got.Port)
}

func TestApplyProxyV1_ReplacesAuthWhenProvided(t *testing.T) {
	repo := &memProxyRepo{byID: map[string]*domain.Proxy{
		"edge": {
			Id:       "edge",
			Protocol: domain.ProtocolHTTP,
			Host:     "10.0.0.1",
			Port:     8080,
			Auth:     &domain.BasicAuth{User: "u", Pass: "secret"},
		},
	}}
	svc := NewService(repo, nil, nil, nil, nil, nil, nopPublisher{})

	got, err := svc.ApplyProxyV1(context.Background(), schema.Metadata{Name: "edge"}, proxyv1.ProxySpecV1{
		Protocol: "http",
		Host:     "10.0.0.1",
		Port:     8080,
		Auth:     &proxyv1.AuthSpec{User: "new", Pass: "newpass"},
	})
	require.NoError(t, err)
	require.NotNil(t, got.Auth)
	assert.Equal(t, "new", got.Auth.User)
	assert.Equal(t, "newpass", got.Auth.Pass)
}
