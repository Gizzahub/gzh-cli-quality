// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package cache

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Gizzahub/gzh-cli-quality/tools"
)

// mockTool is a mock implementation of tools.QualityTool for testing.
type mockTool struct {
	name    string
	version string
	configs []string
}

func (m *mockTool) Name() string                              { return m.name }
func (m *mockTool) Language() string                          { return "Go" }
func (m *mockTool) Type() tools.ToolType                      { return tools.FORMAT }
func (m *mockTool) IsAvailable() bool                         { return true }
func (m *mockTool) Install() error                            { return nil }
func (m *mockTool) GetVersion() (string, error)               { return m.version, nil }
func (m *mockTool) Upgrade() error                            { return nil }
func (m *mockTool) FindConfigFiles(root string) []string      { return m.configs }
func (m *mockTool) Execute(ctx context.Context, files []string, options tools.ExecuteOptions) (*tools.Result, error) {
	return nil, nil
}

func TestGenerateKey(t *testing.T) {
	// Create test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	testContent := []byte("package main\n\nfunc main() {}\n")

	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := &mockTool{
		name:    "gofumpt",
		version: "v0.7.0",
		configs: []string{},
	}

	options := tools.ExecuteOptions{
		ProjectRoot: tmpDir,
		Fix:         true,
	}

	// Test key generation
	key, err := GenerateKey(testFile, tool, options)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	// Validate key fields
	if key.FilePath == "" {
		t.Error("FilePath is empty")
	}

	if key.FileHash == "" {
		t.Error("FileHash is empty")
	}

	if key.ToolName != "gofumpt" {
		t.Errorf("ToolName = %s, want gofumpt", key.ToolName)
	}

	if key.ToolVersion != "v0.7.0" {
		t.Errorf("ToolVersion = %s, want v0.7.0", key.ToolVersion)
	}

	if key.OptionsHash == "" {
		t.Error("OptionsHash is empty")
	}
}

func TestGenerateKey_DifferentContent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two files with different content
	file1 := filepath.Join(tmpDir, "file1.go")
	file2 := filepath.Join(tmpDir, "file2.go")

	if err := os.WriteFile(file1, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(file2, []byte("package test"), 0644); err != nil {
		t.Fatal(err)
	}

	tool := &mockTool{name: "gofumpt", version: "v0.7.0"}
	options := tools.ExecuteOptions{ProjectRoot: tmpDir}

	key1, err := GenerateKey(file1, tool, options)
	if err != nil {
		t.Fatal(err)
	}

	key2, err := GenerateKey(file2, tool, options)
	if err != nil {
		t.Fatal(err)
	}

	// Keys should be different (different file hash)
	if key1.FileHash == key2.FileHash {
		t.Error("Expected different file hashes for different content")
	}
}

func TestGenerateKey_DifferentOptions(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	tool := &mockTool{name: "gofumpt", version: "v0.7.0"}

	options1 := tools.ExecuteOptions{Fix: true}
	options2 := tools.ExecuteOptions{Fix: false}

	key1, _ := GenerateKey(testFile, tool, options1)
	key2, _ := GenerateKey(testFile, tool, options2)

	// Keys should be different (different options hash)
	if key1.OptionsHash == key2.OptionsHash {
		t.Error("Expected different options hashes for different options")
	}
}

