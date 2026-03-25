package server

import (
	"context"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	entrypointv1 "github.com/aknEvrnky/pgway/internal/schema/entrypoint/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *ControlPlaneServer) ApplyEntrypointV1(ctx context.Context, req *controlplanev1.ApplyEntrypointV1Request) (*controlplanev1.ApplyEntrypointV1Response, error) {
	if req.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "spec is required")
	}

	meta := metaFromProto(req.Metadata)
	spec := entrypointSpecFromProto(req.Spec)

	ep, err := s.cp.ApplyEntrypointV1(ctx, meta, spec)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "apply entrypoint: %v", err)
	}

	return &controlplanev1.ApplyEntrypointV1Response{
		Entrypoint: entrypointToProto(ep),
	}, nil
}

func (s *ControlPlaneServer) GetEntrypoint(ctx context.Context, req *controlplanev1.GetEntrypointRequest) (*controlplanev1.GetEntrypointResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	ep, err := s.cp.GetEntrypoint(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "entrypoint: %v", err)
	}

	return &controlplanev1.GetEntrypointResponse{
		Entrypoint: entrypointToProto(ep),
	}, nil
}

func (s *ControlPlaneServer) ListEntrypoints(ctx context.Context, request *controlplanev1.ListEntrypointsRequest) (*controlplanev1.ListEntrypointsResponse, error) {
	entrypoints, err := s.cp.ListEntrypoints(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list entrypoints: %v", err)
	}

	var results = make([]*controlplanev1.Entrypoint, 0, len(entrypoints))
	for _, ep := range entrypoints {
		results = append(results, entrypointToProto(ep))
	}

	return &controlplanev1.ListEntrypointsResponse{
		Entrypoints: results,
	}, nil
}

func (s *ControlPlaneServer) DeleteEntrypoint(ctx context.Context, req *controlplanev1.DeleteEntrypointRequest) (*controlplanev1.DeleteEntrypointResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	err := s.cp.DeleteEntrypoint(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete entrypoint: %v", err)
	}

	return &controlplanev1.DeleteEntrypointResponse{}, nil
}

func entrypointSpecFromProto(pb *controlplanev1.EntrypointSpecV1) entrypointv1.EntrypointSpecV1 {
	if pb == nil {
		return entrypointv1.EntrypointSpecV1{}
	}

	return entrypointv1.EntrypointSpecV1{
		Title:    pb.Title,
		Protocol: pb.Protocol,
		Host:     pb.Host,
		Port:     uint16(pb.Port),
		FlowId:   pb.FlowId,
	}
}

func entrypointToProto(ep *domain.Entrypoint) *controlplanev1.Entrypoint {
	if ep == nil {
		return nil
	}

	return &controlplanev1.Entrypoint{
		Id:        ep.Id,
		Title:     ep.Title,
		Protocol:  string(ep.Protocol),
		Host:      ep.Host,
		Port:      uint32(ep.Port),
		FlowId:    ep.FlowId,
		CreatedAt: timestamppb.New(ep.CreatedAt),
		UpdatedAt: timestamppb.New(ep.UpdatedAt),
	}
}
