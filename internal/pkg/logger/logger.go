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

// NewStructured creates a structured logger with additional fields
func NewStructured(level, format string, fields map[string]interface{}) *zap.Logger {
	logger := New(level, format)

	// Add structured fields
	var zapFields []zap.Field
	for key, value := range fields {
		zapFields = append(zapFields, zap.Any(key, value))
	}

	return logger.With(zapFields...)
}

// NewForService creates a logger specifically configured for a service
func NewForService(serviceName, level, format string) *zap.Logger {
	logger := New(level, format)
	return logger.With(
		zap.String("service", serviceName),
		zap.String("component", "oceanproxy"),
	)
}

// LogLevel represents available log levels
type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	FatalLevel LogLevel = "fatal"
	PanicLevel LogLevel = "panic"
)

// LogFormat represents available log formats
type LogFormat string

const (
	JSONFormat    LogFormat = "json"
	ConsoleFormat LogFormat = "console"
)

// Config represents logger configuration
type Config struct {
	Level    LogLevel               `yaml:"level" json:"level"`
	Format   LogFormat              `yaml:"format" json:"format"`
	FilePath string                 `yaml:"file_path,omitempty" json:"file_path,omitempty"`
	Fields   map[string]interface{} `yaml:"fields,omitempty" json:"fields,omitempty"`
}

// NewFromConfig creates a logger from configuration
func NewFromConfig(config Config) (*zap.Logger, error) {
	level := string(config.Level)
	format := string(config.Format)

	if config.FilePath != "" {
		return NewWithFile(level, format, config.FilePath)
	}

	if len(config.Fields) > 0 {
		return NewStructured(level, format, config.Fields), nil
	}

	return New(level, format), nil
}

// GetDefaultConfig returns default logger configuration
func GetDefaultConfig() Config {
	return Config{
		Level:  InfoLevel,
		Format: JSONFormat,
	}
}

// GetDevelopmentConfig returns development logger configuration
func GetDevelopmentConfig() Config {
	return Config{
		Level:  DebugLevel,
		Format: ConsoleFormat,
		Fields: map[string]interface{}{
			"environment": "development",
		},
	}
}

// GetProductionConfig returns production logger configuration
func GetProductionConfig() Config {
	return Config{
		Level:  InfoLevel,
		Format: JSONFormat,
		Fields: map[string]interface{}{
			"environment": "production",
		},
	}
}
