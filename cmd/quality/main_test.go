// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/Gizzahub/gzh-cli-quality"
	"github.com/stretchr/testify/assert"
)

func TestNewQualityCmd_Help(t *testing.T) {
	cmd := quality.NewQualityCmd()

	assert.NotNil(t, cmd)
	assert.Equal(t, "quality", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Test that help command works
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	assert.NoError(t, err)
	output := buf.String()
	// Check that help output contains key information
	assert.Contains(t, output, "quality")
	assert.Contains(t, output, "Available Commands")
	assert.Contains(t, output, "run")
	assert.Contains(t, output, "check")
}

func TestNewQualityCmd_Version(t *testing.T) {
	cmd := quality.NewQualityCmd()
	cmd.Version = "test-version"

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "test-version")
}

func TestNewQualityCmd_Subcommands(t *testing.T) {
	cmd := quality.NewQualityCmd()

	expectedCommands := []string{
		"run",
		"check",
		"init",
		"analyze",
		"install",
		"upgrade",
		"version",
		"list",
		"tool",
	}

	for _, cmdName := range expectedCommands {
		subCmd, _, err := cmd.Find([]string{cmdName})
		assert.NoError(t, err, "Command %s should exist", cmdName)
		assert.NotNil(t, subCmd, "Command %s should not be nil", cmdName)
		assert.Equal(t, cmdName, subCmd.Name(), "Command name should match")
	}
}

func TestRunCommand_Flags(t *testing.T) {
	cmd := quality.NewQualityCmd()
	runCmd, _, err := cmd.Find([]string{"run"})
	assert.NoError(t, err)
	assert.NotNil(t, runCmd)

	expectedFlags := []string{
		"files",
		"fix",
		"format-only",
		"lint-only",
		"workers",
		"extra-args",
		"dry-run",
		"verbose",
		"report",
		"output",
		"since",
		"staged",
		"changed",
	}

	for _, flagName := range expectedFlags {
		flag := runCmd.Flags().Lookup(flagName)
		assert.NotNil(t, flag, "Flag %s should exist", flagName)
	}
}

func TestCheckCommand_Exists(t *testing.T) {
	cmd := quality.NewQualityCmd()
	checkCmd, _, err := cmd.Find([]string{"check"})
	assert.NoError(t, err)
	assert.NotNil(t, checkCmd)
	assert.Equal(t, "check", checkCmd.Name())
}

func TestInitCommand_Exists(t *testing.T) {
	cmd := quality.NewQualityCmd()
	initCmd, _, err := cmd.Find([]string{"init"})
	assert.NoError(t, err)
	assert.NotNil(t, initCmd)
	assert.Equal(t, "init", initCmd.Name())
}

func TestAnalyzeCommand_Exists(t *testing.T) {
	cmd := quality.NewQualityCmd()
	analyzeCmd, _, err := cmd.Find([]string{"analyze"})
	assert.NoError(t, err)
	assert.NotNil(t, analyzeCmd)
	assert.Equal(t, "analyze", analyzeCmd.Name())
}

func TestListCommand_Exists(t *testing.T) {
	cmd := quality.NewQualityCmd()
	listCmd, _, err := cmd.Find([]string{"list"})
	assert.NoError(t, err)
	assert.NotNil(t, listCmd)
	assert.Equal(t, "list", listCmd.Name())
}

func TestToolCommand_Exists(t *testing.T) {
	cmd := quality.NewQualityCmd()
	toolCmd, _, err := cmd.Find([]string{"tool"})
	assert.NoError(t, err)
	assert.NotNil(t, toolCmd)
	assert.Equal(t, "tool", toolCmd.Name())
}

func TestVersionCommand_Exists(t *testing.T) {
	cmd := quality.NewQualityCmd()
	versionCmd, _, err := cmd.Find([]string{"version"})
	assert.NoError(t, err)
	assert.NotNil(t, versionCmd)
	assert.Equal(t, "version", versionCmd.Name())
}

func TestInstallCommand_Exists(t *testing.T) {
	cmd := quality.NewQualityCmd()
	installCmd, _, err := cmd.Find([]string{"install"})
	assert.NoError(t, err)
	assert.NotNil(t, installCmd)
	assert.Equal(t, "install", installCmd.Name())
}

func TestUpgradeCommand_Exists(t *testing.T) {
	cmd := quality.NewQualityCmd()
	upgradeCmd, _, err := cmd.Find([]string{"upgrade"})
	assert.NoError(t, err)
	assert.NotNil(t, upgradeCmd)
	assert.Equal(t, "upgrade", upgradeCmd.Name())
}

func TestMainVersion_Variables(t *testing.T) {
	// Test that version variables exist and have sensible defaults
	assert.NotEmpty(t, version)
	assert.NotEmpty(t, commit)
	assert.NotEmpty(t, date)
}

// Integration-style tests that verify command execution doesn't panic
func TestRunCommand_DryRun(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	assert.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		assert.NoError(t, err)
	}()

	err = os.Chdir(tmpDir)
	assert.NoError(t, err)

	// Create a simple Go file
	err = os.WriteFile("main.go", []byte("package main\n\nfunc main() {}\n"), 0o644)
	assert.NoError(t, err)

	cmd := quality.NewQualityCmd()
	cmd.SetArgs([]string{"run", "--dry-run"})

	// Should not error on dry-run
	err = cmd.Execute()
	// Note: This might error if no tools are available, which is OK for testing
	// We're mainly testing that the command structure is correct
	if err != nil {
		assert.Contains(t, err.Error(), "failed", "Error should be descriptive")
	}
}

func TestListCommand_Execute(t *testing.T) {
	cmd := quality.NewQualityCmd()

	// Capture stdout since list command prints directly
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd.SetArgs([]string{"list"})
	err := cmd.Execute()

	w.Close()
	os.Stdout = old

	assert.NoError(t, err)

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should list available tools
	assert.Contains(t, output, "품질 도구")
}
