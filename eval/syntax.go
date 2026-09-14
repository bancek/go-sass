// Copyright 2018 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

import (
	"path/filepath"
)

// dart-source: lib/src/syntax.dart
// Syntax is an enum of syntaxes that Sass can parse.
type Syntax int

const (
	// SyntaxSCSS is the CSS-superset SCSS syntax.
	SyntaxSCSS Syntax = iota
	// SyntaxSass is the whitespace-sensitive indented syntax.
	SyntaxSass
	// SyntaxCSS is the plain CSS syntax, which disallows special Sass features.
	SyntaxCSS
)

// String returns the display name of the syntax ("SCSS", "Sass", "CSS").
//
// Matches Dart: Syntax.toString
func (s Syntax) String() string {
	switch s {
	case SyntaxSCSS:
		return "SCSS"
	case SyntaxSass:
		return "Sass"
	case SyntaxCSS:
		return "CSS"
	default:
		return ""
	}
}

// SyntaxForPath returns the default syntax to use for a file loaded from path.
//
// The match is on the path extension and is case-sensitive: ".sass" selects
// the indented syntax, ".css" selects plain CSS, and everything else —
// including extensionless paths and dotfiles such as ".sass", whose extension
// is empty — falls through to SCSS.
func SyntaxForPath(path string) Syntax {
	// Dart: p.extension(path) is case-sensitive — syntax.dart
	ext := filepath.Ext(path)
	switch ext {
	case ".sass":
		return SyntaxSass
	case ".css":
		return SyntaxCSS
	default:
		return SyntaxSCSS
	}
}
