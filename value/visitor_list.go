// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/serialize.dart (visitList/visitMap section:
// _separatorString, _elementNeedsParens, _writeMapElement)

// listSepStr returns the separator text for lists with the given
// separator: the style's comma spelling, a tight or spaced slash, a single
// space, or nothing for undecided single-element lists (which never need a
// separator even though the lookup runs eagerly).
//
// Matches Dart: _SerializeVisitor._separatorString
func (sv *SerializeVisitor) listSepStr(sep ListSeparator) string {
	switch sep {
	case ListSeparatorComma:
		return sv.commaSep()
	case ListSeparatorSlash:
		if sv.isCompressed() {
			return "/"
		}
		return " / "
	case ListSeparatorSpace:
		return " "
	default:
		return ""
	}
}

// elementNeedsParens reports whether a nested unbracketed multi-element
// list needs parentheses inside an outer list with the given separator: a
// comma element inside comma lists, a comma or slash element inside slash
// lists, and any decided-separator element inside space lists. It keeps
// inspect output unambiguous about nesting.
//
// Matches Dart: _SerializeVisitor._elementNeedsParens
func elementNeedsParens(separator ListSeparator, v Value) bool {
	var l *SassList
	switch val := v.(type) {
	case *SassList:
		l = val
	case *SassArgumentList:
		l = &val.list
	default:
		return false
	}
	if len(l.Contents) <= 1 || l.HasBrackets() {
		return false
	}
	switch separator {
	case ListSeparatorComma:
		return l.Separator() == ListSeparatorComma
	case ListSeparatorSlash:
		return l.Separator() == ListSeparatorComma || l.Separator() == ListSeparatorSlash
	default:
		return l.Separator() != ListSeparatorUndecided
	}
}

// writeMapElement writes one map key or value, parenthesizing unbracketed
// comma lists so the entry boundaries stay unambiguous.
//
// Matches Dart: _SerializeVisitor._writeMapElement
func (sv *SerializeVisitor) writeMapElement(v Value) error {
	var l *SassList
	switch val := v.(type) {
	case *SassList:
		l = val
	case *SassArgumentList:
		l = &val.list
	}
	needsParens := l != nil && l.Separator() == ListSeparatorComma && !l.HasBrackets()
	if needsParens {
		_ = sv.sb.WriteByte('(')
	}
	if _, err := v.AcceptVoid(sv); err != nil {
		return err
	}
	if needsParens {
		_ = sv.sb.WriteByte(')')
	}
	return nil
}
