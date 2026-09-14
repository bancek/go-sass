package value

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bancek/go-sass/box"
	"github.com/bancek/go-sass/sasscommon"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func cssSpan() sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte("css test"), nil)
	return sasscommon.NewFileSpan(fs, 0, 8)
}

func cssStrValue(s string) sasscommon.CssValue[string] {
	return sasscommon.NewCssValue(s, cssSpan())
}

func makSelList(name string) (*SelectorList, error) {
	class := NewClassSelector(name, sasscommon.BogusSpan)
	compound, err := NewCompoundSelector([]SimpleSelector{class}, sasscommon.BogusSpan)
	if err != nil {
		return nil, err
	}
	comp := NewComplexSelectorComponent(compound, nil, sasscommon.BogusSpan)
	cs, err := NewComplexSelector(nil, []*ComplexSelectorComponent{comp}, sasscommon.BogusSpan, false)
	if err != nil {
		return nil, err
	}
	return NewSelectorList([]*ComplexSelector{cs}, sasscommon.BogusSpan)
}

// mockVisitor implements CssVisitor[struct{}] for accept dispatch tests.
// It implements both the immutable and modifiable CSS visitor interfaces.
type mockVisitor struct {
	visited string
}

func (v *mockVisitor) VisitCssAtRule(node CssAtRule) (struct{}, error) {
	v.visited = "CssAtRule"
	return struct{}{}, nil
}

func (v *mockVisitor) VisitCssComment(node CssComment) (struct{}, error) {
	v.visited = "CssComment"
	return struct{}{}, nil
}

func (v *mockVisitor) VisitCssDeclaration(node CssDeclaration) (struct{}, error) {
	v.visited = "CssDeclaration"
	return struct{}{}, nil
}

func (v *mockVisitor) VisitCssImport(node CssImport) (struct{}, error) {
	v.visited = "CssImport"
	return struct{}{}, nil
}

func (v *mockVisitor) VisitCssKeyframeBlock(node CssKeyframeBlock) (struct{}, error) {
	v.visited = "CssKeyframeBlock"
	return struct{}{}, nil
}

func (v *mockVisitor) VisitCssMediaRule(node CssMediaRule) (struct{}, error) {
	v.visited = "CssMediaRule"
	return struct{}{}, nil
}

func (v *mockVisitor) VisitCssStyleRule(node CssStyleRule) (struct{}, error) {
	v.visited = "CssStyleRule"
	return struct{}{}, nil
}

func (v *mockVisitor) VisitCssStylesheet(node *CssStylesheet) (struct{}, error) {
	v.visited = "CssStylesheet"
	return struct{}{}, nil
}

func (v *mockVisitor) VisitCssSupportsRule(node CssSupportsRule) (struct{}, error) {
	v.visited = "CssSupportsRule"
	return struct{}{}, nil
}

type mockModifiableVisitor struct {
	visited string
}

func (v *mockModifiableVisitor) VisitModifiableCssAtRule(node *ModifiableCssAtRule) (struct{}, error) {
	v.visited = "ModifiableCssAtRule"
	return struct{}{}, nil
}

func (v *mockModifiableVisitor) VisitModifiableCssComment(node *ModifiableCssComment) (struct{}, error) {
	v.visited = "ModifiableCssComment"
	return struct{}{}, nil
}

func (v *mockModifiableVisitor) VisitModifiableCssDeclaration(node *ModifiableCssDeclaration) (struct{}, error) {
	v.visited = "ModifiableCssDeclaration"
	return struct{}{}, nil
}

func (v *mockModifiableVisitor) VisitModifiableCssImport(node *ModifiableCssImport) (struct{}, error) {
	v.visited = "ModifiableCssImport"
	return struct{}{}, nil
}

func (v *mockModifiableVisitor) VisitModifiableCssKeyframeBlock(node *ModifiableCssKeyframeBlock) (struct{}, error) {
	v.visited = "ModifiableCssKeyframeBlock"
	return struct{}{}, nil
}

func (v *mockModifiableVisitor) VisitModifiableCssMediaRule(node *ModifiableCssMediaRule) (struct{}, error) {
	v.visited = "ModifiableCssMediaRule"
	return struct{}{}, nil
}

