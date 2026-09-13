package server

import (
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/event"
	"github.com/stretchr/testify/assert"
)

// changeEventFromProto is tested via client change mapping in change_test.go.
func TestChangeEventProtoRoundTrip(t *testing.T) {
	in := event.ChangeEvent{
		ID:           "ep-1",
		ResourceType: event.ResourceTypeEntrypoint,
		ChangeKind:   event.ChangeKindSaved,
	}
	out := changeEventFromProto(changeEventToProto(in))
	assert.Equal(t, in, out)
}

func TestChangeEventFromProtoNil(t *testing.T) {
	assert.Equal(t, event.ChangeEvent{}, changeEventFromProto(nil))
}
