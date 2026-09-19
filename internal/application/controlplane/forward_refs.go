package controlplane

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/ports"
)

func (s *Service) requireProxy(ctx context.Context, applyType, applyName, proxyID string) error {
	if _, err := s.proxyRepo.Find(ctx, proxyID); err != nil {
		return newResourceMissingRef(applyType, applyName, string(ports.ResourceTypeProxy), proxyID)
	}
	return nil
}

func (s *Service) requirePool(ctx context.Context, applyType, applyName, poolID string) error {
	if _, err := s.poolRepo.Find(ctx, poolID); err != nil {
		return newResourceMissingRef(applyType, applyName, string(ports.ResourceTypePool), poolID)
	}
	return nil
}

func (s *Service) requireBalancer(ctx context.Context, applyType, applyName, balancerID string) error {
	if _, err := s.lbRepo.Find(ctx, balancerID); err != nil {
		return newResourceMissingRef(applyType, applyName, string(ports.ResourceTypeBalancer), balancerID)
	}
	return nil
}

func (s *Service) requireRouter(ctx context.Context, applyType, applyName, routerID string) error {
	if _, err := s.routerRepo.Find(ctx, routerID); err != nil {
		return newResourceMissingRef(applyType, applyName, string(ports.ResourceTypeRouter), routerID)
	}
	return nil
}

func (s *Service) requireFlow(ctx context.Context, applyType, applyName, flowID string) error {
	if _, err := s.flowRepo.Find(ctx, flowID); err != nil {
		return newResourceMissingRef(applyType, applyName, string(ports.ResourceTypeFlow), flowID)
	}
	return nil
}
