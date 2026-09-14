// Copyright 2023 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/interpolation_map.dart

import (
	"fmt"
	"math"
	"sort"

	"github.com/bancek/go-sass/sasscommon"
)

// InterpolationMap maps locations in a string generated from an
// Interpolation to the original source code in the interpolation.
//
// After an interpolation is evaluated to plain text (for example a selector
// that is then re-parsed), error spans point into the generated string. This
// map translates them back to the user's source: offsets into generated text
// resolve through targetOffsets to either literal source locations or the
// spans of the expressions that produced them.
type InterpolationMap struct {
	interpolation *Interpolation
	targetOffsets []int
	lineStarts    []int // lazily built from file text
}

// NewInterpolationMap creates a new InterpolationMap that maps the given
// targetOffsets in the generated string to the contents of the interpolation.
//
// Each target offset at index i corresponds to the character in the generated
// string after interpolation.contents[i]. There is always one fewer offset
// than contents, because the last component runs to the end of the string;
// a mismatch returns an ArgumentError.
// (Dart: InterpolationMap.new, which validates the same count.)
func NewInterpolationMap(interpolation *Interpolation, targetOffsets []int) (*InterpolationMap, error) {
	expected := int(math.Max(0, float64(len(interpolation.Contents)-1)))
	if len(targetOffsets) != expected {
		return nil, &sasscommon.ArgumentError{Name: "targetOffsets", Message: fmt.Sprintf(
			"InterpolationMap must have %d targetOffsets if the interpolation has %d components.",
			expected, len(interpolation.Contents),
		)}
	}
	offsets := make([]int, len(targetOffsets))
	copy(offsets, targetOffsets)
	return &InterpolationMap{
		interpolation: interpolation,
		targetOffsets: offsets,
	}, nil
}

// MapSpan maps a span in the string generated from this interpolation to its
// original source. Returns target as-is if it's already been mapped.
//
// Each endpoint maps independently: endpoints inside an interpolated
// expression resolve to that expression's span, endpoints in literal text
// resolve to source locations. Mixed spans stretch to cover the interpolation
// syntax itself (via the expand helpers), and two expression endpoints merge
// with Expand. (Dart: InterpolationMap.mapSpan.)
func (m *InterpolationMap) MapSpan(target sasscommon.Span) (sasscommon.Span, error) {
	mapped, err := m.isMapped(target)
	if err != nil {
		return nil, err
	}
	if mapped {
		return target, nil
	}

	start, err := target.StartLocation()
	if err != nil {
		return nil, err
	}
	end, err := target.EndLocation()
	if err != nil {
		return nil, err
	}
	startLoc, err := m.mapLocation(start)
	if err != nil {
		return nil, err
	}
	endLoc, err := m.mapLocation(end)
	if err != nil {
		return nil, err
	}

	switch start := startLoc.(type) {
	case sasscommon.FileSpan:
		switch end := endLoc.(type) {
		case sasscommon.FileSpan:
			return start.Expand(end)
		case sasscommon.SourceLocation:
			interpSpan, err := m.interpolation.Span()
			if err != nil {
				return nil, err
			}
			interpFile, err := interpSpan.File()
			if err != nil {
				return nil, err
			}
			startStart, err := start.StartLocation()
			if err != nil {
				return nil, err
			}
			offs, err := m.expandInterpolationSpanLeft(startStart.Offset)
			if err != nil {
				return nil, err
			}
			return sasscommon.NewSimpleFileSpan(interpFile, offs, end.Offset), nil
		}

	case sasscommon.SourceLocation:
		switch end := endLoc.(type) {
		case sasscommon.FileSpan:
			interpSpan, err := m.interpolation.Span()
			if err != nil {
				return nil, err
			}
			interpFile, err := interpSpan.File()
			if err != nil {
				return nil, err
			}
			endEnd, err := end.EndLocation()
			if err != nil {
				return nil, err
			}
			offs, err := m.expandInterpolationSpanRight(endEnd.Offset)
			if err != nil {
				return nil, err
			}
			return sasscommon.NewSimpleFileSpan(interpFile, start.Offset, offs), nil
		case sasscommon.SourceLocation:
			interpSpan, err := m.interpolation.Span()
			if err != nil {
				return nil, err
			}
			interpFile, err := interpSpan.File()
			if err != nil {
				return nil, err
			}
			return sasscommon.NewSimpleFileSpan(interpFile, start.Offset, end.Offset), nil
		}
	}

	panic("[BUG] Unreachable")
}

