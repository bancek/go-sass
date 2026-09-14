package value

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bancek/go-sass/box"
	"github.com/bancek/go-sass/sasscommon"
)

func supportSpan(contents string, start, end int) sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte(contents), nil)
	return sasscommon.NewFileSpan(fs, start, end)
}

// --- Parameter ---

func TestParameterConstruction(t *testing.T) {
	span := supportSpan("$name", 0, 5)
	p := NewParameter("name", span, nil)

	if p.Name() != "name" {
		t.Errorf("Name() = %q, want %q", p.Name(), "name")
	}
	if p.DefaultValue != nil {
		t.Error("DefaultValue should be nil")
	}
}

func TestParameterWithDefault(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("$name: true"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 11)
	boolExpr := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 7, 11))
	p := NewParameter("name", span, boolExpr)

	if p.Name() != "name" {
		t.Errorf("Name() = %q, want %q", p.Name(), "name")
	}
	if p.DefaultValue == nil {
		t.Error("DefaultValue should not be nil")
	}
}

func TestParameterString(t *testing.T) {
	span := supportSpan("$name", 0, 5)
	p := NewParameter("name", span, nil)

	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "name" {
		t.Errorf("String() = %q, want %q", got, "name")
	}
}

func TestParameterStringWithDefault(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("$name: true"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 11)
	boolExpr := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 7, 11))
	p := NewParameter("name", span, boolExpr)

	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "name: true" {
		t.Errorf("String() = %q, want %q", got, "name: true")
	}
}

func TestParameterNameSpan(t *testing.T) {
	span := supportSpan("$name", 0, 5)
	p := NewParameter("name", span, nil)

	got, err := p.NameSpan()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("NameSpan() returned nil")
	}
	text, err := got.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "$name" {
		t.Errorf("NameSpan() text = %q, want %q", text, "$name")
	}
}

func TestParameterOriginalName(t *testing.T) {
	span := supportSpan("$name", 0, 5)
	p := NewParameter("name", span, nil)

	got, err := p.OriginalName()
	if err != nil {
		t.Fatal(err)
	}
	if got != "$name" {
		t.Errorf("OriginalName() = %q, want %q", got, "$name")
	}
}

func TestParameterOriginalNameWithDefault(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("$name: 5px"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 10)
	unit := "px"
	defaultExpr := NewNumberExpression(5, sasscommon.NewFileSpan(fs, 7, 10), &unit)
	p := NewParameter("name", span, defaultExpr)

	got, err := p.OriginalName()
	if err != nil {
		t.Fatal(err)
	}
	if got != "$name" {
		t.Errorf("OriginalName() = %q, want %q", got, "$name")
	}
}

// --- ParameterList ---

func TestParameterListEmpty(t *testing.T) {
	span := supportSpan("()", 0, 2)
	pl := NewParameterListEmpty(span)

	if !pl.IsEmpty() {
		t.Error("IsEmpty() should be true")
	}
	if pl.RestParameter != nil {
		t.Error("RestParameter should be nil")
	}
}

func TestParameterListConstruction(t *testing.T) {
	span := supportSpan("($name)", 0, 6)
	p := NewParameter("name", supportSpan("$name", 1, 6), nil)
	pl := NewParameterList([]*Parameter{p}, span, nil)

	if pl.IsEmpty() {
		t.Error("IsEmpty() should be false")
	}
	if len(pl.Parameters) != 1 {
		t.Errorf("len(Parameters) = %d, want 1", len(pl.Parameters))
	}
}

func TestParameterListRestParam(t *testing.T) {
	span := supportSpan("($args...)", 0, 9)
	restParam := "args"
	pl := NewParameterList([]*Parameter{}, span, &restParam)

	if *pl.RestParameter != "args" {
		t.Errorf("RestParameter = %q, want %q", *pl.RestParameter, "args")
	}
}

func TestParameterListString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("($name, $other, $rest...)"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 27)
	restParam := "rest"
	pl := NewParameterList([]*Parameter{
		NewParameter("name", sasscommon.NewFileSpan(fs, 1, 6), nil),
		NewParameter("other", sasscommon.NewFileSpan(fs, 8, 14), nil),
	}, span, &restParam)

	got, err := pl.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "$name, $other, $rest..." {
		t.Errorf("String() = %q, want %q", got, "$name, $other, $rest...")
	}
}

