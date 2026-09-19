package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr string
	}{
		{
			name:    "empty input",
			input:   "",
			wantLen: 0,
		},
		{
			name: "single proxy",
			input: `kind: Proxy
version: v1
metadata:
  name: p1
spec:
  url: http://127.0.0.1:8080
`,
			wantLen: 1,
		},
		{
			name: "multi-doc",
			input: `kind: Proxy
version: v1
metadata:
  name: p1
spec:
  url: http://127.0.0.1:8080
---
kind: Pool
version: v1
metadata:
  name: pool-1
spec:
  type: static
  members:
    - proxy_id: p1
`,
			wantLen: 2,
		},
		{
			name: "missing kind",
			input: `version: v1
metadata:
  name: p1
spec:
  url: http://127.0.0.1:8080
`,
			wantErr: "kind is required",
		},
		{
			name: "missing version",
			input: `kind: Proxy
metadata:
  name: p1
spec:
  url: http://127.0.0.1:8080
`,
			wantErr: "version is required",
		},
		{
			name:    "invalid yaml",
			input:   "kind: [\n",
			wantErr: "decode yaml document",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseYAML([]byte(tt.input))
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, tt.wantLen)

			if tt.name == "single proxy" {
				assert.Equal(t, "Proxy", got[0].Kind)
				assert.Equal(t, "v1", got[0].Version)
				assert.Equal(t, "p1", got[0].Metadata.Name)
				assert.Contains(t, string(got[0].SpecRaw), "127.0.0.1")
			}
			if tt.name == "multi-doc" {
				assert.Equal(t, "Proxy/v1", got[0].Key())
				assert.Equal(t, "Pool/v1", got[1].Key())
			}
		})
	}
}
