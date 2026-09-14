package eval

import (
	"strings"
	"testing"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassio"
	"github.com/bancek/go-sass/sasslogger"
	"github.com/bancek/go-sass/value"
)

// cssTestVisitor creates an EvaluateVisitor with a root stylesheet as parent
// and root, for testing CSS visitor methods.
func cssTestVisitor(t *testing.T, logger sasslogger.Logger) *EvaluateVisitor {
	t.Helper()
	if logger == nil {
		logger = &recordingLogger{}
	}
	ic := NewImportCacheWithOptions(nil, nil, "", false, sassio.NewDefaultIO(), nil)
	v := NewEvaluateVisitor(ic, logger)
	v.ec.SetCallableNode(testNode{span: testSpan()})

	rootSpan := testSpan()
	root := value.NewModifiableCssStylesheet(rootSpan)
	v.root = root
	v.parent = root
	v.member = "root stylesheet"

	return v
}

// makeTestSelectorList creates a simple ".foo" SelectorList for tests.
func makeTestSelectorList(t *testing.T) *value.SelectorList {
	t.Helper()
	classSel := value.NewClassSelector(".foo", testSpan())
	compound, err := value.NewCompoundSelector([]value.SimpleSelector{classSel}, testSpan())
	if err != nil {
		t.Fatalf("NewCompoundSelector: %v", err)
	}
	comp := value.NewComplexSelectorComponent(compound, nil, testSpan())
	complexSel, err := value.NewComplexSelector(nil, []*value.ComplexSelectorComponent{comp}, testSpan(), false)
	if err != nil {
		t.Fatalf("NewComplexSelector: %v", err)
	}
	selList, err := value.NewSelectorList([]*value.ComplexSelector{complexSel}, testSpan())
	if err != nil {
		t.Fatalf("NewSelectorList: %v", err)
	}
	return selList
}

// ===========================================================================
// VisitCssStylesheet tests
// ===========================================================================

