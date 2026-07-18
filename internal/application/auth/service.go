package auth

import (
	"context"
	"crypto/subtle"
	"fmt"
	"sync"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const minPasswordLength = 8

// dummyHash keeps bcrypt cost constant on unknown-user login attempts so the
// response time does not reveal whether a username exists. Computed lazily so
// importing the package does not pay a bcrypt round at startup.
var dummyHash = sync.OnceValue(func() []byte {
	hash, _ := bcrypt.GenerateFromPassword([]byte("pgway-dummy-password"), bcrypt.DefaultCost)
	return hash
})

// Service owns user accounts and their sessions: bootstrap/init, login/logout
// and user CRUD. Token verification lives in Authenticator, agent credential
// mechanics in AgentCredentialService — one struct per port boundary.
type Service struct {
	users      ports.UserRepositoryPort
	tokens     ports.TokenRepositoryPort
	defaultTTL time.Duration

	mu             sync.Mutex
	bootstrapToken string
}

func NewService(
	users ports.UserRepositoryPort,
	tokens ports.TokenRepositoryPort,
	defaultTTL time.Duration,
) *Service {
	return &Service{
		users:      users,
		tokens:     tokens,
		defaultTTL: defaultTTL,
	}
}

// Bootstrap generates a one-time bootstrap token when no users exist yet.
// The token lives only in memory; a restart before init produces a new one.
func (s *Service) Bootstrap(ctx context.Context) error {
	count, err := s.users.Count(ctx)
	if err != nil {
		return fmt.Errorf("counting users: %w", err)
	}

	if count > 0 {
		return nil
	}

	token, err := generateToken()
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.bootstrapToken = token
	s.mu.Unlock()

	zap.L().Warn("no users found — initialize with: pgctl init --bootstrap-token <token>",
		zap.String("bootstrap_token", token))

	return nil
}

// BootstrapToken returns the pending bootstrap token, or empty when the
// system is initialized. Used by tests and status reporting.
func (s *Service) BootstrapToken() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bootstrapToken
}

func (s *Service) InitAdmin(ctx context.Context, bootstrapToken, username, password string) (*domain.User, string, error) {
	count, err := s.users.Count(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("counting users: %w", err)
	}

	if count > 0 {
		return nil, "", ErrAlreadyInitialized
	}

	s.mu.Lock()
	expected := s.bootstrapToken
	s.mu.Unlock()

	if expected == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(bootstrapToken)) != 1 {
		return nil, "", ErrInvalidBootstrapToken
	}

	user, err := s.createUser(ctx, username, password, domain.RoleAdmin)
	if err != nil {
		return nil, "", err
	}

	// bootstrap token is single-use
	s.mu.Lock()
	s.bootstrapToken = ""
	s.mu.Unlock()

	token, err := s.issueToken(ctx, user.Id, s.defaultTTL)
	if err != nil {
		return nil, "", err
	}

	zap.L().Info("admin user initialized", zap.String("username", user.Id))

	return user, token, nil
}

func (s *Service) Login(ctx context.Context, username, password string, ttl time.Duration, noExpiry bool) (string, error) {
	user, err := s.users.Find(ctx, username)
	if err != nil {
		// burn a bcrypt round anyway, see dummyHash
		_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(password))
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	if noExpiry {
		ttl = 0
	} else if ttl <= 0 {
		ttl = s.defaultTTL
	}

	return s.issueToken(ctx, user.Id, ttl)
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if err := s.tokens.Delete(ctx, hashToken(token)); err != nil {
		return ErrInvalidToken
	}
	return nil
}

func (s *Service) CreateUser(ctx context.Context, username, password string, role domain.Role) (*domain.User, string, error) {
	generated := ""

	if password == "" {
		generated = generatePassword()
		password = generated
	}

	if role == "" {
		role = domain.RoleMember
	}

	user, err := s.createUser(ctx, username, password, role)
	if err != nil {
		return nil, "", err
	}

	zap.L().Info("user created", zap.String("username", user.Id), zap.String("role", string(user.Role)))

	return user, generated, nil
}

func (s *Service) ListUsers(ctx context.Context, params domain.ListParams, filter domain.UserFilter) (domain.ListResult[domain.User], error) {
	if params.PageSize > domain.DefaultMaxPageSize {
		params.PageSize = domain.DefaultMaxPageSize
	}
	return s.users.List(ctx, params, filter)
}

func (s *Service) DeleteUser(ctx context.Context, username string) error {
	user, err := s.users.Find(ctx, username)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}

	if user.IsAdmin() {
		admins, err := s.users.List(ctx, domain.ListParams{}, domain.UserFilter{Role: string(domain.RoleAdmin)})
		if err != nil {
			return fmt.Errorf("counting admins: %w", err)
		}
		if admins.TotalCount <= 1 {
			return ErrLastAdmin
		}
	}

	if err := s.users.Delete(ctx, username); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	if err := s.tokens.DeleteByUserId(ctx, username); err != nil {
		return fmt.Errorf("revoking tokens: %w", err)
	}

	zap.L().Info("user deleted", zap.String("username", username))

	return nil
}

func (s *Service) ChangePassword(ctx context.Context, username, oldPassword, newPassword string) error {
	user, err := s.users.Find(ctx, username)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	return s.setPassword(ctx, user, newPassword)
}

func (s *Service) ResetPassword(ctx context.Context, username, newPassword string) error {
	user, err := s.users.Find(ctx, username)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}

	return s.setPassword(ctx, user, newPassword)
}

func (s *Service) setPassword(ctx context.Context, user *domain.User, newPassword string) error {
	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}

	user.PasswordHash = hash
	user.UpdatedAt = time.Now()

	if err := s.users.Save(ctx, user); err != nil {
		return fmt.Errorf("persisting user: %w", err)
	}

	// a password change invalidates every session of the user
	if err := s.tokens.DeleteByUserId(ctx, user.Id); err != nil {
		return fmt.Errorf("revoking tokens: %w", err)
	}

	zap.L().Info("password changed", zap.String("username", user.Id))

	return nil
}

func (s *Service) createUser(ctx context.Context, username, password string, role domain.Role) (*domain.User, error) {
	if _, err := s.users.Find(ctx, username); err == nil {
		return nil, ErrUserExists
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &domain.User{
		Id:           username,
		PasswordHash: hash,
		Role:         role,
	}
	user.CreatedAt = now
	user.UpdatedAt = now

	if err := user.Validate(); err != nil {
		return nil, fmt.Errorf("user validation: %w", err)
	}

	if err := s.users.Save(ctx, user); err != nil {
		return nil, fmt.Errorf("persisting user: %w", err)
	}

	return user, nil
}

func (s *Service) issueToken(ctx context.Context, userId string, ttl time.Duration) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	now := time.Now()
	record := &domain.Token{
		Hash:   hashToken(token),
		UserId: userId,
	}
	record.CreatedAt = now
	record.UpdatedAt = now

	if ttl > 0 {
		expires := now.Add(ttl)
		record.ExpiresAt = &expires
	}

	if err := record.Validate(); err != nil {
		return "", ErrInvalidToken
	}

	if err := s.tokens.Save(ctx, record); err != nil {
		return "", fmt.Errorf("persisting token: %w", err)
	}

	return token, nil
}

func hashPassword(password string) (string, error) {
	if len(password) < minPasswordLength {
		return "", ErrWeakPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}

	return string(hash), nil
}

var _ ports.UserManager = (*Service)(nil)
var _ ports.AuthManager = (*Service)(nil)
