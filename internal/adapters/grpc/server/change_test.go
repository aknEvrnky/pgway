package server

import (
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/ports"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestChangeEventProtoRoundTrip(t *testing.T) {
	in := ports.ChangeEvent{
		ID:           "ep-1",
		ResourceType: ports.ResourceTypeEntrypoint,
		ChangeKind:   ports.ChangeKindSaved,
	}
	out := changeEventFromProto(changeEventToProto(in))
	assert.Equal(t, in, out)
}

func TestChangeEventFromProtoNil(t *testing.T) {
	assert.Equal(t, ports.ChangeEvent{}, changeEventFromProto(nil))
}

func TestGracefulStopWithTimeoutEmptyServer(t *testing.T) {
	s := grpc.NewServer()
	done := make(chan struct{})
	go func() {
		GracefulStopWithTimeout(s, 100*time.Millisecond)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}
