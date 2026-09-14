// Copyright 2023 Google LLC. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasscommon

// dart-source: lib/src/util/lazy_file_span.dart

import (
	"net/url"
)

// LazyFileSpan is a FileSpan whose underlying span is built on first use
// and cached afterwards. It ports Dart's LazyFileSpan: every FileSpan
// member resolves the builder once, then delegates, so expensive span
// construction (interpolation mapping, deferred parsing) costs nothing
// unless the span is actually inspected.
//
// The resolved span is shared by all later calls; builders must therefore
// be deterministic.
//
// Matches Dart: LazyFileSpan (lib/src/util/lazy_file_span.dart)
type LazyFileSpan struct {
	builder func() (FileSpan, error)
	span    *FileSpan
}

var _ FileSpan = (*LazyFileSpan)(nil)

// NewLazyFileSpan creates a span that defers builder until the underlying
// span is first needed. Building never happens at construction time, so
// call sites can pass expensive computations without paying for spans that
// are never rendered.
//
// Matches Dart: LazyFileSpan constructor
func NewLazyFileSpan(builder func() (FileSpan, error)) *LazyFileSpan {
	return &LazyFileSpan{builder: builder}
}

// Get resolves and caches the underlying span, running the builder only on
// the first call. Builder failures propagate and are not cached, so a later
// Get retries the build.
//
// Matches Dart: LazyFileSpan.span (cached builder getter)
func (l *LazyFileSpan) Get() (FileSpan, error) {
	if l.span == nil {
		s, err := l.builder()
		if err != nil {
			return nil, err
		}
		l.span = &s
	}
	return *l.span, nil
}

// --- FileSpan interface implementation (each resolves via Get, then delegates) ---

// StartLocation resolves the underlying span, then reports its start.
func (l *LazyFileSpan) StartLocation() (SourceLocation, error) {
	s, err := l.Get()
	if err != nil {
		return SourceLocation{}, err
	}
	return s.StartLocation()
}

// EndLocation resolves the underlying span, then reports its end.
func (l *LazyFileSpan) EndLocation() (SourceLocation, error) {
	s, err := l.Get()
	if err != nil {
		return SourceLocation{}, err
	}
	return s.EndLocation()
}

// SpanText resolves the underlying span, then reports its text.
func (l *LazyFileSpan) SpanText() (string, error) {
	s, err := l.Get()
	if err != nil {
		return "", err
	}
	return s.SpanText()
}

// Length resolves the underlying span, then reports its length.
func (l *LazyFileSpan) Length() (int, error) {
	s, err := l.Get()
	if err != nil {
		return 0, err
	}
	return s.Length()
}

// SourceURL resolves the underlying span, then reports its source URL.
func (l *LazyFileSpan) SourceURL() (*url.URL, error) {
	s, err := l.Get()
	if err != nil {
		return nil, err
	}
	return s.SourceURL()
}

// File resolves the underlying span, then reports its source file.
func (l *LazyFileSpan) File() (*FileSource, error) {
	s, err := l.Get()
	if err != nil {
		return nil, err
	}
	return s.File()
}

// Context resolves the underlying span, then reports its surrounding source context.
func (l *LazyFileSpan) Context() (string, error) {
	s, err := l.Get()
	if err != nil {
		return "", err
	}
	return s.Context()
}

// Expand resolves the underlying span, then expands the underlying span to cover other.
func (l *LazyFileSpan) Expand(other FileSpan) (FileSpan, error) {
	s, err := l.Get()
	if err != nil {
		return nil, err
	}
	return s.Expand(other)
}

// Subspan resolves the underlying span, then slices the underlying span.
func (l *LazyFileSpan) Subspan(start, end int) (FileSpan, error) {
	s, err := l.Get()
	if err != nil {
		return nil, err
	}
	return s.Subspan(start, end)
}

// TrimRight resolves the underlying span, then trims trailing whitespace from the underlying span.
func (l *LazyFileSpan) TrimRight() (FileSpan, error) {
	s, err := l.Get()
	if err != nil {
		return nil, err
	}
	return s.TrimRight()
}

// WithoutInitialAtRule resolves the underlying span, then strips a leading at-rule from the underlying span.
func (l *LazyFileSpan) WithoutInitialAtRule() (FileSpan, error) {
	s, err := l.Get()
	if err != nil {
		return nil, err
	}
	return s.WithoutInitialAtRule()
}

// InitialQuoted resolves the underlying span, then takes the leading quoted string of the underlying span.
func (l *LazyFileSpan) InitialQuoted() (FileSpan, error) {
	s, err := l.Get()
	if err != nil {
		return nil, err
	}
	return s.InitialQuoted()
}

// InitialIdentifier resolves the underlying span, then takes the leading identifier of the underlying span, folding in includeLeading characters first.
func (l *LazyFileSpan) InitialIdentifier(includeLeading int) (FileSpan, error) {
	s, err := l.Get()
	if err != nil {
		return nil, err
	}
	return s.InitialIdentifier(includeLeading)
}

// Before resolves the underlying span, then takes the region before sub within the underlying span.
func (l *LazyFileSpan) Before(sub FileSpan) (FileSpan, error) {
	s, err := l.Get()
	if err != nil {
		return nil, err
	}
	return s.Before(sub)
}

// After resolves the underlying span, then takes the region after sub within the underlying span.
func (l *LazyFileSpan) After(sub FileSpan) (FileSpan, error) {
	s, err := l.Get()
	if err != nil {
		return nil, err
	}
	return s.After(sub)
}

// Between resolves the underlying span, then takes the region between the underlying span end and other start.
func (l *LazyFileSpan) Between(other FileSpan) (FileSpan, error) {
	s, err := l.Get()
	if err != nil {
		return nil, err
	}
	return s.Between(other)
}

// Contains resolves the underlying span, then reports whether the underlying span contains target.
func (l *LazyFileSpan) Contains(target FileSpan) (bool, error) {
	s, err := l.Get()
	if err != nil {
		return false, err
	}
	return s.Contains(target)
}

// String resolves the underlying span, then renders the underlying span text.
func (l *LazyFileSpan) String() (string, error) {
	s, err := l.Get()
	if err != nil {
		return "", err
	}
	return s.String()
}

// Message resolves the underlying span, then renders the headed message for the underlying span.
func (l *LazyFileSpan) Message(message string, opts HighlightOptions) (string, error) {
	s, err := l.Get()
	if err != nil {
		return "", err
	}
	return s.Message(message, opts)
}

// Highlight resolves the underlying span, then renders the highlight for the underlying span.
func (l *LazyFileSpan) Highlight(opts HighlightOptions) (string, error) {
	s, err := l.Get()
	if err != nil {
		return "", err
	}
	return s.Highlight(opts)
}

// HighlightMultiple resolves the underlying span, then renders the multi-span highlight for the underlying span.
func (l *LazyFileSpan) HighlightMultiple(primaryLabel string, secondarySpans map[FileSpan]string, opts HighlightOptions) (string, error) {
	s, err := l.Get()
	if err != nil {
		return "", err
	}
	return s.HighlightMultiple(primaryLabel, secondarySpans, opts)
}

// MessageMultiple resolves the underlying span, then renders the headed multi-span message for the underlying span.
func (l *LazyFileSpan) MessageMultiple(message, primaryLabel string, secondarySpans map[FileSpan]string, opts HighlightOptions) (string, error) {
	s, err := l.Get()
	if err != nil {
		return "", err
	}
	return s.MessageMultiple(message, primaryLabel, secondarySpans, opts)
}
