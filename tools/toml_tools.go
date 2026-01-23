// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tools

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// TaploTool implements TOML formatting and linting using taplo.
type TaploTool struct {
	*BaseTool
}

// NewTaploTool creates a new taplo tool.
func NewTaploTool() *TaploTool {
	tool := &TaploTool{
		BaseTool: NewBaseTool("taplo", "TOML", "taplo", BOTH),
	}

	tool.SetInstallCommand([]string{"cargo", "install", "taplo-cli"})
	tool.SetConfigPatterns([]string{"taplo.toml", ".taplo.toml"})

	return tool
}

// BuildCommand builds the taplo command.
func (t *TaploTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	var args []string

	// Determine operation mode
	if options.FormatOnly || options.Fix {
		args = append(args, "fmt")
	} else {
		args = append(args, "lint")
	}

	// Add config file if specified
	if options.ConfigFile != "" {
		args = append(args, "--config", options.ConfigFile)
	}

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Filter TOML files
	tomlFiles := FilterFilesByExtensions(files, []string{".toml"})
	if len(tomlFiles) > 0 {
		args = append(args, tomlFiles...)
	} else {
		args = append(args, "**/*.toml")
	}

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// ParseOutput parses taplo text output.
func (t *TaploTool) ParseOutput(output string) []Issue {
	if strings.TrimSpace(output) == "" {
		return []Issue{}
	}

	var issues []Issue
	// Format: error[rule]: message
	//   --> file:line:column
	re := regexp.MustCompile(`(error|warning)\[([^\]]+)\]:\s+(.+)`)
	locRe := regexp.MustCompile(`-->\s+(.+):(\d+):(\d+)`)

	lines := strings.Split(output, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		matches := re.FindStringSubmatch(line)
		if len(matches) == 4 {
			issue := Issue{
				Severity: matches[1],
				Rule:     matches[2],
				Message:  matches[3],
			}

			// Look for location in next line
			if i+1 < len(lines) {
				locMatches := locRe.FindStringSubmatch(lines[i+1])
				if len(locMatches) == 4 {
					issue.File = locMatches[1]
					issue.Line, _ = strconv.Atoi(locMatches[2])
					issue.Column, _ = strconv.Atoi(locMatches[3])
					i++
				}
			}

			issues = append(issues, issue)
		}
	}

	return issues
}

// Ensure TOML tools implement QualityTool interface.
var (
	_ QualityTool = (*TaploTool)(nil)
)
