// Copyright 2017 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

// dart-source: lib/src/importer/utils.dart (isValidUrlScheme section;
// resolveImportPath and the fromImport zone helpers live in
// resolve_import_path.go and importer_canonicalize_context.go)

import "regexp"

var validURLSchemeRegexp = regexp.MustCompile(`^[a-z0-9+.-]+$`)

// IsValidURLScheme returns whether scheme is a valid URL scheme.
//
// Only lowercase alphanumerics plus "+", ".", and "-" qualify, so the check
// stays a pure syntax gate with no filesystem access. Callers use it to tell
// genuine schemes apart from bare relative paths before dispatching to
// importers.
//
// Matches Dart: isValidUrlScheme in lib/src/importer/utils.dart
func IsValidURLScheme(scheme string) bool {
	return validURLSchemeRegexp.MatchString(scheme)
}
