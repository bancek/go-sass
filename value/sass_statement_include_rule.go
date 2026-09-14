// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/include_rule.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// IncludeRule is a mixin invocation via @include (or the indented-syntax +).
type IncludeRule struct {
	// OriginalName is the mixin name as written, before _-to-hyphen folding.
	OriginalName string
	// Content is the block invoked for @content rules in the mixin,
	// or nil when the invocation passes no content block.
	Content   *ContentBlock
	arguments *ArgumentList
	namespace *string
	name      string
	span      sasscommon.FileSpan
}

// NewIncludeRule creates a mixin invocation of originalName with arguments.
// A nil namespace means the call is unqualified; a nil content means no
// content block is passed.
func NewIncludeRule(
	originalName string,
	arguments *ArgumentList,
	span sasscommon.FileSpan,
	namespace *string,
	content *ContentBlock,
) *IncludeRule {
	return &IncludeRule{
		OriginalName: originalName,
		Content:      content,
		arguments:    arguments,
		namespace:    namespace,
		name:         strings.ReplaceAll(originalName, "_", "-"),
		span:         span,
	}
}

// Arguments returns the arguments passed to the mixin.
func (r *IncludeRule) Arguments() *ArgumentList { return r.arguments }

// Namespace returns the namespace the mixin was invoked under,
// or nil for an unqualified call.
func (r *IncludeRule) Namespace() *string { return r.namespace }

// Name returns the mixin name with underscores folded to hyphens.
func (r *IncludeRule) Name() string { return r.name }