func (v *mockModifiableVisitor) VisitModifiableCssStyleRule(node *ModifiableCssStyleRule) (struct{}, error) {
	v.visited = "ModifiableCssStyleRule"
	return struct{}{}, nil
}

func (v *mockModifiableVisitor) VisitModifiableCssStylesheet(node *ModifiableCssStylesheet) (struct{}, error) {
	v.visited = "ModifiableCssStylesheet"
	return struct{}{}, nil
}

func (v *mockModifiableVisitor) VisitModifiableCssSupportsRule(node *ModifiableCssSupportsRule) (struct{}, error) {
	v.visited = "ModifiableCssSupportsRule"
	return struct{}{}, nil
}

// ---------------------------------------------------------------------------
// CssStylesheet tests
// ---------------------------------------------------------------------------

func TestCssStylesheetConstruction(t *testing.T) {
	span := cssSpan()
	ss := NewCssStylesheet(nil, span)

	s, err := ss.Span()
	if err != nil {
		t.Fatal(err)
	}
	if s != span {
		t.Error("span mismatch")
	}
	if len(ss.Children()) != 0 {
		t.Errorf("expected 0 children, got %d", len(ss.Children()))
	}
}

func TestCssStylesheetEmpty(t *testing.T) {
	ss := NewCssStylesheetEmpty(nil)
	if len(ss.Children()) != 0 {
		t.Errorf("expected 0 children, got %d", len(ss.Children()))
	}
	if ss.IsChildless() {
		t.Error("stylesheet should not be childless")
	}
}

func TestCssStylesheetIsInvisible(t *testing.T) {
	span := cssSpan()
	ss := NewCssStylesheet(nil, span)
	if !ss.IsInvisible() {
		t.Error("empty stylesheet should be invisible")
	}
	ss = NewCssStylesheet([]CssNode{}, span)
	if !ss.IsInvisible() {
		t.Error("stylesheet with empty children should be invisible")
	}
}

func TestCssStylesheetAcceptVoid(t *testing.T) {
	span := cssSpan()
	ss := NewCssStylesheet(nil, span)
	var v mockVisitor
	_, err := ss.AcceptVoid(&v)
	if err != nil {
		t.Fatal(err)
	}
	if v.visited != "CssStylesheet" {
		t.Errorf("expected CssStylesheet, got %s", v.visited)
	}
}

// ---------------------------------------------------------------------------
// ModifiableCssStylesheet tests
// ---------------------------------------------------------------------------

func TestModifiableCssStylesheetConstruction(t *testing.T) {
	span := cssSpan()
	ss := NewModifiableCssStylesheet(span)

	s, err := ss.Span()
	if err != nil {
		t.Fatal(err)
	}
	if s != span {
		t.Error("span mismatch")
	}
	if len(ss.Children()) != 0 {
		t.Errorf("expected 0 children, got %d", len(ss.Children()))
	}
	if ss.Parent() != nil {
		t.Error("root stylesheet should have nil parent")
	}
}

func TestModifiableCssStylesheetAddChild(t *testing.T) {
	span := cssSpan()
	ss := NewModifiableCssStylesheet(span)
	child := NewModifiableCssComment("/* test */", span)

	err := ss.AddChild(child)
	if err != nil {
		t.Fatal(err)
	}
	if len(ss.Children()) != 1 {
		t.Errorf("expected 1 child, got %d", len(ss.Children()))
	}
	if child.Parent() != ss {
		t.Error("child parent not set")
	}
}

func TestModifiableCssStylesheetCopyWithoutChildren(t *testing.T) {
	span := cssSpan()
	ss := NewModifiableCssStylesheet(span)
	child := NewModifiableCssComment("/* test */", span)
	ss.AddChild(child)

	copy, err := ss.CopyWithoutChildren()
	if err != nil {
		t.Fatal(err)
	}
	if len(copy.Children()) != 0 {
		t.Errorf("copy should have 0 children, got %d", len(copy.Children()))
	}
	s, _ := copy.Span()
	if s != span {
		t.Error("copy span mismatch")
	}
}

func TestModifiableCssStylesheetEqualsIgnoringChildren(t *testing.T) {
	span := cssSpan()
	a := NewModifiableCssStylesheet(span)
	a.AddChild(NewModifiableCssComment("/* test */", span))
	b := NewModifiableCssStylesheet(span)
	b.AddChild(NewModifiableCssComment("/* different */", span))

	eq, err := a.EqualsIgnoringChildren(b)
	if err != nil {
		t.Fatal(err)
	}
	if !eq {
		t.Error("stylesheets with different children should be equal ignoring children")
	}
}

