// Package bench holds workload benchmarks for the Sass compiler.
//
// Workloads: bootstrap (the Bootstrap entry stylesheet from the
// bootstrap-main submodule) and huge / huge10 (the tracked bench/huge.scss,
// with huge10 built as a 10x in-memory concatenation at bench time).
//
// These benchmarks are measurement-only: run them with
// `go test -bench=. -benchmem ./bench/` and compare user CPU across builds
// (interleaved A/B, median of several runs). Byte-identity against Dart Sass
// is a separate gate (see docs/CONTRIBUTING.md), not asserted per iteration.
//
// A missing bootstrap-main checkout skips the bootstrap benchmark instead of
// failing: check out submodules with `git submodule update --init`.
package bench

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bancek/go-sass/compile"
	"github.com/bancek/go-sass/sassio"
)

func benchmarkPath(b *testing.B, path string) {
	b.Helper()
	if _, err := os.Stat(path); err != nil {
		b.Skipf("missing workload (check out submodules): %s", path)
	}
	io := sassio.NewDefaultIO()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := compile.Compile(path, io, nil); err != nil {
			b.Fatalf("compile %s: %v", path, err)
		}
	}
}

func benchmarkSource(b *testing.B, name, source string) {
	b.Helper()
	io := sassio.NewDefaultIO()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := compile.CompileString(source, io, nil); err != nil {
			b.Fatalf("compile %s: %v", name, err)
		}
	}
}

// BenchmarkBootstrap compiles bootstrap-main/scss/bootstrap.scss.
func BenchmarkBootstrap(b *testing.B) {
	benchmarkPath(b, filepath.Join("..", "bootstrap-main", "scss", "bootstrap.scss"))
}

func loadHuge(b *testing.B) string {
	b.Helper()
	// Test binaries run with the package directory as CWD.
	content, err := os.ReadFile("huge.scss")
	if err != nil {
		b.Fatalf("read huge.scss: %v", err)
	}
	return string(content)
}

// BenchmarkHuge compiles the tracked bench/huge.scss workload.
func BenchmarkHuge(b *testing.B) {
	huge := loadHuge(b)
	benchmarkSource(b, "huge", huge)
}

// BenchmarkHuge10 compiles a 10x concatenation of bench/huge.scss,
// built in memory at bench time (no extra tracked file).
func BenchmarkHuge10(b *testing.B) {
	huge := loadHuge(b)
	var sb strings.Builder
	sb.Grow(len(huge) * 10)
	for range 10 {
		sb.WriteString(huge)
	}
	benchmarkSource(b, "huge10", sb.String())
}
