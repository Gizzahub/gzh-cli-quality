// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package report

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Gizzahub/gzh-cli-quality/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReportGenerator(t *testing.T) {
	projectRoot := "/test/project"
	generator := NewReportGenerator(projectRoot)

	assert.NotNil(t, generator)
	assert.Equal(t, projectRoot, generator.projectRoot)
}

func TestGenerateReport_EmptyResults(t *testing.T) {
	generator := NewReportGenerator("/test/project")

	report := generator.GenerateReport([]*tools.Result{}, 5*time.Second, 10)

	assert.NotNil(t, report)
	assert.Equal(t, "/test/project", report.ProjectRoot)
	assert.Equal(t, 10, report.TotalFiles)
	assert.Equal(t, 5*time.Second, report.Duration)
	assert.Empty(t, report.ToolResults)
	assert.Empty(t, report.IssuesByFile)
	assert.Equal(t, 0, report.Summary.TotalTools)
	assert.Equal(t, 0, report.Summary.TotalIssues)
}

func TestGenerateReport_SingleToolSuccess(t *testing.T) {
	generator := NewReportGenerator("/test/project")

	results := []*tools.Result{
		{
			Tool:           "gofmt",
			Language:       "Go",
			Success:        true,
			FilesProcessed: 5,
			Duration:       "2s",
			Issues:         []tools.Issue{},
		},
	}

	report := generator.GenerateReport(results, 3*time.Second, 5)

	require.NotNil(t, report)
	assert.Equal(t, 1, len(report.ToolResults))
	assert.Equal(t, "gofmt", report.ToolResults[0].Tool)
	assert.Equal(t, "Go", report.ToolResults[0].Language)
	assert.True(t, report.ToolResults[0].Success)
	assert.Equal(t, 5, report.ToolResults[0].FilesProcessed)
	assert.Equal(t, 2*time.Second, report.ToolResults[0].Duration)
	assert.Equal(t, 0, report.ToolResults[0].IssuesFound)

	// Check summary
	assert.Equal(t, 1, report.Summary.TotalTools)
	assert.Equal(t, 1, report.Summary.SuccessfulTools)
	assert.Equal(t, 0, report.Summary.FailedTools)
	assert.Equal(t, 0, report.Summary.TotalIssues)
}

func TestGenerateReport_WithIssues(t *testing.T) {
	generator := NewReportGenerator("/test/project")

	results := []*tools.Result{
		{
			Tool:           "golint",
			Language:       "Go",
			Success:        true,
			FilesProcessed: 3,
			Duration:       "1.5s",
			Issues: []tools.Issue{
				{
					File:       "main.go",
					Line:       10,
					Column:     5,
					Severity:   "error",
					Rule:       "unused-var",
					Message:    "Variable 'x' is unused",
					Suggestion: "Remove unused variable",
				},
				{
					File:       "main.go",
					Line:       15,
					Column:     8,
					Severity:   "warning",
					Rule:       "missing-doc",
					Message:    "Function needs documentation",
					Suggestion: "",
				},
				{
					File:       "utils.go",
					Line:       5,
					Column:     1,
					Severity:   "info",
					Rule:       "style",
					Message:    "Consider using camelCase",
					Suggestion: "",
				},
			},
		},
	}

	report := generator.GenerateReport(results, 2*time.Second, 3)

	require.NotNil(t, report)
	assert.Equal(t, 1, len(report.ToolResults))
	assert.Equal(t, 3, report.ToolResults[0].IssuesFound)

	// Check issues by file
	assert.Equal(t, 2, len(report.IssuesByFile))
	assert.Contains(t, report.IssuesByFile, "main.go")
	assert.Contains(t, report.IssuesByFile, "utils.go")
	assert.Equal(t, 2, len(report.IssuesByFile["main.go"]))
	assert.Equal(t, 1, len(report.IssuesByFile["utils.go"]))

	// Check specific issue
	mainIssue := report.IssuesByFile["main.go"][0]
	assert.Equal(t, "main.go", mainIssue.File)
	assert.Equal(t, 10, mainIssue.Line)
	assert.Equal(t, 5, mainIssue.Column)
	assert.Equal(t, "error", mainIssue.Severity)
	assert.Equal(t, "unused-var", mainIssue.Rule)
	assert.Equal(t, "Variable 'x' is unused", mainIssue.Message)
	assert.Equal(t, "golint", mainIssue.Tool)
	assert.Equal(t, "Remove unused variable", mainIssue.Suggestion)

	// Check summary
	assert.Equal(t, 1, report.Summary.TotalTools)
	assert.Equal(t, 1, report.Summary.SuccessfulTools)
	assert.Equal(t, 3, report.Summary.TotalIssues)
	assert.Equal(t, 1, report.Summary.ErrorIssues)
	assert.Equal(t, 1, report.Summary.WarningIssues)
	assert.Equal(t, 1, report.Summary.InfoIssues)
	assert.Equal(t, 2, report.Summary.FilesWithIssues)
}

