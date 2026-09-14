// Copyright 2025 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/boolean_operator.dart

// BooleanOperator is a binary boolean operation in a @supports condition or
// an if() condition operation. Plain CSS supports only conjunctions (and)
// and disjunctions (or).
//
// Matches Dart: BooleanOperator enum.
type BooleanOperator int

const (
	// BooleanOperatorAnd is the conjunction operator, `and`.
	BooleanOperatorAnd BooleanOperator = iota
	// BooleanOperatorOr is the disjunction operator, `or`.
	BooleanOperatorOr
)

var booleanOperatorNames = [...]string{
	BooleanOperatorAnd: "and",
	BooleanOperatorOr:  "or",
}

func (op BooleanOperator) String() string { return booleanOperatorNames[op] }
