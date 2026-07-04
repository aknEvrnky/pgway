package client

import (
	"context"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

func (c *Client) CreateUser(ctx context.Context, username, password string, role domain.Role) (*domain.User, string, error) {
	resp, err := c.user.CreateUser(ctx, &controlplanev1.CreateUserRequest{
		Username: username,
		Password: password,
		Role:     string(role),
	})
	if err != nil {
		return nil, "", err
	}

	return userFromProto(resp.User), resp.GeneratedPassword, nil
}

func (c *Client) ListUsers(ctx context.Context, params domain.ListParams, filter domain.UserFilter) (domain.ListResult[domain.User], error) {
	resp, err := c.user.ListUsers(ctx, &controlplanev1.ListUsersRequest{
		PageSize:  int32(params.PageSize),
		PageToken: params.Cursor,
		Search:    filter.Search,
		Role:      filter.Role,
	})
	if err != nil {
		return domain.ListResult[domain.User]{}, err
	}

	items := make([]*domain.User, 0, len(resp.Users))
	for _, u := range resp.Users {
		items = append(items, userFromProto(u))
	}

	return domain.ListResult[domain.User]{
		Items:      items,
		NextCursor: resp.NextPageToken,
		TotalCount: int(resp.TotalCount),
	}, nil
}

func (c *Client) DeleteUser(ctx context.Context, username string) error {
	_, err := c.user.DeleteUser(ctx, &controlplanev1.DeleteUserRequest{Username: username})
	return err
}

func (c *Client) ChangePassword(ctx context.Context, username, oldPassword, newPassword string) error {
	_, err := c.user.ChangePassword(ctx, &controlplanev1.ChangePasswordRequest{
		Username:    username,
		OldPassword: oldPassword,
		NewPassword: newPassword,
	})
	return err
}

// ResetPassword maps to the same RPC; the server treats a change targeting
// another user as an admin reset.
func (c *Client) ResetPassword(ctx context.Context, username, newPassword string) error {
	_, err := c.user.ChangePassword(ctx, &controlplanev1.ChangePasswordRequest{
		Username:    username,
		NewPassword: newPassword,
	})
	return err
}

func userFromProto(pb *controlplanev1.User) *domain.User {
	if pb == nil {
		return nil
	}

	user := &domain.User{
		Id:   pb.Id,
		Role: domain.Role(pb.Role),
	}
	user.CreatedAt = pb.CreatedAt.AsTime()
	user.UpdatedAt = pb.UpdatedAt.AsTime()

	return user
}
