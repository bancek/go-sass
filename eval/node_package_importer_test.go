package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sassurl"
)

// writeNodePackage lays out a package under <dir>/node_modules/p and returns a
// NodePackageImporter rooted at <dir>.
func writeNodePackage(t *testing.T, dir string, files map[string]string) *NodePackageImporter {
	t.Helper()
	for name, contents := range files {
		path := filepath.Join(dir, "node_modules", "p", name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return NewNodePackageImporter(dir, newDefaultIO())
}

func canonicalizePkg(t *testing.T, imp *NodePackageImporter, fromImport bool) string {
	t.Helper()
	u, err := sassurl.Parse("pkg:p")
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewCanonicalizeContext(nil, fromImport)
	result, err := imp.Canonicalize(u, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("Canonicalize returned nil")
	}
	return result.String()
}

// Dart loads the import-only variant through exports for @import but not for
// @use (node_package.dart _resolveImportOnly, #2772).
func TestNodePackageExportsImportOnlyForImport(t *testing.T) {
	dir := t.TempDir()
	imp := writeNodePackage(t, dir, map[string]string{
		"package.json":     `{"name": "p", "exports": {".": "./main.scss"}}`,
		"main.scss":        "a { b: 1; }",
		"main.import.scss": "a { b: 2; }",
	})

	want := "file://" + filepath.ToSlash(filepath.Join(dir, "node_modules", "p", "main.import.scss"))
	if got := canonicalizePkg(t, imp, true); got != want {
		t.Errorf("import: got %q, want %q", got, want)
	}
	want = "file://" + filepath.ToSlash(filepath.Join(dir, "node_modules", "p", "main.scss"))
	if got := canonicalizePkg(t, imp, false); got != want {
		t.Errorf("use: got %q, want %q", got, want)
	}
}

// Without an import-only sibling, @import falls back to the regular file.
func TestNodePackageExportsMissingImportOnlyFallsBack(t *testing.T) {
	dir := t.TempDir()
	imp := writeNodePackage(t, dir, map[string]string{
		"package.json": `{"name": "p", "exports": {".": "./main.scss"}}`,
		"main.scss":    "a { b: 1; }",
	})

	want := "file://" + filepath.ToSlash(filepath.Join(dir, "node_modules", "p", "main.scss"))
	if got := canonicalizePkg(t, imp, true); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// The sass manifest key also loads the import-only variant for @import.
func TestNodePackageSassKeyImportOnlyForImport(t *testing.T) {
	dir := t.TempDir()
	imp := writeNodePackage(t, dir, map[string]string{
		"package.json":         `{"name": "p", "sass": "./via-sass.scss"}`,
		"via-sass.scss":        "a { b: 1; }",
		"via-sass.import.scss": "a { b: 2; }",
	})

	want := "file://" + filepath.ToSlash(filepath.Join(dir, "node_modules", "p", "via-sass.import.scss"))
	if got := canonicalizePkg(t, imp, true); got != want {
		t.Errorf("import: got %q, want %q", got, want)
	}
	want = "file://" + filepath.ToSlash(filepath.Join(dir, "node_modules", "p", "via-sass.scss"))
	if got := canonicalizePkg(t, imp, false); got != want {
		t.Errorf("use: got %q, want %q", got, want)
	}
}

// The package-root index fallback honors the import context for the
// index / _index partial variants (importer/utils.dart _ifInImport, #2772).
func TestNodePackageIndexFallbackImportOnly(t *testing.T) {
	for _, tc := range []struct{ index, importOnly string }{
		{"index.scss", "index.import.scss"},
		{"_index.scss", "_index.import.scss"},
	} {
		dir := t.TempDir()
		imp := writeNodePackage(t, dir, map[string]string{
			"package.json": `{"name": "p"}`,
			tc.index:       "a { b: 1; }",
			tc.importOnly:  "a { b: 2; }",
		})

		want := "file://" + filepath.ToSlash(filepath.Join(dir, "node_modules", "p", tc.importOnly))
		if got := canonicalizePkg(t, imp, true); got != want {
			t.Errorf("%s import: got %q, want %q", tc.index, got, want)
		}
		want = "file://" + filepath.ToSlash(filepath.Join(dir, "node_modules", "p", tc.index))
		if got := canonicalizePkg(t, imp, false); got != want {
			t.Errorf("%s use: got %q, want %q", tc.index, got, want)
		}
	}
}

func TestGetMainExport(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{
			name:     "string export",
			input:    "./src/index.scss",
			expected: "./src/index.scss",
		},
		{
			name:     "OrderedJSON without dot keys returns it as conditions map",
			input:    orderedmap.NewOrderedJSONFromPairs(orderedmap.Pair[string, any]{Key: "sass", Val: "./sass/"}, orderedmap.Pair[string, any]{Key: "default", Val: "./default/"}),
			expected: orderedmap.NewOrderedJSONFromPairs(orderedmap.Pair[string, any]{Key: "sass", Val: "./sass/"}, orderedmap.Pair[string, any]{Key: "default", Val: "./default/"}),
		},
		{
			name:     "OrderedJSON with dot key '.' returns its value",
			input:    orderedmap.NewOrderedJSONFromPairs(orderedmap.Pair[string, any]{Key: ".", Val: "./fallback/"}, orderedmap.Pair[string, any]{Key: "sass", Val: "./sass/"}),
			expected: "./fallback/",
		},
		{
			name:     "OrderedJSON with other dot keys but no '.' returns nil",
			input:    orderedmap.NewOrderedJSONFromPairs(orderedmap.Pair[string, any]{Key: "./sass", Val: "./sass/"}),
			expected: nil,
		},
		{
			name:     "nil returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "non-matching type (int) returns nil",
			input:    42,
			expected: nil,
		},
		{
			// Dart: List<String> list never matches JSON-decoded List<dynamic>.
			// Go: JSON arrays are []any, so they fall through to default.
			name:     "array ([]any from JSON) falls through to nil",
			input:    []any{".", "./src/"},
			expected: nil,
		},
		{
			// Verify that []string also falls through to nil.
			name:     "[]string falls through to nil",
			input:    []string{".", "./src/"},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMainExport(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("getMainExport(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// TestGetMainExport_JSONRoundtrip simulates the real code path:
// package.json is unmarshaled into *orderedmap.OrderedJSON.
func TestGetMainExport_JSONRoundtrip(t *testing.T) {
	tests := []struct {
		name       string
		jsonExport string // JSON for the "exports" field value
		want       any
	}{
		{
			name:       "string exports",
			jsonExport: `"./index.scss"`,
			want:       "./index.scss",
		},
		{
			name:       "object exports with dot key",
			jsonExport: `{".": "./fallback/", "sass": "./sass/"}`,
			want:       "./fallback/",
		},
		{
			// Dart: List<String> list never matches List<dynamic>.
			// Go: arrays fall through to default → nil.
			name:       "array exports",
			jsonExport: `["."]`,
			want:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := fmt.Sprintf(`{"exports": %s}`, tt.jsonExport)

			// Simulate the actual unmarshal path: *orderedmap.OrderedJSON
			// (matches node_package_importer.go:106).
			manifest := &orderedmap.OrderedJSON{}
			if err := json.Unmarshal([]byte(raw), manifest); err != nil {
				t.Fatal(err)
			}
			exportsRaw, _ := manifest.Get("exports")
			t.Logf("*orderedmap.OrderedJSON exports type: %T", exportsRaw)

			got := getMainExport(exportsRaw)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getMainExport(%v) = %v, want %v", exportsRaw, got, tt.want)
			}
		})
	}
}
