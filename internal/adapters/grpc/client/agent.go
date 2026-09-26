package client

import (
	"context"
	"fmt"
	"time"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c *Client) CreateRegistrationToken(ctx context.Context, ttl time.Duration) (string, error) {
	resp, err := c.agent.CreateRegistrationToken(ctx, &controlplanev1.CreateRegistrationTokenRequest{
		TtlSeconds: int64(ttl / time.Second),
	})
	if err != nil {
		return "", err
	}
	return resp.RegistrationToken, nil
}

func (c *Client) Register(ctx context.Context, regToken string, agent domain.Agent) (*domain.Agent, string, error) {
	resp, err := c.agent.Register(ctx, &controlplanev1.RegisterRequest{
		RegistrationToken: regToken,
		Info: &controlplanev1.AgentInfo{
			Name:     agent.Id,
			Hostname: agent.Hostname,
			Version:  agent.Version,
			Labels:   agent.Labels,
		},
	})
	if err != nil {
		return nil, "", mapAgentAuth(err)
	}

	return &domain.Agent{Id: resp.AgentId}, resp.AgentToken, nil
}

func (c *Client) Heartbeat(ctx context.Context) (time.Time, error) {
	resp, err := c.agent.Heartbeat(ctx, &controlplanev1.HeartbeatRequest{})
	if err != nil {
		return time.Time{}, mapAgentAuth(err)
	}
	return resp.TokenExpiresAt.AsTime(), nil
}

func (c *Client) Deregister(ctx context.Context) error {
	_, err := c.agent.Deregister(ctx, &controlplanev1.DeregisterRequest{})
	return mapAgentAuth(err)
}

func mapAgentAuth(err error) error {
	if err == nil {
		return nil
	}
	if status.Code(err) == codes.Unauthenticated {
		return fmt.Errorf("%w: %w", ports.ErrAgentUnauthenticated, err)
	}
	return err
}

func (c *Client) ListAgents(ctx context.Context, params domain.ListParams, filter domain.AgentFilter) (domain.ListResult[domain.Agent], error) {
	resp, err := c.agent.ListAgents(ctx, &controlplanev1.ListAgentsRequest{
		PageSize:  int32(params.PageSize),
		PageToken: params.Cursor,
		Search:    filter.Search,
		Labels:    filter.Labels,
	})
	if err != nil {
		return domain.ListResult[domain.Agent]{}, err
	}

	items := make([]*domain.Agent, 0, len(resp.Agents))
	for _, a := range resp.Agents {
		items = append(items, agentFromProto(a))
	}

	return domain.ListResult[domain.Agent]{
		Items:      items,
		NextCursor: resp.NextPageToken,
		TotalCount: int(resp.TotalCount),
	}, nil
}

// ListAgentsDetailed returns agents with derived status from the server.
// Used by CLI display; AgentManager.ListAgents drops status (not on domain).
func (c *Client) ListAgentsDetailed(ctx context.Context, params domain.ListParams, filter domain.AgentFilter) ([]*controlplanev1.Agent, error) {
	resp, err := c.agent.ListAgents(ctx, &controlplanev1.ListAgentsRequest{
		PageSize:  int32(params.PageSize),
		PageToken: params.Cursor,
		Search:    filter.Search,
		Labels:    filter.Labels,
	})
	if err != nil {
		return nil, err
	}
	return resp.Agents, nil
}

func (c *Client) DeleteAgent(ctx context.Context, name string) error {
	_, err := c.agent.DeleteAgent(ctx, &controlplanev1.DeleteAgentRequest{Name: name})
	return err
}

func agentFromProto(pb *controlplanev1.Agent) *domain.Agent {
	if pb == nil || pb.Info == nil {
		return nil
	}

	a := &domain.Agent{
		Id:       pb.Info.Name,
		Hostname: pb.Info.Hostname,
		Version:  pb.Info.Version,
		Labels:   pb.Info.Labels,
	}
	if pb.CreatedAt != nil {
		a.CreatedAt = pb.CreatedAt.AsTime()
	}
	if pb.UpdatedAt != nil {
		a.UpdatedAt = pb.UpdatedAt.AsTime()
	}
	if pb.LastHeartbeat != nil {
		hb := pb.LastHeartbeat.AsTime()
		a.LastHeartbeat = &hb
	}
	return a
}
