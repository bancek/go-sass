// Copyright 2024 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package eval

// dart-source: lib/src/importer/node_package.dart

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassio"
)

// validScssExtensions is the set of file extensions that Sass can parse.
// NodePackageImporter only resolves files with these extensions.
var validScssExtensions = map[string]bool{
	".scss": true,
	".sass": true,
	".css":  true,
}

// NodePackageImporter resolves pkg: URLs using the Node resolution algorithm.
//
// Pkg: is a non-canonical scheme: canonicalization may observe the containing
// stylesheet, so resolution starts from the containing file's directory and
// falls back to the entry-point directory. Only .scss/.sass/.css targets are
// honored; anything else fails rather than resolving to an unparseable file.
//
// Matches Dart: NodePackageImporter in lib/src/importer/node_package.dart
type NodePackageImporter struct {
	entryPointDirectory string
	io                  sassio.IO
	cwd                 *FilesystemImporter
}

// NewNodePackageImporter creates a Node package importer with the given entry point.
//
// The entry directory is absolutized so later working-directory changes cannot
// shift fallback resolution. File loads delegate to a working-directory
// filesystem importer held on the struct. Dart throws when the importer is
// built without a filesystem; Go records the same guard on IsJS for the
// future Node.js binding surface.
//
// Matches Dart: NodePackageImporter(entryPointDirectory)
func NewNodePackageImporter(entryPointDirectory string, io sassio.IO) *NodePackageImporter {
	// Matches Dart: if (isBrowser) { throw "The Node package importer cannot be used without a filesystem."; }
	if IsJS {
		// This will be enabled when Node.js bindings are implemented
	}
	abs, err := filepath.Abs(entryPointDirectory)
	if err != nil {
		abs = entryPointDirectory
	}
	return &NodePackageImporter{
		entryPointDirectory: abs,
		io:                  io,
		cwd:                 NewFilesystemImporterCwd(io),
	}
}

// IsNonCanonicalScheme reports whether scheme needs load-site-dependent
// resolution. Only pkg: qualifies: the importer never returns it from
// Canonicalize, but absolute pkg: URLs may observe the containing URL while
// resolving.
func (i *NodePackageImporter) IsNonCanonicalScheme(scheme string) bool {
	return scheme == "pkg"
}

