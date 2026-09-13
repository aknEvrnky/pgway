package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestSetLevel(t *testing.T) {
	t.Cleanup(func() {
		require.NoError(t, SetLevel("info"))
	})

	tests := []struct {
		name string
		in   string
		want zapcore.Level
	}{
		{name: "debug", in: "debug", want: zap.DebugLevel},
		{name: "info", in: "info", want: zap.InfoLevel},
		{name: "warn", in: "warn", want: zap.WarnLevel},
		{name: "warning alias", in: "warning", want: zap.WarnLevel},
		{name: "error", in: "error", want: zap.ErrorLevel},
		{name: "case insensitive", in: "DEBUG", want: zap.DebugLevel},
		{name: "empty defaults to info", in: "", want: zap.InfoLevel},
		{name: "whitespace", in: "  debug  ", want: zap.DebugLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, SetLevel(tt.in))
			assert.True(t, level.Enabled(tt.want))
			if tt.want < zap.FatalLevel {
				assert.False(t, level.Enabled(tt.want-1))
			}
		})
	}
}

func TestSetLevelRejectsUnknown(t *testing.T) {
	t.Cleanup(func() {
		require.NoError(t, SetLevel("info"))
	})

	err := SetLevel("verbose")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log_level")
}
