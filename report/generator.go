// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Gizzahub/gzh-cli-quality/tools"
)

// Report represents a quality report.
type Report struct {
	Timestamp    time.Time          `json:"timestamp"`
	ProjectRoot  string             `json:"project_root"`
	TotalFiles   int                `json:"total_files"`
	Duration     time.Duration      `json:"duration"`
	Summary      Summary            `json:"summary"`
	ToolResults  []ToolResult       `json:"tool_results"`
	IssuesByFile map[string][]Issue `json:"issues_by_file"`
}

// Summary contains report summary information.
type Summary struct {
	TotalTools      int `json:"total_tools"`
	SuccessfulTools int `json:"successful_tools"`
	FailedTools     int `json:"failed_tools"`
	TotalIssues     int `json:"total_issues"`
	ErrorIssues     int `json:"error_issues"`
	WarningIssues   int `json:"warning_issues"`
	InfoIssues      int `json:"info_issues"`
	FilesWithIssues int `json:"files_with_issues"`
}

// ToolResult represents the result of a single tool execution.
type ToolResult struct {
	Tool           string        `json:"tool"`
	Language       string        `json:"language"`
	Success        bool          `json:"success"`
	Duration       time.Duration `json:"duration"`
	FilesProcessed int           `json:"files_processed"`
	IssuesFound    int           `json:"issues_found"`
	Error          string        `json:"error,omitempty"`
}

// Issue represents a quality issue.
type Issue struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	Column     int    `json:"column"`
	Severity   string `json:"severity"`
	Rule       string `json:"rule"`
	Message    string `json:"message"`
	Tool       string `json:"tool"`
	Suggestion string `json:"suggestion,omitempty"`
}

// ReportGenerator generates quality reports.
type ReportGenerator struct {
	projectRoot string
}

// NewReportGenerator creates a new report generator.
func NewReportGenerator(projectRoot string) *ReportGenerator {
	return &ReportGenerator{
		projectRoot: projectRoot,
	}
}

// GenerateReport creates a report from quality results.
func (g *ReportGenerator) GenerateReport(results []*tools.Result, duration time.Duration, totalFiles int) *Report {
	report := &Report{
		Timestamp:    time.Now(),
		ProjectRoot:  g.projectRoot,
		TotalFiles:   totalFiles,
		Duration:     duration,
		ToolResults:  make([]ToolResult, 0, len(results)),
		IssuesByFile: make(map[string][]Issue),
	}

	// Process results
	for _, result := range results {
		toolResult := ToolResult{
			Tool:           result.Tool,
			Language:       result.Language,
			Success:        result.Success,
			Duration:       result.Duration,
			FilesProcessed: result.FilesProcessed,
			IssuesFound:    len(result.Issues),
		}

		if result.Error != "" {
			toolResult.Error = result.Error
		}

		report.ToolResults = append(report.ToolResults, toolResult)

		// Process issues
		for _, issue := range result.Issues {
			reportIssue := Issue{
				File:       issue.File,
				Line:       issue.Line,
				Column:     issue.Column,
				Severity:   issue.Severity,
				Rule:       issue.Rule,
				Message:    issue.Message,
				Tool:       result.Tool,
				Suggestion: issue.Suggestion,
			}

			report.IssuesByFile[issue.File] = append(report.IssuesByFile[issue.File], reportIssue)
		}
	}

	// Calculate summary
	report.Summary = g.calculateSummary(report)

	return report
}

// calculateSummary calculates report summary statistics.
func (g *ReportGenerator) calculateSummary(report *Report) Summary {
	summary := Summary{
		TotalTools: len(report.ToolResults),
	}

	for _, result := range report.ToolResults {
		if result.Success {
			summary.SuccessfulTools++
		} else {
			summary.FailedTools++
		}
		summary.TotalIssues += result.IssuesFound
	}

	// Count issues by severity
	for _, issues := range report.IssuesByFile {
		for _, issue := range issues {
			switch strings.ToLower(issue.Severity) {
			case "error":
				summary.ErrorIssues++
			case "warning":
				summary.WarningIssues++
			default:
				summary.InfoIssues++
			}
		}
	}

	summary.FilesWithIssues = len(report.IssuesByFile)

	return summary
}

// SaveJSON saves the report as JSON.
func (g *ReportGenerator) SaveJSON(report *Report, outputPath string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	return nil
}

