package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseByteSize(t *testing.T) {
	tests := []struct {
		in      string
		want    ByteSize
		wantErr string
	}{
		{in: "0", want: 0},
		{in: "2048", want: 2048},
		{in: "10MiB", want: 10 << 20},
		{in: "10mib", want: 10 << 20},
		{in: "512KiB", want: 512 << 10},
		{in: "1GiB", want: 1 << 30},
		{in: "1.5MiB", want: ByteSize(1.5 * float64(1<<20))},
		{in: "10M", want: 10 << 20},
		{in: "100MB", want: 100 * 1000 * 1000},
		{in: "  10MiB  ", want: 10 << 20},
		{in: "", wantErr: "empty"},
		{in: "MiB", wantErr: "missing number"},
		{in: "-1", wantErr: "negative"},
		{in: "10XiB", wantErr: "unknown unit"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseByteSize(tt.in)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestByteSize_UnmarshalText(t *testing.T) {
	var b ByteSize
	require.NoError(t, b.UnmarshalText([]byte("10MiB")))
	assert.Equal(t, ByteSize(10<<20), b)
	assert.Equal(t, int64(10<<20), int64(b))
}
