// Package testutil provides testing utilities and helpers.
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TempDir creates a temporary directory for testing.
// It automatically cleans up after the test.
func TempDir(t *testing.T, pattern string) string {
	t.Helper()

	dir, err := os.MkdirTemp("", pattern)
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Errorf("failed to remove temp dir: %v", err)
		}
	})

	return dir
}

// TempFile creates a temporary file with content for testing.
// It automatically cleans up after the test.
func TempFile(t *testing.T, dir, pattern, content string) string {
	t.Helper()

	if dir == "" {
		dir = t.TempDir()
	}

	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer file.Close()

	if content != "" {
		if _, err := file.WriteString(content); err != nil {
			t.Fatalf("failed to write to temp file: %v", err)
		}
	}

	return file.Name()
}

// WriteFile writes content to a file in the given directory.
// It creates the directory if it doesn't exist.
func WriteFile(t *testing.T, dir, filename, content string) string {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	return path
}

// AssertEqual asserts that two values are equal.
func AssertEqual(t *testing.T, got, want interface{}) {
	t.Helper()

	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// AssertNotEqual asserts that two values are not equal.
func AssertNotEqual(t *testing.T, got, want interface{}) {
	t.Helper()

	if got == want {
		t.Errorf("got %v, expected not equal to %v", got, want)
	}
}

// AssertNil asserts that value is nil.
func AssertNil(t *testing.T, value interface{}) {
	t.Helper()

	if value != nil {
		t.Errorf("expected nil, got %v", value)
	}
}

// AssertNotNil asserts that value is not nil.
func AssertNotNil(t *testing.T, value interface{}) {
	t.Helper()

	if value == nil {
		t.Errorf("expected non-nil value")
	}
}

// AssertError asserts that an error occurred.
func AssertError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

// AssertNoError asserts that no error occurred.
func AssertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// AssertErrorIs asserts that error matches target.
func AssertErrorIs(t *testing.T, err, target error) {
	t.Helper()

	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}

	if !isError(err, target) {
		t.Errorf("error %v is not %v", err, target)
	}
}

// AssertStringContains asserts that string contains substring.
func AssertStringContains(t *testing.T, str, substr string) {
	t.Helper()

	if !contains(str, substr) {
		t.Errorf("string %q does not contain %q", str, substr)
	}
}

// AssertFileExists asserts that a file exists.
func AssertFileExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("file does not exist: %s", path)
	}
}

// AssertFileNotExists asserts that a file does not exist.
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err == nil {
		t.Errorf("file exists but should not: %s", path)
	}
}

// Helper functions

func isError(err, target error) bool {
	if err == nil || target == nil {
		return err == target
	}
	// Simple string comparison for now
	return err.Error() == target.Error()
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
