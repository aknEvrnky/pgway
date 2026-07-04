package client

import (
	"context"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/schema"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type Client struct {
	proxy      controlplanev1.ProxyServiceClient
	pool       controlplanev1.PoolServiceClient
	balancer   controlplanev1.BalancerServiceClient
	router     controlplanev1.RouterServiceClient
	flow       controlplanev1.FlowServiceClient
	entrypoint controlplanev1.EntrypointServiceClient
	user       controlplanev1.UserServiceClient
	auth       controlplanev1.AuthServiceClient

	conn *grpc.ClientConn
}

// NewClient connects to the control plane. token authenticates every call;
// it may be empty for the exempt RPCs (Login, InitAdmin).
func NewClient(addr, token string) (*Client, error) {
	conn, err := grpc.NewClient(
		"passthrough:///"+addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(bearerInterceptor(token)),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		proxy:      controlplanev1.NewProxyServiceClient(conn),
		pool:       controlplanev1.NewPoolServiceClient(conn),
		balancer:   controlplanev1.NewBalancerServiceClient(conn),
		router:     controlplanev1.NewRouterServiceClient(conn),
		flow:       controlplanev1.NewFlowServiceClient(conn),
		entrypoint: controlplanev1.NewEntrypointServiceClient(conn),
		user:       controlplanev1.NewUserServiceClient(conn),
		auth:       controlplanev1.NewAuthServiceClient(conn),
		conn:       conn,
	}, nil
}

// bearerInterceptor attaches the bearer token to every outgoing call.
func bearerInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if token != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func metaToProto(meta schema.Metadata) *controlplanev1.Metadata {
	return &controlplanev1.Metadata{
		Name:   meta.Name,
		Labels: meta.Labels,
	}
}