// Canonicalize resolves u to an absolute file: URL, or returns nil when
// another importer should handle it.
//
// File: URLs delegate to the working-directory filesystem importer; any other
// non-pkg: scheme is declined. A pkg: URL must carry no host, port,
// credentials, leading slash, empty path, query, or fragment — violations
// fail. The bare specifier splits into package name plus subpath; names that
// cannot be valid Node package names (relative, backslashed, encoded, or a
// bare scope) return nil so a later importer may try them. Lookup then walks
// up to the nearest node_modules root and tries, in order, the exports map,
// the sass/style manifest keys plus index fallback (root loads only), and the
// subpath relative to the package root. Exports hits outside the parseable
// extensions fail; a missing package.json surfaces as a read error, while a
// present-but-malformed one fails as a parse error naming the package.
//
// Matches Dart: NodePackageImporter.canonicalize
func (i *NodePackageImporter) Canonicalize(u *url.URL, ctx *CanonicalizeContext) (*url.URL, error) {
	if u.Scheme == "file" {
		return i.cwd.Canonicalize(u, ctx)
	}
	if u.Scheme != "pkg" {
		return nil, nil
	}

	if u.Host != "" || u.User != nil {
		return nil, sasscommon.NewSassScriptException("A pkg: URL must not have a host, port, username or password.", nil)
	}
	if strings.HasPrefix(u.Path, "/") {
		return nil, sasscommon.NewSassScriptException("A pkg: URL's path must not begin with /.", nil)
	}
	if u.Path == "" {
		return nil, sasscommon.NewSassScriptException("A pkg: URL must not have an empty path.", nil)
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return nil, sasscommon.NewSassScriptException("A pkg: URL must not have a query or fragment.", nil)
	}

	baseDirectory := i.entryPointDirectory
	containingURL := ctx.ContainingURL()
	if containingURL != nil && containingURL.Scheme == "file" {
		// Reading the containing URL marks this canonicalization as
		// load-site specific, so the import cache will not share the result
		// across sites. Resolution prefers the containing file's directory;
		// the entry-point directory is only the fallback.
		baseDirectory = filepath.Dir(containingURL.Path)
	}

	packageName, subpath := packageNameAndSubpath(u.Path)

	if strings.HasPrefix(packageName, ".") ||
		strings.Contains(packageName, "\\") ||
		strings.Contains(packageName, "%") ||
		(strings.HasPrefix(packageName, "@") && !strings.Contains(filepath.ToSlash(packageName), "/")) {
		// Not a valid Node package name: decline so another importer in the
		// chain may handle the URL rather than failing the load here.
		return nil, nil
	}

	packageRoot := resolvePackageRoot(i.io, packageName, baseDirectory)
	if packageRoot == "" {
		return nil, nil
	}

	jsonPath := filepath.Join(packageRoot, "package.json")
	// A missing manifest is a plain read error (the package simply may not
	// be installed here); only a present-but-unparseable file becomes a
	// "Failed to parse ..." stylesheet error.
	jsonBytes, err := i.io.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", jsonPath, err)
	}

	packageManifest := &orderedmap.OrderedJSON{}
	if err := json.Unmarshal(jsonBytes, packageManifest); err != nil {
		return nil, sasscommon.NewSassScriptException(fmt.Sprintf("Failed to parse %s for \"pkg:%s\": %v", jsonPath, packageName, err), nil)
	}

	path, err := resolvePackageExports(i.io, packageRoot, subpath, packageManifest, packageName)
	if err != nil {
		return nil, err
	}
	if path != "" {
		if validScssExtensions[filepath.Ext(path)] {
			// Matches Dart: p.toUri(p.canonicalize(_resolveImportOnly(resolved))) —
			// uses path normalization, NOT io.canonicalize (symlink resolution).
			// The exports branch normalizes lexically only (no realpath), and
			// prefers the @import import-only variant when the load runs
			// under an @import rule.
			resolved := resolveImportOnly(i.io, path, ctx.FromImport)
			return &url.URL{Scheme: "file", Path: filepath.ToSlash(filepath.Clean(resolved))}, nil
		}
		// An exports hit that is not a stylesheet is a hard error, not a
		// decline: the mapping unambiguously owns this URL.
		return nil, sasscommon.NewSassScriptException(
			fmt.Sprintf("The export for '%s' in '%s' resolved to '%s', which is not a '.scss', '.sass', or '.css' file.",
				orRoot(subpath), packageName, path), nil)
	}

	if subpath == "" {
		// Root loads fall back past exports to the sass/style manifest keys
		// and then an index file at the package root; a miss here declines
		// so the remaining importers may try.
		rootPath := resolvePackageRootValues(i.io, packageRoot, packageManifest, ctx.FromImport)
		if rootPath != "" {
			// Matches Dart: p.toUri(p.canonicalize(rootPath))
			return &url.URL{Scheme: "file", Path: filepath.ToSlash(filepath.Clean(rootPath))}, nil
		}
		return nil, nil
	}

	// Subpath loads fall back to the subpath relative to the package root,
	// resolved with the usual partial/extension/index rules.
	subpathInRoot := filepath.Join(packageRoot, subpath)
	return i.cwd.Canonicalize(&url.URL{Scheme: "file", Path: subpathInRoot}, ctx)
}

// Load reads a previously canonicalized file: URL through the
// working-directory filesystem importer.
//
// Matches Dart: NodePackageImporter.load
func (i *NodePackageImporter) Load(u *url.URL) (*ImporterResult, error) {
	return i.cwd.Load(u)
}

// CouldCanonicalize always returns true: any URL could in principle resolve
// through node_modules, so invalidation must conservatively assume a match.
//
// Matches Dart: inherited Importer.couldCanonicalize — unconditionally returns true
func (i *NodePackageImporter) CouldCanonicalize(u *url.URL, canonicalURL *url.URL) bool {
	return true
}

// ModificationTime always reports the current time, marking package URLs as
// perpetually stale. Dart's base importer behaves the same way rather than
// stating node_modules contents.
//
// Matches Dart: inherited Importer.modificationTime — returns DateTime.now()
func (i *NodePackageImporter) ModificationTime(canonicalURL *url.URL) (time.Time, error) {
	return time.Now(), nil
}

