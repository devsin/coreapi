package logger

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var global *zap.Logger

// New constructs a zap logger based on environment and level.
func New(env, level string) (*zap.Logger, error) {
	var cfg zap.Config
	if strings.EqualFold(env, "prod") {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
	}

	lvl, err := zapcore.ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	cfg.Level = zap.NewAtomicLevelAt(lvl)

	l, err := cfg.Build(zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		return nil, err
	}

	global = l
	return l, nil
}

// L returns the global logger if initialized, otherwise a no-op logger.
func L() *zap.Logger {
	if global != nil {
		return global
	}
	return zap.NewNop()
}
