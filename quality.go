// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package quality

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Gizzahub/gzh-cli-quality/cache"
	"github.com/Gizzahub/gzh-cli-quality/config"
	"github.com/Gizzahub/gzh-cli-quality/detector"
	"github.com/Gizzahub/gzh-cli-quality/executor"
	"github.com/Gizzahub/gzh-cli-quality/report"
	"github.com/Gizzahub/gzh-cli-quality/tools"
)

const (
	statusSuccess = "✅"
	statusFailure = "❌"
)

// QualityManager manages the quality command functionality.
type QualityManager struct {
	registry     tools.ToolRegistry
	analyzer     *detector.ProjectAnalyzer
	executor     *executor.ParallelExecutor
	planner      *executor.ExecutionPlanner
	config       *config.Config
	cacheManager *cache.CacheManager
}

// NewQualityManager creates a new quality manager.
func NewQualityManager() *QualityManager {
	registry := tools.NewRegistry()

	// Register all available tools
	registerAllTools(registry)

	// Load configuration
	cfg, err := config.LoadConfig("")
	if err != nil {
		// Use default config if loading fails
		cfg = config.DefaultConfig()
	}

	analyzer := detector.NewProjectAnalyzer()
	adapter := &ProjectAnalyzerAdapter{analyzer}
	planner := executor.NewExecutionPlanner(adapter)

	// Initialize cache manager based on config
	var cacheManager *cache.CacheManager
	if cfg.Cache.Enabled {
		maxAge := parseDuration(cfg.Cache.MaxAge, 7*24*time.Hour)
		cacheManager, err = cache.NewCacheManager(cfg.GetCacheDirectory(), cfg.Cache.MaxSize, maxAge)
		if err != nil {
			// If cache initialization fails, continue without cache
			fmt.Printf("⚠️ 캐시 초기화 실패: %v (캐시 없이 계속 진행)\n", err)
			cacheManager = nil
		}
	}

	// Create executor with or without cache
	var parallelExecutor *executor.ParallelExecutor
	if cacheManager != nil {
		parallelExecutor = executor.NewParallelExecutorWithCache(runtime.NumCPU(), 10*time.Minute, cacheManager)
	} else {
		parallelExecutor = executor.NewParallelExecutor(runtime.NumCPU(), 10*time.Minute)
	}

	return &QualityManager{
		registry:     registry,
		analyzer:     analyzer,
		executor:     parallelExecutor,
		planner:      planner,
		config:       cfg,
		cacheManager: cacheManager,
	}
}

// parseDuration parses a duration string like "7d", "24h", "30m"
func parseDuration(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}

	// Handle day suffix "d"
	if s != "" && s[len(s)-1] == 'd' {
		days := 0
		_, err := fmt.Sscanf(s, "%dd", &days)
		if err == nil && days > 0 {
			return time.Duration(days) * 24 * time.Hour
		}
	}

	// Try standard duration parsing
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultVal
	}
	return d
}

// NewQualityCmd creates the quality command.
func NewQualityCmd() *cobra.Command {
	manager := NewQualityManager()

	cmd := &cobra.Command{
		Use:   "quality",
		Short: "통합 코드 품질 도구 (포매팅 + 린팅)",
		Long: `gz quality는 여러 프로그래밍 언어의 코드 포매팅과 린팅을 통합 제공합니다.

주요 명령어:
  run     모든 포매팅 및 린팅 도구 실행 (기본)
  check   린팅만 실행 (변경 없이 검사)
  init    프로젝트 설정 파일 자동 생성

도구 실행:
  tool        개별 도구 직접 실행
    gofumpt   Go 포매터
    ruff      Python 포매터+린터
    prettier  JavaScript 포매터
    clippy    Rust 린터
    ... (모든 설치된 도구)

관리 명령어:
  analyze  프로젝트 분석 및 권장 도구 표시
  install  품질 도구 설치
  upgrade  품질 도구 업그레이드
  version  품질 도구 버전 확인
  list     사용 가능한 품질 도구 목록 표시

사용 예시:
  gz quality run                      # 모든 도구 실행
  gz quality tool ruff --changed     # ruff로 변경된 파일만 처리
  gz quality tool gofumpt --staged   # gofumpt로 staged 파일만 처리
  gz quality run --format-only       # 포매팅 도구만 실행
  gz quality check --lint-only       # 린팅 도구만 실행`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(manager.newRunCmd())
	cmd.AddCommand(manager.newCheckCmd())
	cmd.AddCommand(manager.newInitCmd())
	cmd.AddCommand(manager.newAnalyzeCmd())
	cmd.AddCommand(manager.newInstallCmd())
	cmd.AddCommand(manager.newUpgradeCmd())
	cmd.AddCommand(manager.newVersionCmd())
	cmd.AddCommand(manager.newListCmd())
	cmd.AddCommand(manager.newToolCmd())

	// Cache management commands
	cmd.AddCommand(manager.newCacheClearCmd())
	cmd.AddCommand(manager.newCacheStatsCmd())

	// Language-specific subcommands removed - use direct tool commands instead

	return cmd
}

// addCommonExecutionFlags adds common execution flags to a command.
func addCommonExecutionFlags(cmd *cobra.Command) {
	cmd.Flags().StringSliceP("files", "f", nil, "특정 파일들만 처리")
	cmd.Flags().IntP("workers", "w", runtime.NumCPU(), "병렬 실행 워커 수")
	cmd.Flags().StringSlice("extra-args", nil, "도구에 전달할 추가 인수")
	cmd.Flags().Bool("dry-run", false, "실제 실행하지 않고 계획만 표시")
	cmd.Flags().BoolP("verbose", "v", false, "상세 출력")
	cmd.Flags().String("report", "", "리포트 생성 (json, html, markdown)")
	cmd.Flags().String("output", "", "리포트 출력 파일 경로")
}

// addGitFilterFlags adds Git-based filtering flags to a command.
func addGitFilterFlags(cmd *cobra.Command) {
	cmd.Flags().String("since", "", "특정 커밋 이후 변경된 파일만 처리 (예: HEAD~1, main)")
	cmd.Flags().Bool("staged", false, "Git staged 파일만 처리")
	cmd.Flags().Bool("changed", false, "변경된 파일만 처리 (staged + modified + untracked)")
}

// addCacheFlags adds cache control flags to a command.
func addCacheFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("cache", true, "결과 캐싱 활성화 (기본: 활성)")
	cmd.Flags().Bool("no-cache", false, "결과 캐싱 비활성화")
}