func TestCacheKey_String(t *testing.T) {
	key := CacheKey{
		FilePath:    "/path/to/file.go",
		FileHash:    "a1b2c3d4e5f6g7h8",
		ToolName:    "gofumpt",
		ToolVersion: "v0.7.0",
		ConfigHash:  "i9j0k1l2m3n4o5p6",
		OptionsHash: "q7r8s9t0u1v2w3x4",
	}

	str := key.String()

	// Should be: gofumpt-v0.7.0-a1b2c3d4-i9j0k1l2-q7r8s9t0
	expected := "gofumpt-v0.7.0-a1b2c3d4-i9j0k1l2-q7r8s9t0"
	if str != expected {
		t.Errorf("String() = %s, want %s", str, expected)
	}
}

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name    string
		key     CacheKey
		wantErr bool
	}{
		{
			name: "valid key",
			key: CacheKey{
				FilePath:    "/path/to/file.go",
				FileHash:    "abc123",
				ToolName:    "gofumpt",
				ToolVersion: "v0.7.0",
			},
			wantErr: false,
		},
		{
			name: "empty file path",
			key: CacheKey{
				FileHash:    "abc123",
				ToolName:    "gofumpt",
				ToolVersion: "v0.7.0",
			},
			wantErr: true,
		},
		{
			name: "empty file hash",
			key: CacheKey{
				FilePath:    "/path/to/file.go",
				ToolName:    "gofumpt",
				ToolVersion: "v0.7.0",
			},
			wantErr: true,
		},
		{
			name: "empty tool name",
			key: CacheKey{
				FilePath:    "/path/to/file.go",
				FileHash:    "abc123",
				ToolVersion: "v0.7.0",
			},
			wantErr: true,
		},
		{
			name: "empty tool version",
			key: CacheKey{
				FilePath: "/path/to/file.go",
				FileHash: "abc123",
				ToolName: "gofumpt",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHashFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := []byte("package main\n")

	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := hashFile(testFile)
	if err != nil {
		t.Fatalf("hashFile failed: %v", err)
	}

	// Hash should be deterministic
	hash2, _ := hashFile(testFile)
	if hash != hash2 {
		t.Error("hashFile should be deterministic")
	}

	// Hash should change when content changes
	if err := os.WriteFile(testFile, []byte("package test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	hash3, _ := hashFile(testFile)
	if hash == hash3 {
		t.Error("hashFile should change when content changes")
	}
}

func TestHashFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Test empty list
	hash, err := hashFiles([]string{})
	if err != nil {
		t.Errorf("hashFiles([]) failed: %v", err)
	}
	if hash != "" {
		t.Error("hashFiles([]) should return empty string")
	}

	// Test with files
	file1 := filepath.Join(tmpDir, "config1.yml")
	file2 := filepath.Join(tmpDir, "config2.yml")

	if err := os.WriteFile(file1, []byte("key: value1"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(file2, []byte("key: value2"), 0644); err != nil {
		t.Fatal(err)
	}

	hash1, err := hashFiles([]string{file1, file2})
	if err != nil {
		t.Fatalf("hashFiles failed: %v", err)
	}

	// Hash should be deterministic regardless of order (sorted internally)
	hash2, _ := hashFiles([]string{file2, file1})
	if hash1 != hash2 {
		t.Error("hashFiles should be deterministic regardless of order")
	}

	// Test with non-existent files (should not error, just skip)
	hash3, err := hashFiles([]string{file1, "/nonexistent/file.yml"})
	if err != nil {
		t.Errorf("hashFiles should not error on non-existent files: %v", err)
	}

	// Hash should still be generated from existing files
	if hash3 == "" {
		t.Error("hashFiles should generate hash from existing files")
	}
}

func TestHashOptions(t *testing.T) {
	tests := []struct {
		name string
		opt1 tools.ExecuteOptions
		opt2 tools.ExecuteOptions
		same bool
	}{
		{
			name: "same options",
			opt1: tools.ExecuteOptions{Fix: true},
			opt2: tools.ExecuteOptions{Fix: true},
			same: true,
		},
		{
			name: "different fix flag",
			opt1: tools.ExecuteOptions{Fix: true},
			opt2: tools.ExecuteOptions{Fix: false},
			same: false,
		},
		{
			name: "different extra args",
			opt1: tools.ExecuteOptions{ExtraArgs: []string{"--verbose"}},
			opt2: tools.ExecuteOptions{ExtraArgs: []string{"--quiet"}},
			same: false,
		},
		{
			name: "same extra args different order",
			opt1: tools.ExecuteOptions{ExtraArgs: []string{"--verbose", "--color"}},
			opt2: tools.ExecuteOptions{ExtraArgs: []string{"--color", "--verbose"}},
			same: true, // Sorted internally
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := hashOptions(tt.opt1)
			hash2 := hashOptions(tt.opt2)

			if tt.same && hash1 != hash2 {
				t.Error("Expected same hashes")
			}

			if !tt.same && hash1 == hash2 {
				t.Error("Expected different hashes")
			}
		})
	}
}
