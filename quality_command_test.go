package quality_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	quality "github.com/Gizzahub/gzh-cli-quality"
)

func TestQualityRootArguments(t *testing.T) {
	t.Chdir(t.TempDir())
	// 생성자가 사용자 설정을 읽거나 사용자 캐시를 만들지 않게 로컬 설정을 둔다.
	settings := []byte("cache:\n  enabled: false\n")
	require.NoError(t, os.WriteFile(".gzquality.yaml", settings, 0o600))
	for _, tc := range []struct {
		name     string
		embedded bool
		args     []string
		wantErr  bool
	}{
		{name: "standalone unknown", args: []string{"nosuch"}, wantErr: true},
		{name: "embedded unknown", embedded: true, args: []string{"quality", "nosuch"}, wantErr: true},
		{name: "standalone bare"},
		{name: "embedded bare", embedded: true, args: []string{"quality"}},
		{name: "standalone help", args: []string{"--help"}},
		{name: "embedded help", embedded: true, args: []string{"quality", "--help"}},
		{name: "known subcommand help", embedded: true, args: []string{"quality", "run", "--help"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := quality.NewQualityCmd()
			if tc.embedded {
				parent := &cobra.Command{Use: "gz"}
				parent.AddCommand(root)
				root = parent
			}
			var output bytes.Buffer
			root.SetOut(&output)
			root.SetErr(&output)
			root.SetArgs(tc.args)
			err := root.Execute()
			if tc.wantErr {
				require.ErrorContains(t, err, `unknown command "nosuch"`)
			} else {
				require.NoError(t, err)
				require.Contains(t, output.String(), "Usage:")
			}
		})
	}
	got, err := os.ReadFile(".gzquality.yaml")
	require.NoError(t, err)
	require.Equal(t, settings, got)
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	require.Len(t, entries, 1, "help and argument validation must not create project files")
}
