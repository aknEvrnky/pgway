package auth

import (
	"context"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
)

// Authenticator resolves bearer tokens to principals. It is the single
// verification point shared by every transport (the gRPC interceptor today, a
// REST middleware tomorrow). Issuance lives elsewhere: Service mints user
// tokens, AgentCredentialService mints agent tokens.
type Authenticator struct {
	users  ports.UserRepositoryPort
	agents ports.AgentRepositoryPort
	tokens ports.TokenRepositoryPort
}

func NewAuthenticator(
	users ports.UserRepositoryPort,
	agents ports.AgentRepositoryPort,
	tokens ports.TokenRepositoryPort,
) *Authenticator {
	return &Authenticator{
		users:  users,
		agents: agents,
		tokens: tokens,
	}
}

// Authenticate resolves a bearer token to its principal. Tokens that are
// expired or whose subject no longer exists (deleted user or agent) are
// deleted on sight and rejected.
func (a *Authenticator) Authenticate(ctx context.Context, token string) (*domain.Principal, error) {
	if token == "" {
		return nil, ErrInvalidToken
	}

	record, err := a.tokens.Find(ctx, hashToken(token))
	if err != nil {
		return nil, ErrInvalidToken
	}

	if record.IsExpired(time.Now()) {
		_ = a.tokens.Delete(ctx, record.Hash)
		return nil, ErrInvalidToken
	}

	if record.UserId != "" {
		user, err := a.users.Find(ctx, record.UserId)
		if err != nil {
			// user deleted while token still stored
			_ = a.tokens.Delete(ctx, record.Hash)
			return nil, ErrInvalidToken
		}

		return &domain.Principal{User: user}, nil
	}

	if record.AgentId != "" {
		agent, err := a.agents.Find(ctx, record.AgentId)
		if err != nil {
			// agent deleted while token still stored
			_ = a.tokens.Delete(ctx, record.Hash)
			return nil, ErrInvalidToken
		}

		return &domain.Principal{Agent: agent}, nil
	}

	// a token without a subject should be impossible; treat it as garbage
	_ = a.tokens.Delete(ctx, record.Hash)
	return nil, ErrInvalidToken
}

var _ ports.TokenAuthenticator = (*Authenticator)(nil)
