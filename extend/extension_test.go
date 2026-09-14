package extend

import (
	"net/url"
	"testing"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/value"
)

func realSpan() sasscommon.FileSpan {
	raw := []byte(".a")
	fileURL, _ := url.Parse("test.scss")
	fs := sasscommon.NewFileSource(raw, fileURL)
	return fs.Span(0, len(raw))
}

func class(name string) value.SimpleSelector {
	return value.NewClassSelector(name, sasscommon.BogusSpan)
}

func compound(simples ...value.SimpleSelector) (*value.CompoundSelector, error) {
	return value.NewCompoundSelector(simples, sasscommon.BogusSpan)
}

func complexSelector(simples ...value.SimpleSelector) (*value.ComplexSelector, error) {
	comp, err := value.NewCompoundSelector(simples, sasscommon.BogusSpan)
	if err != nil {
		return nil, err
	}
	compComponent := value.NewComplexSelectorComponent(comp, nil, sasscommon.BogusSpan)
	return value.NewComplexSelector(nil, []*value.ComplexSelectorComponent{compComponent}, sasscommon.BogusSpan, false)
}

func complexSelectorWithCombinator(first, second value.SimpleSelector, comb value.Combinator) (*value.ComplexSelector, error) {
	c1, err := value.NewCompoundSelector([]value.SimpleSelector{first}, sasscommon.BogusSpan)
	if err != nil {
		return nil, err
	}
	c2, err := value.NewCompoundSelector([]value.SimpleSelector{second}, sasscommon.BogusSpan)
	if err != nil {
		return nil, err
	}
	combCss := sasscommon.CssValue[value.Combinator]{Value: comb}
	comp1 := value.NewComplexSelectorComponent(c1, nil, sasscommon.BogusSpan)
	comp2 := value.NewComplexSelectorComponent(c2, []sasscommon.CssValue[value.Combinator]{combCss}, sasscommon.BogusSpan)
	return value.NewComplexSelector(nil, []*value.ComplexSelectorComponent{comp1, comp2}, sasscommon.BogusSpan, false)
}

func selectorList(complexes ...*value.ComplexSelector) (*value.SelectorList, error) {
	return value.NewSelectorList(complexes, sasscommon.BogusSpan)
}

func extendRule(optional bool) *value.ExtendRule {
	interp := value.NewInterpolationPlain(".foo", sasscommon.BogusSpan)
	return value.NewExtendRule(interp, sasscommon.BogusSpan, optional)
}

func TestNewExtension(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	ext := NewExtension(extender, target, sasscommon.BogusSpan, nil, false)
	if ext == nil {
		t.Fatal("NewExtension returned nil")
	}
	if !ext.Extender().Selector.Equals(extender) {
		t.Error("extender selector mismatch")
	}
	if !ext.Target().Equals(target) {
		t.Error("target mismatch")
	}
	if ext.IsOptional() {
		t.Error("should not be optional")
	}
	_, err = ext.Span()
	if err != nil {
		t.Error("span() should not error")
	}
}

func TestNewExtensionOptional(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	ext := NewExtension(extender, target, sasscommon.BogusSpan, nil, true)
	if !ext.IsOptional() {
		t.Error("should be optional")
	}
}

func TestExtensionMediaContext(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	mediaContext := []*value.CssMediaQuery{
		value.NewCssMediaQueryType(&[]string{"screen"}[0], nil, nil),
	}
	ext := NewExtension(extender, target, sasscommon.BogusSpan, mediaContext, false)
	if ext.MediaContext() == nil {
		t.Error("mediaContext should not be nil")
	}
	if len(ext.MediaContext()) != 1 {
		t.Errorf("mediaContext len = %d, want 1", len(ext.MediaContext()))
	}
}

func TestExtensionWithExtender(t *testing.T) {
	extender1, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	ext := NewExtension(extender1, target, sasscommon.BogusSpan, nil, false)

	newExtender, err := complexSelector(class("c"))
	if err != nil {
		t.Fatal(err)
	}
	newExt := ext.WithExtender(newExtender)
	if !newExt.Extender().Selector.Equals(newExtender) {
		t.Error("new extender selector mismatch")
	}
	if !newExt.Target().Equals(target) {
		t.Error("target should be preserved")
	}
}

func TestExtensionString(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	ext := NewExtension(extender, target, sasscommon.BogusSpan, nil, false)
	s, err := ext.String()
	if err != nil {
		t.Fatal(err)
	}
	if len(s) == 0 {
		t.Error("string should not be empty")
	}
}

func TestExtensionStringOptional(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	ext := NewExtension(extender, target, sasscommon.BogusSpan, nil, true)
	s, err := ext.String()
	if err != nil {
		t.Fatal(err)
	}
	if len(s) == 0 {
		t.Error("string should not be empty")
	}
}

func TestExtensionIsMerged(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	ext := NewExtension(extender, target, sasscommon.BogusSpan, nil, false)
	if ext.IsMerged() {
		t.Error("base extension should not be merged")
	}
}

