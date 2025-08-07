// pkg/logger/logger.go - Simple but complete logger implementation
package logger

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a new zap logger with the specified level and format
func New(level, format string) *zap.Logger {
	// Parse log level
	var zapLevel zapcore.Level
	switch strings.ToLower(level) {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn", "warning":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	case "fatal":
		zapLevel = zapcore.FatalLevel
	case "panic":
		zapLevel = zapcore.PanicLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	// Configure encoder
	var encoder zapcore.Encoder
	var config zapcore.EncoderConfig

	if strings.ToLower(format) == "json" {
		config = zap.NewProductionEncoderConfig()
		config.TimeKey = "timestamp"
		config.EncodeTime = zapcore.ISO8601TimeEncoder
		encoder = zapcore.NewJSONEncoder(config)
	} else {
		config = zap.NewDevelopmentEncoderConfig()
		config.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncodeTime = zapcore.ISO8601TimeEncoder
		encoder = zapcore.NewConsoleEncoder(config)
	}

	// Create core
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		zapLevel,
	)

	// Add caller information and stack trace for errors
	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.AddCallerSkip(0),
	)

	return logger
}

// NewWithFile creates a logger that writes to both stdout and a file
func NewWithFile(level, format, filePath string) (*zap.Logger, error) {
	// Parse log level
	var zapLevel zapcore.Level
	switch strings.ToLower(level) {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn", "warning":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	case "fatal":
		zapLevel = zapcore.FatalLevel
	case "panic":
		zapLevel = zapcore.PanicLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	// Open log file
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Configure encoder
	var encoder zapcore.Encoder
	var config zapcore.EncoderConfig

	if strings.ToLower(format) == "json" {
		config = zap.NewProductionEncoderConfig()
		config.TimeKey = "timestamp"
		config.EncodeTime = zapcore.ISO8601TimeEncoder
		encoder = zapcore.NewJSONEncoder(config)
	} else {
		config = zap.NewDevelopmentEncoderConfig()
		config.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncodeTime = zapcore.ISO8601TimeEncoder
		encoder = zapcore.NewConsoleEncoder(config)
	}

	// Create multi-writer core (both stdout and file)
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapLevel),
		zapcore.NewCore(encoder, zapcore.AddSync(file), zapLevel),
	)

	// Add caller information and stack trace for errors
	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.AddCallerSkip(0),
	)

	return logger, nil
}

// NewForService creates a logger specifically configured for a service
func NewForService(serviceName, level, format string) *zap.Logger {
	logger := New(level, format)
	return logger.With(
		zap.String("service", serviceName),
		zap.String("component", "oceanproxy"),
	)
}
