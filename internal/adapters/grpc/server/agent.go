package server

import (
	"context"
	"errors"
	"time"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/agent"
	"github.com/aknEvrnky/pgway/internal/application/auth"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AgentServer struct {
	controlplanev1.UnimplementedAgentServiceServer

	agents               ports.AgentManager
	heartbeatThreshold   time.Duration
	agentTokenTTL        time.Duration
	registrationTokenTTL time.Duration
}

func NewAgentServer(
	agents ports.AgentManager,
	heartbeatThreshold time.Duration,
	agentTokenTTL time.Duration,
	registrationTokenTTL time.Duration,
) *AgentServer {
	return &AgentServer{
		agents:               agents,
		heartbeatThreshold:   heartbeatThreshold,
		agentTokenTTL:        agentTokenTTL,
		registrationTokenTTL: registrationTokenTTL,
	}
}

func (s *AgentServer) CreateRegistrationToken(ctx context.Context, req *controlplanev1.CreateRegistrationTokenRequest) (*controlplanev1.CreateRegistrationTokenResponse, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	ttl := time.Duration(req.TtlSeconds) * time.Second
	if ttl <= 0 {
		ttl = s.registrationTokenTTL
	}

	token, err := s.agents.CreateRegistrationToken(ctx, ttl)
	if err != nil {
		return nil, mapAgentError("create registration token", err)
	}

	return &controlplanev1.CreateRegistrationTokenResponse{
		RegistrationToken: token,
		ExpiresAt:         timestamppb.New(time.Now().Add(ttl)),
	}, nil
}

func (s *AgentServer) Register(ctx context.Context, req *controlplanev1.RegisterRequest) (*controlplanev1.RegisterResponse, error) {
	if req.RegistrationToken == "" {
		return nil, status.Error(codes.InvalidArgument, "registration_token is required")
	}
	if req.Info == nil || req.Info.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "info.name is required")
	}

	registered, agentToken, err := s.agents.Register(ctx, req.RegistrationToken, domain.Agent{
		Id:       req.Info.Name,
		Hostname: req.Info.Hostname,
		Version:  req.Info.Version,
		Labels:   req.Info.Labels,
	})
	if err != nil {
		return nil, mapAgentError("register", err)
	}

	return &controlplanev1.RegisterResponse{
		AgentId:        registered.Id,
		AgentToken:     agentToken,
		TokenExpiresAt: timestamppb.New(time.Now().Add(s.agentTokenTTL)),
	}, nil
}

func (s *AgentServer) Heartbeat(ctx context.Context, _ *controlplanev1.HeartbeatRequest) (*controlplanev1.HeartbeatResponse, error) {
	if err := requireAgent(ctx); err != nil {
		return nil, err
	}

	expires, err := s.agents.Heartbeat(ctx)
	if err != nil {
		return nil, mapAgentError("heartbeat", err)
	}

	if p, ok := ports.PrincipalFromContext(ctx); ok && p.Agent != nil {
		zap.L().Info("agent heartbeat",
			zap.String("agent_id", p.Agent.Id),
			zap.Time("token_expires_at", expires),
		)
	}

	return &controlplanev1.HeartbeatResponse{
		TokenExpiresAt: timestamppb.New(expires),
	}, nil
}

func (s *AgentServer) Deregister(ctx context.Context, _ *controlplanev1.DeregisterRequest) (*controlplanev1.DeregisterResponse, error) {
	if err := requireAgent(ctx); err != nil {
		return nil, err
	}

	if err := s.agents.Deregister(ctx); err != nil {
		return nil, mapAgentError("deregister", err)
	}

	return &controlplanev1.DeregisterResponse{}, nil
}

func (s *AgentServer) ListAgents(ctx context.Context, req *controlplanev1.ListAgentsRequest) (*controlplanev1.ListAgentsResponse, error) {
	if err := requireUser(ctx); err != nil {
		return nil, err
	}

	params := domain.ListParams{
		PageSize: int(req.PageSize),
		Cursor:   req.PageToken,
	}
	filter := domain.AgentFilter{
		Search: req.Search,
		Labels: req.Labels,
	}

	result, err := s.agents.ListAgents(ctx, params, filter)
	if err != nil {
		return nil, mapAgentError("list agents", err)
	}

	now := time.Now()
	agents := make([]*controlplanev1.Agent, 0, len(result.Items))
	for _, a := range result.Items {
		agents = append(agents, agentToProto(a, now, s.heartbeatThreshold))
	}

	return &controlplanev1.ListAgentsResponse{
		Agents:        agents,
		NextPageToken: result.NextCursor,
		TotalCount:    int32(result.TotalCount),
	}, nil
}

func (s *AgentServer) DeleteAgent(ctx context.Context, req *controlplanev1.DeleteAgentRequest) (*controlplanev1.DeleteAgentResponse, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	if err := s.agents.DeleteAgent(ctx, req.Name); err != nil {
		return nil, mapAgentError("delete agent", err)
	}

	return &controlplanev1.DeleteAgentResponse{}, nil
}

func requireAgent(ctx context.Context) error {
	principal, ok := ports.PrincipalFromContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "no principal in call context")
	}
	if principal.Agent == nil {
		return status.Error(codes.PermissionDenied, "agent principal required")
	}
	return nil
}

func requireUser(ctx context.Context) error {
	principal, ok := ports.PrincipalFromContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "no principal in call context")
	}
	if principal.User == nil {
		return status.Error(codes.PermissionDenied, "user principal required")
	}
	return nil
}

func mapAgentError(op string, err error) error {
	switch {
	case errors.Is(err, auth.ErrInvalidRegistrationToken),
		errors.Is(err, auth.ErrInvalidToken),
		errors.Is(err, agent.ErrTokenRequired):
		return status.Errorf(codes.Unauthenticated, "%s: %v", op, err)
	case errors.Is(err, agent.ErrAgentRequired):
		return status.Errorf(codes.PermissionDenied, "%s: %v", op, err)
	case errors.Is(err, domain.ErrAgentExists):
		return status.Errorf(codes.AlreadyExists, "%s: %v", op, err)
	case errors.Is(err, agent.ErrAgentNotFound):
		return status.Errorf(codes.NotFound, "%s: %v", op, err)
	default:
		return status.Errorf(codes.Internal, "%s: %v", op, err)
	}
}

func agentToProto(a *domain.Agent, now time.Time, threshold time.Duration) *controlplanev1.Agent {
	pb := &controlplanev1.Agent{
		Info: &controlplanev1.AgentInfo{
			Name:     a.Id,
			Hostname: a.Hostname,
			Version:  a.Version,
			Labels:   a.Labels,
		},
		Status:    string(a.Status(now, threshold)),
		CreatedAt: timestamppb.New(a.CreatedAt),
		UpdatedAt: timestamppb.New(a.UpdatedAt),
	}
	if a.LastHeartbeat != nil {
		pb.LastHeartbeat = timestamppb.New(*a.LastHeartbeat)
	}
	return pb
}
