package value

import "testing"

func TestUnaryOperatorSyntax(t *testing.T) {
	tests := []struct {
		op     UnaryOperator
		syntax string
	}{
		{UnaryOperatorPlus, "+"},
		{UnaryOperatorMinus, "-"},
		{UnaryOperatorDivide, "/"},
		{UnaryOperatorNot, "not"},
	}

	for _, tt := range tests {
		if got := tt.op.OperatorSyntax(); got != tt.syntax {
			t.Errorf("OperatorSyntax() = %q, want %q", got, tt.syntax)
		}
	}
}

func TestUnaryOperatorString(t *testing.T) {
	tests := []struct {
		op   UnaryOperator
		name string
	}{
		{UnaryOperatorPlus, "plus"},
		{UnaryOperatorMinus, "minus"},
		{UnaryOperatorDivide, "divide"},
		{UnaryOperatorNot, "not"},
	}

	for _, tt := range tests {
		if got := tt.op.String(); got != tt.name {
			t.Errorf("String() = %q, want %q", got, tt.name)
		}
	}
}
