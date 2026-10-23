package quality

import (
	"bytes"
	"os"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestHomebrewCaskQuarantineFlight(t *testing.T) {
	data, err := os.ReadFile(".goreleaser.yml")
	require.NoError(t, err)
	var config struct {
		Casks []struct {
			Name        string `yaml:"name"`
			CustomBlock string `yaml:"custom_block"`
		} `yaml:"homebrew_casks"`
	}
	require.NoError(t, yaml.Unmarshal(data, &config))
	require.Len(t, config.Casks, 1)
	require.Equal(t, "gz-quality", config.Casks[0].Name)
	flight, err := template.New("custom_block").Option("missingkey=error").Parse(config.Casks[0].CustomBlock)
	require.NoError(t, err)
	var rendered bytes.Buffer
	require.NoError(t, flight.Execute(&rendered, nil))
	require.Equal(t, `postflight_steps do
  on_macos do
    run "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "{{staged_path}}/gz-quality"]
  end
end
`, rendered.String(), "flight must preserve Homebrew's runtime token and scope xattr removal to the binary on macOS")
}
