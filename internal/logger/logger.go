// Package logger provides simple structured logging for quality tools.
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

// Level represents logging level.
type Level int

const (
	// LevelDebug is for debug messages.
	LevelDebug Level = iota
	// LevelInfo is for informational messages.
	LevelInfo
	// LevelWarn is for warning messages.
	LevelWarn
	// LevelError is for error messages.
	LevelError
)

// String returns the string representation of the level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging with levels.
type Logger struct {
	component string
	level     Level
	logger    *log.Logger
	output    io.Writer
}

// New creates a new logger for a component.
func New(component string) *Logger {
	return &Logger{
		component: component,
		level:     LevelInfo,
		logger:    log.New(os.Stderr, "", 0),
		output:    os.Stderr,
	}
}

// SetLevel sets the minimum logging level.
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// SetOutput sets the output writer.
func (l *Logger) SetOutput(w io.Writer) {
	l.output = w
	l.logger.SetOutput(w)
}

// log writes a log message if the level is enabled.
func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("[%s] %s [%s] %s",
		timestamp,
		level.String(),
		l.component,
		msg,
	)

	l.logger.Println(logLine)
}

// Debug logs a debug message.
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs an info message.
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message.
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// WithField returns a message with a key-value field.
func (l *Logger) WithField(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

// WithFields returns a message with multiple key-value fields.
func (l *Logger) WithFields(fields map[string]string) string {
	var parts []string
	for k, v := range fields {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, " ")
}

// Global logger instance.
var defaultLogger = New("default")

// SetDefaultLevel sets the level for the default logger.
func SetDefaultLevel(level Level) {
	defaultLogger.SetLevel(level)
}

// SetDefaultOutput sets the output for the default logger.
func SetDefaultOutput(w io.Writer) {
	defaultLogger.SetOutput(w)
}

// Debug logs a debug message using the default logger.
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// Info logs an info message using the default logger.
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Warn logs a warning message using the default logger.
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// Error logs an error message using the default logger.
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}
