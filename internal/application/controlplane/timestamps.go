package controlplane

import (
	"errors"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

// resolveCreatedAt keeps CreatedAt on update; treats not-found as create;
// propagates unexpected Find errors so transient DB failures cannot reset timestamps.
func resolveCreatedAt(existingCreated time.Time, findErr error, now time.Time) (time.Time, error) {
	if findErr == nil {
		return existingCreated, nil
	}
	if errors.Is(findErr, domain.ErrNotFound) {
		return now, nil
	}
	return time.Time{}, findErr
}