// newRunCmd creates the run subcommand.
func (m *QualityManager) newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "모든 포매팅 및 린팅 도구 실행",
		Long: `모든 사용 가능한 포매팅 및 린팅 도구를 자동으로 감지하여 실행합니다.
프로젝트의 언어를 자동으로 감지하고 적절한 도구들을 병렬로 실행합니다.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return m.runQuality(cmd, args)
		},
	}

	// Common flags
	addCommonExecutionFlags(cmd)
	addGitFilterFlags(cmd)
	addCacheFlags(cmd)

	// Run-specific flags
	cmd.Flags().BoolP("fix", "x", false, "자동 수정 적용 (지원하는 도구만)")
	cmd.Flags().Bool("format-only", false, "포매팅만 실행")
	cmd.Flags().Bool("lint-only", false, "린팅만 실행")

	return cmd
}

// executionOptions holds common options for run/check commands.
type executionOptions struct {
	files        []string
	fix          bool
	formatOnly   bool
	lintOnly     bool
	workers      int
	extraArgs    []string
	dryRun       bool
	verbose      bool
	reportFormat string
	outputPath   string
	since        string
	staged       bool
	changed      bool
	cacheEnabled bool
	// Display customization
	emptyMessage  string
	executePrefix string
}

// parseExecutionOptions parses common flags from a cobra command.
func parseExecutionOptions(cmd *cobra.Command) (*executionOptions, error) {
	opts := &executionOptions{}

	opts.files, _ = cmd.Flags().GetStringSlice("files")
	opts.fix, _ = cmd.Flags().GetBool("fix")
	opts.formatOnly, _ = cmd.Flags().GetBool("format-only")
	opts.lintOnly, _ = cmd.Flags().GetBool("lint-only")
	opts.workers, _ = cmd.Flags().GetInt("workers")
	opts.extraArgs, _ = cmd.Flags().GetStringSlice("extra-args")
	opts.dryRun, _ = cmd.Flags().GetBool("dry-run")
	opts.verbose, _ = cmd.Flags().GetBool("verbose")
	opts.reportFormat, _ = cmd.Flags().GetString("report")
	opts.outputPath, _ = cmd.Flags().GetString("output")

	// Git-based flags
	opts.since, _ = cmd.Flags().GetString("since")
	opts.staged, _ = cmd.Flags().GetBool("staged")
	opts.changed, _ = cmd.Flags().GetBool("changed")

	// Cache control flags
	cacheEnabled, _ := cmd.Flags().GetBool("cache")
	noCache, _ := cmd.Flags().GetBool("no-cache")
	opts.cacheEnabled = cacheEnabled && !noCache

	return opts, nil
}

// executeQuality is the common execution logic for run/check commands.
func (m *QualityManager) executeQuality(ctx context.Context, opts *executionOptions) error {
	m.updateCacheState(opts.cacheEnabled)

	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := m.validateGitFlags(opts.since, opts.staged, opts.changed); err != nil {
		return err
	}

	planOptions := executor.PlanOptions{
		Files:      opts.files,
		Fix:        opts.fix,
		FormatOnly: opts.formatOnly,
		LintOnly:   opts.lintOnly,
		ExtraArgs:  opts.extraArgs,
		Since:      opts.since,
		Staged:     opts.staged,
		Changed:    opts.changed,
	}

	plan, err := m.planner.CreatePlan(projectRoot, m.registry, planOptions)
	if err != nil {
		return fmt.Errorf("failed to create execution plan: %w", err)
	}

	if len(plan.Tasks) == 0 {
		fmt.Println(opts.emptyMessage)
		return nil
	}

	m.displayPlan(plan, opts.verbose)

	if opts.dryRun {
		fmt.Println("✨ 드라이런 모드: 실제 실행하지 않습니다.")
		return nil
	}

	cacheStatus := ""
	if m.executor.CacheEnabled() {
		cacheStatus = " (캐시 활성)"
	}
	fmt.Printf("%s %d개 작업을 %d개 워커로 실행합니다...%s\n",
		opts.executePrefix, len(plan.Tasks), opts.workers, cacheStatus)

	startTime := time.Now()
	results, err := m.executor.ExecuteParallel(ctx, plan, opts.workers)
	duration := time.Since(startTime)

	if err != nil {
		fmt.Printf("❌ 실행 중 오류 발생: %v\n", err)
		return err
	}

	m.displayResults(results, duration, opts.verbose)

	if opts.reportFormat != "" {
		if err := m.generateReport(results, duration, plan.TotalFiles, projectRoot, opts.reportFormat, opts.outputPath); err != nil {
			fmt.Printf("⚠️ 리포트 생성 실패: %v\n", err)
		}
	}

	return nil
}

// runQuality executes the main quality command logic.
func (m *QualityManager) runQuality(cmd *cobra.Command, _ []string) error {
	opts, err := parseExecutionOptions(cmd)
	if err != nil {
		return err
	}

	opts.emptyMessage = "🎯 처리할 작업이 없습니다."
	opts.executePrefix = "🚀"

	return m.executeQuality(cmd.Context(), opts)
}

// displayPlan shows the execution plan.
func (m *QualityManager) displayPlan(plan *tools.ExecutionPlan, verbose bool) {
	fmt.Printf("📋 실행 계획 (%d개 작업, %d개 파일, 예상 소요시간: %s)\n",
		len(plan.Tasks), plan.TotalFiles, plan.EstimatedDuration)

	if verbose {
		// Group tasks by language
		langTasks := make(map[string][]tools.Task)
		for _, task := range plan.Tasks {
			lang := task.Tool.Language()
			langTasks[lang] = append(langTasks[lang], task)
		}

		for lang, tasks := range langTasks {
			fmt.Printf("  %s:\n", lang)
			for _, task := range tasks {
				fmt.Printf("    - %s (%s) - %d개 파일\n",
					task.Tool.Name(), task.Tool.Type().String(), len(task.Files))
			}
		}
	}
}

// displayResults shows the execution results.
func (m *QualityManager) displayResults(results []*tools.Result, duration time.Duration, verbose bool) {
	fmt.Printf("\n✅ 완료! 총 소요시간: %v\n", duration.Round(time.Millisecond))

	successful := 0
	totalIssues := 0
	cachedCount := 0

	for _, result := range results {
		if result.Success {
			successful++
		}
		if result.Cached {
			cachedCount++
		}
		totalIssues += len(result.Issues)

		if verbose || !result.Success {
			status := statusSuccess
			if !result.Success {
				status = statusFailure
			}

			cachedLabel := ""
			if result.Cached {
				cachedLabel = " (캐시됨)"
			}

			fmt.Printf("%s %s (%s): %d개 파일, %v%s\n",
				status, result.Tool, result.Language, result.FilesProcessed, result.Duration, cachedLabel)

			if result.Error != "" {
				fmt.Printf("   오류: %s\n", result.Error)
			}

			if len(result.Issues) > 0 {
				fmt.Printf("   이슈: %d개\n", len(result.Issues))
				if verbose {
					for _, issue := range result.Issues {
						fmt.Printf("     %s:%d:%d: %s (%s)\n",
							issue.File, issue.Line, issue.Column, issue.Message, issue.Rule)
					}
				}
			}
		}
	}

	// Summary with cache info
	cacheInfo := ""
	if cachedCount > 0 {
		cacheInfo = fmt.Sprintf(", %d개 캐시 히트", cachedCount)
	}
	fmt.Printf("\n📊 요약: %d/%d 도구 성공, %d개 이슈 발견%s\n",
		successful, len(results), totalIssues, cacheInfo)
}

// newAnalyzeCmd creates the analyze subcommand.
func (m *QualityManager) newAnalyzeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "analyze",
		Short: "프로젝트 분석 및 권장 도구 표시",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			analysis, err := m.analyzer.AnalyzeProject(projectRoot, m.registry)
			if err != nil {
				return fmt.Errorf("failed to analyze project: %w", err)
			}

			fmt.Printf("🔍 프로젝트 분석: %s\n\n", analysis.ProjectRoot)

			// Show detected languages
			fmt.Println("감지된 언어:")
			for lang, files := range analysis.Languages {
				fmt.Printf("  %s: %d개 파일\n", lang, len(files))
			}

			// Show available tools
			fmt.Printf("\n사용 가능한 도구 (%d개):\n", len(analysis.AvailableTools))
			for _, tool := range analysis.AvailableTools {
				fmt.Printf("  ✅ %s\n", tool)
			}

			// Show recommended tools
			fmt.Println("\n권장 도구:")
			for lang, tools := range analysis.RecommendedTools {
				fmt.Printf("  %s: %s\n", lang, strings.Join(tools, ", "))
			}

			// Show config files
			if len(analysis.ConfigFiles) > 0 {
				fmt.Println("\n발견된 설정 파일:")
				for tool, config := range analysis.ConfigFiles {
					fmt.Printf("  %s: %s\n", tool, config)
				}
			}

			// Show issues
			if len(analysis.Issues) > 0 {
				fmt.Println("\n이슈:")
				for _, issue := range analysis.Issues {
					fmt.Printf("  ⚠️  %s\n", issue)
				}
			}

			return nil
		},
	}
}

// forEachTool executes an action on specified tools or all tools if none specified.
func (m *QualityManager) forEachTool(args []string, action func(tools.QualityTool) error, successMsg, failMsg string) {
	if len(args) == 0 {
		for _, tool := range m.registry.GetTools() {
			if err := action(tool); err != nil {
				fmt.Printf("❌ %s %s: %v\n", tool.Name(), failMsg, err)
			} else {
				fmt.Printf("✅ %s %s\n", tool.Name(), successMsg)
			}
		}
		return
	}

	for _, toolName := range args {
		tool := m.registry.FindTool(toolName)
		if tool == nil {
			fmt.Printf("❌ 도구를 찾을 수 없습니다: %s\n", toolName)
			continue
		}
		if err := action(tool); err != nil {
			fmt.Printf("❌ %s %s: %v\n", toolName, failMsg, err)
		} else {
			fmt.Printf("✅ %s %s\n", toolName, successMsg)
		}
	}
}

// groupToolsByLanguage groups tools by their language.
func groupToolsByLanguage(toolList []tools.QualityTool) map[string][]tools.QualityTool {
	langTools := make(map[string][]tools.QualityTool)
	for _, tool := range toolList {
		lang := tool.Language()
		langTools[lang] = append(langTools[lang], tool)
	}
	return langTools
}

// newInstallCmd creates the install subcommand.
func (m *QualityManager) newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install [tool-name...]",
		Short: "품질 도구 설치",
		Long:  "지정된 도구를 설치합니다. 도구명을 지정하지 않으면 모든 도구를 설치합니다.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Println("🔧 모든 품질 도구를 설치합니다...")
			}
			m.forEachTool(args, m.installTool, "설치 완료", "설치 실패")
			return nil
		},
	}
}

// newUpgradeCmd creates the upgrade subcommand.
func (m *QualityManager) newUpgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade [tool-name...]",
		Short: "품질 도구 업그레이드",
		Long:  "지정된 도구를 최신 버전으로 업그레이드합니다. 도구명을 지정하지 않으면 모든 도구를 업그레이드합니다.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Println("🔄 모든 품질 도구를 업그레이드합니다...")
			}
			m.forEachTool(args, m.upgradeTool, "업그레이드 완료", "업그레이드 실패")
			return nil
		},
	}
}

// newVersionCmd creates the version subcommand.
func (m *QualityManager) newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version [tool-name...]",
		Short: "품질 도구 버전 확인",
		Long:  "설치된 품질 도구들의 버전을 표시합니다. 도구명을 지정하지 않으면 모든 도구의 버전을 표시합니다.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Println("📋 설치된 품질 도구 버전:")
				for lang, toolList := range groupToolsByLanguage(m.registry.GetTools()) {
					fmt.Printf("\n%s:\n", lang)
					for _, tool := range toolList {
						m.showToolVersion(tool)
					}
				}
			} else {
				for _, toolName := range args {
					tool := m.registry.FindTool(toolName)
					if tool == nil {
						fmt.Printf("❌ 도구를 찾을 수 없습니다: %s\n", toolName)
						continue
					}
					m.showToolVersion(tool)
				}
			}
			return nil
		},
	}
}

// newListCmd creates the list subcommand.
func (m *QualityManager) newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "사용 가능한 품질 도구 목록 표시",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("📋 사용 가능한 품질 도구:")
			for lang, toolList := range groupToolsByLanguage(m.registry.GetTools()) {
				fmt.Printf("\n%s:\n", lang)
				for _, tool := range toolList {
					status := "❌"
					if tool.IsAvailable() {
						status = "✅"
					}
					fmt.Printf("  %s %s (%s)\n", status, tool.Name(), tool.Type().String())
				}
			}
			return nil
		},
	}
}

// installTool installs a specific tool.
func (m *QualityManager) installTool(tool tools.QualityTool) error {
	if tool.IsAvailable() {
		return nil // Already installed
	}

	return tool.Install()
}

// upgradeTool upgrades a specific tool.
func (m *QualityManager) upgradeTool(tool tools.QualityTool) error {
	if !tool.IsAvailable() {
		fmt.Printf("📦 %s is not installed, installing...\n", tool.Name())
		return tool.Install()
	}

	// Show current version before upgrade
	if version, err := tool.GetVersion(); err == nil {
		fmt.Printf("📦 Current %s version: %s\n", tool.Name(), version)
	}

	return tool.Upgrade()
}

// showToolVersion displays the version of a tool.
func (m *QualityManager) showToolVersion(tool tools.QualityTool) {
	if !tool.IsAvailable() {
		fmt.Printf("  ❌ %s: not installed\n", tool.Name())
		return
	}

	version, err := tool.GetVersion()
	if err != nil {
		fmt.Printf("  ⚠️  %s: error getting version (%v)\n", tool.Name(), err)
		return
	}

	status := "✅"
	fmt.Printf("  %s %s: %s\n", status, tool.Name(), version)
}

// generateReport creates and saves a quality report.
func (m *QualityManager) generateReport(results []*tools.Result, duration time.Duration, totalFiles int, projectRoot, format, outputPath string) error {
	generator := report.NewReportGenerator(projectRoot)
	qualityReport := generator.GenerateReport(results, duration, totalFiles)

	// Determine output path if not specified
	if outputPath == "" {
		outputPath = generator.GetReportPath(format)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	switch format {
	case "json":
		if err := generator.SaveJSON(qualityReport, outputPath); err != nil {
			return err
		}
	case "html":
		if err := generator.SaveHTML(qualityReport, outputPath); err != nil {
			return err
		}
	case "markdown", "md":
		if err := generator.SaveMarkdown(qualityReport, outputPath); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported report format: %s (supported: json, html, markdown)", format)
	}

	fmt.Printf("📄 리포트 생성 완료: %s\n", outputPath)
	return nil
}

// newCheckCmd creates the check subcommand.
func (m *QualityManager) newCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "린팅만 실행 (변경 없이 검사)",
		Long: `코드를 변경하지 않고 린팅만 수행합니다.