func TestExtenderSpecificityDefault(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	ext := NewExtender(extender, nil, false)
	if ext.Specificity != extender.Specificity() {
		t.Errorf("specificity = %d, want %d", ext.Specificity, extender.Specificity())
	}
}

func TestExtenderSpecificityExplicit(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	spec := 500
	ext := NewExtender(extender, &spec, false)
	if ext.Specificity != 500 {
		t.Errorf("specificity = %d, want 500", ext.Specificity)
	}
}

func TestExtenderIsOriginal(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	ext := NewExtender(extender, nil, true)
	if !ext.IsOriginal {
		t.Error("should be original")
	}
}

func TestAssertCompatibleMediaContextBothNil(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	ext := NewExtension(extender, target, sasscommon.BogusSpan, nil, false)
	err = ext.Extender().AssertCompatibleMediaContext(nil)
	if err != nil {
		t.Errorf("should not error: %v", err)
	}
}

func TestAssertCompatibleMediaContextExtenderHasNoMediaContext(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	ext := NewExtension(extender, target, sasscommon.BogusSpan, nil, false)
	mediaContext := []*value.CssMediaQuery{
		value.NewCssMediaQueryType(&[]string{"screen"}[0], nil, nil),
	}
	err = ext.Extender().AssertCompatibleMediaContext(mediaContext)
	if err != nil {
		t.Errorf("should not error when extender has no media context: %v", err)
	}
}

func TestAssertCompatibleMediaContextMatching(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	mediaContext := []*value.CssMediaQuery{
		value.NewCssMediaQueryType(&[]string{"screen"}[0], nil, nil),
	}
	ext := NewExtension(extender, target, sasscommon.BogusSpan, mediaContext, false)
	err = ext.Extender().AssertCompatibleMediaContext(mediaContext)
	if err != nil {
		t.Errorf("should not error on matching contexts: %v", err)
	}
}

func TestAssertCompatibleMediaContextConflicting(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	mediaContext1 := []*value.CssMediaQuery{
		value.NewCssMediaQueryType(&[]string{"screen"}[0], nil, nil),
	}
	mediaContext2 := []*value.CssMediaQuery{
		value.NewCssMediaQueryType(&[]string{"print"}[0], nil, nil),
	}
	ext := NewExtension(extender, target, sasscommon.BogusSpan, mediaContext1, false)
	err = ext.Extender().AssertCompatibleMediaContext(mediaContext2)
	if err == nil {
		t.Error("should error on conflicting media contexts")
	}
}

func TestMediaQueriesEqualBothNil(t *testing.T) {
	if !mediaQueriesEqual(nil, nil) {
		t.Error("two nil slices should be equal")
	}
}

func TestMediaQueriesEqualOneNilOneNot(t *testing.T) {
	mq := []*value.CssMediaQuery{
		value.NewCssMediaQueryType(&[]string{"screen"}[0], nil, nil),
	}
	if mediaQueriesEqual(mq, nil) {
		t.Error("nil vs non-nil should not be equal")
	}
	if mediaQueriesEqual(nil, mq) {
		t.Error("non-nil vs nil should not be equal")
	}
}

func TestMediaQueriesEqualSame(t *testing.T) {
	mq1 := []*value.CssMediaQuery{
		value.NewCssMediaQueryType(&[]string{"screen"}[0], nil, nil),
	}
	mq2 := []*value.CssMediaQuery{
		value.NewCssMediaQueryType(&[]string{"screen"}[0], nil, nil),
	}
	if !mediaQueriesEqual(mq1, mq2) {
		t.Error("equal media queries should be equal")
	}
}

func TestMediaQueriesEqualDifferent(t *testing.T) {
	mq1 := []*value.CssMediaQuery{
		value.NewCssMediaQueryType(&[]string{"screen"}[0], nil, nil),
	}
	mq2 := []*value.CssMediaQuery{
		value.NewCssMediaQueryType(&[]string{"print"}[0], nil, nil),
	}
	if mediaQueriesEqual(mq1, mq2) {
		t.Error("different media queries should not be equal")
	}
}

func TestMediaQueriesEqualDifferentLength(t *testing.T) {
	mq1 := []*value.CssMediaQuery{
		value.NewCssMediaQueryType(&[]string{"screen"}[0], nil, nil),
	}
	mq2 := []*value.CssMediaQuery{
		value.NewCssMediaQueryType(&[]string{"screen"}[0], nil, nil),
		value.NewCssMediaQueryType(&[]string{"print"}[0], nil, nil),
	}
	if mediaQueriesEqual(mq1, mq2) {
		t.Error("different length queries should not be equal")
	}
}

func TestExtenderString(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	ext := NewExtender(extender, nil, false)
	s, err := ext.String()
	if err != nil {
		t.Fatal(err)
	}
	if len(s) == 0 {
		t.Error("string should not be empty")
	}
}
