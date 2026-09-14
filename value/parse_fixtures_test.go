package value

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadFixture reads an input file and its golden .txt from test/fixtures/.
// name is a subdirectory-relative path like "scss/01-variable" or "css/basic".
// The file extension is inferred from the subdirectory name.
func loadFixture(t *testing.T, name string) (input []byte, want string) {
	t.Helper()
	base := filepath.Join("../test/fixtures", name)

	var ext string
	switch {
	case strings.HasPrefix(name, "css/"):
		ext = ".css"
	case strings.HasPrefix(name, "sass/"):
		ext = ".sass"
	default:
		ext = ".scss"
	}

	input, err := os.ReadFile(base + ext)
	if err != nil {
		t.Fatalf("reading %s: %v", ext, err)
	}

	wantBytes, err := os.ReadFile(base + ".txt")
	if err != nil {
		t.Fatalf("reading .txt golden: %v", err)
	}

	return input, string(wantBytes)
}

func scssFixtures(t *testing.T) []string {
	t.Helper()
	return listFixtures(t, "scss")
}

func sassFixtures(t *testing.T) []string {
	t.Helper()
	return listFixtures(t, "sass")
}

func cssFixtures(t *testing.T) []string {
	t.Helper()
	return listFixtures(t, "css")
}

func listFixtures(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir("../test/fixtures/" + dir)
	if err != nil {
		t.Fatalf("reading fixtures/%s: %v", dir, err)
	}
	var names []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".expect.css") || strings.HasSuffix(name, ".expect.scss") {
			continue
		}
		if before, ok := strings.CutSuffix(name, ".scss"); ok {
			names = append(names, before)
		} else if before, ok := strings.CutSuffix(name, ".sass"); ok {
			names = append(names, before)
		} else if before, ok := strings.CutSuffix(name, ".css"); ok {
			names = append(names, before)
		}
	}
	return names
}