// packageNameAndSubpath splits a bare import specifier into its package name
// and subpath, if one exists.
//
// Because this is a bare import specifier and not a path, we always use "/"
// to avoid invalid values on non-Posix machines. A scoped name (@org/pkg)
// consumes two segments; the remainder, if any, becomes the subpath.
//
// Matches Dart: NodePackageImporter._packageNameAndSubpath
func packageNameAndSubpath(specifier string) (string, string) {
	parts := strings.Split(specifier, "/")
	name := filepath.FromSlash(parts[0])
	parts = parts[1:]

	if strings.HasPrefix(name, "@") && len(parts) > 0 {
		name = filepath.Join(name, filepath.FromSlash(parts[0]))
		parts = parts[1:]
	}

	var subpath string
	if len(parts) > 0 {
		subpath = filepath.FromSlash(strings.Join(parts, "/"))
	}
	return name, subpath
}

// resolvePackageRoot returns an absolute path to the root directory for the
// most proximate installed packageName.
//
// Walk up directories checking for node_modules/<packageName>. The walk ends
// at the filesystem root with "" when nothing is installed above the base.
//
// Matches Dart: NodePackageImporter._resolvePackageRoot
func resolvePackageRoot(io sassio.IO, packageName string, baseDirectory string) string {
	for {
		potentialPackage := filepath.Join(baseDirectory, "node_modules", packageName)
		if dirExists(io, potentialPackage) {
			return potentialPackage
		}
		parent := filepath.Dir(baseDirectory)
		if parent == baseDirectory {
			return ""
		}
		baseDirectory = parent
	}
}

// resolvePackageRootValues returns a file path specified by the sass or style
// values in a package manifest, or an index file relative to the package root.
//
// The sass key wins over style; only values with a stylesheet extension are
// honored, so a JavaScript "style" entry does not shadow the index fallback.
// Only root loads (no subpath) reach this fallback.
//
// Matches Dart: NodePackageImporter._resolvePackageRootValues
func resolvePackageRootValues(io sassio.IO, packageRoot string, packageManifest *orderedmap.OrderedJSON, fromImport bool) string {
	if sassVal, ok := packageManifest.Get("sass"); ok {
		if s, ok := sassVal.(string); ok && validScssExtensions[filepath.Ext(s)] {
			return resolveImportOnly(io, filepath.Join(packageRoot, s), fromImport)
		}
	}
	if styleVal, ok := packageManifest.Get("style"); ok {
		if s, ok := styleVal.(string); ok && validScssExtensions[filepath.Ext(s)] {
			return resolveImportOnly(io, filepath.Join(packageRoot, s), fromImport)
		}
	}
	result, _ := resolveImportPath(io, filepath.Join(packageRoot, "index"), fromImport)
	return result
}

// resolveImportOnly returns path or, if necessary, the import-only variant that
// should be loaded instead: for @import, a sibling <name>.import<ext> file takes
// precedence when it exists.
//
// Matches Dart: NodePackageImporter._resolveImportOnly
func resolveImportOnly(io sassio.IO, path string, fromImport bool) string {
	if !fromImport {
		return path
	}
	ext := filepath.Ext(path)
	importOnly := withoutExtension(path) + ".import" + ext
	if fileExists(io, importOnly) {
		return importOnly
	}
	return path
}

// resolvePackageExports returns a file path specified by a subpath in the
// exports section of package.json.
//
// Without an exports field there is nothing to resolve here. Extensionless
// subpaths get a second pass with an appended index, since "pkg:name/dir"
// conventionally means its index file.
//
// Matches Dart: NodePackageImporter._resolvePackageExports
func resolvePackageExports(io sassio.IO, packageRoot string, subpath string, packageManifest *orderedmap.OrderedJSON, packageName string) (string, error) {
	exportsRaw, ok := packageManifest.Get("exports")
	if !ok || exportsRaw == nil {
		return "", nil
	}

	subpathVariants := exportsToCheck(subpath, false)
	path, err := nodePackageExportsResolve(io, packageRoot, subpathVariants, exportsRaw, subpath, packageName)
	if err != nil {
		return "", err
	}
	if path != "" {
		return path, nil
	}

	// Dart: tries index when subpath is null OR has no extension. An explicit
	// extension that misses exports is final; an extensionless subpath may
	// still mean its directory index.
	if subpath == "" || filepath.Ext(subpath) == "" {
		subpathIndexVariants := exportsToCheck(subpath, true)
		path, err := nodePackageExportsResolve(io, packageRoot, subpathIndexVariants, exportsRaw, subpath, packageName)
		if err != nil {
			return "", err
		}
		if path != "" {
			return path, nil
		}
	}

	return "", nil
}

