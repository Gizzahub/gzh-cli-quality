package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type commandTool interface {
	QualityTool
	BuildCommand([]string, ExecuteOptions) *exec.Cmd
	ParseOutput(string) []Issue
}

func adapterTools() []commandTool {
	return []commandTool{NewGofumptTool(), NewGoimportsTool(), NewGolangciLintTool(), NewGosecTool(), NewGovulncheckTool(), NewGciTool(), NewGolinesTool(), NewBlackTool(), NewRuffTool(), NewPylintTool(), NewMypyTool(), NewBanditTool(), NewRustfmtTool(), NewClippyTool(), NewCargoFmtTool(), NewPrettierTool(), NewESLintTool(), NewTSCTool(), NewClangFormatTool(), NewClangTidyTool(), NewGoogleJavaFormatTool(), NewCheckstyleTool(), NewSpotbugsTool(), NewKtlintTool(), NewDetektTool(), NewStylelintTool(), NewHadolintTool(), NewMarkdownlintTool(), NewBufTool(), NewShellcheckTool(), NewShfmtTool(), NewSqlfluffTool(), NewTaploTool(), NewYamllintTool()}
}

func fakeExecutable(t *testing.T, dir, name, script string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0o755))
	return path
}

// Every registered adapter must execute its own arguments rather than the promoted
// BaseTool implementation, and preserve caller environment and working directory.
func TestAdapterExecutionDispatch(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, tool := range adapterTools() {
		t.Run(tool.Name(), func(t *testing.T) {
			name := filepath.Base(tool.BuildCommand(nil, ExecuteOptions{}).Args[0])
			fakeExecutable(t, dir, name, `{
printf 'cwd=%s\nenv=%s\n' "$PWD" "$QUALITY_TEST_VALUE"
printf 'arg=%s\n' "$@"
} > "$QUALITY_TEST_LOG"`)
			files := []string{"source.go", "source.py", "source.rs", "source.js", "source.ts", "source.cpp", "source.java", "source.kt", "source.css", "Dockerfile", "source.md", "source.proto", "source.sh", "source.sql", "source.toml", "source.yaml"}
			for _, options := range []ExecuteOptions{{}, {Fix: true, ConfigFile: "custom.conf", ExtraArgs: []string{"--test-extra"}}, {FormatOnly: true}, {LintOnly: true}} {
				options.ProjectRoot = dir
				log := filepath.Join(dir, "arguments.log")
				options.Env = map[string]string{"QUALITY_TEST_VALUE": "value with spaces", "QUALITY_TEST_LOG": log}
				result, err := tool.Execute(context.Background(), files, options)
				require.NoError(t, err)
				require.True(t, result.Success, result.Error)
				require.Equal(t, len(files), result.FilesProcessed)
				observed, err := os.ReadFile(log)
				require.NoError(t, err)
				require.Contains(t, string(observed), "cwd="+dir)
				require.Contains(t, string(observed), "env=value with spaces")
				if _, ruff := tool.(*RuffTool); !ruff || options.FormatOnly || options.LintOnly {
					for _, arg := range tool.BuildCommand(files, options).Args[1:] {
						require.Contains(t, string(observed), "arg="+arg+"\n")
					}
				}
			}
			// Exercise project-wide defaults, configuration and invalid output branches.
			require.NotEmpty(t, tool.BuildCommand(nil, ExecuteOptions{Fix: true, ConfigFile: "custom.conf", ExtraArgs: []string{"--test-extra"}}).Args)
			require.Empty(t, tool.ParseOutput(""))
			require.Empty(t, tool.ParseOutput("not a diagnostic"))
		})
	}
}

