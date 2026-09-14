package value

import (
	"testing"

	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscommon"
)

func TestArgumentListEmpty(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("()"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 2)
	al := NewArgumentListEmpty(span)

	if !al.IsEmpty() {
		t.Error("empty argument list should be empty")
	}
	if al.Rest != nil {
		t.Error("empty argument list rest should be nil")
	}
}

func TestArgumentListPositionalOnly(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("(1, 2)"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 6)
	args := NewArgumentList(
		[]Expression{
			NewBooleanExpression(true, sasscommon.NewSimpleFileSpan(fs, 1, 2)),
			NewBooleanExpression(false, sasscommon.NewSimpleFileSpan(fs, 4, 5)),
		},
		orderedmap.New[string, Expression](),
		nil,
		span,
		nil, nil,
	)

	if args.IsEmpty() {
		t.Error("should not be empty")
	}
}

func TestArgumentListString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("()"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 2)
	al := NewArgumentListEmpty(span)

	got, err := al.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "()" {
		t.Errorf("String() = %q, want '()'", got)
	}
}

func TestArgumentListStringPositional(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("(true)"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 6)
	args := NewArgumentList(
		[]Expression{
			NewBooleanExpression(true, sasscommon.NewSimpleFileSpan(fs, 1, 5)),
		},
		orderedmap.New[string, Expression](),
		nil,
		span,
		nil, nil,
	)

	got, err := args.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "(true)" {
		t.Errorf("String() = %q, want '(true)'", got)
	}
}

func TestArgumentListStringNamed(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("($a: true)"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 10)
	named := orderedmap.New[string, Expression]()
	named.Put("a", NewBooleanExpression(true, sasscommon.NewSimpleFileSpan(fs, 6, 10)))
	args := NewArgumentList(
		nil,
		named,
		nil,
		span,
		nil, nil,
	)

	got, err := args.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "($a: true)" {
		t.Errorf("String() = %q, want '($a: true)'", got)
	}
}

func TestArgumentListStringRest(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("(true...)"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 9)
	args := NewArgumentList(
		nil,
		orderedmap.New[string, Expression](),
		nil,
		span,
		NewBooleanExpression(true, sasscommon.NewSimpleFileSpan(fs, 1, 5)),
		nil,
	)

	got, err := args.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "(true...)" {
		t.Errorf("String() = %q, want '(true...)'", got)
	}
}

func TestArgumentListStringKeywordRest(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("(true...)"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 9)
	args := NewArgumentList(
		nil,
		orderedmap.New[string, Expression](),
		nil,
		span,
		nil,
		NewBooleanExpression(true, sasscommon.NewSimpleFileSpan(fs, 1, 5)),
	)

	got, err := args.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "(true...)" {
		t.Errorf("String() = %q, want '(true...)'", got)
	}
}

func TestArgumentListParenthesizeCommaList(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("(a, b)"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 6)
	listExpr := NewListExpression(
		[]Expression{
			NewBooleanExpression(true, span),
			NewBooleanExpression(false, span),
		},
		ListSeparatorComma,
		sasscommon.NewSimpleFileSpan(fs, 1, 4),
		false, // no brackets
	)

	got, err := parenthesizeArgument(listExpr)
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("expected non-empty result")
	}
	// Should wrap in parens since it's an unbracketed comma list with 2+ elements
	if got[0] != '(' || got == "" {
		t.Errorf("expected parenthesized, got %q", got)
	}
}

func TestArgumentListParenthesizeNonList(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("true"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 4)

	got, err := parenthesizeArgument(NewBooleanExpression(true, span))
	if err != nil {
		t.Fatal(err)
	}
	if got != "true" {
		t.Errorf("parenthesizeArgument = %q, want 'true'", got)
	}
}
