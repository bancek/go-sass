// Copyright 2020 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/supports_condition/anything.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// SupportsAnything is a forwards-compatible <general-enclosed> condition:
// arbitrary parenthesized text preserved for browsers to interpret.
type SupportsAnything struct {
	// Contents is the opaque condition text.
	Contents *Interpolation
	span     sasscommon.FileSpan
}

// NewSupportsAnything creates an opaque general-enclosed condition.
func NewSupportsAnything(contents *Interpolation, span sasscommon.FileSpan) *SupportsAnything {
	return &SupportsAnything{Contents: contents, span: span}
}

func (a *SupportsAnything) Span() (sasscommon.FileSpan, error) { return a.span, nil }
func (a *SupportsAnything) IsAstNode()                         {}
func (a *SupportsAnything) IsSassNode()                        {}
func (a *SupportsAnything) IsSupportsCondition()               {}

// ToInterpolation flattens the condition into source-equivalent text,
// preserving the literal around the interpolated contents.
func (a *SupportsAnything) ToInterpolation() (*Interpolation, error) {
	buf := &InterpolationBuffer{}
	contentsSpan, err := a.Contents.Span()
	if err != nil {
		return nil, err
	}
	beforeSpan, err := a.span.Before(contentsSpan)
	if err != nil {
		return nil, err
	}
	before, err := beforeSpan.SpanText()
	if err != nil {
		return nil, err
	}
	buf.Write(before)
	buf.AddInterpolation(a.Contents)
	afterSpan, err := a.span.After(contentsSpan)
	if err != nil {
		return nil, err
	}
	after, err := afterSpan.SpanText()
	if err != nil {
		return nil, err
	}
	buf.Write(after)
	return buf.Interpolation(a.span)
}

// WithSpan returns a copy of this condition covering span.
func (a *SupportsAnything) WithSpan(span sasscommon.FileSpan) SupportsCondition {
	return NewSupportsAnything(a.Contents, span)
}
func (a *SupportsAnything) String() (string, error) {
	s, err := a.Contents.String()
	if err != nil {
		return "", err
	}
	return "(" + s + ")", nil
}