func TestAdapterParsers(t *testing.T) {
	tests := []struct {
		name   string
		tool   commandTool
		output string
		want   Issue
	}{
		{"clang", NewClangTidyTool(), "src/a.cpp:2:3: warning: unused variable [unused]\nnoise", Issue{File: "src/a.cpp", Line: 2, Column: 3, Severity: "warning", Rule: "unused", Message: "unused variable"}},
		{"checkstyle", NewCheckstyleTool(), `<checkstyle><file name="A.java"><error line="2" column="3" severity="error" message="bad space" source="checks.Space"/></file></checkstyle>`, Issue{File: "A.java", Line: 2, Column: 3, Severity: "error", Rule: "Space", Message: "bad space"}},
		{"spotbugs", NewSpotbugsTool(), `<BugCollection><BugInstance type="NULL" priority="1"><LongMessage>null value</LongMessage><SourceLine sourcepath="A.java" start="2"/></BugInstance></BugCollection>`, Issue{File: "A.java", Line: 2, Severity: "error", Rule: "NULL", Message: "null value"}},
		{"ktlint", NewKtlintTool(), `[{"file":"a.kt","errors":[{"line":2,"column":3,"message":"indent","rule":"indentation"}]}]`, Issue{File: "a.kt", Line: 2, Column: 3, Severity: "error", Rule: "indentation", Message: "indent"}},
		{"detekt", NewDetektTool(), "a.kt:2:3: Complexity - too complex", Issue{File: "a.kt", Line: 2, Column: 3, Severity: "warning", Rule: "Complexity", Message: "too complex"}},
		{"stylelint", NewStylelintTool(), `[{"source":"a.css","warnings":[{"line":2,"column":3,"rule":"color","severity":"error","text":"invalid color"}]}]`, Issue{File: "a.css", Line: 2, Column: 3, Severity: "error", Rule: "color", Message: "invalid color"}},
		{"hadolint", NewHadolintTool(), `[{"file":"Dockerfile","line":2,"column":3,"code":"DL3001","level":"error","message":"bad command"}]`, Issue{File: "Dockerfile", Line: 2, Column: 3, Severity: "error", Rule: "DL3001", Message: "bad command"}},
		{"markdown", NewMarkdownlintTool(), "README.md:2 MD013/line-length Line too long\nnoise", Issue{File: "README.md", Line: 2, Severity: "warning", Rule: "MD013/line-length", Message: "Line too long"}},
		{"buf", NewBufTool(), "{\"path\":\"a.proto\",\"start_line\":2,\"start_column\":3,\"type\":\"PACKAGE\",\"message\":\"missing package\"}\ninvalid\n", Issue{File: "a.proto", Line: 2, Column: 3, Severity: "error", Rule: "PACKAGE", Message: "missing package"}},
		{"shellcheck", NewShellcheckTool(), `[{"file":"a.sh","line":2,"column":3,"level":"warning","code":2086,"message":"quote variable"}]`, Issue{File: "a.sh", Line: 2, Column: 3, Severity: "warning", Rule: "SC2086", Message: "quote variable"}},
		{"sqlfluff", NewSqlfluffTool(), `[{"filepath":"a.sql","violations":[{"start_line_no":2,"start_line_pos":3,"code":"LT01","description":"spacing"}]}]`, Issue{File: "a.sql", Line: 2, Column: 3, Severity: "warning", Rule: "LT01", Message: "spacing"}},
		{"taplo", NewTaploTool(), "error[syntax]: missing value\n  --> a.toml:2:3\n", Issue{File: "a.toml", Line: 2, Column: 3, Severity: "error", Rule: "syntax", Message: "missing value"}},
		{"yaml", NewYamllintTool(), "a.yaml:2:3: [warning] extra spaces (trailing-spaces)\nnoise", Issue{File: "a.yaml", Line: 2, Column: 3, Severity: "warning", Rule: "trailing-spaces", Message: "extra spaces"}},
		{"gosec", NewGosecTool(), `{"Issues":[{"file":"a.go","line":"2","column":"3","severity":"HIGH","rule_id":"G101","details":"secret"}]}`, Issue{File: "a.go", Line: 2, Column: 3, Severity: "HIGH", Rule: "G101", Message: "secret"}},
		{"govulncheck", NewGovulncheckTool(), "{\"finding\":{\"osv\":\"GO-123\",\"trace\":[{\"module\":\"example.org/mod\"}]}}\ninvalid\n{}\n", Issue{Severity: "HIGH", Rule: "GO-123", Message: "Vulnerability in example.org/mod"}},
		{"mypy", NewMypyTool(), "{\"file\":\"a.py\",\"line\":2,\"column\":3,\"severity\":\"error\",\"code\":\"arg-type\",\"message\":\"bad type\"}\ninvalid\n", Issue{File: "a.py", Line: 2, Column: 3, Severity: "error", Rule: "arg-type", Message: "bad type"}},
		{"bandit", NewBanditTool(), `{"results":[{"filename":"a.py","line_number":2,"test_id":"B101","test_name":"assert_used","issue_severity":"HIGH","issue_confidence":"HIGH","issue_text":"assert found"}]}`, Issue{File: "a.py", Line: 2, Severity: "high", Rule: "B101", Message: "[assert_used] assert found (confidence: HIGH)"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { require.Equal(t, []Issue{tt.want}, tt.tool.ParseOutput(tt.output)) })
	}
}

func TestFormatterCheckPreservesFiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	// The fake implements only the adapters' explicit mutation flags. This detects
	// accidental write flags in check mode without installing external formatters.
	script := `write=false
name=${0##*/}
case "$name" in black|rustfmt|cargo|ruff|taplo) write=true;; esac
for arg in "$@"; do
 case "$arg" in
 --check|--dry-run|--diff|--exit-code) write=false;;
 -w|--write|-F|write|fix) write=true;;
 -i) if [ "$name" != shfmt ]; then write=true; fi;;
 esac
done
if $write; then printf 'formatted\n' > "$QUALITY_TEST_FILE"; fi`
	for _, tool := range adapterTools() {
		if tool.Type() == LINT {
			continue
		}
		t.Run(tool.Name(), func(t *testing.T) {
			fakeExecutable(t, dir, filepath.Base(tool.BuildCommand(nil, ExecuteOptions{}).Args[0]), script)
			file := filepath.Join(dir, "input.txt")
			original := []byte("original bytes\n")
			require.NoError(t, os.WriteFile(file, original, 0o644))
			options := ExecuteOptions{ProjectRoot: dir, FormatOnly: true, Env: map[string]string{"QUALITY_TEST_FILE": file}}
			result, err := tool.Execute(context.Background(), nil, options)
			require.NoError(t, err)
			require.True(t, result.Success, result.Error)
			got, err := os.ReadFile(file)
			require.NoError(t, err)
			require.Equal(t, original, got)
			options.Fix = true
			result, err = tool.Execute(context.Background(), nil, options)
			require.NoError(t, err)
			require.True(t, result.Success, result.Error)
			// Black, rustfmt, cargo fmt, Ruff format and Taplo format write by default.
			got, err = os.ReadFile(file)
			require.NoError(t, err)
			require.Equal(t, "formatted\n", string(got))
		})
	}
}