func TestModifiableCssStylesheetClearChildren(t *testing.T) {
	span := cssSpan()
	ss := NewModifiableCssStylesheet(span)
	child := NewModifiableCssComment("/* test */", span)
	ss.AddChild(child)

	ss.ClearChildren()
	if len(ss.Children()) != 0 {
		t.Error("children should be cleared")
	}
	if child.Parent() != nil {
		t.Error("cleared child parent should be nil")
	}
}

func TestModifiableCssStylesheetAcceptModifiable(t *testing.T) {
	span := cssSpan()
	ss := NewModifiableCssStylesheet(span)
	var v mockModifiableVisitor
	err := ss.AcceptModifiableVoid(&v)
	if err != nil {
		t.Fatal(err)
	}
	if v.visited != "ModifiableCssStylesheet" {
		t.Errorf("expected ModifiableCssStylesheet, got %s", v.visited)
	}
}

func TestModifiableCssStylesheetAcceptImmutable(t *testing.T) {
	span := cssSpan()
	ss := NewModifiableCssStylesheet(span)
	var v mockVisitor
	_, err := ss.AcceptVoid(&v)
	if err != nil {
		t.Fatal(err)
	}
	if v.visited != "CssStylesheet" {
		t.Errorf("expected CssStylesheet, got %s", v.visited)
	}
}

// ---------------------------------------------------------------------------
// CssComment / ModifiableCssComment tests
// ---------------------------------------------------------------------------

func TestModifiableCssCommentConstruction(t *testing.T) {
	span := cssSpan()
	c := NewModifiableCssComment("/* hello */", span)

	if c.Text() != "/* hello */" {
		t.Errorf("Text() = %q, want %q", c.Text(), "/* hello */")
	}
	s, err := c.Span()
	if err != nil {
		t.Fatal(err)
	}
	if s != span {
		t.Error("span mismatch")
	}
}

func TestModifiableCssCommentIsPreserved(t *testing.T) {
	span := cssSpan()
	c := NewModifiableCssComment("/*! important */", span)
	if !c.IsPreserved() {
		t.Error("comment starting with /*! should be preserved")
	}

	c2 := NewModifiableCssComment("/* normal */", span)
	if c2.IsPreserved() {
		t.Error("comment starting with /* should not be preserved")
	}
}

func TestModifiableCssCommentIsInvisible(t *testing.T) {
	span := cssSpan()
	c := NewModifiableCssComment("/* test */", span)
	if c.IsInvisible() {
		t.Error("comment should not be invisible (visible in normal mode)")
	}
}

func TestModifiableCssCommentIsInvisibleHidingComments(t *testing.T) {
	span := cssSpan()
	c := NewModifiableCssComment("/* test */", span)
	if !c.IsInvisibleHidingComments() {
		t.Error("non-preserved comment should be invisible when hiding comments")
	}

	c2 := NewModifiableCssComment("/*! important */", span)
	if c2.IsInvisibleHidingComments() {
		t.Error("preserved comment should not be invisible when hiding comments")
	}
}

func TestModifiableCssCommentAccept(t *testing.T) {
	span := cssSpan()
	c := NewModifiableCssComment("/* test */", span)

	var mv mockModifiableVisitor
	err := c.AcceptModifiableVoid(&mv)
	if err != nil {
		t.Fatal(err)
	}
	if mv.visited != "ModifiableCssComment" {
		t.Errorf("expected ModifiableCssComment, got %s", mv.visited)
	}

	var iv mockVisitor
	_, err = c.AcceptVoid(&iv)
	if err != nil {
		t.Fatal(err)
	}
	if iv.visited != "CssComment" {
		t.Errorf("expected CssComment, got %s", iv.visited)
	}
}

// ---------------------------------------------------------------------------
// CssImport / ModifiableCssImport tests
// ---------------------------------------------------------------------------

