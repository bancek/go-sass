package extend

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/value"
)

func TestMergeExtensionsSameExtenderAndTarget(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	left := NewExtension(extender, target, sasscommon.BogusSpan, nil, false)
	right := NewExtension(extender, target, sasscommon.BogusSpan, nil, false)

	merged, err := MergeExtensions(left, right)
	if err != nil {
		t.Fatal(err)
	}
	if !merged.IsMerged() {
		t.Error("should be merged")
	}
}

func TestMergeExtensionsDifferentExtender(t *testing.T) {
	extender1, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	extender2, err := complexSelector(class("c"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	left := NewExtension(extender1, target, sasscommon.BogusSpan, nil, false)
	right := NewExtension(extender2, target, sasscommon.BogusSpan, nil, false)

	_, err = MergeExtensions(left, right)
	if err == nil {
		t.Error("should error on different extenders")
	}
}

func TestMergeExtensionsDifferentTarget(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	left := NewExtension(extender, class("b"), sasscommon.BogusSpan, nil, false)
	right := NewExtension(extender, class("c"), sasscommon.BogusSpan, nil, false)

	_, err = MergeExtensions(left, right)
	if err == nil {
		t.Error("should error on different targets")
	}
}

func TestMergeExtensionsCompatibleMediaContexts(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	mediaContext := []*value.CssMediaQuery{
		value.NewCssMediaQueryType(&[]string{"screen"}[0], nil, nil),
	}
	left := NewExtension(extender, target, sasscommon.BogusSpan, mediaContext, false)
	right := NewExtension(extender, target, sasscommon.BogusSpan, mediaContext, false)

	merged, err := MergeExtensions(left, right)
	if err != nil {
		t.Fatal(err)
	}
	if !merged.IsMerged() {
		t.Error("should be merged")
	}
	if merged.MediaContext() == nil {
		t.Error("merged should have media context")
	}
}

func TestMergeExtensionsIncompatibleMediaContexts(t *testing.T) {
	span := realSpan()
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	mc1 := []*value.CssMediaQuery{
		value.NewCssMediaQueryType(&[]string{"screen"}[0], nil, nil),
	}
	mc2 := []*value.CssMediaQuery{
		value.NewCssMediaQueryType(&[]string{"print"}[0], nil, nil),
	}
	left := NewExtension(extender, target, span, mc1, false)
	right := NewExtension(extender, target, span, mc2, false)

	merged, err := MergeExtensions(left, right)
	if err == nil && merged != nil {
		t.Error("should error on incompatible media contexts")
	}
}

func TestMergeExtensionsRightOptionalNoMedia(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	left := NewExtension(extender, target, sasscommon.BogusSpan, nil, false)
	right := NewExtension(extender, target, sasscommon.BogusSpan, nil, true)

	merged, err := MergeExtensions(left, right)
	if err != nil {
		t.Fatal(err)
	}
	// Right is optional and has no media context -> return left
	if !merged.Extender().Selector.Equals(extender) {
		t.Error("should return left when right is optional with no media context")
	}
}

func TestMergeExtensionsLeftOptionalNoMedia(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	left := NewExtension(extender, target, sasscommon.BogusSpan, nil, true)
	right := NewExtension(extender, target, sasscommon.BogusSpan, nil, false)

	merged, err := MergeExtensions(left, right)
	if err != nil {
		t.Fatal(err)
	}
	// Left is optional and has no media context -> return right
	if !merged.Extender().Selector.Equals(extender) {
		t.Error("should return right when left is optional with no media context")
	}
}

func TestUnmergeExtensionsBase(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	ext := NewExtension(extender, target, sasscommon.BogusSpan, nil, false)

	unmerged := UnmergeExtensions(ext)
	if len(unmerged) != 1 {
		t.Errorf("unmerged count = %d, want 1", len(unmerged))
	}
	if !unmerged[0].Target().Equals(target) {
		t.Error("unmerged target mismatch")
	}
}

func TestUnmergeExtensionsMerged(t *testing.T) {
	extender, err := complexSelector(class("a"))
	if err != nil {
		t.Fatal(err)
	}
	target := class("b")
	left := NewExtension(extender, target, sasscommon.BogusSpan, nil, false)
	right := NewExtension(extender, target, sasscommon.BogusSpan, nil, false)
	merged, err := MergeExtensions(left, right)
	if err != nil {
		t.Fatal(err)
	}

	unmerged := UnmergeExtensions(merged)
	if len(unmerged) != 2 {
		t.Errorf("unmerged count = %d, want 2", len(unmerged))
	}
	for _, ext := range unmerged {
		if !ext.Target().Equals(target) {
			t.Error("unmerged target mismatch")
		}
	}
}
