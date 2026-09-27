package dashboard

import (
	"encoding/base64"
	"errors"
	"math"
	"strconv"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

const defaultListPageSize = 20

var errInvalidPageSize = errors.New("page_size must be a positive integer")

// parsePageSize parses the page_size query value.
// Empty → default. Values above DefaultMaxPageSize are clamped.
// Rejects non-positive values and anything that would not fit in int32
// (CodeQL go/incorrect-integer-conversion on the gRPC List PageSize field).
func parsePageSize(raw string) (int, error) {
	if raw == "" {
		return defaultListPageSize, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, errInvalidPageSize
	}
	if n > math.MaxInt32 {
		return 0, errInvalidPageSize
	}
	if n > domain.DefaultMaxPageSize {
		n = domain.DefaultMaxPageSize
	}
	return n, nil
}

func decodePageToken(token string) (string, error) {
	if token == "" {
		return "", nil
	}
	b, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func encodePageToken(cursor string) string {
	if cursor == "" {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte(cursor))
}
