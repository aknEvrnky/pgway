package client

import (
	"context"
	"io"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/ports"
)

// Watch opens ChangeService.Watch. Once the stream is established (server
// subscription active), afterConnect runs — typically a Bootstrap resync —
// then each hint is republished onto pub until the stream ends or ctx cancels.
func (c *Client) Watch(ctx context.Context, pub ports.EventPublisherPort, afterConnect func(context.Context) error) error {
	stream, err := c.change.Watch(ctx, &controlplanev1.WatchRequest{})
	if err != nil {
		return mapAgentAuth(err)
	}

	if afterConnect != nil {
		if err := afterConnect(ctx); err != nil {
			return mapAgentAuth(err)
		}
	}

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF || ctx.Err() != nil {
				return ctx.Err()
			}
			return mapAgentAuth(err)
		}
		if err := pub.Publish(ctx, changeEventFromProto(msg)); err != nil {
			return err
		}
	}
}

func changeEventFromProto(pb *controlplanev1.ResourceChanged) ports.ChangeEvent {
	if pb == nil {
		return ports.ChangeEvent{}
	}
	return ports.ChangeEvent{
		ID:           pb.Id,
		ResourceType: ports.ResourceType(pb.ResourceType),
		ChangeKind:   ports.ChangeKind(pb.ChangeKind),
	}
}