포맷팅 도구는 실행하지 않고 린터만 실행하여 코드 품질을 검사합니다.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return m.runCheck(cmd, args)
		},
	}

	// Common flags
	addCommonExecutionFlags(cmd)
	addGitFilterFlags(cmd)
	addCacheFlags(cmd)

	return cmd
}

// runCheck executes the check command (lint-only).
func (m *QualityManager) runCheck(cmd *cobra.Command, _ []string) error {
	opts, err := parseExecutionOptions(cmd)
	if err != nil {
		return err
	}

	// Override for check mode
	opts.fix = false // Never fix in check mode
	opts.formatOnly = false
	opts.lintOnly = true // Only run linters
	opts.emptyMessage = "🎯 검사할 작업이 없습니다."
	opts.executePrefix = "🔍"

	return m.executeQuality(cmd.Context(), opts)
}

// newInitCmd creates the init subcommand.
func (m *QualityManager) newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "프로젝트 설정 파일 자동 생성",
		Long: `프로젝트를 분석하여 적절한 .gzquality.yml 설정 파일을 자동으로 생성합니다.
감지된 언어와 사용 가능한 도구를 기반으로 최적화된 설정을 생성합니다.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return m.runInit(cmd, args)
		},
	}
}

// runInit executes the init command.
func (m *QualityManager) runInit(_ *cobra.Command, _ []string) error {
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	configPath := filepath.Join(projectRoot, ".gzquality.yml")

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("❌ 설정 파일이 이미 존재합니다: %s\n", configPath)
		fmt.Println("기존 파일을 삭제한 후 다시 실행하거나 직접 수정하세요.")
		return nil
	}

	// Analyze project
	analysis, err := m.analyzer.AnalyzeProject(projectRoot, m.registry)
	if err != nil {
		return fmt.Errorf("failed to analyze project: %w", err)
	}

	// Generate configuration based on analysis
	cfg := m.generateConfig(analysis)

	// Write config file
	configYAML, err := cfg.ToYAML()
	if err != nil {
		return fmt.Errorf("failed to generate YAML: %w", err)
	}

	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("✅ 설정 파일 생성 완료: %s\n", configPath)
	fmt.Printf("🔍 감지된 언어: %s\n", strings.Join(getLanguageList(analysis.Languages), ", "))
	fmt.Printf("🛠️ 사용 가능한 도구: %d개\n", len(analysis.AvailableTools))

	return nil
}

// Helper functions for init command.
func getLanguageList(languages map[string][]string) []string {
	langs := make([]string, 0, len(languages))
	for lang := range languages {
		langs = append(langs, lang)
	}
	return langs
}

// generateConfig creates a configuration based on project analysis.
func (m *QualityManager) generateConfig(analysis *detector.AnalysisResult) *Config {
	// This would be implemented with a proper Config struct
	// For now, we'll create a simple structure
	return &Config{
		Enabled: true,
		Languages: map[string]*LanguageConfig{
			"Go": {
				Enabled: contains(analysis.Languages, "Go"),
				Tools: map[string]*ToolConfig{
					"gofumpt":       {Enabled: true},
					"goimports":     {Enabled: true},
					"golangci-lint": {Enabled: true},
				},
			},
			"Python": {
				Enabled: contains(analysis.Languages, "Python"),
				Tools: map[string]*ToolConfig{
					"black":  {Enabled: true},
					"ruff":   {Enabled: true},
					"pylint": {Enabled: true},
				},
			},
		},
	}
}

func contains(languages map[string][]string, lang string) bool {
	_, exists := languages[lang]
	return exists
}

// Config structures for YAML generation.
type Config struct {
	Enabled   bool                       `yaml:"enabled"`
	Languages map[string]*LanguageConfig `yaml:"languages"`
}

type LanguageConfig struct {
	Enabled bool                   `yaml:"enabled"`
	Tools   map[string]*ToolConfig `yaml:"tools"`
}

type ToolConfig struct {
	Enabled bool `yaml:"enabled"`
}

func (c *Config) ToYAML() (string, error) {
	// Simple YAML generation - in a real implementation, use yaml package
	var sb strings.Builder

	sb.WriteString("# gzh-manager Quality Configuration\n")
	sb.WriteString("# Auto-generated by 'gz quality init'\n\n")
	sb.WriteString(fmt.Sprintf("enabled: %t\n\n", c.Enabled))
	sb.WriteString("languages:\n")

	for lang, config := range c.Languages {
		if !config.Enabled {
			continue
		}
		sb.WriteString(fmt.Sprintf("  %s:\n", lang))
		sb.WriteString(fmt.Sprintf("    enabled: %t\n", config.Enabled))
		sb.WriteString("    tools:\n")
		for tool, toolConfig := range config.Tools {
			sb.WriteString(fmt.Sprintf("      %s:\n", tool))
			sb.WriteString(fmt.Sprintf("        enabled: %t\n", toolConfig.Enabled))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// registerAllTools registers all available quality tools.
func registerAllTools(registry tools.ToolRegistry) {
	// Go tools
	registry.Register(tools.NewGofumptTool())
	registry.Register(tools.NewGoimportsTool())
	registry.Register(tools.NewGolangciLintTool())
	registry.Register(tools.NewGosecTool())
	registry.Register(tools.NewGovulncheckTool())
	registry.Register(tools.NewGciTool())
	registry.Register(tools.NewGolinesTool())

	// Python tools
	registry.Register(tools.NewBlackTool())
	registry.Register(tools.NewRuffTool())
	registry.Register(tools.NewPylintTool())
	registry.Register(tools.NewMypyTool())
	registry.Register(tools.NewBanditTool())

	// JavaScript/TypeScript tools
	registry.Register(tools.NewPrettierTool())
	registry.Register(tools.NewESLintTool())
	registry.Register(tools.NewTSCTool())

	// Rust tools
	registry.Register(tools.NewRustfmtTool())
	registry.Register(tools.NewClippyTool())
	registry.Register(tools.NewCargoFmtTool())

	// Markdown tools
	registry.Register(tools.NewMarkdownlintTool())

	// Java tools
	registry.Register(tools.NewGoogleJavaFormatTool())
	registry.Register(tools.NewCheckstyleTool())
	registry.Register(tools.NewSpotbugsTool())

	// Kotlin tools
	registry.Register(tools.NewKtlintTool())
	registry.Register(tools.NewDetektTool())

	// Shell tools
	registry.Register(tools.NewShellcheckTool())
	registry.Register(tools.NewShfmtTool())

	// C/C++ tools
	registry.Register(tools.NewClangFormatTool())
	registry.Register(tools.NewClangTidyTool())

	// YAML tools
	registry.Register(tools.NewYamllintTool())

	// SQL tools
	registry.Register(tools.NewSqlfluffTool())

	// Dockerfile tools
	registry.Register(tools.NewHadolintTool())

	// TOML tools
	registry.Register(tools.NewTaploTool())

	// Protobuf tools
	registry.Register(tools.NewBufTool())

	// CSS tools
	registry.Register(tools.NewStylelintTool())
}

// ProjectAnalyzerAdapter adapts detector.ProjectAnalyzer to executor.ProjectAnalyzer interface.
type ProjectAnalyzerAdapter struct {
	analyzer *detector.ProjectAnalyzer
}

func (a *ProjectAnalyzerAdapter) AnalyzeProject(projectRoot string, registry tools.ToolRegistry) (*executor.AnalysisResult, error) {
	result, err := a.analyzer.AnalyzeProject(projectRoot, registry)
	if err != nil {
		return nil, err
	}

	return &executor.AnalysisResult{
		ProjectRoot:      result.ProjectRoot,
		Languages:        result.Languages,
		AvailableTools:   result.AvailableTools,
		RecommendedTools: result.RecommendedTools,
		ConfigFiles:      result.ConfigFiles,
		Issues:           result.Issues,
	}, nil
}

func (a *ProjectAnalyzerAdapter) GetOptimalToolSelection(result *executor.AnalysisResult, registry tools.ToolRegistry) map[string][]tools.QualityTool {
	// Convert back to detector.AnalysisResult
	detectorResult := &detector.AnalysisResult{
		ProjectRoot:      result.ProjectRoot,
		Languages:        result.Languages,
		AvailableTools:   result.AvailableTools,
		RecommendedTools: result.RecommendedTools,
		ConfigFiles:      result.ConfigFiles,
		Issues:           result.Issues,
	}

	return a.analyzer.GetOptimalToolSelection(detectorResult, registry)
}

// newToolCmd creates the tool subcommand for direct tool access.
func (m *QualityManager) newToolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tool [tool-name]",
		Short: "개별 도구 직접 실행",
		Long: `특정 품질 도구를 직접 실행합니다.

