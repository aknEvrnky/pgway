package cli

import (
	"context"
	"fmt"

	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/aknEvrnky/pgway/internal/schema"
	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
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
