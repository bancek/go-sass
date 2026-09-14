package sassclonecss

import (
	"testing"

	"github.com/bancek/go-sass/box"
	"github.com/bancek/go-sass/extend"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/value"
)

func cssSpan() sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte("css test"), nil)
	return sasscommon.NewFileSpan(fs, 0, 8)
}

func cssStrValue(s string) sasscommon.CssValue[string] {
	return sasscommon.NewCssValue(s, cssSpan())
}

func cssDeclValue(s string) sasscommon.CssValue[value.Value] {
	return sasscommon.NewCssValue(value.Value(&value.SassString{Text: s, HasQuotes: false}), cssSpan())
}

func cssMediaQuery(cond string) value.CssMediaQuery {
	q, err := value.NewCssMediaQueryCondition([]string{cond}, nil)
	if err != nil {
		panic(err)
	}
	return *q
}

func makSelList(name string) (*value.SelectorList, error) {
	class := value.NewClassSelector(name, sasscommon.BogusSpan)
	compound, err := value.NewCompoundSelector([]value.SimpleSelector{class}, sasscommon.BogusSpan)
	if err != nil {
		return nil, err
	}
	comp := value.NewComplexSelectorComponent(compound, nil, sasscommon.BogusSpan)
	cs, err := value.NewComplexSelector(nil, []*value.ComplexSelectorComponent{comp}, sasscommon.BogusSpan, false)
	if err != nil {
		return nil, err
	}
	return value.NewSelectorList([]*value.ComplexSelector{cs}, sasscommon.BogusSpan)
}

func makeStore() *extend.DefaultExtensionStore {
	return extend.NewDefaultExtensionStore()
}

func addSelToStore(store *extend.DefaultExtensionStore, name string) (*box.Box[*value.SelectorList], error) {
	sel, err := makSelList(name)
	if err != nil {
		return nil, err
	}
	return store.AddSelector(sel, nil)
}

// ---------------------------------------------------------------------------
// cloneCssStylesheet tests
// ---------------------------------------------------------------------------

func TestCloneCssStylesheet_Empty(t *testing.T) {
	modSheet := value.NewModifiableCssStylesheet(cssSpan())
	frozenSheet := modSheet.ToCssStylesheet()

	result, _, err := CloneCssStylesheet(frozenSheet, makeStore())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Children()) != 0 {
		t.Errorf("expected 0 children, got %d", len(result.Children()))
	}
}

func TestCloneCssStylesheet_Comment(t *testing.T) {
	modSheet := value.NewModifiableCssStylesheet(cssSpan())
	comment := value.NewModifiableCssComment("/* hello */", cssSpan())
	modSheet.AddChild(comment)
	frozenSheet := modSheet.ToCssStylesheet()

	result, _, err := CloneCssStylesheet(frozenSheet, makeStore())
	if err != nil {
		t.Fatal(err)
	}
	children := result.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if cc, ok := children[0].(*value.ModifiableCssComment); ok {
		if cc.Text() != "/* hello */" {
			t.Errorf("expected comment text '/* hello */', got %q", cc.Text())
		}
	} else {
		t.Errorf("expected ModifiableCssComment, got %T", children[0])
	}
}

