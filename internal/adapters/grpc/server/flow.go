package server

import (
	"context"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	flowv1 "github.com/aknEvrnky/pgway/internal/schema/flow/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *ControlPlaneServer) ApplyFlowV1(ctx context.Context, req *controlplanev1.ApplyFlowV1Request) (*controlplanev1.ApplyFlowV1Response, error) {
	if req.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "spec is required")
	}

	meta := metaFromProto(req.Metadata)
	spec := flowSpecFromProto(req.Spec)

	flow, err := s.cp.ApplyFlowV1(ctx, meta, spec)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "apply flow: %v", err)
	}

	return &controlplanev1.ApplyFlowV1Response{
		Flow: flowToProto(flow),
	}, nil
}

func (s *ControlPlaneServer) GetFlow(ctx context.Context, req *controlplanev1.GetFlowRequest) (*controlplanev1.GetFlowResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	flow, err := s.cp.GetFlow(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "flow: %v", err)
	}

	return &controlplanev1.GetFlowResponse{Flow: flowToProto(flow)}, nil
}

func (s *ControlPlaneServer) ListFlows(ctx context.Context, req *controlplanev1.ListFlowsRequest) (*controlplanev1.ListFlowsResponse, error) {
	flows, err := s.cp.ListFlows(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list flows: %v", err)
	}

	results := make([]*controlplanev1.Flow, 0, len(flows))
	for _, f := range flows {
		results = append(results, flowToProto(f))
	}

	return &controlplanev1.ListFlowsResponse{Flows: results}, nil
}

func (s *ControlPlaneServer) DeleteFlow(ctx context.Context, req *controlplanev1.DeleteFlowRequest) (*controlplanev1.DeleteFlowResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	if err := s.cp.DeleteFlow(ctx, req.Name); err != nil {
		return nil, status.Errorf(codes.Internal, "delete flow: %v", err)
	}

	return &controlplanev1.DeleteFlowResponse{}, nil
}

func flowSpecFromProto(pb *controlplanev1.FlowSpecV1) flowv1.FlowSpecV1 {
	if pb == nil {
		return flowv1.FlowSpecV1{}
	}

	return flowv1.FlowSpecV1{
		RouterId:   pb.RouterId,
		BalancerId: pb.BalancerId,
	}
}

func flowToProto(flow *domain.Flow) *controlplanev1.Flow {
	if flow == nil {
		return nil
	}

	return &controlplanev1.Flow{
		Id:         flow.Id,
		RouterId:   flow.RouterId,
		BalancerId: flow.BalancerId,
		CreatedAt:  timestamppb.New(flow.CreatedAt),
		UpdatedAt:  timestamppb.New(flow.UpdatedAt),
	}
}
