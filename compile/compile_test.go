package compile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bancek/go-sass/sassio"
)

func TestCompileBasic(t *testing.T) {
	result, err := CompileString(`a { color: red; }`, sassio.NewDefaultIO(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.CSS(), "{") || !strings.Contains(result.CSS(), "}") {
		t.Errorf("expected CSS with braces, got: %s", result.CSS())
	}
}

func TestSourceMapGeneration(t *testing.T) {
	result, err := CompileString(`a { color: red; }`, sassio.NewDefaultIO(), &CompileOptions{
		SourceMap: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	sm := result.SourceMap()
	if sm == nil {
		t.Fatal("expected source map")
	}

	jsonBytes, err := sm.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed["version"] != float64(3) {
		t.Errorf("version = %v, want 3", parsed["version"])
	}
	if parsed["mappings"] == "" {
		t.Errorf("expected non-empty mappings")
	}
}

func TestSourceMapDisabled(t *testing.T) {
	result, err := CompileString(`a { color: red; }`, sassio.NewDefaultIO(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceMap() != nil {
		t.Error("expected nil source map when disabled")
	}
}

func TestSourceMapCompressed(t *testing.T) {
	result, err := CompileString(`a { color: red; }`, sassio.NewDefaultIO(), &CompileOptions{
		SourceMap: true,
		Style:     OutputStyleCompressed,
	})
	if err != nil {
		t.Fatal(err)
	}

	sm := result.SourceMap()
	if sm == nil {
		t.Fatal("expected source map with compressed output")
	}

	jsonBytes, err := sm.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed["mappings"] == "" {
		t.Errorf("expected non-empty mappings")
	}
}

func TestScssFixtures(t *testing.T) {
	fixtures := scssFixtureNames(t)

	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			scssPath := filepath.Join("../test/fixtures/scss", name+".scss")
			source, err := os.ReadFile(scssPath)
			if err != nil {
				t.Fatalf("reading %s: %v", scssPath, err)
			}

			result, err := CompileString(string(source), sassio.NewDefaultIO(), nil)
			if err != nil {
				t.Fatalf("CompileString(%s): %v", name, err)
			}

			cssPath := filepath.Join("../test/fixtures/scss", name+".expect.css")
			expected, err := os.ReadFile(cssPath)
			if err != nil {
				if werr := os.WriteFile(cssPath, []byte(result.CSS()), 0644); werr != nil {
					t.Fatalf("failed to write expected CSS: %v", werr)
				}
				t.Logf("wrote expected CSS to %s", cssPath)
				return
			}

			got := strings.TrimSpace(result.CSS())
			want := strings.TrimSpace(string(expected))
			if got != want {
				t.Errorf("%s: CSS mismatch\n--- want:\n%s\n\n--- got:\n%s", name, want, got)
			}
		})
	}
}

func scssFixtureNames(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir("../test/fixtures/scss")
	if err != nil {
		t.Fatalf("reading fixtures/scss: %v", err)
	}
	var names []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".scss") {
			base := strings.TrimSuffix(name, ".scss")
			names = append(names, base)
		}
	}
	return names
}

