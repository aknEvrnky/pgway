package client

import (
	"context"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	routerv1 "github.com/aknEvrnky/pgway/internal/schema/router/v1"
)

func (c *Client) ApplyRouterV1(ctx context.Context, meta schema.Metadata, spec routerv1.RouterSpecV1) (*domain.Router, error) {
	resp, err := c.router.ApplyRouterV1(ctx, &controlplanev1.ApplyRouterV1Request{
		Metadata: metaToProto(meta),
		Spec:     routerSpecToProto(spec),
	})
	if err != nil {
		return nil, err
	}

	return routerFromProto(resp.Router), nil
}

func (c *Client) GetRouter(ctx context.Context, name string) (*domain.Router, error) {
	resp, err := c.router.GetRouter(ctx, &controlplanev1.GetRouterRequest{Name: name})
	if err != nil {
		return nil, err
	}

	return routerFromProto(resp.Router), nil
}

func (c *Client) ListRouters(ctx context.Context) ([]*domain.Router, error) {
	resp, err := c.router.ListRouters(ctx, &controlplanev1.ListRoutersRequest{})
	if err != nil {
		return nil, err
	}

	routers := make([]*domain.Router, 0, len(resp.Routers))
	for _, r := range resp.Routers {
		routers = append(routers, routerFromProto(r))
	}

	return routers, nil
}

func (c *Client) DeleteRouter(ctx context.Context, name string) error {
	_, err := c.router.DeleteRouter(ctx, &controlplanev1.DeleteRouterRequest{Name: name})
	return err
}

func routerSpecToProto(spec routerv1.RouterSpecV1) *controlplanev1.RouterSpecV1 {
	rules := make([]*controlplanev1.RuleSpec, 0, len(spec.Rules))
	for _, r := range spec.Rules {
		rules = append(rules, ruleSpecToProto(r))
	}

	return &controlplanev1.RouterSpecV1{
		Title:       spec.Title,
		Description: spec.Description,
		Rules:       rules,
	}
}

func ruleSpecToProto(r routerv1.RuleSpec) *controlplanev1.RuleSpec {
	return &controlplanev1.RuleSpec{
		Id:     r.Id,
		Match:  matchSpecToProto(r.Match),
		Target: r.Target,
	}
}

func matchSpecToProto(m routerv1.MatchSpec) *controlplanev1.MatchSpec {
	pb := &controlplanev1.MatchSpec{
		Type:  m.Type,
		Value: m.Value,
	}

	for _, c := range m.All {
		pb.All = append(pb.All, conditionSpecToProto(c))
	}

	for _, c := range m.Any {
		pb.Any = append(pb.Any, conditionSpecToProto(c))
	}

	if m.Not != nil {
		pb.Not = conditionSpecToProto(*m.Not)
	}

	return pb
}

func conditionSpecToProto(c routerv1.ConditionSpec) *controlplanev1.ConditionSpec {
	return &controlplanev1.ConditionSpec{
		Type:  c.Type,
		Value: c.Value,
	}
}

func routerFromProto(pb *controlplanev1.Router) *domain.Router {
	if pb == nil {
		return nil
	}

	rules := make([]*domain.RouterRule, 0, len(pb.Rules))
	for _, r := range pb.Rules {
		rules = append(rules, routerRuleFromProto(r))
	}

	return &domain.Router{
		Id:          pb.Id,
		Title:       pb.Title,
		Description: pb.Description,
		Rules:       rules,
	}
}

func routerRuleFromProto(pb *controlplanev1.RouterRule) *domain.RouterRule {
	if pb == nil {
		return nil
	}

	return &domain.RouterRule{
		Id:     pb.Id,
		Match:  routerMatchFromProto(pb.Match),
		Target: pb.Target,
	}
}

func routerMatchFromProto(pb *controlplanev1.RouterMatch) domain.RouterMatch {
	if pb == nil {
		return domain.RouterMatch{}
	}

	match := domain.RouterMatch{
		Type:  domain.MatchType(pb.Type),
		Value: pb.Value,
	}

	for _, c := range pb.All {
		match.All = append(match.All, routerConditionFromProto(c))
	}

	for _, c := range pb.Any {
		match.Any = append(match.Any, routerConditionFromProto(c))
	}

	if pb.Not != nil {
		cond := routerConditionFromProto(pb.Not)
		match.Not = &cond
	}

	return match
}

func routerConditionFromProto(pb *controlplanev1.RouterCondition) domain.RouterCondition {
	if pb == nil {
		return domain.RouterCondition{}
	}

	return domain.RouterCondition{
		Type:  domain.MatchType(pb.Type),
		Value: pb.Value,
	}
}
