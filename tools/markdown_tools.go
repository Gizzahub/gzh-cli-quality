// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tools

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// MarkdownlintTool implements Markdown linting using markdownlint-cli2.
type MarkdownlintTool struct {
	*BaseTool
}

// NewMarkdownlintTool creates a new markdownlint tool.
func NewMarkdownlintTool() *MarkdownlintTool {
	tool := &MarkdownlintTool{
		BaseTool: NewBaseTool("markdownlint", "Markdown", "markdownlint-cli2", LINT),
	}

	tool.SetInstallCommand([]string{"npm", "install", "-g", "markdownlint-cli2"})
	tool.SetConfigPatterns([]string{".markdownlint.json", ".markdownlint.yaml", ".markdownlint.yml", ".markdownlint-cli2.jsonc"})

	return tool
}

// BuildCommand builds the markdownlint command.
func (t *MarkdownlintTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{}

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

	// Filter only Markdown files
	mdFiles := FilterFilesByExtensions(files, []string{".md", ".markdown"})
	if len(mdFiles) == 0 {
		args = append(args, "**/*.md")
	} else {
		args = append(args, mdFiles...)
	}

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// ParseOutput parses markdownlint text output.
func (t *MarkdownlintTool) ParseOutput(output string) []Issue {
	if strings.TrimSpace(output) == "" {
		return []Issue{}
	}

	var issues []Issue
	// Format: file:line rule/alias description
	// Example: README.md:10 MD013/line-length Line length
	re := regexp.MustCompile(`^(.+):(\d+)\s+(\S+)\s+(.+)$`)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) == 5 {
			lineNum, _ := strconv.Atoi(matches[2])
			issues = append(issues, Issue{
				File:     matches[1],
				Line:     lineNum,
				Severity: "warning",
				Rule:     matches[3],
				Message:  matches[4],
			})
		}
	}

	return issues
}

// Ensure Markdown tools implement QualityTool interface.
var (
	_ QualityTool = (*MarkdownlintTool)(nil)
)
