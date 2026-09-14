// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

import "github.com/bancek/go-sass/sasscommon"

// dart-source: lib/src/ast/selector/attribute.dart

// AttributeOperator defines the semantics of an AttributeSelector value match.
type AttributeOperator int

const (
	// AttributeOperatorEqual matches when the attribute value exactly equals
	// the given value: `=`.
	AttributeOperatorEqual AttributeOperator = iota // =
	// AttributeOperatorInclude matches when the attribute value is a
	// whitespace-separated list of words containing the given value: `~=`.
	AttributeOperatorInclude // ~=
	// AttributeOperatorDash matches when the attribute value is exactly the
	// given value or starts with it followed by a dash: `|=`.
	AttributeOperatorDash // |=
	// AttributeOperatorPrefix matches when the attribute value begins with
	// the given value: `^=`.
	AttributeOperatorPrefix // ^=
	// AttributeOperatorSuffix matches when the attribute value ends with the
	// given value: `$=`.
	AttributeOperatorSuffix // $=
	// AttributeOperatorSubstring matches when the attribute value contains
	// the given value: `*=`.
	AttributeOperatorSubstring // *=
)

// String returns the operator's token text, or "?" for an unknown operator.
func (op AttributeOperator) String() string {
	switch op {
	case AttributeOperatorEqual:
		return "="
	case AttributeOperatorInclude:
		return "~="
	case AttributeOperatorDash:
		return "|="
	case AttributeOperatorPrefix:
		return "^="
	case AttributeOperatorSuffix:
		return "$="
	case AttributeOperatorSubstring:
		return "*="
	default:
		return "?"
	}
}

// AttributeSelector selects for elements with a given attribute, optionally
// asserting something about the attribute's value.
type AttributeSelector struct {
	base SelectorBase
	// Name is the attribute being selected for.
	Name QualifiedName
	// Op defines how Value is compared. Nil matches any element carrying the
	// attribute regardless of value, and is nil exactly when Value is nil.
	Op *AttributeOperator
	// Value asserts something about the attribute's value, with precise
	// semantics defined by Op. Nil matches any value.
	Value *string
	// Modifier controls how the match is processed, such as the selectors-4
	// case-sensitivity flag. It is always nil when Op is nil.
	Modifier *string
}

// NewAttributeSelector creates a selector matching any element with an
// attribute of the given name, regardless of value.
func NewAttributeSelector(name QualifiedName, span sasscommon.FileSpan) *AttributeSelector {
	return &AttributeSelector{
		base: NewSelectorBase(span),
		Name: name,
	}
}

// NewAttributeSelectorWithOperator creates a selector matching an element
// with an attribute named name whose value matches value under op's
// semantics, with an optional case-sensitivity modifier.
func NewAttributeSelectorWithOperator(
	name QualifiedName,
	op AttributeOperator,
	value string,
	span sasscommon.FileSpan,
	modifier *string,
) *AttributeSelector {
	return &AttributeSelector{
		base:     NewSelectorBase(span),
		Name:     name,
		Op:       &op,
		Value:    &value,
		Modifier: modifier,
	}
}

// Span returns the source span where this selector was written.
func (s *AttributeSelector) Span() (sasscommon.FileSpan, error)         { return s.base.Span() }
func (s *AttributeSelector) IsAstNode()                                 {}
func (s *AttributeSelector) IsSelector()                                {}
func (s *AttributeSelector) ContainsParentSelector() (bool, error)      { return false, nil }
func (s *AttributeSelector) IsInvisibleOtherThanBogusCombinators() bool { return s.IsInvisible() }
func (s *AttributeSelector) IsBogus() bool                              { return false }
func (s *AttributeSelector) IsBogusOtherThanLeadingCombinator() bool    { return s.IsBogus() }
func (s *AttributeSelector) IsUseless() bool                            { return false }

// AssertNotBogus warns through warn when this selector is not valid CSS.
// An attribute selector is never bogus, so this only forwards to the shared
// helper for uniformity.
func (s *AttributeSelector) AssertNotBogus(name *string, warn WarnLogger) error {
	return selectorAssertNotBogus(s, name, warn)
}

// Specificity returns the default simple-selector specificity of 1000.
func (s *AttributeSelector) Specificity() int { return s.base.Specificity() }

func (s *AttributeSelector) IsSimpleSelector() {}

// AcceptVoid dispatches to VisitAttributeSelector on v.
func (s *AttributeSelector) AcceptVoid(v SelectorVisitor[struct{}]) (struct{}, error) {
	return v.VisitAttributeSelector(s)
}

// AcceptBool dispatches to VisitAttributeSelector on v.
func (s *AttributeSelector) AcceptBool(v SelectorVisitor[bool]) (bool, error) {
	return v.VisitAttributeSelector(s)
}

// AcceptParentSelector dispatches to VisitAttributeSelector on v.
func (s *AttributeSelector) AcceptParentSelector(v SelectorVisitor[*ParentSelector]) (*ParentSelector, error) {
	return v.VisitAttributeSelector(s)
}

func (s *AttributeSelector) HasComplicatedSuperselectorSemantics() bool {
	return false
}

// AddSuffix always fails: an attribute selector cannot take an identifier
// suffix, so nesting resolution reports a script error.
func (s *AttributeSelector) AddSuffix(suffix string) (SimpleSelector, error) {
	return nil, &sasscommon.SassScriptException{Message: "attribute selector cannot have a suffix"}
}

// String renders this selector in inspect mode.
func (s *AttributeSelector) String() (string, error) {
	return SerializeSelector(s, true)
}

// HashCode folds the attribute name, operator, value, and modifier together.
// Spans never contribute to selector hashes; value and modifier only apply
// when an operator is present.
func (s *AttributeSelector) HashCode() int {
	h := hashCombine(stringHashCode(s.Name.Name), hashPtr(s.Name.Namespace))
	if s.Op != nil {
		h = hashCombine(h, int(*s.Op))
		if s.Value != nil {
			h = hashCombine(h, stringHashCode(*s.Value))
			if s.Modifier != nil {
				h = hashCombine(h, stringHashCode(*s.Modifier))
			}
		}
	}
	return h
}

// IsSuperselector delegates to the shared base-class check: attribute
// selectors only match structurally equal selectors or subselector pseudos
// wrapping them.
func (s *AttributeSelector) IsSuperselector(other SimpleSelector) (bool, error) {
	return simpleIsSuperselector(s, other)
}
