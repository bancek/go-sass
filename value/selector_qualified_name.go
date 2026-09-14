// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/selector/qualified_name.dart

// QualifiedName is a CSS qualified name: an identifier with an optional
// namespace prefix, as defined by https://www.w3.org/TR/css3-namespace/#css-qnames.
type QualifiedName struct {
	// Name is the identifier name.
	Name string

	// Namespace is the optional namespace prefix.
	//
	// If nil, Name belongs to the default namespace.
	// If "", Name belongs to no namespace.
	// If "*", Name belongs to any namespace.
	Namespace *string
}

// NewQualifiedName creates a name in the default namespace (a nil Namespace).
func NewQualifiedName(name string) QualifiedName {
	return QualifiedName{Name: name}
}

// NewQualifiedNameWithNamespace creates a name in the given namespace. A nil
// namespace means the default namespace, "" means no namespace, and "*"
// means any namespace.
func NewQualifiedNameWithNamespace(name string, namespace *string) QualifiedName {
	return QualifiedName{Name: name, Namespace: namespace}
}

// Equal reports whether two qualified names match, comparing the namespace
// by value rather than by pointer: a nil namespace only equals nil.
func (q QualifiedName) Equal(other QualifiedName) bool {
	if q.Name != other.Name {
		return false
	}
	if q.Namespace == nil && other.Namespace == nil {
		return true
	}
	if q.Namespace == nil || other.Namespace == nil {
		return false
	}
	return *q.Namespace == *other.Namespace
}

// String renders the name with its namespace prefix (`ns|name`), or the bare
// name when no namespace is set.
func (q QualifiedName) String() string {
	if q.Namespace == nil {
		return q.Name
	}
	return *q.Namespace + "|" + q.Name
}

// HashCode folds Name and Namespace together. The span plays no part: like
// Dart's hashCode, only the qualified name itself contributes.
func (q QualifiedName) HashCode() int {
	return hashCombine(stringHashCode(q.Name), hashPtr(q.Namespace))
}