사용 가능한 도구:
  gofumpt       Go 포매터
  goimports     Go 임포트 정리
  golangci-lint Go 린터
  ruff          Python 포매터+린터
  black         Python 포매터
  pylint        Python 린터
  prettier      JavaScript 포매터
  eslint        JavaScript 린터
  tsc           TypeScript 린터
  rustfmt       Rust 포매터
  clippy        Rust 린터
  cargo-fmt     Rust 포매터

사용 예시:
  gz quality tool gofumpt --staged    # gofumpt로 staged 파일만 처리
  gz quality tool ruff --changed      # ruff로 변경된 파일만 처리
  gz quality tool prettier --fix      # prettier로 자동 수정 적용`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			toolName := args[0]
			tool := m.registry.FindTool(toolName)
			if tool == nil {
				return fmt.Errorf("도구를 찾을 수 없습니다: %s. 'gz quality list'로 사용 가능한 도구를 확인하세요", toolName)
			}

			return m.runDirectTool(cmd, args[1:], tool)
		},
	}

	// Add flags for tool commands
	m.addDirectToolFlags(cmd)

	// Add individual tool subcommands for better discoverability
	m.addDirectToolCommands(cmd)

	return cmd
}

// addDirectToolCommands adds direct tool commands under tool subcommand.
func (m *QualityManager) addDirectToolCommands(parentCmd *cobra.Command) {
	allTools := m.registry.GetTools()

	for _, tool := range allTools {
		// Create a closure function to capture the tool properly
		func(currentTool tools.QualityTool) {
			toolName := currentTool.Name()
			toolCmd := &cobra.Command{
				Use:   toolName,
				Short: fmt.Sprintf("%s %s 도구 실행", currentTool.Language(), currentTool.Type().String()),
				Long:  fmt.Sprintf("%s 언어의 %s 도구를 직접 실행합니다.", currentTool.Language(), toolName),
				RunE: func(cmd *cobra.Command, args []string) error {
					return m.runDirectTool(cmd, args, currentTool)
				},
			}

			// Add common flags for direct tool commands
			m.addDirectToolFlags(toolCmd)
			parentCmd.AddCommand(toolCmd)
		}(tool)
	}
}

