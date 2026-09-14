package extend

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/value"
)

func TestDefaultExtensionStoreIsEmpty(t *testing.T) {
	store := NewDefaultExtensionStore()
	if !store.IsEmpty() {
		t.Error("new store should be empty")
	}
}

func TestDefaultExtensionStoreIsEmptyAfterAddExtension(t *testing.T) {
	store := NewDefaultExtensionStore()
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	err = store.AddExtension(
		&value.SelectorList{Components: []*value.ComplexSelector{extender}},
		target,
		extendRule(false),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if store.IsEmpty() {
		t.Error("store should not be empty after adding extension")
	}
}

func TestDefaultExtensionStoreSimpleSelectors(t *testing.T) {
	store := NewDefaultExtensionStore()
	complex, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	sel, err := selectorList(complex)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.AddSelector(sel, nil)
	if err != nil {
		t.Fatal(err)
	}

	selectors := store.SimpleSelectors()
	hasA := false
	for _, s := range selectors {
		if cs, ok := s.(*value.ClassSelector); ok && cs.Name == "a" {
			hasA = true
			break
		}
	}
	if !hasA {
		t.Error("should have class selector 'a'")
	}
}

func TestDefaultExtensionStoreAddSelectorNoExtensions(t *testing.T) {
	store := NewDefaultExtensionStore()
	complex, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	sel, err := selectorList(complex)
	if err != nil {
		t.Fatal(err)
	}

	result, err := store.AddSelector(sel, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Value() == nil {
		t.Fatal("result.Value() should not be nil")
	}
	if !result.Value().Components[0].Equals(complex) {
		t.Error("result should contain the original selector")
	}
}

func TestDefaultExtensionStoreAddSelectorWithExtension(t *testing.T) {
	store := NewDefaultExtensionStore()

	// .a { @extend .b; }
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	err = store.AddExtension(
		&value.SelectorList{Components: []*value.ComplexSelector{extender}},
		target,
		extendRule(false),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	// .b { color: red; }
	complexB, err := complexSelector(class("b"))
	if err != nil {
		t.Fatal(err)
	}
	selB, err := selectorList(complexB)
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.AddSelector(selB, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Should contain both .b and .a
	foundB := false
	foundA := false
	for _, c := range result.Value().Components {
		for _, comp := range c.Components {
			for _, simple := range comp.Selector.Components {
				if cs, ok := simple.(*value.ClassSelector); ok {
					if cs.Name == "b" {
						foundB = true
					}
					if cs.Name == "a" {
						foundA = true
					}
				}
			}
		}
	}
	if !foundB {
		t.Error("expected .b in result")
	}
	if !foundA {
		t.Error("expected .a in result")
	}
}

func TestDefaultExtensionStoreRetroactive(t *testing.T) {
	store := NewDefaultExtensionStore()

	// Register .b first
	complexB, err := complexSelector(class("b"))
	if err != nil {
		t.Fatal(err)
	}
	selB, err := selectorList(complexB)
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.AddSelector(selB, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Value() == nil {
		t.Fatal("result.Value() should not be nil")
	}
	if result.Value().Components[0].Equals(complexB) {
		// good - only .b at this point
	} else {
		t.Error("should only contain .b before extension is added")
	}

	// Now add extension .a extends .b
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	err = store.AddExtension(
		&value.SelectorList{Components: []*value.ComplexSelector{extender}},
		target,
		extendRule(false),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	// The box should have been updated to include .a
	if result.Value() == nil {
		t.Fatal("result.Value() should not be nil")
	}
	foundA := false
	for _, c := range result.Value().Components {
		for _, comp := range c.Components {
			for _, simple := range comp.Selector.Components {
				if cs, ok := simple.(*value.ClassSelector); ok && cs.Name == "a" {
					foundA = true
				}
			}
		}
	}
	if !foundA {
		t.Error("retroactive extension should add .a to the selector")
	}
}

func TestDefaultExtensionStoreChainedExtension(t *testing.T) {
	store := NewDefaultExtensionStore()

	// .a extends .b — .a is now an extender for .b
	extenderA, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	err = store.AddExtension(
		&value.SelectorList{Components: []*value.ComplexSelector{extenderA}},
		class("b"),
		extendRule(false),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	// .c extends .a — .a is now a TARGET of .c extension.
	// Since .a is in extensionsByExtender (as extender for .b),
	// the chain propagates: .c should also extend .b
	extenderC, err := complexSelector(class("c"))
	if err != nil {
		t.Fatal(err)
	}
	err = store.AddExtension(
		&value.SelectorList{Components: []*value.ComplexSelector{extenderC}},
		class("a"),
		extendRule(false),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	// .b { color: red; } — should get both .a (direct) and .c (chained)
	complexB, err := complexSelector(class("b"))
	if err != nil {
		t.Fatal(err)
	}
	selB, err := selectorList(complexB)
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.AddSelector(selB, nil)
	if err != nil {
		t.Fatal(err)
	}

	foundB := false
	foundA := false
	foundC := false
	for _, c := range result.Value().Components {
		for _, comp := range c.Components {
			for _, simple := range comp.Selector.Components {
				if cs, ok := simple.(*value.ClassSelector); ok {
					if cs.Name == "b" {
						foundB = true
					}
					if cs.Name == "a" {
						foundA = true
					}
					if cs.Name == "c" {
						foundC = true
					}
				}
			}
		}
	}
	if !foundB {
		t.Error("expected .b in result")
	}
	if !foundA {
		t.Error("expected .a in result from direct extension")
	}
	if !foundC {
		t.Error("expected .c in result from chained extension (.c extends .a extends .b)")
	}
}

func TestDefaultExtensionStoreAddExtensionUselessExtender(t *testing.T) {
	store := NewDefaultExtensionStore()

	// Create a truly useless extender: ComplexSelector with >1 leading combinator
	c1, err := value.NewCompoundSelector([]value.SimpleSelector{class("a")}, sasscommon.BogusSpan)
	if err != nil {
		t.Fatal(err)
	}
	cc := value.NewComplexSelectorComponent(c1, nil, sasscommon.BogusSpan)
	comb1 := sasscommon.NewCssValue(value.CombinatorChild, sasscommon.BogusSpan)
	comb2 := sasscommon.NewCssValue(value.CombinatorChild, sasscommon.BogusSpan)
	useless, err := value.NewComplexSelector(
		[]sasscommon.CssValue[value.Combinator]{comb1, comb2},
		[]*value.ComplexSelectorComponent{cc},
		sasscommon.BogusSpan,
		false,
	)
	if err != nil {
		t.Fatal(err)
	}

	if !useless.IsUseless() {
		t.Fatalf("extender should be useless, IsUseless=%v", useless.IsUseless())
	}

	err = store.AddExtension(
		&value.SelectorList{Components: []*value.ComplexSelector{useless}},
		class("b"),
		extendRule(false),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	// The target entry is registered in extensions even if extender is useless,
	// but no actual extension should be created for the useless extender.
	found := store.ExtensionsWhereTarget(func(s value.SimpleSelector) bool {
		if cs, ok := s.(*value.ClassSelector); ok {
			return cs.Name == "b"
		}
		return false
	})
	if len(found) != 0 {
		t.Errorf("useless extender should not create extensions, got %d", len(found))
	}
}

func TestDefaultExtensionStoreComplexExtension(t *testing.T) {
	store := NewDefaultExtensionStore()

	// .x .y { @extend .z; }
	extender, err := complexSelectorWithCombinator(class("x"), class("y"), value.CombinatorChild)
	if err != nil {
		t.Fatal(err)
	}
	err = store.AddExtension(
		&value.SelectorList{Components: []*value.ComplexSelector{extender}},
		class("z"),
		extendRule(false),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	// .z { color: red; }
	complexZ, err := complexSelector(class("z"))
	if err != nil {
		t.Fatal(err)
	}
	selZ, err := selectorList(complexZ)
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.AddSelector(selZ, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Should contain .z and .x .y
	if len(result.Value().Components) < 2 {
		t.Errorf("expected at least 2 complex selectors, got %d", len(result.Value().Components))
	}
}

func TestDefaultExtensionStoreClone(t *testing.T) {
	store := NewDefaultExtensionStore()

	// Register a selector
	complexA, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	sel, err := selectorList(complexA)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.AddSelector(sel, nil)
	if err != nil {
		t.Fatal(err)
	}

	cloned, oldToNew := store.Clone()
	if cloned == nil {
		t.Fatal("cloned store should not be nil")
	}
	if oldToNew == nil {
		t.Fatal("oldToNew should not be nil (should be empty map)")
	}

	// Both should reflect the same initial state
	if cloned.IsEmpty() != store.IsEmpty() {
		t.Error("clone and original should have same isEmpty")
	}
}

func TestDefaultExtensionStoreAddExtensions(t *testing.T) {
	store1 := NewDefaultExtensionStore()
	store2 := NewDefaultExtensionStore()

	// store2 has .a { @extend .b; }
	extenderA, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	err = store2.AddExtension(
		&value.SelectorList{Components: []*value.ComplexSelector{extenderA}},
		class("b"),
		extendRule(false),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Merge store2 into store1
	err = store1.AddExtensions([]ExtensionStore{store2})
	if err != nil {
		t.Fatal(err)
	}
	if store1.IsEmpty() {
		t.Error("store1 should not be empty after addExtensions")
	}

	// Now adding .b should get extended by .a
	complexB, err := complexSelector(class("b"))
	if err != nil {
		t.Fatal(err)
	}
	selB, err := selectorList(complexB)
	if err != nil {
		t.Fatal(err)
	}
	result, err := store1.AddSelector(selB, nil)
	if err != nil {
		t.Fatal(err)
	}

	foundA := false
	for _, c := range result.Value().Components {
		for _, comp := range c.Components {
			for _, simple := range comp.Selector.Components {
				if cs, ok := simple.(*value.ClassSelector); ok && cs.Name == "a" {
					foundA = true
				}
			}
		}
	}
	if !foundA {
		t.Error("expected .a in result after merging stores")
	}
}

func TestDefaultExtensionStoreAddExtensionsPrivatePlaceholder(t *testing.T) {
	store1 := NewDefaultExtensionStore()
	store2 := NewDefaultExtensionStore()

	// store2 has .a { @extend %-private; }
	extenderA, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	privateTarget := value.NewPlaceholderSelector("_private", sasscommon.BogusSpan)
	err = store2.AddExtension(
		&value.SelectorList{Components: []*value.ComplexSelector{extenderA}},
		privateTarget,
		extendRule(false),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Merge store2 into store1 — private placeholder extension should be skipped
	err = store1.AddExtensions([]ExtensionStore{store2})
	if err != nil {
		t.Fatal(err)
	}
	if !store1.IsEmpty() {
		t.Error("store1 should remain empty — private placeholder extensions should be filtered")
	}
}

func TestDefaultExtensionStoreExtensionsWhereTarget(t *testing.T) {
	store := NewDefaultExtensionStore()

	// .a { @extend .b; }
	extenderA, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	err = store.AddExtension(
		&value.SelectorList{Components: []*value.ComplexSelector{extenderA}},
		class("b"),
		extendRule(false),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	found := store.ExtensionsWhereTarget(func(s value.SimpleSelector) bool {
		if cs, ok := s.(*value.ClassSelector); ok {
			return cs.Name == "b"
		}
		return false
	})
	if len(found) == 0 {
		t.Error("should find extension for .b")
	}
}

func TestDefaultExtensionStoreExtensionsWhereTargetOptional(t *testing.T) {
	store := NewDefaultExtensionStore()

	// .a { @extend .b !optional; }
	extenderA, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	err = store.AddExtension(
		&value.SelectorList{Components: []*value.ComplexSelector{extenderA}},
		class("b"),
		extendRule(true),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	found := store.ExtensionsWhereTarget(func(s value.SimpleSelector) bool { return true })
	if len(found) != 0 {
		t.Error("optional extensions should not be returned by ExtensionsWhereTarget")
	}
}

func TestDefaultExtensionStoreExtendStatic(t *testing.T) {
	// selector-extend(.c, .a, .b) — add .a wherever .b appears in .c
	complexSource, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	source, err := selectorList(complexSource)
	if err != nil {
		t.Fatal(err)
	}

	complexTarget, err := complexSelector(class("b"))
	if err != nil {
		t.Fatal(err)
	}
	targets, err := selectorList(complexTarget)
	if err != nil {
		t.Fatal(err)
	}

	complexSelectorB, err := complexSelector(class("b"))
	if err != nil {
		t.Fatal(err)
	}
	sel, err := selectorList(complexSelectorB)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ExtendStatic(sel, source, targets, sasscommon.BogusSpan)
	if err != nil {
		t.Fatal(err)
	}

	foundA := false
	for _, c := range result.Components {
		for _, comp := range c.Components {
			for _, simple := range comp.Selector.Components {
				if cs, ok := simple.(*value.ClassSelector); ok && cs.Name == "a" {
					foundA = true
				}
			}
		}
	}
	if !foundA {
		t.Error("expected .a in extendStatic result")
	}
}

func TestDefaultExtensionStoreReplaceStatic(t *testing.T) {
	// selector-replace(.b, .a, .b) — replace .b with .a
	complexSource, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	source, err := selectorList(complexSource)
	if err != nil {
		t.Fatal(err)
	}

	complexTarget, err := complexSelector(class("b"))
	if err != nil {
		t.Fatal(err)
	}
	targets, err := selectorList(complexTarget)
	if err != nil {
		t.Fatal(err)
	}

	complexSelectorB, err := complexSelector(class("b"))
	if err != nil {
		t.Fatal(err)
	}
	sel, err := selectorList(complexSelectorB)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ReplaceStatic(sel, source, targets, sasscommon.BogusSpan)
	if err != nil {
		t.Fatal(err)
	}

	// Replace mode: .b should be replaced by .a, original .b should not remain
	hasB := false
	hasA := false
	for _, c := range result.Components {
		for _, comp := range c.Components {
			for _, simple := range comp.Selector.Components {
				if cs, ok := simple.(*value.ClassSelector); ok {
					if cs.Name == "b" {
						hasB = true
					}
					if cs.Name == "a" {
						hasA = true
					}
				}
			}
		}
	}
	if hasB {
		t.Error("replace mode should not keep original .b")
	}
	if !hasA {
		t.Error("expected .a in replaceStatic result")
	}
}

func TestDefaultExtensionStoreMediaContext(t *testing.T) {
	store := NewDefaultExtensionStore()

	// .a { @extend .b; } inside @media screen
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	mediaContext := []*value.CssMediaQuery{
		value.NewCssMediaQueryType(&[]string{"screen"}[0], nil, nil),
	}
	err = store.AddExtension(
		&value.SelectorList{Components: []*value.ComplexSelector{extender}},
		target,
		extendRule(false),
		mediaContext,
	)
	if err != nil {
		t.Fatal(err)
	}

	// .b inside @media screen (same context) should get extended by .a
	complexB, err := complexSelector(class("b"))
	if err != nil {
		t.Fatal(err)
	}
	selB, err := selectorList(complexB)
	if err != nil {
		t.Fatal(err)
	}
	result, err := store.AddSelector(selB, mediaContext)
	if err != nil {
		t.Fatal(err)
	}

	foundA := false
	for _, c := range result.Value().Components {
		for _, comp := range c.Components {
			for _, simple := range comp.Selector.Components {
				if cs, ok := simple.(*value.ClassSelector); ok && cs.Name == "a" {
					foundA = true
				}
			}
		}
	}
	if !foundA {
		t.Error("expected .a from extension inside same media context")
	}
}

func TestDefaultExtensionStoreSealBox(t *testing.T) {
	store := NewDefaultExtensionStore()

	complex, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	sel, err := selectorList(complex)
	if err != nil {
		t.Fatal(err)
	}

	sealed, err := store.AddSelector(sel, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sealed == nil {
		t.Fatal("sealed box should not be nil")
	}
	if sealed.Value() == nil {
		t.Fatal("sealed box value should not be nil")
	}
}