func TestGenerateReport_ToolWithError(t *testing.T) {
	generator := NewReportGenerator("/test/project")

	results := []*tools.Result{
		{
			Tool:           "eslint",
			Language:       "JavaScript",
			Success:        false,
			FilesProcessed: 0,
			Duration:       "0s",
			Error:          errors.New("tool not found"),
			Issues:         []tools.Issue{},
		},
	}

	report := generator.GenerateReport(results, 1*time.Second, 5)

	require.NotNil(t, report)
	assert.Equal(t, 1, len(report.ToolResults))
	assert.False(t, report.ToolResults[0].Success)
	assert.Equal(t, "tool not found", report.ToolResults[0].Error)

	// Check summary
	assert.Equal(t, 1, report.Summary.TotalTools)
	assert.Equal(t, 0, report.Summary.SuccessfulTools)
	assert.Equal(t, 1, report.Summary.FailedTools)
}

func TestGenerateReport_MultipleTools(t *testing.T) {
	generator := NewReportGenerator("/test/project")

	results := []*tools.Result{
		{
			Tool:           "gofmt",
			Language:       "Go",
			Success:        true,
			FilesProcessed: 5,
			Duration:       "1s",
			Issues:         []tools.Issue{},
		},
		{
			Tool:           "golint",
			Language:       "Go",
			Success:        true,
			FilesProcessed: 5,
			Duration:       "2s",
			Issues: []tools.Issue{
				{
					File:     "main.go",
					Line:     10,
					Severity: "warning",
					Rule:     "test-rule",
					Message:  "Test message",
				},
			},
		},
		{
			Tool:           "govet",
			Language:       "Go",
			Success:        false,
			FilesProcessed: 0,
			Duration:       "0s",
			Error:          errors.New("command failed"),
			Issues:         []tools.Issue{},
		},
	}

	report := generator.GenerateReport(results, 5*time.Second, 5)

	require.NotNil(t, report)
	assert.Equal(t, 3, len(report.ToolResults))

	// Check summary
	assert.Equal(t, 3, report.Summary.TotalTools)
	assert.Equal(t, 2, report.Summary.SuccessfulTools)
	assert.Equal(t, 1, report.Summary.FailedTools)
	assert.Equal(t, 1, report.Summary.TotalIssues)
	assert.Equal(t, 1, report.Summary.WarningIssues)
}

func TestGenerateReport_InvalidDuration(t *testing.T) {
	generator := NewReportGenerator("/test/project")

	results := []*tools.Result{
		{
			Tool:           "test-tool",
			Language:       "Go",
			Success:        true,
			FilesProcessed: 1,
			Duration:       "invalid-duration",
			Issues:         []tools.Issue{},
		},
	}

	report := generator.GenerateReport(results, 1*time.Second, 1)

	require.NotNil(t, report)
	assert.Equal(t, time.Duration(0), report.ToolResults[0].Duration)
}

func TestSaveJSON(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewReportGenerator(tmpDir)

	report := &Report{
		Timestamp:   time.Date(2025, 1, 27, 12, 0, 0, 0, time.UTC),
		ProjectRoot: tmpDir,
		TotalFiles:  5,
		Duration:    3 * time.Second,
		Summary: Summary{
			TotalTools:      2,
			SuccessfulTools: 2,
			TotalIssues:     1,
		},
		ToolResults: []ToolResult{
			{
				Tool:           "gofmt",
				Language:       "Go",
				Success:        true,
				FilesProcessed: 5,
			},
		},
		IssuesByFile: make(map[string][]Issue),
	}

	outputPath := filepath.Join(tmpDir, "report.json")
	err := generator.SaveJSON(report, outputPath)

	require.NoError(t, err)
	assert.FileExists(t, outputPath)

	// Verify JSON content
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	var loadedReport Report
	err = json.Unmarshal(data, &loadedReport)
	require.NoError(t, err)

	assert.Equal(t, tmpDir, loadedReport.ProjectRoot)
	assert.Equal(t, 5, loadedReport.TotalFiles)
	assert.Equal(t, 2, loadedReport.Summary.TotalTools)
}

