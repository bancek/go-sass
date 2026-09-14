// Copyright 2018 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

// TODO move to separate files, matching dart files

// dart-source: lib/src/import_cache.dart (ImportCache, CanonicalizeResult) +
// lib/src/importer/filesystem.dart (FilesystemImporter section, merged here;
// the remaining lib/src/importer/*.dart kinds live in importer*.go and
// node_package_importer.go)

import (
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/functions"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassio"
	"github.com/bancek/go-sass/sassurl"
	"github.com/bancek/go-sass/value"
)

// CanonicalizeResult is a canonicalized URL and the importer that canonicalized it.
//
// This also includes the URL that was originally passed to the importer, which
// may be resolved relative to a base URL. The cache keeps the original around
// so stack traces and invalidation can speak in the URL the user wrote rather
// than the resolved file path.
type CanonicalizeResult struct {
	// Importer that canonicalized the URL.
	Importer Importer
	// Canonical URL returned by the importer.
	CanonicalURL *url.URL
	// URL passed to the importer, resolved relative to the base URL when one
	// applies.
	OriginalURL *url.URL
}

// PackageConfig resolves package: URLs to file: URLs.
//
// This mirrors the resolver shape from Dart's package_config package: a pure
// mapping consulted before any filesystem lookup, kept as an interface so
// hosts can substitute their own resolution.
type PackageConfig interface {
	// Resolve resolves a package: URL to a file: URL, or returns nil if the
	// package is unknown.
	Resolve(u *url.URL) *url.URL
}

// FilesystemImporter loads files from a load path on the filesystem, either
// relative to the path passed to [NewFilesystemImporter] or absolute `file:`
// URLs.
//
// Use [NewFilesystemImporterNoLoadPath] to only load absolute `file:` URLs and
// URLs relative to the current file. Folded into this file from Dart's
// importer/filesystem.dart because the import cache owns the importer chain;
// the remaining importer kinds keep one file each.
//
// Matches Dart: FilesystemImporter in lib/src/importer/filesystem.dart
type FilesystemImporter struct {
	// The path relative to which this importer looks for files.
	//
	// If this is nil, this importer will only load absolute file: URLs and URLs
	// relative to the current file.
	loadPath *string

	// Whether loading from files from this importer's loadPath is deprecated.
	// Set only for the implicit working-directory importer, so the
	// deprecation fires exactly when that fallback resolves a load.
	loadPathDeprecated bool

	// The IO implementation used for filesystem operations. If nil,
	// [DefaultIO] is used.
	io sassio.IO

	// Go-specific adaptation: Dart's top-level warnForDeprecation function is
	// accessible through its evaluation context. Go lacks Zone, so the warn
	// function is stored on the importer directly.
	warnFn func(message string, deprecation *deprecation.Deprecation) error
}

// NewFilesystemImporter creates an importer that loads files relative to
// loadPath.
//
// The loadPath is converted to an absolute path, so later renames of the
// working directory cannot shift resolution out from under the cache.
//
// Matches Dart: FilesystemImporter(loadPath)
func NewFilesystemImporter(loadPath string, io sassio.IO) *FilesystemImporter {
	abs, err := filepath.Abs(loadPath)
	if err != nil {
		abs = loadPath
	}
	return &FilesystemImporter{loadPath: &abs, io: io}
}

// NewFilesystemImporterCwd creates a FilesystemImporter that loads files
// relative to the current working directory.
//
// Deprecated: Use NewFilesystemImporterNoLoadPath or NewFilesystemImporter(".")
// instead. Historically this was the convenient default for file: URL loads
// where the load path did not matter, but an implicit working directory on
// the load path proved surprising, so callers should now choose explicitly
// between no load path and an explicit "." entry. Matches Dart:
// FilesystemImporter.cwd
func NewFilesystemImporterCwd(io sassio.IO) *FilesystemImporter {
	abs, err := filepath.Abs(".")
	if err != nil {
		return &FilesystemImporter{loadPath: new(string), io: io}
	}
	return &FilesystemImporter{loadPath: &abs, loadPathDeprecated: true, io: io}
}

