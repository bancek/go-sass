// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/selector.dart

import (
	"fmt"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
)

// WarnLogger receives the bogus-combinator deprecation warning emitted by
// AssertNotBogus. It is satisfied by the host logger passed into custom
// Sass functions, the only callers allowed to trigger this warning.
type WarnLogger interface {
	WarnDeprecation(message string, deprecation *deprecation.Deprecation) error
}

// Selector is a node in the abstract syntax tree for a selector.
//
// This selector tree is mostly plain CSS, but also may contain a
// ParentSelector or a PlaceholderSelector.
//
// Selectors have structural equality semantics.
type Selector interface {
	sasscommon.AstNode
	IsSelector()
	IsInvisible() bool
	IsUseless() bool
	IsBogus() bool
	ContainsParentSelector() (bool, error)
	IsInvisibleOtherThanBogusCombinators() bool
	IsBogusOtherThanLeadingCombinator() bool
	AssertNotBogus(name *string, warn WarnLogger) error
	HashCode() int
	AcceptVoid(v SelectorVisitor[struct{}]) (struct{}, error)
	AcceptBool(v SelectorVisitor[bool]) (bool, error)
	AcceptParentSelector(v SelectorVisitor[*ParentSelector]) (*ParentSelector, error)
	String() (string, error)
}

// SelectorBase holds the source span shared by every selector node. Concrete
// selectors embed it anonymously so Span and the default specificity promote
// automatically.
type SelectorBase struct {
	span sasscommon.FileSpan
}

// NewSelectorBase creates the shared span storage for a selector node.
func NewSelectorBase(span sasscommon.FileSpan) SelectorBase {
	return SelectorBase{span: span}
}

// Span returns the source span where this selector was written.
func (s SelectorBase) Span() (sasscommon.FileSpan, error) { return s.span, nil }

// Specificity returns the default simple-selector specificity of 1000.
// Specificity is measured in base 1000: type selectors count 1, ID selectors
// count 1000 squared, and the universal selector counts 0.
func (s SelectorBase) Specificity() int { return 1000 }

// selectorAssertNotBogus warns through warn when s is not valid CSS (for
// example a stray or doubled combinator kept for backwards compatibility).
// name, when non-nil, is the custom-function variable the selector came from.
// The warning points at https://sass-lang.com/d/bogus-combinators and will
// become a hard error in Dart Sass 2.0.0.
func selectorAssertNotBogus(s Selector, name *string, warn WarnLogger) error {
	if s.IsBogus() {
		prefix := ""
		if name != nil {
			prefix = "$" + *name + ": "
		}
		str, err := s.String()
		if err != nil {
			return err
		}
		return warn.WarnDeprecation(
			fmt.Sprintf("%s%s is not valid CSS.\nThis will be an error in Dart Sass 2.0.0.\n\nMore info: https://sass-lang.com/d/bogus-combinators", prefix, str),
			deprecation.BogusCombinators,
		)
	}
	return nil
}
