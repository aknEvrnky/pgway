package server

import (
	"context"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	routerv1 "github.com/aknEvrnky/pgway/internal/schema/router/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *ControlPlaneServer) ApplyRouterV1(ctx context.Context, req *controlplanev1.ApplyRouterV1Request) (*controlplanev1.ApplyRouterV1Response, error) {
	if req.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "spec is required")
	}

	meta := metaFromProto(req.Metadata)
	spec := routerSpecFromProto(req.Spec)

	router, err := s.cp.ApplyRouterV1(ctx, meta, spec)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "apply router: %v", router)
	}

	return &controlplanev1.ApplyRouterV1Response{
		Router: routerToProto(router),
	}, nil
}

func (s *ControlPlaneServer) GetRouter(ctx context.Context, req *controlplanev1.GetRouterRequest) (*controlplanev1.GetRouterResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	router, err := s.cp.GetRouter(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "router: %v", err)
	}

	return &controlplanev1.GetRouterResponse{
		Router: routerToProto(router),
	}, nil
}

func (s *ControlPlaneServer) ListRouters(ctx context.Context, req *controlplanev1.ListRoutersRequest) (*controlplanev1.ListRoutersResponse, error) {
	routers, err := s.cp.ListRouters(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list routers: %v", err)
	}

	var results = make([]*controlplanev1.Router, 0, len(routers))

	for _, r := range routers {
		results = append(results, routerToProto(r))
	}

	return &controlplanev1.ListRoutersResponse{
		Routers: results,
	}, nil
}

func (s *ControlPlaneServer) DeleteRouter(ctx context.Context, req *controlplanev1.DeleteRouterRequest) (*controlplanev1.DeleteRouterResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	err := s.cp.DeleteRouter(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete router: %v", err)
	}

	return &controlplanev1.DeleteRouterResponse{}, nil
}

func routerSpecFromProto(pb *controlplanev1.RouterSpecV1) routerv1.RouterSpecV1 {
	if pb == nil {
		return routerv1.RouterSpecV1{}
	}

	var rules = make([]routerv1.RuleSpec, 0, len(pb.Rules))

	for _, r := range pb.Rules {
		rules = append(rules, ruleSpecFromProto(r))
	}

	return routerv1.RouterSpecV1{
		Title:       pb.Title,
		Description: pb.Description,
		Rules:       rules,
	}
}

func ruleSpecFromProto(pb *controlplanev1.RuleSpec) routerv1.RuleSpec {
	if pb == nil {
		return routerv1.RuleSpec{}
	}

	return routerv1.RuleSpec{
		Id:     pb.Id,
		Match:  matchSpecFromProto(pb.Match),
		Target: pb.Target,
	}
}

func matchSpecFromProto(pb *controlplanev1.MatchSpec) routerv1.MatchSpec {
	if pb == nil {
		return routerv1.MatchSpec{}
	}

	var allConds = make([]routerv1.ConditionSpec, 0, len(pb.All))
	for _, spec := range pb.All {
		allConds = append(allConds, conditionSpecFromProto(spec))
	}

	var anyConds = make([]routerv1.ConditionSpec, 0, len(pb.Any))
	for _, spec := range pb.Any {
		anyConds = append(anyConds, conditionSpecFromProto(spec))
	}

	var notCond *routerv1.ConditionSpec = nil

	if pb.Not != nil {
		cond := conditionSpecFromProto(pb.Not)
		notCond = &cond
	}

	return routerv1.MatchSpec{
		Type:  pb.Type,
		Value: pb.Value,
		All:   allConds,
		Any:   anyConds,
		Not:   notCond,
	}
}

func conditionSpecFromProto(pb *controlplanev1.ConditionSpec) routerv1.ConditionSpec {
	if pb == nil {
		return routerv1.ConditionSpec{}
	}

	return routerv1.ConditionSpec{
		Type:  pb.Type,
		Value: pb.Value,
	}
}

func routerToProto(router *domain.Router) *controlplanev1.Router {
	if router == nil {
		return nil
	}

	var rules = make([]*controlplanev1.RouterRule, 0, len(router.Rules))
	for _, r := range router.Rules {
		rules = append(rules, rulesToProto(r))
	}

	return &controlplanev1.Router{
		Id:          router.Id,
		Title:       router.Title,
		Description: router.Description,
		Rules:       rules,
	}
}

func rulesToProto(rules *domain.RouterRule) *controlplanev1.RouterRule {
	if rules == nil {
		return nil
	}

	var allConds = make([]*controlplanev1.RouterCondition, 0, len(rules.Match.All))
	for _, c := range rules.Match.All {
		allConds = append(allConds, conditionToProto(c))
	}

	var anyConds = make([]*controlplanev1.RouterCondition, 0, len(rules.Match.Any))
	for _, c := range rules.Match.Any {
		anyConds = append(anyConds, conditionToProto(c))
	}

	var notCond *controlplanev1.RouterCondition = nil
	if rules.Match.Not != nil {
		c := conditionToProto(*rules.Match.Not)
		notCond = c
	}

	return &controlplanev1.RouterRule{
		Id: rules.Id,
		Match: &controlplanev1.RouterMatch{
			Type:  string(rules.Match.Type),
			Value: rules.Match.Value,
			All:   allConds,
			Any:   anyConds,
			Not:   notCond,
		},
		Target: rules.Target,
	}
}

func conditionToProto(cond domain.RouterCondition) *controlplanev1.RouterCondition {
	return &controlplanev1.RouterCondition{
		Type:  string(cond.Type),
		Value: cond.Value,
	}
}