// MapException maps the span within an error from the generated string
// back to the original source. Returns the original error if its span is
// nil or already mapped.
//
// An empty interpolation maps everything to the whole interpolation span.
// Otherwise the shape of the result depends on what the span covers: purely
// literal text keeps a single span, while any overlap with an interpolated
// expression produces a multi-span error whose secondary label points at the
// generated output ("error in interpolated output") so both locations stay
// visible. Unchanged spans return the original error value.
//
// Matches Dart: InterpolationMap.mapException
func (m *InterpolationMap) MapException(ssfErr *sasscommon.SourceSpanFormatException) error {
	target := ssfErr.Span
	if target == nil {
		return ssfErr
	}

	if len(m.interpolation.Contents) == 0 {
		mapped, err := m.isMapped(target)
		if err != nil {
			return err
		}
		if mapped {
			return ssfErr
		}
		interpSpan, err := m.interpolation.Span()
		if err != nil {
			return err
		}
		return &sasscommon.SourceSpanFormatException{
			Message: ssfErr.Message,
			Span:    interpSpan,
			Source:  ssfErr.Source,
		}
	}

	source, err := m.MapSpan(target)
	if err != nil {
		return err
	}
	sourceFS, ok := source.(sasscommon.FileSpan)
	if !ok {
		return ssfErr
	}
	if source == target {
		return ssfErr
	}

	start, err := target.StartLocation()
	if err != nil {
		return err
	}
	end, err := target.EndLocation()
	if err != nil {
		return err
	}
	startIndex := m.indexInContents(start)
	endIndex := m.indexInContents(end)

	hasExpression := false
	for i := startIndex; i <= endIndex && i < len(m.interpolation.Contents); i++ {
		if _, ok := m.interpolation.Contents[i].(Expression); ok {
			hasExpression = true
			break
		}
	}

	if !hasExpression {
		return &sasscommon.SourceSpanFormatException{
			Message: ssfErr.Message,
			Span:    sourceFS,
			Source:  ssfErr.Source,
		}
	}

	return &sasscommon.MultiSourceSpanFormatError{
		Message:      ssfErr.Message,
		Span:         sourceFS,
		PrimaryLabel: "",
		Source:       ssfErr.Source,
		Secondary:    map[sasscommon.FileSpan]string{target: "error in interpolated output"},
	}
}

// isMapped returns whether span has already been mapped by this mapper.
//
// A span counts as mapped when its file is the interpolation's own source
// file — file identity, not offset comparison, is the test. (Dart: _isMapped.)
func (m *InterpolationMap) isMapped(target sasscommon.Span) (bool, error) {
	if tfs, ok := target.(sasscommon.FileSpan); ok {
		targetFile, err := tfs.File()
		if err != nil {
			return false, err
		}
		interpSpan, err := m.interpolation.Span()
		if err != nil {
			return false, err
		}
		interpFile, err := interpSpan.File()
		if err != nil {
			return false, err
		}
		return targetFile == interpFile, nil
	}
	return false, nil
}

// mapLocation maps a location in the generated string to its original source.
//
// Returns an filespan.FileSpan if the location points to text generated from an
// Expression, or an filespan.SourceLocation if it points to literal text.
//
// An index landing on an expression yields that expression's full span. An
// index in literal text converts to a source offset: the generated offset
// within the current component plus the source offset where that component
// starts (the interpolation start for the first component, otherwise just
// past the previous expression's closing `}`). Line starts cache lazily from
// the source file for the offset-to-location conversion. (Dart: _mapLocation,
// including the caveat that unnecessary escapes in the source map slightly
// off — accepted as too unlikely to justify a reparse.)
func (m *InterpolationMap) mapLocation(target sasscommon.SourceLocation) (any, error) {
	if len(m.interpolation.Contents) == 0 {
		interpSpan, err := m.interpolation.Span()
		if err != nil {
			return nil, err
		}
		return interpSpan, nil
	}

	index := m.indexInContents(target)
	if expr, ok := m.interpolation.Contents[index].(Expression); ok {
		exprSpan, err := expr.Span()
		if err != nil {
			return nil, err
		}
		return exprSpan, nil
	}

	var previousOffset int
	if index == 0 {
		interpSpan, err := m.interpolation.Span()
		if err != nil {
			return nil, err
		}
		startLoc, err := interpSpan.StartLocation()
		if err != nil {
			return nil, err
		}
		previousOffset = startLoc.Offset
	} else {
		prevExpr := m.interpolation.Contents[index-1].(Expression)
		prevExprSpan, err := prevExpr.Span()
		if err != nil {
			return nil, err
		}
		endLoc, err := prevExprSpan.EndLocation()
		if err != nil {
			return nil, err
		}
		offs, err := m.expandInterpolationSpanRight(endLoc.Offset)
		if err != nil {
			return nil, err
		}
		previousOffset = offs
	}

	offsetInString := target.Offset
	if index > 0 {
		offsetInString -= m.targetOffsets[index-1]
	}

	if m.lineStarts == nil {
		interpSpan, err := m.interpolation.Span()
		if err != nil {
			return nil, err
		}
		f, err := interpSpan.File()
		if err != nil {
			return nil, err
		}
		if f != nil {
			m.lineStarts = buildLineStarts(f.Text())
		} else {
			m.lineStarts = []int{0}
		}
	}
	return sourceLocationFromOffset(m.lineStarts, previousOffset+offsetInString), nil
}

