package server

import (
	"context"
	"errors"
	"time"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/auth"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthServer struct {
	controlplanev1.UnimplementedUserServiceServer
	controlplanev1.UnimplementedAuthServiceServer

	users ports.UserManager
	auth  ports.AuthManager
}

func NewAuthServer(users ports.UserManager, authManager ports.AuthManager) *AuthServer {
	return &AuthServer{
		users: users,
		auth:  authManager,
	}
}

// --- AuthService ---

func (s *AuthServer) InitAdmin(ctx context.Context, req *controlplanev1.InitAdminRequest) (*controlplanev1.InitAdminResponse, error) {
	if req.Username == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password are required")
	}

	user, token, err := s.auth.InitAdmin(ctx, req.BootstrapToken, req.Username, req.Password)
	if err != nil {
		return nil, mapAuthError("init admin", err)
	}

	return &controlplanev1.InitAdminResponse{
		User:  userToProto(user),
		Token: token,
	}, nil
}

func (s *AuthServer) Login(ctx context.Context, req *controlplanev1.LoginRequest) (*controlplanev1.LoginResponse, error) {
	if req.Username == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password are required")
	}

	ttl := time.Duration(req.TtlSeconds) * time.Second

	token, err := s.auth.Login(ctx, req.Username, req.Password, ttl, req.NoExpiry)
	if err != nil {
		return nil, mapAuthError("login", err)
	}

	return &controlplanev1.LoginResponse{Token: token}, nil
}

func (s *AuthServer) Logout(ctx context.Context, req *controlplanev1.LogoutRequest) (*controlplanev1.LogoutResponse, error) {
	token, ok := auth.TokenFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no token in call context")
	}

	if err := s.auth.Logout(ctx, token); err != nil {
		return nil, mapAuthError("logout", err)
	}

	return &controlplanev1.LogoutResponse{}, nil
}

// --- UserService ---

func (s *AuthServer) CreateUser(ctx context.Context, req *controlplanev1.CreateUserRequest) (*controlplanev1.CreateUserResponse, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "username is required")
	}

	user, generated, err := s.users.CreateUser(ctx, req.Username, req.Password, domain.Role(req.Role))
	if err != nil {
		return nil, mapAuthError("create user", err)
	}

	return &controlplanev1.CreateUserResponse{
		User:              userToProto(user),
		GeneratedPassword: generated,
	}, nil
}

func (s *AuthServer) ListUsers(ctx context.Context, req *controlplanev1.ListUsersRequest) (*controlplanev1.ListUsersResponse, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	cursor, err := decodeCursor(req.GetPageToken())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid page_token")
	}

	params := domain.ListParams{
		PageSize: int(req.GetPageSize()),
		Cursor:   cursor,
	}

	filter := domain.UserFilter{
		Search: req.GetSearch(),
		Role:   req.GetRole(),
	}

	result, err := s.users.ListUsers(ctx, params, filter)
	if err != nil {
		return nil, mapAuthError("list users", err)
	}

	users := make([]*controlplanev1.User, 0, len(result.Items))
	for _, u := range result.Items {
		users = append(users, userToProto(u))
	}

	return &controlplanev1.ListUsersResponse{
		Users:         users,
		NextPageToken: encodeCursor(result.NextCursor),
		TotalCount:    int32(result.TotalCount),
	}, nil
}

func (s *AuthServer) DeleteUser(ctx context.Context, req *controlplanev1.DeleteUserRequest) (*controlplanev1.DeleteUserResponse, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "username is required")
	}

	if err := s.users.DeleteUser(ctx, req.Username); err != nil {
		return nil, mapAuthError("delete user", err)
	}

	return &controlplanev1.DeleteUserResponse{}, nil
}

func (s *AuthServer) ChangePassword(ctx context.Context, req *controlplanev1.ChangePasswordRequest) (*controlplanev1.ChangePasswordResponse, error) {
	actor, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no user in call context")
	}

	if req.NewPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "new_password is required")
	}

	target := req.Username
	if target == "" {
		target = actor.Id
	}

	// resetting someone else's password is an admin operation and skips the
	// old-password check
	if target != actor.Id {
		if !actor.IsAdmin() {
			return nil, status.Error(codes.PermissionDenied, "admin role required")
		}

		if err := s.users.ResetPassword(ctx, target, req.NewPassword); err != nil {
			return nil, mapAuthError("reset password", err)
		}

		return &controlplanev1.ChangePasswordResponse{}, nil
	}

	if req.OldPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "old_password is required")
	}

	if err := s.users.ChangePassword(ctx, target, req.OldPassword, req.NewPassword); err != nil {
		return nil, mapAuthError("change password", err)
	}

	return &controlplanev1.ChangePasswordResponse{}, nil
}

// --- helpers ---

func requireAdmin(ctx context.Context) error {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "no user in call context")
	}

	if !user.IsAdmin() {
		return status.Error(codes.PermissionDenied, "admin role required")
	}

	return nil
}

func mapAuthError(op string, err error) error {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials),
		errors.Is(err, auth.ErrInvalidToken),
		errors.Is(err, auth.ErrInvalidBootstrapToken):
		return status.Errorf(codes.Unauthenticated, "%s: %v", op, err)
	case errors.Is(err, auth.ErrAlreadyInitialized),
		errors.Is(err, auth.ErrLastAdmin):
		return status.Errorf(codes.FailedPrecondition, "%s: %v", op, err)
	case errors.Is(err, auth.ErrUserExists):
		return status.Errorf(codes.AlreadyExists, "%s: %v", op, err)
	case errors.Is(err, auth.ErrWeakPassword):
		return status.Errorf(codes.InvalidArgument, "%s: %v", op, err)
	default:
		return status.Errorf(codes.Internal, "%s: %v", op, err)
	}
}

func userToProto(u *domain.User) *controlplanev1.User {
	return &controlplanev1.User{
		Id:        u.Id,
		Role:      string(u.Role),
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
}
