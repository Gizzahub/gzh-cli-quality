// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tools

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// YamllintTool implements YAML linting using yamllint.
type YamllintTool struct {
	*BaseTool
}

// NewYamllintTool creates a new yamllint tool.
func NewYamllintTool() *YamllintTool {
	tool := &YamllintTool{
		BaseTool: NewBaseTool("yamllint", "YAML", "yamllint", LINT),
	}

	tool.SetInstallCommand([]string{"uv", "tool", "install", "yamllint"})
	tool.SetConfigPatterns([]string{".yamllint", ".yamllint.yaml", ".yamllint.yml"})

	return tool
}

// BuildCommand builds the yamllint command.
func (t *YamllintTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{"-f", "parsable"} // Parsable output format

	// Add config file if specified
	if options.ConfigFile != "" {
		args = append(args, "-c", options.ConfigFile)
	}

	// Add strict mode
	args = append(args, "-s")

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Filter YAML files
	yamlFiles := FilterFilesByExtensions(files, []string{".yaml", ".yml"})
	if len(yamlFiles) > 0 {
		args = append(args, yamlFiles...)
	} else {
		args = append(args, ".")
	}

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// ParseOutput parses yamllint parsable output.
func (t *YamllintTool) ParseOutput(output string) []Issue {
	if strings.TrimSpace(output) == "" {
		return []Issue{}
	}

	var issues []Issue
	// Format: file:line:column: [severity] message (rule)
	re := regexp.MustCompile(`^(.+):(\d+):(\d+):\s+\[(warning|error)\]\s+(.+)\s+\((.+)\)$`)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) == 7 {
			lineNum, _ := strconv.Atoi(matches[2])
			colNum, _ := strconv.Atoi(matches[3])

			issues = append(issues, Issue{
				File:     matches[1],
				Line:     lineNum,
				Column:   colNum,
				Severity: matches[4],
				Message:  matches[5],
				Rule:     matches[6],
			})
		}
	}

	return issues
}

// Ensure YAML tools implement QualityTool interface.
var (
	_ QualityTool = (*YamllintTool)(nil)
)
