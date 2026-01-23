// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tools

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
)

// KtlintTool implements Kotlin linting and formatting using ktlint.
type KtlintTool struct {
	*BaseTool
}

// NewKtlintTool creates a new ktlint tool.
func NewKtlintTool() *KtlintTool {
	tool := &KtlintTool{
		BaseTool: NewBaseTool("ktlint", "Kotlin", "ktlint", BOTH),
	}

	tool.SetInstallCommand([]string{"brew", "install", "ktlint"})
	tool.SetConfigPatterns([]string{".editorconfig", ".ktlint"})

	return tool
}

// BuildCommand builds the ktlint command.
func (t *KtlintTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{}

	// Add format flag if requested
	if options.Fix || options.FormatOnly {
		args = append(args, "-F") // Format mode
	}

	// Output format for parsing (only in lint mode)
	if !options.Fix && !options.FormatOnly {
		args = append(args, "--reporter=json")
	}

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Filter only Kotlin files
	ktFiles := FilterFilesByExtensions(files, []string{".kt", ".kts"})
	if len(ktFiles) == 0 {
		args = append(args, "**/*.kt", "**/*.kts")
	} else {
		args = append(args, ktFiles...)
	}

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// ParseOutput parses ktlint JSON output.
func (t *KtlintTool) ParseOutput(output string) []Issue {
	if strings.TrimSpace(output) == "" {
		return []Issue{}
	}

	var ktlintResults []struct {
		File   string `json:"file"`
		Errors []struct {
			Line    int    `json:"line"`
			Column  int    `json:"column"`
			Message string `json:"message"`
			Rule    string `json:"rule"`
		} `json:"errors"`
	}

	if err := json.Unmarshal([]byte(output), &ktlintResults); err != nil {
		return []Issue{}
	}

	var issues []Issue
	for _, file := range ktlintResults {
		for _, e := range file.Errors {
			issues = append(issues, Issue{
				File:     file.File,
				Line:     e.Line,
				Column:   e.Column,
				Severity: "error",
				Rule:     e.Rule,
				Message:  e.Message,
			})
		}
	}

	return issues
}

// DetektTool implements Kotlin static analysis using detekt.
type DetektTool struct {
	*BaseTool
}

// NewDetektTool creates a new detekt tool.
func NewDetektTool() *DetektTool {
	tool := &DetektTool{
		BaseTool: NewBaseTool("detekt", "Kotlin", "detekt", LINT),
	}

	tool.SetInstallCommand([]string{"brew", "install", "detekt"})
	tool.SetConfigPatterns([]string{"detekt.yml", "detekt.yaml", ".detekt.yml", "config/detekt/detekt.yml"})

	return tool
}

// BuildCommand builds the detekt command.
func (t *DetektTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{}

	// Add config file if specified
	if options.ConfigFile != "" {
		args = append(args, "--config", options.ConfigFile)
	}

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Filter only Kotlin files and determine input
	ktFiles := FilterFilesByExtensions(files, []string{".kt", ".kts"})
	if len(ktFiles) > 0 {
		// Get unique directories
		dirs := make(map[string]bool)
		for _, file := range ktFiles {
			dirs[filepath.Dir(file)] = true
		}
		for dir := range dirs {
			args = append(args, "--input", dir)
		}
	} else {
		args = append(args, "--input", ".")
	}

	// Report format
	args = append(args, "--report", "json:detekt-report.json")

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// ParseOutput parses detekt JSON output.
func (t *DetektTool) ParseOutput(output string) []Issue {
	if strings.TrimSpace(output) == "" {
		return []Issue{}
	}

	// Detekt outputs to file, so we parse text output for immediate results
	// Format: file:line:column: RuleName - message
	var issues []Issue
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, ":") {
			continue
		}

		// Try to parse detekt text format
		parts := strings.SplitN(line, ":", 4)
		if len(parts) >= 4 {
			lineNum := 0
			colNum := 0
			parseIntSafe(parts[1], &lineNum)
			parseIntSafe(parts[2], &colNum)

			msg := strings.TrimSpace(parts[3])
			rule := ""
			if idx := strings.Index(msg, " - "); idx != -1 {
				rule = strings.TrimSpace(msg[:idx])
				msg = strings.TrimSpace(msg[idx+3:])
			}

			issues = append(issues, Issue{
				File:     parts[0],
				Line:     lineNum,
				Column:   colNum,
				Severity: "warning",
				Rule:     rule,
				Message:  msg,
			})
		}
	}

	return issues
}

// parseIntSafe safely parses an integer string.
func parseIntSafe(s string, result *int) {
	s = strings.TrimSpace(s)
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	*result = n
}

// Ensure Kotlin tools implement QualityTool interface.
var (
	_ QualityTool = (*KtlintTool)(nil)
	_ QualityTool = (*DetektTool)(nil)
)
