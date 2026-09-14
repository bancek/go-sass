// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/import/static.dart

import (
	"github.com/bancek/go-sass/sasscommon"
)

// StaticImport is an import that produces a plain CSS @import rule.
//
// Matches Dart: StaticImport
type StaticImport struct {
	// URL is the URL for this import, already containing quotes.
	URL *Interpolation
	// Modifiers are the media or supports queries attached to this import,
	// or nil when none are attached.
	Modifiers *Interpolation
	span      sasscommon.FileSpan
}

// NewStaticImport creates a plain-CSS import with the given URL, span, and
// optional modifiers.
//
// Matches Dart: StaticImport.new
func NewStaticImport(url *Interpolation, span sasscommon.FileSpan, modifiers *Interpolation) *StaticImport {
	return &StaticImport{URL: url, span: span, Modifiers: modifiers}
}

func (i *StaticImport) Span() (sasscommon.FileSpan, error) { return i.span, nil }
func (i *StaticImport) IsImport()                          {}
func (i *StaticImport) IsSassNode()                        {}
func (i *StaticImport) IsAstNode()                         {}

// String returns a debug representation.
// Note: This differs structurally from Dart's StaticImport.toString() which
// is infallible. Go Interpolation.String() can return errors (a Go language
// limitation replacing Dart's exception-free toString()). The serialized
// output is identical for all well-formed AST nodes.
func (i *StaticImport) String() (string, error) {
	if i.Modifiers != nil {
		urlStr, err := i.URL.String()
		if err != nil {
			return "", err
		}
		modStr, err := i.Modifiers.String()
		if err != nil {
			return "", err
		}
		return urlStr + " " + modStr, nil
	}
	return i.URL.String()
}
