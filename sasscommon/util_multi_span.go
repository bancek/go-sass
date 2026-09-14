// Copyright 2022 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasscommon

// dart-source: lib/src/util/multi_span.dart

import (
	"maps"
	"net/url"
)

// MultiSpan is a FileSpan wrapper with secondary spans attached, so that
// Message can create multi-span messages.
//
// This is used to transparently support multi-span messages in situations that
// need to be backwards-compatible with single spans, such as logger
// invocations. To match the source_span package, separate APIs should
// generally be preferred over this class wherever backwards compatibility
// isn't a concern.
//
// Matches Dart: MultiSpan (lib/src/util/multi_span.dart)
type MultiSpan struct {
	// Primary is the span highlighted as the main location.
	Primary FileSpan
	// PrimaryLabel labels Primary to distinguish it from SecondarySpans.
	PrimaryLabel string
	// SecondarySpans maps each extra span to the label shown beside it.
	SecondarySpans map[FileSpan]string
}

// NewMultiSpan wraps primary with primaryLabel and a copy of secondarySpans,
// so later mutations of the caller's map cannot leak into the span. The
// defensive copy mirrors Dart's unmodifiable-map construction.
//
// Matches Dart: MultiSpan constructor
func NewMultiSpan(primary FileSpan, primaryLabel string, secondarySpans map[FileSpan]string) *MultiSpan {
	ss := make(map[FileSpan]string, len(secondarySpans))
	maps.Copy(ss, secondarySpans)
	return &MultiSpan{Primary: primary, PrimaryLabel: primaryLabel, SecondarySpans: ss}
}

// SpanStart returns the primary span's start location; ordering and
// messaging key off the primary span alone.
//
// Matches Dart: MultiSpan.start (delegated to the primary span)
func (m *MultiSpan) SpanStart() (SourceLocation, error) { return m.Primary.StartLocation() }

// SpanEnd returns the primary span's end location.
//
// Matches Dart: MultiSpan.end (delegated to the primary span)
func (m *MultiSpan) SpanEnd() (SourceLocation, error) { return m.Primary.EndLocation() }

// SpanText returns the primary span's covered text.
//
// Matches Dart: MultiSpan.text (delegated to the primary span)
func (m *MultiSpan) SpanText() (string, error) { return m.Primary.SpanText() }

// Context returns the surrounding source lines of the primary span.
// Matches Dart: MultiSpan.context
func (m *MultiSpan) Context() (string, error) { return m.Primary.Context() }

// SourceURL returns the source URL of the primary span.
// Matches Dart: MultiSpan.sourceUrl
func (m *MultiSpan) SourceURL() (*url.URL, error) { return m.Primary.SourceURL() }

// SpanFile returns the primary span's source URL as a string, or "" when
// the URL is unknown. It exists for string-oriented logging paths that
// cannot carry a URL value.
func (m *MultiSpan) SpanFile() (string, error) {
	u, err := m.Primary.SourceURL()
	if err != nil {
		return "", err
	}
	if u != nil {
		return u.String(), nil
	}
	return "", nil
}

// SpanLength returns the primary span's length in characters.
//
// Matches Dart: MultiSpan.length (delegated to the primary span)
func (m *MultiSpan) SpanLength() (int, error) { return m.Primary.Length() }

// String returns the primary span's text, so a MultiSpan prints as its
// primary span wherever a plain span is expected.
//
// Matches Dart: MultiSpan.toString (delegated to the primary span)
func (m *MultiSpan) String() (string, error) { return m.Primary.String() }

// CompareTo orders this span against other by start offset: -1 when the
// primary starts first, 1 when it starts later, 0 on ties. Length is not a
// tiebreaker here (unlike the base span comparison), because multi-spans
// sort by position only for message rendering.
//
// Matches Dart: MultiSpan.compareTo (delegated to the primary span)
func (m *MultiSpan) CompareTo(other FileSpan) (int, error) {
	loc, err := m.Primary.StartLocation()
	if err != nil {
		return 0, err
	}
	otherLoc, err := other.StartLocation()
	if err != nil {
		return 0, err
	}
	if loc.Offset < otherLoc.Offset {
		return -1, nil
	}
	if loc.Offset > otherLoc.Offset {
		return 1, nil
	}
	return 0, nil
}

// Union returns a plain span covering both the primary span and other,
// dropping the secondary set: the union is a single range, so labels for
// the extra spans no longer apply.
//
// Matches Dart: MultiSpan.union (delegated to the primary span)
func (m *MultiSpan) Union(other FileSpan) (FileSpan, error) {
	expanded, err := m.Primary.Expand(other)
	if err != nil {
		return nil, err
	}
	return expanded, nil
}

// Expand returns a MultiSpan whose primary covers both the old primary and
// other, keeping the labels and secondaries attached to the widened span.
//
// Matches Dart: MultiSpan.expand
func (m *MultiSpan) Expand(other FileSpan) (*MultiSpan, error) {
	expanded, err := m.Primary.Expand(other)
	if err != nil {
		return nil, err
	}
	return m.withPrimary(expanded), nil
}

// Subspan returns a MultiSpan narrowed to [start, end) within the primary
// span, keeping the labels and secondaries. A nil end means the primary's
// end, expressed in primary-relative offsets since FileSpan subspans are
// caller-relative.
//
// Matches Dart: MultiSpan.subspan
func (m *MultiSpan) Subspan(start int, end *int) (*MultiSpan, error) {
	var e int
	if end != nil {
		e = *end
	} else {
		endLoc, err := m.Primary.EndLocation()
		if err != nil {
			return nil, err
		}
		startLoc, err := m.Primary.StartLocation()
		if err != nil {
			return nil, err
		}
		e = endLoc.Offset - startLoc.Offset
	}
	sub, err := m.Primary.Subspan(start, e)
	if err != nil {
		return nil, err
	}
	return m.withPrimary(sub), nil
}

