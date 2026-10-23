package quality

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReleaseCoverageBoundary(t *testing.T) {
	for _, tc := range []struct {
		name     string
		covered  int
		wantPass bool
	}{
		{"below-rounded-boundary", 7999, false},
		{"exact-boundary", 8000, true},
		{"above-boundary", 8001, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profile := filepath.Join(t.TempDir(), "coverage.out")
			data := fmt.Sprintf("mode: set\ngithub.com/Gizzahub/gzh-cli-quality/quality.go:30.1,31.1 %d 1\ngithub.com/Gizzahub/gzh-cli-quality/quality.go:32.1,33.1 %d 0\n", tc.covered, 10000-tc.covered)
			require.NoError(t, os.WriteFile(profile, []byte(data), 0o600))
			output, err := exec.Command("make", "release-coverage-check", "COVERAGE_OUT="+profile).CombinedOutput()
			if tc.wantPass {
				require.NoError(t, err, "%s", output)
			} else {
				require.Error(t, err, "%s", output)
			}
		})
	}
	t.Run("missing-report", func(t *testing.T) {
		output, err := exec.Command("make", "release-coverage-check", "COVERAGE_OUT="+filepath.Join(t.TempDir(), "missing.out")).CombinedOutput()
		require.Error(t, err, "%s", output)
	})
}
