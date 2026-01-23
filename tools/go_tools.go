// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tools

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// GofumptTool implements Go formatting using gofumpt.
type GofumptTool struct {
	*BaseTool
}

// NewGofumptTool creates a new gofumpt tool.
func NewGofumptTool() *GofumptTool {
	tool := &GofumptTool{
		BaseTool: NewBaseTool("gofumpt", "Go", "gofumpt", FORMAT),
	}

	tool.SetInstallCommand([]string{"go", "install", "mvdan.cc/gofumpt@latest"})
	tool.SetConfigPatterns([]string{".gofumpt"})

	return tool
}

// BuildCommand builds the gofumpt command.
func (t *GofumptTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{"-w"} // Always write changes

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Filter only Go files
	goFiles := FilterFilesByExtensions(files, []string{".go"})
	args = append(args, goFiles...)

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// GoimportsTool implements Go import formatting using goimports.
type GoimportsTool struct {
	*BaseTool
}

// NewGoimportsTool creates a new goimports tool.
func NewGoimportsTool() *GoimportsTool {
	tool := &GoimportsTool{
		BaseTool: NewBaseTool("goimports", "Go", "goimports", FORMAT),
	}

	tool.SetInstallCommand([]string{"go", "install", "golang.org/x/tools/cmd/goimports@latest"})

	return tool
}

// BuildCommand builds the goimports command.
func (t *GoimportsTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{"-w"} // Always write changes

	// Add local import setting if project root is available
	if options.ProjectRoot != "" {
		// Try to determine module name from go.mod
		if modName := getGoModuleName(options.ProjectRoot); modName != "" {
			args = append(args, "-local", modName)
		}
	}

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Filter only Go files
	goFiles := FilterFilesByExtensions(files, []string{".go"})
	args = append(args, goFiles...)

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// GolangciLintTool implements Go linting using golangci-lint.
type GolangciLintTool struct {
	*BaseTool
}

// NewGolangciLintTool creates a new golangci-lint tool.
func NewGolangciLintTool() *GolangciLintTool {
	tool := &GolangciLintTool{
		BaseTool: NewBaseTool("golangci-lint", "Go", "golangci-lint", LINT),
	}

	tool.SetInstallCommand([]string{"go", "install", "github.com/golangci/golangci-lint/cmd/golangci-lint@latest"})
	tool.SetConfigPatterns([]string{".golangci.yml", ".golangci.yaml", "golangci.yml", "golangci.yaml"})

	return tool
}

// BuildCommand builds the golangci-lint command.
func (t *GolangciLintTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{"run"}

	// Add config file if specified
	if options.ConfigFile != "" {
		args = append(args, "-c", options.ConfigFile)
	}

	// Add fix flag if requested
	if options.Fix {
		args = append(args, "--fix")
	}

	// Output format for parsing
	args = append(args, "--out-format", "json")

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Add file patterns or directories
	if len(files) > 0 {
		// Filter only Go files and their directories
		goFiles := FilterFilesByExtensions(files, []string{".go"})
		if len(goFiles) > 0 {
			// Get unique directories
			dirs := make(map[string]bool)
			for _, file := range goFiles {
				dirs[filepath.Dir(file)] = true
			}

			for dir := range dirs {
				args = append(args, dir)
			}
		}
	} else {
		args = append(args, "./...")
	}

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// ParseOutput parses golangci-lint JSON output.
func (t *GolangciLintTool) ParseOutput(output string) []Issue {
	if strings.TrimSpace(output) == "" {
		return []Issue{}
	}

	var lintResults struct {
		Issues []struct {
			FromLinter  string   `json:"FromLinter"`
			Text        string   `json:"Text"`
			Severity    string   `json:"Severity"`
			SourceLines []string `json:"SourceLines"`
			Replacement *struct {
				NewLines []string `json:"NewLines"`
			} `json:"Replacement,omitempty"`
			Pos struct {
				Filename string `json:"Filename"`
				Offset   int    `json:"Offset"`
				Line     int    `json:"Line"`
				Column   int    `json:"Column"`
			} `json:"Pos"`
		} `json:"Issues"`
	}

	if err := json.Unmarshal([]byte(output), &lintResults); err != nil {
		// Fallback to plain text parsing
		return t.parseTextOutput(output)
	}

	issues := make([]Issue, 0, len(lintResults.Issues))
	for _, item := range lintResults.Issues {
		issue := Issue{
			File:     item.Pos.Filename,
			Line:     item.Pos.Line,
			Column:   item.Pos.Column,
			Severity: item.Severity,
			Rule:     item.FromLinter,
			Message:  item.Text,
		}

		if item.Replacement != nil && len(item.Replacement.NewLines) > 0 {
			issue.Suggestion = strings.Join(item.Replacement.NewLines, "\n")
		}

		issues = append(issues, issue)
	}

	return issues
}

// parseTextOutput parses plain text output as fallback.
func (t *GolangciLintTool) parseTextOutput(output string) []Issue {
	return ParseTextLines(output, GolangciLintParseConfig)
}

// getGoModuleName extracts module name from go.mod file.
func getGoModuleName(projectRoot string) string {
	cmd := exec.Command("go", "list", "-m")
	cmd.Dir = projectRoot

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// GosecTool implements Go security scanning using gosec.
type GosecTool struct {
	*BaseTool
}

// NewGosecTool creates a new gosec tool.
func NewGosecTool() *GosecTool {
	tool := &GosecTool{
		BaseTool: NewBaseTool("gosec", "Go", "gosec", LINT),
	}

	tool.SetInstallCommand([]string{"go", "install", "github.com/securego/gosec/v2/cmd/gosec@latest"})

	return tool
}

// BuildCommand builds the gosec command.
func (t *GosecTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{"-fmt=json", "-quiet"}

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Add file patterns or directories
	if len(files) > 0 {
		goFiles := FilterFilesByExtensions(files, []string{".go"})
		if len(goFiles) > 0 {
			dirs := make(map[string]bool)
			for _, file := range goFiles {
				dirs[filepath.Dir(file)] = true
			}
			for dir := range dirs {
				args = append(args, dir+"/...")
			}
		}
	} else {
		args = append(args, "./...")
	}

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// ParseOutput parses gosec JSON output.
func (t *GosecTool) ParseOutput(output string) []Issue {
	if strings.TrimSpace(output) == "" {
		return []Issue{}
	}

	var gosecResults struct {
		Issues []struct {
			Severity   string `json:"severity"`
			Confidence string `json:"confidence"`
			RuleID     string `json:"rule_id"`
			Details    string `json:"details"`
			File       string `json:"file"`
			Line       string `json:"line"`
			Column     string `json:"column"`
		} `json:"Issues"`
	}

	if err := json.Unmarshal([]byte(output), &gosecResults); err != nil {
		return []Issue{}
	}

	issues := make([]Issue, 0, len(gosecResults.Issues))
	for _, item := range gosecResults.Issues {
		line := 0
		col := 0
		fmt.Sscanf(item.Line, "%d", &line)
		fmt.Sscanf(item.Column, "%d", &col)

		issues = append(issues, Issue{
			File:     item.File,
			Line:     line,
			Column:   col,
			Severity: item.Severity,
			Rule:     item.RuleID,
			Message:  item.Details,
		})
	}

	return issues
}

// GovulncheckTool implements Go vulnerability scanning using govulncheck.
type GovulncheckTool struct {
	*BaseTool
}

// NewGovulncheckTool creates a new govulncheck tool.
func NewGovulncheckTool() *GovulncheckTool {
	tool := &GovulncheckTool{
		BaseTool: NewBaseTool("govulncheck", "Go", "govulncheck", LINT),
	}

	tool.SetInstallCommand([]string{"go", "install", "golang.org/x/vuln/cmd/govulncheck@latest"})

	return tool
}

// BuildCommand builds the govulncheck command.
func (t *GovulncheckTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{"-json"}

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// govulncheck works on packages, not individual files
	args = append(args, "./...")

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// ParseOutput parses govulncheck JSON output.
func (t *GovulncheckTool) ParseOutput(output string) []Issue {
	if strings.TrimSpace(output) == "" {
		return []Issue{}
	}

	// govulncheck outputs JSON lines, parse each line
	var issues []Issue
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var msg struct {
			Finding *struct {
				OSV   string `json:"osv"`
				Trace []struct {
					Module  string `json:"module"`
					Package string `json:"package"`
				} `json:"trace"`
			} `json:"finding"`
		}

		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		if msg.Finding != nil && len(msg.Finding.Trace) > 0 {
			issues = append(issues, Issue{
				Severity: "HIGH",
				Rule:     msg.Finding.OSV,
				Message:  fmt.Sprintf("Vulnerability in %s", msg.Finding.Trace[0].Module),
			})
		}
	}

	return issues
}

// GciTool implements Go import grouping using gci.
type GciTool struct {
	*BaseTool
}

// NewGciTool creates a new gci tool.
func NewGciTool() *GciTool {
	tool := &GciTool{
		BaseTool: NewBaseTool("gci", "Go", "gci", FORMAT),
	}

	tool.SetInstallCommand([]string{"go", "install", "github.com/daixiang0/gci@latest"})
	tool.SetConfigPatterns([]string{".gci.yml", ".gci.yaml"})

	return tool
}

// BuildCommand builds the gci command.
func (t *GciTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{"write"}

	// Default section order: standard, default, prefix
	if options.ProjectRoot != "" {
		if modName := getGoModuleName(options.ProjectRoot); modName != "" {
			args = append(args, "-s", "standard", "-s", "default", "-s", "prefix("+modName+")")
		} else {
			args = append(args, "-s", "standard", "-s", "default")
		}
	} else {
		args = append(args, "-s", "standard", "-s", "default")
	}

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Filter only Go files
	goFiles := FilterFilesByExtensions(files, []string{".go"})
	if len(goFiles) > 0 {
		args = append(args, goFiles...)
	} else {
		args = append(args, ".")
	}

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// GolinesTool implements Go long line splitting using golines.
type GolinesTool struct {
	*BaseTool
}

// NewGolinesTool creates a new golines tool.
func NewGolinesTool() *GolinesTool {
	tool := &GolinesTool{
		BaseTool: NewBaseTool("golines", "Go", "golines", FORMAT),
	}

	tool.SetInstallCommand([]string{"go", "install", "github.com/segmentio/golines@latest"})

	return tool
}

// BuildCommand builds the golines command.
func (t *GolinesTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{"-w", "-m", "120"} // Write changes, max 120 chars

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Filter only Go files
	goFiles := FilterFilesByExtensions(files, []string{".go"})
	if len(goFiles) > 0 {
		args = append(args, goFiles...)
	} else {
		args = append(args, ".")
	}

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// Ensure Go tools implement QualityTool interface.
var (
	_ QualityTool = (*GofumptTool)(nil)
	_ QualityTool = (*GoimportsTool)(nil)
	_ QualityTool = (*GolangciLintTool)(nil)
	_ QualityTool = (*GosecTool)(nil)
	_ QualityTool = (*GovulncheckTool)(nil)
	_ QualityTool = (*GciTool)(nil)
	_ QualityTool = (*GolinesTool)(nil)
)