// addDirectToolFlags adds flags for direct tool commands.
func (m *QualityManager) addDirectToolFlags(cmd *cobra.Command) {
	cmd.Flags().StringSliceP("files", "f", nil, "특정 파일들만 처리")
	cmd.Flags().BoolP("fix", "x", false, "자동 수정 적용 (지원하는 도구만)")
	cmd.Flags().IntP("workers", "w", 1, "병렬 실행 워커 수 (기본값: 1, 단일 도구)")
	cmd.Flags().StringSlice("extra-args", nil, "도구에 전달할 추가 인수")
	cmd.Flags().Bool("dry-run", false, "실제 실행하지 않고 계획만 표시")
	cmd.Flags().BoolP("verbose", "v", false, "상세 출력")

	addGitFilterFlags(cmd)
}

// runDirectTool executes a specific tool directly.
func (m *QualityManager) runDirectTool(cmd *cobra.Command, _ []string, tool tools.QualityTool) error {
	ctx := cmd.Context()

	// Get flags
	files, _ := cmd.Flags().GetStringSlice("files")
	fix, _ := cmd.Flags().GetBool("fix")
	workers, _ := cmd.Flags().GetInt("workers")
	extraArgs, _ := cmd.Flags().GetStringSlice("extra-args")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Git-based flags
	since, _ := cmd.Flags().GetString("since")
	staged, _ := cmd.Flags().GetBool("staged")
	changed, _ := cmd.Flags().GetBool("changed")

	// Get project root
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Validate Git flags
	if err := m.validateGitFlags(since, staged, changed); err != nil {
		return err
	}

	// Create execution plan with specific tool filter
	planOptions := executor.PlanOptions{
		Files:      files,
		Fix:        fix,
		ExtraArgs:  extraArgs,
		Language:   tool.Language(),
		ToolFilter: []string{tool.Name()}, // Only this specific tool
		Since:      since,
		Staged:     staged,
		Changed:    changed,
	}

	plan, err := m.planner.CreatePlan(projectRoot, m.registry, planOptions)
	if err != nil {
		return fmt.Errorf("failed to create execution plan: %w", err)
	}

	if len(plan.Tasks) == 0 {
		fmt.Printf("🎯 %s 도구로 처리할 파일이 없습니다.\n", tool.Name())
		return nil
	}

	// Display plan
	m.displayPlan(plan, verbose)

	if dryRun {
		fmt.Println("✨ 드라이런 모드: 실제 실행하지 않습니다.")
		return nil
	}

	// Execute plan
	fmt.Printf("🚀 %s: %d개 작업을 %d개 워커로 실행합니다...\n", tool.Name(), len(plan.Tasks), workers)

	startTime := time.Now()
	results, err := m.executor.ExecuteParallel(ctx, plan, workers)
	duration := time.Since(startTime)

	if err != nil {
		fmt.Printf("❌ 실행 중 오류 발생: %v\n", err)
		return err
	}

	// Display results
	m.displayResults(results, duration, verbose)

	return nil
}

