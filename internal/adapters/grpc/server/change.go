package server

import (
	"context"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/ports"
	"google.golang.org/grpc"
)

// ChangeServer bridges the in-process event bus to Watch streams.
// Slow consumers are already soft-dropped by the memory bus buffer.
type ChangeServer struct {
	controlplanev1.UnimplementedChangeServiceServer
	events   ports.EventSubscriberPort
	shutdown context.Context
}

func NewChangeServer(events ports.EventSubscriberPort, shutdown context.Context) *ChangeServer {
	if shutdown == nil {
		shutdown = context.Background()
	}
	return &ChangeServer{events: events, shutdown: shutdown}
}

func RegisterChange(s grpc.ServiceRegistrar, srv *ChangeServer) {
	controlplanev1.RegisterChangeServiceServer(s, srv)
}

func (s *ChangeServer) Watch(_ *controlplanev1.WatchRequest, stream controlplanev1.ChangeService_WatchServer) error {
	// Merge stream lifetime with process shutdown. Without this,
	// GracefulStop blocks forever on open Watch streams: it waits for
	// handlers to return, while Watch waits for the stream context that
	// GracefulStop does not cancel.
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()
	stop := context.AfterFunc(s.shutdown, cancel)
	defer stop()

	ch := s.events.Subscribe(ctx)

	for e := range ch {
		if err := stream.Send(changeEventToProto(e)); err != nil {
			return err
		}
	}
	return nil
}

func changeEventToProto(e ports.ChangeEvent) *controlplanev1.ResourceChanged {
	return &controlplanev1.ResourceChanged{
		ResourceType: string(e.ResourceType),
		ChangeKind:   string(e.ChangeKind),
		Id:           e.ID,
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
