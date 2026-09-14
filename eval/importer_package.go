// Copyright 2017 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

import (
	"fmt"
	"net/url"
	"time"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassio"
)

// dart-source: lib/src/importer/package.dart
// PackageImporter resolves package: imports to file: URIs using a
// [PackageConfig].
//
// A package: URL is first mapped to a file: URL through the resolver, then
// handed to the working-directory filesystem importer for the usual
// partial/extension/index lookup and loading.
//
// Matches Dart: PackageImporter in lib/src/importer/package.dart
type PackageImporter struct {
	config PackageConfig
	io     sassio.IO
	cwd    *FilesystemImporter
}

// NewPackageImporter creates a PackageImporter that resolves package: URLs
// through config and loads the resulting files relative to the working
// directory.
func NewPackageImporter(config PackageConfig, io sassio.IO) *PackageImporter {
	return &PackageImporter{
		config: config,
		io:     io,
		cwd:    NewFilesystemImporterCwd(io),
	}
}

// Canonicalize resolves u to an absolute file: URL, or returns nil when
// another importer should handle it.
//
// File: URLs delegate straight to the working-directory filesystem importer.
// A package: URL is mapped through the package configuration first: an
// unknown package fails with "Unknown package.", a resolution to a
// non-file: scheme fails with "Unsupported URL ...", and anything else falls
// through to the same filesystem lookup. All other schemes return nil.
//
// Matches Dart: PackageImporter.canonicalize
func (i *PackageImporter) Canonicalize(u *url.URL, ctx *CanonicalizeContext) (*url.URL, error) {
	if u.Scheme == "file" {
		return i.cwd.Canonicalize(u, ctx)
	}
	if u.Scheme != "package" {
		return nil, nil
	}

	resolved := i.config.Resolve(u)
	if resolved == nil {
		return nil, sasscommon.NewSassScriptException("Unknown package.", nil)
	}
	if resolved.Scheme != "" && resolved.Scheme != "file" {
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Unsupported URL %s.", resolved), nil)
	}

	return i.cwd.Canonicalize(resolved, ctx)
}

// Load reads a previously canonicalized file: URL through the
// working-directory filesystem importer.
//
// Matches Dart: PackageImporter.load
func (i *PackageImporter) Load(u *url.URL) (*ImporterResult, error) {
	return i.cwd.Load(u)
}

// CouldCanonicalize reports whether u could resolve through the package
// configuration: file:, package:, and relative URLs are candidates, judged by
// path basename the way the filesystem importer judges them. Anything else
// belongs to another importer.
//
// Matches Dart: PackageImporter.couldCanonicalize
func (i *PackageImporter) CouldCanonicalize(u *url.URL, canonicalURL *url.URL) bool {
	if u.Scheme != "file" && u.Scheme != "package" && u.Scheme != "" {
		return false
	}
	return i.cwd.CouldCanonicalize(&url.URL{Path: u.Path}, canonicalURL)
}

// IsNonCanonicalScheme always returns false: package: URLs canonicalize to
// file: URLs, so no scheme needs load-site-dependent resolution.
func (i *PackageImporter) IsNonCanonicalScheme(scheme string) bool {
	return false
}

// ModificationTime reports the on-disk modification time through the
// working-directory filesystem importer.
//
// Matches Dart: PackageImporter.modificationTime
func (i *PackageImporter) ModificationTime(canonicalURL *url.URL) (time.Time, error) {
	return i.cwd.ModificationTime(canonicalURL)
}

// String returns a human-readable description of this importer.
//
// Matches Dart: PackageImporter.toString
func (i *PackageImporter) String() string {
	return "package:..."
}