// NewFilesystemImporterNoLoadPath creates an importer that only loads absolute
// file: URLs and URLs relative to the current file.
//
// With no load path, scheme-less URLs never fall back to the working
// directory: only an explicit file: URL (or a relative load against the base
// importer) can resolve.
//
// Matches Dart: FilesystemImporter.noLoadPath
func NewFilesystemImporterNoLoadPath(io sassio.IO) *FilesystemImporter {
	return &FilesystemImporter{io: io}
}

// Canonicalize resolves u to an absolute file: URL, or returns nil when
// another importer should handle it.
//
// Absolute file: URLs resolve directly through the import-path lookup.
// fileURLToOSPath converts a file: URL path to an OS filesystem path.
//
// URL paths always use forward slashes with a leading slash, so a Windows
// path arrives as "/C:/dir/file" — a form Go's filepath (no volume, not
// absolute) and the OS both reject. The leading slash is stripped before
// an ASCII drive letter, mirroring Dart's p.fromUri. Everywhere else the
// URL path is already a valid OS path (forward slashes are accepted
// Windows separators), so it passes through untouched. Unconditional, not
// runtime-gated: only Windows produces such paths, and Dart agrees with
// the stripped form there — the only OS where they exist.
//
// Matches Dart: p.fromUri in package:path (drive-letter branch).
func fileURLToOSPath(path string) string {
	if len(path) > 2 && path[0] == '/' && path[2] == ':' &&
		((path[1] >= 'A' && path[1] <= 'Z') || (path[1] >= 'a' && path[1] <= 'z')) {
		return path[1:]
	}
	return path
}

// Scheme-less URLs resolve against the load path; an importer without one
// only serves file: URLs and relative-to-current-file loads, so it declines
// rather than falling back to the working directory. Any other scheme belongs
// to a different importer. A hit is normalized through io.Canonicalize so the
// cache keys on the true on-disk location.
//
// Matches Dart: FilesystemImporter.canonicalize in
// lib/src/importer/filesystem.dart
func (i *FilesystemImporter) Canonicalize(u *url.URL, ctx *CanonicalizeContext) (*url.URL, error) {
	var resolved string
	if u.Scheme == "file" {
		r, err := resolveImportPath(i.io, fileURLToOSPath(u.Path), ctx.FromImport)
		if err != nil {
			return nil, err
		}
		resolved = r
	} else if u.Scheme != "" {
		return nil, nil
	} else if i.loadPath != nil {
		r, err := resolveImportPath(i.io, filepath.Join(*i.loadPath, u.Path), ctx.FromImport)
		if err != nil {
			return nil, err
		}
		resolved = r

		if resolved != "" && i.loadPathDeprecated {
			// The implicit working-directory load path still resolves, but
			// each successful use warns so callers migrate to an explicit
			// load path or importer.
			if i.warnFn != nil {
				if err := i.warnFn(
					"Using the current working directory as an implicit load path is "+
						"deprecated. Either add it as an explicit load path or importer, or "+
						"load this stylesheet from a different URL.",
					deprecation.FsImporterCwd,
				); err != nil {
					return nil, err
				}
			}
		}
	} else {
		return nil, nil
	}

	if resolved == "" {
		return nil, nil
	}

	canonicalized, err := i.io.Canonicalize(resolved)
	if err != nil {
		return nil, err
	}
	canonical := &url.URL{Scheme: "file", Path: filepath.ToSlash(canonicalized)}
	return canonical, nil
}

// Load reads the file at u and infers its parse syntax from the path.
//
// Only file: URLs are loadable here; anything else reports an error rather
// than nil because a canonical file URL uniquely belongs to this importer.
//
// Matches Dart: FilesystemImporter.load in lib/src/importer/filesystem.dart
func (i *FilesystemImporter) Load(u *url.URL) (*ImporterResult, error) {
	if u.Scheme != "file" {
		return nil, fmt.Errorf("cannot load non-file URL: %s", u)
	}
	contents, err := i.io.ReadFile(fileURLToOSPath(u.Path))
	if err != nil {
		return nil, err
	}
	syntax := SyntaxForPath(u.Path)
	return NewImporterResult(string(contents), syntax, u)
}

