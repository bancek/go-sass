package eval

import "testing"

func TestImporterResultSourceMapURLDataFallback(t *testing.T) {
	r, err := NewImporterResult("a b", SyntaxSCSS, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.SourceMapURL().String(); got != "data:;charset=utf-8,a%20b" {
		t.Errorf("SourceMapURL = %q, want %q", got, "data:;charset=utf-8,a%20b")
	}
}