func TestModifiableCssImportConstruction(t *testing.T) {
	span := cssSpan()
	url := cssStrValue(`"foo.css"`)
	imp := NewModifiableCssImport(url, span, nil)

	if imp.URL() != url {
		t.Error("url mismatch")
	}
	if imp.Modifiers() != nil {
		t.Error("modifiers should be nil")
	}
	s, err := imp.Span()
	if err != nil {
		t.Fatal(err)
	}
	if s != span {
		t.Error("span mismatch")
	}
}

func TestModifiableCssImportWithModifiers(t *testing.T) {
	span := cssSpan()
	url := cssStrValue(`"foo.css"`)
	mod := cssStrValue("screen")
	imp := NewModifiableCssImport(url, span, &mod)

	if imp.Modifiers() == nil {
		t.Fatal("modifiers should not be nil")
	}
	if *imp.Modifiers() != mod {
		t.Error("modifiers mismatch")
	}
}

func TestModifiableCssImportIsInvisible(t *testing.T) {
	span := cssSpan()
	url := cssStrValue(`"foo.css"`)
	imp := NewModifiableCssImport(url, span, nil)
	if imp.IsInvisible() {
		t.Error("import should not be invisible")
	}
}

func TestModifiableCssImportAccept(t *testing.T) {
	span := cssSpan()
	url := cssStrValue(`"foo.css"`)
	imp := NewModifiableCssImport(url, span, nil)

	var mv mockModifiableVisitor
	err := imp.AcceptModifiableVoid(&mv)
	if err != nil {
		t.Fatal(err)
	}
	if mv.visited != "ModifiableCssImport" {
		t.Errorf("expected ModifiableCssImport, got %s", mv.visited)
	}
}

// ---------------------------------------------------------------------------
// CssDeclaration / ModifiableCssDeclaration tests
// ---------------------------------------------------------------------------

func cssDeclValue(s string) sasscommon.CssValue[Value] {
	return sasscommon.NewCssValue(Value(&SassString{Text: s, HasQuotes: false}), cssSpan())
}

func TestModifiableCssDeclarationConstruction(t *testing.T) {
	span := cssSpan()
	name := cssStrValue("color")
	val := cssDeclValue("red")
	d, err := NewModifiableCssDeclaration(name, val, span, true, nil)
	if err != nil {
		t.Fatal(err)
	}

	if d.Name() != name {
		t.Error("name mismatch")
	}
	if d.Value() != val {
		t.Error("value mismatch")
	}
	if !d.ParsedAsSassScript() {
		t.Error("should be parsed as SassScript")
	}
}

