package client

import (
	"context"
	"time"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

func (c *Client) InitAdmin(ctx context.Context, bootstrapToken, username, password string) (*domain.User, string, error) {
	resp, err := c.auth.InitAdmin(ctx, &controlplanev1.InitAdminRequest{
		BootstrapToken: bootstrapToken,
		Username:       username,
		Password:       password,
	})
	if err != nil {
		return nil, "", err
	}

	return userFromProto(resp.User), resp.Token, nil
}

func (c *Client) Login(ctx context.Context, username, password string, ttl time.Duration, noExpiry bool) (string, error) {
	resp, err := c.auth.Login(ctx, &controlplanev1.LoginRequest{
		Username:   username,
		Password:   password,
		TtlSeconds: int64(ttl / time.Second),
		NoExpiry:   noExpiry,
	})
	if err != nil {
		return "", err
	}

	return resp.Token, nil
}

// Logout revokes the credential this client was dialed with; the server
// resolves the token from the call metadata, so the argument is not sent.
func (c *Client) Logout(ctx context.Context, _ string) error {
	_, err := c.auth.Logout(ctx, &controlplanev1.LogoutRequest{})
	return err
}
