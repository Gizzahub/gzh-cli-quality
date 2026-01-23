// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tools

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// ClangFormatTool implements C/C++ formatting using clang-format.
type ClangFormatTool struct {
	*BaseTool
}

// NewClangFormatTool creates a new clang-format tool.
func NewClangFormatTool() *ClangFormatTool {
	tool := &ClangFormatTool{
		BaseTool: NewBaseTool("clang-format", "C/C++", "clang-format", FORMAT),
	}

	tool.SetInstallCommand([]string{"pacman", "-S", "--noconfirm", "clang"})
	tool.SetConfigPatterns([]string{".clang-format", "_clang-format"})

	return tool
}

// BuildCommand builds the clang-format command.
func (t *ClangFormatTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{"-i"} // In-place formatting

	// Add style if config not found
	if options.ConfigFile != "" {
		args = append(args, "--style=file:"+options.ConfigFile)
	}

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Filter C/C++ files
	cppFiles := FilterFilesByExtensions(files, []string{".c", ".h", ".cpp", ".hpp", ".cc", ".cxx", ".hxx"})
	if len(cppFiles) > 0 {
		args = append(args, cppFiles...)
	}

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// ClangTidyTool implements C/C++ linting using clang-tidy.
type ClangTidyTool struct {
	*BaseTool
}

// NewClangTidyTool creates a new clang-tidy tool.
func NewClangTidyTool() *ClangTidyTool {
	tool := &ClangTidyTool{
		BaseTool: NewBaseTool("clang-tidy", "C/C++", "clang-tidy", LINT),
	}

	tool.SetInstallCommand([]string{"pacman", "-S", "--noconfirm", "clang"})
	tool.SetConfigPatterns([]string{".clang-tidy"})

	return tool
}

// BuildCommand builds the clang-tidy command.
func (t *ClangTidyTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{}

	// Add config file if specified
	if options.ConfigFile != "" {
		args = append(args, "--config-file="+options.ConfigFile)
	}

	// Add fix flag if requested
	if options.Fix {
		args = append(args, "--fix")
	}

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Filter C/C++ files
	cppFiles := FilterFilesByExtensions(files, []string{".c", ".cpp", ".cc", ".cxx"})
	if len(cppFiles) > 0 {
		args = append(args, cppFiles...)
	}

	// Add -- to separate clang-tidy args from compiler args
	args = append(args, "--")

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// ParseOutput parses clang-tidy text output.
func (t *ClangTidyTool) ParseOutput(output string) []Issue {
	if strings.TrimSpace(output) == "" {
		return []Issue{}
	}

	var issues []Issue
	// Format: file:line:column: severity: message [check-name]
	re := regexp.MustCompile(`^(.+):(\d+):(\d+):\s+(warning|error|note):\s+(.+)\s+\[(.+)\]$`)

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

// Ensure C/C++ tools implement QualityTool interface.
var (
	_ QualityTool = (*ClangFormatTool)(nil)
	_ QualityTool = (*ClangTidyTool)(nil)
)