func TestModifiableCssDeclarationNotParsedAsSassScript(t *testing.T) {
	span := cssSpan()
	name := cssStrValue("--custom")
	val := cssDeclValue("blue")
	d, err := NewModifiableCssDeclaration(name, val, span, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if d.ParsedAsSassScript() {
		t.Error("should not be parsed as SassScript")
	}
}

func TestModifiableCssDeclarationNotParsedError(t *testing.T) {
	span := cssSpan()
	name := cssStrValue("color")
	num := NewUnitlessNumber(42)
	val := sasscommon.NewCssValue(Value(num), span)
	_, err := NewModifiableCssDeclaration(name, val, span, false, nil)
	if err == nil {
		t.Error("expected error when parsedAsSassScript=false with non-string value")
	}
}

func TestModifiableCssDeclarationIsCustomProperty(t *testing.T) {
	span := cssSpan()
	name := cssStrValue("--custom-prop")
	val := cssDeclValue("red")
	d, _ := NewModifiableCssDeclaration(name, val, span, true, nil)

	if !d.IsCustomProperty() {
		t.Error("should be a custom property")
	}

	name2 := cssStrValue("color")
	d2, _ := NewModifiableCssDeclaration(name2, val, span, true, nil)
	if d2.IsCustomProperty() {
		t.Error("color should not be a custom property")
	}
}

func TestModifiableCssDeclarationString(t *testing.T) {
	span := cssSpan()
	name := cssStrValue("color")
	val := cssDeclValue("red")
	d, _ := NewModifiableCssDeclaration(name, val, span, true, nil)

	s := fmt.Sprint(d)
	// Go's SassString doesn't implement fmt.Stringer (returns (string,error)),
	// so CssValue[Value].String() uses the default struct format of the inner type.
	if !strings.Contains(s, "color") || !strings.Contains(s, "red") {
		t.Errorf("String() = %q, expected it to contain 'color: red;' (got raw Go struct format)", s)
	}
}

func TestModifiableCssDeclarationAccept(t *testing.T) {
	span := cssSpan()
	name := cssStrValue("color")
	val := cssDeclValue("red")
	d, _ := NewModifiableCssDeclaration(name, val, span, true, nil)

	var mv mockModifiableVisitor
	err := d.AcceptModifiableVoid(&mv)
	if err != nil {
		t.Fatal(err)
	}
	if mv.visited != "ModifiableCssDeclaration" {
		t.Errorf("expected ModifiableCssDeclaration, got %s", mv.visited)
	}
}

// ---------------------------------------------------------------------------
// CssAtRule / ModifiableCssAtRule tests
// ---------------------------------------------------------------------------

func TestModifiableCssAtRuleConstruction(t *testing.T) {
	span := cssSpan()
	name := cssStrValue("media")
	r := NewModifiableCssAtRule(name, span, false, nil)

	if r.Name() != name {
		t.Error("name mismatch")
	}
	if r.Value() != nil {
		t.Error("value should be nil")
	}
	if r.IsChildless() {
		t.Error("should not be childless")
	}
}

func TestModifiableCssAtRuleChildless(t *testing.T) {
	span := cssSpan()
	name := cssStrValue("import")
	r := NewModifiableCssAtRule(name, span, true, nil)

	if !r.IsChildless() {
		t.Error("should be childless")
	}

	child := NewModifiableCssComment("/* test */", span)
	err := r.AddChild(child)
	if err == nil {
		t.Error("expected error when adding child to childless at-rule")
	}
}

func TestModifiableCssAtRuleWithValue(t *testing.T) {
	span := cssSpan()
	name := cssStrValue("import")
	val := cssStrValue(`"foo.css"`)
	r := NewModifiableCssAtRule(name, span, true, &val)

	if r.Value() == nil {
		t.Fatal("value should not be nil")
	}
	if *r.Value() != val {
		t.Error("value mismatch")
	}
}

func TestModifiableCssAtRuleAddChild(t *testing.T) {
	span := cssSpan()
	name := cssStrValue("media")
	r := NewModifiableCssAtRule(name, span, false, nil)
	child := NewModifiableCssComment("/* test */", span)

	err := r.AddChild(child)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Children()) != 1 {
		t.Errorf("expected 1 child, got %d", len(r.Children()))
	}
	if child.Parent() != r {
		t.Error("child parent not set")
	}
}

func TestModifiableCssAtRuleCopyWithoutChildren(t *testing.T) {
	span := cssSpan()
	name := cssStrValue("media")
	r := NewModifiableCssAtRule(name, span, false, nil)
	r.AddChild(NewModifiableCssComment("/* test */", span))

	copy, err := r.CopyWithoutChildren()
	if err != nil {
		t.Fatal(err)
	}
	if len(copy.Children()) != 0 {
		t.Error("copy should have no children")
	}
}

func TestModifiableCssAtRuleEqualsIgnoringChildren(t *testing.T) {
	span := cssSpan()
	name := cssStrValue("media")
	a := NewModifiableCssAtRule(name, span, false, nil)
	b := NewModifiableCssAtRule(name, span, false, nil)

	eq, err := a.EqualsIgnoringChildren(b)
	if err != nil {
		t.Fatal(err)
	}
	if !eq {
		t.Error("identical at-rules should be equal")
	}

	c := NewModifiableCssAtRule(cssStrValue("supports"), span, false, nil)
	eq, _ = a.EqualsIgnoringChildren(c)
	if eq {
		t.Error("different-named at-rules should not be equal")
	}
}

// ---------------------------------------------------------------------------
// CssStyleRule / ModifiableCssStyleRule tests
// ---------------------------------------------------------------------------

func TestModifiableCssStyleRuleConstruction(t *testing.T) {
	span := cssSpan()
	sel, err := makSelList("foo")
	if err != nil {
		t.Fatal(err)
	}
	selBox := box.NewModifiableBox(sel).Seal()
	r := NewModifiableCssStyleRule(selBox, span, nil, false)

	if r.Selector() == nil {
		t.Error("selector should not be nil")
	}
	if r.FromPlainCss() {
		t.Error("should not be from plain css")
	}
	s, err := r.Span()
	if err != nil {
		t.Fatal(err)
	}
	if s != span {
		t.Error("span mismatch")
	}
}