// validateGitFlags validates Git-based filtering flags.
func (m *QualityManager) validateGitFlags(since string, staged, changed bool) error {
	// Count how many Git flags are set
	gitFlagCount := 0
	if since != "" {
		gitFlagCount++
	}
	if staged {
		gitFlagCount++
	}
	if changed {
		gitFlagCount++
	}

	// Only one Git flag can be used at a time
	if gitFlagCount > 1 {
		return fmt.Errorf("only one of --since, --staged, or --changed can be used at a time")
	}

	return nil
}

// updateCacheState enables or disables caching based on the flag.
func (m *QualityManager) updateCacheState(enabled bool) {
	if m.cacheManager != nil {
		m.cacheManager.SetEnabled(enabled)
	}
}

// newCacheClearCmd creates the cache-clear subcommand.
func (m *QualityManager) newCacheClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cache-clear",
		Short: "캐시 삭제",
		Long:  "모든 캐시 항목을 삭제합니다.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if m.cacheManager == nil {
				fmt.Println("⚠️ 캐시가 비활성화되어 있습니다.")
				return nil
			}

			if err := m.cacheManager.InvalidateAll(); err != nil {
				return fmt.Errorf("캐시 삭제 실패: %w", err)
			}

			fmt.Println("✅ 캐시가 삭제되었습니다.")
			return nil
		},
	}
}