// NameSpan returns the span covering just the mixin name, skipping a leading
// + in the indented syntax and any namespace qualifier.
func (r *IncludeRule) NameSpan() (sasscommon.FileSpan, error) {
	spanText, err := r.span.SpanText()
	if err != nil {
		return nil, err
	}
	var startSpan sasscommon.FileSpan
	if strings.HasPrefix(spanText, "+") {
		var length int
		length, err = r.span.Length()
		if err != nil {
			return nil, err
		}
		startSpan, err = r.span.Subspan(1, length)
		if err != nil {
			return nil, err
		}
		startSpan, err = trimLeftFileSpan(startSpan)
		if err != nil {
			return nil, err
		}
	} else {
		startSpan, err = r.span.WithoutInitialAtRule()
		if err != nil {
			return nil, err
		}
	}
	if r.namespace != nil {
		startSpan, err = withoutNamespace(startSpan)
		if err != nil {
			return nil, err
		}
	}
	result, err := initialIdentifier(startSpan)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// NamespaceSpan returns the span covering the namespace qualifier,
// or nil when the call is unqualified.
func (r *IncludeRule) NamespaceSpan() (*sasscommon.FileSpan, error) {
	if r.namespace == nil {
		return nil, nil
	}
	spanText, err := r.span.SpanText()
	if err != nil {
		return nil, err
	}
	var startSpan sasscommon.FileSpan
	if strings.HasPrefix(spanText, "+") {
		var length int
		length, err = r.span.Length()
		if err != nil {
			return nil, err
		}
		startSpan, err = r.span.Subspan(1, length)
		if err != nil {
			return nil, err
		}
		startSpan, err = trimLeftFileSpan(startSpan)
		if err != nil {
			return nil, err
		}
	} else {
		startSpan, err = r.span.WithoutInitialAtRule()
		if err != nil {
			return nil, err
		}
	}
	result, err := initialIdentifier(startSpan)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SpanWithoutContent returns the invocation span without the trailing
// content block, or the full span when no block was passed.
func (r *IncludeRule) SpanWithoutContent() (sasscommon.FileSpan, error) {
	if r.Content == nil {
		return r.span, nil
	}
	argSpan, err := r.arguments.Span()
	if err != nil {
		return nil, err
	}
	argEndLoc, err := argSpan.EndLocation()
	if err != nil {
		return nil, err
	}
	spanStartLoc, err := r.span.StartLocation()
	if err != nil {
		return nil, err
	}
	result, err := r.span.Subspan(0, argEndLoc.Offset-spanStartLoc.Offset)
	if err != nil {
		return nil, err
	}
	text, err := result.SpanText()
	if err != nil {
		return nil, err
	}
	trimStart := 0
	for trimStart < len(text) && (text[trimStart] == ' ' || text[trimStart] == '\t' || text[trimStart] == '\n' || text[trimStart] == '\r' || text[trimStart] == '\f') {
		trimStart++
	}
	trimEnd := len(text)
	for trimEnd > trimStart && (text[trimEnd-1] == ' ' || text[trimEnd-1] == '\t' || text[trimEnd-1] == '\n' || text[trimEnd-1] == '\r' || text[trimEnd-1] == '\f') {
		trimEnd--
	}
	if trimStart > 0 || trimEnd < len(text) {
		var err error
		result, err = result.Subspan(trimStart, trimEnd)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
func (r *IncludeRule) Span() (sasscommon.FileSpan, error) { return r.span, nil }
func (r *IncludeRule) IsStatement()                       {}
func (r *IncludeRule) IsSassNode()                        {}
func (r *IncludeRule) IsAstNode()                         {}
func (r *IncludeRule) IsCallableInvocation()              {}
func (r *IncludeRule) isSassReference()                   {}

func trimLeftFileSpan(s sasscommon.FileSpan) (sasscommon.FileSpan, error) {
	text, err := s.SpanText()
	if err != nil {
		return nil, err
	}
	start := 0
	for start < len(text) && (text[start] == ' ' || text[start] == '\t' || text[start] == '\n' || text[start] == '\r' || text[start] == '\f') {
		start++
	}
	if start == 0 {
		return s, nil
	}
	length, err := s.Length()
	if err != nil {
		return nil, err
	}
	return s.Subspan(start, length)
}

func withoutNamespace(s sasscommon.FileSpan) (sasscommon.FileSpan, error) {
	afterIdent, err := withoutInitialIdentifier(s)
	if err != nil {
		return nil, err
	}
	text, err := afterIdent.SpanText()
	if err != nil {
		return nil, err
	}
	if len(text) == 0 || text[0] != '.' {
		return afterIdent, nil
	}
	length, err := afterIdent.Length()
	if err != nil {
		return nil, err
	}
	result, err := afterIdent.Subspan(1, length)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func withoutInitialIdentifier(s sasscommon.FileSpan) (sasscommon.FileSpan, error) {
	text, err := s.SpanText()
	if err != nil {
		return nil, err
	}
	pos, err := scanIdentName(text, 0, s)
	if err != nil {
		return nil, err
	}
	length, err := s.Length()
	if err != nil {
		return nil, err
	}
	result, err := s.Subspan(pos, length)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func initialIdentifier(s sasscommon.FileSpan) (sasscommon.FileSpan, error) {
	text, err := s.SpanText()
	if err != nil {
		return nil, err
	}
	pos, err := scanIdentName(text, 0, s)
	if err != nil {
		return nil, err
	}
	result, err := s.Subspan(0, pos)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func scanIdentName(text string, pos int, span sasscommon.FileSpan) (int, error) {
	for pos < len(text) {
		ch := text[pos]
		switch {
		case ch == '\\':
			var err error
			pos, err = consumeEscapedName(text, pos, span)
			if err != nil {
				return pos, err
			}
		case isNameByte(text[pos]):
			pos++
		default:
			return pos, nil
		}
	}
	return pos, nil
}

func consumeEscapedName(text string, pos int, span sasscommon.FileSpan) (int, error) {
	if pos >= len(text) || text[pos] != '\\' {
		return pos, nil
	}
	pos++
	if pos >= len(text) {
		return pos, nil
	}
	ch := text[pos]
	switch {
	case ch == '\n' || ch == '\r' || ch == '\f':
		return pos, &sasscommon.ScanError{Message: "Expected escape sequence.", Span: span}
	case isHexByte(text[pos]):
		for i := 0; i < 6 && pos < len(text) && isHexByte(text[pos]); i++ {
			pos++
		}
		if pos < len(text) && isWhitespaceByte(text[pos]) {
			pos++
		}
	default:
		pos++
	}
	return pos, nil
}

func isNameByte(b byte) bool {
	return b == '_' ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') ||
		b == '-' ||
		b >= 0x80
}

func isHexByte(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

func isWhitespaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

func (r *IncludeRule) String() (string, error) {
	var builder strings.Builder
	builder.WriteString("@include ")
	if r.namespace != nil {
		builder.WriteString(*r.namespace)
		builder.WriteString(".")
	}
	builder.WriteString(r.name)
	if len(r.arguments.Positional) > 0 || r.arguments.Named.Len() > 0 || r.arguments.Rest != nil || r.arguments.KeywordRest != nil {
		builder.WriteString("(")
		argsStr, err := r.arguments.String()
		if err != nil {
			return "", err
		}
		builder.WriteString(argsStr)
		builder.WriteString(")")
	}
	if r.Content == nil {
		builder.WriteString(";")
	} else {
		builder.WriteString(" ")
		contentStr, err := r.Content.String()
		if err != nil {
			return "", err
		}
		builder.WriteString(contentStr)
	}
	return builder.String(), nil
}