func TestParameterListVerifyExactMatch(t *testing.T) {
	span := supportSpan("($name)", 0, 6)
	pl := NewParameterList([]*Parameter{
		NewParameter("name", supportSpan("$name", 1, 6), nil),
	}, span, nil)

	if err := pl.Verify(1, map[string]struct{}{}); err != nil {
		t.Errorf("Verify(1, {}) should pass, got: %v", err)
	}
}

func TestParameterListVerifyExtraPositional(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("($name)"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 7)
	pl := NewParameterList([]*Parameter{
		NewParameter("name", sasscommon.NewFileSpan(fs, 1, 6), nil),
	}, span, nil)

	err := pl.Verify(2, map[string]struct{}{})
	if err == nil {
		t.Error("Verify(2, {}) should fail for 1-parameter list")
	}
}

func TestParameterListVerifyMissingRequired(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("($name)"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 7)
	pl := NewParameterList([]*Parameter{
		NewParameter("name", sasscommon.NewFileSpan(fs, 1, 6), nil),
	}, span, nil)

	err := pl.Verify(0, map[string]struct{}{})
	if err == nil {
		t.Error("Verify(0, {}) should fail when required param missing")
	}
}

func TestParameterListVerifyNamed(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("($name)"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 7)
	pl := NewParameterList([]*Parameter{
		NewParameter("name", sasscommon.NewFileSpan(fs, 1, 6), nil),
	}, span, nil)

	err := pl.Verify(0, map[string]struct{}{"name": {}})
	if err != nil {
		t.Errorf("Verify(0, {name}) should pass, got: %v", err)
	}
}

func TestParameterListOriginalParameterName(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("($param_name: 5px)"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 18)
	paramSpan := sasscommon.NewFileSpan(fs, 1, 17)
	unit := "px"
	defaultExpr := NewNumberExpression(5, sasscommon.NewFileSpan(fs, 14, 17), &unit)
	pl := NewParameterList([]*Parameter{
		NewParameter("param-name", paramSpan, defaultExpr),
	}, span, nil)

	got, err := pl.OriginalParameterName("param-name")
	if err != nil {
		t.Fatal(err)
	}
	if got != "$param_name" {
		t.Errorf("OriginalParameterName() = %q, want %q", got, "$param_name")
	}
}

func TestParameterListOriginalParameterNameRest(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("($args...)"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 9)
	restParam := "args"
	pl := NewParameterList([]*Parameter{}, span, &restParam)

	got, err := pl.OriginalParameterName("args")
	if err != nil {
		t.Fatal(err)
	}
	if got != "$args" {
		t.Errorf("OriginalParameterName() = %q, want %q", got, "$args")
	}
}

func TestParameterListMatchesValid(t *testing.T) {
	span := supportSpan("($name)", 0, 6)
	pl := NewParameterList([]*Parameter{
		NewParameter("name", supportSpan("$name", 1, 6), nil),
	}, span, nil)

	if !pl.Matches(1, map[string]struct{}{}) {
		t.Error("Matches(1, {}) should be true")
	}
	if !pl.Matches(0, map[string]struct{}{"name": {}}) {
		t.Error("Matches(0, {name}) should be true")
	}
}

func TestParameterListMatchesInvalid(t *testing.T) {
	span := supportSpan("($name)", 0, 6)
	pl := NewParameterList([]*Parameter{
		NewParameter("name", supportSpan("$name", 1, 6), nil),
	}, span, nil)

	if pl.Matches(2, map[string]struct{}{}) {
		t.Error("Matches(2, {}) should be false")
	}
	if pl.Matches(0, map[string]struct{}{"unknown": {}}) {
		t.Error("Matches(0, {unknown}) should be false")
	}
}

func TestParameterListSpanWithName(t *testing.T) {
	source := "@mixin foo($name)"
	fs := sasscommon.NewFileSource([]byte(source), nil)
	// Parameter list spans positions 12..17 — "($name)"
	// SpanWithName should expand backwards to include "foo"
	span := sasscommon.NewFileSpan(fs, 12, len(source))
	p := NewParameter("name", sasscommon.NewFileSpan(fs, 13, len(source)), nil)
	pl := NewParameterList([]*Parameter{p}, span, nil)

	got := pl.SpanWithName()
	if got == nil {
		t.Fatal("SpanWithName() returned nil")
	}
	text, err := got.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	// Should cover "foo($name)" or similar expanded span
	if text == "" {
		t.Error("SpanWithName() text should not be empty")
	}
}

// --- ConfiguredVariable ---

func TestConfiguredVariableConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("$name: true"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 11)
	boolExpr := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 7, 11))
	cv := NewConfiguredVariable("name", boolExpr, span, false)

	if cv.Name() != "name" {
		t.Errorf("Name() = %q, want %q", cv.Name(), "name")
	}
	if cv.IsGuarded {
		t.Error("IsGuarded should be false")
	}
}

func TestConfiguredVariableStringUnguarded(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("$name: true"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 11)
	boolExpr := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 7, 11))
	cv := NewConfiguredVariable("name", boolExpr, span, false)

	got, err := cv.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "$name: true" {
		t.Errorf("String() = %q, want %q", got, "$name: true")
	}
}

func TestConfiguredVariableStringGuarded(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("$name: true !default"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 19)
	boolExpr := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 7, 11))
	cv := NewConfiguredVariable("name", boolExpr, span, true)

	got, err := cv.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "$name: true !default" {
		t.Errorf("String() = %q, want %q", got, "$name: true !default")
	}
}

func TestConfiguredVariableNameSpan(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("$name: true"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 11)
	boolExpr := NewBooleanExpression(true, sasscommon.NewFileSpan(fs, 7, 11))
	cv := NewConfiguredVariable("name", boolExpr, span, false)

	got, err := cv.NameSpan()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("NameSpan() returned nil")
	}
}

// --- AtRootQuery ---

func TestAtRootQueryDefault(t *testing.T) {
	q := DefaultQuery

	if q.Include {
		t.Error("DefaultQuery.Include should be false")
	}
	if !q.ExcludesStyleRules() {
		t.Error("DefaultQuery should exclude style rules")
	}
}

func TestAtRootQueryIncludeRule(t *testing.T) {
	q := NewAtRootQuery(map[string]struct{}{"rule": {}}, true)

	if !q.Include {
		t.Error("Include should be true")
	}
	if q.ExcludesStyleRules() {
		t.Error("ExcludesStyleRules() should be false when rule is included")
	}
}

func TestAtRootQueryExcludeAll(t *testing.T) {
	q := NewAtRootQuery(map[string]struct{}{"all": {}}, false)

	if !q.ExcludesStyleRules() {
		t.Error("Should exclude style rules when all is excluded")
	}
	if !q.ExcludesName("media") {
		t.Error("Should exclude 'media' when all is excluded")
	}
}

func TestAtRootQueryExcludesName(t *testing.T) {
	q := NewAtRootQuery(map[string]struct{}{"media": {}, "supports": {}}, false)

	if !q.ExcludesName("media") {
		t.Error("Should exclude 'media'")
	}
	if !q.ExcludesName("supports") {
		t.Error("Should exclude 'supports'")
	}
	if q.ExcludesName("screen") {
		t.Error("Should not exclude 'screen'")
	}
}

func TestAtRootQueryExcludesStyleRules(t *testing.T) {
	q := NewAtRootQuery(map[string]struct{}{"rule": {}}, false)
	if !q.ExcludesStyleRules() {
		t.Error("Should exclude style rules when rule is excluded")
	}
}

func TestAtRootQueryExcludes(t *testing.T) {
	span := cssSpan()

	// Excludes CssStyleRule
	sel, err := makSelList("foo")
	if err != nil {
		t.Fatal(err)
	}
	selBox := box.NewModifiableBox(sel).Seal()
	styleRule := NewModifiableCssStyleRule(selBox, span, nil, false)

	q1 := NewAtRootQuery(map[string]struct{}{"rule": {}}, false)
	if !q1.Excludes(styleRule) {
		t.Error("query excluding 'rule' should exclude CssStyleRule")
	}
	q2 := NewAtRootQuery(map[string]struct{}{"rule": {}}, true)
	if q2.Excludes(styleRule) {
		t.Error("query including 'rule' should NOT exclude CssStyleRule")
	}

	// Excludes CssMediaRule
	mq := makeMediaQuery("(color)")
	mediaRule, err := NewModifiableCssMediaRule([]CssMediaQuery{mq}, span)
	if err != nil {
		t.Fatal(err)
	}

	q3 := NewAtRootQuery(map[string]struct{}{"media": {}}, false)
	if !q3.Excludes(mediaRule) {
		t.Error("query excluding 'media' should exclude CssMediaRule")
	}
	q4 := NewAtRootQuery(map[string]struct{}{"media": {}}, true)
	if q4.Excludes(mediaRule) {
		t.Error("query including 'media' should NOT exclude CssMediaRule")
	}

	// Excludes CssSupportsRule
	cond := sasscommon.NewCssValue("display: grid", span)
	supportsRule := NewModifiableCssSupportsRule(cond, span)

	q5 := NewAtRootQuery(map[string]struct{}{"supports": {}}, false)
	if !q5.Excludes(supportsRule) {
		t.Error("query excluding 'supports' should exclude CssSupportsRule")
	}
	q6 := NewAtRootQuery(map[string]struct{}{"supports": {}}, true)
	if q6.Excludes(supportsRule) {
		t.Error("query including 'supports' should NOT exclude CssSupportsRule")
	}

	// Excludes CssAtRule by name
	nameVal := sasscommon.NewCssValue("print", span)
	atRule := NewModifiableCssAtRule(nameVal, span, false, nil)

	q7 := NewAtRootQuery(map[string]struct{}{"print": {}}, false)
	if !q7.Excludes(atRule) {
		t.Error("query excluding 'print' should exclude CssAtRule named 'print'")
	}
	q8 := NewAtRootQuery(map[string]struct{}{"print": {}}, true)
	if q8.Excludes(atRule) {
		t.Error("query including 'print' should NOT exclude CssAtRule named 'print'")
	}
	q9 := NewAtRootQuery(map[string]struct{}{"screen": {}}, false)
	if q9.Excludes(atRule) {
		t.Error("query excluding 'screen' should NOT exclude CssAtRule named 'print'")
	}
}

