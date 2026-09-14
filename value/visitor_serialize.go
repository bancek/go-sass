// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/serialize.dart (top-level serialize functions,
// OutputStyle, LineFeed, SerializeResult, and _SerializeVisitor core state
// plus buffer utilities; visitCss* methods live in visitor_css.go,
// visitColor in visitor_color.go, value visits in visitor_value.go,
// calculation helpers in visitor_calc.go, list helpers in visitor_list.go,
// number emission in visitor_number.go, string emission in visitor_string.go,
// selector visits in visitor_selector.go)

import (
	"fmt"
	"io"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sasslogger"
	"github.com/bancek/go-sass/sourcemap"
	"github.com/bancek/go-sass/sourcemapbuffer"
)

// OutputStyle selects the generated CSS style. Dart declares only these two
// spellings; there are no others.
type OutputStyle int

const (
	// OutputStyleExpanded is the standard multi-line style with each
	// declaration on its own line.
	//
	// Covers Dart's OutputStyle.expanded:
	//
	//	.sidebar {
	//	  width: 100px;
	//	}
	OutputStyleExpanded OutputStyle = iota
	// OutputStyleCompressed produces as few bytes of output as possible.
	//
	// Covers Dart's OutputStyle.compressed:
	//
	//	.sidebar{width:100px}
	OutputStyleCompressed
)

// String returns the Sass-level name of the style ("expanded"/"compressed").
func (s OutputStyle) String() string {
	switch s {
	case OutputStyleExpanded:
		return "expanded"
	case OutputStyleCompressed:
		return "compressed"
	default:
		return "?"
	}
}

// LineFeed is one of the line-feed sequences the serializer can emit.
// It ports Dart's LineFeed enum, which pairs each sequence's Sass-level
// name with the text written to the buffer.
type LineFeed struct {
	// Name is the Sass-level name of the sequence ("cr", "crlf", "lf",
	// "lfcr"). It ports Dart's LineFeed.name field.
	Name string
	// Text is the literal text emitted for the sequence. It ports Dart's
	// LineFeed.text field.
	Text string
}

var (
	// LineFeedCr emits a single carriage return. It ports Dart's LineFeed.cr.
	LineFeedCr = LineFeed{"cr", "\r"}
	// LineFeedCrLf emits a carriage return followed by a line feed. It ports
	// Dart's LineFeed.crlf.
	LineFeedCrLf = LineFeed{"crlf", "\r\n"}
	// LineFeedLf emits a single line feed. It ports Dart's LineFeed.lf.
	LineFeedLf = LineFeed{"lf", "\n"}
	// LineFeedLfCr emits a line feed followed by a carriage return. It ports
	// Dart's LineFeed.lfcr.
	LineFeedLfCr = LineFeed{"lfcr", "\n\r"}
)

// String returns the sequence name, matching Dart's LineFeed.toString which
// returns name rather than the emitted text.
func (lf LineFeed) String() string { return lf.Name }

// SerializeOptions carries the named parameters of Dart's top-level
// serialize function. Nil pointers take Dart's defaults: spaces for
// indentation, width 2, LF line feeds.
type SerializeOptions struct {
	// Style controls the output style and defaults to expanded.
	Style OutputStyle
	// Inspect selects the unambiguous source-structure representation
	// instead of valid CSS. Valid SCSS is still emitted, but the result
	// may not be valid CSS.
	Inspect     bool
	UseSpaces   *bool // nil means true (use spaces)
	IndentWidth *int  // nil means use default (2)
	// LineFeed selects the line-feed sequence and defaults to LF.
	LineFeed *LineFeed
	// Logger receives warnings raised while serializing. Like Dart's
	// logger parameter, it is only meaningful for statement-level
	// serialization and may not be the main user-provided logger for
	// plain values.
	Logger sasslogger.Logger
	// SourceMap requests source-map output in the returned result.
	SourceMap bool
	// Charset enables the @charset declaration (or BOM in compressed
	// mode) when the output contains non-ASCII characters. It defaults
	// to true when SerializeWithSourceMap builds its own defaults.
	Charset bool
}

