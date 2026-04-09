// Package logger предоставляет инициализацию логгера приложения.
package logger

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Init создаёт и возвращает новый экземпляр логгера.
//
// На текущем этапе используется production-настройка zap.
// Уровень логирования передаётся строкой, например:
// "debug", "info", "warn", "error".
func Init(level string) (*zap.Logger, error) {
	parsedLevel, err := parseLevel(level)
	if err != nil {
		return nil, err
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(parsedLevel)
	cfg.Encoding = "json"
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.MessageKey = "msg"
	cfg.EncoderConfig.LevelKey = "level"
	cfg.EncoderConfig.CallerKey = "caller"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	log, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("build logger: %w", err)
	}

	return log, nil
}

// parseLevel преобразует строковое значение уровня логирования
// в уровень zap.
func parseLevel(level string) (zapcore.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("unsupported log level: %q", level)
	}
}