// exportsToCheck returns a list of all possible variations of subpath with
// extensions and partials.
//
// If there is no subpath, returns a single "" value, which is used in
// nodePackageExportsResolve to denote the main package export. Otherwise each
// candidate appears both plain and partial-prefixed, with .scss/.sass/.css
// tried when no stylesheet extension is given; an already-partial basename
// suppresses the doubled "_" variants.
//
// Matches Dart: NodePackageImporter._exportsToCheck
func exportsToCheck(subpath string, addIndex bool) []string {
	if subpath == "" && addIndex {
		subpath = "index"
	} else if subpath != "" && addIndex {
		subpath = filepath.Join(subpath, "index")
	}
	if subpath == "" {
		return []string{""}
	}

	var paths []string
	if validScssExtensions[filepath.Ext(subpath)] {
		paths = append(paths, subpath)
	} else {
		paths = append(paths, subpath, subpath+".scss", subpath+".sass", subpath+".css")
	}

	basename := filepath.Base(subpath)
	if strings.HasPrefix(basename, "_") {
		return paths
	}

	dirname := filepath.Dir(subpath)
	for _, p := range paths {
		if dirname == "." {
			paths = append(paths, "_"+filepath.Base(p))
		} else {
			paths = append(paths, filepath.Join(dirname, "_"+filepath.Base(p)))
		}
	}
	return paths
}

