package value

import "testing"

func TestBinaryOperatorName(t *testing.T) {
	tests := []struct {
		op   BinaryOperator
		name string
	}{
		{BinaryOperatorSingleEquals, "single equals"},
		{BinaryOperatorOr, "or"},
		{BinaryOperatorAnd, "and"},
		{BinaryOperatorEquals, "equals"},
		{BinaryOperatorNotEquals, "not equals"},
		{BinaryOperatorGreaterThan, "greater than"},
		{BinaryOperatorGreaterThanOrEquals, "greater than or equals"},
		{BinaryOperatorLessThan, "less than"},
		{BinaryOperatorLessThanOrEquals, "less than or equals"},
		{BinaryOperatorPlus, "plus"},
		{BinaryOperatorMinus, "minus"},
		{BinaryOperatorTimes, "times"},
		{BinaryOperatorDividedBy, "divided by"},
		{BinaryOperatorModulo, "modulo"},
	}

	for _, tt := range tests {
		if got := tt.op.Name(); got != tt.name {
			t.Errorf("Name() = %q, want %q", got, tt.name)
		}
	}
}

func TestBinaryOperatorPrecedence(t *testing.T) {
	tests := []struct {
		op   BinaryOperator
		prec int
	}{
		{BinaryOperatorSingleEquals, 0},
		{BinaryOperatorOr, 1},
		{BinaryOperatorAnd, 2},
		{BinaryOperatorEquals, 3},
		{BinaryOperatorNotEquals, 3},
		{BinaryOperatorGreaterThan, 4},
		{BinaryOperatorGreaterThanOrEquals, 4},
		{BinaryOperatorLessThan, 4},
		{BinaryOperatorLessThanOrEquals, 4},
		{BinaryOperatorPlus, 5},
		{BinaryOperatorMinus, 5},
		{BinaryOperatorTimes, 6},
		{BinaryOperatorDividedBy, 6},
		{BinaryOperatorModulo, 6},
	}

	for _, tt := range tests {
		if got := tt.op.Precedence(); got != tt.prec {
			t.Errorf("Precedence() = %d, want %d", got, tt.prec)
		}
	}
}

func TestBinaryOperatorSyntax(t *testing.T) {
	tests := []struct {
		op     BinaryOperator
		syntax string
	}{
		{BinaryOperatorSingleEquals, "="},
		{BinaryOperatorOr, "or"},
		{BinaryOperatorAnd, "and"},
		{BinaryOperatorEquals, "=="},
		{BinaryOperatorNotEquals, "!="},
		{BinaryOperatorGreaterThan, ">"},
		{BinaryOperatorGreaterThanOrEquals, ">="},
		{BinaryOperatorLessThan, "<"},
		{BinaryOperatorLessThanOrEquals, "<="},
		{BinaryOperatorPlus, "+"},
		{BinaryOperatorMinus, "-"},
		{BinaryOperatorTimes, "*"},
		{BinaryOperatorDividedBy, "/"},
		{BinaryOperatorModulo, "%"},
	}

	for _, tt := range tests {
		if got := tt.op.OperatorSyntax(); got != tt.syntax {
			t.Errorf("OperatorSyntax() = %q, want %q", got, tt.syntax)
		}
	}
}

func TestBinaryOperatorIsAssociative(t *testing.T) {
	tests := []struct {
		op          BinaryOperator
		associative bool
	}{
		{BinaryOperatorSingleEquals, false},
		{BinaryOperatorOr, true},
		{BinaryOperatorAnd, true},
		{BinaryOperatorEquals, false},
		{BinaryOperatorNotEquals, false},
		{BinaryOperatorGreaterThan, false},
		{BinaryOperatorGreaterThanOrEquals, false},
		{BinaryOperatorLessThan, false},
		{BinaryOperatorLessThanOrEquals, false},
		{BinaryOperatorPlus, true},
		{BinaryOperatorMinus, false},
		{BinaryOperatorTimes, true},
		{BinaryOperatorDividedBy, false},
		{BinaryOperatorModulo, false},
	}

	for _, tt := range tests {
		if got := tt.op.IsAssociative(); got != tt.associative {
			t.Errorf("IsAssociative() = %v, want %v", got, tt.associative)
		}
	}
}

func TestBinaryOperatorString(t *testing.T) {
	if got := BinaryOperatorPlus.String(); got != "plus" {
		t.Errorf("String() = %q, want %q", got, "plus")
	}
}
