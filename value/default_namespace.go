package value

// dart-source: lib/src/utils.dart (defaultNamespace)

import (
	"strings"

	"github.com/bancek/go-sass/sassurl"
)

// DefaultNamespace returns the default Sass namespace for a stylesheet URL.
//
// The second result is false when the derived name is not a valid Sass
// identifier; callers still get the raw name for error reporting.
//
// Matches Dart: defaultNamespace
func DefaultNamespace(u *sassurl.URL) (string, bool) {
	path := sassurl.DecodeRelativePath(u.Path)
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		path = path[idx+1:]
	}
	path = strings.TrimPrefix(path, "_")
	if dot := strings.Index(path, "."); dot >= 0 {
		path = path[:dot]
	}
	if _, err := ParseIdentifier(path); err != nil {
		return path, false
	}
	return path, true
}
