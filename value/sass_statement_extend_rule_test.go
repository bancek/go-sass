package value

import (
	"strings"
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestNewExtendRule(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@extend .foo;"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 13)
	sel := NewInterpolationPlain(".foo", sasscommon.NewSimpleFileSpan(fs, 8, 12))
	r := NewExtendRule(sel, span, false)

	if r.Selector != sel {
		t.Error("selector field should match constructor arg")
	}
	if r.IsOptional {
		t.Error("is_optional should be false")
	}
}

func TestNewExtendRuleOptional(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@extend .foo !optional;"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 22)
	sel := NewInterpolationPlain(".foo", sasscommon.NewSimpleFileSpan(fs, 8, 12))
	r := NewExtendRule(sel, span, true)

	if !r.IsOptional {
		t.Error("is_optional should be true")
	}
}

func TestExtendRuleSpan(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@extend .foo;"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 13)
	sel := NewInterpolationPlain(".foo", sasscommon.NewSimpleFileSpan(fs, 8, 12))
	r := NewExtendRule(sel, span, false)

	got, err := r.Span()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("Span() returned nil")
	}
}

func TestExtendRuleString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@extend .foo;"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 13)
	sel := NewInterpolationPlain(".foo", sasscommon.NewSimpleFileSpan(fs, 8, 12))
	r := NewExtendRule(sel, span, false)

	got, err := r.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "@extend ") {
		t.Errorf("String() = %q, want prefix '@extend '", got)
	}
	if !strings.HasSuffix(got, ";") {
		t.Errorf("String() = %q, want suffix ';'", got)
	}
}

func TestExtendRuleStringOptional(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@extend .foo !optional;"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 22)
	sel := NewInterpolationPlain(".foo", sasscommon.NewSimpleFileSpan(fs, 8, 12))
	r := NewExtendRule(sel, span, true)

	got, err := r.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "!optional") {
		t.Errorf("String() = %q, want '!optional' for optional extend", got)
	}
}

func TestExtendRuleMarkerInterfaces(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("@extend .foo;"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 13)
	sel := NewInterpolationPlain(".foo", sasscommon.NewSimpleFileSpan(fs, 8, 12))
	r := NewExtendRule(sel, span, false)

	// marker methods should not panic
	r.IsStatement()
	r.IsSassNode()
	r.IsAstNode()
}
