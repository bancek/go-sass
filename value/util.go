// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/utils.dart (declarationName, toSentence)

import (
	"fmt"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// declarationName returns the variable name (including the leading $) from a
// span that covers a variable declaration, which includes the variable name as
// well as the colon and expression following it.
//
// This isn't particularly efficient, and should only be used for error messages.
//
// Matches Dart: declarationName (lib/src/utils.dart)
func declarationName(span sasscommon.FileSpan) (string, error) {
	text, err := span.SpanText()
	if err != nil {
		return "", err
	}
	colonIdx := strings.Index(text, ":")
	if colonIdx < 0 {
		return "", fmt.Errorf("declarationName: no colon found in span text %q", text)
	}
	return trimAsciiRight(text[:colonIdx], false), nil
}

// toSentence converts [items] into an English sentence, separating each word
// with [conjunction].
//
// Matches Dart: toSentence (lib/src/utils.dart)
func toSentence(items []string, conjunction string) string {
	if conjunction == "" {
		conjunction = "and"
	}
	if len(items) == 1 {
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " " + conjunction + " " + items[len(items)-1]
}
