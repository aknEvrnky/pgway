package ports

import "context"

// StoragePinger cheaply checks that durable storage is usable (readiness).
type StoragePinger interface {
	Ping(ctx context.Context) error
}