func TestSaveHTML(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewReportGenerator(tmpDir)

	report := &Report{
		Timestamp:   time.Date(2025, 1, 27, 12, 0, 0, 0, time.UTC),
		ProjectRoot: tmpDir,
		TotalFiles:  5,
		Duration:    3 * time.Second,
		Summary: Summary{
			TotalTools:      1,
			SuccessfulTools: 1,
			TotalIssues:     0,
		},
		ToolResults: []ToolResult{
			{
				Tool:           "gofmt",
				Language:       "Go",
				Success:        true,
				FilesProcessed: 5,
				Duration:       2 * time.Second,
			},
		},
		IssuesByFile: make(map[string][]Issue),
	}

	outputPath := filepath.Join(tmpDir, "report.html")
	err := generator.SaveHTML(report, outputPath)

	require.NoError(t, err)
	assert.FileExists(t, outputPath)

	// Verify HTML content
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	html := string(data)
	assert.Contains(t, html, "<!DOCTYPE html>")
	assert.Contains(t, html, "Code Quality Report")
	assert.Contains(t, html, "gofmt")
	assert.Contains(t, html, "✅ 성공")
}

func TestSaveHTML_WithIssues(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewReportGenerator(tmpDir)

	report := &Report{
		Timestamp:   time.Now(),
		ProjectRoot: tmpDir,
		TotalFiles:  2,
		Duration:    2 * time.Second,
		Summary: Summary{
			TotalTools:      1,
			SuccessfulTools: 1,
			TotalIssues:     2,
			ErrorIssues:     1,
			WarningIssues:   1,
		},
		ToolResults: []ToolResult{
			{
				Tool:           "golint",
				Language:       "Go",
				Success:        true,
				FilesProcessed: 2,
				IssuesFound:    2,
			},
		},
		IssuesByFile: map[string][]Issue{
			"main.go": {
				{
					File:     "main.go",
					Line:     10,
					Column:   5,
					Severity: "error",
					Rule:     "test-rule",
					Message:  "Test error",
					Tool:     "golint",
				},
				{
					File:     "main.go",
					Line:     15,
					Column:   8,
					Severity: "warning",
					Rule:     "warn-rule",
					Message:  "Test warning",
					Tool:     "golint",
				},
			},
		},
	}

	outputPath := filepath.Join(tmpDir, "report-with-issues.html")
	err := generator.SaveHTML(report, outputPath)

	require.NoError(t, err)

	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	html := string(data)
	assert.Contains(t, html, "main.go")
	assert.Contains(t, html, "Test error")
	assert.Contains(t, html, "Test warning")
	assert.Contains(t, html, "파일별 이슈")
}

func TestSaveHTML_FailedTool(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewReportGenerator(tmpDir)

	report := &Report{
		Timestamp:   time.Now(),
		ProjectRoot: tmpDir,
		TotalFiles:  0,
		Duration:    1 * time.Second,
		Summary: Summary{
			TotalTools:  1,
			FailedTools: 1,
		},
		ToolResults: []ToolResult{
			{
				Tool:     "broken-tool",
				Language: "Go",
				Success:  false,
				Error:    "tool execution failed",
			},
		},
		IssuesByFile: make(map[string][]Issue),
	}

	outputPath := filepath.Join(tmpDir, "report-failed.html")
	err := generator.SaveHTML(report, outputPath)

	require.NoError(t, err)

	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	html := string(data)
	assert.Contains(t, html, "broken-tool")
	assert.Contains(t, html, "❌ 실패")
	assert.Contains(t, html, "tool execution failed")
}

func TestSaveMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewReportGenerator(tmpDir)

	report := &Report{
		Timestamp:   time.Date(2025, 1, 27, 12, 0, 0, 0, time.UTC),
		ProjectRoot: tmpDir,
		TotalFiles:  5,
		Duration:    3 * time.Second,
		Summary: Summary{
			TotalTools:      1,
			SuccessfulTools: 1,
			TotalIssues:     0,
		},
		ToolResults: []ToolResult{
			{
				Tool:           "gofmt",
				Language:       "Go",
				Success:        true,
				FilesProcessed: 5,
				Duration:       2 * time.Second,
			},
		},
		IssuesByFile: make(map[string][]Issue),
	}

	outputPath := filepath.Join(tmpDir, "report.md")
	err := generator.SaveMarkdown(report, outputPath)

	require.NoError(t, err)
	assert.FileExists(t, outputPath)

	// Verify Markdown content
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	md := string(data)
	assert.Contains(t, md, "# 🎯 Code Quality Report")
	assert.Contains(t, md, "## 📊 Summary")
	assert.Contains(t, md, "## 🛠️ Tool Results")
	assert.Contains(t, md, "gofmt")
	assert.Contains(t, md, "✅")
}

