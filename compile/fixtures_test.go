package compile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadScssFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("../test/fixtures/scss", name+".scss")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(source)
}

func loadExpectedCSS(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("../test/fixtures/scss", name+".expect.css")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("no expected CSS for %s: %v", name, err)
	}
	return strings.TrimSpace(string(source))
}
