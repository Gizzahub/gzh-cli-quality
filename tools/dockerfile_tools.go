// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tools

import (
	"encoding/json"
	"os/exec"
	"strings"
)

// HadolintTool implements Dockerfile linting using hadolint.
type HadolintTool struct {
	*BaseTool
}

// NewHadolintTool creates a new hadolint tool.
func NewHadolintTool() *HadolintTool {
	tool := &HadolintTool{
		BaseTool: NewBaseTool("hadolint", "Dockerfile", "hadolint", LINT),
	}

	tool.SetInstallCommand([]string{"brew", "install", "hadolint"})
	tool.SetConfigPatterns([]string{".hadolint.yaml", ".hadolint.yml", "hadolint.yaml"})

	return tool
}

// BuildCommand builds the hadolint command.
func (t *HadolintTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{"-f", "json"} // JSON output for parsing

	// Add config file if specified
	if options.ConfigFile != "" {
		args = append(args, "-c", options.ConfigFile)
	}

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Filter Dockerfile files
	dockerFiles := filterDockerfiles(files)
	if len(dockerFiles) > 0 {
		args = append(args, dockerFiles...)
	} else {
		args = append(args, "Dockerfile")
	}

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// filterDockerfiles filters files to only include Dockerfiles.
func filterDockerfiles(files []string) []string {
	var dockerFiles []string
	for _, file := range files {
		lower := strings.ToLower(file)
		if strings.Contains(lower, "dockerfile") || strings.HasSuffix(lower, ".dockerfile") {
			dockerFiles = append(dockerFiles, file)
		}
	}
	return dockerFiles
}

// ParseOutput parses hadolint JSON output.
func (t *HadolintTool) ParseOutput(output string) []Issue {
	if strings.TrimSpace(output) == "" {
		return []Issue{}
	}

	var hadolintResults []struct {
		Line    int    `json:"line"`
		Code    string `json:"code"`
		Message string `json:"message"`
		Column  int    `json:"column"`
		File    string `json:"file"`
		Level   string `json:"level"`
	}

	if err := json.Unmarshal([]byte(output), &hadolintResults); err != nil {
		return []Issue{}
	}

	issues := make([]Issue, 0, len(hadolintResults))
	for _, item := range hadolintResults {
		severity := "warning"
		switch item.Level {
		case "error":
			severity = "error"
		case "warning":
			severity = "warning"
		case "info", "style":
			severity = "info"
		}

		issues = append(issues, Issue{
			File:     item.File,
			Line:     item.Line,
			Column:   item.Column,
			Severity: severity,
			Rule:     item.Code,
			Message:  item.Message,
		})
	}

	return issues
}

// Ensure Dockerfile tools implement QualityTool interface.
var (
	_ QualityTool = (*HadolintTool)(nil)
)