// CouldCanonicalize reports, without touching the filesystem, whether url
// could resolve to canonicalURL through this importer.
//
// Both sides must be file-like, and the basenames must match up to the "_"
// partial prefix and the file extension. Over-reporting is tolerated; a false
// negative would corrupt watch-mode invalidation.
//
// Matches Dart: FilesystemImporter.couldCanonicalize in
// lib/src/importer/filesystem.dart
func (i *FilesystemImporter) CouldCanonicalize(url *url.URL, canonicalURL *url.URL) bool {
	if url.Scheme != "file" && url.Scheme != "" {
		return false
	}
	if canonicalURL.Scheme != "file" {
		return false
	}

	basename := filepath.Base(url.Path)
	canonicalBasename := filepath.Base(canonicalURL.Path)
	if !strings.HasPrefix(basename, "_") && strings.HasPrefix(canonicalBasename, "_") {
		canonicalBasename = canonicalBasename[1:]
	}

	return basename == canonicalBasename ||
		basename == withoutExtension(canonicalBasename)
}

// IsNonCanonicalScheme always returns false: the filesystem importer only
// ever canonicalizes to file: URLs, so no scheme needs load-site-dependent
// resolution.
func (i *FilesystemImporter) IsNonCanonicalScheme(scheme string) bool {
	return false
}