// SerializeResult is the outcome of converting a CSS tree to text. It ports
// Dart's SerializeResult record.
type SerializeResult struct {
	// CSS is the serialized stylesheet text, including any @charset/BOM
	// prefix.
	CSS string
	// SourceMap maps the output back to the source files, or nil when
	// source mapping was disabled for the compilation.
	SourceMap *sourcemap.SingleMapping // nil if source mapping was disabled
}

// Serialize converts node to a CSS string, defaulting to expanded output
// with spaces, indent width 2, LF line feeds, and charset handling on when
// opts is nil.
//
// When opts requests a source map, the returned result carries the mapping
// from the original Sass files to the compiled CSS.
//
// Matches Dart: serialize (top-level function)
func Serialize(node CssNode, opts *SerializeOptions) (*SerializeResult, error) {
	return SerializeWithSourceMap(node, opts, nil)
}

// SerializeWithSourceMap converts node to a CSS string and, when source
// mapping is enabled (via opts or a non-nil bldr), records how the output
// maps back to the source files. A nil bldr with SourceMap set builds an
// internal mapping; entries accumulate into bldr when one is supplied.
//
// Like Dart's serialize, a @charset declaration (or a BOM in compressed
// mode) is prepended when charset handling is on and the output holds
// non-ASCII characters.
//
// Matches Dart: serialize (top-level function)
func SerializeWithSourceMap(node CssNode, opts *SerializeOptions, bldr *sourcemap.Builder) (*SerializeResult, error) {
	var style OutputStyle
	inspect := false
	useSpaces := true
	indentWidth := 2
	lineFeed := LineFeedLf
	var logger sasslogger.Logger
	charset := true
	if opts != nil {
		style = opts.Style
		inspect = opts.Inspect
		if opts.UseSpaces != nil {
			useSpaces = *opts.UseSpaces
		}
		if opts.IndentWidth != nil {
			indentWidth = *opts.IndentWidth
		}
		if opts.LineFeed != nil {
			lineFeed = *opts.LineFeed
		}
		logger = opts.Logger
		charset = opts.Charset
	}
	if indentWidth < 0 || indentWidth > 10 {
		return nil, &sasscommon.RangeError{Name: "indentWidth", Message: fmt.Sprintf("must be between 0 and 10, was %d", indentWidth)}
	}

	var internalBldr *sourcemap.Builder
	if opts != nil && opts.SourceMap {
		internalBldr = sourcemap.NewBuilder("")
	}
	var buf sourcemapbuffer.SourceMapBuffer
	if internalBldr != nil {
		buf = sourcemapbuffer.NewDefaultSourceMapBuffer(internalBldr)
	} else if bldr != nil {
		buf = sourcemapbuffer.NewDefaultSourceMapBuffer(bldr)
	} else {
		buf = sourcemapbuffer.NewNoSourceMapBuffer()
	}

	sv := &SerializeVisitor{
		quote:           true,
		style:           style,
		inspect:         inspect,
		lineFeed:        lineFeed,
		indentCharacter: ' ',
		indentWidth:     indentWidth,
		sb:              buf,
		logger:          logger,
	}
	if !useSpaces {
		sv.indentCharacter = '\t'
	}
	_, err := node.AcceptVoid(sv)
	if err != nil {
		return nil, err
	}
	css := sv.sb.String()
	prefix := ""
	if charset {
		for _, ch := range css {
			if ch > 0x7F {
				if sv.isCompressed() {
					prefix = "\uFEFF"
				} else {
					prefix = "@charset \"UTF-8\";\n"
				}
				break
			}
		}
	}
	css = prefix + css

	result := &SerializeResult{CSS: css}
	if internalBldr != nil || bldr != nil {
		var err error
		result.SourceMap, err = buf.BuildSourceMap(prefix)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// SerializeValue converts value to a CSS string with quotes preserved.
// Values that cannot be represented in plain CSS raise a script exception.
//
// Matches Dart: serializeValue (top-level function, quote parameter)
func SerializeValue(v Value, quote bool) (string, error) {
	return SerializeValueFull(v, quote, false)
}

// SerializeValueFull converts value to a CSS string. When inspect is set,
// it emits an unambiguous rendering of the source structure instead; the
// result is valid SCSS but may not be valid CSS. When quote is false,
// quoted strings are emitted without their quotes.
//
// Matches Dart: serializeValue (top-level function, inspect and quote
// parameters)
func SerializeValueFull(v Value, quote bool, inspect bool) (string, error) {
	sv := &SerializeVisitor{
		sb:              sourcemapbuffer.NewNoSourceMapBuffer(),
		quote:           quote,
		inspect:         inspect,
		lineFeed:        LineFeedLf,
		indentCharacter: ' ',
		indentWidth:     2,
	}
	_, err := v.AcceptVoid(sv)
	if err != nil {
		return "", err
	}
	return sv.sb.String(), nil
}

// SerializeValueInspect converts value in inspect mode, emitting an
// unambiguous rendering of the source structure for the meta.inspect
// path. The result is valid SCSS but may not be valid CSS.
//
// Matches Dart: serializeValue (top-level function, inspect: true)
func SerializeValueInspect(v Value) (string, error) {
	return SerializeValueFull(v, true, true)
}

// SerializeSelector converts selector to a CSS string. When inspect is
// set, it emits an unambiguous rendering of the source structure instead;
// the result is valid SCSS but may not be valid CSS. Selectors that cannot
// be represented in plain CSS raise a script exception.
//
// Matches Dart: serializeSelector (top-level function)
func SerializeSelector(s Selector, inspect bool) (string, error) {
	sv := &SerializeVisitor{
		sb:              sourcemapbuffer.NewNoSourceMapBuffer(),
		inspect:         inspect,
		lineFeed:        LineFeedLf,
		indentCharacter: ' ',
		indentWidth:     2,
	}
	_, err := s.AcceptVoid(sv)
	if err != nil {
		return "", err
	}
	return sv.sb.String(), nil
}

// SerializeVisitor converts CSS syntax trees to plain strings. It is the
// Go owner of Dart's _SerializeVisitor: one visitor implementing the CSS,
// value, and selector visitor interfaces, writing into a source-map-aware
// buffer. Every visit method reports (struct{}, error) through AcceptVoid
// because no serialization step is infallible.
//
// Matches Dart: _SerializeVisitor (class documentation)
type SerializeVisitor struct {
	// sb holds the CSS produced so far. It can be swapped out temporarily
	// to capture one chunk of serialization as a string (see capture).
	sb sourcemapbuffer.SourceMapBuffer
	// indentation tracks the current nesting depth of the CSS output.
	indentation int
	// style selects expanded versus compressed output.
	style OutputStyle
	// inspect selects the unambiguous source-structure rendering over
	// valid CSS.
	inspect bool
	// quote controls whether quoted strings keep their quotes.
	quote bool
	// lineFeed is the character sequence written for each line break.
	lineFeed LineFeed
	// indentCharacter is the space or tab used for one indentation level.
	indentCharacter int
	// indentWidth is the number of indent characters per level (0-10,
	// validated like Dart's RangeError.checkValueInInterval).
	indentWidth int
	// logger takes warnings raised during statement-level serialization.
	// It is kept even when unused so deprecation churn stays small, and
	// it is not guaranteed to be the main user-provided logger for plain
	// value serialization.
	logger sasslogger.Logger
}

// isCompressed reports whether compressed output is being emitted.
// It ports Dart's _SerializeVisitor._isCompressed getter.
func (sv *SerializeVisitor) isCompressed() bool {
	return sv.style == OutputStyleCompressed
}

// commaSep returns the list-item separator for the current style:
// ", " expanded, "," compressed. It ports Dart's _commaSeparator getter.
func (sv *SerializeVisitor) commaSep() string {
	if sv.isCompressed() {
		return ","
	}
	return ", "
}

// ---- Helper methods ----

// writeByteTimes writes b to the buffer times repetitions. It ports Dart's
// _writeTimes helper.
func (sv *SerializeVisitor) writeByteTimes(b byte, times int) {
	for range times {
		_ = sv.sb.WriteByte(b)
	}
}

// writeIndentation writes the current indentation level, emitting nothing
// in compressed mode. It ports Dart's _writeIndentation.
func (sv *SerializeVisitor) writeIndentation() {
	if sv.isCompressed() {
		return
	}
	sv.writeByteTimes(byte(sv.indentCharacter), sv.indentation*sv.indentWidth)
}

// writeLineFeed emits a line feed unless the output is compressed.
// It ports Dart's _writeLineFeed.
func (sv *SerializeVisitor) writeLineFeed() {
	if !sv.isCompressed() {
		_, _ = sv.sb.WriteString(sv.lineFeed.Text)
	}
}

// writeOptionalSpace emits one space unless the output is compressed.
// It ports Dart's _writeOptionalSpace.
func (sv *SerializeVisitor) writeOptionalSpace() {
	if !sv.isCompressed() {
		_ = sv.sb.WriteByte(' ')
	}
}

// writeBetween runs cb for each item in items, writing sep between
// consecutive items. It ports Dart's _writeBetween helper.
func writeBetween[T any](sb io.Writer, items []T, sep string, cb func(T) error) error {
	for i, item := range items {
		if i > 0 {
			if _, err := io.WriteString(sb, sep); err != nil {
				return err
			}
		}
		if err := cb(item); err != nil {
			return err
		}
	}
	return nil
}

// forNode runs cb and associates every byte it writes with the node's
// source span, so the source map can trace the output back. It routes
// through the source-map buffer's ForSpan, which is a no-op mapping on a
// buffer without source-map support.
//
// Matches Dart: _SerializeVisitor._for
func (sv *SerializeVisitor) forNode(node sasscommon.AstNode, cb func() error) error {
	span, err := node.Span()
	if err != nil {
		return err
	}
	return sv.sb.ForSpan(span, cb)
}

// write emits value's text together with its source span, so mapped
// output stays aligned with the originating span.
//
// Matches Dart: _SerializeVisitor._write
func (sv *SerializeVisitor) write(value sasscommon.CssValue[string]) error {
	return sv.forNode(value, func() error {
		_, err := sv.sb.WriteString(value.Value)
		return err
	})
}

// isInvisible reports whether node is omitted from the output. Inspect
// mode shows everything; otherwise compressed output additionally hides
// comments while expanded output hides only structurally invisible nodes.
//
// Matches Dart: _SerializeVisitor._isInvisible
func (sv *SerializeVisitor) isInvisible(node CssNode) bool {
	if sv.inspect {
		return false
	}
	if sv.isCompressed() {
		return node.IsInvisibleHidingComments()
	}
	return node.IsInvisible()
}

// requiresSemicolon reports whether node needs a semicolon after it: a
// parent node with no children does, while any other parent leaves the
// separator to its children; comments never take one and every other
// node does.
//
// Matches Dart: _SerializeVisitor._requiresSemicolon
func (sv *SerializeVisitor) requiresSemicolon(node CssNode) bool {
	if pn, ok := node.(CssParentNode); ok {
		return pn.IsChildless()
	}
	_, isComment := node.(CssComment)
	return !isComment
}

// isTrailingComment reports whether node is a comment trailing previous
// on the same rendered line. previous may be either a sibling of node or
// the parent when node is the first visible child. Compressed mode never
// treats comments as trailing (short-circuited before the span work,
// since whitespace is compressed away regardless), and the check bails
// out when either span lookup fails or the URLs differ.
//
// Matches Dart: _SerializeVisitor._isTrailingComment
func (sv *SerializeVisitor) isTrailingComment(node CssNode, previous CssNode) (bool, error) {
	if sv.isCompressed() {
		return false, nil
	}
	if _, ok := node.(CssComment); !ok {
		return false, nil
	}
	nodeSpan, err := node.Span()
	if err != nil {
		return false, err
	}
	previousSpan, err := previous.Span()
	if err != nil {
		return false, err
	}
	na, err := nodeSpan.SourceURL()
	if err != nil {
		return false, err
	}
	nb, err := previousSpan.SourceURL()
	if err != nil {
		return false, err
	}
	if (na == nil) != (nb == nil) || (na != nil && na.String() != nb.String()) {
		return false, nil
	}
	contains, err := spanContains(previousSpan, nodeSpan)
	if err != nil {
		return false, err
	}
	if !contains {
		nodeStartLoc, err := nodeSpan.StartLocation()
		if err != nil {
			return false, err
		}
		previousEndLoc, err := previousSpan.EndLocation()
		if err != nil {
			return false, err
		}
		return nodeStartLoc.Line == previousEndLoc.Line, nil
	}
	nodeStartLoc, err := nodeSpan.StartLocation()
	if err != nil {
		return false, err
	}
	previousStartLoc, err := previousSpan.StartLocation()
	if err != nil {
		return false, err
	}
	searchFrom := nodeStartLoc.Offset - previousStartLoc.Offset - 1
	if searchFrom < 0 {
		return false, nil
	}

	searchText, err := previousSpan.SpanText()
	if err != nil {
		return false, err
	}
	if searchFrom >= len(searchText) {
		return false, nil
	}
	endOffset := max(strings.LastIndex(searchText[:searchFrom], "{"), 0)
	endLine := previousStartLoc.Line
	for i := range endOffset {
		if searchText[i] == '\n' {
			endLine++
		}
	}
	return nodeStartLoc.Line == endLine, nil
}

// spanContains reports whether the outer span fully encloses the inner
// span. It is Go-only glue standing in for Dart's FileSpan.contains,
// which has no counterpart on the Go span type.
//
// Matches Dart: FileSpan.contains (as used by _isTrailingComment)
func spanContains(outer, inner sasscommon.FileSpan) (bool, error) {
	outerStart, err := outer.StartLocation()
	if err != nil {
		return false, err
	}
	innerStart, err := inner.StartLocation()
	if err != nil {
		return false, err
	}
	outerEnd, err := outer.EndLocation()
	if err != nil {
		return false, err
	}
	innerEnd, err := inner.EndLocation()
	if err != nil {
		return false, err
	}
	return outerStart.Offset <= innerStart.Offset && outerEnd.Offset >= innerEnd.Offset, nil
}

// visitCssNode dispatches a single CSS node back through this visitor.
// It ports the child.accept(this) calls inside Dart's _visitChildren.
func (sv *SerializeVisitor) visitCssNode(node CssNode) (struct{}, error) {
	return node.AcceptVoid(sv)
}

// visitChildren emits parent's children inside a brace block, separating
// them with semicolons, line feeds, and trailing-comment spacing following
// the same rules as the stylesheet-level loop. A lone trailing comment
// after the last child is kept on the same line; otherwise the block
// closes on its own indented line.
//
// Matches Dart: _SerializeVisitor._visitChildren
func (sv *SerializeVisitor) visitChildren(parent CssParentNode) (struct{}, error) {
	_ = sv.sb.WriteByte('{')
	var prePrevious, previous CssNode
	for _, child := range parent.Children() {
		if sv.isInvisible(child) {
			continue
		}
		if previous != nil && sv.requiresSemicolon(previous) {
			_ = sv.sb.WriteByte(';')
		}
		isTrailing, err := sv.isTrailingComment(child, previousOrParent(previous, parent))
		if err != nil {
			return struct{}{}, err
		}
		if isTrailing {
			sv.writeOptionalSpace()
			if err := sv.withoutIndent(func() error {
				_, err := sv.visitCssNode(child)
				return err
			}); err != nil {
				return struct{}{}, err
			}
		} else {
			sv.writeLineFeed()
			if err := sv.withIndent(func() error {
				_, err := sv.visitCssNode(child)
				return err
			}); err != nil {
				return struct{}{}, err
			}
		}
		prePrevious = previous
		previous = child
	}
	if previous != nil {
		if sv.requiresSemicolon(previous) && !sv.isCompressed() {
			_ = sv.sb.WriteByte(';')
		}
		isTrailing, err := sv.isTrailingComment(previous, parent)
		if err != nil {
			return struct{}{}, err
		}
		if prePrevious == nil && isTrailing {
			sv.writeOptionalSpace()
		} else {
			sv.writeLineFeed()
			sv.writeIndentation()
		}
	}
	_ = sv.sb.WriteByte('}')
	return struct{}{}, nil
}

// previousOrParent returns previous when a preceding sibling was already
// emitted, falling back to parent for the first visible child. It is
// Go-only glue for Dart's `previous ?? parent` expression in
// _visitChildren's trailing-comment check.
func previousOrParent(previous CssNode, parent CssParentNode) CssNode {
	if previous != nil {
		return previous
	}
	return parent
}

// withIndent runs cb one indentation level deeper. It ports Dart's
// _indent helper.
func (sv *SerializeVisitor) withIndent(cb func() error) error {
	sv.indentation++
	err := cb()
	sv.indentation--
	return err
}

// withoutIndent runs cb with indentation suppressed, restoring the prior
// depth afterwards. It ports Dart's _withoutIndentation helper, used for
// trailing comments that share their parent's line.
func (sv *SerializeVisitor) withoutIndent(cb func() error) error {
	saved := sv.indentation
	sv.indentation = 0
	err := cb()
	sv.indentation = saved
	return err
}

// ---- Character utilities ----

// hexCharFor returns the lowercase hex digit for a 0-15 value. It ports
// the hexCharFor helper used when emitting hex colors and escapes.
func hexCharFor(number int) int {
	if number < 0xA {
		return '0' + number
	}
	return 'a' - 0xA + number
}

// isHexDig reports whether ch is an ASCII hexadecimal digit. It ports the
// isHex check used to decide whether an escape needs a disambiguating
// trailing space.
func isHexDig(ch int) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

// isPrivateBMP reports whether ch falls in the Basic Multilingual Plane
// private-use area. Expanded output escapes such characters so readers can
// tell glyph-font code points apart; compressed output passes them through.
// It ports the isPrivateUseBMP check used by _tryPrivateUseCharacter.
func isPrivateBMP(ch int) bool {
	return ch >= 0xE000 && ch <= 0xF8FF
}

// isPrivateHighSurrogate reports whether ch is a high surrogate that can
// open a supplementary private-use character. It ports the
// isPrivateUseHighSurrogate check used by _tryPrivateUseCharacter.
func isPrivateHighSurrogate(ch int) bool {
	return ch>>7 == 0x1B7
}

// isSupplementaryPrivateUse reports whether ch is a supplementary-plane
// private-use code point. It ports the surrogate-pair branch of
// _tryPrivateUseCharacter.
func isSupplementaryPrivateUse(ch int) bool {
	return (ch >= 0xF0000 && ch <= 0xFFFFD) || (ch >= 0x100000 && ch <= 0x10FFFD)
}

// utf8Decode decodes the first UTF-8 sequence in s, returning the
// replacement character and a width of 1 for truncated or otherwise
// invalid input. It is Go-only glue letting the string-emission loops walk
// Dart UTF-16 code units as Go runes.
func utf8Decode(s string) (rune, int) {
	if len(s) == 0 {
		return 0, 0
	}
	ch := s[0]
	if ch < 0x80 {
		return rune(ch), 1
	}
	if ch < 0xC0 {
		return 0xFFFD, 1
	}
	if ch < 0xE0 {
		if len(s) < 2 {
			return 0xFFFD, 1
		}
		return rune(ch&0x1F)<<6 | rune(s[1]&0x3F), 2
	}
	if ch < 0xF0 {
		if len(s) < 3 {
			return 0xFFFD, 1
		}
		return rune(ch&0x0F)<<12 | rune(s[1]&0x3F)<<6 | rune(s[2]&0x3F), 3
	}
	if len(s) < 4 {
		return 0xFFFD, 1
	}
	return rune(ch&0x07)<<18 | rune(s[1]&0x3F)<<12 | rune(s[2]&0x3F)<<6 | rune(s[3]&0x3F), 4
}
