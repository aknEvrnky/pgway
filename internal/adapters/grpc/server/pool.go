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
	cursor, err := decodeCursor(req.GetPageToken())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid page_token")
	}

	params := domain.ListParams{
		PageSize: int(req.GetPageSize()),
		Cursor:   cursor,
	}

	filter := domain.PoolFilter{
		Search: req.GetSearch(),
		Type:   req.GetType(),
	}

	result, err := s.cp.ListPools(ctx, params, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list pools: %v", err)
	}

	pools := make([]*controlplanev1.Pool, 0, len(result.Items))
	for _, p := range result.Items {
		pools = append(pools, poolToProto(p))
	}

	return &controlplanev1.ListPoolsResponse{
		Pools:         pools,
		NextPageToken: encodeCursor(result.NextCursor),
		TotalCount:    int32(result.TotalCount),
	}, nil
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

	spec := poolv1.PoolSpecV1{
		Title: pb.Title,
		Type:  pb.Type,
	}

	if len(pb.Members) > 0 {
		spec.Members = make([]poolv1.PoolMemberSpec, 0, len(pb.Members))
		for _, m := range pb.Members {
			ms := poolv1.PoolMemberSpec{ProxyId: m.ProxyId}
			if m.Weight != 0 {
				w := int(m.Weight)
				ms.Weight = &w
			}
			spec.Members = append(spec.Members, ms)
		}
	}

	if pb.Selector != nil {
		spec.Selector = &poolv1.SelectorSpec{
			Allow: pb.Selector.Allow,
		}
	}

	return spec
}

func poolToProto(pool *domain.Pool) *controlplanev1.Pool {
	if pool == nil {
		return nil
	}

	pb := &controlplanev1.Pool{
		Id:        pool.Id,
		Title:     pool.Title,
		Type:      string(pool.Type),
		Labels:    pool.Labels,
		Selector:  selectorToProto(pool.Selector),
		CreatedAt: timestamppb.New(pool.CreatedAt),
		UpdatedAt: timestamppb.New(pool.UpdatedAt),
	}

	if len(pool.Members) > 0 {
		pb.Members = make([]*controlplanev1.PoolMember, 0, len(pool.Members))
		for _, m := range pool.Members {
			pb.Members = append(pb.Members, &controlplanev1.PoolMember{
				ProxyId: m.ProxyId,
				Weight:  int32(m.Weight),
			})
		}
	}

	return pb
}

func selectorToProto(selector *domain.LabelSelector) *controlplanev1.SelectorSpec {
	if selector == nil {
		return nil
	}

	return &controlplanev1.SelectorSpec{
		Allow: selector.Allow,
	}
}
