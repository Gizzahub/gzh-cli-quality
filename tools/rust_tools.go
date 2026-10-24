// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	severityInfo    = "info"
	severityError   = "error"
	severityWarning = "warning"
)

// RustfmtTool implements Rust formatting using rustfmt.
type RustfmtTool struct {
	*BaseTool
}

// NewRustfmtTool creates a new rustfmt tool.
func NewRustfmtTool() *RustfmtTool {
	tool := &RustfmtTool{
		BaseTool: NewBaseTool("rustfmt", "Rust", "rustfmt", FORMAT),
	}

	tool.SetInstallCommand([]string{"rustup", "component", "add", "rustfmt"})
	tool.SetVersionUncacheable() // rustup may select a component using cwd or environment.
	tool.SetConfigPatterns([]string{"rustfmt.toml", ".rustfmt.toml"})

	tool.adapter = tool
	return tool
}

// VersionProbe reports the actual configured command; toolchain probes are not cached.
func (t *RustfmtTool) VersionProbe() (executable string, flags []string) {
	return t.executable, rustProbeArgs(t.BaseTool, []string{"--version"})
}

// GetVersion reports the rustfmt component selected by the current process context.
func (t *RustfmtTool) GetVersion() (string, error) {
	return t.GetVersionForExecution(context.Background(), ExecuteOptions{})
}

// GetVersionForExecution probes rustfmt in the same directory and environment as execution.
func (t *RustfmtTool) GetVersionForExecution(ctx context.Context, options ExecuteOptions) (string, error) {
	return rustComponentVersion(ctx, t.BaseTool, []string{"--version"}, "rustfmt", options)
}