func TestVisitCssStylesheetEmpty(t *testing.T) {
	v := cssTestVisitor(t, nil)
	node := value.NewCssStylesheet([]value.CssNode{}, testSpan())

	_, err := v.VisitCssStylesheet(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVisitCssStylesheetWithChildren(t *testing.T) {
	v := cssTestVisitor(t, nil)
	comment := value.NewModifiableCssComment("/* hello */", testSpan())
	node := value.NewCssStylesheet([]value.CssNode{comment}, testSpan())

	_, err := v.VisitCssStylesheet(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v.root.Children()) != 1 {
		t.Fatalf("expected 1 child, got %d", len(v.root.Children()))
	}
}

// ===========================================================================
// VisitCssComment tests
// ===========================================================================

func TestVisitCssCommentAddsChild(t *testing.T) {
	v := cssTestVisitor(t, nil)
	comment := value.NewModifiableCssComment("/* hello */", testSpan())

	_, err := v.VisitCssComment(comment)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v.root.Children()) != 1 {
		t.Fatalf("expected 1 child, got %d", len(v.root.Children()))
	}
}

func TestVisitCssCommentIncrementsEndOfImports(t *testing.T) {
	v := cssTestVisitor(t, nil)
	v.endOfImports = 0
	comment := value.NewModifiableCssComment("/* hello */", testSpan())

	_, err := v.VisitCssComment(comment)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.endOfImports != 1 {
		t.Errorf("endOfImports = %d, want 1", v.endOfImports)
	}
}

func TestVisitCssCommentNotAtRoot(t *testing.T) {
	v := cssTestVisitor(t, nil)
	atName := sasscommon.NewCssValue("media", testSpan())
	atRule := value.NewModifiableCssAtRule(atName, testSpan(), false, nil)
	v.parent = atRule

	comment := value.NewModifiableCssComment("/* hello */", testSpan())
	_, err := v.VisitCssComment(comment)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.endOfImports != 0 {
		t.Errorf("endOfImports should be 0 when parent != root, got %d", v.endOfImports)
	}
}

// ===========================================================================
// VisitCssImport tests
// ===========================================================================

func TestVisitCssImportParentNotRoot(t *testing.T) {
	v := cssTestVisitor(t, nil)
	atName := sasscommon.NewCssValue("custom", testSpan())
	atRule := value.NewModifiableCssAtRule(atName, testSpan(), false, nil)
	v.parent = atRule

	imp := value.NewModifiableCssImport(
		sasscommon.NewCssValue(`"foo.css"`, testSpan()),
		testSpan(),
		nil,
	)
	_, err := v.VisitCssImport(imp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(atRule.Children()) != 1 {
		t.Fatalf("expected 1 child, got %d", len(atRule.Children()))
	}
}

func TestVisitCssImportAtRootEndOfImports(t *testing.T) {
	v := cssTestVisitor(t, nil)
	v.endOfImports = 0

	imp := value.NewModifiableCssImport(
		sasscommon.NewCssValue(`"foo.css"`, testSpan()),
		testSpan(),
		nil,
	)
	_, err := v.VisitCssImport(imp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.endOfImports != 1 {
		t.Errorf("endOfImports = %d, want 1", v.endOfImports)
	}
}

func TestVisitCssImportAtRootOutOfOrder(t *testing.T) {
	v := cssTestVisitor(t, nil)
	comment := value.NewModifiableCssComment("/* hello */", testSpan())
	v.root.AddChild(comment)
	v.endOfImports = 0

	imp := value.NewModifiableCssImport(
		sasscommon.NewCssValue(`"foo.css"`, testSpan()),
		testSpan(),
		nil,
	)
	_, err := v.VisitCssImport(imp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v.outOfOrderImports) != 1 {
		t.Errorf("outOfOrderImports = %d, want 1", len(v.outOfOrderImports))
	}
}

// ===========================================================================
// VisitCssDeclaration tests
// ===========================================================================

func TestVisitCssDeclarationAddsChild(t *testing.T) {
	v := cssTestVisitor(t, nil)
	decl, declErr := value.NewModifiableCssDeclaration(
		sasscommon.NewCssValue("color", testSpan()),
		sasscommon.NewCssValue(value.Value(&value.SassString{Text: "red"}), testSpan()),
		testSpan(),
		true,
		nil,
	)
	if declErr != nil {
		t.Fatalf("NewModifiableCssDeclaration: %v", declErr)
	}
	_, err := v.VisitCssDeclaration(decl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v.root.Children()) != 1 {
		t.Fatalf("expected 1 child, got %d", len(v.root.Children()))
	}
}

// ===========================================================================
// VisitCssKeyframeBlock tests
// ===========================================================================

func TestVisitCssKeyframeBlockAddsChild(t *testing.T) {
	v := cssTestVisitor(t, nil)
	kb := value.NewModifiableCssKeyframeBlock(
		sasscommon.NewCssValue([]string{"10%"}, testSpan()),
		testSpan(),
	)
	_, err := v.VisitCssKeyframeBlock(kb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v.root.Children()) != 1 {
		t.Fatalf("expected 1 child, got %d", len(v.root.Children()))
	}
}

// ===========================================================================
// VisitCssAtRule tests
// ===========================================================================

func TestVisitCssAtRuleChildless(t *testing.T) {
	v := cssTestVisitor(t, nil)
	atNode := value.NewModifiableCssAtRule(
		sasscommon.NewCssValue("import", testSpan()),
		testSpan(),
		true,
		nil,
	)
	_, err := v.VisitCssAtRule(atNode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v.root.Children()) != 1 {
		t.Fatalf("expected 1 child, got %d", len(v.root.Children()))
	}
}

func TestVisitCssAtRuleNonChildless(t *testing.T) {
	v := cssTestVisitor(t, nil)
	atNode := value.NewModifiableCssAtRule(
		sasscommon.NewCssValue("media", testSpan()),
		testSpan(),
		false,
		nil,
	)
	_, err := v.VisitCssAtRule(atNode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v.root.Children()) != 1 {
		t.Fatalf("expected 1 child, got %d", len(v.root.Children()))
	}
}

func TestVisitCssAtRuleDeclarationNameError(t *testing.T) {
	v := cssTestVisitor(t, nil)
	v.declarationName = "color"

	atNode := value.NewModifiableCssAtRule(
		sasscommon.NewCssValue("media", testSpan()),
		testSpan(),
		false,
		nil,
	)
	_, err := v.VisitCssAtRule(atNode)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	want := strings.Join([]string{
		`Error: At-rules may not be used within nested declarations.`,
		`  ╷`,
		`1 │ test { color: red; }`,
		`  │ ^^^^`,
		`  ╵`,
		`  - 1:1  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestVisitCssAtRuleRestoresFlags(t *testing.T) {
	v := cssTestVisitor(t, nil)
	atNode := value.NewModifiableCssAtRule(
		sasscommon.NewCssValue("keyframes", testSpan()),
		testSpan(),
		false,
		nil,
	)
	_, err := v.VisitCssAtRule(atNode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.inKeyframes {
		t.Error("inKeyframes should be restored to false after VisitCssAtRule")
	}
}

// ===========================================================================
// VisitCssSupportsRule tests
// ===========================================================================

func TestVisitCssSupportsRuleAddsChild(t *testing.T) {
	v := cssTestVisitor(t, nil)
	suppNode := value.NewModifiableCssSupportsRule(
		sasscommon.NewCssValue("display: grid", testSpan()),
		testSpan(),
	)
	_, err := v.VisitCssSupportsRule(suppNode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v.root.Children()) != 1 {
		t.Fatalf("expected 1 child, got %d", len(v.root.Children()))
	}
}

func TestVisitCssSupportsRuleDeclarationNameError(t *testing.T) {
	v := cssTestVisitor(t, nil)
	v.declarationName = "color"

	suppNode := value.NewModifiableCssSupportsRule(
		sasscommon.NewCssValue("display: grid", testSpan()),
		testSpan(),
	)
	_, err := v.VisitCssSupportsRule(suppNode)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	want := strings.Join([]string{
		`Error: Supports rules may not be used within nested declarations.`,
		`  ╷`,
		`1 │ test { color: red; }`,
		`  │ ^^^^`,
		`  ╵`,
		`  - 1:1  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// ===========================================================================
// VisitCssStyleRule tests
// ===========================================================================

func TestVisitCssStyleRuleDeclarationNameError(t *testing.T) {
	v := cssTestVisitor(t, nil)
	v.declarationName = "color"

	selList := makeTestSelectorList(t)
	sr := value.NewModifiableCssStyleRule(nil, testSpan(), selList, false)
	_, err := v.VisitCssStyleRule(sr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "Style rules may not be used within nested declarations.") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestVisitCssStyleRuleInKeyframesError(t *testing.T) {
	v := cssTestVisitor(t, nil)
	v.inKeyframes = true
	kb := value.NewModifiableCssKeyframeBlock(
		sasscommon.NewCssValue([]string{"10%"}, testSpan()),
		testSpan(),
	)
	v.parent = kb

	selList := makeTestSelectorList(t)
	sr := value.NewModifiableCssStyleRule(nil, testSpan(), selList, false)
	_, err := v.VisitCssStyleRule(sr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "Style rules may not be used within keyframe blocks.") {
		t.Errorf("error = %q", err.Error())
	}
}

// ===========================================================================
// VisitCssMediaRule tests
// ===========================================================================

func TestVisitCssMediaRuleDeclarationNameError(t *testing.T) {
	v := cssTestVisitor(t, nil)
	v.declarationName = "color"

	screen := "screen"
	mq := value.NewCssMediaQueryType(&screen, nil, nil)
	mr, _ := value.NewModifiableCssMediaRule([]value.CssMediaQuery{*mq}, testSpan())
	_, err := v.VisitCssMediaRule(mr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "Media rules may not be used within nested declarations.") {
		t.Errorf("error = %q", err.Error())
	}
}

// ===========================================================================
// withStyleRule tests
// ===========================================================================

func TestWithStyleRuleSaveRestore(t *testing.T) {
	v := cssTestVisitor(t, nil)

	selList := makeTestSelectorList(t)
	sr := value.NewModifiableCssStyleRule(nil, testSpan(), selList, false)

	called := false
	err := v.withStyleRule(sr, func() error {
		called = true
		if v.styleRuleIgnoringAtRoot == nil {
			t.Error("styleRuleIgnoringAtRoot should be set inside callback")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("callback was not called")
	}
	if v.styleRuleIgnoringAtRoot != nil {
		t.Error("styleRuleIgnoringAtRoot should be nil after callback")
	}
}

// ===========================================================================
// hasCssNesting tests
// ===========================================================================

func TestHasCssNestingFalseWhenNoStyleRule(t *testing.T) {
	v := cssTestVisitor(t, nil)
	v.styleRuleIgnoringAtRoot = nil

	if v.hasCssNesting() {
		t.Error("hasCssNesting should be false without a style rule")
	}
}

func TestHasCssNestingFalseWithSingleStyleRule(t *testing.T) {
	v := cssTestVisitor(t, nil)

	selList := makeTestSelectorList(t)
	sr := value.NewModifiableCssStyleRule(nil, testSpan(), selList, false)
	v.styleRuleIgnoringAtRoot = sr

	if v.hasCssNesting() {
		t.Error("hasCssNesting should be false with one style rule")
	}
}

// ===========================================================================
// Imported CSS Visitor tests
// ===========================================================================

func TestImportedCssComment(t *testing.T) {
	v := cssTestVisitor(t, nil)
	iv := &importedCssVisitor{v: v}

	node := value.NewModifiableCssComment("/* imported */", testSpan())
	_, err := iv.VisitCssComment(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImportedCssDeclaration(t *testing.T) {
	v := cssTestVisitor(t, nil)
	iv := &importedCssVisitor{v: v}

	node, declErr := value.NewModifiableCssDeclaration(
		sasscommon.NewCssValue("color", testSpan()),
		sasscommon.NewCssValue(value.Value(&value.SassString{Text: "red"}), testSpan()),
		testSpan(),
		true,
		nil,
	)
	if declErr != nil {
		t.Fatalf("NewModifiableCssDeclaration: %v", declErr)
	}
	_, err := iv.VisitCssDeclaration(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImportedCssStyleRule(t *testing.T) {
	v := cssTestVisitor(t, nil)
	iv := &importedCssVisitor{v: v}

	selList := makeTestSelectorList(t)
	sr := value.NewModifiableCssStyleRule(nil, testSpan(), selList, false)
	_, err := iv.VisitCssStyleRule(sr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImportedCssKeyframeBlockError(t *testing.T) {
	v := cssTestVisitor(t, nil)
	iv := &importedCssVisitor{v: v}

	node := value.NewModifiableCssKeyframeBlock(
		sasscommon.NewCssValue([]string{"10%"}, testSpan()),
		testSpan(),
	)
	_, err := iv.VisitCssKeyframeBlock(node)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	want := "visitCssKeyframeBlock() should never be called"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestImportedCssAtRuleChildless(t *testing.T) {
	v := cssTestVisitor(t, nil)
	iv := &importedCssVisitor{v: v}

	node := value.NewModifiableCssAtRule(
		sasscommon.NewCssValue("import", testSpan()),
		testSpan(),
		true,
		nil,
	)
	_, err := iv.VisitCssAtRule(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImportedCssAtRuleNonChildless(t *testing.T) {
	v := cssTestVisitor(t, nil)
	iv := &importedCssVisitor{v: v}

	node := value.NewModifiableCssAtRule(
		sasscommon.NewCssValue("media", testSpan()),
		testSpan(),
		false,
		nil,
	)
	_, err := iv.VisitCssAtRule(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImportedCssSupportsRule(t *testing.T) {
	v := cssTestVisitor(t, nil)
	iv := &importedCssVisitor{v: v}

	node := value.NewModifiableCssSupportsRule(
		sasscommon.NewCssValue("display: grid", testSpan()),
		testSpan(),
	)
	_, err := iv.VisitCssSupportsRule(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImportedCssMediaRuleUnmerged(t *testing.T) {
	v := cssTestVisitor(t, nil)
	v.mediaQueries = nil
	iv := &importedCssVisitor{v: v}

	screen := "screen"
	mq := value.NewCssMediaQueryType(&screen, nil, nil)
	node, _ := value.NewModifiableCssMediaRule([]value.CssMediaQuery{*mq}, testSpan())
	_, err := iv.VisitCssMediaRule(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImportedCssImportAtRootEndOfImports(t *testing.T) {
	v := cssTestVisitor(t, nil)
	v.endOfImports = 0
	iv := &importedCssVisitor{v: v}

	node := value.NewModifiableCssImport(
		sasscommon.NewCssValue(`"foo.css"`, testSpan()),
		testSpan(),
		nil,
	)
	_, err := iv.VisitCssImport(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.endOfImports != 1 {
		t.Errorf("endOfImports = %d, want 1", v.endOfImports)
	}
}

func TestImportedCssImportNotAtRoot(t *testing.T) {
	v := cssTestVisitor(t, nil)
	iv := &importedCssVisitor{v: v}

	atName := sasscommon.NewCssValue("custom", testSpan())
	atRule := value.NewModifiableCssAtRule(atName, testSpan(), false, nil)
	v.parent = atRule

	node := value.NewModifiableCssImport(
		sasscommon.NewCssValue(`"bar.css"`, testSpan()),
		testSpan(),
		nil,
	)
	_, err := iv.VisitCssImport(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImportedCssStylesheet(t *testing.T) {
	v := cssTestVisitor(t, nil)
	iv := &importedCssVisitor{v: v}

	comment := value.NewModifiableCssComment("/* imported */", testSpan())
	ss := value.NewCssStylesheet([]value.CssNode{comment}, testSpan())
	_, err := iv.VisitCssStylesheet(ss)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