func TestModifiableCssStyleRuleFromPlainCss(t *testing.T) {
	span := cssSpan()
	sel, err := makSelList("foo")
	if err != nil {
		t.Fatal(err)
	}
	selBox := box.NewModifiableBox(sel).Seal()
	r := NewModifiableCssStyleRule(selBox, span, nil, true)

	if !r.FromPlainCss() {
		t.Error("should be from plain css")
	}
}

func TestModifiableCssStyleRuleAddChild(t *testing.T) {
	span := cssSpan()
	sel, err := makSelList("foo")
	if err != nil {
		t.Fatal(err)
	}
	selBox := box.NewModifiableBox(sel).Seal()
	r := NewModifiableCssStyleRule(selBox, span, nil, false)
	child := NewModifiableCssComment("/* test */", span)

	err = r.AddChild(child)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Children()) != 1 {
		t.Errorf("expected 1 child, got %d", len(r.Children()))
	}
}

func TestModifiableCssStyleRuleCopyWithoutChildren(t *testing.T) {
	span := cssSpan()
	sel, err := makSelList("foo")
	if err != nil {
		t.Fatal(err)
	}
	selBox := box.NewModifiableBox(sel).Seal()
	r := NewModifiableCssStyleRule(selBox, span, nil, false)
	r.AddChild(NewModifiableCssComment("/* test */", span))

	copy, err := r.CopyWithoutChildren()
	if err != nil {
		t.Fatal(err)
	}
	if len(copy.Children()) != 0 {
		t.Error("copy should have no children")
	}
	// The copy should share the same selector
	copySR, ok := copy.(*ModifiableCssStyleRule)
	if !ok {
		t.Fatal("copy should be *ModifiableCssStyleRule")
	}
	if copySR.Selector() != r.Selector() {
		t.Error("copy should share the same selector")
	}
}

func TestModifiableCssStyleRuleIsInvisibleBogusSelector(t *testing.T) {
	span := cssSpan()
	sel, err := makSelList("foo")
	if err != nil {
		t.Fatal(err)
	}
	// Normal selector is not invisible
	if sel.IsInvisible() {
		t.Error("normal selector should not be invisible")
	}

	selBox := box.NewModifiableBox(sel).Seal()
	r := NewModifiableCssStyleRule(selBox, span, nil, false)
	// Empty style rule (no children): all children vacuously invisible → invisible
	if !r.IsInvisible() {
		t.Error("empty style rule should be invisible (vacuously)")
	}

	// Add a visible child (not-invisible comment) → style rule becomes visible
	r.AddChild(NewModifiableCssComment("/* visible */", span))
	if r.IsInvisible() {
		t.Error("style rule with visible child should not be invisible")
	}
}

func TestModifiableCssStyleRuleAccept(t *testing.T) {
	span := cssSpan()
	sel, err := makSelList("foo")
	if err != nil {
		t.Fatal(err)
	}
	selBox := box.NewModifiableBox(sel).Seal()
	r := NewModifiableCssStyleRule(selBox, span, nil, false)

	var mv mockModifiableVisitor
	err = r.AcceptModifiableVoid(&mv)
	if err != nil {
		t.Fatal(err)
	}
	if mv.visited != "ModifiableCssStyleRule" {
		t.Errorf("expected ModifiableCssStyleRule, got %s", mv.visited)
	}

	var iv mockVisitor
	_, err = r.AcceptVoid(&iv)
	if err != nil {
		t.Fatal(err)
	}
	if iv.visited != "CssStyleRule" {
		t.Errorf("expected CssStyleRule, got %s", iv.visited)
	}
}

func TestModifiableCssStyleRuleEqualsIgnoringChildren(t *testing.T) {
	span := cssSpan()
	sel, err := makSelList("foo")
	if err != nil {
		t.Fatal(err)
	}
	selBox1 := box.NewModifiableBox(sel).Seal()
	selBox2 := box.NewModifiableBox(sel).Seal()
	a := NewModifiableCssStyleRule(selBox1, span, nil, false)
	b := NewModifiableCssStyleRule(selBox2, span, nil, false)

	eq, err := a.EqualsIgnoringChildren(b)
	if err != nil {
		t.Fatal(err)
	}
	if !eq {
		t.Error("style rules with same selector should be equal ignoring children")
	}
}

