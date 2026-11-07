// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/selector/qualified_name.dart

import (
	"github.com/bancek/go-sass/util"
)

// SelectorNamespace is a namespace for a QualifiedName or UniversalSelector.
//
// Matches Dart: SelectorNamespace (sealed class)
type SelectorNamespace interface {
	isSelectorNamespace()
}

// DefaultSelectorNamespace is the default namespace for the document.
//
// Matches Dart: DefaultSelectorNamespace
type DefaultSelectorNamespace struct{}

// AnySelectorNamespace matches any possible namespace.
//
// Matches Dart: AnySelectorNamespace
type AnySelectorNamespace struct{}

// NoSelectorNamespace is the absence of a namespace.
//
// Matches Dart: NoSelectorNamespace
type NoSelectorNamespace struct{}

// NamedSelectorNamespace is an explicitly-named namespace.
//
// Matches Dart: NamedSelectorNamespace
type NamedSelectorNamespace struct {
	Name string
}

func (DefaultSelectorNamespace) isSelectorNamespace() {}
func (AnySelectorNamespace) isSelectorNamespace()     {}
func (NoSelectorNamespace) isSelectorNamespace()      {}
func (NamedSelectorNamespace) isSelectorNamespace()   {}

var (
	// DefaultSelectorNS is the default namespace for the document.
	DefaultSelectorNS SelectorNamespace = DefaultSelectorNamespace{}
	// AnySelectorNS matches any possible namespace.
	AnySelectorNS SelectorNamespace = AnySelectorNamespace{}
	// NoSelectorNS is the absence of a namespace.
	NoSelectorNS SelectorNamespace = NoSelectorNamespace{}
)

// NamedNamespace creates an explicitly-named selector namespace.
func NamedNamespace(name string) SelectorNamespace {
	return NamedSelectorNamespace{Name: name}
}

func IsDefaultSelectorNamespace(namespace SelectorNamespace) bool {
	if namespace == nil {
		return true
	}
	_, ok := namespace.(DefaultSelectorNamespace)
	return ok
}

func isAnySelectorNamespace(namespace SelectorNamespace) bool {
	_, ok := namespace.(AnySelectorNamespace)
	return ok
}

// selectorNamespaceEqual reports whether two namespaces are equal.
func selectorNamespaceEqual(a, b SelectorNamespace) bool {
	if a == nil {
		a = DefaultSelectorNS
	}
	if b == nil {
		b = DefaultSelectorNS
	}
	switch a := a.(type) {
	case DefaultSelectorNamespace:
		_, ok := b.(DefaultSelectorNamespace)
		return ok
	case AnySelectorNamespace:
		_, ok := b.(AnySelectorNamespace)
		return ok
	case NoSelectorNamespace:
		_, ok := b.(NoSelectorNamespace)
		return ok
	case NamedSelectorNamespace:
		other, ok := b.(NamedSelectorNamespace)
		return ok && a.Name == other.Name
	default:
		return false
	}
}

// selectorNamespaceHashCode returns the Dart-compatible hash of a namespace.
func selectorNamespaceHashCode(namespace SelectorNamespace) int {
	if namespace == nil {
		return 0
	}
	switch namespace := namespace.(type) {
	case DefaultSelectorNamespace:
		return 0
	case AnySelectorNamespace:
		return 1
	case NoSelectorNamespace:
		return 2
	case NamedSelectorNamespace:
		return stringHashCode(namespace.Name)
	default:
		return 0
	}
}

// selectorNamespaceString renders a namespace prefix without the trailing |.
func selectorNamespaceString(namespace SelectorNamespace) (string, error) {
	if namespace == nil {
		return "", nil
	}
	switch namespace := namespace.(type) {
	case DefaultSelectorNamespace, NoSelectorNamespace:
		return "", nil
	case AnySelectorNamespace:
		return "*", nil
	case NamedSelectorNamespace:
		return util.ToCssIdentifier(namespace.Name)
	default:
		return "", nil
	}
}

// QualifiedName is a CSS qualified name: an identifier with an optional
// namespace prefix, as defined by https://www.w3.org/TR/css3-namespace/#css-qnames.
type QualifiedName struct {
	// Name is the identifier name.
	Name string

	// Namespace is the namespace prefix.
	Namespace SelectorNamespace
}

// NewQualifiedName creates a name in the default namespace.
func NewQualifiedName(name string) QualifiedName {
	return QualifiedName{Name: name, Namespace: DefaultSelectorNS}
}

// NewQualifiedNameWithNamespace creates a name in the given namespace. A nil
// namespace means the default namespace.
func NewQualifiedNameWithNamespace(name string, namespace SelectorNamespace) QualifiedName {
	if namespace == nil {
		namespace = DefaultSelectorNS
	}
	return QualifiedName{Name: name, Namespace: namespace}
}

// Equal reports whether two qualified names match.
func (q QualifiedName) Equal(other QualifiedName) bool {
	return q.Name == other.Name && selectorNamespaceEqual(q.Namespace, other.Namespace)
}

// String renders the name with its namespace prefix, or the bare name in the
// default namespace.
func (q QualifiedName) String() string {
	name, err := util.ToCssIdentifier(q.Name)
	if err != nil {
		name = q.Name
	}
	switch q.Namespace.(type) {
	case nil, DefaultSelectorNamespace:
		return name
	case NoSelectorNamespace:
		return "|" + name
	case AnySelectorNamespace:
		return "*|" + name
	}
	namespace, err := selectorNamespaceString(q.Namespace)
	if err != nil {
		namespace = ""
	}
	return namespace + "|" + name
}

// HashCode folds Name and Namespace together. The span plays no part: like
// Dart's hashCode, only the qualified name itself contributes.
func (q QualifiedName) HashCode() int {
	return hashCombine(stringHashCode(q.Name), selectorNamespaceHashCode(q.Namespace))
}
