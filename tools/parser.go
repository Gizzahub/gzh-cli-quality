// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tools

import (
	"regexp"
	"strconv"
	"strings"
)

// Pre-compiled regex patterns for output parsing (avoids runtime compilation overhead).
var (
	// golangci-lint: file:line:col: message (rule)
	golangciLintPattern = regexp.MustCompile(`^(.+):(\d+):(\d+):\s*(.+)\s*\((.+)\)$`)

	// ruff: file:line:col: CODE message
	ruffPattern = regexp.MustCompile(`^(.+):(\d+):(\d+):\s*([A-Z]\d+)\s*(.+)$`)

	// eslint: file:line:col: severity message (rule)
	eslintPattern = regexp.MustCompile(`^(.+):(\d+):(\d+):\s*(error|warning|info)\s*(.+?)\s*(?:\((.+)\))?$`)

	// tsc: file(line,col): severity TS####: message
	tscPattern = regexp.MustCompile(`^(.+?)\((\d+),(\d+)\):\s*(error|warning)\s*TS(\d+):\s*(.+)$`)

	// pylint: file:line:col: CODE: message
	pylintPattern = regexp.MustCompile(`^(.+):(\d+):(\d+):\s*([A-Z]\d+):\s*(.+)$`)

	// Generic: file:line:col: message
	genericPattern = regexp.MustCompile(`^(.+):(\d+):(\d+):\s*(.+)$`)
)

// TextParseConfig configures how to parse text output.
type TextParseConfig struct {
	Pattern         *regexp.Regexp
	FileIndex       int
	LineIndex       int
	ColumnIndex     int
	SeverityIndex   int    // 0 means no severity captured
	RuleIndex       int    // 0 means no rule captured
	MessageIndex    int
	DefaultSeverity string
	RulePrefix      string // Optional prefix to add to rule (e.g., "TS" for TypeScript)
}

// ParseTextLines parses text output lines using the given configuration.
func ParseTextLines(output string, config TextParseConfig) []Issue {
	var issues []Issue

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := config.Pattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		issue := Issue{
			Severity: config.DefaultSeverity,
		}

		// Extract file
		if config.FileIndex > 0 && config.FileIndex < len(matches) {
			issue.File = matches[config.FileIndex]
		}

		// Extract line number
		if config.LineIndex > 0 && config.LineIndex < len(matches) {
			if lineNum, err := strconv.Atoi(matches[config.LineIndex]); err == nil {
				issue.Line = lineNum
			}
		}

		// Extract column number
		if config.ColumnIndex > 0 && config.ColumnIndex < len(matches) {
			if colNum, err := strconv.Atoi(matches[config.ColumnIndex]); err == nil {
				issue.Column = colNum
			}
		}

		// Extract severity (if captured)
		if config.SeverityIndex > 0 && config.SeverityIndex < len(matches) && matches[config.SeverityIndex] != "" {
			issue.Severity = matches[config.SeverityIndex]
		}

		// Extract rule (if captured)
		if config.RuleIndex > 0 && config.RuleIndex < len(matches) && matches[config.RuleIndex] != "" {
			issue.Rule = config.RulePrefix + matches[config.RuleIndex]
		}

		// Extract message
		if config.MessageIndex > 0 && config.MessageIndex < len(matches) {
			issue.Message = matches[config.MessageIndex]
		}

		issues = append(issues, issue)
	}

	return issues
}

// Common parse configurations for each tool.
var (
	GolangciLintParseConfig = TextParseConfig{
		Pattern:         golangciLintPattern,
		FileIndex:       1,
		LineIndex:       2,
		ColumnIndex:     3,
		SeverityIndex:   0,
		RuleIndex:       5,
		MessageIndex:    4,
		DefaultSeverity: "error",
	}

	RuffParseConfig = TextParseConfig{
		Pattern:         ruffPattern,
		FileIndex:       1,
		LineIndex:       2,
		ColumnIndex:     3,
		SeverityIndex:   0,
		RuleIndex:       4,
		MessageIndex:    5,
		DefaultSeverity: "error",
	}

	ESLintParseConfig = TextParseConfig{
		Pattern:         eslintPattern,
		FileIndex:       1,
		LineIndex:       2,
		ColumnIndex:     3,
		SeverityIndex:   4,
		RuleIndex:       6,
		MessageIndex:    5,
		DefaultSeverity: "error",
	}

	TSCParseConfig = TextParseConfig{
		Pattern:         tscPattern,
		FileIndex:       1,
		LineIndex:       2,
		ColumnIndex:     3,
		SeverityIndex:   4,
		RuleIndex:       5,
		MessageIndex:    6,
		DefaultSeverity: "error",
		RulePrefix:      "TS",
	}

	PylintParseConfig = TextParseConfig{
		Pattern:         pylintPattern,
		FileIndex:       1,
		LineIndex:       2,
		ColumnIndex:     3,
		SeverityIndex:   0,
		RuleIndex:       4,
		MessageIndex:    5,
		DefaultSeverity: "error",
	}
)