// ModificationTime returns the modification time of the file at canonicalURL.
//
// Non-file: URLs have no on-disk time, so they report now (always stale);
// file: URLs stat through the io seam. There is intentionally no Go
// counterpart to Dart's bare modificationTime passthrough beyond this: the
// caller substitutes now on error.
//
// Matches Dart: FilesystemImporter.modificationTime in
// lib/src/importer/filesystem.dart
func (i *FilesystemImporter) ModificationTime(canonicalURL *url.URL) (time.Time, error) {
	if canonicalURL.Scheme != "file" {
		return time.Now(), nil
	}
	info, err := i.io.Stat(canonicalURL.Path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// String returns a human-readable description of this importer: the load path,
// or a placeholder when this importer serves absolute file URLs only.
//
// Matches Dart: FilesystemImporter.toString in lib/src/importer/filesystem.dart
func (i *FilesystemImporter) String() string {
	if i.loadPath == nil {
		return "<absolute file importer>"
	}
	return *i.loadPath
}

// canonicalizeKey is a composite key for canonicalization caches, pairing the
// URL text with the @import/@use distinction (and, for the per-importer
// cache, the importer consulted).
type canonicalizeKey struct {
	url       string
	importer  Importer
	forImport bool
}

// ImportCache is an in-memory cache of parsed stylesheets that have been
// imported by Sass.
//
// It resolves new imports through the importer chain and caches both the
// canonicalizations and the parsed stylesheets, so repeated loads reuse the
// existing parse tree. The cache is created up front and held by the
// evaluator for the run (ownership, not borrowing).
type ImportCache struct {
	// The importers to use when loading new Sass files.
	importers []Importer

	// Whether to parse StyleRule selectors when loading new Sass files.
	parseSelectors bool

	// The canonicalized URLs for each non-canonical URL.
	//
	// The forImport in each key is true when this canonicalization is for an
	// @import rule. Otherwise, it's for a @use or @forward rule.
	//
	// This cache covers loads that go through the entire chain of importers,
	// but it doesn't cover individual loads or loads in which any importer
	// accesses containingUrl.
	canonicalizeCache map[canonicalizeKey]*CanonicalizeResult

	// Like canonicalizeCache but also includes the specific importer in the key.
	//
	// This is used to cache both relative imports from the base importer and
	// individual importer results in the case where some other component of the
	// importer chain isn't cacheable.
	perImporterCanonicalizeCache map[canonicalizeKey]*CanonicalizeResult

	// A map from the keys in perImporterCanonicalizeCache that are generated for
	// relative URL loads against the base importer to the original relative URLs
	// that were loaded.
	//
	// This is used to invalidate the cache when files are changed.
	nonCanonicalRelativeUrls map[canonicalizeKey]*url.URL

	// The parsed stylesheets for each canonicalized import URL.
	importCache map[string]*value.Stylesheet

	// The import results for each canonicalized import URL.
	resultsCache map[string]*ImporterResult

	// A map from canonical URLs to the most recent time at which those URLs were
	// loaded from their importers.
	loadTimes map[string]time.Time

	// warnFn is called to emit deprecation warnings during canonicalization.
	warnFn func(message string, deprecation *deprecation.Deprecation) error
}

// NewImportCache creates an import cache that resolves imports using importers.
//
// This is the explicit-list entry point: callers that already assembled the
// full chain (user importers, load paths, SASS_PATH entries, package config)
// pass it directly. Use NewImportCacheWithOptions to build that chain from
// compile options instead.
func NewImportCache(importers []Importer, parseSelectors bool) *ImportCache {
	return &ImportCache{
		importers:                    importers,
		parseSelectors:               parseSelectors,
		canonicalizeCache:            make(map[canonicalizeKey]*CanonicalizeResult),
		perImporterCanonicalizeCache: make(map[canonicalizeKey]*CanonicalizeResult),
		nonCanonicalRelativeUrls:     make(map[canonicalizeKey]*url.URL),
		importCache:                  make(map[string]*value.Stylesheet),
		resultsCache:                 make(map[string]*ImporterResult),
		loadTimes:                    make(map[string]time.Time),
	}
}

// toImporters converts the user's importers, loadPaths, and packageConfig
// options into a single list of importers.
//
// Order matches Dart: user importers first, then each load path as a
// filesystem importer, then each SASS_PATH entry (semicolon-separated on
// Windows, colon-separated elsewhere), then package: resolution. A nil
// package config contributes no importer. Dart also short-circuits to
// user-only importers in browser builds; Go has no browser target, so the
// full chain always applies.
//
// Matches Dart: ImportCache._toImporters
func toImporters(importers []Importer, loadPaths []string, sassPath string, io sassio.IO, packageConfig PackageConfig) []Importer {
	var result []Importer
	result = append(result, importers...)
	for _, path := range loadPaths {
		result = append(result, NewFilesystemImporter(path, io))
	}
	if sassPath != "" {
		sep := ";"
		if runtime.GOOS != "windows" {
			sep = ":"
		}
		for path := range strings.SplitSeq(sassPath, sep) {
			result = append(result, NewFilesystemImporter(path, io))
		}
	}
	if packageConfig != nil {
		result = append(result, NewPackageImporter(packageConfig, io))
	}
	return result
}

// NewImportCacheWithOptions creates an import cache with load paths and
// importers, matching the Dart ImportCache({importers, loadPaths, ...})
// constructor.
//
// Imports are resolved by trying, in order:
//   - Each importer in importers.
//   - Each load path in loadPaths.
//   - Each load path specified in the SASS_PATH environment variable.
//   - package: URL resolution using PackageImporter.
//
// Matches Dart: ImportCache
func NewImportCacheWithOptions(importers []Importer, loadPaths []string, sassPath string, parseSelectors bool, io sassio.IO, packageConfig PackageConfig) *ImportCache {
	return NewImportCache(toImporters(importers, loadPaths, sassPath, io, packageConfig), parseSelectors)
}

// NewImportCacheNone creates an import cache without any importers.
//
// Only relative loads against the base importer can still resolve; every
// global-chain lookup misses.
//
// Matches Dart: ImportCache.none
func NewImportCacheNone(parseSelectors bool) *ImportCache {
	return NewImportCache(nil, parseSelectors)
}

// SetWarnDeprecationFn sets the function used to emit deprecation warnings
// during canonicalization. This is called by the evaluator to wire up the
// visitor's warn function, so the implicit working-directory fallback can warn
// through the same channel as the rest of the compile.
func (c *ImportCache) SetWarnDeprecationFn(fn func(message string, deprecation *deprecation.Deprecation) error) {
	c.warnFn = fn
}

// NewImportCacheOnly creates an import cache with only the given importers.
//
// Unlike NewImportCacheWithOptions, load paths, SASS_PATH, and the package
// configuration contribute nothing: the chain is exactly the list passed in.
//
// Matches Dart: ImportCache.only
func NewImportCacheOnly(importers []Importer, parseSelectors bool) *ImportCache {
	imps := make([]Importer, len(importers))
	copy(imps, importers)
	return NewImportCache(imps, parseSelectors)
}

// Canonicalize canonicalizes url according to one of this cache's importers.
//
// The baseURL should be the canonical URL of the stylesheet that contains the
// load, if it exists.
//
// Returns the importer that was used to canonicalize url, the canonical URL,
// and the URL that was passed to the importer (which may be resolved relative
// to baseURL if it's passed).
//
// If baseImporter is non-nil, this first tries to use baseImporter to
// canonicalize url (resolved relative to baseURL if passed).
//
// If any importers understand url, returns the CanonicalizeResult with that
// importer, the canonicalized URL, and the original URL. Otherwise, returns nil.
//
// Two cache tiers back this: the global cache covers loads through the whole
// importer chain, while the per-importer cache covers base-importer relative
// loads and per-importer results once some chain element proves uncacheable
// by reading the containing URL.
//
// Matches Dart: ImportCache.canonicalize
func (c *ImportCache) Canonicalize(u *url.URL, baseImporter Importer, baseURL *url.URL, forImport bool) (*CanonicalizeResult, error) {
	if baseImporter != nil && u.Scheme == "" {
		resolvedURL := u
		if baseURL != nil {
			resolvedURL = sassurl.Resolve(baseURL, u)
		}
		key := canonicalizeKey{url: resolvedURL.String(), importer: baseImporter, forImport: forImport}
		if result, ok := c.perImporterCanonicalizeCache[key]; ok {
			// Dart: putIfAbsent returns existing value without recomputation.
			// Non-canonical relative URL tracking is set only on first compute.
			// A cached miss (nil) falls through to the global importer loop
			// rather than short-circuiting, since a later importer in the
			// chain may still recognize the URL.
			if result != nil {
				return result, nil
			}
			// Base importer couldn't canonicalize; fall through to the
			// global importer loop.
		} else {
			result, cacheable, err := c.canonicalizeWith(baseImporter, resolvedURL, baseURL, forImport)
			if err != nil {
				return nil, err
			}
			// Relative loads never observe the containing URL, so Dart
			// asserts they are always cacheable; only a cacheable first
			// computation is stored here.
			if cacheable {
				if baseURL != nil {
					c.nonCanonicalRelativeUrls[key] = u
				}
				c.perImporterCanonicalizeCache[key] = result
				if result != nil {
					return result, nil
				}
			}
		}
	}

	key := canonicalizeKey{url: u.String(), forImport: forImport}
	if result, ok := c.canonicalizeCache[key]; ok {
		return result, nil
	}

	// Each individual call to a canonicalize() override may not be cacheable
	// (specifically, if it has access to containingUrl it's too
	// context-sensitive to usefully cache). We want to cache a given URL across
	// the entire importer chain, so we use cacheable to track whether all
	// canonicalize() calls we've attempted are cacheable. Only if they are, do
	// we store the result in the cache.
	cacheable := true
	for i, imp := range c.importers {
		perKey := canonicalizeKey{url: u.String(), importer: imp, forImport: forImport}
		if result, ok := c.perImporterCanonicalizeCache[perKey]; ok {
			if result != nil {
				return result, nil
			}
			continue
		}

		cr, crCacheable, err := c.canonicalizeWith(imp, u, baseURL, forImport)
		if err != nil {
			return nil, err
		}

		switch {
		case cr != nil && crCacheable && cacheable:
			c.canonicalizeCache[key] = cr
			return cr, nil

		case crCacheable && !cacheable:
			c.perImporterCanonicalizeCache[perKey] = cr
			if cr != nil {
				return cr, nil
			}

		case !crCacheable:
			// If this is the first uncacheable result, add all previous results
			// to the per-importer cache so we don't have to re-run them for
			// future uses of this importer. Uncacheable hits themselves stay
			// out of the per-importer cache because they may vary by load
			// site; only the earlier misses are pinned.
			if cacheable {
				for j := range i {
					c.perImporterCanonicalizeCache[canonicalizeKey{url: u.String(), importer: c.importers[j], forImport: forImport}] = nil
				}
				cacheable = false
			}
			// Dart does not cache uncacheable results in perImporterCanonicalizeCache.
			if cr != nil {
				return cr, nil
			}

		default:
			// result is nil and cacheable is true and global is still cacheable
			// just loop to the next importer
		}
	}

	// Only a chain where every importer stayed cacheable lands in the global
	// cache; a single context-sensitive importer keeps the miss local so later
	// loads re-run the chain.
	if cacheable {
		c.canonicalizeCache[key] = nil
	}
	return nil, nil
}

// canonicalizeWith calls importer.Canonicalize and returns both the result and
// whether that result is cacheable at all.
//
// The containing URL is only offered when the load has a base URL and the URL
// is relative or uses a non-canonical scheme; that restriction keeps canonical
// URLs context-independent. A result that read the containing URL is usable
// once but not shareable. A canonical URL that reuses a scheme the importer
// declared non-canonical is rejected outright.
//
// Matches Dart: ImportCache._canonicalize
func (c *ImportCache) canonicalizeWith(imp Importer, u *url.URL, baseURL *url.URL, forImport bool) (*CanonicalizeResult, bool, error) {
	passContainingURL := baseURL != nil &&
		(u.Scheme == "" || imp.IsNonCanonicalScheme(u.Scheme))

	containingURL := (*url.URL)(nil)
	if passContainingURL {
		containingURL = baseURL
	}
	ctx := NewCanonicalizeContext(containingURL, forImport)

	if fs, ok := imp.(*FilesystemImporter); ok {
		fs.warnFn = c.warnFn
	}

	result, err := imp.Canonicalize(u, ctx)

	cacheable := !passContainingURL || !ctx.WasContainingURLAccessed()

	if err != nil {
		return nil, cacheable, err
	}
	if result == nil {
		return nil, cacheable, nil
	}

	// Relative canonical URLs (empty scheme) should throw an error starting in
	// Dart Sass 2.0.0, but for now, they only emit a deprecation warning in
	// the evaluator. Go therefore accepts them here and leaves the warning to
	// the load site.
	if result.Scheme != "" && imp.IsNonCanonicalScheme(result.Scheme) {
		return nil, cacheable, sasscommon.NewSassScriptException(
			fmt.Sprintf("Importer %v canonicalized %v to %v, which uses a "+
				"scheme declared as non-canonical.",
				imp, u, result),
			nil,
		)
	}

	return &CanonicalizeResult{Importer: imp, CanonicalURL: result, OriginalURL: u}, cacheable, nil
}

// Import tries to import url using one of this cache's importers.
//
// If baseImporter is non-nil, this first tries to use baseImporter to import
// url (resolved relative to baseURL).
//
// If any importers can import url, returns the importer and the parsed
// stylesheet. Otherwise, returns nil.
//
// Caches the result of the import and uses cached results if possible: the
// canonicalization above decides the identity, and ImportCanonical owns the
// parse-tree cache keyed on it.
//
// Matches Dart: ImportCache.import
func (c *ImportCache) Import(u *url.URL, baseImporter Importer, baseURL *url.URL, forImport bool) (Importer, *value.Stylesheet, error) {
	result, err := c.Canonicalize(u, baseImporter, baseURL, forImport)
	if err != nil {
		return nil, nil, err
	}
	if result == nil {
		return nil, nil, nil
	}
	stylesheet, err := c.ImportCanonical(result.Importer, result.CanonicalURL, result.OriginalURL)
	if err != nil {
		return nil, nil, err
	}
	if stylesheet == nil {
		return nil, nil, nil
	}
	return result.Importer, stylesheet, nil
}

// ImportCanonical tries to load the canonicalized canonicalURL using importer.
//
// If importer can import canonicalURL, returns the imported Stylesheet.
// Otherwise returns nil.
//
// If passed, the originalURL represents the URL that was canonicalized into
// canonicalURL. It's used to resolve a relative canonical URL, which importers
// may return for legacy reasons.
//
// Caches the result of the import and uses cached results if possible. A nil
// load is cached as a nil parse tree so the miss is not retried; the load
// timestamp and raw result are recorded only for successful loads.
//
// Matches Dart: ImportCache.importCanonical
func (c *ImportCache) ImportCanonical(imp Importer, canonicalURL *url.URL, originalURL *url.URL) (*value.Stylesheet, error) {
	key := canonicalURL.String()
	if cached, ok := c.importCache[key]; ok {
		return cached, nil
	}

	loadTime := time.Now()
	result, err := imp.Load(canonicalURL)
	if err != nil {
		return nil, err
	}
	if result == nil {
		c.importCache[key] = nil
		return nil, nil
	}

	c.loadTimes[key] = loadTime
	c.resultsCache[key] = result

	// For backwards-compatibility, relative canonical URLs are resolved
	// relative to originalURL. Canonical URLs are meant to be absolute, but
	// legacy importers may still hand back a relative one.
	stylesheetURL := canonicalURL
	if originalURL != nil {
		stylesheetURL = sassurl.Resolve(originalURL, canonicalURL)
	}

	stylesheet, err := parseStylesheet(result.Contents, stylesheetURL, result.Syntax, c.parseSelectors)
	if err != nil {
		return nil, err
	}
	c.importCache[key] = stylesheet
	return stylesheet, nil
}

// Humanize returns a human-friendly URL for canonicalURL to use in a stack trace.
//
// Returns canonicalURL as-is if it hasn't been loaded by this cache.
//
// When several original URLs canonicalize to the same target, the shortest
// original path wins, but scheme-less originals are skipped: they can be
// ambiguous with file: URLs resolved against the working directory. The
// displayed URL keeps the user's original directory shape yet swaps in the
// canonical basename, so a request for "example" that resolved to
// "_example.scss" still shows the partial name.
//
// Matches Dart: ImportCache.humanize
func (c *ImportCache) Humanize(canonicalURL *url.URL) string {
	canonicalStr := canonicalURL.String()

	// If multiple original URLs canonicalize to the same thing, choose the
	// shortest one. Length is compared on the URL path (not the full string),
	// while the full original URL is what gets displayed.
	var shortestOriginal string
	var shortestPathLen int
	for _, result := range c.canonicalizeCache {
		// Ignore original URLs that don't have schemes, because these can be
		// ambiguous with `file:` URLs resolved relative to the current working
		// directory. See sass/dart-sass#2777.
		if result != nil && result.CanonicalURL.String() == canonicalStr && result.OriginalURL.Scheme != "" {
			// Dart compares by url.path.length, then resolves using the full URL.
			if shortestOriginal == "" || len(result.OriginalURL.Path) < shortestPathLen {
				shortestOriginal = result.OriginalURL.String()
				shortestPathLen = len(result.OriginalURL.Path)
			}
		}
	}

	if shortestOriginal != "" {
		// Use the canonicalized basename so that we display e.g.
		// package:example/_example.scss rather than package:example/example
		// in stack traces. Only the final segment is swapped; the original
		// directory and scheme are preserved.
		base := filepath.Base(canonicalURL.Path)
		resolved, err := sassurl.Parse(shortestOriginal)
		if err == nil {
			humanized := resolved.ResolveReference(&url.URL{Path: base})
			return humanized.String()
		}
	}

	// If we don't have an original URL cached, display the canonical URL as-is.
	return canonicalStr
}

// SourceMapURL returns the URL to use in the source map to refer to canonicalURL.
//
// Returns canonicalURL as-is if it hasn't been loaded by this cache. When the
// importer supplied its own source-map URL at load time, that URL wins; the
// data: URL fallback lives on ImporterResult, not here.
//
// Matches Dart: ImportCache.sourceMapUrl
func (c *ImportCache) SourceMapURL(canonicalURL *url.URL) string {
	key := canonicalURL.String()
	if result, ok := c.resultsCache[key]; ok && result.sourceMapURL != nil {
		return result.SourceMapURL().String()
	}
	return canonicalURL.String()
}

// LoadTime returns the most recent time the stylesheet at canonicalURL was
// loaded from its importer, or nil if it has never been loaded.
//
// Internal in Dart (@internal): watch-mode bookkeeping, not public API. The
// timestamp is taken before the load runs, so a slow load still reports when
// it started.
//
// Matches Dart: ImportCache.loadTime
func (c *ImportCache) LoadTime(canonicalURL *url.URL) *time.Time {
	t, ok := c.loadTimes[canonicalURL.String()]
	if !ok {
		return nil
	}
	return &t
}

// ClearCanonicalize clears all cached canonicalizations that could potentially
// produce canonicalURL.
//
// Undocumented in Dart (@nodoc) and internal there (@internal): watch-mode
// invalidation, not public API. Both cache tiers are swept, consulting each
// importer's cheap CouldCanonicalize gate rather than re-running resolution.
//
// Matches Dart: ImportCache.clearCanonicalize
func (c *ImportCache) ClearCanonicalize(canonicalURL *url.URL) {
	for k := range c.canonicalizeCache {
		for _, imp := range c.importers {
			if u := parseURLOrNil(k.url); u != nil && imp.CouldCanonicalize(u, canonicalURL) {
				delete(c.canonicalizeCache, k)
				break
			}
		}
	}

	for k := range c.perImporterCanonicalizeCache {
		if k.importer != nil {
			if u := parseURLOrNil(k.url); u != nil && k.importer.CouldCanonicalize(u, canonicalURL) {
				delete(c.perImporterCanonicalizeCache, k)
			}
		}
	}
}

// ClearImport clears the cached parse tree for the stylesheet with the given
// canonicalURL. Has no effect if not cached.
//
// Undocumented in Dart (@nodoc) and internal there (@internal): watch-mode
// invalidation, not public API. Both the raw load result and the parsed
// stylesheet are dropped so the next import reloads from the importer.
//
// Matches Dart: ImportCache.clearImport
func (c *ImportCache) ClearImport(canonicalURL *url.URL) {
	key := canonicalURL.String()
	delete(c.resultsCache, key)
	delete(c.importCache, key)
}

// parseStylesheet parses the given contents as a Sass stylesheet.
//
// The parser is chosen by syntax (SCSS, indented Sass, or plain CSS with the
// disallowed-function gate), and the parsed tree carries u as its source URL
// plus the parseSelectors flag for pre-parsed style-rule selectors.
func parseStylesheet(contents string, u *url.URL, syntax Syntax, parseSelectors bool) (*value.Stylesheet, error) {
	bytes := []byte(contents)
	switch syntax {
	case SyntaxSCSS:
		parser := value.NewScssParser(bytes, u, parseSelectors)
		return parser.Parse()
	case SyntaxCSS:
		parser := value.NewCssParser(bytes, u, parseSelectors, functions.DisallowedFunctionNames())
		return parser.Parse()
	case SyntaxSass:
		parser := value.NewSassParser(bytes, u, parseSelectors)
		return parser.Parse()
	default:
		panic(fmt.Sprintf("unknown syntax: %v", syntax))
	}
}

// parseURLOrNil parses a URL string, returning nil on error.
//
// Used when sweeping string-keyed canonicalization caches during
// invalidation: an unparsable key simply cannot match the gate check.
func parseURLOrNil(rawURL string) *url.URL {
	u, err := sassurl.Parse(rawURL)
	if err != nil {
		return nil
	}
	return u
}

// SortedURLKeys returns the keys of a map[*url.URL]T sorted by string
// representation. Useful for deterministic iteration.
func SortedURLKeys[T any](m map[*url.URL]T) []*url.URL {
	keys := make([]*url.URL, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].String() < keys[j].String()
	})
	return keys
}