func TestCloneCssStylesheet_Declaration(t *testing.T) {
	modSheet := value.NewModifiableCssStylesheet(cssSpan())
	span := cssSpan()
	name := cssStrValue("color")
	val := cssDeclValue("red")
	d, err := value.NewModifiableCssDeclaration(name, val, span, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	modSheet.AddChild(d)
	frozenSheet := modSheet.ToCssStylesheet()

	result, _, err := CloneCssStylesheet(frozenSheet, makeStore())
	if err != nil {
		t.Fatal(err)
	}
	children := result.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if dd, ok := children[0].(*value.ModifiableCssDeclaration); ok {
		if dd.ParsedAsSassScript() != true {
			t.Error("expected parsed_as_sass_script = true")
		}
	} else {
		t.Errorf("expected ModifiableCssDeclaration, got %T", children[0])
	}
}

func TestCloneCssStylesheet_Import(t *testing.T) {
	modSheet := value.NewModifiableCssStylesheet(cssSpan())
	span := cssSpan()
	url := cssStrValue(`"foo.css"`)
	imp := value.NewModifiableCssImport(url, span, nil)
	modSheet.AddChild(imp)
	frozenSheet := modSheet.ToCssStylesheet()

	result, _, err := CloneCssStylesheet(frozenSheet, makeStore())
	if err != nil {
		t.Fatal(err)
	}
	children := result.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if ii, ok := children[0].(*value.ModifiableCssImport); ok {
		if ii.URL().Value != `"foo.css"` {
			t.Errorf("expected url \"foo.css\", got %q", ii.URL().Value)
		}
	} else {
		t.Errorf("expected ModifiableCssImport, got %T", children[0])
	}
}

func TestCloneCssStylesheet_AtRuleChildless(t *testing.T) {
	modSheet := value.NewModifiableCssStylesheet(cssSpan())
	span := cssSpan()
	atRule := value.NewModifiableCssAtRule(cssStrValue("import"), span, true, nil)
	modSheet.AddChild(atRule)
	frozenSheet := modSheet.ToCssStylesheet()

	result, _, err := CloneCssStylesheet(frozenSheet, makeStore())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Children()) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children()))
	}
}

func TestCloneCssStylesheet_AtRuleWithChildren(t *testing.T) {
	modSheet := value.NewModifiableCssStylesheet(cssSpan())
	span := cssSpan()
	atRule := value.NewModifiableCssAtRule(cssStrValue("media"), span, false, nil)
	child := value.NewModifiableCssComment("/* nested */", span)
	atRule.AddChild(child)
	modSheet.AddChild(atRule)
	frozenSheet := modSheet.ToCssStylesheet()

	result, _, err := CloneCssStylesheet(frozenSheet, makeStore())
	if err != nil {
		t.Fatal(err)
	}
	children := result.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if ar, ok := children[0].(*value.ModifiableCssAtRule); ok {
		if len(ar.Children()) != 1 {
			t.Errorf("expected 1 nested child, got %d", len(ar.Children()))
		}
	}
}

func TestCloneCssStylesheet_StyleRule(t *testing.T) {
	store := makeStore()
	selBox, err := addSelToStore(store, ".foo")
	if err != nil {
		t.Fatal(err)
	}

	span := cssSpan()
	rule := value.NewModifiableCssStyleRule(selBox, span, nil, false)
	modSheet := value.NewModifiableCssStylesheet(span)
	modSheet.AddChild(rule)
	frozenSheet := modSheet.ToCssStylesheet()

	_, newStore, err := CloneCssStylesheet(frozenSheet, store)
	if err != nil {
		t.Fatal(err)
	}
	if newStore == nil {
		t.Fatal("new store should not be nil")
	}
}

func TestCloneCssStylesheet_MediaRule(t *testing.T) {
	modSheet := value.NewModifiableCssStylesheet(cssSpan())
	span := cssSpan()
	media, err := value.NewModifiableCssMediaRule([]value.CssMediaQuery{cssMediaQuery("(color)")}, span)
	if err != nil {
		t.Fatal(err)
	}
	media.AddChild(value.NewModifiableCssComment("/* inside */", span))
	modSheet.AddChild(media)
	frozenSheet := modSheet.ToCssStylesheet()

	result, _, err := CloneCssStylesheet(frozenSheet, makeStore())
	if err != nil {
		t.Fatal(err)
	}
	children := result.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if mr, ok := children[0].(*value.ModifiableCssMediaRule); ok {
		if len(mr.Children()) != 1 {
			t.Errorf("expected 1 nested child, got %d", len(mr.Children()))
		}
	}
}

