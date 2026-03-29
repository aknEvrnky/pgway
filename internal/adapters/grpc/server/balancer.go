package server

import (
	"context"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	balancerv1 "github.com/aknEvrnky/pgway/internal/schema/balancer/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *ControlPlaneServer) ApplyBalancerV1(ctx context.Context, req *controlplanev1.ApplyBalancerV1Request) (*controlplanev1.ApplyBalancerV1Response, error) {
	if req.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "spec is required")
	}

	meta := metaFromProto(req.Metadata)
	spec := balancerSpecFromProto(req.Spec)

	balancer, err := s.cp.ApplyBalancerV1(ctx, meta, spec)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "apply balancer: %v", err)
	}

	return &controlplanev1.ApplyBalancerV1Response{
		Balancer: balancerToProto(balancer),
	}, nil
}

func (s *ControlPlaneServer) GetBalancer(ctx context.Context, req *controlplanev1.GetBalancerRequest) (*controlplanev1.GetBalancerResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	balancer, err := s.cp.GetBalancer(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "balancer: %v", err)
	}

	return &controlplanev1.GetBalancerResponse{
		Balancer: balancerToProto(balancer),
	}, nil
}

func (s *ControlPlaneServer) ListBalancers(ctx context.Context, req *controlplanev1.ListBalancersRequest) (*controlplanev1.ListBalancersResponse, error) {
	cursor, err := decodeCursor(req.GetPageToken())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid page_token")
	}

	params := domain.ListParams{
		PageSize: int(req.GetPageSize()),
		Cursor:   cursor,
	}

	filter := domain.BalancerFilter{
		Search: req.GetSearch(),
		Type:   req.GetType(),
		PoolId: req.GetPoolId(),
	}

	result, err := s.cp.ListBalancers(ctx, params, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list balancers: %v", err)
	}

	balancers := make([]*controlplanev1.LoadBalancer, 0, len(result.Items))
	for _, bl := range result.Items {
		balancers = append(balancers, balancerToProto(bl))
	}

	return &controlplanev1.ListBalancersResponse{
		Balancers:     balancers,
		NextPageToken: encodeCursor(result.NextCursor),
		TotalCount:    int32(result.TotalCount),
	}, nil
}

func (s *ControlPlaneServer) DeleteBalancer(ctx context.Context, req *controlplanev1.DeleteBalancerRequest) (*controlplanev1.DeleteBalancerResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	err := s.cp.DeleteBalancer(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete balancer: %v", err)
	}

	return &controlplanev1.DeleteBalancerResponse{}, nil
}

func balancerSpecFromProto(pb *controlplanev1.BalancerSpecV1) balancerv1.BalancerSpecV1 {
	if pb == nil {
		return balancerv1.BalancerSpecV1{}
	}

	return balancerv1.BalancerSpecV1{
		Title:  pb.Title,
		Type:   pb.Type,
		PoolId: pb.PoolId,
	}
}

func balancerToProto(bl *domain.LoadBalancer) *controlplanev1.LoadBalancer {
	if bl == nil {
		return nil
	}

	return &controlplanev1.LoadBalancer{
		Id:        bl.Id,
		Title:     bl.Title,
		Type:      string(bl.Type),
		PoolId:    bl.PoolId,
		CreatedAt: timestamppb.New(bl.CreatedAt),
		UpdatedAt: timestamppb.New(bl.UpdatedAt),
	}
}
