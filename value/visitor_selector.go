// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/serialize.dart (visitAttributeSelector,
// visitClassSelector, visitComplexSelector, visitCompoundSelector,
// visitIDSelector, visitSelectorList, visitParentSelector,
// visitPlaceholderSelector, visitPseudoSelector, visitTypeSelector,
// visitUniversalSelector, and _writeCombinators section)

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// ---- SelectorVisitor implementations ----

// VisitSelectorList writes the visible complex selectors separated by
// commas. In normal mode invisible complexes are skipped; in inspect mode
// all are kept. A complex flagged with a line break restarts on its own
// indented line, otherwise the comma is followed by an optional space.
//
// Matches Dart: _SerializeVisitor.visitSelectorList
func (sv *SerializeVisitor) VisitSelectorList(list *SelectorList) (struct{}, error) {
	complexes := list.Components
	if !sv.inspect {
		var filtered []*ComplexSelector
		for _, c := range complexes {
			if !c.IsInvisible() {
				filtered = append(filtered, c)
			}
		}
		complexes = filtered
	}
	first := true
	for _, c := range complexes {
		if first {
			first = false
		} else {
			_ = sv.sb.WriteByte(',')
			if c.LineBreak {
				sv.writeLineFeed()
				sv.writeIndentation()
			} else {
				sv.writeOptionalSpace()
			}
		}
		if _, err := sv.VisitComplexSelector(c); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

// VisitComplexSelector writes a complex selector: leading combinators,
// then each compound with its trailing combinators. Components are spaced
// apart; in compressed mode the space is dropped when combinators already
// separate them.
//
// Matches Dart: _SerializeVisitor.visitComplexSelector
func (sv *SerializeVisitor) VisitComplexSelector(complex *ComplexSelector) (struct{}, error) {
	if err := sv.writeCombinators(complex.LeadingCombinators); err != nil {
		return struct{}{}, err
	}
	if len(complex.LeadingCombinators) > 0 && len(complex.Components) > 0 {
		sv.writeOptionalSpace()
	}
	for i, component := range complex.Components {
		if _, err := sv.VisitCompoundSelector(component.Selector); err != nil {
			return struct{}{}, err
		}
		if len(component.Combinators) > 0 {
			sv.writeOptionalSpace()
		}
		if err := sv.writeCombinators(component.Combinators); err != nil {
			return struct{}{}, err
		}
		if i != len(complex.Components)-1 &&
			(!sv.isCompressed() || len(component.Combinators) == 0) {
			_ = sv.sb.WriteByte(' ')
		}
	}
	return struct{}{}, nil
}

// writeCombinators writes combinators back to back, spaced in expanded
// mode and glued in compressed mode.
//
// Matches Dart: _SerializeVisitor._writeCombinators
func (sv *SerializeVisitor) writeCombinators(combinators []sasscommon.CssValue[Combinator]) error {
	sep := " "
	if sv.isCompressed() {
		sep = ""
	}
	return writeBetween(sv.sb, combinators, sep, func(cv sasscommon.CssValue[Combinator]) error {
		_, _ = sv.sb.WriteString(cv.Value.String())
		return nil
	})
}

// VisitCompoundSelector writes each simple selector in turn. A compound
// whose components were all optimized away matches everything, so the
// universal selector is emitted rather than leaving the output empty.
//
// Matches Dart: _SerializeVisitor.visitCompoundSelector
func (sv *SerializeVisitor) VisitCompoundSelector(compound *CompoundSelector) (struct{}, error) {
	start := sv.sb.Len()
	for _, simple := range compound.Components {
		if _, err := simple.AcceptVoid(sv); err != nil {
			return struct{}{}, err
		}
	}
	// If we emit an empty compound, it's because all of the components got
	// optimized out because they match all selectors, so we just emit the
	// universal selector.
	if sv.sb.Len() == start {
		_ = sv.sb.WriteByte('*')
	}
	return struct{}{}, nil
}

// VisitAttributeSelector writes an [name operator value modifier]
// selector. Values that parse as identifiers print bare (except --
// prefixed names, which IE11 rejects unquoted and so stay quoted);
// anything else is quoted, and the modifier follows after a separating
// space.
//
// Matches Dart: _SerializeVisitor.visitAttributeSelector
func (sv *SerializeVisitor) VisitAttributeSelector(attr *AttributeSelector) (struct{}, error) {
	_ = sv.sb.WriteByte('[')
	_, _ = sv.sb.WriteString(attr.Name.String())
	if attr.Op != nil {
		_, _ = sv.sb.WriteString(attr.Op.String())
		if attr.Value != nil {
			isIdent := IsIdentifier(*attr.Value) &&
				!strings.HasPrefix(*attr.Value, "--")
			if isIdent {
				_, _ = sv.sb.WriteString(*attr.Value)
				if attr.Modifier != nil {
					_ = sv.sb.WriteByte(' ')
				}
			} else {
				sv.visitQuotedString(*attr.Value)
				if attr.Modifier != nil {
					sv.writeOptionalSpace()
				}
			}
		}
	}
	if attr.Modifier != nil {
		_, _ = sv.sb.WriteString(*attr.Modifier)
	}
	_ = sv.sb.WriteByte(']')
	return struct{}{}, nil
}

// VisitClassSelector writes a .name class selector.
//
// Matches Dart: _SerializeVisitor.visitClassSelector
func (sv *SerializeVisitor) VisitClassSelector(klass *ClassSelector) (struct{}, error) {
	_ = sv.sb.WriteByte('.')
	_, _ = sv.sb.WriteString(klass.Name)
	return struct{}{}, nil
}

// VisitIDSelector writes a #name ID selector.
//
// Matches Dart: _SerializeVisitor.visitIDSelector
func (sv *SerializeVisitor) VisitIDSelector(id *IDSelector) (struct{}, error) {
	_ = sv.sb.WriteByte('#')
	_, _ = sv.sb.WriteString(id.Name)
	return struct{}{}, nil
}

// VisitParentSelector writes the & parent selector with any suffix glued
// on directly.
//
// Matches Dart: _SerializeVisitor.visitParentSelector
func (sv *SerializeVisitor) VisitParentSelector(parent *ParentSelector) (struct{}, error) {
	_ = sv.sb.WriteByte('&')
	if parent.Suffix != nil {
		_, _ = sv.sb.WriteString(*parent.Suffix)
	}
	return struct{}{}, nil
}

// VisitPlaceholderSelector writes a %name placeholder selector.
//
// Matches Dart: _SerializeVisitor.visitPlaceholderSelector
func (sv *SerializeVisitor) VisitPlaceholderSelector(placeholder *PlaceholderSelector) (struct{}, error) {
	_ = sv.sb.WriteByte('%')
	_, _ = sv.sb.WriteString(placeholder.Name)
	return struct{}{}, nil
}

// VisitPseudoSelector writes a :name or ::name pseudo with its optional
// argument and/or nested selector list. A :not() around an invisible
// (placeholder-only) selector matches everything, so it emits nothing,
// mirroring the universal-selector equivalence.
//
// Matches Dart: _SerializeVisitor.visitPseudoSelector
func (sv *SerializeVisitor) VisitPseudoSelector(pseudo *PseudoSelector) (struct{}, error) {
	// :not(%a) is semantically identical to *.
	if pseudo.Name == "not" && pseudo.Selector != nil {
		if sl, ok := pseudo.Selector.(*SelectorList); ok && sl.IsInvisible() {
			return struct{}{}, nil
		}
	}
	_ = sv.sb.WriteByte(':')
	if !pseudo.IsSyntacticClass {
		_ = sv.sb.WriteByte(':')
	}
	_, _ = sv.sb.WriteString(pseudo.Name)
	if pseudo.Argument == nil && pseudo.Selector == nil {
		return struct{}{}, nil
	}
	_ = sv.sb.WriteByte('(')
	if pseudo.Argument != nil {
		_, _ = sv.sb.WriteString(*pseudo.Argument)
		if pseudo.Selector != nil {
			_ = sv.sb.WriteByte(' ')
		}
	}
	if pseudo.Selector != nil {
		if _, err := pseudo.Selector.AcceptVoid(sv); err != nil {
			return struct{}{}, err
		}
	}
	_ = sv.sb.WriteByte(')')
	return struct{}{}, nil
}

// VisitTypeSelector writes a type (element-name) selector.
//
// Matches Dart: _SerializeVisitor.visitTypeSelector
func (sv *SerializeVisitor) VisitTypeSelector(typeSel *TypeSelector) (struct{}, error) {
	_, _ = sv.sb.WriteString(typeSel.Name.String())
	return struct{}{}, nil
}

// VisitUniversalSelector writes a * universal selector with its optional
// namespace prefix (namespace|*).
//
// Matches Dart: _SerializeVisitor.visitUniversalSelector
func (sv *SerializeVisitor) VisitUniversalSelector(universal *UniversalSelector) (struct{}, error) {
	if universal.Namespace != nil {
		_, _ = sv.sb.WriteString(*universal.Namespace)
		_ = sv.sb.WriteByte('|')
	}
	_ = sv.sb.WriteByte('*')
	return struct{}{}, nil
}
