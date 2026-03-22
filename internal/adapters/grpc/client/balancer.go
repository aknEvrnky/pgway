package client

import (
	"context"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	balancerv1 "github.com/aknEvrnky/pgway/internal/schema/balancer/v1"
)

func (c *Client) ApplyBalancerV1(ctx context.Context, meta schema.Metadata, spec balancerv1.BalancerSpecV1) (*domain.LoadBalancer, error) {
	resp, err := c.balancer.ApplyBalancerV1(ctx, &controlplanev1.ApplyBalancerV1Request{
		Metadata: metaToProto(meta),
		Spec: &controlplanev1.BalancerSpecV1{
			Title:  spec.Title,
			Type:   spec.Type,
			PoolId: spec.PoolId,
		},
	})
	if err != nil {
		return nil, err
	}

	return balancerFromProto(resp.Balancer), nil
}

func (c *Client) GetBalancer(ctx context.Context, name string) (*domain.LoadBalancer, error) {
	resp, err := c.balancer.GetBalancer(ctx, &controlplanev1.GetBalancerRequest{Name: name})
	if err != nil {
		return nil, err
	}

	return balancerFromProto(resp.Balancer), nil
}

func (c *Client) ListBalancers(ctx context.Context) ([]*domain.LoadBalancer, error) {
	resp, err := c.balancer.ListBalancers(ctx, &controlplanev1.ListBalancersRequest{})
	if err != nil {
		return nil, err
	}

	balancers := make([]*domain.LoadBalancer, 0, len(resp.Balancers))
	for _, b := range resp.Balancers {
		balancers = append(balancers, balancerFromProto(b))
	}

	return balancers, nil
}

func (c *Client) DeleteBalancer(ctx context.Context, name string) error {
	_, err := c.balancer.DeleteBalancer(ctx, &controlplanev1.DeleteBalancerRequest{Name: name})
	return err
}

func balancerFromProto(pb *controlplanev1.LoadBalancer) *domain.LoadBalancer {
	if pb == nil {
		return nil
	}

	return &domain.LoadBalancer{
		Id:     pb.Id,
		Title:  pb.Title,
		Type:   domain.BalancerType(pb.Type),
		PoolId: pb.PoolId,
	}
}