func TestCloneCssStylesheet_SupportsRule(t *testing.T) {
	modSheet := value.NewModifiableCssStylesheet(cssSpan())
	span := cssSpan()
	cond := cssStrValue("display: grid")
	supports := value.NewModifiableCssSupportsRule(cond, span)
	supports.AddChild(value.NewModifiableCssComment("/* inside */", span))
	modSheet.AddChild(supports)
	frozenSheet := modSheet.ToCssStylesheet()

	result, _, err := CloneCssStylesheet(frozenSheet, makeStore())
	if err != nil {
		t.Fatal(err)
	}
	children := result.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if sr, ok := children[0].(*value.ModifiableCssSupportsRule); ok {
		if sr.Condition().Value != "display: grid" {
			t.Errorf("expected condition 'display: grid', got %q", sr.Condition().Value)
		}
	}
}

func TestCloneCssStylesheet_KeyframeBlock(t *testing.T) {
	modSheet := value.NewModifiableCssStylesheet(cssSpan())
	span := cssSpan()
	selSpan := cssSpan()
	sel := sasscommon.NewCssValue([]string{"10%"}, selSpan)
	block := value.NewModifiableCssKeyframeBlock(sel, span)
	block.AddChild(value.NewModifiableCssComment("/* inside */", span))
	modSheet.AddChild(block)
	frozenSheet := modSheet.ToCssStylesheet()

	result, _, err := CloneCssStylesheet(frozenSheet, makeStore())
	if err != nil {
		t.Fatal(err)
	}
	children := result.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if kb, ok := children[0].(*value.ModifiableCssKeyframeBlock); ok {
		if len(kb.Children()) != 1 {
			t.Errorf("expected 1 nested child, got %d", len(kb.Children()))
		}
	}
}

func TestCloneCssStylesheet_Nested(t *testing.T) {
	store := makeStore()
	selBox, err := addSelToStore(store, ".outer")
	if err != nil {
		t.Fatal(err)
	}

	span := cssSpan()
	outer := value.NewModifiableCssStyleRule(selBox, span, nil, false)
	media, err := value.NewModifiableCssMediaRule([]value.CssMediaQuery{cssMediaQuery("(color)")}, span)
	if err != nil {
		t.Fatal(err)
	}
	innerSelBox, err := addSelToStore(store, ".inner")
	if err != nil {
		t.Fatal(err)
	}
	inner := value.NewModifiableCssStyleRule(innerSelBox, span, nil, false)
	inner.AddChild(value.NewModifiableCssComment("/* deep */", span))
	media.AddChild(inner)
	outer.AddChild(media)

	modSheet := value.NewModifiableCssStylesheet(span)
	modSheet.AddChild(outer)
	frozenSheet := modSheet.ToCssStylesheet()

	result, _, err := CloneCssStylesheet(frozenSheet, store)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Children()) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children()))
	}
}

func TestCloneCssStylesheet_IsGroupEnd(t *testing.T) {
	modSheet := value.NewModifiableCssStylesheet(cssSpan())
	span := cssSpan()
	comment := value.NewModifiableCssComment("/* test */", span)
	comment.SetIsGroupEnd(true)
	modSheet.AddChild(comment)
	frozenSheet := modSheet.ToCssStylesheet()

	result, _, err := CloneCssStylesheet(frozenSheet, makeStore())
	if err != nil {
		t.Fatal(err)
	}
	children := result.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if cc, ok := children[0].(*value.ModifiableCssComment); ok {
		if !cc.IsGroupEnd() {
			t.Error("expected is_group_end = true on cloned child")
		}
	}
}

func TestCloneCssStylesheet_DeepCopy(t *testing.T) {
	modSheet := value.NewModifiableCssStylesheet(cssSpan())
	span := cssSpan()
	comment := value.NewModifiableCssComment("/* original */", span)
	modSheet.AddChild(comment)
	frozenSheet := modSheet.ToCssStylesheet()

	result, _, err := CloneCssStylesheet(frozenSheet, makeStore())
	if err != nil {
		t.Fatal(err)
	}

	// Add a child to original — should not affect clone
	modSheet.AddChild(value.NewModifiableCssComment("/* new */", span))
	if len(modSheet.Children()) != 2 {
		t.Errorf("original should have 2 children, got %d", len(modSheet.Children()))
	}
	if len(result.Children()) != 1 {
		t.Errorf("clone should still have 1 child, got %d", len(result.Children()))
	}
}