// indexInContents returns the index in interpolation.Contents at which target
// points.
//
// Target offsets mark component ends, so the first offset past the location
// selects its component; a location past every offset belongs to the final
// component. (Dart: _indexInContents.)
func (m *InterpolationMap) indexInContents(target sasscommon.SourceLocation) int {
	for i := 0; i < len(m.targetOffsets); i++ {
		if target.Offset < m.targetOffsets[i] {
			return i
		}
	}
	return len(m.interpolation.Contents) - 1
}

// expandInterpolationSpanLeft returns the offset of the interpolation's
// opening `#`, given the start offset of a FileSpan covering an interpolated
// expression.
//
// The scan walks left looking for `#{`, skipping over block comments so a
// `#{` inside `/* ... */` is not mistaken for the opener. A `#{` inside a
// single-line comment can still fool it, but the result feeds error reporting
// only, where that imprecision is acceptable. Empty sources fall back to two
// characters before the start (the assumed `#{` width).
//
// Note that this can be tricked by a `#{` that appears within a single-line
// comment before the expression, but since it's only used for error reporting
// that's probably fine. (Dart: _expandInterpolationSpanLeft.)
func (m *InterpolationMap) expandInterpolationSpanLeft(startOffset int) (int, error) {
	var source string
	interpSpan, err := m.interpolation.Span()
	if err != nil {
		return 0, err
	}
	f, err := interpSpan.File()
	if err != nil {
		return 0, err
	}
	if f != nil {
		source = f.Text()
	}
	if source == "" {
		return startOffset - 2, nil
	}
	i := startOffset - 1
	for i >= 0 {
		prev := source[i]
		i--
		if prev == '{' {
			if i >= 0 && source[i] == '#' {
				break
			}
		} else if prev == '/' {
			if i >= 0 {
				second := source[i]
				i--
				if second == '*' {
					for {
						char := source[i]
						i--
						if char != '*' {
							continue
						}
						for i >= 0 && source[i] == '*' {
							i--
						}
						if i >= 0 && source[i] == '/' {
							break
						}
					}
				}
			}
		}
	}
	return i, nil
}

// expandInterpolationSpanRight returns the offset of the interpolation's
// closing `}`, given the end offset of a FileSpan covering an interpolated
// expression.
//
// The scan walks right to the first `}` outside comments, skipping `//` line
// comments through their terminator and `/* ... */` blocks in full —
// otherwise a brace inside a comment would end the search early. Empty sources
// fall back to one character past the end. (Dart: _expandInterpolationSpanRight.)
func (m *InterpolationMap) expandInterpolationSpanRight(endOffset int) (int, error) {
	var source string
	interpSpan, err := m.interpolation.Span()
	if err != nil {
		return 0, err
	}
	f, err := interpSpan.File()
	if err != nil {
		return 0, err
	}
	if f != nil {
		source = f.Text()
	}
	if source == "" {
		return endOffset + 1, nil
	}
	i := endOffset
	for i < len(source) {
		next := source[i]
		i++
		if next == '}' {
			break
		}
		if next == '/' {
			if i < len(source) {
				second := source[i]
				i++
				if second == '/' {
					for i < len(source) {
						ch := source[i]
						i++
						if ch == '\n' || ch == '\r' || ch == '\f' {
							break
						}
					}
				} else if second == '*' {
					for {
						char := source[i]
						i++
						if char != '*' {
							continue
						}
						for i < len(source) && source[i] == '*' {
							i++
						}
						if i < len(source) && source[i] == '/' {
							i++
							break
						}
					}
				}
			}
		}
	}
	return i, nil
}

// sourceLocationFromOffset creates a SourceLocation with line/column
// computed from the given offset within the source text, using a
// pre-built line starts cache for O(log n) lookup.
func sourceLocationFromOffset(lineStarts []int, offset int) sasscommon.SourceLocation {
	line := sort.SearchInts(lineStarts, offset+1) - 1
	if line < 0 {
		line = 0
	}
	col := offset - lineStarts[line]
	return sasscommon.SourceLocation{Offset: offset, Line: line, Column: col}
}

// buildLineStarts returns byte offsets of the start of each line in sourceText.
func buildLineStarts(sourceText string) []int {
	starts := []int{0}
	for i, b := range sourceText {
		if b == '\n' || (b == '\r' && (i+1 >= len(sourceText) || sourceText[i+1] != '\n')) {
			starts = append(starts, i+1)
		}
	}
	return starts
}