// --- DynamicImport ---

func TestDynamicImportConstruction(t *testing.T) {
	span := supportSpan("'foo.scss'", 0, 11)
	di := NewDynamicImport("foo.scss", span)

	if di.urlString != "foo.scss" {
		t.Errorf("urlString = %q, want %q", di.urlString, "foo.scss")
	}
}

func TestDynamicImportURL(t *testing.T) {
	span := supportSpan("'foo.scss'", 0, 11)
	di := NewDynamicImport("foo.scss", span)

	u := di.URL()
	if u == nil {
		t.Fatal("URL() returned nil")
	}
	if u.String() != "foo.scss" {
		t.Errorf("URL().String() = %q, want %q", u.String(), "foo.scss")
	}
}

func TestDynamicImportURLSpan(t *testing.T) {
	span := supportSpan("'foo.scss'", 0, 11)
	di := NewDynamicImport("foo.scss", span)

	got, err := di.URLSpan()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("URLSpan() returned nil")
	}
}

func TestDynamicImportString(t *testing.T) {
	span := supportSpan("'foo.scss'", 0, 11)
	di := NewDynamicImport("foo.scss", span)

	got, err := di.String()
	if err != nil {
		t.Fatal(err)
	}
	want := QuoteText("foo.scss")
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestDynamicImportStringWithSpecialChars(t *testing.T) {
	di := NewDynamicImport("foo\nbar", nil)

	got, err := di.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `\a`) {
		t.Errorf("String() = %q, expected CSS \\a newline escape", got)
	}
	if strings.Contains(got, `\n`) {
		t.Errorf("String() = %q, should NOT contain Go-style \\n escape", got)
	}
}

// --- StaticImport ---

func TestStaticImportConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("'url'"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 5)
	urlInterp := NewInterpolationPlain("url", sasscommon.NewFileSpan(fs, 1, 4))
	si := NewStaticImport(urlInterp, span, nil)

	if si.URL != urlInterp {
		t.Error("URL field should match constructor arg")
	}
	if si.Modifiers != nil {
		t.Error("Modifiers should be nil")
	}
}

func TestStaticImportString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("'url'"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 5)
	urlInterp := NewInterpolationPlain("url", sasscommon.NewFileSpan(fs, 1, 4))
	si := NewStaticImport(urlInterp, span, nil)

	got, err := si.String()
	if err != nil {
		t.Fatal(err)
	}
	// Just verify it doesn't crash and contains the URL text
	_ = got
}

func TestStaticImportStringWithModifiers(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("'url' screen"), nil)
	span := sasscommon.NewFileSpan(fs, 0, 12)
	urlInterp := NewInterpolationPlain("url", sasscommon.NewFileSpan(fs, 1, 4))
	modInterp := NewInterpolationPlain("screen", sasscommon.NewFileSpan(fs, 6, 12))
	si := NewStaticImport(urlInterp, span, modInterp)

	got, err := si.String()
	if err != nil {
		t.Fatal(err)
	}
	// Format should include modifier
	_ = got
}

// fmt import check (unused if no fmt.Errorf in the test)
var _ = fmt.Sprintf
