package logger

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.Logger
	level  = zap.NewAtomicLevelAt(zap.InfoLevel)
)

func init() {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	config := zap.Config{
		Level:             level,
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling:          nil,
		Encoding:          "json",
		EncoderConfig:     encoderCfg,
		OutputPaths: []string{
			"stderr",
		},
		ErrorOutputPaths: []string{
			"stderr",
		},
		InitialFields: map[string]interface{}{
			"pid": os.Getpid(),
		},
	}

	logger = zap.Must(config.Build())

	zap.ReplaceGlobals(logger)
}

// SetLevel maps a config log_level string onto the global zap logger.
// Accepted values (case-insensitive): debug, info, warn, error, dpanic, panic, fatal.
func SetLevel(name string) error {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		name = "info"
	}

	var l zapcore.Level
	if err := l.UnmarshalText([]byte(name)); err != nil {
		return fmt.Errorf("invalid log_level %q (want debug|info|warn|error): %w", name, err)
	}
	level.SetLevel(l)
	return nil
}
