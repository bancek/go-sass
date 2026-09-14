// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/selector/simple.dart

import (
	"net/url"

	"github.com/bancek/go-sass/sasslogger"
)

// SimpleSelectorEquals compares simple selectors by structural equality,
// ignoring spans. It is used as the comparator for ordered maps keyed by
// simple selectors.
var SimpleSelectorEquals = func(a, b SimpleSelector) bool { return a.Equals(b) }

// SimpleSelector is a selector with no combinators: a class, ID, type,
// universal, attribute, placeholder, parent, or pseudo selector. Every simple
// selector carries a base-1000 specificity and participates in unification
// (the extend-algorithm half of which lives in selector_extend_functions.go,
// owned by V4a) and superselector checks.
type SimpleSelector interface {
	Selector
	IsSimpleSelector()
	Specificity() int
	HasComplicatedSuperselectorSemantics() bool
	AddSuffix(string) (SimpleSelector, error)
	IsSuperselector(SimpleSelector) (bool, error)
	Unify([]SimpleSelector) ([]SimpleSelector, error)
	Equals(SimpleSelector) bool
}

// ParseSimpleSelector parses a simple selector from contents. If url is
// non-nil it names the file contents came from; allowParent controls whether
// a parent selector is accepted; logger receives deprecation warnings and may
// be nil to use the default logger. It returns an error when parsing fails.
func ParseSimpleSelector(contents string, url *url.URL, allowParent bool, logger sasslogger.Logger) (SimpleSelector, error) {
	opts := &SelectorParserOptions{
		AllowParent: &allowParent,
		Logger:      logger,
	}
	p := NewSelectorParser(contents, url, nil, opts)
	return p.ParseSimpleSelector()
}

// simpleIsSuperselector implements the shared base-class superselector check:
// identical selectors are superselectors of each other, and a plain selector
// is a superselector of a subselector pseudo (such as :is, :matches, :where,
// :any, :nth-child, or :nth-last-child) when it is a superselector of some
// simple selector in the last compound of every complex in the pseudo's
// argument. Pseudo-specific refinements (for example ::slotted) live on the
// concrete selector types.
func simpleIsSuperselector(self, other SimpleSelector) (bool, error) {
	if self.Equals(other) {
		return true, nil
	}
	if p, ok := other.(*PseudoSelector); ok && p.IsClass {
		if list, ok := p.Selector.(*SelectorList); ok {
			normalized := p.NormalizedName
			if normalized == "is" || normalized == "matches" || normalized == "where" ||
				normalized == "any" || normalized == "nth-child" || normalized == "nth-last-child" {
				for _, complex := range list.Components {
					if len(complex.Components) == 0 {
						return false, nil
					}
					lastComp := complex.Components[len(complex.Components)-1]
					found := false
					for _, comp := range lastComp.Selector.Components {
						ok, err := self.IsSuperselector(comp)
						if err != nil {
							return false, err
						}
						if ok {
							found = true
							break
						}
					}
					if !found {
						return false, nil
					}
				}
				return true, nil
			}
		}
	}
	return false, nil
}
