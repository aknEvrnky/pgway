package cli

import (
	"context"
	"fmt"

	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/aknEvrnky/pgway/internal/schema"
	balancerv1 "github.com/aknEvrnky/pgway/internal/schema/balancer/v1"
	entrypointv1 "github.com/aknEvrnky/pgway/internal/schema/entrypoint/v1"
	flowv1 "github.com/aknEvrnky/pgway/internal/schema/flow/v1"
	poolv1 "github.com/aknEvrnky/pgway/internal/schema/pool/v1"
	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
	routerv1 "github.com/aknEvrnky/pgway/internal/schema/router/v1"
	"gopkg.in/yaml.v3"
)

type Dispatcher struct {
	controlPlane ports.ControlPlane
}

func NewDispatcher(cp ports.ControlPlane) *Dispatcher {
	return &Dispatcher{controlPlane: cp}
}

func (d *Dispatcher) Apply(ctx context.Context, raw schema.RawResource) error {
	switch raw.Key() {
	case "Proxy/v1":
		return d.applyProxyV1(ctx, raw)
	case "Pool/v1":
		return d.applyPoolV1(ctx, raw)
	case "LoadBalancer/v1":
		return d.applyBalancerV1(ctx, raw)
	case "Router/v1":
		return d.applyRouterV1(ctx, raw)
	case "Flow/v1":
		return d.applyFlowV1(ctx, raw)
	case "Entrypoint/v1":
		return d.applyEntrypointV1(ctx, raw)
	default:
		return fmt.Errorf("unknown resource: %s", raw.Key())
	}
}

func (d *Dispatcher) ApplyAll(ctx context.Context, resources []schema.RawResource) error {
	for i, raw := range resources {
		if err := d.Apply(ctx, raw); err != nil {
			return fmt.Errorf("resource[%d] (%s): %w", i, raw.Key(), err)
		}
	}
	return nil
}

func (d *Dispatcher) applyProxyV1(ctx context.Context, raw schema.RawResource) error {
	var spec proxyv1.ProxySpecV1
	if err := yaml.Unmarshal(raw.SpecRaw, &spec); err != nil {
		return fmt.Errorf("decode proxy spec: %w", err)
	}

	proxy, err := d.controlPlane.ApplyProxyV1(ctx, raw.Metadata, spec)
	if err != nil {
		return err
	}

	fmt.Printf("proxy/%s applied\n", proxy.Id)
	return nil
}

func (d *Dispatcher) applyPoolV1(ctx context.Context, raw schema.RawResource) error {
	var spec poolv1.PoolSpecV1
	if err := yaml.Unmarshal(raw.SpecRaw, &spec); err != nil {
		return fmt.Errorf("decode pool spec: %w", err)
	}

	pool, err := d.controlPlane.ApplyPoolV1(ctx, raw.Metadata, spec)
	if err != nil {
		return err
	}

	fmt.Printf("pool/%s applied\n", pool.Id)
	return nil
}

func (d *Dispatcher) applyBalancerV1(ctx context.Context, raw schema.RawResource) error {
	var spec balancerv1.BalancerSpecV1
	if err := yaml.Unmarshal(raw.SpecRaw, &spec); err != nil {
		return fmt.Errorf("decode balancer spec: %w", err)
	}

	balancer, err := d.controlPlane.ApplyBalancerV1(ctx, raw.Metadata, spec)
	if err != nil {
		return err
	}

	fmt.Printf("balancer/%s applied\n", balancer.Id)
	return nil
}

func (d *Dispatcher) applyRouterV1(ctx context.Context, raw schema.RawResource) error {
	var spec routerv1.RouterSpecV1
	if err := yaml.Unmarshal(raw.SpecRaw, &spec); err != nil {
		return fmt.Errorf("decode router spec: %w", err)
	}

	router, err := d.controlPlane.ApplyRouterV1(ctx, raw.Metadata, spec)
	if err != nil {
		return err
	}

	fmt.Printf("router/%s applied\n", router.Id)
	return nil
}

func (d *Dispatcher) applyFlowV1(ctx context.Context, raw schema.RawResource) error {
	var spec flowv1.FlowSpecV1
	if err := yaml.Unmarshal(raw.SpecRaw, &spec); err != nil {
		return fmt.Errorf("decode flow spec: %w", err)
	}

	flow, err := d.controlPlane.ApplyFlowV1(ctx, raw.Metadata, spec)
	if err != nil {
		return err
	}

	fmt.Printf("flow/%s applied\n", flow.Id)
	return nil
}

func (d *Dispatcher) applyEntrypointV1(ctx context.Context, raw schema.RawResource) error {
	var spec entrypointv1.EntrypointSpecV1
	if err := yaml.Unmarshal(raw.SpecRaw, &spec); err != nil {
		return fmt.Errorf("decode flow spec: %w", err)
	}

	ep, err := d.controlPlane.ApplyEntrypointV1(ctx, raw.Metadata, spec)
	if err != nil {
		return err
	}

	fmt.Printf("entrypoint/%s applied\n", ep.Id)
	return nil
}
