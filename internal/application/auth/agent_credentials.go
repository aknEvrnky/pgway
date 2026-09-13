package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
)

type AgentCredentialService struct {
	tokens    ports.TokenRepositoryPort
	regTokens ports.RegistrationTokenRepositoryPort
}

func (s *AgentCredentialService) CreateRegistrationToken(ctx context.Context, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		return "", fmt.Errorf("TTL must be greater than 0")
	}

	token, err := generateRegistrationToken()

	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	hash := hashToken(token)
	expires := time.Now().Add(ttl)

	regToken := &domain.RegistrationToken{
		Hash:      hash,
		ExpiresAt: &expires,
	}

	regToken.CreatedAt = time.Now()
	regToken.UpdatedAt = time.Now()

	err = s.regTokens.Save(ctx, regToken)

	if err != nil {
		return "", fmt.Errorf("persist registration token: %w", err)
	}

	return token, nil
}

func (s *AgentCredentialService) ConsumeRegistrationToken(ctx context.Context, token string) error {
	hashedToken := hashToken(token)
	regToken, err := s.regTokens.Consume(ctx, hashedToken)

	if err != nil {
		return ErrInvalidRegistrationToken
	}

	if regToken.IsExpired(time.Now()) {
		return ErrInvalidRegistrationToken
	}

	return nil
}

func (s *AgentCredentialService) IssueAgentToken(ctx context.Context, agentId string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		return "", fmt.Errorf("TTL must be greater than 0")
	}

	tokenStr, err := generateToken()

	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	hash := hashToken(tokenStr)
	expires := time.Now().Add(ttl)

	token := &domain.Token{
		Hash:      hash,
		AgentId:   agentId,
		ExpiresAt: &expires,
	}

	token.CreatedAt = time.Now()
	token.UpdatedAt = time.Now()

	err = token.Validate()

	if err != nil {
		return "", ErrInvalidToken
	}

	err = s.tokens.Save(ctx, token)

	if err != nil {
		return "", fmt.Errorf("persist token: %w", err)
	}

	return tokenStr, nil
}

func (s *AgentCredentialService) ExtendAgentToken(ctx context.Context, rawToken string, ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("TTL must be greater than 0")
	}

	hash := hashToken(rawToken)
	record, err := s.tokens.Find(ctx, hash)

	if err != nil || record.IsExpired(time.Now()) {
		return ErrInvalidToken
	}

	if record.AgentId == "" {
		return ErrInvalidToken
	}

	expires := time.Now().Add(ttl)
	record.ExpiresAt = &expires
	record.UpdatedAt = time.Now()

	err = s.tokens.Save(ctx, record)

	if err != nil {
		return fmt.Errorf("persist token: %w", err)
	}

	return nil
}

func (s *AgentCredentialService) RevokeAgentTokens(ctx context.Context, agentId string) error {
	if agentId == "" {
		return fmt.Errorf("agentID cannot be empty")
	}

	err := s.tokens.DeleteByAgentId(ctx, agentId)

	if err != nil {
		return fmt.Errorf("delete agent tokens: %w", err)
	}

	return nil
}

func NewAgentCredentialService(
	tokens ports.TokenRepositoryPort,
	regTokens ports.RegistrationTokenRepositoryPort,
) *AgentCredentialService {
	return &AgentCredentialService{
		tokens:    tokens,
		regTokens: regTokens,
	}
}

var _ ports.AgentCredentials = (*AgentCredentialService)(nil)
