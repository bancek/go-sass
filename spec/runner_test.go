//go:build spec

package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const defaultSpecPath = "../sass-spec/spec"

var specStatsResult *specStats

func TestMain(m *testing.M) {
	code := m.Run()
	if specStatsResult != nil {
		total := specStatsResult.passed + specStatsResult.failed + specStatsResult.todo
		fmt.Printf("%d/%d passed (%d todo)\n", specStatsResult.passed, total, specStatsResult.todo)
	}
	os.Exit(code)
}

func TestSpecSuite(t *testing.T) {
	specPath := os.Getenv("SASS_SPEC_PATH")
	specRoot := defaultSpecPath
	if specPath == "" {
		specPath = defaultSpecPath
		// SASS_SPEC is a convenience: just the sub-path under the default spec root.
		if sub := os.Getenv("SASS_SPEC"); sub != "" {
			specPath = filepath.Join(defaultSpecPath, sub)
		}
	} else {
		specRoot = findSpecRoot(specPath)
	}
	specStatsResult = RunSpecs(t, specPath, specRoot)
}

func findSpecRoot(specPath string) string {
	abs, err := filepath.Abs(specPath)
	if err != nil {
		return specPath
	}
	if strings.HasSuffix(abs, ".hrx") {
		abs = filepath.Dir(abs)
	}
	for dir := abs; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if filepath.Base(dir) == "spec" {
			return dir
		}
	}
	return specPath
}
