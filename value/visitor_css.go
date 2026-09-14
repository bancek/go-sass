// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/serialize.dart (visitCssStylesheet,
// visitCssComment, visitCssAtRule, visitCssMediaRule, visitCssImport,
// visitCssKeyframeBlock, visitCssStyleRule, visitCssSupportsRule,
// visitCssDeclaration, and the declaration/comment indentation helpers
// _writeFoldedValue, _writeReindentedValue, _minimumIndentation,
// _writeWithIndent)

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// ---- CssVisitor implementations ----

// VisitCssStylesheet writes each visible child in order, separating them
// with semicolons, line feeds, and a blank line after group ends. A comment
// that trails the previous child on the same line is separated by a space
// instead of a line feed. The final semicolon is emitted only in expanded
// mode.
//
// Matches Dart: _SerializeVisitor.visitCssStylesheet
func (sv *SerializeVisitor) VisitCssStylesheet(node *CssStylesheet) (struct{}, error) {
	var previous CssNode
	for _, child := range node.Children() {
		if sv.isInvisible(child) {
			continue
		}
		if previous != nil {
			if sv.requiresSemicolon(previous) {
				_ = sv.sb.WriteByte(';')
			}
			isTrailing, err := sv.isTrailingComment(child, previous)
			if err != nil {
				return struct{}{}, err
			}
			if isTrailing {
				sv.writeOptionalSpace()
			} else {
				sv.writeLineFeed()
				if previous.IsGroupEnd() {
					sv.writeLineFeed()
				}
			}
		}
		previous = child
		if _, err := sv.visitCssNode(child); err != nil {
			return struct{}{}, err
		}
	}
	if previous != nil && sv.requiresSemicolon(previous) && !sv.isCompressed() {
		_ = sv.sb.WriteByte(';')
	}
	return struct{}{}, nil
}

// sourceMapRe matches sourceMappingURL/sourceURL comments, which the
// serializer always drops. It ports the RegExp Dart builds inline in
// visitCssComment.
var sourceMapRe = regexp.MustCompile(`(?s)/\*# source(Mapping)?URL=`)

// VisitCssComment writes a comment bound to its source span. Compressed
// output keeps only preserved (/*!*/) comments; sourceMappingURL comments
// are always ignored. Multi-line comments are re-indented relative to the
// current depth, clamped to the comment's own start column.
//
// Matches Dart: _SerializeVisitor.visitCssComment
func (sv *SerializeVisitor) VisitCssComment(node CssComment) (struct{}, error) {
	nodeSpan, err := node.Span()
	if err != nil {
		return struct{}{}, err
	}
	return struct{}{}, sv.sb.ForSpan(nodeSpan, func() error {
		if sv.isCompressed() && !node.IsPreserved() {
			return nil
		}
		if sourceMapRe.MatchString(node.Text()) {
			return nil
		}
		if minIndent := sv.minimumIndentation(node.Text()); minIndent != nil {
			innerLoc, err := nodeSpan.StartLocation()
			if err != nil {
				return err
			}
			actualMin := int(math.Min(float64(*minIndent), float64(innerLoc.Column)))
			sv.writeIndentation()
			sv.writeWithIndent(node.Text(), actualMin)
		} else {
			sv.writeIndentation()
			_, _ = sv.sb.WriteString(node.Text())
		}
		return nil
	})
}

// VisitCssAtRule writes an at-rule head (@name plus its value) bound to
// the node's span, then serializes any child block after an optional
// space. Childless rules emit the head alone.
//
// Matches Dart: _SerializeVisitor.visitCssAtRule
func (sv *SerializeVisitor) VisitCssAtRule(node CssAtRule) (struct{}, error) {
	sv.writeIndentation()
	nodeSpan, err := node.Span()
	if err != nil {
		return struct{}{}, err
	}
	if err := sv.sb.ForSpan(nodeSpan, func() error {
		_ = sv.sb.WriteByte('@')
		_, _ = sv.sb.WriteString(node.Name().Value)
		if v := node.Value(); v != nil {
			_ = sv.sb.WriteByte(' ')
			_, _ = sv.sb.WriteString(v.Value)
		}
		return nil
	}); err != nil {
		return struct{}{}, err
	}
	if !node.IsChildless() {
		sv.writeOptionalSpace()
		return sv.visitChildren(node)
	}
	return struct{}{}, nil
}

