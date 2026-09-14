package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestBinaryOperationExpressionConstruction(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("1 + 2"), nil)
	_ = sasscommon.NewSimpleFileSpan(fs, 0, 5)
	left := NewNumberExpression(1, sasscommon.NewSimpleFileSpan(fs, 0, 1), nil)
	right := NewNumberExpression(2, sasscommon.NewSimpleFileSpan(fs, 4, 5), nil)
	expr := NewBinaryOperationExpression(BinaryOperatorPlus, left, right)

	if expr.Operator != BinaryOperatorPlus {
		t.Errorf("Operator = %v, want Plus", expr.Operator)
	}
	if expr.AllowsSlash() {
		t.Error("AllowsSlash should be false for non-slash binary op")
	}
}

func TestBinaryOperationExpressionSlash(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("1 / 2"), nil)
	_ = sasscommon.NewSimpleFileSpan(fs, 0, 5)
	left := NewNumberExpression(1, sasscommon.NewSimpleFileSpan(fs, 0, 1), nil)
	right := NewNumberExpression(2, sasscommon.NewSimpleFileSpan(fs, 4, 5), nil)
	expr := NewBinaryOperationExpressionSlash(left, right)

	if expr.Operator != BinaryOperatorDividedBy {
		t.Errorf("Operator = %v, want DividedBy", expr.Operator)
	}
	if !expr.AllowsSlash() {
		t.Error("AllowsSlash should be true for slash binary op")
	}
}

func TestBinaryOperationExpressionString(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("1 + 2"), nil)
	_ = sasscommon.NewSimpleFileSpan(fs, 0, 5)
	left := NewNumberExpression(1, sasscommon.NewSimpleFileSpan(fs, 0, 1), nil)
	right := NewNumberExpression(2, sasscommon.NewSimpleFileSpan(fs, 4, 5), nil)
	expr := NewBinaryOperationExpression(BinaryOperatorPlus, left, right)

	got, err := expr.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "1 + 2" {
		t.Errorf("String() = %q, want '1 + 2'", got)
	}
}

func TestBinaryOperationExpressionSpan(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("1 + 2"), nil)
	_ = sasscommon.NewSimpleFileSpan(fs, 0, 5)
	left := NewNumberExpression(1, sasscommon.NewSimpleFileSpan(fs, 0, 1), nil)
	right := NewNumberExpression(2, sasscommon.NewSimpleFileSpan(fs, 4, 5), nil)
	expr := NewBinaryOperationExpression(BinaryOperatorPlus, left, right)

	got, err := expr.Span()
	if err != nil {
		t.Fatal(err)
	}
	_ = got
	if got == nil {
		t.Fatal("Span should not be nil")
	}
}

func TestBinaryOperationExpressionOperatorSpan(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("1 + 2"), nil)
	left := NewNumberExpression(1, sasscommon.NewSimpleFileSpan(fs, 0, 1), nil)
	right := NewNumberExpression(2, sasscommon.NewSimpleFileSpan(fs, 4, 5), nil)
	expr := NewBinaryOperationExpression(BinaryOperatorPlus, left, right)

	got, err := expr.OperatorSpan()
	if err != nil {
		t.Fatal(err)
	}
	_ = got
	if got == nil {
		t.Fatal("OperatorSpan should not be nil")
	}
}

func TestBinaryOperationExpressionSourceInterpolation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("1 + 2"), nil)
	left := NewNumberExpression(1, sasscommon.NewSimpleFileSpan(fs, 0, 1), nil)
	right := NewNumberExpression(2, sasscommon.NewSimpleFileSpan(fs, 4, 5), nil)
	expr := NewBinaryOperationExpression(BinaryOperatorPlus, left, right)
	if expr.SourceInterpolation() != nil {
		t.Error("SourceInterpolation() should be nil")
	}
}
