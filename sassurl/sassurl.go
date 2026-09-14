// Package sassurl provides Dart-compatible URL parsing for Sass.
//
// Go's url.Parse puts data for opaque URIs (like "sass:color") in u.Opaque
// instead of u.Path, while Dart's Uri.parse puts them in path.
// Parse normalizes this so that u.Path always contains the path data and
// u.String() still returns the original URL form.
package sassurl

import (
	"net/url"
	"path"
	"strings"
)

// Parse parses a raw URL string and normalizes it for Sass compatibility.
//
// For opaque URIs like "sass:color", Go puts the path data in u.Opaque.
// Dart's Uri.parse puts it in path. Parse normalizes this by moving u.Opaque
// to u.RawPath (preserving the original encoding for String()), setting
// u.Path to the decoded form, and setting u.OmitHost so that u.String()
// returns the original form (e.g. "sass:color" rather than "sass://color").
//
// Matches Dart: Uri.parse
func Parse(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Opaque != "" && u.Path == "" {
		if decoded, dErr := url.PathUnescape(u.Opaque); dErr == nil {
			u.RawPath = u.Opaque
			u.Path = decoded
		} else {
			u.Path = u.Opaque
		}
		u.Opaque = ""
		u.OmitHost = true
	}
	return u, nil
}

// Resolve resolves a URI reference against a base URI, matching Dart's
// Uri.resolve semantics for opaque URIs.
//
// Go's url.URL.ResolveReference does not correctly resolve relative URLs
// against opaque URIs (like "u:foo/bar"), because it treats the opaque
// part as non-hierarchical. Dart's Uri.resolve treats the path as
// hierarchical. This function handles the opaque case by manually
// resolving the path components.
//
// Matches Dart: Uri.resolve
func Resolve(base *url.URL, ref *url.URL) *url.URL {
	if ref.IsAbs() {
		return ref
	}
	// Determine the base path: either from Opaque (for url.Parse result)
	// or from Path when OmitHost is set (for Parse-normalized result).
	basePath := base.Opaque
	if basePath == "" && base.OmitHost && base.Path != "" {
		basePath = base.Path
	}
	if basePath == "" {
		return base.ResolveReference(ref)
	}

	refPath := refPath(ref)
	if strings.HasPrefix(refPath, "/") {
		// Absolute path relative to scheme
		cleaned := path.Clean(refPath)
		cleaned = strings.TrimPrefix(cleaned, "/")
		return &url.URL{Scheme: base.Scheme, Opaque: cleaned}
	}
	dir := basePath
	if idx := strings.LastIndex(dir, "/"); idx >= 0 {
		dir = dir[:idx]
	} else {
		dir = ""
	}
	var resolved string
	if dir == "" {
		resolved = refPath
	} else {
		resolved = dir + "/" + refPath
	}
	cleaned := path.Clean(resolved)
	if cleaned == "." {
		cleaned = ""
	}
	return &url.URL{Scheme: base.Scheme, Opaque: cleaned}
}

// refPath returns the path component of a reference URL, matching Dart's
// resolution logic which uses the path for relative references.
func refPath(ref *url.URL) string {
	if ref.Opaque != "" {
		return ref.Opaque
	}
	return ref.Path
}
