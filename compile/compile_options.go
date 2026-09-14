// Copyright 2018 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package compile

// dart-source: lib/src/compile.dart (CompileOptions maps to compile/compileString parameters)

import (
	"net/url"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/eval"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasslogger"
)

// OutputStyle selects the formatting of the compiled CSS output.
//
// It mirrors Dart's OutputStyle: expanded output spreads each rule across
// lines for readability, while compressed output strips whitespace for
// minimal size.
type OutputStyle int

const (
	// OutputStyleExpanded formats CSS across multiple lines with
	// indentation. It is the zero value, matching Dart's default style.
	OutputStyleExpanded OutputStyle = iota
	// OutputStyleCompressed strips most whitespace, emitting minimal CSS
	// on as few lines as possible.
	OutputStyleCompressed
)

// CompileOptions controls the compilation behavior.
type CompileOptions struct {
	// Importers is the list of custom importers to use when loading stylesheets.
	Importers []eval.Importer

	// Importer is the entrypoint importer used to resolve relative imports
	// from the entrypoint stylesheet. It flows separately from Importers
	// and is not prepended to the importers list.
	// Matches Dart: compileString Importer? importer parameter
	Importer eval.Importer

	// LoadPaths is the list of filesystem paths to search for stylesheets.
	LoadPaths []string

	// SassPath from SASS_PATH env variable.
	SassPath string

	// URL is the URL of the source stylesheet. If nil, the stylesheet is treated
	// as having no URL and relative imports will not resolve.
	URL *url.URL

	// PackageConfig resolves package: URLs to file: URLs. If nil, package:
	// imports are not supported.
	PackageConfig eval.PackageConfig

	// NodePackageImporter is the Node.js package importer used to resolve
	// pkg: URLs using the Node resolution algorithm. If nil, pkg: URLs are
	// not supported. Matches Dart: pkgImporters option (simplified to single importer).
	NodePackageImporter *eval.NodePackageImporter

	// Functions is the list of user-defined callables to make available.
	// Matches Dart: compile() functions parameter.
	Functions []sasscallable.Callable

	// Logger receives warnings and debug output. A nil logger gets the
	// default; CompileString always wraps it for deprecation processing,
	// applying the silence/fatal/future lists with repetition limiting.
	Logger sasslogger.Logger

	// QuietDeps silences warnings coming from dependency stylesheets
	// resolved through the import cache. Matches Dart's quietDeps
	// parameter.
	QuietDeps bool

	// SourceMap enables source-map generation, exposed on the result's
	// SourceMap. Matches Dart's sourceMap parameter.
	SourceMap bool

	// EmitErrorCss renders a Sass failure as CSS (an explanatory comment
	// plus a body::before rule) in a successful result instead of
	// returning the error. Only spanned errors qualify; unspanned script
	// errors still fail the compilation.
	EmitErrorCss bool

	// Style selects expanded or compressed CSS output. The zero value is
	// expanded, matching Dart's default.
	Style OutputStyle

	// EmbedSources controls whether source file contents are embedded in the
	// generated source map. Matches Dart: --embed-sources.
	EmbedSources bool

	// EmbedSourceMap controls whether the generated source map is embedded in
	// the output CSS as a data URL. Matches Dart: --embed-source-map.
	EmbedSourceMap bool

	// SourceMapURLs is "relative" (default) or "absolute", controlling how
	// URLs in the generated source map link to source files.
	// Matches Dart: --source-map-urls.
	SourceMapURLs string

	// Charset controls whether to emit a @charset declaration or BOM for
	// non-ASCII stylesheets. Matches Dart: compile() charset parameter.
	Charset bool

	// SilenceDeprecations contains deprecation types whose warnings should be
	// suppressed. Matches Dart: compile() silenceDeprecations parameter.
	SilenceDeprecations []*deprecation.Deprecation

	// FatalDeprecations contains deprecation types that should cause an error.
	// Matches Dart: compile() fatalDeprecations parameter.
	FatalDeprecations []*deprecation.Deprecation

	// FutureDeprecations contains future deprecation types to enable early.
	// Matches Dart: compile() futureDeprecations parameter.
	FutureDeprecations []*deprecation.Deprecation

	// Verbose controls whether all deprecation warnings are shown (no repetition
	// limiting). Matches Dart: compile() verbose parameter.
	Verbose bool

	// Unicode controls whether to use Unicode glyphs in output (warnings,
	// errors). Defaults to true. Matches Dart: --unicode CLI option.
	Unicode bool

	// AlertColor enables terminal colors in error/log formatted strings.
	// Matches Dart: compileString/compile color parameter
	AlertColor bool

	// AlertAscii encodes formatted messages in ASCII.
	// Matches Dart: compileString/compile ascii parameter (term_glyph.ascii)
	AlertAscii bool

	// Syntax is the syntax to use for parsing. If nil/zero (SyntaxSCSS), defaults to SCSS.
	// Matches Dart: compile/compileString syntax parameter.
	Syntax eval.Syntax
}
