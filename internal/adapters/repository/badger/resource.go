package badger

import "encoding/json"

type StoredResource[T any] struct {
	StorageVersion string            `json:"storage_version"`
	Kind           string            `json:"kind"`
	Labels         map[string]string `json:"labels"`
	Spec           T                 `json:"spec"`
}

func unmarshal[T any](data []byte) (StoredResource[T], error) {
	var r StoredResource[T]
	err := json.Unmarshal(data, &r)
	return r, err
}
