package badger

import (
	"encoding/json"
	"time"
)

type StoredResource[T any] struct {
	StorageVersion string            `json:"storage_version"`
	Kind           string            `json:"kind"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	Labels         map[string]string `json:"labels"`
	Spec           T                 `json:"spec"`
}

func unmarshal[T any](data []byte) (StoredResource[T], error) {
	var r StoredResource[T]
	err := json.Unmarshal(data, &r)
	return r, err
}
