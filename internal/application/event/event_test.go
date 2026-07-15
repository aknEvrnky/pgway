package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceType_IsValid(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		isValid      bool
	}{
		{
			name:         "it is a valid resource type",
			resourceType: "proxy",
			isValid:      true,
		},
		{
			name:         "it is an invalid resource type",
			resourceType: "not_saved",
			isValid:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rtype := ResourceType(tt.resourceType)
			assert.Equal(t, tt.isValid, rtype.IsValid())
		})

	}
}

func TestChangeKind_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		kind    string
		isValid bool
	}{
		{
			name:    "it is a valid change kind",
			kind:    "saved",
			isValid: true,
		},
		{
			name:    "it is an invalid change kind",
			kind:    "usser",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind := ChangeKind(tt.kind)
			assert.Equal(t, tt.isValid, kind.IsValid())
		})

	}
}

func TestChangeEvent_Validate(t *testing.T) {
	tests := []struct {
		name        string
		event       ChangeEvent
		expectedErr error
	}{
		{
			name: "it validates the valid event",
			event: ChangeEvent{
				ID:           "foo-bar-id",
				ResourceType: ResourceTypeProxy,
				ChangeKind:   ChangeKindSaved,
			},
			expectedErr: nil,
		},
		{
			name: "it fails if no id",
			event: ChangeEvent{
				ID:           "",
				ResourceType: ResourceTypeProxy,
				ChangeKind:   ChangeKindSaved,
			},
			expectedErr: ErrChangeEventInvalid,
		},
		{
			name: "it fails if invalid resource type",
			event: ChangeEvent{
				ID:           "some-id",
				ResourceType: "bananas",
				ChangeKind:   ChangeKindSaved,
			},
			expectedErr: ErrChangeEventInvalid,
		},
		{
			name: "it fails if invalid change type",
			event: ChangeEvent{
				ID:           "some-id",
				ResourceType: ResourceTypeRouter,
				ChangeKind:   "truncated",
			},
			expectedErr: ErrChangeEventInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}
