// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement.dart

import "fmt"

// Statement is a statement in a Sass syntax tree.
//
// It is the root interface of every rule, declaration, and comment node.
// The Accept methods dispatch to the matching StatementVisitor method by
// result shape: AcceptValue for value-producing traversals, AcceptBool for
// predicate searches, and AcceptVoid for effectful traversals.
type Statement interface {
	SassNode
	IsStatement()
	AcceptValue(v StatementVisitor[Value]) (Value, error)
	AcceptBool(v StatementVisitor[bool]) (bool, error)
	AcceptVoid(v StatementVisitor[struct{}]) (struct{}, error)
}

// statementChildString converts a Statement child to its string
// representation, handling both String() string and String() (string, error)
// method signatures with error propagation.
func statementChildString(child Statement) (string, error) {
	if s, ok := any(child).(interface{ String() (string, error) }); ok {
		return s.String()
	}
	if s, ok := any(child).(interface{ String() string }); ok {
		return s.String(), nil
	}
	return fmt.Sprintf("%v", child), nil
}
