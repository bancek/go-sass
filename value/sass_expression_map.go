// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/expression/map.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// MapExpression is a map literal.
//
// Matches Dart: MapExpression
type MapExpression struct {
	// Pairs are the entries of this map. This is a list of pairs rather
	// than a map because two keys may hold equal expressions (for example
	// from unique-id() calls) and both entries must be preserved.
	Pairs []struct {
		Key   Expression
		Value Expression
	}
	span sasscommon.FileSpan
}

// NewMapExpression creates a map literal, copying pairs.
//
// Matches Dart: MapExpression.new
func NewMapExpression(pairs []struct {
	Key   Expression
	Value Expression
}, span sasscommon.FileSpan) *MapExpression {
	immutable := make([]struct {
		Key   Expression
		Value Expression
	}, len(pairs))
	copy(immutable, pairs)
	return &MapExpression{Pairs: immutable, span: span}
}

func (e *MapExpression) Span() (sasscommon.FileSpan, error)  { return e.span, nil }
func (e *MapExpression) SourceInterpolation() *Interpolation { return nil }
func (e *MapExpression) IsExpression()                       {}
func (e *MapExpression) IsSassNode()                         {}
func (e *MapExpression) IsAstNode()                          {}

func (e *MapExpression) String() (string, error) {
	parts := make([]string, len(e.Pairs))
	for i, p := range e.Pairs {
		keyStr, err := p.Key.String()
		if err != nil {
			return "", err
		}
		valStr, err := p.Value.String()
		if err != nil {
			return "", err
		}
		parts[i] = keyStr + ": " + valStr
	}
	return "(" + strings.Join(parts, ", ") + ")", nil
}
