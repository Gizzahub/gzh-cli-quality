// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tools

import (
	"encoding/json"
	"os/exec"
	"strings"
)

// StylelintTool implements CSS linting using stylelint.
type StylelintTool struct {
	*BaseTool
}

// NewStylelintTool creates a new stylelint tool.
func NewStylelintTool() *StylelintTool {
	tool := &StylelintTool{
		BaseTool: NewBaseTool("stylelint", "CSS", "stylelint", LINT),
	}

	tool.SetInstallCommand([]string{"npm", "install", "-g", "stylelint", "stylelint-config-standard"})
	tool.SetConfigPatterns([]string{".stylelintrc", ".stylelintrc.json", ".stylelintrc.yml", "stylelint.config.js"})

	return tool
}

// BuildCommand builds the stylelint command.
func (t *StylelintTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{"-f", "json"} // JSON output for parsing

	// Add config file if specified
	if options.ConfigFile != "" {
		args = append(args, "--config", options.ConfigFile)
	}

	// Add fix flag if requested
	if options.Fix {
		args = append(args, "--fix")
	}

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Filter CSS files
	cssFiles := FilterFilesByExtensions(files, []string{".css", ".scss", ".sass", ".less"})
	if len(cssFiles) > 0 {
		args = append(args, cssFiles...)
	} else {
		args = append(args, "**/*.css", "**/*.scss")
	}

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// ParseOutput parses stylelint JSON output.
func (t *StylelintTool) ParseOutput(output string) []Issue {
	if strings.TrimSpace(output) == "" {
		return []Issue{}
	}

	var stylelintResults []struct {
		Source   string `json:"source"`
		Warnings []struct {
			Line     int    `json:"line"`
			Column   int    `json:"column"`
			Rule     string `json:"rule"`
			Severity string `json:"severity"`
			Text     string `json:"text"`
		} `json:"warnings"`
	}

	if err := json.Unmarshal([]byte(output), &stylelintResults); err != nil {
		return []Issue{}
	}

	var issues []Issue
	for _, file := range stylelintResults {
		for _, w := range file.Warnings {
			issues = append(issues, Issue{
				File:     file.Source,
				Line:     w.Line,
				Column:   w.Column,
				Severity: w.Severity,
				Rule:     w.Rule,
				Message:  w.Text,
			})
		}
	}

	return issues
}

// Ensure CSS tools implement QualityTool interface.
var (
	_ QualityTool = (*StylelintTool)(nil)
)