// SaveHTML saves the report as HTML.
func (g *ReportGenerator) SaveHTML(report *Report, outputPath string) error {
	html := g.generateHTML(report)

	if err := os.WriteFile(outputPath, []byte(html), 0o644); err != nil {
		return fmt.Errorf("failed to write HTML report: %w", err)
	}

	return nil
}

// generateHTML creates an HTML report.
func (g *ReportGenerator) generateHTML(report *Report) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html lang="ko">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Quality Report</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; border-radius: 8px; padding: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { border-bottom: 2px solid #e0e0e0; padding-bottom: 20px; margin-bottom: 30px; }
        .header h1 { margin: 0; color: #333; font-size: 2em; }
        .meta { color: #666; margin-top: 10px; }
        .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .stat-card { background: #f8f9fa; padding: 20px; border-radius: 6px; text-align: center; }
        .stat-value { font-size: 2em; font-weight: bold; color: #007bff; }
        .stat-label { color: #666; margin-top: 5px; }
        .error { color: #dc3545; }
        .warning { color: #ffc107; }
        .success { color: #28a745; }
        .section { margin-bottom: 30px; }
        .section h2 { color: #333; border-bottom: 1px solid #e0e0e0; padding-bottom: 10px; }
        .tool-results { display: grid; gap: 15px; }
        .tool-result { background: #f8f9fa; padding: 15px; border-radius: 6px; border-left: 4px solid #007bff; }
        .tool-result.failed { border-left-color: #dc3545; }
        .issues-table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        .issues-table th, .issues-table td { padding: 12px; text-align: left; border-bottom: 1px solid #e0e0e0; }
        .issues-table th { background: #f8f9fa; font-weight: 600; }
        .severity-error { color: #dc3545; font-weight: bold; }
        .severity-warning { color: #ffc107; font-weight: bold; }
        .severity-info { color: #17a2b8; }
    </style>
</head>
<body>`)

	// Header
	sb.WriteString(`<div class="container">
        <div class="header">
            <h1>🎯 Code Quality Report</h1>
            <div class="meta">
                <div><strong>프로젝트:</strong> ` + report.ProjectRoot + `</div>
                <div><strong>생성 시간:</strong> ` + report.Timestamp.Format("2006-01-02 15:04:05") + `</div>
                <div><strong>분석 시간:</strong> ` + report.Duration.String() + `</div>
            </div>
        </div>`)

	// Summary
	sb.WriteString(`<div class="summary">
            <div class="stat-card">
                <div class="stat-value">` + fmt.Sprintf("%d", report.TotalFiles) + `</div>
                <div class="stat-label">총 파일 수</div>
            </div>
            <div class="stat-card">
                <div class="stat-value success">` + fmt.Sprintf("%d", report.Summary.SuccessfulTools) + `</div>
                <div class="stat-label">성공한 도구</div>
            </div>
            <div class="stat-card">
                <div class="stat-value error">` + fmt.Sprintf("%d", report.Summary.FailedTools) + `</div>
                <div class="stat-label">실패한 도구</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">` + fmt.Sprintf("%d", report.Summary.TotalIssues) + `</div>
                <div class="stat-label">총 이슈</div>
            </div>
        </div>`)

	// Tool Results
	sb.WriteString(`<div class="section">
            <h2>🛠️ 도구 실행 결과</h2>
            <div class="tool-results">`)

	for _, result := range report.ToolResults {
		status := "success"
		if !result.Success {
			status = "failed"
		}

		sb.WriteString(`<div class="tool-result ` + status + `">
                <h3>` + result.Tool + ` (` + result.Language + `)</h3>
                <p><strong>상태:</strong> `)

		if result.Success {
			sb.WriteString(`<span class="success">✅ 성공</span>`)
		} else {
			sb.WriteString(`<span class="error">❌ 실패</span>`)
		}

		sb.WriteString(`</p>
                <p><strong>처리 파일:</strong> ` + fmt.Sprintf("%d", result.FilesProcessed) + `개</p>
                <p><strong>소요 시간:</strong> ` + result.Duration.String() + `</p>
                <p><strong>발견 이슈:</strong> ` + fmt.Sprintf("%d", result.IssuesFound) + `개</p>`)

		if result.Error != "" {
			sb.WriteString(`<p><strong>오류:</strong> <code>` + result.Error + `</code></p>`)
		}

		sb.WriteString(`</div>`)
	}

	sb.WriteString(`</div></div>`)

	// Issues by File
	if len(report.IssuesByFile) > 0 {
		sb.WriteString(`<div class="section">
                <h2>📋 파일별 이슈</h2>`)

		// Sort files by issue count
		type fileIssues struct {
			file   string
			issues []Issue
		}

		var sortedFiles []fileIssues
		for file, issues := range report.IssuesByFile {
			sortedFiles = append(sortedFiles, fileIssues{file, issues})
		}

		sort.Slice(sortedFiles, func(i, j int) bool {
			return len(sortedFiles[i].issues) > len(sortedFiles[j].issues)
		})

		for _, fileData := range sortedFiles {
			sb.WriteString(`<h3>📄 ` + fileData.file + ` (` + fmt.Sprintf("%d", len(fileData.issues)) + `개 이슈)</h3>
                    <table class="issues-table">
                        <thead>
                            <tr>
                                <th>라인</th>
                                <th>열</th>
                                <th>심각도</th>
                                <th>규칙</th>
                                <th>메시지</th>
                                <th>도구</th>
                            </tr>
                        </thead>
                        <tbody>`)

			for _, issue := range fileData.issues {
				sb.WriteString(`<tr>
                            <td>` + fmt.Sprintf("%d", issue.Line) + `</td>
                            <td>` + fmt.Sprintf("%d", issue.Column) + `</td>
                            <td><span class="severity-` + strings.ToLower(issue.Severity) + `">` + issue.Severity + `</span></td>
                            <td><code>` + issue.Rule + `</code></td>
                            <td>` + issue.Message + `</td>
                            <td>` + issue.Tool + `</td>
                        </tr>`)
			}

			sb.WriteString(`</tbody></table>`)
		}

		sb.WriteString(`</div>`)
	}

	sb.WriteString(`</div></body></html>`)

	return sb.String()
}

// SaveMarkdown saves the report as Markdown.
func (g *ReportGenerator) SaveMarkdown(report *Report, outputPath string) error {
	md := g.generateMarkdown(report)

	if err := os.WriteFile(outputPath, []byte(md), 0o644); err != nil {
		return fmt.Errorf("failed to write Markdown report: %w", err)
	}

	return nil
}

// generateMarkdown creates a Markdown report.
func (g *ReportGenerator) generateMarkdown(report *Report) string {
	var sb strings.Builder

	sb.WriteString("# 🎯 Code Quality Report\n\n")
	sb.WriteString("## 📊 Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **프로젝트**: %s\n", report.ProjectRoot))
	sb.WriteString(fmt.Sprintf("- **생성 시간**: %s\n", report.Timestamp.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("- **분석 시간**: %s\n", report.Duration.String()))
	sb.WriteString(fmt.Sprintf("- **총 파일 수**: %d\n", report.TotalFiles))
	sb.WriteString(fmt.Sprintf("- **성공한 도구**: %d/%d\n", report.Summary.SuccessfulTools, report.Summary.TotalTools))
	sb.WriteString(fmt.Sprintf("- **총 이슈**: %d (오류: %d, 경고: %d, 정보: %d)\n\n",
		report.Summary.TotalIssues, report.Summary.ErrorIssues, report.Summary.WarningIssues, report.Summary.InfoIssues))

	sb.WriteString("## 🛠️ Tool Results\n\n")
	sb.WriteString("| 도구 | 언어 | 상태 | 파일 수 | 이슈 수 | 소요 시간 |\n")
	sb.WriteString("|------|------|------|---------|---------|----------|\n")

	for _, result := range report.ToolResults {
		status := "✅"
		if !result.Success {
			status = "❌"
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %d | %s |\n",
			result.Tool, result.Language, status, result.FilesProcessed, result.IssuesFound, result.Duration.String()))
	}

	if len(report.IssuesByFile) > 0 {
		sb.WriteString("\n## 📋 Issues by File\n\n")

		for file, issues := range report.IssuesByFile {
			sb.WriteString(fmt.Sprintf("### 📄 %s (%d issues)\n\n", file, len(issues)))
			sb.WriteString("| Line | Column | Severity | Rule | Message | Tool |\n")
			sb.WriteString("|------|--------|----------|------|---------|------|\n")

			for _, issue := range issues {
				sb.WriteString(fmt.Sprintf("| %d | %d | %s | `%s` | %s | %s |\n",
					issue.Line, issue.Column, issue.Severity, issue.Rule, issue.Message, issue.Tool))
			}

			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// GetReportPath generates a report file path.
func (g *ReportGenerator) GetReportPath(format string) string {
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("quality-report-%s.%s", timestamp, format)
	return filepath.Join(g.projectRoot, "tmp", filename)
}
