package client

import (
	"context"
	"io"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/event"
	"github.com/aknEvrnky/pgway/internal/ports"
)

// Watch opens ChangeService.Watch and republishes each hint onto pub until
// the stream ends or ctx is canceled.
func (c *Client) Watch(ctx context.Context, pub ports.EventPublisherPort) error {
	stream, err := c.change.Watch(ctx, &controlplanev1.WatchRequest{})
	if err != nil {
		return err
	}

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF || ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		if err := pub.Publish(ctx, changeEventFromProto(msg)); err != nil {
			return err
		}
	}
}

func changeEventFromProto(pb *controlplanev1.ResourceChanged) event.ChangeEvent {
	if pb == nil {
		return event.ChangeEvent{}
	}
	return event.ChangeEvent{
		ID:           pb.Id,
		ResourceType: event.ResourceType(pb.ResourceType),
		ChangeKind:   event.ChangeKind(pb.ChangeKind),
	}
}
