// Package logger provides structured JSON logging using slog.
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

var defaultLogger *slog.Logger

// CtxKey is the type for context keys.
type CtxKey string

const (
	// RequestIDKey is the context key for request ID.
	RequestIDKey CtxKey = "request_id"
	// UserIDKey is the context key for user ID.
	UserIDKey CtxKey = "user_id"
)

// Init initializes the global logger.
func Init(level, format string, output io.Writer) {
	if output == nil {
		output = os.Stdout
	}

	lvl := parseLevel(level)

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     lvl,
		AddSource: lvl == slog.LevelDebug,
	}

	if strings.ToLower(format) == "text" {
		handler = slog.NewTextHandler(output, opts)
	} else {
		handler = slog.NewJSONHandler(output, opts)
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

// parseLevel converts a string log level to slog.Level.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Logger returns the default logger.
func Logger() *slog.Logger {
	if defaultLogger == nil {
		Init("info", "json", nil)
	}
	return defaultLogger
}

// WithContext returns a logger with context values (request_id, user_id).
func WithContext(ctx context.Context) *slog.Logger {
	l := Logger()

	if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
		l = l.With("request_id", reqID)
	}
	if userID, ok := ctx.Value(UserIDKey).(string); ok && userID != "" {
		l = l.With("user_id", userID)
	}

	return l
}

// Debug logs at debug level.
func Debug(msg string, args ...any) {
	Logger().Debug(msg, args...)
}

// Info logs at info level.
func Info(msg string, args ...any) {
	Logger().Info(msg, args...)
}

// Warn logs at warn level.
func Warn(msg string, args ...any) {
	Logger().Warn(msg, args...)
}

// Error logs at error level.
func Error(msg string, args ...any) {
	Logger().Error(msg, args...)
}

// DebugContext logs at debug level with context.
func DebugContext(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Debug(msg, args...)
}

// InfoContext logs at info level with context.
func InfoContext(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Info(msg, args...)
}

// WarnContext logs at warn level with context.
func WarnContext(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Warn(msg, args...)
}

// ErrorContext logs at error level with context.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Error(msg, args...)
}