// nodePackageExportsResolve returns the path to one subpath variant, resolved
// in the exports of a package manifest.
//
// Returns an error if multiple subpathVariants match, and returns ("", nil) if
// none match. Matches are deduplicated first, so two variants resolving to
// the same file count as one; genuinely distinct hits fail the load rather
// than guessing.
//
// Implementation of PACKAGE_EXPORTS_RESOLVE from the Node.js resolution
// algorithm specification.
//
// Matches Dart: NodePackageImporter._nodePackageExportsResolve
func nodePackageExportsResolve(io sassio.IO, packageRoot string, subpathVariants []string, exportsRaw any, subpath string, packageName string) (string, error) {
	if exportsMap, ok := exportsRaw.(*orderedmap.OrderedJSON); ok {
		// An exports map mixes subpath keys ("./...") with condition keys
		// ("sass", "default", ...) at one level: Dart rejects it outright
		// rather than guessing which reading applies.
		// Matches Dart: exports.keys.any((key) => key.startsWith('.')) &&
		//                exports.keys.any((key) => !key.startsWith('.'))
		hasDotKeys := false
		hasNonDotKeys := false
		for _, key := range exportsMap.Keys() {
			if strings.HasPrefix(key, ".") {
				hasDotKeys = true
			} else {
				hasNonDotKeys = true
			}
			if hasDotKeys && hasNonDotKeys {
				break
			}
		}
		if hasDotKeys && hasNonDotKeys {
			// Matches Dart: throw "`exports` in $packageName can not have both conditions and paths at the same level.\nFound ...keys...in ...package.json"
			var keyList []string
			for _, key := range exportsMap.Keys() {
				keyList = append(keyList, fmt.Sprintf(`"%s"`, key))
			}
			return "", sasscommon.NewSassScriptException(
				fmt.Sprintf("`exports` in %s can not have both conditions and paths at the same level.\n"+
					"Found %s in %s.",
					packageName, strings.Join(keyList, ","), filepath.Join(packageRoot, "package.json")), nil)
		}
	}

	var matches []string
	for _, variant := range subpathVariants {
		var variantResult string
		if variant == "" {
			mainExport := getMainExport(exportsRaw)
			if mainExport != nil {
				result, err := packageTargetResolve(io, "", mainExport, packageRoot, "")
				if err != nil {
					return "", err
				}
				variantResult = result
			}
		} else if exportsMap, ok := exportsRaw.(*orderedmap.OrderedJSON); ok {
			// A conditions-only map (no "./" keys) cannot satisfy a subpath
			// lookup, so this variant misses without consulting the map.
			hasDotKeys := false
			for _, key := range exportsMap.Keys() {
				if strings.HasPrefix(key, ".") {
					hasDotKeys = true
					break
				}
			}
			if !hasDotKeys {
				continue
			}

			matchKey := "./" + filepath.ToSlash(variant)
			// Exact subpath keys win over "*" patterns; a "*" in the looked-up
			// key itself never counts as an exact hit.
			if target, ok := exportsMap.Get(matchKey); ok && target != nil && !strings.Contains(matchKey, "*") {
				result, err := packageTargetResolve(io, variant, target, packageRoot, "")
				if err != nil {
					return "", err
				}
				variantResult = result
			} else {
				// No exact key: collect single-"*" pattern keys, most
				// specific first, and take the first whose base/trailer
				// frame the lookup.
				var expansionKeys []string
				for _, key := range exportsMap.Keys() {
					if strings.Count(key, "*") == 1 {
						expansionKeys = append(expansionKeys, key)
					}
				}
				sort.Slice(expansionKeys, func(i, j int) bool {
					return compareExpansionKeys(expansionKeys[i], expansionKeys[j]) < 0
				})

				for _, expansionKey := range expansionKeys {
					parts := strings.SplitN(expansionKey, "*", 2)
					patternBase := parts[0]
					patternTrailer := parts[1]

					if !strings.HasPrefix(matchKey, patternBase) {
						continue
					}
					// A bare base with no wildcard span matches nothing: the
					// "*" must capture at least the separator-adjacent text.
					if matchKey == patternBase {
						continue
					}
					if patternTrailer != "" && (!strings.HasSuffix(matchKey, patternTrailer) || len(matchKey) < len(expansionKey)) {
						continue
					}

					target, ok := exportsMap.Get(expansionKey)
					if !ok || target == nil {
						continue
					}

					// The "*" span is sliced out of the lookup key and
					// substituted into the target by packageTargetResolve.
					patternMatch := matchKey[len(patternBase) : len(matchKey)-len(patternTrailer)]
					result, err := packageTargetResolve(io, variant, target, packageRoot, patternMatch)
					if err != nil {
						return "", err
					}
					variantResult = result
					break
				}
			}
		}
		if variantResult != "" {
			matches = append(matches, variantResult)
		}
	}

	// Matches Dart: .nonNulls.toSet().toList(). Unmatched variants contribute
	// nothing, so only successful resolutions vote; identical hits collapse
	// to one before the ambiguity check.
	var deduped []string
	seen := make(map[string]struct{})
	for _, m := range matches {
		if _, ok := seen[m]; !ok {
			seen[m] = struct{}{}
			deduped = append(deduped, m)
		}
	}
	matches = deduped

	// Matches Dart: return switch (matches) { [var path] => path, [] => null, var paths => throw ... }.
	// Zero hits decline to the next resolution stage; several distinct hits
	// fail the load with the full candidate list instead of guessing.
	switch len(matches) {
	case 0:
		return "", nil
	case 1:
		return matches[0], nil
	default:
		return "", sasscommon.NewSassScriptException(
			fmt.Sprintf("Unable to determine which of multiple potential resolutions "+
				"found for %s in %s should be used. \n\nFound:\n%s",
				orRoot(subpath), packageName, strings.Join(matches, "\n")), nil)
	}
}

// compareExpansionKeys implements the PATTERN_KEY_COMPARE comparator from
// the Node.js resolution algorithm specification: longer pattern bases sort
// first, then exact keys before patterns, then longer keys before shorter
// ones.
//
// Matches Dart: NodePackageImporter._compareExpansionKeys
func compareExpansionKeys(keyA, keyB string) int {
	baseLengthA := len(keyA)
	if idx := strings.Index(keyA, "*"); idx >= 0 {
		baseLengthA = idx + 1
	}
	baseLengthB := len(keyB)
	if idx := strings.Index(keyB, "*"); idx >= 0 {
		baseLengthB = idx + 1
	}
	if baseLengthA > baseLengthB {
		return -1
	}
	if baseLengthB > baseLengthA {
		return 1
	}
	if !strings.Contains(keyA, "*") {
		return 1
	}
	if !strings.Contains(keyB, "*") {
		return -1
	}
	if len(keyA) > len(keyB) {
		return -1
	}
	if len(keyB) > len(keyA) {
		return 1
	}
	return 0
}

