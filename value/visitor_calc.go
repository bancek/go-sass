// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/serialize.dart (visitCalculation section:
// _writeCalculationValue, _writeCalculationUnits,
// _parenthesizeCalculationRhs)

import (
	"math"
)

// writeCalculationValue writes one calculation argument. Finite numbers
// with complex units print their value plus the unit expression; other
// finite numbers serialize normally. Infinities and NaN gain the
// infinity/-infinity/NaN spellings with their units. Operations recurse
// into both sides, parenthesizing where precedence demands it and keeping
// spaces around the operator in expanded output (and always around
// low-precedence plus/minus, where CSS requires them). Any other value
// serializes through the main visitor.
//
// Matches Dart: _SerializeVisitor._writeCalculationValue
func (sv *SerializeVisitor) writeCalculationValue(arg any) error {
	switch v := arg.(type) {
	case SassNumber:
		if !math.IsInf(v.NumValue(), 0) && !math.IsNaN(v.NumValue()) {
			if v.HasComplexUnits() {
				sv.writeNumber(v.NumValue())
				if len(v.NumNumeratorUnits()) > 0 {
					_, _ = sv.sb.WriteString(v.NumNumeratorUnits()[0])
					sv.writeCalcUnits(v.NumNumeratorUnits()[1:], v.NumDenominatorUnits(), false)
				} else {
					sv.writeCalcUnits(nil, v.NumDenominatorUnits(), false)
				}
			} else {
				_, err := v.AcceptVoid(sv)
				return err
			}
		} else {
			switch {
			case math.IsInf(v.NumValue(), 1):
				_, _ = sv.sb.WriteString("infinity")
			case math.IsInf(v.NumValue(), -1):
				_, _ = sv.sb.WriteString("-infinity")
			case math.IsNaN(v.NumValue()):
				_, _ = sv.sb.WriteString("NaN")
			}
			sv.writeCalcUnits(v.NumNumeratorUnits(), v.NumDenominatorUnits(), false)
		}

	case *CalculationOperation:
		op := v.Operator
		prec := op.Precedence()
		leftNeedsParens := false
		if left, ok := v.Left.(*CalculationOperation); ok {
			leftNeedsParens = left.Operator.Precedence() < prec
		}
		if leftNeedsParens {
			_ = sv.sb.WriteByte('(')
		}
		if err := sv.writeCalculationValue(v.Left); err != nil {
			return err
		}
		if leftNeedsParens {
			_ = sv.sb.WriteByte(')')
		}
		ws := !sv.isCompressed() || prec == 1
		if ws {
			_ = sv.sb.WriteByte(' ')
		}
		_, _ = sv.sb.WriteString(op.Operator())
		if ws {
			_ = sv.sb.WriteByte(' ')
		}
		rightNeedsParens := parenthesizeCalcRHSRight(op, v.Right)
		if rightNeedsParens {
			_ = sv.sb.WriteByte('(')
		}
		if err := sv.writeCalculationValue(v.Right); err != nil {
			return err
		}
		if rightNeedsParens {
			_ = sv.sb.WriteByte(')')
		}

	case Value:
		_, err := v.AcceptVoid(sv)
		return err

	default:
		panic("BUG: unexpected calculation value type")
	}
	return nil
}

// parenthesizeCalcRHS reports whether a right-hand operation nested under
// outer needs parentheses: always under division, never under plus, and
// under the remaining operators when the inner operation is plus or minus.
//
// Matches Dart: _SerializeVisitor._parenthesizeCalculationRhs
func parenthesizeCalcRHS(outer, right CalculationOperator) bool {
	switch outer {
	case CalculationOperatorDividedBy:
		return true
	case CalculationOperatorPlus:
		return false
	default:
		return right == CalculationOperatorPlus || right == CalculationOperatorMinus
	}
}

// parenthesizeCalcRHSRight extends parenthesizeCalcRHS to a concrete right
// operand: operations use the operator rule, while a number divided by a
// complex- (or unit-carrying non-finite-) unit number is parenthesized so
// the units stay bound to the divisor. Anything else needs no parens. It
// is Go-only glue factoring the right-operand dispatch out of the
// operator-level rule.
//
// Matches Dart: _SerializeVisitor._writeCalculationValue
// (parenthesizeRight computation)
func parenthesizeCalcRHSRight(outer CalculationOperator, right any) bool {
	if op, ok := right.(*CalculationOperation); ok {
		return parenthesizeCalcRHS(outer, op.Operator)
	}
	if n, ok := right.(SassNumber); ok {
		if outer != CalculationOperatorDividedBy {
			return false
		}
		if math.IsInf(n.NumValue(), 0) || math.IsNaN(n.NumValue()) {
			return n.HasUnits()
		}
		return n.HasComplexUnits()
	}
	return false
}

// writeCalcUnits writes the complex numerator and denominator units beyond the
// first numerator unit as `* 1<unit>` / `/ 1<unit>`.
//
// If negative is true, the resulting unit expression gets a negative sign
// (Dart's `negative` parameter; no caller sets it yet — kept for parity).
func (sv *SerializeVisitor) writeCalcUnits(numeratorUnits, denominatorUnits []string, negative bool) {
	for _, unit := range numeratorUnits {
		sv.writeOptionalSpace()
		_ = sv.sb.WriteByte('*')
		sv.writeOptionalSpace()
		if negative {
			_ = sv.sb.WriteByte('-')
			negative = false
		}
		_ = sv.sb.WriteByte('1')
		_, _ = sv.sb.WriteString(unit)
	}
	for _, unit := range denominatorUnits {
		sv.writeOptionalSpace()
		_ = sv.sb.WriteByte('/')
		sv.writeOptionalSpace()
		if negative {
			_ = sv.sb.WriteByte('-')
			negative = false
		}
		_ = sv.sb.WriteByte('1')
		_, _ = sv.sb.WriteString(unit)
	}
}
