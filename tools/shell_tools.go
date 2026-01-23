// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tools

import (
	"encoding/json"
	"os/exec"
	"strings"
)

// ShellcheckTool implements shell script linting using shellcheck.
type ShellcheckTool struct {
	*BaseTool
}

// NewShellcheckTool creates a new shellcheck tool.
func NewShellcheckTool() *ShellcheckTool {
	tool := &ShellcheckTool{
		BaseTool: NewBaseTool("shellcheck", "Shell", "shellcheck", LINT),
	}

	tool.SetInstallCommand([]string{"pacman", "-S", "--noconfirm", "shellcheck"})
	tool.SetConfigPatterns([]string{".shellcheckrc"})

	return tool
}

// BuildCommand builds the shellcheck command.
func (t *ShellcheckTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{"-f", "json"} // JSON output for parsing

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Filter only shell files
	shFiles := FilterFilesByExtensions(files, []string{".sh", ".bash", ".zsh", ".ksh"})
	if len(shFiles) > 0 {
		args = append(args, shFiles...)
	} else {
		// Default to common script locations
		args = append(args, "*.sh")
	}

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// ParseOutput parses shellcheck JSON output.
func (t *ShellcheckTool) ParseOutput(output string) []Issue {
	if strings.TrimSpace(output) == "" {
		return []Issue{}
	}

	var shellcheckResults []struct {
		File    string `json:"file"`
		Line    int    `json:"line"`
		Column  int    `json:"column"`
		Level   string `json:"level"`
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal([]byte(output), &shellcheckResults); err != nil {
		return []Issue{}
	}

	issues := make([]Issue, 0, len(shellcheckResults))
	for _, item := range shellcheckResults {
		severity := "info"
		switch item.Level {
		case "error":
			severity = "error"
		case "warning":
			severity = "warning"
		case "style":
			severity = "info"
		}

		issues = append(issues, Issue{
			File:     item.File,
			Line:     item.Line,
			Column:   item.Column,
			Severity: severity,
			Rule:     "SC" + itoa(item.Code),
			Message:  item.Message,
		})
	}

	return issues
}

// itoa converts int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// ShfmtTool implements shell script formatting using shfmt.
type ShfmtTool struct {
	*BaseTool
}

// NewShfmtTool creates a new shfmt tool.
func NewShfmtTool() *ShfmtTool {
	tool := &ShfmtTool{
		BaseTool: NewBaseTool("shfmt", "Shell", "shfmt", FORMAT),
	}

	tool.SetInstallCommand([]string{"go", "install", "mvdan.cc/sh/v3/cmd/shfmt@latest"})
	tool.SetConfigPatterns([]string{".editorconfig"})

	return tool
}

// BuildCommand builds the shfmt command.
func (t *ShfmtTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{"-w"} // Write changes

	// Add indent settings
	args = append(args, "-i", "2") // 2 space indent

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Filter only shell files
	shFiles := FilterFilesByExtensions(files, []string{".sh", ".bash", ".zsh", ".ksh"})
	if len(shFiles) > 0 {
		args = append(args, shFiles...)
	} else {
		args = append(args, ".")
	}

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// Ensure Shell tools implement QualityTool interface.
var (
	_ QualityTool = (*ShellcheckTool)(nil)
	_ QualityTool = (*ShfmtTool)(nil)
)
