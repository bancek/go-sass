// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/css/comment.dart

// CssComment is a plain CSS comment. This is always a multi-line comment.
//
// Matches Dart: CssComment. Text includes the delimiters; IsPreserved marks
// `/*!` loud comments kept in compressed mode.
type CssComment interface {
	CssNode
	// Text returns the comment including /* and */.
	Text() string
	// IsPreserved reports whether the comment starts with /*!.
	IsPreserved() bool
}
