// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package executor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Gizzahub/gzh-cli-quality/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock tool for testing
type mockTool struct {
	name         string
	language     string
	toolType     tools.ToolType
	executeFunc  func(ctx context.Context, files []string, options tools.ExecuteOptions) (*tools.Result, error)
	validateFunc func() error
}

func (m *mockTool) Name() string       { return m.name }
func (m *mockTool) Language() string   { return m.language }
func (m *mockTool) Type() tools.ToolType { return m.toolType }
func (m *mockTool) IsAvailable() bool  { return m.validateFunc() == nil }
func (m *mockTool) Install() error     { return nil }
func (m *mockTool) GetVersion() (string, error) { return "1.0.0", nil }
func (m *mockTool) Upgrade() error     { return nil }
func (m *mockTool) FindConfigFiles(projectRoot string) []string { return nil }
func (m *mockTool) Execute(ctx context.Context, files []string, options tools.ExecuteOptions) (*tools.Result, error) {
	return m.executeFunc(ctx, files, options)
}

// Mock analyzer for testing
type mockAnalyzer struct {
	analyzeFunc  func(projectRoot string, registry tools.ToolRegistry) (*AnalysisResult, error)
	selectionFunc func(result *AnalysisResult, registry tools.ToolRegistry) map[string][]tools.QualityTool
}

func (m *mockAnalyzer) AnalyzeProject(projectRoot string, registry tools.ToolRegistry) (*AnalysisResult, error) {
	return m.analyzeFunc(projectRoot, registry)
}

func (m *mockAnalyzer) GetOptimalToolSelection(result *AnalysisResult, registry tools.ToolRegistry) map[string][]tools.QualityTool {
	return m.selectionFunc(result, registry)
}

// Mock registry for testing
type mockRegistry struct {
	tools map[string]tools.QualityTool
}

func (m *mockRegistry) Register(tool tools.QualityTool) {
	m.tools[tool.Name()] = tool
}

func (m *mockRegistry) GetTool(name string) (tools.QualityTool, bool) {
	tool, exists := m.tools[name]
	return tool, exists
}

func (m *mockRegistry) GetToolsForLanguage(language string) []tools.QualityTool {
	var result []tools.QualityTool
	for _, tool := range m.tools {
		if tool.Language() == language {
			result = append(result, tool)
		}
	}
	return result
}

func (m *mockRegistry) GetAllTools() []tools.QualityTool {
	var result []tools.QualityTool
	for _, tool := range m.tools {
		result = append(result, tool)
	}
	return result
}

func (m *mockRegistry) FindTool(name string) tools.QualityTool {
	tool, _ := m.GetTool(name)
	return tool
}

func (m *mockRegistry) GetTools() []tools.QualityTool {
	return m.GetAllTools()
}

func (m *mockRegistry) GetToolsByLanguage(language string) []tools.QualityTool {
	return m.GetToolsForLanguage(language)
}

func (m *mockRegistry) GetToolsByType(toolType tools.ToolType) []tools.QualityTool {
	var result []tools.QualityTool
	for _, tool := range m.tools {
		if tool.Type() == toolType {
			result = append(result, tool)
		}
	}
	return result
}

// Helper to create a simple git repository for testing
func setupTestGitRepo(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	err := cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	return tmpDir
}

// Helper to create and commit a file in a git repo
func createAndCommitFile(t *testing.T, repoDir, filename, content string) {
	t.Helper()

	filePath := filepath.Join(repoDir, filename)
	err := os.WriteFile(filePath, []byte(content), 0o644)
	require.NoError(t, err)

	cmd := exec.Command("git", "add", filename)
	cmd.Dir = repoDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "Add "+filename)
	cmd.Dir = repoDir
	err = cmd.Run()
	require.NoError(t, err)
}

// Tests for ParallelExecutor

