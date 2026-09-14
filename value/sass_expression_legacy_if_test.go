package value

import (
	"testing"

	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscommon"
)

func modernSpan() sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte("modern"), nil)
	return sasscommon.NewFileSpan(fs, 0, 6)
}

func TestLegacyIfExpressionModernSuggestionThreeArgs(t *testing.T) {
	span := modernSpan()
	cond := NewStringExpressionPlain("$cond", span, false)
	ifTrue := NewNumberExpression(1, span, nil)
	ifFalse := NewNumberExpression(2, span, nil)
	args := NewArgumentList(
		[]Expression{cond, ifTrue, ifFalse},
		orderedmap.New[string, Expression](),
		map[string]sasscommon.FileSpan{},
		span,
		nil,
		nil,
	)
	expr := NewLegacyIfExpression(args, span)
	got, err := expr.ModernSuggestion()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected non-nil")
	}
	want := "if(sass($cond): 1; else: 2)"
	if *got != want {
		t.Errorf("got %q, want %q", *got, want)
	}
}

func TestLegacyIfExpressionModernSuggestionNullElse(t *testing.T) {
	span := modernSpan()
	cond := NewStringExpressionPlain("$cond", span, false)
	ifTrue := NewNumberExpression(1, span, nil)
	ifFalse := NewNullExpression(span)
	args := NewArgumentList(
		[]Expression{cond, ifTrue, ifFalse},
		orderedmap.New[string, Expression](),
		map[string]sasscommon.FileSpan{},
		span,
		nil,
		nil,
	)
	expr := NewLegacyIfExpression(args, span)
	got, err := expr.ModernSuggestion()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected non-nil")
	}
	want := "if(sass($cond): 1)"
	if *got != want {
		t.Errorf("got %q, want %q", *got, want)
	}
}

func TestLegacyIfExpressionModernSuggestionNullIfTrue(t *testing.T) {
	span := modernSpan()
	cond := NewStringExpressionPlain("$cond", span, false)
	ifTrue := NewNullExpression(span)
	ifFalse := NewNumberExpression(2, span, nil)
	args := NewArgumentList(
		[]Expression{cond, ifTrue, ifFalse},
		orderedmap.New[string, Expression](),
		map[string]sasscommon.FileSpan{},
		span,
		nil,
		nil,
	)
	expr := NewLegacyIfExpression(args, span)
	got, err := expr.ModernSuggestion()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected non-nil")
	}
	want := "if(not sass($cond): 2)"
	if *got != want {
		t.Errorf("got %q, want %q", *got, want)
	}
}

func TestLegacyIfExpressionModernSuggestionWrongArgs(t *testing.T) {
	span := modernSpan()
	cond := NewStringExpressionPlain("$cond", span, false)
	args := NewArgumentList(
		[]Expression{cond},
		orderedmap.New[string, Expression](),
		map[string]sasscommon.FileSpan{},
		span,
		nil,
		nil,
	)
	expr := NewLegacyIfExpression(args, span)
	got, err := expr.ModernSuggestion()
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("expected nil for wrong arg count, got %q", *got)
	}
}

func TestLegacyIfExpressionModernSuggestionWithNamed(t *testing.T) {
	span := modernSpan()
	cond := NewStringExpressionPlain("$cond", span, false)
	ifTrue := NewNumberExpression(1, span, nil)
	ifFalse := NewNumberExpression(2, span, nil)
	named := orderedmap.New[string, Expression]()
	named.Put("extra", NewNumberExpression(3, span, nil))
	args := NewArgumentList(
		[]Expression{cond, ifTrue, ifFalse},
		named,
		map[string]sasscommon.FileSpan{},
		span,
		nil,
		nil,
	)
	expr := NewLegacyIfExpression(args, span)
	got, err := expr.ModernSuggestion()
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("expected nil when named args present, got %q", *got)
	}
}
