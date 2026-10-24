// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package tools

import (
	"context"
	"debug/buildinfo"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// BaseTool provides common functionality for quality tools.
type BaseTool struct {
	name           string
	language       string
	toolType       ToolType
	executable     string
	installCmd     []string
	configPatterns []string
	// versionUncacheable marks a version output that the executable file
	// alone does not determine; see VersionProber.VersionCacheable.
	versionUncacheable bool
	// versionCommand, when set, replaces versionFlags; see SetVersionCommand.
	versionCommand []string
	adapter        interface {
		BuildCommand([]string, ExecuteOptions) *exec.Cmd
		ParseOutput(string) []Issue
	}
}

// NewBaseTool creates a new base tool.
func NewBaseTool(name, language, executable string, toolType ToolType) *BaseTool {
	return &BaseTool{
		name:       name,
		language:   language,
		toolType:   toolType,
		executable: executable,
	}
}

// Name returns the tool name.
func (t *BaseTool) Name() string {
	return t.name
}

// Language returns the programming language.
func (t *BaseTool) Language() string {
	return t.language
}

// Type returns the tool type.
func (t *BaseTool) Type() ToolType {
	return t.toolType
}

// IsAvailable checks if the tool is installed and available.
func (t *BaseTool) IsAvailable() bool {
	_, err := exec.LookPath(t.executable)
	return err == nil
}

// SetInstallCommand sets the command to install this tool.
func (t *BaseTool) SetInstallCommand(cmd []string) {
	t.installCmd = cmd
}

// Install attempts to install the tool automatically.
func (t *BaseTool) Install() error {
	if len(t.installCmd) == 0 {
		return fmt.Errorf("no install command configured for %s", t.name)
	}

	cmd := exec.Command(t.installCmd[0], t.installCmd[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install %s: %w\nOutput: %s", t.name, err, string(output))
	}

	return nil
}

// versionFlags are tried in order by GetVersion; the first non-empty output wins.
var versionFlags = []string{"--version", "-v", "-V", "version"}

// VersionProbe reports the executable and the arguments GetVersion runs: the
// command set by SetVersionCommand, or else the ordered versionFlags.
func (t *BaseTool) VersionProbe() (executable string, flags []string) {
	if len(t.versionCommand) > 0 {
		return t.executable, slices.Clone(t.versionCommand)
	}
	return t.executable, slices.Clone(versionFlags)
}

// SetVersionCommand makes GetVersion run the executable once with args instead
// of trying versionFlags, for a tool that is a subcommand of its executable
// (`cargo clippy --version` reports clippy, `cargo --version` reports cargo).
func (t *BaseTool) SetVersionCommand(args ...string) {
	t.versionCommand = slices.Clone(args)
}

// VersionCacheable reports whether callers may reuse a version recorded for
// the same executable file. It is true unless SetVersionUncacheable was called.
func (t *BaseTool) VersionCacheable() bool {
	return !t.versionUncacheable
}

// SetVersionUncacheable declares that the version output depends on more than
// the executable file (environment, network), so it must be probed every time.
func (t *BaseTool) SetVersionUncacheable() {
	t.versionUncacheable = true
}

// GetVersion returns the version of the installed tool. A Go program with no
// version flag (goimports) is identified by the build information embedded in
// its executable. "unknown" means neither source named a version.
func (t *BaseTool) GetVersion() (string, error) {
	path, err := exec.LookPath(t.executable)
	if err != nil {
		return "", fmt.Errorf("tool %s is not installed", t.name)
	}

	if len(t.versionCommand) > 0 {
		output, err := exec.Command(t.executable, t.versionCommand...).Output()
		if err != nil {
			return "", fmt.Errorf("failed to get %s version: %w", t.name, err)
		}
		if version := strings.TrimSpace(string(output)); version != "" {
			return version, nil
		}
		return "unknown", nil
	}

	for _, flag := range versionFlags {
		cmd := exec.Command(t.executable, flag)
		output, err := cmd.Output()
		if err == nil {
			version := strings.TrimSpace(string(output))
			if version != "" {
				return version, nil
			}
		}
	}

	if info, err := buildinfo.ReadFile(path); err == nil {
		if version := buildInfoVersion(info); version != "" {
			return version, nil
		}
	}

	return "unknown", nil
}

// buildInfoVersion names a Go executable by its main module version and the
// Go release that built it, since the standard library's go/format and
// go/printer shape a formatter's output too. A build from a modified checkout
// returns "": two such builds can differ while sharing one version (Go stamps
// both "+dirty"). A "(devel)" build counts only with a clean VCS revision.
func buildInfoVersion(info *buildinfo.BuildInfo) string {
	if info.Main.Path == "" {
		return ""
	}
	var revision, modified string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value
		}
	}
	if modified == "true" {
		return ""
	}
	version := info.Main.Version
	if version == "" || version == "(devel)" {
		if revision == "" || modified != "false" {
			return ""
		}
		version = "(devel) " + revision
	}
	return info.Main.Path + " " + version + " " + info.GoVersion
}

