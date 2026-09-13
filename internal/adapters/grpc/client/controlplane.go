package client

import (
	"context"
	"sync/atomic"

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
	agent      controlplanev1.AgentServiceClient
	change     controlplanev1.ChangeServiceClient

	conn  *grpc.ClientConn
	token atomic.Value // string
}

// NewClient connects to the control plane. token authenticates every call;
// it may be empty for the exempt RPCs (Login, InitAdmin, Register). Use
// SetToken to swap credentials after Register without redialing.
func NewClient(addr, token string) (*Client, error) {
	c := &Client{}
	c.token.Store(token)

	conn, err := grpc.NewClient(
		"passthrough:///"+addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(c.unaryBearer),
		grpc.WithStreamInterceptor(c.streamBearer),
	)
	if err != nil {
		return nil, err
	}

	c.proxy = controlplanev1.NewProxyServiceClient(conn)
	c.pool = controlplanev1.NewPoolServiceClient(conn)
	c.balancer = controlplanev1.NewBalancerServiceClient(conn)
	c.router = controlplanev1.NewRouterServiceClient(conn)
	c.flow = controlplanev1.NewFlowServiceClient(conn)
	c.entrypoint = controlplanev1.NewEntrypointServiceClient(conn)
	c.user = controlplanev1.NewUserServiceClient(conn)
	c.auth = controlplanev1.NewAuthServiceClient(conn)
	c.agent = controlplanev1.NewAgentServiceClient(conn)
	c.change = controlplanev1.NewChangeServiceClient(conn)
	c.conn = conn
	return c, nil
}

// SetToken replaces the bearer used by subsequent unary and stream calls.
func (c *Client) SetToken(token string) {
	c.token.Store(token)
}

// Token returns the current bearer (may be empty).
func (c *Client) Token() string {
	v, _ := c.token.Load().(string)
	return v
}

func (c *Client) currentToken() string {
	v, _ := c.token.Load().(string)
	return v
}

func (c *Client) unaryBearer(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if tok := c.currentToken(); tok != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+tok)
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}

func (c *Client) streamBearer(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	if tok := c.currentToken(); tok != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+tok)
	}
	return streamer(ctx, desc, cc, method, opts...)
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