func TestCompressOutput(t *testing.T) {
	result, err := CompileString(`a { color: red; }`, sassio.NewDefaultIO(), &CompileOptions{
		Style: OutputStyleCompressed,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := result.CSS()
	// Compressed output should have no newlines within the rule
	if strings.Contains(got, "\n") {
		t.Errorf("compressed output should not contain newlines, got: %q", got)
	}
	if !strings.Contains(got, "a{color:red}") {
		t.Errorf("compressed output should be minified, got: %q", got)
	}
}

func TestCharsetExpanded(t *testing.T) {
	// SCSS with a non-ASCII character should get @charset in expanded mode
	result, err := CompileString(`a { content: "café"; }`, sassio.NewDefaultIO(), nil)
	if err != nil {
		t.Fatal(err)
	}
	css := result.CSS()
	if !strings.HasPrefix(css, "@charset \"UTF-8\";\n") {
		t.Errorf("expanded non-ASCII CSS should have @charset prefix, got: %q", css)
	}
}

func TestCharsetCompressed(t *testing.T) {
	result, err := CompileString(`a { content: "café"; }`, sassio.NewDefaultIO(), &CompileOptions{
		Style:   OutputStyleCompressed,
		Charset: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	css := result.CSS()
	if !strings.HasPrefix(css, "\uFEFF") {
		t.Errorf("compressed non-ASCII CSS should have BOM prefix, got: %q", css)
	}
}

func TestCharsetAllAscii(t *testing.T) {
	// All-ASCII CSS should NOT get @charset or BOM
	result, err := CompileString(`a { color: red; }`, sassio.NewDefaultIO(), nil)
	if err != nil {
		t.Fatal(err)
	}
	css := result.CSS()
	if strings.HasPrefix(css, "@charset") {
		t.Errorf("all-ASCII CSS should not have @charset, got: %q", css)
	}
	if strings.HasPrefix(css, "\uFEFF") {
		t.Errorf("all-ASCII CSS should not have BOM, got: %q", css)
	}
}

func TestCharsetDisabled(t *testing.T) {
	// charset=false should suppress prefix even for non-ASCII
	result, err := CompileString(`a { content: "café"; }`, sassio.NewDefaultIO(), &CompileOptions{
		Charset: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	css := result.CSS()
	if strings.HasPrefix(css, "@charset") || strings.HasPrefix(css, "\uFEFF") {
		t.Errorf("charset=false should suppress prefix, got: %q", css)
	}
}

func TestCompileNilIo(t *testing.T) {
	_, err := CompileString(`a { color: red; }`, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil io")
	}
	want := "compile string: io must be set"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestCompileStringEmitErrorCss(t *testing.T) {
	result, err := CompileString(`@error "test";`, sassio.NewDefaultIO(), &CompileOptions{
		EmitErrorCss: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	css := result.CSS()
	if !strings.Contains(css, "Error:") {
		t.Errorf("emitErrorCss should produce error CSS, got: %s", css)
	}
}

func TestCompileStringNoEmitErrorCss(t *testing.T) {
	_, err := CompileString(`@error "test";`, sassio.NewDefaultIO(), &CompileOptions{
		EmitErrorCss: false,
	})
	if err == nil {
		t.Fatal("expected error to be propagated")
	}
}

func TestCompileOptionsDefault(t *testing.T) {
	result, err := CompileString(`a { color: red; }`, sassio.NewDefaultIO(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.CSS() == "" {
		t.Error("expected non-empty CSS with nil opts")
	}
}

func TestCompileStringSourceMap(t *testing.T) {
	result, err := CompileString(`a { color: red; }`, sassio.NewDefaultIO(), &CompileOptions{
		SourceMap: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	sm := result.SourceMap()
	if sm == nil {
		t.Fatal("expected source map when SourceMap=true")
	}
}

func TestCompileStylesheetErrorSass(t *testing.T) {
	tmpDir := t.TempDir()
	scssPath := filepath.Join(tmpDir, "error.scss")
	if err := os.WriteFile(scssPath, []byte("@error \"test\";\n"), 0644); err != nil {
		t.Fatal(err)
	}
	err := CompileStylesheet(sassio.NewDefaultIO(), scssPath, "", &CompileOptions{
		EmitErrorCss: false,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	se, ok := err.(*StylesheetError)
	if !ok {
		t.Fatalf("expected StylesheetError, got %T", err)
	}
	if se.ExitCode != 65 {
		t.Errorf("exit code = %d, want 65", se.ExitCode)
	}
	if !strings.Contains(se.Message, "Error:") {
		t.Errorf("message should contain 'Error:', got %q", se.Message)
	}
}

func TestCompileStylesheetFileOutput(t *testing.T) {
	tmpDir := t.TempDir()
	scssPath := filepath.Join(tmpDir, "input.scss")
	dest := filepath.Join(tmpDir, "output.css")
	if err := os.WriteFile(scssPath, []byte("a { color: red; }"), 0644); err != nil {
		t.Fatal(err)
	}
	err := CompileStylesheet(sassio.NewDefaultIO(), scssPath, dest, &CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(data))
	if !strings.Contains(got, "a") {
		t.Errorf("expected CSS in output file, got: %s", got)
	}
}

func TestStylesheetError(t *testing.T) {
	e := &StylesheetError{ExitCode: 65, Message: "test error"}
	if e.Error() != "test error" {
		t.Errorf("Error() = %q, want %q", e.Error(), "test error")
	}
	if e.ExitCode != 65 {
		t.Errorf("ExitCode = %d, want 65", e.ExitCode)
	}
}

// Dart `visitIfExpression` renders unresolved branch values with `toCssString`,
// not inspect form (#2808): a comma list loses its inspect parens, while quoted
// strings keep their quotes.
func TestModernIfRendersBranchesAsCSS(t *testing.T) {
	result, err := CompileString(`a { b: if(css(): (x, y)); c: if(css(): "s"); }`, sassio.NewDefaultIO(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.CSS(), "b: if(css(): x, y);") {
		t.Errorf("expected CSS-serialized comma list, got: %s", result.CSS())
	}
	if !strings.Contains(result.CSS(), `c: if(css(): "s");`) {
		t.Errorf("expected quoted string preserved, got: %s", result.CSS())
	}
}
