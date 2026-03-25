package server

import (
	"context"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	poolv1 "github.com/aknEvrnky/pgway/internal/schema/pool/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *ControlPlaneServer) ApplyPoolV1(ctx context.Context, req *controlplanev1.ApplyPoolV1Request) (*controlplanev1.ApplyPoolV1Response, error) {
	if req.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "spec is required")
	}

	meta := metaFromProto(req.Metadata)
	spec := poolSpecFromProto(req.Spec)

	pool, err := s.cp.ApplyPoolV1(ctx, meta, spec)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "apply pool: %v", err)
	}

	return &controlplanev1.ApplyPoolV1Response{
		Pool: poolToProto(pool),
	}, nil
}

func (s *ControlPlaneServer) GetPool(ctx context.Context, req *controlplanev1.GetPoolRequest) (*controlplanev1.GetPoolResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	pool, err := s.cp.GetPool(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "pool: %v", err)
	}

	return &controlplanev1.GetPoolResponse{Pool: poolToProto(pool)}, nil
}

func (s *ControlPlaneServer) ListPools(ctx context.Context, req *controlplanev1.ListPoolsRequest) (*controlplanev1.ListPoolsResponse, error) {
	pools, err := s.cp.ListPools(ctx)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "list pools: %v", err)
	}

	var results = make([]*controlplanev1.Pool, 0, len(pools))
	for _, p := range pools {
		results = append(results, poolToProto(p))
	}

	return &controlplanev1.ListPoolsResponse{Pools: results}, nil
}

func (s *ControlPlaneServer) DeletePool(ctx context.Context, req *controlplanev1.DeletePoolRequest) (*controlplanev1.DeletePoolResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	err := s.cp.DeletePool(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete pool: %v", err)
	}

	return &controlplanev1.DeletePoolResponse{}, nil
}

func poolSpecFromProto(pb *controlplanev1.PoolSpecV1) poolv1.PoolSpecV1 {
	if pb == nil {
		return poolv1.PoolSpecV1{}
	}

	return poolv1.PoolSpecV1{
		Title:    pb.Title,
		Type:     pb.Type,
		ProxyIds: pb.ProxyIds,
		Selector: func() *poolv1.SelectorSpec {
			if pb.Selector == nil {
				return nil
			}

			return &poolv1.SelectorSpec{
				Allow: pb.Selector.Allow,
			}
		}(),
	}
}

func poolToProto(pool *domain.Pool) *controlplanev1.Pool {
	if pool == nil {
		return nil
	}

	return &controlplanev1.Pool{
		Id:        pool.Id,
		Title:     pool.Title,
		Type:      string(pool.Type),
		Labels:    pool.Labels,
		ProxyIds:  pool.ProxyIds,
		Selector:  selectorToProto(pool.Selector),
		CreatedAt: timestamppb.New(pool.CreatedAt),
		UpdatedAt: timestamppb.New(pool.UpdatedAt),
	}

}

func selectorToProto(selector *domain.LabelSelector) *controlplanev1.SelectorSpec {
	if selector == nil {
		return nil
	}

	return &controlplanev1.SelectorSpec{
		Allow: selector.Allow,
	}
}
