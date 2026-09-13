package badger_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	badgerutil "github.com/aknEvrnky/pgway/integration/testutil/badger"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

func newTestRegToken(hash string, expiresAt time.Time) *domain.RegistrationToken {
	exp := expiresAt.UTC()
	return &domain.RegistrationToken{Hash: hash, ExpiresAt: &exp}
}

func TestRegistrationTokenRepository(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Save and Consume roundtrip", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		regToken := newTestRegToken("some-hash", time.Now().Add(time.Hour))
		require.NoError(t, store.RegistrationTokens.Save(ctx, regToken))

		got, err := store.RegistrationTokens.Consume(ctx, "some-hash")
		require.NoError(t, err)
		assert.Equal(t, regToken, got)

		// single-use: the record is burned, a second consume must fail
		_, err = store.RegistrationTokens.Consume(ctx, "some-hash")
		assert.ErrorContains(t, err, "not found")
	})

	t.Run("Consume unknown hash", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		_, err := store.RegistrationTokens.Consume(ctx, "ghost")
		assert.ErrorContains(t, err, "not found")
	})

	t.Run("Save rejects token without expiry", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		err := store.RegistrationTokens.Save(ctx, &domain.RegistrationToken{Hash: "no-expiry"})
		assert.ErrorIs(t, err, domain.ErrTokenMustExpire)
	})

	t.Run("Save rejects already expired token", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		err := store.RegistrationTokens.Save(ctx, newTestRegToken("stale", time.Now().Add(-time.Minute)))
		assert.ErrorContains(t, err, "already expired")
	})

	t.Run("expired token becomes unconsumable via native TTL", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		exp := time.Now().Add(time.Second).UTC()
		require.NoError(t, store.RegistrationTokens.Save(ctx, &domain.RegistrationToken{Hash: "short-lived", ExpiresAt: &exp}))

		require.Eventually(t, func() bool {
			if time.Now().Before(exp) {
				return false // guard: no destructive probe before the expiry moment
			}
			_, err := store.RegistrationTokens.Consume(ctx, "short-lived")
			return err != nil
		}, 5*time.Second, 100*time.Millisecond)
	})

	// The star of this suite: the single-use guarantee under real concurrency.
	// Consume is a get+delete in ONE badger txn; SSI conflict detection makes
	// every losing txn fail at commit, so exactly one caller may succeed.
	t.Run("concurrent consume: exactly one wins", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		require.NoError(t, store.RegistrationTokens.Save(ctx, newTestRegToken("contested", time.Now().Add(time.Hour))))

		const workers = 8
		start := make(chan struct{}) // gate so the goroutines actually contend
		var wg sync.WaitGroup
		var successes atomic.Int32

		for range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				if _, err := store.RegistrationTokens.Consume(ctx, "contested"); err == nil {
					successes.Add(1)
				}
			}()
		}

		close(start)
		wg.Wait()

		assert.Equal(t, int32(1), successes.Load(), "single-use: exactly one concurrent consumer may win")
		_, err := store.RegistrationTokens.Consume(ctx, "contested")
		assert.Error(t, err, "the contested record must be gone")
	})
}
