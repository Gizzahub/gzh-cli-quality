// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tools

import (
	"encoding/json"
	"os/exec"
	"strings"
)

// BufTool implements Protobuf linting and formatting using buf.
type BufTool struct {
	*BaseTool
}

// NewBufTool creates a new buf tool.
func NewBufTool() *BufTool {
	tool := &BufTool{
		BaseTool: NewBaseTool("buf", "Protobuf", "buf", BOTH),
	}

	tool.SetInstallCommand([]string{"go", "install", "github.com/bufbuild/buf/cmd/buf@latest"})
	tool.SetConfigPatterns([]string{"buf.yaml", "buf.gen.yaml"})

	return tool
}

// BuildCommand builds the buf command.
func (t *BufTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	var args []string

	// Determine operation mode
	if options.FormatOnly {
		args = append(args, "format", "-w") // Write changes
	} else {
		args = append(args, "lint", "--error-format", "json")
	}

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// buf works on directories, not individual files
	if len(files) > 0 {
		// Get directory from first proto file
		protoFiles := FilterFilesByExtensions(files, []string{".proto"})
		if len(protoFiles) > 0 {
			args = append(args, protoFiles[0])
		}
	}

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// ParseOutput parses buf JSON output.
func (t *BufTool) ParseOutput(output string) []Issue {
	if strings.TrimSpace(output) == "" {
		return []Issue{}
	}

	var issues []Issue

	// buf outputs JSON lines
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var item struct {
			Path        string `json:"path"`
			StartLine   int    `json:"start_line"`
			StartColumn int    `json:"start_column"`
			EndLine     int    `json:"end_line"`
			EndColumn   int    `json:"end_column"`
			Type        string `json:"type"`
			Message     string `json:"message"`
		}

		if err := json.Unmarshal([]byte(line), &item); err != nil {
			continue
		}

		issues = append(issues, Issue{
			File:     item.Path,
			Line:     item.StartLine,
			Column:   item.StartColumn,
			Severity: "error",
			Rule:     item.Type,
			Message:  item.Message,
		})
	}

	return issues
}

// Ensure Protobuf tools implement QualityTool interface.
var (
	_ QualityTool = (*BufTool)(nil)
)