func TestNewParallelExecutor(t *testing.T) {
	tests := []struct {
		name           string
		maxWorkers     int
		timeout        time.Duration
		expectedWorkers int
		expectedTimeout time.Duration
	}{
		{
			name:           "Valid parameters",
			maxWorkers:     8,
			timeout:        10 * time.Minute,
			expectedWorkers: 8,
			expectedTimeout: 10 * time.Minute,
		},
		{
			name:           "Zero workers defaults to 4",
			maxWorkers:     0,
			timeout:        5 * time.Minute,
			expectedWorkers: 4,
			expectedTimeout: 5 * time.Minute,
		},
		{
			name:           "Negative workers defaults to 4",
			maxWorkers:     -1,
			timeout:        5 * time.Minute,
			expectedWorkers: 4,
			expectedTimeout: 5 * time.Minute,
		},
		{
			name:           "Zero timeout defaults to 5 minutes",
			maxWorkers:     8,
			timeout:        0,
			expectedWorkers: 8,
			expectedTimeout: 5 * time.Minute,
		},
		{
			name:           "Negative timeout defaults to 5 minutes",
			maxWorkers:     8,
			timeout:        -1 * time.Minute,
			expectedWorkers: 8,
			expectedTimeout: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewParallelExecutor(tt.maxWorkers, tt.timeout)
			assert.Equal(t, tt.expectedWorkers, executor.maxWorkers)
			assert.Equal(t, tt.expectedTimeout, executor.timeout)
		})
	}
}

func TestParallelExecutor_Execute(t *testing.T) {
	executor := NewParallelExecutor(4, 1*time.Minute)

	tool := &mockTool{
		name:     "test-tool",
		language: "Go",
		toolType: tools.FORMAT,
		executeFunc: func(ctx context.Context, files []string, options tools.ExecuteOptions) (*tools.Result, error) {
			return &tools.Result{
				Tool:           "test-tool",
				Success:        true,
				FilesProcessed: len(files),
			}, nil
		},
		validateFunc: func() error { return nil },
	}

	plan := &tools.ExecutionPlan{
		Tasks: []tools.Task{
			{
				Tool:     tool,
				Files:    []string{"file1.go", "file2.go"},
				Priority: 10,
			},
		},
		TotalFiles:        2,
		EstimatedDuration: "1s",
	}

	ctx := context.Background()
	results, err := executor.Execute(ctx, plan)

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "test-tool", results[0].Tool)
	assert.True(t, results[0].Success)
	assert.Equal(t, 2, results[0].FilesProcessed)
}

func TestParallelExecutor_ExecuteParallel(t *testing.T) {
	executor := NewParallelExecutor(4, 1*time.Minute)

	tool1 := &mockTool{
		name:     "tool1",
		language: "Go",
		toolType: tools.FORMAT,
		executeFunc: func(ctx context.Context, files []string, options tools.ExecuteOptions) (*tools.Result, error) {
			time.Sleep(10 * time.Millisecond) // Simulate work
			return &tools.Result{
				Tool:           "tool1",
				Success:        true,
				FilesProcessed: len(files),
			}, nil
		},
		validateFunc: func() error { return nil },
	}

	tool2 := &mockTool{
		name:     "tool2",
		language: "Go",
		toolType: tools.LINT,
		executeFunc: func(ctx context.Context, files []string, options tools.ExecuteOptions) (*tools.Result, error) {
			time.Sleep(10 * time.Millisecond) // Simulate work
			return &tools.Result{
				Tool:           "tool2",
				Success:        true,
				FilesProcessed: len(files),
			}, nil
		},
		validateFunc: func() error { return nil },
	}

	plan := &tools.ExecutionPlan{
		Tasks: []tools.Task{
			{
				Tool:     tool1,
				Files:    []string{"file1.go"},
				Priority: 10, // Higher priority
			},
			{
				Tool:     tool2,
				Files:    []string{"file2.go"},
				Priority: 5, // Lower priority
			},
		},
		TotalFiles:        2,
		EstimatedDuration: "1s",
	}

	ctx := context.Background()
	results, err := executor.ExecuteParallel(ctx, plan, 2)

	require.NoError(t, err)
	require.Len(t, results, 2)

	// Verify both tools ran
	toolNames := make(map[string]bool)
	for _, result := range results {
		toolNames[result.Tool] = true
	}
	assert.True(t, toolNames["tool1"])
	assert.True(t, toolNames["tool2"])
}