// VisitCssMediaRule writes an @media head with its comma-separated query
// list, then the child block. In compressed mode the space after @media is
// dropped unless the first query carries a modifier or type, or is a lone
// "(not ...)" condition that would otherwise glue to the keyword.
//
// Matches Dart: _SerializeVisitor.visitCssMediaRule
func (sv *SerializeVisitor) VisitCssMediaRule(node CssMediaRule) (struct{}, error) {
	sv.writeIndentation()
	nodeSpan, err := node.Span()
	if err != nil {
		return struct{}{}, err
	}
	if err := sv.sb.ForSpan(nodeSpan, func() error {
		_, _ = sv.sb.WriteString("@media")
		queries := node.Queries()
		if len(queries) > 0 {
			first := queries[0]
			if !sv.isCompressed() ||
				first.Modifier != nil ||
				first.Type != nil ||
				(len(first.Conditions) == 1 && strings.HasPrefix(first.Conditions[0], "(not ")) {
				_ = sv.sb.WriteByte(' ')
			}
			if err := writeBetween(sv.sb, queries, sv.commaSep(), sv.writeMediaQuery); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return struct{}{}, err
	}
	sv.writeOptionalSpace()
	return sv.visitChildren(node)
}

// writeMediaQuery writes one media query: an optional modifier, an
// optional type joined to its conditions with " and ", and otherwise the
// conditions joined by the conjunction ("and"/"or", tightened in
// compressed mode). A lone "(not ...)" condition is rewritten into the
// "not ..." prefix form.
//
// Matches Dart: _SerializeVisitor._visitMediaQuery
func (sv *SerializeVisitor) writeMediaQuery(q CssMediaQuery) error {
	if q.Modifier != nil {
		_, _ = sv.sb.WriteString(*q.Modifier)
		_ = sv.sb.WriteByte(' ')
	}
	if q.Type != nil {
		_, _ = sv.sb.WriteString(*q.Type)
		if len(q.Conditions) > 0 {
			_, _ = sv.sb.WriteString(" and ")
		}
	}
	if len(q.Conditions) == 1 && strings.HasPrefix(q.Conditions[0], "(not ") {
		_, _ = sv.sb.WriteString("not ")
		cond := q.Conditions[0]
		_, _ = sv.sb.WriteString(cond[len("(not ") : len(cond)-1])
	} else {
		operator := "and"
		if !q.Conjunction {
			operator = "or"
		}
		sep := ""
		if sv.isCompressed() {
			sep = operator + " "
		} else {
			sep = " " + operator + " "
		}
		return writeBetween(sv.sb, q.Conditions, sep, func(s string) error { sv.sb.WriteString(s); return nil })
	}
	return nil
}

// VisitCssImport writes an @import with its URL span-mapped and any
// trailing modifiers. The URL keeps its own span mapping nested inside the
// rule's span, matching Dart's _for(node.url, ...) nesting.
//
// Matches Dart: _SerializeVisitor.visitCssImport
func (sv *SerializeVisitor) VisitCssImport(node CssImport) (struct{}, error) {
	sv.writeIndentation()
	nodeSpan, err := node.Span()
	if err != nil {
		return struct{}{}, err
	}
	return struct{}{}, sv.sb.ForSpan(nodeSpan, func() error {
		_, _ = sv.sb.WriteString("@import")
		sv.writeOptionalSpace()
		// Dart: _for(node.url, () => _writeImportUrl(node.url.value))
		urlSpan, err := node.URL().Span()
		if err != nil {
			return err
		}
		if err := sv.sb.ForSpan(urlSpan, func() error {
			sv.writeImportURL(node.URL().Value)
			return nil
		}); err != nil {
			return err
		}
		if mod := node.Modifiers(); mod != nil {
			sv.writeOptionalSpace()
			_, _ = sv.sb.WriteString(mod.Value)
		}
		return nil
	})
}

// writeImportURL writes an import URL, unwrapping url(...) in compressed
// mode so the surrounding call can be dropped and the whitespace between
// @import and the URL removed. Bare unwrapped contents gain quotes to stay
// valid; already-quoted contents are emitted as-is.
//
// Matches Dart: _SerializeVisitor._writeImportUrl
func (sv *SerializeVisitor) writeImportURL(url string) {
	if !sv.isCompressed() || url[0] != 'u' {
		_, _ = sv.sb.WriteString(url)
		return
	}
	urlContents := url[4 : len(url)-1]
	maybeQuote := urlContents[0]
	if maybeQuote == '\'' || maybeQuote == '"' {
		_, _ = sv.sb.WriteString(urlContents)
	} else {
		sv.visitQuotedString(urlContents)
	}
}

// VisitCssKeyframeBlock writes a keyframe selector list bound to its
// span, then the declaration block after an optional space.
//
// Matches Dart: _SerializeVisitor.visitCssKeyframeBlock
func (sv *SerializeVisitor) VisitCssKeyframeBlock(node CssKeyframeBlock) (struct{}, error) {
	sv.writeIndentation()
	selSpan, err := node.Selector().Span()
	if err != nil {
		return struct{}{}, err
	}
	if err := sv.sb.ForSpan(selSpan, func() error {
		return writeBetween(sv.sb, node.Selector().Value, sv.commaSep(), func(s string) error { _, err := sv.sb.WriteString(s); return err })
	}); err != nil {
		return struct{}{}, err
	}
	sv.writeOptionalSpace()
	return sv.visitChildren(node)
}

// VisitCssStyleRule writes the selector (span-mapped) followed by its
// child block after an optional space.
//
// Matches Dart: _SerializeVisitor.visitCssStyleRule
func (sv *SerializeVisitor) VisitCssStyleRule(node CssStyleRule) (struct{}, error) {
	sv.writeIndentation()
	selSpan, err := node.Selector().Span()
	if err != nil {
		return struct{}{}, err
	}
	if err := sv.sb.ForSpan(selSpan, func() error {
		_, err := node.Selector().AcceptVoid(sv)
		return err
	}); err != nil {
		return struct{}{}, err
	}
	sv.writeOptionalSpace()
	return sv.visitChildren(node)
}

// VisitCssSupportsRule writes an @supports head with its condition, then
// the child block. In compressed mode the space after @supports is dropped
// when the condition opens with a parenthesis.
//
// Matches Dart: _SerializeVisitor.visitCssSupportsRule
func (sv *SerializeVisitor) VisitCssSupportsRule(node CssSupportsRule) (struct{}, error) {
	sv.writeIndentation()
	nodeSpan, err := node.Span()
	if err != nil {
		return struct{}{}, err
	}
	if err := sv.sb.ForSpan(nodeSpan, func() error {
		_, _ = sv.sb.WriteString("@supports")
		if !(sv.isCompressed() && len(node.Condition().Value) > 0 && node.Condition().Value[0] == '(') {
			_ = sv.sb.WriteByte(' ')
		}
		_, _ = sv.sb.WriteString(node.Condition().Value)
		return nil
	}); err != nil {
		return struct{}{}, err
	}
	sv.writeOptionalSpace()
	return sv.visitChildren(node)
}

// VisitCssDeclaration writes a declaration head (name plus colon) and its
// value. Custom properties parsed as plain text are folded (compressed) or
// re-indented (expanded) to preserve their whitespace-sensitive contents;
// SassScript values serialize through the value visitor with the
// declaration's map span, converting script-level failures into spanned
// stylesheet errors at the value's span.
//
// Matches Dart: _SerializeVisitor.visitCssDeclaration
func (sv *SerializeVisitor) VisitCssDeclaration(node CssDeclaration) (struct{}, error) {
	sv.writeIndentation()
	nameSpan, err := node.Name().Span()
	if err != nil {
		return struct{}{}, err
	}
	if err := sv.sb.ForSpan(nameSpan, func() error {
		_, _ = sv.sb.WriteString(node.Name().Value)
		return nil
	}); err != nil {
		return struct{}{}, err
	}
	_ = sv.sb.WriteByte(':')
	if !node.ParsedAsSassScript() {
		valSpan, err := node.Value().Span()
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, sv.sb.ForSpan(valSpan, func() error {
			if sv.isCompressed() {
				return sv.writeFoldedValue(node)
			} else {
				return sv.writeReindentedValue(node)
			}
		})
	}
	sv.writeOptionalSpace()
	val := node.Value().Value
	if v, ok := val.(Value); ok {
		valueSpan, err := node.Value().Span()
		if err != nil {
			return struct{}{}, err
		}
		if err := sv.sb.ForSpan(node.ValueSpanForMap(), func() error {
			_, err := v.AcceptVoid(sv)
			return err
		}); err != nil {
			if msse, ok := errors.AsType[*sasscommon.MultiSpanSassScriptException](err); ok {
				return struct{}{}, sasscommon.ThrowWithTrace(
					&sasscommon.MultiSpanSassException{
						Message:      msse.Message,
						Span:         valueSpan,
						PrimaryLabel: msse.PrimaryLabel,
						Secondary:    msse.SecondarySpans,
					},
					msse,
				)
			}
			if sse, ok := errors.AsType[*sasscommon.SassScriptException](err); ok {
				return struct{}{}, sasscommon.ThrowWithTrace(
					&sasscommon.SassException{
						Message: sse.Message,
						Span:    valueSpan,
					},
					sse,
				)
			}
			return struct{}{}, err
		}
	} else {
		s, err := val.String()
		if err != nil {
			return struct{}{}, err
		}
		_, _ = sv.sb.WriteString(s)
	}
	return struct{}{}, nil
}

// writeFoldedValue emits a custom-property value with every newline and
// its following whitespace collapsed to a single space, for compressed
// output where line breaks carry no meaning.
//
// Matches Dart: _SerializeVisitor._writeFoldedValue
func (sv *SerializeVisitor) writeFoldedValue(node CssDeclaration) error {
	val := node.Value().Value
	s, ok := val.(*SassString)
	if !ok {
		return fmt.Errorf("expected SassString for CSS declaration value, got %T", val)
	}
	text := strings.ReplaceAll(s.Text, "\r\n", "\n")
	for i := 0; i < len(text); {
		ch := text[i]
		if ch != '\n' {
			_ = sv.sb.WriteByte(ch)
			i++
			continue
		}
		_ = sv.sb.WriteByte(' ')
		i++
		for i < len(text) {
			next := text[i]
			if next == ' ' || next == '\t' || next == '\n' || next == '\r' || next == '\f' || next == '\v' {
				i++
			} else {
				break
			}
		}
	}
	return nil
}

// writeReindentedValue emits a custom-property value re-indented relative
// to the current depth, for expanded output. Values without newlines are
// written as-is; values whose later lines carry no indentation keep a
// single trailing space; otherwise the value's least-indented line is
// aligned to the declaration's start column.
//
// Matches Dart: _SerializeVisitor._writeReindentedValue
func (sv *SerializeVisitor) writeReindentedValue(node CssDeclaration) error {
	val := node.Value().Value
	s, ok := val.(*SassString)
	if !ok {
		return fmt.Errorf("expected SassString for CSS declaration value, got %T", val)
	}
	text := s.Text
	minIndent := sv.minimumIndentation(text)
	switch {
	case minIndent == nil:
		_, _ = sv.sb.WriteString(text)
	case *minIndent == -1:
		_, _ = sv.sb.WriteString(trimAsciiRight(text, true))
		_ = sv.sb.WriteByte(' ')
	default:
		nameSpan, err := node.Name().Span()
		if err != nil {
			return err
		}
		nameLoc, err := nameSpan.StartLocation()
		if err != nil {
			return err
		}
		col := nameLoc.Column
		actualMin := int(math.Min(float64(*minIndent), float64(col)))
		sv.writeWithIndent(text, actualMin)
	}
	return nil
}

// trimAsciiRight strips trailing ASCII whitespace, optionally stopping at
// whitespace preceded by a backslash so escape continuations survive. It
// ports Dart's trimAsciiRight helper used for the unindented-lines case of
// _writeReindentedValue.
func trimAsciiRight(s string, excludeEscape bool) string {
	end := len(s)
	for end > 0 {
		ch := s[end-1]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f' {
			if excludeEscape && end >= 2 && s[end-2] == '\\' {
				break
			}
			end--
		} else {
			break
		}
	}
	return s[:end]
}

// minimumIndentation returns the indentation of the least-indented
// non-empty line after the first in text. It returns nil when text holds
// no newline and -1 when newlines exist but no later line is indented.
// Only spaces and tabs count as indentation, matching Dart's scanner loop.
//
// Matches Dart: _SerializeVisitor._minimumIndentation
func (sv *SerializeVisitor) minimumIndentation(text string) *int {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	firstNewline := strings.IndexByte(text, '\n')
	if firstNewline == -1 {
		return nil
	}
	if firstNewline == len(text)-1 {
		v := -1
		return &v
	}
	var min *int
	rest := text[firstNewline+1:]
	for line := range strings.SplitSeq(rest, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		indent := 0
		for _, ch := range line {
			if ch == ' ' || ch == '\t' {
				indent++
			} else {
				break
			}
		}
		if min == nil || indent < *min {
			min = &indent
		}
	}
	if min == nil {
		v := -1
		return &v
	}
	return min
}

// writeWithIndent writes text while replacing minimumIndentation with the
// current indentation on every non-empty line after the first. The first
// line goes out as-is; blank runs between lines are preserved; trailing
// whitespace at the end of the text collapses to a single space because it
// can still matter for custom properties. Never splits a code point: the
// scan advances rune by rune like Dart's UTF-16 scanner.
//
// Matches Dart: _SerializeVisitor._writeWithIndent
func (sv *SerializeVisitor) writeWithIndent(text string, minimumIndentation int) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	i := 0

	// Write the first line as-is, consuming the trailing newline without
	// writing it. Direct port of Dart's _writeWithIndent (mirrors rust
	// serialize/css.rs write_with_indent after commit 7c302c37).
	for i < len(text) {
		if text[i] == '\n' {
			i++
			break
		}
		_ = sv.sb.WriteByte(text[i])
		i++
	}

	for {
		// Scan forward until we hit non-whitespace or the end of [text].
		// newlines starts at 1 (the line break already consumed by the content
		// loop) and counts only ADDITIONAL blank lines.
		lineStart := i
		newlines := 1
	inner:
		for {
			if i >= len(text) {
				// Preserve the fact that whitespace exists (custom properties).
				_ = sv.sb.WriteByte(' ')
				return
			}
			switch text[i] {
			case ' ', '\t':
				i++
			case '\n':
				lineStart = i + 1
				newlines++
				i++
			default:
				break inner
			}
		}

		sv.writeByteTimes('\n', newlines)
		sv.writeIndentation()

		if i > lineStart+minimumIndentation {
			_, _ = sv.sb.WriteString(text[lineStart+minimumIndentation : i])
		}

		// Scan and write until we hit a newline or the end of [text]. Consume
		// the newline so the next whitespace scan doesn't double-count it.
		for {
			if i >= len(text) {
				return
			}
			if text[i] == '\n' {
				i++
				break
			}
			_ = sv.sb.WriteByte(text[i])
			i++
		}
	}
}

var _ CssVisitor[struct{}] = (*SerializeVisitor)(nil)
