package client

import (
	"context"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	flowv1 "github.com/aknEvrnky/pgway/internal/schema/flow/v1"
)

func (c *Client) ApplyFlowV1(ctx context.Context, meta schema.Metadata, spec flowv1.FlowSpecV1) (*domain.Flow, error) {
	resp, err := c.flow.ApplyFlowV1(ctx, &controlplanev1.ApplyFlowV1Request{
		Metadata: metaToProto(meta),
		Spec: &controlplanev1.FlowSpecV1{
			RouterId:   spec.RouterId,
			BalancerId: spec.BalancerId,
		},
	})
	if err != nil {
		return nil, err
	}

	return flowFromProto(resp.Flow), nil
}

func (c *Client) GetFlow(ctx context.Context, name string) (*domain.Flow, error) {
	resp, err := c.flow.GetFlow(ctx, &controlplanev1.GetFlowRequest{Name: name})
	if err != nil {
		return nil, err
	}

	return flowFromProto(resp.Flow), nil
}

func (c *Client) ListFlows(ctx context.Context) ([]*domain.Flow, error) {
	resp, err := c.flow.ListFlows(ctx, &controlplanev1.ListFlowsRequest{})
	if err != nil {
		return nil, err
	}

	flows := make([]*domain.Flow, 0, len(resp.Flows))
	for _, f := range resp.Flows {
		flows = append(flows, flowFromProto(f))
	}

	return flows, nil
}

func (c *Client) DeleteFlow(ctx context.Context, name string) error {
	_, err := c.flow.DeleteFlow(ctx, &controlplanev1.DeleteFlowRequest{Name: name})
	return err
}

func flowFromProto(pb *controlplanev1.Flow) *domain.Flow {
	if pb == nil {
		return nil
	}

	return &domain.Flow{
		Id:         pb.Id,
		RouterId:   pb.RouterId,
		BalancerId: pb.BalancerId,
	}
}
