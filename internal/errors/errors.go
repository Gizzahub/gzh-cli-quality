// Package errors provides quality-specific error types and utilities.
package errors

import (
	"errors"
	"fmt"
)

// Common quality-related errors.
var (
	// ErrToolNotFound indicates a required tool is not installed.
	ErrToolNotFound = errors.New("tool not found")

	// ErrInvalidConfig indicates invalid configuration.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrCacheCorrupted indicates cache data is corrupted.
	ErrCacheCorrupted = errors.New("cache corrupted")

	// ErrLanguageNotSupported indicates unsupported language.
	ErrLanguageNotSupported = errors.New("language not supported")

	// ErrExecutionFailed indicates tool execution failed.
	ErrExecutionFailed = errors.New("execution failed")

	// ErrInvalidPath indicates invalid file or directory path.
	ErrInvalidPath = errors.New("invalid path")

	// ErrNoFilesToCheck indicates no files found to check.
	ErrNoFilesToCheck = errors.New("no files to check")

	// ErrParsingFailed indicates output parsing failed.
	ErrParsingFailed = errors.New("parsing failed")
)

// QualityError represents a quality check error with context.
type QualityError struct {
	// Op is the operation that failed (e.g., "lint", "format").
	Op string

	// Tool is the tool name that failed (e.g., "golangci-lint", "ruff").
	Tool string

	// Path is the file or directory path where error occurred.
	Path string

	// Err is the underlying error.
	Err error
}

// Error implements the error interface.
func (e *QualityError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s: %s failed for %s: %v", e.Op, e.Tool, e.Path, e.Err)
	}
	return fmt.Sprintf("%s: %s failed: %v", e.Op, e.Tool, e.Err)
}

// Unwrap returns the underlying error.
func (e *QualityError) Unwrap() error {
	return e.Err
}

// New creates a new QualityError.
func New(op, tool string, err error) *QualityError {
	return &QualityError{
		Op:   op,
		Tool: tool,
		Err:  err,
	}
}

// Wrap wraps an error with operation and tool context.
func Wrap(op, tool string, err error) error {
	if err == nil {
		return nil
	}
	return &QualityError{
		Op:   op,
		Tool: tool,
		Err:  err,
	}
}

// WrapWithPath wraps an error with operation, tool, and path context.
func WrapWithPath(op, tool, path string, err error) error {
	if err == nil {
		return nil
	}
	return &QualityError{
		Op:   op,
		Tool: tool,
		Path: path,
		Err:  err,
	}
}

// Is checks if err is a specific error type.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target.
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}
