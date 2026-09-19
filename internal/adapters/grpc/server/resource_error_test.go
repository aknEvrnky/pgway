package server

import (
	"errors"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/controlplane"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapResourceError(t *testing.T) {
	t.Parallel()

	t.Run("in use maps to FailedPrecondition", func(t *testing.T) {
		err := &controlplane.ResourceInUseError{
			ResourceType: "proxy",
			Name:         "p1",
			Dependents:   []controlplane.ResourceDependent{{Type: "pool", Name: "static-1"}},
		}
		mapped := mapResourceError("delete proxy", err)
		st, ok := status.FromError(mapped)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
		assert.Contains(t, st.Message(), "still referenced by pool")
	})

	t.Run("missing ref maps to FailedPrecondition", func(t *testing.T) {
		err := &controlplane.ResourceMissingRefError{
			ResourceType: "flow",
			Name:         "edge",
			MissingType:  "balancer",
			MissingName:  "lb-1",
		}
		mapped := mapResourceError("apply flow", err)
		st, ok := status.FromError(mapped)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
		assert.Contains(t, st.Message(), `balancer "lb-1" not found`)
	})

	t.Run("other errors map to Internal", func(t *testing.T) {
		mapped := mapResourceError("delete proxy", errors.New("boom"))
		st, ok := status.FromError(mapped)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})
}