func TestParallelExecutor_ExecuteParallel_Timeout(t *testing.T) {
	// Very short timeout
	executor := NewParallelExecutor(1, 10*time.Millisecond)

	tool := &mockTool{
		name:     "slow-tool",
		language: "Go",
		toolType: tools.FORMAT,
		executeFunc: func(ctx context.Context, files []string, options tools.ExecuteOptions) (*tools.Result, error) {
			// Simulate slow execution
			time.Sleep(1 * time.Second)
			return &tools.Result{
				Tool:    "slow-tool",
				Success: true,
			}, nil
		},
		validateFunc: func() error { return nil },
	}

	plan := &tools.ExecutionPlan{
		Tasks: []tools.Task{
			{
				Tool:     tool,
				Files:    []string{"file1.go"},
				Priority: 10,
			},
		},
		TotalFiles:        1,
		EstimatedDuration: "1s",
	}

	ctx := context.Background()
	_, err := executor.ExecuteParallel(ctx, plan, 1)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

// Tests for ExecutionPlanner

func TestNewExecutionPlanner(t *testing.T) {
	analyzer := &mockAnalyzer{}
	planner := NewExecutionPlanner(analyzer)

	assert.NotNil(t, planner)
	assert.Equal(t, analyzer, planner.analyzer)
}

func TestExecutionPlanner_CreatePlan(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	goFile := filepath.Join(tmpDir, "main.go")
	err := os.WriteFile(goFile, []byte("package main\n\nfunc main() {}\n"), 0o644)
	require.NoError(t, err)

	tool := &mockTool{
		name:     "gofmt",
		language: "Go",
		toolType: tools.FORMAT,
		executeFunc: func(ctx context.Context, files []string, options tools.ExecuteOptions) (*tools.Result, error) {
			return &tools.Result{
				Tool:    "gofmt",
				Success: true,
			}, nil
		},
		validateFunc: func() error { return nil },
	}

	registry := &mockRegistry{
		tools: map[string]tools.QualityTool{
			"gofmt": tool,
		},
	}

	analyzer := &mockAnalyzer{
		analyzeFunc: func(projectRoot string, reg tools.ToolRegistry) (*AnalysisResult, error) {
			return &AnalysisResult{
				ProjectRoot: projectRoot,
				Languages: map[string][]string{
					"Go": {goFile},
				},
				AvailableTools: []string{"gofmt"},
				ConfigFiles:    map[string]string{},
			}, nil
		},
		selectionFunc: func(result *AnalysisResult, reg tools.ToolRegistry) map[string][]tools.QualityTool {
			return map[string][]tools.QualityTool{
				"Go": {tool},
			}
		},
	}

	planner := NewExecutionPlanner(analyzer)

	plan, err := planner.CreatePlan(tmpDir, registry, PlanOptions{})

	require.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Len(t, plan.Tasks, 1)
	assert.Equal(t, "gofmt", plan.Tasks[0].Tool.Name())
	assert.Equal(t, 10, plan.Tasks[0].Priority) // FORMAT priority
}

func TestExecutionPlanner_CreatePlan_WithFormatOnly(t *testing.T) {
	tmpDir := t.TempDir()

	formatTool := &mockTool{
		name:        "gofmt",
		language:    "Go",
		toolType:    tools.FORMAT,
		executeFunc: func(ctx context.Context, files []string, options tools.ExecuteOptions) (*tools.Result, error) {
			return &tools.Result{Tool: "gofmt", Success: true}, nil
		},
		validateFunc: func() error { return nil },
	}

	lintTool := &mockTool{
		name:        "golint",
		language:    "Go",
		toolType:    tools.LINT,
		executeFunc: func(ctx context.Context, files []string, options tools.ExecuteOptions) (*tools.Result, error) {
			return &tools.Result{Tool: "golint", Success: true}, nil
		},
		validateFunc: func() error { return nil },
	}

	registry := &mockRegistry{
		tools: map[string]tools.QualityTool{
			"gofmt":  formatTool,
			"golint": lintTool,
		},
	}

	analyzer := &mockAnalyzer{
		analyzeFunc: func(projectRoot string, reg tools.ToolRegistry) (*AnalysisResult, error) {
			return &AnalysisResult{
				ProjectRoot: projectRoot,
				Languages:   map[string][]string{"Go": {"main.go"}},
			}, nil
		},
		selectionFunc: func(result *AnalysisResult, reg tools.ToolRegistry) map[string][]tools.QualityTool {
			return map[string][]tools.QualityTool{
				"Go": {formatTool, lintTool},
			}
		},
	}

	planner := NewExecutionPlanner(analyzer)

	plan, err := planner.CreatePlan(tmpDir, registry, PlanOptions{
		FormatOnly: true,
	})

	require.NoError(t, err)
	assert.Len(t, plan.Tasks, 1)
	assert.Equal(t, "gofmt", plan.Tasks[0].Tool.Name())
}

func TestExecutionPlanner_CreatePlan_WithLintOnly(t *testing.T) {
	tmpDir := t.TempDir()

	formatTool := &mockTool{
		name:        "gofmt",
		language:    "Go",
		toolType:    tools.FORMAT,
		executeFunc: func(ctx context.Context, files []string, options tools.ExecuteOptions) (*tools.Result, error) {
			return &tools.Result{Tool: "gofmt", Success: true}, nil
		},
		validateFunc: func() error { return nil },
	}

	lintTool := &mockTool{
		name:        "golint",
		language:    "Go",
		toolType:    tools.LINT,
		executeFunc: func(ctx context.Context, files []string, options tools.ExecuteOptions) (*tools.Result, error) {
			return &tools.Result{Tool: "golint", Success: true}, nil
		},
		validateFunc: func() error { return nil },
	}

	registry := &mockRegistry{
		tools: map[string]tools.QualityTool{
			"gofmt":  formatTool,
			"golint": lintTool,
		},
	}

	analyzer := &mockAnalyzer{
		analyzeFunc: func(projectRoot string, reg tools.ToolRegistry) (*AnalysisResult, error) {
			return &AnalysisResult{
				ProjectRoot: projectRoot,
				Languages:   map[string][]string{"Go": {"main.go"}},
			}, nil
		},
		selectionFunc: func(result *AnalysisResult, reg tools.ToolRegistry) map[string][]tools.QualityTool {
			return map[string][]tools.QualityTool{
				"Go": {formatTool, lintTool},
			}
		},
	}

	planner := NewExecutionPlanner(analyzer)

	plan, err := planner.CreatePlan(tmpDir, registry, PlanOptions{
		LintOnly: true,
	})

	require.NoError(t, err)
	assert.Len(t, plan.Tasks, 1)
	assert.Equal(t, "golint", plan.Tasks[0].Tool.Name())
}

// Tests for GitUtils

func TestGitUtils_IsGitRepository(t *testing.T) {
	t.Run("Valid git repository", func(t *testing.T) {
		repoDir := setupTestGitRepo(t)
		gitUtils := &GitUtils{projectRoot: repoDir}

		assert.True(t, gitUtils.IsGitRepository())
	})

	t.Run("Not a git repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		gitUtils := &GitUtils{projectRoot: tmpDir}

		assert.False(t, gitUtils.IsGitRepository())
	})
}

