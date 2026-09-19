package controlplane

import (
	"context"
	"fmt"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
)

func (s *Service) rejectIfPoolsReferenceProxy(ctx context.Context, proxyID string) error {
	result, err := s.poolRepo.List(ctx, domain.ListParams{PageSize: 0}, domain.PoolFilter{ProxyId: proxyID})
	if err != nil {
		return fmt.Errorf("checking pools for proxy %q: %w", proxyID, err)
	}
	if len(result.Items) == 0 {
		return nil
	}
	deps := make([]ResourceDependent, 0, len(result.Items))
	for _, p := range result.Items {
		deps = append(deps, ResourceDependent{Type: string(ports.ResourceTypePool), Name: p.Id})
	}
	return newResourceInUse(string(ports.ResourceTypeProxy), proxyID, deps)
}

func (s *Service) rejectIfBalancersReferencePool(ctx context.Context, poolID string) error {
	result, err := s.lbRepo.List(ctx, domain.ListParams{PageSize: 0}, domain.BalancerFilter{PoolId: poolID})
	if err != nil {
		return fmt.Errorf("checking balancers for pool %q: %w", poolID, err)
	}
	if len(result.Items) == 0 {
		return nil
	}
	deps := make([]ResourceDependent, 0, len(result.Items))
	for _, lb := range result.Items {
		deps = append(deps, ResourceDependent{Type: string(ports.ResourceTypeBalancer), Name: lb.Id})
	}
	return newResourceInUse(string(ports.ResourceTypePool), poolID, deps)
}

func (s *Service) rejectIfFlowsOrRoutersReferenceBalancer(ctx context.Context, balancerID string) error {
	flows, err := s.flowRepo.List(ctx, domain.ListParams{PageSize: 0}, domain.FlowFilter{BalancerId: balancerID})
	if err != nil {
		return fmt.Errorf("checking flows for balancer %q: %w", balancerID, err)
	}
	routers, err := s.routerRepo.List(ctx, domain.ListParams{PageSize: 0}, domain.RouterFilter{TargetBalancerId: balancerID})
	if err != nil {
		return fmt.Errorf("checking routers for balancer %q: %w", balancerID, err)
	}
	if len(flows.Items) == 0 && len(routers.Items) == 0 {
		return nil
	}
	deps := make([]ResourceDependent, 0, len(flows.Items)+len(routers.Items))
	for _, f := range flows.Items {
		deps = append(deps, ResourceDependent{Type: string(ports.ResourceTypeFlow), Name: f.Id})
	}
	for _, r := range routers.Items {
		deps = append(deps, ResourceDependent{Type: string(ports.ResourceTypeRouter), Name: r.Id})
	}
	return newResourceInUse(string(ports.ResourceTypeBalancer), balancerID, deps)
}

func (s *Service) rejectIfFlowsReferenceRouter(ctx context.Context, routerID string) error {
	result, err := s.flowRepo.List(ctx, domain.ListParams{PageSize: 0}, domain.FlowFilter{RouterId: routerID})
	if err != nil {
		return fmt.Errorf("checking flows for router %q: %w", routerID, err)
	}
	if len(result.Items) == 0 {
		return nil
	}
	deps := make([]ResourceDependent, 0, len(result.Items))
	for _, f := range result.Items {
		deps = append(deps, ResourceDependent{Type: string(ports.ResourceTypeFlow), Name: f.Id})
	}
	return newResourceInUse(string(ports.ResourceTypeRouter), routerID, deps)
}

func (s *Service) rejectIfEntrypointsReferenceFlow(ctx context.Context, flowID string) error {
	result, err := s.epRepo.List(ctx, domain.ListParams{PageSize: 0}, domain.EntrypointFilter{FlowId: flowID})
	if err != nil {
		return fmt.Errorf("checking entrypoints for flow %q: %w", flowID, err)
	}
	if len(result.Items) == 0 {
		return nil
	}
	deps := make([]ResourceDependent, 0, len(result.Items))
	for _, ep := range result.Items {
		deps = append(deps, ResourceDependent{Type: string(ports.ResourceTypeEntrypoint), Name: ep.Id})
	}
	return newResourceInUse(string(ports.ResourceTypeFlow), flowID, deps)
}