// packageTargetResolve returns a file path for subpath, as resolved in the
// exports object. Verifies the file exists relative to packageRoot.
//
// Instances of * will be replaced with patternMatch. String targets must be
// package-root-relative ("./..."); condition maps honor only sass, style, and
// default in manifest order with nil entries skipped; arrays resolve
// first-hit-wins. Anything else fails as an invalid exports value.
//
// Implementation of PACKAGE_TARGET_RESOLVE from the Node.js resolution
// algorithm specification.
//
// Matches Dart: NodePackageImporter._packageTargetResolve
func packageTargetResolve(io sassio.IO, subpath string, exports any, packageRoot string, patternMatch string) (string, error) {
	switch val := exports.(type) {
	case string:
		if !strings.HasPrefix(val, "./") {
			// Matches Dart: throw "Export '$string' must be a path relative to the package root at '$packageRoot'."
			return "", sasscommon.NewSassScriptException(fmt.Sprintf("Export '%s' must be a path relative to the package root at '%s'.", val, packageRoot), nil)
		}
		if patternMatch != "" {
			// A pattern hit must name a file that actually exists; a miss
			// declines this variant rather than falling back to the
			// unexpanded path.
			replaced := strings.Replace(val, "*", patternMatch, 1)
			path := filepath.Join(packageRoot, filepath.FromSlash(replaced))
			path = filepath.Clean(path)
			if fileExists(io, path) {
				return path, nil
			}
			return "", nil
		}
		return filepath.Join(packageRoot, filepath.FromSlash(val)), nil

	case *orderedmap.OrderedJSON:
		// Dart iterates map.pairs in insertion order and filters to
		// sass → style → default — node_package.dart
		for _, key := range val.Keys() {
			if key != "sass" && key != "style" && key != "default" {
				continue
			}
			value, _ := val.Get(key)
			if value == nil {
				continue
			}
			result, err := packageTargetResolve(io, subpath, value, packageRoot, patternMatch)
			if err != nil {
				return "", err
			}
			if result != "" {
				return result, nil
			}
		}
		return "", nil

	case []any:
		if len(val) == 0 {
			return "", nil
		}
		for _, item := range val {
			if item == nil {
				continue
			}
			result, err := packageTargetResolve(io, subpath, item, packageRoot, patternMatch)
			if err != nil {
				return "", err
			}
			if result != "" {
				return result, nil
			}
		}
		return "", nil

	default:
		// Matches Dart: throw "Invalid 'exports' value $exports in ${p.join(packageRoot, 'package.json')}."
		return "", sasscommon.NewSassScriptException(fmt.Sprintf("Invalid 'exports' value %v in %s.", exports, filepath.Join(packageRoot, "package.json")), nil)
	}
}

// getMainExport returns a path to a package's export without a subpath.
//
// Plain strings resolve as-is; condition maps without subpath keys resolve as
// a whole; otherwise only the "." entry applies. Array-style exports never
// match here: Dart's typed List<String> pattern cannot match JSON-decoded
// lists, so both ports silently ignore them.
//
// Matches Dart: NodePackageImporter._getMainExport
func getMainExport(exports any) any {
	switch val := exports.(type) {
	case string:
		return val
	// Dart: List<String> list — node_package.dart
	// Dart's pattern-matching checks the type parameter, so List<String>
	// never matches JSON-decoded List<dynamic>. The result: array-style
	// "exports" fields (e.g. ["."]) fall through to null in both Dart and Go.
	// Per Node.js spec, []any would correctly handle arrays; we match Dart's
	// behavior of silently ignoring them.
	case *orderedmap.OrderedJSON:
		hasDotKeys := false
		for _, key := range val.Keys() {
			if strings.HasPrefix(key, ".") {
				hasDotKeys = true
				break
			}
		}
		if !hasDotKeys {
			return val
		}
		if export, ok := val.Get("."); ok {
			return export
		}
		return nil
	default:
		return nil
	}
}

// orRoot returns "root" if subpath is empty, otherwise subpath.
//
// Used only in error reporting so root-package failures read naturally
// instead of quoting an empty string.
func orRoot(subpath string) string {
	if subpath == "" {
		return "root"
	}
	return subpath
}
