// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tools

import (
	"encoding/json"
	"os/exec"
	"strings"
)

// SqlfluffTool implements SQL linting and formatting using sqlfluff.
type SqlfluffTool struct {
	*BaseTool
}

// NewSqlfluffTool creates a new sqlfluff tool.
func NewSqlfluffTool() *SqlfluffTool {
	tool := &SqlfluffTool{
		BaseTool: NewBaseTool("sqlfluff", "SQL", "sqlfluff", BOTH),
	}

	tool.SetInstallCommand([]string{"uv", "tool", "install", "sqlfluff"})
	tool.SetConfigPatterns([]string{".sqlfluff", "setup.cfg", "pyproject.toml"})

	return tool
}

// BuildCommand builds the sqlfluff command.
func (t *SqlfluffTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	var args []string

	// Determine operation mode
	if options.FormatOnly || options.Fix {
		args = append(args, "fix")
	} else {
		args = append(args, "lint")
	}

	// Output format for parsing (only in lint mode)
	if !options.FormatOnly && !options.Fix {
		args = append(args, "--format", "json")
	}

	// Add dialect (default to ansi)
	args = append(args, "--dialect", "ansi")

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Filter SQL files
	sqlFiles := FilterFilesByExtensions(files, []string{".sql"})
	if len(sqlFiles) > 0 {
		args = append(args, sqlFiles...)
	} else {
		args = append(args, ".")
	}

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// ParseOutput parses sqlfluff JSON output.
func (t *SqlfluffTool) ParseOutput(output string) []Issue {
	if strings.TrimSpace(output) == "" {
		return []Issue{}
	}

	var sqlfluffResults []struct {
		Filepath   string `json:"filepath"`
		Violations []struct {
			StartLineNo   int    `json:"start_line_no"`
			StartLinePos  int    `json:"start_line_pos"`
			Code          string `json:"code"`
			Description   string `json:"description"`
			Name          string `json:"name"`
		} `json:"violations"`
	}

	if err := json.Unmarshal([]byte(output), &sqlfluffResults); err != nil {
		return []Issue{}
	}

	var issues []Issue
	for _, file := range sqlfluffResults {
		for _, v := range file.Violations {
			issues = append(issues, Issue{
				File:     file.Filepath,
				Line:     v.StartLineNo,
				Column:   v.StartLinePos,
				Severity: "warning",
				Rule:     v.Code,
				Message:  v.Description,
			})
		}
	}

	return issues
}

// Ensure SQL tools implement QualityTool interface.
var (
	_ QualityTool = (*SqlfluffTool)(nil)
)