func TestExecutionCancellationAndFailures(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	fakeExecutable(t, dir, "yamllint", `printf 'a.yaml:2:3: [error] bad value (syntax)\n'; exit 1`)
	result, err := NewYamllintTool().Execute(context.Background(), []string{"a.yaml"}, ExecuteOptions{})
	require.NoError(t, err)
	require.False(t, result.Success)
	require.Len(t, result.Issues, 1)
	fakeExecutable(t, dir, "yamllint", `exec sleep 10`)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	result, err = NewYamllintTool().Execute(ctx, nil, ExecuteOptions{})
	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "deadline exceeded")
	require.Less(t, result.Duration, 2*time.Second)
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	result, err = NewYamllintTool().Execute(ctx, nil, ExecuteOptions{})
	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "canceled")
}

func TestFormatterDifferencesFailCheck(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, tool := range []commandTool{NewGofumptTool(), NewGoimportsTool(), NewGciTool(), NewGolinesTool(), NewShfmtTool()} {
		t.Run(tool.Name(), func(t *testing.T) {
			fakeExecutable(t, dir, tool.Name(), `printf 'input.go\n'`)
			result, err := tool.Execute(context.Background(), nil, ExecuteOptions{})
			require.NoError(t, err)
			require.False(t, result.Success)
			require.Equal(t, "formatting changes required", result.Error)
			fakeExecutable(t, dir, tool.Name(), `exit 0`)
			result, err = tool.Execute(context.Background(), nil, ExecuteOptions{})
			require.NoError(t, err)
			require.True(t, result.Success, result.Error)
		})
	}
}
