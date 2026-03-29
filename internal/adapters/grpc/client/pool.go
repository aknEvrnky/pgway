package client

import (
	"context"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	poolv1 "github.com/aknEvrnky/pgway/internal/schema/pool/v1"
)

func (c *Client) ApplyPoolV1(ctx context.Context, meta schema.Metadata, spec poolv1.PoolSpecV1) (*domain.Pool, error) {
	resp, err := c.pool.ApplyPoolV1(ctx, &controlplanev1.ApplyPoolV1Request{
		Metadata: metaToProto(meta),
		Spec:     poolSpecToProto(spec),
	})
	if err != nil {
		return nil, err
	}

	return poolFromProto(resp.Pool), nil
}

func (c *Client) GetPool(ctx context.Context, name string) (*domain.Pool, error) {
	resp, err := c.pool.GetPool(ctx, &controlplanev1.GetPoolRequest{Name: name})
	if err != nil {
		return nil, err
	}

	return poolFromProto(resp.Pool), nil
}

func (c *Client) ListPools(ctx context.Context, params domain.ListParams) (domain.ListResult[domain.Pool], error) {
	resp, err := c.pool.ListPools(ctx, &controlplanev1.ListPoolsRequest{
		PageSize:  int32(params.PageSize),
		PageToken: params.Cursor,
	})
	if err != nil {
		return domain.ListResult[domain.Pool]{}, err
	}

	items := make([]*domain.Pool, 0, len(resp.Pools))
	for _, p := range resp.Pools {
		items = append(items, poolFromProto(p))
	}

	return domain.ListResult[domain.Pool]{
		Items:      items,
		NextCursor: resp.NextPageToken,
		TotalCount: int(resp.TotalCount),
	}, nil
}

func (c *Client) DeletePool(ctx context.Context, name string) error {
	_, err := c.pool.DeletePool(ctx, &controlplanev1.DeletePoolRequest{Name: name})
	return err
}

func poolSpecToProto(spec poolv1.PoolSpecV1) *controlplanev1.PoolSpecV1 {
	pb := &controlplanev1.PoolSpecV1{
		Title:    spec.Title,
		Type:     spec.Type,
		ProxyIds: spec.ProxyIds,
	}

	if spec.Selector != nil {
		pb.Selector = &controlplanev1.SelectorSpec{
			Allow: spec.Selector.Allow,
		}
	}

	return pb
}

func poolFromProto(pb *controlplanev1.Pool) *domain.Pool {
	if pb == nil {
		return nil
	}

	pool := &domain.Pool{
		Id:       pb.Id,
		Title:    pb.Title,
		Type:     domain.PoolType(pb.Type),
		Labels:   pb.Labels,
		ProxyIds: pb.ProxyIds,
	}

	if pb.Selector != nil {
		pool.Selector = &domain.LabelSelector{
			Allow: pb.Selector.Allow,
		}
	}

	return pool
}
