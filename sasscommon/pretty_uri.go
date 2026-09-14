// Copyright (c) 2012, the Dart project authors.  Please see the AUTHORS file
// for details. All rights reserved. Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasscommon

// dart-source: (external) package:path

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// PrettyUri renders uri for human consumption: file: URIs become paths
// relative to the working directory whenever the relative form is no longer
// than the absolute one (avoiding ugly "../" chains back to the root),
// and every other scheme renders as-is.
//
// It ports Dart's Context.prettyUri from package:path, which likewise
// prefers the relative path only when it is actually shorter, and may
// return either URI- or path-formatted text. The empty-scheme branch covers
// plain paths that carry no scheme.
//
// Matches Dart: Context.prettyUri (package:path)
func PrettyUri(uri *url.URL) string {
	if uri.Scheme == "file" || uri.Scheme == "" {
		absPath := uri.Path
		if cwd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(cwd, absPath); err == nil {
				if len(strings.Split(rel, string(filepath.Separator))) <=
					len(strings.Split(absPath, string(filepath.Separator))) {
					return filepath.ToSlash(rel)
				}
			}
		}
		return absPath
	}
	return uri.String()
}
