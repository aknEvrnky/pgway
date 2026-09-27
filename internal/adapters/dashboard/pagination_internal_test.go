package dashboard

import (
	"math"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePageSize(t *testing.T) {
	t.Parallel()

	n, err := parsePageSize("")
	require.NoError(t, err)
	assert.Equal(t, defaultListPageSize, n)

	n, err = parsePageSize("10")
	require.NoError(t, err)
	assert.Equal(t, 10, n)

	n, err = parsePageSize("9999")
	require.NoError(t, err)
	assert.Equal(t, domain.DefaultMaxPageSize, n)

	_, err = parsePageSize("0")
	assert.ErrorIs(t, err, errInvalidPageSize)

	_, err = parsePageSize("-1")
	assert.ErrorIs(t, err, errInvalidPageSize)

	_, err = parsePageSize("abc")
	assert.ErrorIs(t, err, errInvalidPageSize)

	_, err = parsePageSize("9223372036854775807") // MaxInt64 — exceeds MaxInt32
	if math.MaxInt == math.MaxInt64 {
		assert.ErrorIs(t, err, errInvalidPageSize)
	}
}

func TestPageTokenRoundTrip(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", encodePageToken(""))
	tok := encodePageToken("cursor-1")
	got, err := decodePageToken(tok)
	require.NoError(t, err)
	assert.Equal(t, "cursor-1", got)

	_, err = decodePageToken("!!")
	assert.Error(t, err)
}
