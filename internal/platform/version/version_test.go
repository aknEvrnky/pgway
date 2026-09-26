package version_test

import (
	"testing"

	"github.com/aknEvrnky/pgway/internal/platform/version"
	"github.com/stretchr/testify/assert"
)

func TestString_Defaults(t *testing.T) {
	assert.Equal(t, "dev", version.String())
}

func TestLine(t *testing.T) {
	got := version.Line("pgctl")
	assert.Equal(t, "pgctl version dev (commit none, built unknown)", got)
}

func TestString_EmptyFallsBack(t *testing.T) {
	prev := version.Version
	t.Cleanup(func() { version.Version = prev })
	version.Version = ""
	assert.Equal(t, "dev", version.String())
}
