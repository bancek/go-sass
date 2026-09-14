// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/interface/value.dart

// ListValue is the interface implemented by SassScript values that behave as
// lists. Both [SassList] and [SassArgumentList] implement this interface.
//
// This provides a way to narrow [Value] to only list-like types, matching
// Dart's `SassList` class hierarchy where `SassArgumentList extends SassList`.
//
// Matches Dart: SassList (as the visitList parameter type)
type ListValue interface {
	Value
}

// ValueVisitor is the interface for visitors that traverse SassScript
// values, with one method per value type and a result type parameter
// selecting what the traversal produces. Error returns exist because no
// visit is guaranteed infallible.
//
// Matches Dart: ValueVisitor (lib/src/visitor/interface/value.dart)
type ValueVisitor[T any] interface {
	// VisitBoolean visits a boolean value.
	VisitBoolean(*SassBoolean) (T, error)
	// VisitNull visits the null value.
	VisitNull() (T, error)
	// VisitString visits a string value.
	VisitString(*SassString) (T, error)
	// VisitNumber visits a number value.
	VisitNumber(SassNumber) (T, error)
	// VisitColor visits a color value.
	VisitColor(*SassColor) (T, error)
	// VisitList visits a list value (either list-like type behind
	// ListValue).
	VisitList(ListValue) (T, error)
	// VisitMap visits a map value.
	VisitMap(*SassMap) (T, error)
	// VisitCalculation visits a calculation value.
	VisitCalculation(*SassCalculation) (T, error)
	// VisitFunction visits a first-class function value.
	VisitFunction(*SassFunction) (T, error)
	// VisitMixin visits a first-class mixin value.
	VisitMixin(*SassMixin) (T, error)
}
