package controlplane

import (
	"context"
	"fmt"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
	"github.com/oklog/ulid/v2"
	"go.uber.org/zap"
)

func (s *Service) ApplyProxyV1(ctx context.Context, meta schema.Metadata, spec proxyv1.ProxySpecV1) (*domain.Proxy, error) {
	// 1 - validate spec
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("spec validation: %w", err)
	}

	// 2 - assign ID
	if meta.Name == "" {
		meta.Name = ulid.Make().String()
		zap.L().Info("generated proxy name", zap.String("name", meta.Name))
	}

	// 3 - create domain from spec
	proxy, err := proxyFromSpecV1(meta, spec)
	if err != nil {
		return nil, fmt.Errorf("domain conversion: %w", err)
	}

	// 4 - domain validation
	if err := proxy.Validate(); err != nil {
		return nil, fmt.Errorf("proxy validation: %w", err)
	}

	// 5 - persist
	if err := s.proxyRepo.Save(ctx, proxy); err != nil {
		return nil, fmt.Errorf("persisting proxy: %w", err)
	}

	zap.L().Info("proxy applied", zap.String("name", proxy.Id))

	return proxy, nil
}

func (s *Service) GetProxy(ctx context.Context, name string) (*domain.Proxy, error) {
	if name == "" {
		return nil, fmt.Errorf("proxy name is required")
	}

	proxy, err := s.proxyRepo.Find(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("find proxy: %w", err)
	}

	return proxy, nil
}

func (s *Service) ListProxies(ctx context.Context) ([]*domain.Proxy, error) {
	proxies, err := s.proxyRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list proxies: %w", err)
	}

	return proxies, nil
}

func (s *Service) DeleteProxy(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("proxy name is required")
	}

	if err := s.proxyRepo.Delete(ctx, name); err != nil {
		return fmt.Errorf("delete proxy: %w", err)
	}

	zap.L().Info("proxy deleted", zap.String("name", name))
	return nil
}

func proxyFromSpecV1(meta schema.Metadata, spec proxyv1.ProxySpecV1) (*domain.Proxy, error) {
	if spec.URL != "" {
		proxy, err := domain.NewProxyFromURL(spec.URL)
		if err != nil {
			return nil, fmt.Errorf("parsing URL: %q %w", spec.URL, err)
		}

		proxy.Id = meta.Name
		proxy.Labels = meta.Labels

		return proxy, nil
	}

	// manual field mapping
	return &domain.Proxy{
		Id:       meta.Name,
		Protocol: domain.Protocol(spec.Protocol),
		Host:     spec.Host,
		Port:     spec.Port,
		Auth:     authFromSpec(spec.Auth),
		Labels:   meta.Labels,
	}, nil
}

func authFromSpec(spec *proxyv1.AuthSpec) *domain.BasicAuth {
	if spec == nil {
		return nil
	}

	return &domain.BasicAuth{
		User: spec.User,
		Pass: spec.Pass,
	}
}
