// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tools

import (
	"encoding/xml"
	"os/exec"
	"strings"
)

// GoogleJavaFormatTool implements Java formatting using google-java-format.
type GoogleJavaFormatTool struct {
	*BaseTool
}

// NewGoogleJavaFormatTool creates a new google-java-format tool.
func NewGoogleJavaFormatTool() *GoogleJavaFormatTool {
	tool := &GoogleJavaFormatTool{
		BaseTool: NewBaseTool("google-java-format", "Java", "google-java-format", FORMAT),
	}

	tool.SetInstallCommand([]string{"brew", "install", "google-java-format"})

	return tool
}

// BuildCommand builds the google-java-format command.
func (t *GoogleJavaFormatTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{"-i"} // In-place formatting

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Filter only Java files
	javaFiles := FilterFilesByExtensions(files, []string{".java"})
	if len(javaFiles) > 0 {
		args = append(args, javaFiles...)
	}

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// CheckstyleTool implements Java linting using checkstyle.
type CheckstyleTool struct {
	*BaseTool
}

// NewCheckstyleTool creates a new checkstyle tool.
func NewCheckstyleTool() *CheckstyleTool {
	tool := &CheckstyleTool{
		BaseTool: NewBaseTool("checkstyle", "Java", "checkstyle", LINT),
	}

	tool.SetInstallCommand([]string{"brew", "install", "checkstyle"})
	tool.SetConfigPatterns([]string{"checkstyle.xml", ".checkstyle.xml", "config/checkstyle/checkstyle.xml"})

	return tool
}

// BuildCommand builds the checkstyle command.
func (t *CheckstyleTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{"-f", "xml"} // XML output for parsing

	// Add config file
	if options.ConfigFile != "" {
		args = append(args, "-c", options.ConfigFile)
	} else {
		args = append(args, "-c", "/google_checks.xml") // Default to Google checks
	}

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// Filter only Java files
	javaFiles := FilterFilesByExtensions(files, []string{".java"})
	if len(javaFiles) > 0 {
		args = append(args, javaFiles...)
	} else {
		args = append(args, ".")
	}

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// ParseOutput parses checkstyle XML output.
func (t *CheckstyleTool) ParseOutput(output string) []Issue {
	if strings.TrimSpace(output) == "" {
		return []Issue{}
	}

	var checkstyleResult struct {
		XMLName xml.Name `xml:"checkstyle"`
		Files   []struct {
			Name   string `xml:"name,attr"`
			Errors []struct {
				Line     int    `xml:"line,attr"`
				Column   int    `xml:"column,attr"`
				Severity string `xml:"severity,attr"`
				Message  string `xml:"message,attr"`
				Source   string `xml:"source,attr"`
			} `xml:"error"`
		} `xml:"file"`
	}

	if err := xml.Unmarshal([]byte(output), &checkstyleResult); err != nil {
		return []Issue{}
	}

	var issues []Issue
	for _, file := range checkstyleResult.Files {
		for _, e := range file.Errors {
			// Extract rule name from source (e.g., "com.puppycrawl.tools.checkstyle.checks.whitespace.WhitespaceAfterCheck")
			rule := e.Source
			if idx := strings.LastIndex(e.Source, "."); idx != -1 {
				rule = e.Source[idx+1:]
			}

			issues = append(issues, Issue{
				File:     file.Name,
				Line:     e.Line,
				Column:   e.Column,
				Severity: e.Severity,
				Rule:     rule,
				Message:  e.Message,
			})
		}
	}

	return issues
}

// SpotbugsTool implements Java bug detection using spotbugs.
type SpotbugsTool struct {
	*BaseTool
}

// NewSpotbugsTool creates a new spotbugs tool.
func NewSpotbugsTool() *SpotbugsTool {
	tool := &SpotbugsTool{
		BaseTool: NewBaseTool("spotbugs", "Java", "spotbugs", LINT),
	}

	tool.SetInstallCommand([]string{"brew", "install", "spotbugs"})
	tool.SetConfigPatterns([]string{"spotbugs.xml", ".spotbugs.xml", "spotbugs-exclude.xml"})

	return tool
}

// BuildCommand builds the spotbugs command.
func (t *SpotbugsTool) BuildCommand(files []string, options ExecuteOptions) *exec.Cmd {
	args := []string{"-textui", "-xml:withMessages"}

	// Add exclude filter if config specified
	if options.ConfigFile != "" {
		args = append(args, "-exclude", options.ConfigFile)
	}

	// Add extra flags if provided
	args = append(args, options.ExtraArgs...)

	// SpotBugs works on compiled classes, not source files
	// Users should specify the build output directory
	if len(files) > 0 {
		args = append(args, files...)
	} else {
		args = append(args, "build/classes", "target/classes") // Common build directories
	}

	cmd := exec.Command(t.executable, args...)

	if options.ProjectRoot != "" {
		cmd.Dir = options.ProjectRoot
	}

	return cmd
}

// ParseOutput parses spotbugs XML output.
func (t *SpotbugsTool) ParseOutput(output string) []Issue {
	if strings.TrimSpace(output) == "" {
		return []Issue{}
	}

	var spotbugsResult struct {
		XMLName   xml.Name `xml:"BugCollection"`
		BugInstances []struct {
			Type     string `xml:"type,attr"`
			Priority int    `xml:"priority,attr"`
			Category string `xml:"category,attr"`
			Message  string `xml:"LongMessage"`
			SourceLine struct {
				SourcePath string `xml:"sourcepath,attr"`
				Start      int    `xml:"start,attr"`
				End        int    `xml:"end,attr"`
			} `xml:"SourceLine"`
		} `xml:"BugInstance"`
	}

	if err := xml.Unmarshal([]byte(output), &spotbugsResult); err != nil {
		return []Issue{}
	}

	var issues []Issue
	for _, bug := range spotbugsResult.BugInstances {
		severity := "info"
		switch bug.Priority {
		case 1:
			severity = "error"
		case 2:
			severity = "warning"
		}

		issues = append(issues, Issue{
			File:     bug.SourceLine.SourcePath,
			Line:     bug.SourceLine.Start,
			Severity: severity,
			Rule:     bug.Type,
			Message:  bug.Message,
		})
	}

	return issues
}

// Ensure Java tools implement QualityTool interface.
var (
	_ QualityTool = (*GoogleJavaFormatTool)(nil)
	_ QualityTool = (*CheckstyleTool)(nil)
	_ QualityTool = (*SpotbugsTool)(nil)
)