func TestSaveMarkdown_WithIssues(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewReportGenerator(tmpDir)

	report := &Report{
		Timestamp:   time.Now(),
		ProjectRoot: tmpDir,
		TotalFiles:  1,
		Duration:    1 * time.Second,
		Summary: Summary{
			TotalTools:      1,
			SuccessfulTools: 1,
			TotalIssues:     1,
			ErrorIssues:     1,
		},
		ToolResults: []ToolResult{
			{
				Tool:           "golint",
				Language:       "Go",
				Success:        true,
				FilesProcessed: 1,
				IssuesFound:    1,
			},
		},
		IssuesByFile: map[string][]Issue{
			"main.go": {
				{
					File:     "main.go",
					Line:     10,
					Column:   5,
					Severity: "error",
					Rule:     "test-rule",
					Message:  "Test issue",
					Tool:     "golint",
				},
			},
		},
	}

	outputPath := filepath.Join(tmpDir, "report-with-issues.md")
	err := generator.SaveMarkdown(report, outputPath)

	require.NoError(t, err)

	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	md := string(data)
	assert.Contains(t, md, "## 📋 Issues by File")
	assert.Contains(t, md, "### 📄 main.go")
	assert.Contains(t, md, "Test issue")
	assert.Contains(t, md, "test-rule")
}

func TestGetReportPath(t *testing.T) {
	tmpDir := t.TempDir()
	generator := NewReportGenerator(tmpDir)

	tests := []struct {
		format   string
		expected string
	}{
		{"json", ".json"},
		{"html", ".html"},
		{"md", ".md"},
		{"txt", ".txt"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			path := generator.GetReportPath(tt.format)

			assert.Contains(t, path, "quality-report-")
			assert.Contains(t, path, tt.expected)
			assert.Contains(t, path, filepath.Join(tmpDir, "tmp"))
			assert.True(t, strings.HasSuffix(path, tt.expected))
		})
	}
}

func TestGenerateHTML_IssuesSortedByCount(t *testing.T) {
	generator := NewReportGenerator("/test")

	report := &Report{
		Timestamp:   time.Now(),
		ProjectRoot: "/test",
		TotalFiles:  3,
		Duration:    1 * time.Second,
		Summary:     Summary{},
		ToolResults: []ToolResult{},
		IssuesByFile: map[string][]Issue{
			"file1.go": {
				{File: "file1.go", Line: 1, Severity: "error", Rule: "rule1", Message: "msg1"},
			},
			"file2.go": {
				{File: "file2.go", Line: 1, Severity: "error", Rule: "rule2", Message: "msg2"},
				{File: "file2.go", Line: 2, Severity: "warning", Rule: "rule3", Message: "msg3"},
				{File: "file2.go", Line: 3, Severity: "info", Rule: "rule4", Message: "msg4"},
			},
			"file3.go": {
				{File: "file3.go", Line: 1, Severity: "warning", Rule: "rule5", Message: "msg5"},
				{File: "file3.go", Line: 2, Severity: "warning", Rule: "rule6", Message: "msg6"},
			},
		},
	}

	html := generator.generateHTML(report)

	// file2.go should appear first (3 issues)
	file2Index := strings.Index(html, "file2.go")
	file3Index := strings.Index(html, "file3.go")
	file1Index := strings.Index(html, "file1.go")

	assert.True(t, file2Index < file3Index, "file2.go should appear before file3.go")
	assert.True(t, file3Index < file1Index, "file3.go should appear before file1.go")
}

func TestCalculateSummary_SeverityCaseMixing(t *testing.T) {
	generator := NewReportGenerator("/test")

	report := &Report{
		ToolResults: []ToolResult{
			{Success: true, IssuesFound: 3},
		},
		IssuesByFile: map[string][]Issue{
			"test.go": {
				{Severity: "Error"},   // Capital E
				{Severity: "WARNING"}, // All caps
				{Severity: "info"},    // lowercase
			},
		},
	}

	summary := generator.calculateSummary(report)

	assert.Equal(t, 1, summary.ErrorIssues)
	assert.Equal(t, 1, summary.WarningIssues)
	assert.Equal(t, 1, summary.InfoIssues)
}
