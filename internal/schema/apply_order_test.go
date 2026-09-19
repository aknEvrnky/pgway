package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSortByApplyOrder(t *testing.T) {
	t.Parallel()

	t.Run("reverse file order becomes dependency order", func(t *testing.T) {
		in := []RawResource{
			{Kind: "Entrypoint", Version: "v1", Metadata: Metadata{Name: "ep"}},
			{Kind: "Flow", Version: "v1", Metadata: Metadata{Name: "flow"}},
			{Kind: "Router", Version: "v1", Metadata: Metadata{Name: "router"}},
			{Kind: "LoadBalancer", Version: "v1", Metadata: Metadata{Name: "lb"}},
			{Kind: "Pool", Version: "v1", Metadata: Metadata{Name: "pool"}},
			{Kind: "Proxy", Version: "v1", Metadata: Metadata{Name: "proxy"}},
		}
		out := SortByApplyOrder(in)
		require.Len(t, out, 6)
		kinds := make([]string, len(out))
		for i, r := range out {
			kinds[i] = r.Kind
		}
		assert.Equal(t, []string{"Proxy", "Pool", "LoadBalancer", "Router", "Flow", "Entrypoint"}, kinds)
	})

	t.Run("stable within kind", func(t *testing.T) {
		in := []RawResource{
			{Kind: "Proxy", Version: "v1", Metadata: Metadata{Name: "b"}},
			{Kind: "Proxy", Version: "v1", Metadata: Metadata{Name: "a"}},
			{Kind: "Pool", Version: "v1", Metadata: Metadata{Name: "p"}},
		}
		out := SortByApplyOrder(in)
		assert.Equal(t, "b", out[0].Metadata.Name)
		assert.Equal(t, "a", out[1].Metadata.Name)
		assert.Equal(t, "p", out[2].Metadata.Name)
	})

	t.Run("does not mutate input", func(t *testing.T) {
		in := []RawResource{
			{Kind: "Pool", Version: "v1"},
			{Kind: "Proxy", Version: "v1"},
		}
		_ = SortByApplyOrder(in)
		assert.Equal(t, "Pool", in[0].Kind)
	})
}
