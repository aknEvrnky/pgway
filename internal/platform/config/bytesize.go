package config

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ByteSize is a byte count that unmarshals from YAML/env as either a bare
// integer or a human-readable size (e.g. "10MiB", "512KiB", "100MB").
type ByteSize int64

// UnmarshalText implements encoding.TextUnmarshaler for viper/mapstructure.
func (b *ByteSize) UnmarshalText(text []byte) error {
	n, err := ParseByteSize(string(text))
	if err != nil {
		return err
	}
	*b = n
	return nil
}

// ParseByteSize parses a byte size string.
//
// Supported forms:
//   - bare integer: "0", "2048"
//   - binary IEC: KiB, MiB, GiB, TiB (1024-based)
//   - SI: KB, MB, GB, TB (1000-based)
//   - short binary: K, M, G, T (1024-based)
//
// Unit matching is case-insensitive. Surrounding whitespace is ignored.
func ParseByteSize(s string) (ByteSize, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("byte size: empty")
	}

	// Split leading number (int or float, optional leading minus) from unit.
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	startDigits := i
	for i < len(s) {
		c := rune(s[i])
		if unicode.IsDigit(c) || c == '.' {
			i++
			continue
		}
		break
	}
	if i == startDigits {
		return 0, fmt.Errorf("byte size: missing number: %q", s)
	}

	numStr := s[:i]
	unit := strings.TrimSpace(s[i:])

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("byte size: invalid number %q: %w", numStr, err)
	}
	if num < 0 {
		return 0, fmt.Errorf("byte size: negative not allowed: %q", s)
	}

	mult, err := byteSizeMultiplier(unit)
	if err != nil {
		return 0, err
	}

	bytes := num * float64(mult)
	if bytes > float64(^uint64(0)>>1) { // > MaxInt64
		return 0, fmt.Errorf("byte size: overflow: %q", s)
	}
	return ByteSize(bytes), nil
}

func byteSizeMultiplier(unit string) (int64, error) {
	switch strings.ToLower(unit) {
	case "", "b", "byte", "bytes":
		return 1, nil
	case "k", "kib":
		return 1 << 10, nil
	case "m", "mib":
		return 1 << 20, nil
	case "g", "gib":
		return 1 << 30, nil
	case "t", "tib":
		return 1 << 40, nil
	case "kb":
		return 1000, nil
	case "mb":
		return 1000 * 1000, nil
	case "gb":
		return 1000 * 1000 * 1000, nil
	case "tb":
		return 1000 * 1000 * 1000 * 1000, nil
	default:
		return 0, fmt.Errorf("byte size: unknown unit %q", unit)
	}
}