// ---------------------------------------------------------------------------
// CssMediaRule / ModifiableCssMediaRule tests
// ---------------------------------------------------------------------------

func makeMediaQuery(cond string) CssMediaQuery {
	q, err := NewCssMediaQueryCondition([]string{cond}, nil)
	if err != nil {
		panic(err)
	}
	return *q
}

func TestModifiableCssMediaRuleConstruction(t *testing.T) {
	span := cssSpan()
	queries := []CssMediaQuery{makeMediaQuery("(color)")}
	r, err := NewModifiableCssMediaRule(queries, span)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Queries()) != 1 {
		t.Errorf("expected 1 query, got %d", len(r.Queries()))
	}
}

func TestModifiableCssMediaRuleEmptyQueries(t *testing.T) {
	span := cssSpan()
	_, err := NewModifiableCssMediaRule([]CssMediaQuery{}, span)
	if err == nil {
		t.Error("expected error for empty queries")
	}
}

func TestModifiableCssMediaRuleAddChild(t *testing.T) {
	span := cssSpan()
	r, err := NewModifiableCssMediaRule([]CssMediaQuery{makeMediaQuery("(color)")}, span)
	if err != nil {
		t.Fatal(err)
	}
	child := NewModifiableCssComment("/* test */", span)
	r.AddChild(child)
	if len(r.Children()) != 1 {
		t.Errorf("expected 1 child, got %d", len(r.Children()))
	}
}

func TestModifiableCssMediaRuleCopyWithoutChildren(t *testing.T) {
	span := cssSpan()
	r, err := NewModifiableCssMediaRule([]CssMediaQuery{makeMediaQuery("(color)")}, span)
	if err != nil {
		t.Fatal(err)
	}
	r.AddChild(NewModifiableCssComment("/* test */", span))

	copy, err := r.CopyWithoutChildren()
	if err != nil {
		t.Fatal(err)
	}
	if len(copy.Children()) != 0 {
		t.Error("copy should have no children")
	}
}

func TestModifiableCssMediaRuleEqualsIgnoringChildren(t *testing.T) {
	span := cssSpan()
	a, _ := NewModifiableCssMediaRule([]CssMediaQuery{makeMediaQuery("(color)")}, span)
	b, _ := NewModifiableCssMediaRule([]CssMediaQuery{makeMediaQuery("(color)")}, span)

	eq, err := a.EqualsIgnoringChildren(b)
	if err != nil {
		t.Fatal(err)
	}
	if !eq {
		t.Error("media rules with same queries should be equal")
	}
}

// ---------------------------------------------------------------------------
// CssSupportsRule / ModifiableCssSupportsRule tests
// ---------------------------------------------------------------------------

func TestModifiableCssSupportsRuleConstruction(t *testing.T) {
	span := cssSpan()
	cond := cssStrValue("display: grid")
	r := NewModifiableCssSupportsRule(cond, span)

	if r.Condition() != cond {
		t.Error("condition mismatch")
	}
	s, err := r.Span()
	if err != nil {
		t.Fatal(err)
	}
	if s != span {
		t.Error("span mismatch")
	}
}

func TestModifiableCssSupportsRuleCopyWithoutChildren(t *testing.T) {
	span := cssSpan()
	cond := cssStrValue("display: grid")
	r := NewModifiableCssSupportsRule(cond, span)
	r.AddChild(NewModifiableCssComment("/* test */", span))

	copy, err := r.CopyWithoutChildren()
	if err != nil {
		t.Fatal(err)
	}
	if len(copy.Children()) != 0 {
		t.Error("copy should have no children")
	}
}

// ---------------------------------------------------------------------------
// CssKeyframeBlock / ModifiableCssKeyframeBlock tests
// ---------------------------------------------------------------------------

func TestModifiableCssKeyframeBlockConstruction(t *testing.T) {
	span := cssSpan()
	sel := sasscommon.NewCssValue([]string{"10%"}, span)
	b := NewModifiableCssKeyframeBlock(sel, span)

	if len(b.Selector().Value) != 1 || b.Selector().Value[0] != "10%" {
		t.Errorf("selector mismatch: got %v", b.Selector().Value)
	}
	s, err := b.Span()
	if err != nil {
		t.Fatal(err)
	}
	if s != span {
		t.Error("span mismatch")
	}
}