// BuildCommand builds the rustfmt command.
func (t *RustfmtTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{}
	if !options.Fix {
		args = append(args, "--check")
	}

	// Add config file if specified
	if options.ConfigFile != "" {
		args = append(args, "--config-path", options.ConfigFile)
	}

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Filter only Rust files
	rustFiles := FilterFilesByExtensions(files, []string{".rs"})
	args = append(args, rustFiles...)

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// ClippyTool implements Rust linting using clippy.
type ClippyTool struct {
	*BaseTool
}

// NewClippyTool creates a new clippy tool.
func NewClippyTool() *ClippyTool {
	tool := &ClippyTool{
		BaseTool: NewBaseTool("clippy", "Rust", "cargo", LINT),
	}

	// cargo is a rustup proxy or a separate install; clippy-driver may change
	// without cargo, so the version is probed every time.
	tool.SetVersionCommand("clippy", "--version")
	tool.SetVersionUncacheable()

	tool.SetInstallCommand([]string{"rustup", "component", "add", "clippy"})
	tool.SetConfigPatterns([]string{"clippy.toml", ".clippy.toml", "Cargo.toml"})

	tool.adapter = tool
	return tool
}

// VersionProbe reports the actual configured command; component probes are not cached.
func (t *ClippyTool) VersionProbe() (executable string, flags []string) {
	return t.executable, rustProbeArgs(t.BaseTool, []string{"clippy", "--version"})
}

// GetVersion reports clippy's version, rather than cargo's launcher version.
func (t *ClippyTool) GetVersion() (string, error) {
	return t.GetVersionForExecution(context.Background(), ExecuteOptions{})
}

// GetVersionForExecution probes the component selected by the execution's directory
// and environment, including rustup's toolchain overrides.
func (t *ClippyTool) GetVersionForExecution(ctx context.Context, options ExecuteOptions) (string, error) {
	return rustComponentVersion(ctx, t.BaseTool, []string{"clippy", "--version"}, "clippy", options)
}

func rustProbeArgs(tool *BaseTool, defaults []string) []string {
	if len(tool.versionCommand) > 0 {
		return slices.Clone(tool.versionCommand)
	}
	return slices.Clone(defaults)
}

var rustVersionToken = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?(\+[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$`)

func rustComponentVersion(ctx context.Context, tool *BaseTool, args []string, component string, options ExecuteOptions) (string, error) {
	cmd := exec.CommandContext(ctx, tool.executable, rustProbeArgs(tool, args)...)
	applyExecutionOptions(cmd, options)
	cmd.WaitDelay = time.Second
	output, err := cmd.Output()
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && strings.TrimSpace(string(exitErr.Stderr)) != "" {
			return "", fmt.Errorf("failed to get %s version: %w: %s", tool.name, err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("failed to get %s version: %w", tool.name, err)
	}
	version := strings.TrimSpace(string(output))
	fields := strings.Fields(version)
	if len(fields) < 2 || fields[0] != component || !rustVersionToken.MatchString(fields[1]) {
		return "unknown", nil
	}
	return version, nil
}

// BuildCommand builds the clippy command.
func (t *ClippyTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{"clippy"}

	// Add fix flag if requested
	if options.Fix {
		args = append(args, "--fix")
	}

	// Output format for parsing
	args = append(args, "--message-format", "json")

	// Add extra flags
	args = append(args, options.ExtraArgs...)

	// Clippy works on the entire project, not individual files
	args = append(args, "--", "-D", "warnings")

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// ParseOutput parses clippy JSON output.
func (t *ClippyTool) ParseOutput(output string) []Issue {
	if strings.TrimSpace(output) == "" {
		return []Issue{}
	}

	lines := strings.Split(output, "\n")
	issues := make([]Issue, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var clippyMessage struct {
			Message struct {
				Message string `json:"message"`
				Code    *struct {
					Code string `json:"code"`
				} `json:"code"`
				Level string `json:"level"`
				Spans []struct {
					FileName    string `json:"file_name"`
					LineStart   int    `json:"line_start"`
					ColumnStart int    `json:"column_start"`
				} `json:"spans"`
			} `json:"message"`
			Target struct {
				Name string `json:"name"`
			} `json:"target"`
		}

		if err := json.Unmarshal([]byte(line), &clippyMessage); err != nil {
			continue
		}

		msg := clippyMessage.Message
		if len(msg.Spans) == 0 {
			continue
		}

		severity := severityInfo
		switch msg.Level {
		case severityError:
			severity = severityError
		case severityWarning:
			severity = severityWarning
		}

		rule := ""
		if msg.Code != nil {
			rule = msg.Code.Code
		}

		span := msg.Spans[0]
		issues = append(issues, Issue{
			File:     span.FileName,
			Line:     span.LineStart,
			Column:   span.ColumnStart,
			Severity: severity,
			Rule:     rule,
			Message:  msg.Message,
		})
	}

	return issues
}

// CargoFmtTool implements Rust formatting using cargo fmt.
type CargoFmtTool struct {
	*BaseTool
}

// NewCargoFmtTool creates a new cargo fmt tool.
func NewCargoFmtTool() *CargoFmtTool {
	tool := &CargoFmtTool{
		BaseTool: NewBaseTool("cargo-fmt", "Rust", "cargo", FORMAT),
	}

	// rustfmt may change without cargo; see NewClippyTool.
	tool.SetVersionCommand("fmt", "--version")
	tool.SetVersionUncacheable()

	tool.SetInstallCommand([]string{"rustup", "component", "add", "rustfmt"})
	tool.SetConfigPatterns([]string{"rustfmt.toml", ".rustfmt.toml"})

	tool.adapter = tool
	return tool
}

// VersionProbe reports the actual configured command; component probes are not cached.
func (t *CargoFmtTool) VersionProbe() (executable string, flags []string) {
	return t.executable, rustProbeArgs(t.BaseTool, []string{"fmt", "--version"})
}

// GetVersion reports rustfmt's version, rather than cargo's launcher version.
func (t *CargoFmtTool) GetVersion() (string, error) {
	return t.GetVersionForExecution(context.Background(), ExecuteOptions{})
}

// GetVersionForExecution probes rustfmt in the same execution context as cargo fmt.
func (t *CargoFmtTool) GetVersionForExecution(ctx context.Context, options ExecuteOptions) (string, error) {
	return rustComponentVersion(ctx, t.BaseTool, []string{"fmt", "--version"}, "rustfmt", options)
}

// BuildCommand builds the cargo fmt command.
func (t *CargoFmtTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{"fmt"}
	if !options.Fix {
		args = append(args, "--check")
	}

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// cargo fmt works on the entire project
	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// Ensure Rust tools implement QualityTool interface.
var (
	_ QualityTool = (*RustfmtTool)(nil)
	_ QualityTool = (*ClippyTool)(nil)
	_ QualityTool = (*CargoFmtTool)(nil)
)
