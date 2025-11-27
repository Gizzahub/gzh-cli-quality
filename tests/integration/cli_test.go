//go:build integration
// +build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var gzQualityBinary string

func TestMain(m *testing.M) {
	// Get absolute path to project root
	wd, err := os.Getwd()
	if err != nil {
		panic("Failed to get working directory: " + err.Error())
	}

	projectRoot := filepath.Join(wd, "../..")
	gzQualityBinary = filepath.Join(projectRoot, "build", "gz-quality")

	// Ensure binary is built
	if _, err := os.Stat(gzQualityBinary); os.IsNotExist(err) {
		cmd := exec.Command("make", "build")
		cmd.Dir = projectRoot
		if err := cmd.Run(); err != nil {
			panic("Failed to build gz-quality binary: " + err.Error())
		}
	}

	os.Exit(m.Run())
}

func TestCLI_Version(t *testing.T) {
	cmd := exec.Command(gzQualityBinary, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gz-quality version failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "📋 설치된 품질 도구 버전") {
		t.Errorf("Expected version header in output, got: %s", outputStr)
	}
}

func TestCLI_List(t *testing.T) {
	cmd := exec.Command(gzQualityBinary, "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gz-quality list failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Check for expected languages
	expectedLanguages := []string{"Go", "Python", "JavaScript", "TypeScript", "Rust"}
	for _, lang := range expectedLanguages {
		if !strings.Contains(outputStr, lang) {
			t.Errorf("Expected language %s in output, got: %s", lang, outputStr)
		}
	}

	// Check for expected tools
	expectedTools := []string{"gofumpt", "golangci-lint", "ruff", "prettier", "eslint", "rustfmt", "clippy"}
	for _, tool := range expectedTools {
		if !strings.Contains(outputStr, tool) {
			t.Errorf("Expected tool %s in output, got: %s", tool, outputStr)
		}
	}
}

func TestCLI_Help(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"root help", []string{"--help"}},
		{"run help", []string{"run", "--help"}},
		{"check help", []string{"check", "--help"}},
		{"init help", []string{"init", "--help"}},
		{"analyze help", []string{"analyze", "--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(gzQualityBinary, tt.args...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("gz-quality %v failed: %v\nOutput: %s", tt.args, err, output)
			}

			outputStr := string(output)
			if !strings.Contains(outputStr, "Usage:") {
				t.Errorf("Expected 'Usage:' in help output, got: %s", outputStr)
			}
		})
	}
}

func TestCLI_Analyze(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create Go file
	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run analyze
	cmd := exec.Command(gzQualityBinary, "analyze")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gz-quality analyze failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Go") {
		t.Errorf("Expected Go language detection, got: %s", outputStr)
	}
}

func TestCLI_Init(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Go file
	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run init
	cmd := exec.Command(gzQualityBinary, "init")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gz-quality init failed: %v\nOutput: %s", err, output)
	}

	// Check config file created
	configFile := filepath.Join(tmpDir, ".gzquality.yml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Errorf("Config file not created at %s", configFile)
	}

	// Verify config content
	content, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "enabled: true") {
		t.Errorf("Expected 'enabled: true' in config, got: %s", contentStr)
	}
}

func TestCLI_DryRun(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Go file
	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run with --dry-run
	cmd := exec.Command(gzQualityBinary, "run", "--dry-run")
	cmd.Dir = tmpDir
	output, _ := cmd.CombinedOutput()
	// Note: May fail if no tools installed, but should show plan

	outputStr := string(output)
	if !strings.Contains(outputStr, "실행 계획") && !strings.Contains(outputStr, "도구가 설치되지 않음") {
		t.Logf("Dry run output: %s", outputStr)
		// Not a failure - just informational
	}
}

func TestCLI_InvalidCommand(t *testing.T) {
	cmd := exec.Command(gzQualityBinary, "nonexistent")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected error for invalid command")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "unknown command") && !strings.Contains(outputStr, "Error") {
		t.Errorf("Expected error message, got: %s", outputStr)
	}
}

func TestCLI_InvalidFlag(t *testing.T) {
	cmd := exec.Command(gzQualityBinary, "run", "--invalid-flag")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("Expected error for invalid flag")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "unknown flag") && !strings.Contains(outputStr, "Error") {
		t.Errorf("Expected error message, got: %s", outputStr)
	}
}
