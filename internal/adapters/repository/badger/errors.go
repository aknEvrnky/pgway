package badger

import (
	"fmt"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

func errNotFound(kind, id string) error {
	return fmt.Errorf("%s %q not found: %w", kind, id, domain.ErrNotFound)
}