// newCacheStatsCmd creates the cache-stats subcommand.
func (m *QualityManager) newCacheStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cache-stats",
		Short: "캐시 통계 표시",
		Long:  "캐시 사용 현황 및 통계를 표시합니다.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if m.cacheManager == nil {
				fmt.Println("⚠️ 캐시가 비활성화되어 있습니다.")
				return nil
			}

			stats := m.cacheManager.Stats()

			fmt.Println("📊 캐시 통계:")
			fmt.Printf("  캐시 디렉토리: %s\n", m.config.GetCacheDirectory())
			fmt.Printf("  캐시 항목: %d개\n", stats.Entries)
			fmt.Printf("  캐시 크기: %s\n", formatBytes(stats.SizeBytes))
			fmt.Printf("  최대 크기: %s\n", formatBytes(m.config.Cache.MaxSize))
			fmt.Printf("  캐시 히트: %d회\n", stats.HitCount)
			fmt.Printf("  캐시 미스: %d회\n", stats.MissCount)
			fmt.Printf("  히트율: %.1f%%\n", stats.HitRate*100)

			if !stats.OldestEntry.IsZero() {
				fmt.Printf("  가장 오래된 항목: %s\n", stats.OldestEntry.Format("2006-01-02 15:04:05"))
			}
			if !stats.NewestEntry.IsZero() {
				fmt.Printf("  가장 최근 항목: %s\n", stats.NewestEntry.Format("2006-01-02 15:04:05"))
			}

			return nil
		},
	}
}

// formatBytes formats byte size to human readable format.
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
