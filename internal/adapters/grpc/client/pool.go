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

func (c *Client) ListPools(ctx context.Context, params domain.ListParams, filter domain.PoolFilter) (domain.ListResult[domain.Pool], error) {
	resp, err := c.pool.ListPools(ctx, &controlplanev1.ListPoolsRequest{
		PageSize:  int32(params.PageSize),
		PageToken: params.Cursor,
		Search:    filter.Search,
		Type:      filter.Type,
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
		Title: spec.Title,
		Type:  spec.Type,
	}

	if len(spec.Members) > 0 {
		pb.Members = make([]*controlplanev1.PoolMember, 0, len(spec.Members))
		for _, m := range spec.Members {
			weight := int32(0)
			if m.Weight != nil {
				weight = int32(*m.Weight)
			}
			pb.Members = append(pb.Members, &controlplanev1.PoolMember{
				ProxyId: m.ProxyId,
				Weight:  weight,
			})
		}
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
		Id:     pb.Id,
		Title:  pb.Title,
		Type:   domain.PoolType(pb.Type),
		Labels: pb.Labels,
	}

	if len(pb.Members) > 0 {
		pool.Members = make([]domain.PoolMember, 0, len(pb.Members))
		for _, m := range pb.Members {
			weight := int(m.Weight)
			if weight == 0 {
				weight = 1
			}
			pool.Members = append(pool.Members, domain.PoolMember{
				ProxyId: m.ProxyId,
				Weight:  weight,
			})
		}
	}

	if pb.Selector != nil {
		pool.Selector = &domain.LabelSelector{
			Allow: pb.Selector.Allow,
		}
	}

	return pool
}
