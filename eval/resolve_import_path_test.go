package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bancek/go-sass/sassio"
)

func TestResolveImportPath_ExactExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "style.scss")
	if err := os.WriteFile(path, []byte("a {}"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	result, err := resolveImportPath(d, path, false)
	if err != nil {
		t.Fatal(err)
	}
	if result != path {
		t.Fatalf("got %q, want %q", result, path)
	}
}

func TestResolveImportPath_NoExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "style.scss")
	if err := os.WriteFile(path, []byte("a {}"), 0644); err != nil {
		t.Fatal(err)
	}
	noExt := filepath.Join(dir, "style")

	d := newDefaultIO()
	result, err := resolveImportPath(d, noExt, false)
	if err != nil {
		t.Fatal(err)
	}
	if result != path {
		t.Fatalf("got %q, want %q", result, path)
	}
}

func TestResolveImportPath_Partial(t *testing.T) {
	dir := t.TempDir()
	partial := filepath.Join(dir, "_partial.scss")
	if err := os.WriteFile(partial, []byte("a {}"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	base := filepath.Join(dir, "partial.scss")
	result, err := resolveImportPath(d, base, false)
	if err != nil {
		t.Fatal(err)
	}
	if result != partial {
		t.Fatalf("got %q, want %q", result, partial)
	}
}

func TestResolveImportPath_ImportSuffix(t *testing.T) {
	dir := t.TempDir()
	importFile := filepath.Join(dir, "style.import.scss")
	if err := os.WriteFile(importFile, []byte("a {}"), 0644); err != nil {
		t.Fatal(err)
	}
	regularFile := filepath.Join(dir, "style.scss")
	if err := os.WriteFile(regularFile, []byte("b {}"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	result, err := resolveImportPath(d, filepath.Join(dir, "style"), true)
	if err != nil {
		t.Fatal(err)
	}
	if result != importFile {
		t.Fatalf("got %q, want %q (import file should take priority)", result, importFile)
	}
}

func TestResolveImportPath_ImportSuffix_NoFromImport(t *testing.T) {
	dir := t.TempDir()
	importFile := filepath.Join(dir, "style.import.scss")
	if err := os.WriteFile(importFile, []byte("a {}"), 0644); err != nil {
		t.Fatal(err)
	}
	regularFile := filepath.Join(dir, "style.scss")
	if err := os.WriteFile(regularFile, []byte("b {}"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	result, err := resolveImportPath(d, filepath.Join(dir, "style"), false)
	if err != nil {
		t.Fatal(err)
	}
	if result != regularFile {
		t.Fatalf("got %q, want %q (without fromImport, import file should not be used)", result, regularFile)
	}
}

func TestResolveImportPath_Index(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "mypackage")
	if err := os.Mkdir(pkg, 0755); err != nil {
		t.Fatal(err)
	}
	index := filepath.Join(pkg, "_index.scss")
	if err := os.WriteFile(index, []byte("a {}"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	result, err := resolveImportPath(d, pkg, false)
	if err != nil {
		t.Fatal(err)
	}
	if result != index {
		t.Fatalf("got %q, want %q", result, index)
	}
}

func TestResolveImportPath_NotFound(t *testing.T) {
	d := newDefaultIO()
	result, err := resolveImportPath(d, "/nonexistent/path/to/file", false)
	if err != nil {
		t.Fatal(err)
	}
	if result != "" {
		t.Fatalf("got %q, want empty string (not found)", result)
	}
}

func TestResolveImportPath_Ambiguous(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "style.scss")
	b := filepath.Join(dir, "style.sass")
	if err := os.WriteFile(a, []byte("a {}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	_, err := resolveImportPath(d, filepath.Join(dir, "style"), false)
	if err == nil {
		t.Fatal("expected error for ambiguous import")
	}
	want := strings.Join([]string{
		"It's not clear which file to import. Found:",
		"  " + filepath.ToSlash(b),
		"  " + filepath.ToSlash(a),
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q\n  want %q", err.Error(), want)
	}
}

func TestResolveImportPath_Ambiguous_WithPartial(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "style.scss")
	b := filepath.Join(dir, "_style.scss")
	if err := os.WriteFile(a, []byte("a {}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	_, err := resolveImportPath(d, filepath.Join(dir, "style"), false)
	if err == nil {
		t.Fatal("expected error for ambiguous import")
	}
	want := strings.Join([]string{
		"It's not clear which file to import. Found:",
		"  " + filepath.ToSlash(b),
		"  " + filepath.ToSlash(a),
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q\n  want %q", err.Error(), want)
	}
}

func TestResolveImportPath_SassExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "style.sass")
	if err := os.WriteFile(path, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	result, err := resolveImportPath(d, path, false)
	if err != nil {
		t.Fatal(err)
	}
	if result != path {
		t.Fatalf("got %q, want %q", result, path)
	}
}

func TestResolveImportPath_CssExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "style.css")
	if err := os.WriteFile(path, []byte("a {}"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	result, err := resolveImportPath(d, path, false)
	if err != nil {
		t.Fatal(err)
	}
	if result != path {
		t.Fatalf("got %q, want %q", result, path)
	}
}

func TestResolveImportPath_CssExtension_Import(t *testing.T) {
	dir := t.TempDir()
	importFile := filepath.Join(dir, "style.import.css")
	if err := os.WriteFile(importFile, []byte("a {}"), 0644); err != nil {
		t.Fatal(err)
	}
	regularFile := filepath.Join(dir, "style.css")
	if err := os.WriteFile(regularFile, []byte("b {}"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	result, err := resolveImportPath(d, filepath.Join(dir, "style.css"), true)
	if err != nil {
		t.Fatal(err)
	}
	if result != importFile {
		t.Fatalf("got %q, want %q (import file should take priority)", result, importFile)
	}
}

func TestResolveImportPath_DirNotFound(t *testing.T) {
	d := newDefaultIO()
	result, err := resolveImportPath(d, "/nonexistent/dir/path", false)
	if err != nil {
		t.Fatal(err)
	}
	if result != "" {
		t.Fatalf("got %q, want empty string", result)
	}
}

func newDefaultIO() sassio.IO {
	return sassio.NewDefaultIO()
}
