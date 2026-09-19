package controlplane

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceInUseError_Error(t *testing.T) {
	t.Parallel()

	t.Run("single dependent", func(t *testing.T) {
		err := newResourceInUse("router", "main", []ResourceDependent{
			{Type: "flow", Name: "edge"},
		})
		assert.Equal(t, `cannot delete router "main": still referenced by flow "edge"`, err.Error())
	})

	t.Run("caps at five and reports remainder", func(t *testing.T) {
		deps := make([]ResourceDependent, 7)
		for i := range deps {
			deps[i] = ResourceDependent{Type: "pool", Name: string(rune('a' + i))}
		}
		err := newResourceInUse("proxy", "p1", deps)
		msg := err.Error()
		assert.Contains(t, msg, `cannot delete proxy "p1": still referenced by`)
		assert.Contains(t, msg, "and 2 more")
		assert.NotContains(t, msg, `pool "f"`) // 6th dependent (index 5) omitted from named list
	})

	t.Run("errors.As", func(t *testing.T) {
		err := error(newResourceInUse("pool", "p", []ResourceDependent{{Type: "balancer", Name: "lb"}}))
		var inUse *ResourceInUseError
		require.True(t, errors.As(err, &inUse))
		assert.Equal(t, "pool", inUse.ResourceType)
		assert.Equal(t, "p", inUse.Name)
	})
}

func TestResourceMissingRefError_Error(t *testing.T) {
	t.Parallel()

	err := newResourceMissingRef("flow", "edge", "balancer", "lb-1")
	assert.Equal(t, `cannot apply flow "edge": balancer "lb-1" not found`, err.Error())

	var missing *ResourceMissingRefError
	require.True(t, errors.As(error(err), &missing))
	assert.Equal(t, "balancer", missing.MissingType)
	assert.Equal(t, "lb-1", missing.MissingName)
}