// Upgrade attempts to upgrade the tool to the latest version.
func (t *BaseTool) Upgrade() error {
	if !t.IsAvailable() {
		return fmt.Errorf("tool %s is not installed, use Install() instead", t.name)
	}

	// For most tools, upgrade is the same as install
	return t.Install()
}

// SetConfigPatterns sets the configuration file patterns to search for.
func (t *BaseTool) SetConfigPatterns(patterns []string) {
	t.configPatterns = patterns
}

// FindConfigFiles returns configuration files the tool would use.
func (t *BaseTool) FindConfigFiles(projectRoot string) []string {
	var configs []string

	for _, pattern := range t.configPatterns {
		configPath := filepath.Join(projectRoot, pattern)
		if _, err := os.Stat(configPath); err == nil {
			configs = append(configs, configPath)
		}
	}

	return configs
}

// ExecuteCommand runs a command and returns the result.
func (t *BaseTool) ExecuteCommand(ctx context.Context, cmd *exec.Cmd, files []string) (*Result, error) {
	startTime := time.Now()

	result := &Result{
		Tool:     t.name,
		Language: t.language,
		Success:  false,
		Issues:   []Issue{},
	}

	// Rebuild with the caller's context while preserving command execution options.
	contextCmd := exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
	contextCmd.Args = cmd.Args
	contextCmd.Dir, contextCmd.Env, contextCmd.Stdin = cmd.Dir, cmd.Env, cmd.Stdin
	contextCmd.Stdout, contextCmd.Stderr = cmd.Stdout, cmd.Stderr
	contextCmd.ExtraFiles, contextCmd.SysProcAttr = cmd.ExtraFiles, cmd.SysProcAttr
	contextCmd.WaitDelay = time.Second
	output, err := contextCmd.CombinedOutput()
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	duration := time.Since(startTime)
	result.Duration = duration
	result.Output = string(output)
	result.FilesProcessed = len(files)

	if err != nil {
		result.Error = err.Error()
		return result, nil //nolint:nilerr // 오류를 결과에 캡처하여 반환하므로 에러는 무시
	}

	result.Success = true
	return result, nil
}

// ParseOutput parses tool output into issues (to be implemented by specific tools).
func (t *BaseTool) ParseOutput(output string) []Issue {
	// Default implementation returns empty slice
	// Specific tools should override this method
	return []Issue{}
}

// BuildCommand builds the command to execute (to be implemented by specific tools).
func (t *BaseTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	// Default implementation - specific tools should override
	args := append([]string{}, files...)
	if len(options.ExtraArgs) > 0 {
		args = append(args, options.ExtraArgs...)
	}

	cmd := exec.Command(t.executable, args...)

	// Set working directory
	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	// Set environment variables
	if len(options.Env) > 0 {
		env := os.Environ()
		for k, v := range options.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	return cmd
}

// Execute runs the tool on the specified files.
func (t *BaseTool) Execute(ctx context.Context, files []string, options ExecuteOptions) (*Result, error) {
	if !t.IsAvailable() {
		return &Result{
			Tool:     t.name,
			Language: t.language,
			Success:  false,
			Error:    fmt.Sprintf("tool %s is not available", t.name),
		}, nil
	}

	adapter := t.adapter
	if adapter == nil {
		adapter = t
	}
	cmd := adapter.BuildCommand(files, options)
	applyExecutionOptions(cmd, options)
	result, err := t.ExecuteCommand(ctx, cmd, files)
	if err != nil {
		return result, err
	}

	// Successful tools may also report warnings or findings.
	if result.Output != "" {
		result.Issues = adapter.ParseOutput(result.Output)
	}

	// These formatters report differences on stdout without a failing exit code.
	// Their check modes emit only filenames or diffs, so any output means the
	// requested formatting check did not pass.
	if result.Success && !options.Fix && strings.TrimSpace(result.Output) != "" {
		switch t.name {
		case "gofumpt", "goimports", "gci", "golines", "shfmt":
			result.Success = false
			result.Error = "formatting changes required"
		}
	}

	return result, nil
}

// FilterFilesByExtensions filters files by supported extensions.
func FilterFilesByExtensions(files, extensions []string) []string {
	var filtered []string
	extMap := make(map[string]bool)

	for _, ext := range extensions {
		extMap[strings.ToLower(ext)] = true
	}

	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file))
		if extMap[ext] {
			filtered = append(filtered, file)
		}
	}

	return filtered
}

// applyExecutionOptions keeps adapter execution consistent with BaseTool execution.
func applyExecutionOptions(cmd *exec.Cmd, options ExecuteOptions) {
	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}
	if len(options.Env) > 0 {
		cmd.Env = cmd.Environ()
		for key, value := range options.Env {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
}