func TestModifiableCssKeyframeBlockAddChild(t *testing.T) {
	span := cssSpan()
	sel := sasscommon.NewCssValue([]string{"10%"}, span)
	b := NewModifiableCssKeyframeBlock(sel, span)
	child := NewModifiableCssComment("/* test */", span)

	err := b.AddChild(child)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Children()) != 1 {
		t.Errorf("expected 1 child, got %d", len(b.Children()))
	}
}

func TestModifiableCssKeyframeBlockCopyWithoutChildren(t *testing.T) {
	span := cssSpan()
	sel := sasscommon.NewCssValue([]string{"10%"}, span)
	b := NewModifiableCssKeyframeBlock(sel, span)
	b.AddChild(NewModifiableCssComment("/* test */", span))

	copy, err := b.CopyWithoutChildren()
	if err != nil {
		t.Fatal(err)
	}
	if len(copy.Children()) != 0 {
		t.Error("copy should have no children")
	}
}

// ---------------------------------------------------------------------------
// Parent chain tests
// ---------------------------------------------------------------------------

func TestHasFollowingSibling(t *testing.T) {
	span := cssSpan()
	root := NewModifiableCssStylesheet(span)
	child1 := NewModifiableCssComment("/* first */", span)
	child2 := NewModifiableCssComment("/* second */", span)

	root.AddChild(child1)
	root.AddChild(child2)

	// child2 is after child1. child2 is a comment (isInvisible=false → visible).
	// child1 has a visible following sibling (child2), so HasFollowingSibling = true.
	if !child1.HasFollowingSibling() {
		t.Error("child1 should have a following sibling (child2 is visible)")
	}
}

func TestRemoveChild(t *testing.T) {
	span := cssSpan()
	root := NewModifiableCssStylesheet(span)
	child1 := NewModifiableCssComment("/* first */", span)
	child2 := NewModifiableCssComment("/* second */", span)

	root.AddChild(child1)
	root.AddChild(child2)

	err := child1.Remove()
	if err != nil {
		t.Fatal(err)
	}

	if len(root.Children()) != 1 {
		t.Errorf("expected 1 child after removal, got %d", len(root.Children()))
	}
	if child1.Parent() != nil {
		t.Error("removed child parent should be nil")
	}
}

func TestParentChainWalk(t *testing.T) {
	span := cssSpan()
	root := NewModifiableCssStylesheet(span)
	mediaName := cssStrValue("media")
	media := NewModifiableCssAtRule(mediaName, span, false, nil)
	child := NewModifiableCssComment("/* test */", span)

	root.AddChild(media)
	media.AddChild(child)

	if child.Parent() != media {
		t.Error("child parent should be media")
	}
	if media.Parent() != root {
		t.Error("media parent should be root")
	}
}

// ---------------------------------------------------------------------------
// EveryCssVisitor tests
// ---------------------------------------------------------------------------

func TestEveryCssVisitorAllChildrenTrue(t *testing.T) {
	span := cssSpan()
	ss := NewCssStylesheet(nil, span)

	v := &EveryCssVisitor{}
	result, err := ss.AcceptBool(v)
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("empty stylesheet should return true")
	}
}

func TestEveryCssVisitorCommentsReturnFalse(t *testing.T) {
	span := cssSpan()
	c := NewModifiableCssComment("/* test */", span)
	v := &EveryCssVisitor{}
	result, err := c.AcceptBool(v)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("comment should return false")
	}
}

func TestEveryCssVisitorDeclarationsReturnFalse(t *testing.T) {
	span := cssSpan()
	name := cssStrValue("color")
	val := cssDeclValue("red")
	d, _ := NewModifiableCssDeclaration(name, val, span, true, nil)
	v := &EveryCssVisitor{}
	result, err := d.AcceptBool(v)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("declaration should return false")
	}
}

func TestEveryCssVisitorImportsReturnFalse(t *testing.T) {
	span := cssSpan()
	url := cssStrValue(`"foo.css"`)
	imp := NewModifiableCssImport(url, span, nil)
	v := &EveryCssVisitor{}
	result, err := imp.AcceptBool(v)
	if err != nil {
		t.Fatal(err)
	}
	if result {
		t.Error("import should return false")
	}
}
