package client

import (
	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/schema"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	proxy      controlplanev1.ProxyServiceClient
	pool       controlplanev1.PoolServiceClient
	balancer   controlplanev1.BalancerServiceClient
	router     controlplanev1.RouterServiceClient
	flow       controlplanev1.FlowServiceClient
	entrypoint controlplanev1.EntrypointServiceClient

	conn *grpc.ClientConn
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		conn:       conn,
	}, nil
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