// Highlight renders the primary plus secondary spans together. The color
// argument accepts either a bool (true selects default colors) or a string
// (used as the primary-span color); anything else disables color. The two
// shapes mirror Dart's dynamic color parameter, which Go's static types
// cannot express directly, so the selection is decoded into HighlightOptions
// here.
//
// Matches Dart: MultiSpan.highlight({dynamic color})
func (m *MultiSpan) Highlight(color any) (string, error) {
	var colorBool bool
	var primaryColor string
	if c, ok := color.(bool); ok {
		colorBool = c
	} else if c, ok := color.(string); ok {
		colorBool = true
		primaryColor = c
	}
	opts := HighlightOptions{Color: colorBool, PrimaryColor: primaryColor}
	return m.Primary.HighlightMultiple(m.PrimaryLabel, m.SecondarySpans, opts)
}

// Message renders message with the "line/column" header plus the primary
// and secondary spans highlighted together. The color argument follows the
// same bool-or-string convention as Highlight.
//
// Matches Dart: MultiSpan.message(String message, {dynamic color})
func (m *MultiSpan) Message(message string, color any) (string, error) {
	var colorBool bool
	var primaryColor string
	if c, ok := color.(bool); ok {
		colorBool = c
	} else if c, ok := color.(string); ok {
		colorBool = true
		primaryColor = c
	}
	opts := HighlightOptions{Color: colorBool, PrimaryColor: primaryColor}
	return m.Primary.MessageMultiple(message, m.PrimaryLabel, m.SecondarySpans, opts)
}

// HighlightMultiple renders the highlight with additionalSpans merged into
// the secondary set under newLabel as the primary label. The receiver's map
// is never mutated: the merge builds a fresh map, so shared MultiSpans stay
// safe to reuse.
//
// Matches Dart: MultiSpan.highlightMultiple(...)
func (m *MultiSpan) HighlightMultiple(newLabel string, additionalSpans map[FileSpan]string, opts HighlightOptions) (string, error) {
	merged := mergeSecondarySpans(m.SecondarySpans, additionalSpans)
	return m.Primary.HighlightMultiple(newLabel, merged, opts)
}

// MessageMultiple renders the headed message with additionalSpans merged
// into the secondary set under newLabel, mirroring HighlightMultiple with
// the "line/column/message" header prepended.
//
// Matches Dart: MultiSpan.messageMultiple(...)
func (m *MultiSpan) MessageMultiple(message, newLabel string, additionalSpans map[FileSpan]string, opts HighlightOptions) (string, error) {
	merged := mergeSecondarySpans(m.SecondarySpans, additionalSpans)
	return m.Primary.MessageMultiple(message, newLabel, merged, opts)
}

// withPrimary copies m with newPrimary as the primary span, sharing the
// label and secondary map with the original. Sharing is safe because the
// maps are treated as immutable after construction (merges always copy).
func (m *MultiSpan) withPrimary(newPrimary FileSpan) *MultiSpan {
	return &MultiSpan{Primary: newPrimary, PrimaryLabel: m.PrimaryLabel, SecondarySpans: m.SecondarySpans}
}

// mergeSecondarySpans combines two secondary-span maps into a fresh map,
// with b's labels winning on key collisions. Every highlight/message merge
// goes through here so no caller can mutate a stored map in place.
func mergeSecondarySpans(a, b map[FileSpan]string) map[FileSpan]string {
	result := make(map[FileSpan]string, len(a)+len(b))
	maps.Copy(result, a)
	maps.Copy(result, b)
	return result
}

// multiSpanFileSpan embeds a FileSpan while overriding Highlight and Message
// to render the stored secondary spans automatically. It matches Dart's
// MultiSpan, which IS a SourceSpan via its mixin: code holding a plain
// FileSpan still gets multi-span output without opting in per call.
type multiSpanFileSpan struct {
	FileSpan
	primaryLabel   string
	secondarySpans map[FileSpan]string
}

// NewMultiSpanFileSpan wraps primary so its Highlight and Message methods
// include the secondary spans under primaryLabel automatically. Use it where
// a plain FileSpan slot must still render multi-span output; prefer the
// explicit HighlightMultiple/MessageMultiple APIs elsewhere, as Dart's docs
// advise.
//
// Matches Dart: MultiSpan constructor (as a FileSpan implementation)
func NewMultiSpanFileSpan(primary FileSpan, primaryLabel string, secondarySpans map[FileSpan]string) FileSpan {
	return &multiSpanFileSpan{
		FileSpan:       primary,
		primaryLabel:   primaryLabel,
		secondarySpans: secondarySpans,
	}
}

// Highlight renders the embedded span together with the stored secondaries.
func (m *multiSpanFileSpan) Highlight(opts HighlightOptions) (string, error) {
	return m.FileSpan.HighlightMultiple(m.primaryLabel, m.secondarySpans, opts)
}

// Message renders the headed message with the stored secondaries included.
func (m *multiSpanFileSpan) Message(message string, opts HighlightOptions) (string, error) {
	return m.FileSpan.MessageMultiple(message, m.primaryLabel, m.secondarySpans, opts)
}
