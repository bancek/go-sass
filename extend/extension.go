// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package extend

// dart-source: lib/src/extend/extension.dart

import (
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/value"
)

// Extension describes one @extend rule: an extender selector reaching a
// target simple selector. The target itself lives in the store's extension
// map key; the Extension carries the extender, the rule span (always a
// mandatory rule's span when any merged branch is mandatory), the media
// context it is restricted to (nil means any context), and whether the rule
// was marked !optional.
type Extension interface {
	// Extender returns the selector doing the extending, such as A in
	// `A {@extend B}`.
	Extender() *Extender
	// Target returns the simple selector being extended.
	Target() value.SimpleSelector
	// MediaContext returns the media context the extension is restricted
	// to, or nil when it applies in any context.
	MediaContext() []*value.CssMediaQuery
	// IsOptional reports whether the @extend rule was marked !optional.
	IsOptional() bool
	// Span returns the source span of the @extend rule.
	Span() (sasscommon.FileSpan, error)
	// IsMerged reports whether this extension merges two branches with the
	// same extender and target.
	IsMerged() bool
	// WithExtender returns a copy of the extension with a new extender
	// selector, preserving target, span, media context, and optionality.
	WithExtender(*value.ComplexSelector) Extension
	// String renders the extension as `extender {@extend target}` with an
	// ` !optional` suffix when optional.
	String() (string, error)
}

// BaseExtension is a single non-merged @extend rule. Merged rules embed it
// and add the left/right branches they were folded from.
type BaseExtension struct {
	extender     *Extender
	target       value.SimpleSelector
	mediaContext []*value.CssMediaQuery
	isOptional   bool
	span         sasscommon.FileSpan
}

// NewExtension records extender {@extend target} for the rule at span.
// A nil mediaContext means the extension applies in any media context.
// The extender is wired back to the new extension so media-context checks
// can find the rule that created it.
func NewExtension(
	extender *value.ComplexSelector,
	target value.SimpleSelector,
	span sasscommon.FileSpan,
	mediaContext []*value.CssMediaQuery,
	optional bool,
) Extension {
	e := &BaseExtension{}
	e.target = target
	e.span = span
	if mediaContext != nil {
		e.mediaContext = mediaContext
	}
	e.isOptional = optional
	e.extender = NewExtender(extender, nil, false)
	e.extender.extension = e
	return e
}

// Extender returns the selector doing the extending.
func (e *BaseExtension) Extender() *Extender { return e.extender }

// Target returns the simple selector being extended.
func (e *BaseExtension) Target() value.SimpleSelector { return e.target }

// MediaContext returns the media context the extension is restricted to,
// or nil when it applies anywhere.
func (e *BaseExtension) MediaContext() []*value.CssMediaQuery { return e.mediaContext }

// IsOptional reports whether the rule was marked !optional.
func (e *BaseExtension) IsOptional() bool { return e.isOptional }

// Span returns the source span of the @extend rule.
func (e *BaseExtension) Span() (sasscommon.FileSpan, error) { return e.span, nil }

// IsMerged reports false for base extensions.
func (e *BaseExtension) IsMerged() bool { return false }

// WithExtender returns an equivalent extension with newExtender as extender.
func (e *BaseExtension) WithExtender(newExtender *value.ComplexSelector) Extension {
	return NewExtension(newExtender, e.target, e.span, e.mediaContext, e.isOptional)
}

// String renders the extension as `extender {@extend target}`, using the
// target's source text and appending ` !optional` when set.
func (e *BaseExtension) String() (string, error) {
	span, err := e.target.(sasscommon.AstNode).Span()
	if err != nil {
		return "", err
	}
	ss, err := span.String()
	if err != nil {
		return "", err
	}
	extenderStr, err := e.extender.String()
	if err != nil {
		return "", err
	}
	s := extenderStr + " {@extend " + ss
	if e.isOptional {
		s += " !optional"
	}
	return s + "}", nil
}

// Extender is a selector that extends another selector, such as A in
// `A {@extend B}`. It carries the originating selector, the minimum
// specificity any generated selector must keep (so trimming cannot weaken
// the extender below its source), and whether it represents a selector that
// was originally in the document rather than one introduced by @extend.
type Extender struct {
	Selector    *value.ComplexSelector
	Specificity int
	IsOriginal  bool
	extension   Extension
}

// NewExtender wraps sel as an extender. A nil specificity defaults to the
// selector's own specificity; original marks document-original selectors so
// trimming preserves them under the first law of extend.
func NewExtender(sel *value.ComplexSelector, specificity *int, original bool) *Extender {
	spec := sel.Specificity()
	if specificity != nil {
		spec = *specificity
	}
	return &Extender{
		Selector:    sel,
		Specificity: spec,
		IsOriginal:  original,
	}
}

// AssertCompatibleMediaContext rejects applying this extender in
// mediaContext when the extension was defined in a different media context.
// Extenders without a creating extension, and extensions defined at the top
// level, are compatible anywhere; a mismatch reports the @extend rule's
// span.
func (e *Extender) AssertCompatibleMediaContext(mediaContext []*value.CssMediaQuery) error {
	ext := e.extension
	if ext == nil {
		return nil
	}
	expectedMediaContext := ext.MediaContext()
	if expectedMediaContext == nil {
		return nil
	}
	if mediaContext != nil && mediaQueriesEqual(expectedMediaContext, mediaContext) {
		return nil
	}
	span, err := ext.Span()
	if err != nil {
		return err
	}
	return &sasscommon.SassException{
		Message: "You may not @extend selectors across media queries.",
		Span:    span,
	}
}

// String renders the extender as its selector text.
func (e *Extender) String() (string, error) {
	return e.Selector.String()
}
