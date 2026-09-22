package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig_WithDefaults(t *testing.T) {
	tests := []struct {
		name               string
		cfg                Config
		defaultServiceName string
		want               Config
	}{
		{
			name:               "empty fields fall back to default service name and dev version",
			cfg:                Config{},
			defaultServiceName: "pgway-cp",
			want:               Config{ServiceName: "pgway-cp", Version: "dev"},
		},
		{
			name:               "service name is trimmed before the fallback check",
			cfg:                Config{ServiceName: "  custom  "},
			defaultServiceName: "pgway-cp",
			want:               Config{ServiceName: "custom", Version: "dev"},
		},
		{
			name:               "explicit service name and version are kept",
			cfg:                Config{ServiceName: "custom", Version: "v1"},
			defaultServiceName: "pgway-dp",
			want:               Config{ServiceName: "custom", Version: "v1"},
		},
		{
			name:               "empty fields fall back to pgway-dp default",
			cfg:                Config{},
			defaultServiceName: "pgway-dp",
			want:               Config{ServiceName: "pgway-dp", Version: "dev"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.cfg.withDefaults(tt.defaultServiceName))
		})
	}
}
