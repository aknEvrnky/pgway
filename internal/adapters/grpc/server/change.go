package server

import (
	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/event"
	"github.com/aknEvrnky/pgway/internal/ports"
	"google.golang.org/grpc"
)

// ChangeServer bridges the in-process event bus to Watch streams.
// Slow consumers are already soft-dropped by the memory bus buffer.
type ChangeServer struct {
	controlplanev1.UnimplementedChangeServiceServer
	events ports.EventSubscriberPort
}

func NewChangeServer(events ports.EventSubscriberPort) *ChangeServer {
	return &ChangeServer{events: events}
}

func RegisterChange(s grpc.ServiceRegistrar, srv *ChangeServer) {
	controlplanev1.RegisterChangeServiceServer(s, srv)
}

func (s *ChangeServer) Watch(_ *controlplanev1.WatchRequest, stream controlplanev1.ChangeService_WatchServer) error {
	ctx := stream.Context()
	ch := s.events.Subscribe(ctx)

	for e := range ch {
		if err := stream.Send(changeEventToProto(e)); err != nil {
			return err
		}
	}
	return nil
}

func changeEventToProto(e event.ChangeEvent) *controlplanev1.ResourceChanged {
	return &controlplanev1.ResourceChanged{
		ResourceType: string(e.ResourceType),
		ChangeKind:   string(e.ChangeKind),
		Id:           e.ID,
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