func TestGitUtils_ValidateCommitish(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	createAndCommitFile(t, repoDir, "file1.txt", "content1")

	gitUtils := &GitUtils{projectRoot: repoDir}

	t.Run("Valid commit reference", func(t *testing.T) {
		err := gitUtils.ValidateCommitish("HEAD")
		assert.NoError(t, err)
	})

	t.Run("Invalid commit reference", func(t *testing.T) {
		err := gitUtils.ValidateCommitish("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid commit reference")
	})
}

func TestGitUtils_GetStagedFiles(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	gitUtils := &GitUtils{projectRoot: repoDir}

	t.Run("No staged files", func(t *testing.T) {
		files, err := gitUtils.GetStagedFiles()
		require.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("With staged files", func(t *testing.T) {
		// Create and stage a file
		testFile := filepath.Join(repoDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test content"), 0o644)
		require.NoError(t, err)

		cmd := exec.Command("git", "add", "test.txt")
		cmd.Dir = repoDir
		err = cmd.Run()
		require.NoError(t, err)

		files, err := gitUtils.GetStagedFiles()
		require.NoError(t, err)
		assert.Contains(t, files, "test.txt")
	})
}

func TestGitUtils_GetAllChangedFiles(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	createAndCommitFile(t, repoDir, "committed.txt", "initial")

	gitUtils := &GitUtils{projectRoot: repoDir}

	// Create staged file
	stagedFile := filepath.Join(repoDir, "staged.txt")
	err := os.WriteFile(stagedFile, []byte("staged"), 0o644)
	require.NoError(t, err)
	cmd := exec.Command("git", "add", "staged.txt")
	cmd.Dir = repoDir
	_ = cmd.Run()

	// Create untracked file
	untrackedFile := filepath.Join(repoDir, "untracked.txt")
	err = os.WriteFile(untrackedFile, []byte("untracked"), 0o644)
	require.NoError(t, err)

	files, err := gitUtils.GetAllChangedFiles()
	require.NoError(t, err)

	// Should contain staged and untracked files as absolute paths
	fileNames := make([]string, len(files))
	for i, f := range files {
		fileNames[i] = filepath.Base(f)
	}

	assert.Contains(t, fileNames, "staged.txt")
	assert.Contains(t, fileNames, "untracked.txt")
}

func TestGitUtils_GetChangedFiles(t *testing.T) {
	repoDir := setupTestGitRepo(t)
	createAndCommitFile(t, repoDir, "file1.txt", "content1")
	createAndCommitFile(t, repoDir, "file2.txt", "content2")

	gitUtils := &GitUtils{projectRoot: repoDir}

	t.Run("Changes since HEAD", func(t *testing.T) {
		// Modify file2.txt
		filePath := filepath.Join(repoDir, "file2.txt")
		err := os.WriteFile(filePath, []byte("modified content"), 0o644)
		require.NoError(t, err)

		files, err := gitUtils.GetChangedFiles("HEAD")
		require.NoError(t, err)
		assert.Contains(t, files, "file2.txt")
	})
}

// Tests for helper functions

func TestMatchesToolType(t *testing.T) {
	tests := []struct {
		name     string
		toolType tools.ToolType
		options  PlanOptions
		expected bool
	}{
		{
			name:     "FORMAT tool with FormatOnly",
			toolType: tools.FORMAT,
			options:  PlanOptions{FormatOnly: true},
			expected: true,
		},
		{
			name:     "LINT tool with FormatOnly",
			toolType: tools.LINT,
			options:  PlanOptions{FormatOnly: true},
			expected: false,
		},
		{
			name:     "LINT tool with LintOnly",
			toolType: tools.LINT,
			options:  PlanOptions{LintOnly: true},
			expected: true,
		},
		{
			name:     "FORMAT tool with LintOnly",
			toolType: tools.FORMAT,
			options:  PlanOptions{LintOnly: true},
			expected: false,
		},
		{
			name:     "BOTH tool with FormatOnly",
			toolType: tools.BOTH,
			options:  PlanOptions{FormatOnly: true},
			expected: true,
		},
		{
			name:     "BOTH tool with LintOnly",
			toolType: tools.BOTH,
			options:  PlanOptions{LintOnly: true},
			expected: true,
		},
		{
			name:     "Any tool with no filter",
			toolType: tools.FORMAT,
			options:  PlanOptions{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := &mockTool{toolType: tt.toolType}
			result := matchesToolType(tool, tt.options)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchesLanguageFilter(t *testing.T) {
	tests := []struct {
		name         string
		toolLanguage string
		filterLang   string
		expected     bool
	}{
		{
			name:         "Matching language",
			toolLanguage: "Go",
			filterLang:   "Go",
			expected:     true,
		},
		{
			name:         "Non-matching language",
			toolLanguage: "Go",
			filterLang:   "Python",
			expected:     false,
		},
		{
			name:         "No filter",
			toolLanguage: "Go",
			filterLang:   "",
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := &mockTool{language: tt.toolLanguage}
			options := PlanOptions{Language: tt.filterLang}
			result := matchesLanguageFilter(tool, options)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchesToolFilter(t *testing.T) {
	tests := []struct {
		name       string
		toolName   string
		toolFilter []string
		expected   bool
	}{
		{
			name:       "Tool in filter",
			toolName:   "gofmt",
			toolFilter: []string{"gofmt", "golint"},
			expected:   true,
		},
		{
			name:       "Tool not in filter",
			toolName:   "gofmt",
			toolFilter: []string{"golint", "govet"},
			expected:   false,
		},
		{
			name:       "No filter",
			toolName:   "gofmt",
			toolFilter: []string{},
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := &mockTool{name: tt.toolName}
			options := PlanOptions{ToolFilter: tt.toolFilter}
			result := matchesToolFilter(tool, options)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIntersectFiles(t *testing.T) {
	tests := []struct {
		name     string
		files1   []string
		files2   []string
		expected []string
	}{
		{
			name:     "Common files",
			files1:   []string{"a.go", "b.go", "c.go"},
			files2:   []string{"b.go", "c.go", "d.go"},
			expected: []string{"b.go", "c.go"},
		},
		{
			name:     "No common files",
			files1:   []string{"a.go", "b.go"},
			files2:   []string{"c.go", "d.go"},
			expected: nil,
		},
		{
			name:     "Empty first list",
			files1:   []string{},
			files2:   []string{"a.go", "b.go"},
			expected: nil,
		},
		{
			name:     "Empty second list",
			files1:   []string{"a.go", "b.go"},
			files2:   []string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := intersectFiles(tt.files1, tt.files2)
			assert.Equal(t, tt.expected, result)
		})
	}
}
