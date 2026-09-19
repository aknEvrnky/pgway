package controlplane

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveCreatedAt(t *testing.T) {
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	existing := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("preserves existing on success", func(t *testing.T) {
		got, err := resolveCreatedAt(existing, nil, now)
		require.NoError(t, err)
		assert.Equal(t, existing, got)
	})

	t.Run("uses now on not found", func(t *testing.T) {
		got, err := resolveCreatedAt(time.Time{}, fmt.Errorf("proxy %q not found: %w", "x", domain.ErrNotFound), now)
		require.NoError(t, err)
		assert.Equal(t, now, got)
	})

	t.Run("propagates other errors", func(t *testing.T) {
		boom := errors.New("db unavailable")
		_, err := resolveCreatedAt(existing, boom, now)
		assert.ErrorIs(t, err, boom)
	})
}
