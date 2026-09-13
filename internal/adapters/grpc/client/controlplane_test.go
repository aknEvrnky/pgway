package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestUnaryBearerAttachesToken(t *testing.T) {
	c := &Client{}
	c.token.Store("")

	var gotCtx context.Context
	invoker := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		gotCtx = ctx
		return nil
	}

	require.NoError(t, c.unaryBearer(context.Background(), "/svc/Method", nil, nil, nil, invoker))
	md, ok := metadata.FromOutgoingContext(gotCtx)
	if ok {
		assert.Empty(t, md.Get("authorization"))
	}

	c.SetToken("agent-tok")
	require.NoError(t, c.unaryBearer(context.Background(), "/svc/Method", nil, nil, nil, invoker))
	md, ok = metadata.FromOutgoingContext(gotCtx)
	require.True(t, ok)
	assert.Equal(t, []string{"Bearer agent-tok"}, md.Get("authorization"))

	c.SetToken("rotated")
	require.NoError(t, c.unaryBearer(context.Background(), "/svc/Method", nil, nil, nil, invoker))
	md, ok = metadata.FromOutgoingContext(gotCtx)
	require.True(t, ok)
	assert.Equal(t, []string{"Bearer rotated"}, md.Get("authorization"))
}

func TestStreamBearerAttachesToken(t *testing.T) {
	c := &Client{}
	c.token.Store("stream-tok")

	var gotCtx context.Context
	streamer := func(ctx context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
		gotCtx = ctx
		return nil, nil
	}

	_, err := c.streamBearer(context.Background(), &grpc.StreamDesc{}, nil, "/svc/Watch", streamer)
	require.NoError(t, err)
	md, ok := metadata.FromOutgoingContext(gotCtx)
	require.True(t, ok)
	assert.Equal(t, []string{"Bearer stream-tok"}, md.Get("authorization"))
}

func TestTokenAccessors(t *testing.T) {
	c := &Client{}
	c.token.Store("")
	assert.Equal(t, "", c.Token())

	c.SetToken("abc")
	assert.Equal(t, "abc", c.Token())
}