// ---------------------------------------------------------------------------
// cloneCssNodeNoExt tests
// ---------------------------------------------------------------------------

func TestCloneCssNodeNoExt_StyleRule(t *testing.T) {
	span := cssSpan()
	selList, err := makSelList(".bar")
	if err != nil {
		t.Fatal(err)
	}
	selBox := box.NewModifiableBox(selList).Seal()
	rule := value.NewModifiableCssStyleRule(selBox, span, nil, false)

	cloned, err := CloneCssNodeNoExt(rule)
	if err != nil {
		t.Fatal(err)
	}

	if sr, ok := cloned.(*value.ModifiableCssStyleRule); ok {
		if sr.Selector() == nil {
			t.Error("cloned selector should not be nil")
		}
	} else {
		t.Errorf("expected ModifiableCssStyleRule, got %T", cloned)
	}
}

func TestCloneCssNodeNoExt_MediaRule(t *testing.T) {
	span := cssSpan()
	media, err := value.NewModifiableCssMediaRule([]value.CssMediaQuery{cssMediaQuery("(min-width: 1px)")}, span)
	if err != nil {
		t.Fatal(err)
	}
	media.AddChild(value.NewModifiableCssComment("/* inside */", span))

	cloned, err := CloneCssNodeNoExt(media)
	if err != nil {
		t.Fatal(err)
	}

	if mr, ok := cloned.(*value.ModifiableCssMediaRule); ok {
		if len(mr.Children()) != 1 {
			t.Errorf("expected 1 child, got %d", len(mr.Children()))
		}
	}
}

func TestCloneCssNodeNoExt_Nested(t *testing.T) {
	span := cssSpan()
	selList, err := makSelList(".outer")
	if err != nil {
		t.Fatal(err)
	}
	selBox := box.NewModifiableBox(selList).Seal()
	outer := value.NewModifiableCssStyleRule(selBox, span, nil, false)

	innerList, err := makSelList(".inner")
	if err != nil {
		t.Fatal(err)
	}
	innerBox := box.NewModifiableBox(innerList).Seal()
	inner := value.NewModifiableCssStyleRule(innerBox, span, nil, false)
	inner.AddChild(value.NewModifiableCssComment("/* deep */", span))
	outer.AddChild(inner)

	cloned, err := CloneCssNodeNoExt(outer)
	if err != nil {
		t.Fatal(err)
	}

	if sr, ok := cloned.(*value.ModifiableCssStyleRule); ok {
		if len(sr.Children()) != 1 {
			t.Errorf("expected 1 child, got %d", len(sr.Children()))
		}
	}
}

func TestCloneCssNodeNoExt_DeepCopy(t *testing.T) {
	span := cssSpan()
	selList, err := makSelList(".test")
	if err != nil {
		t.Fatal(err)
	}
	selBox := box.NewModifiableBox(selList).Seal()
	rule := value.NewModifiableCssStyleRule(selBox, span, nil, false)
	comment := value.NewModifiableCssComment("/* original */", span)
	rule.AddChild(comment)

	cloned, err := CloneCssNodeNoExt(rule)
	if err != nil {
		t.Fatal(err)
	}

	// Add child to original — should not affect clone
	rule.AddChild(value.NewModifiableCssComment("/* new */", span))
	if len(rule.Children()) != 2 {
		t.Errorf("original should have 2 children, got %d", len(rule.Children()))
	}
	if sr, ok := cloned.(*value.ModifiableCssStyleRule); ok {
		if len(sr.Children()) != 1 {
			t.Errorf("clone should have 1 child, got %d", len(sr.Children()))
		}
	}
}
