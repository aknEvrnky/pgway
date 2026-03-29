package client

import (
	"context"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	entrypointv1 "github.com/aknEvrnky/pgway/internal/schema/entrypoint/v1"
)

func (c *Client) ApplyEntrypointV1(ctx context.Context, meta schema.Metadata, spec entrypointv1.EntrypointSpecV1) (*domain.Entrypoint, error) {
	resp, err := c.entrypoint.ApplyEntrypointV1(ctx, &controlplanev1.ApplyEntrypointV1Request{
		Metadata: metaToProto(meta),
		Spec: &controlplanev1.EntrypointSpecV1{
			Title:    spec.Title,
			Protocol: spec.Protocol,
			Host:     spec.Host,
			Port:     uint32(spec.Port),
			FlowId:   spec.FlowId,
		},
	})
	if err != nil {
		return nil, err
	}

	return entrypointFromProto(resp.Entrypoint), nil
}

func (c *Client) GetEntrypoint(ctx context.Context, name string) (*domain.Entrypoint, error) {
	resp, err := c.entrypoint.GetEntrypoint(ctx, &controlplanev1.GetEntrypointRequest{Name: name})
	if err != nil {
		return nil, err
	}

	return entrypointFromProto(resp.Entrypoint), nil
}

func (c *Client) ListEntrypoints(ctx context.Context, params domain.ListParams) (domain.ListResult[domain.Entrypoint], error) {
	resp, err := c.entrypoint.ListEntrypoints(ctx, &controlplanev1.ListEntrypointsRequest{
		PageSize:  int32(params.PageSize),
		PageToken: params.Cursor,
	})
	if err != nil {
		return domain.ListResult[domain.Entrypoint]{}, err
	}

	items := make([]*domain.Entrypoint, 0, len(resp.Entrypoints))
	for _, e := range resp.Entrypoints {
		items = append(items, entrypointFromProto(e))
	}

	return domain.ListResult[domain.Entrypoint]{
		Items:      items,
		NextCursor: resp.NextPageToken,
		TotalCount: int(resp.TotalCount),
	}, nil
}

func (c *Client) DeleteEntrypoint(ctx context.Context, name string) error {
	_, err := c.entrypoint.DeleteEntrypoint(ctx, &controlplanev1.DeleteEntrypointRequest{Name: name})
	return err
}

func entrypointFromProto(pb *controlplanev1.Entrypoint) *domain.Entrypoint {
	if pb == nil {
		return nil
	}

	return &domain.Entrypoint{
		Id:       pb.Id,
		Title:    pb.Title,
		Protocol: domain.Protocol(pb.Protocol),
		Host:     pb.Host,
		Port:     uint16(pb.Port),
		FlowId:   pb.FlowId,
	}
}
