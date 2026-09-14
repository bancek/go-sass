// Copyright 2018 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package compile

// dart-source: lib/src/compile.dart

import (
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/eval"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassio"
	"github.com/bancek/go-sass/sasslogger"
	"github.com/bancek/go-sass/sourcemap"
	"github.com/bancek/go-sass/value"
)

// CompileString compiles a Sass source string and returns the CSS result.
//
// Like Dart's compileString, but with extra options for the node-compatible
// API and the executable. It parses source with the syntax from opts (Sass,
// CSS, or SCSS by default), builds the import cache from opts' importers,
// load paths, and SASS_PATH in that order, wraps the logger for deprecation
// processing, warns when url is relative, and funnels evaluation plus
// serialization through compileStylesheet.
//
// A nil opts gets default options; io must be set so file reads and
// environment lookups never touch the ambient process.
func CompileString(source string, io sassio.IO, opts *CompileOptions) (*CompileResult, error) {
	if opts == nil {
		opts = &CompileOptions{Unicode: true, Charset: true}
	}
	if io == nil {
		return nil, fmt.Errorf("compile string: io must be set")
	}

	var stylesheet *value.Stylesheet
	var err error
	switch opts.Syntax {
	case eval.SyntaxSass:
		parser := value.NewSassParser([]byte(source), opts.URL, false)
		stylesheet, err = parser.Parse()
		if err != nil {
			return nil, err
		}
	case eval.SyntaxCSS:
		parser := value.NewCssParser([]byte(source), opts.URL, false, nil)
		stylesheet, err = parser.Parse()
		if err != nil {
			return nil, err
		}
	default:
		parser := value.NewScssParser([]byte(source), opts.URL, false)
		stylesheet, err = parser.Parse()
		if err != nil {
			return nil, err
		}
	}

	if opts.SassPath == "" {
		opts.SassPath = io.GetEnvironmentVariable("SASS_PATH")
	}

	importCache := eval.NewImportCacheWithOptions(opts.Importers, opts.LoadPaths, opts.SassPath, false, io, opts.PackageConfig)

	logger := opts.Logger
	if logger == nil {
		logger = sasslogger.NewDefaultLogger(opts.Unicode)
	}

	// Always route warnings through a DeprecationProcessingLogger so the
	// silence/fatal/future lists and repetition limiting apply. Validate
	// runs before evaluation, and Summarize is deferred so it reports on
	// success and failure alike.
	// Matches Dart: DeprecationProcessingLogger creation in compile() / compileString()
	deprecationLogger := sasslogger.NewDeprecationProcessingLogger(
		logger,
		opts.SilenceDeprecations,
		opts.FatalDeprecations,
		opts.FutureDeprecations,
		!opts.Verbose,
	)
	deprecationLogger.Validate()
	logger = deprecationLogger
	defer deprecationLogger.Summarize(opts.NodePackageImporter != nil)

	// A relative entry url with no node-package importer can no longer
	// anchor relative imports, so warn for COMPILE_STRING_RELATIVE_URL.
	// Matches Dart: Deprecation.compileStringRelativeUrl
	span, err := stylesheet.Span()
	if err != nil {
		return nil, err
	}
	spanURL, err := span.SourceURL()
	if err != nil {
		return nil, err
	}
	if spanURL != nil && spanURL.Scheme == "" && opts.NodePackageImporter == nil {
		if err := logger.WarnDeprecation(
			fmt.Sprintf(
				"Passing a relative `url` argument (%s) to compileString() or "+
					"related functions is deprecated and will be an error in Dart Sass 2.0.0.\n",
				spanURL,
			),
			nil,
			deprecation.CompileStringRelativeUrl,
			nil,
		); err != nil {
			return nil, err
		}
	}

	if opts.Importer == nil {
		opts.Importer = eval.NewFilesystemImporterNoLoadPath(io)
	}

	return compileStylesheet(
		stylesheet,
		importCache,
		opts.NodePackageImporter,
		opts.Importer,
		opts.Functions,
		logger,
		opts.Style,
		opts.QuietDeps,
		opts.SourceMap,
		opts.Charset,
		opts.EmitErrorCss,
		opts.Unicode,
	)
}

// compileStylesheet evaluates a parsed stylesheet and serializes the
// resulting CSS.
//
// It warns for LEGACY_JS_API when a node-package importer is present, runs
// evaluation (rendering spanned failures as CSS when emitErrorCss asks),
// serializes with an optional source-map builder, rewrites source-map URLs
// against the import cache, and pairs both halves into a CompileResult.
//
// Matches Dart: compile.dart:_compileStylesheet() lines 178-237
func compileStylesheet(
	stylesheet *value.Stylesheet,
	importCache *eval.ImportCache,
	nodeImporter *eval.NodePackageImporter,
	importer eval.Importer,
	functions []sasscallable.Callable,
	logger sasslogger.Logger,
	style OutputStyle,
	quietDeps bool,
	sourceMap bool,
	charset bool,
	emitErrorCss bool,
	unicode bool,
) (*CompileResult, error) {
	if nodeImporter != nil {
		// The node-package importer is the legacy JS API surface, so its
		// mere presence warns once per compilation.
		// Matches Dart: Deprecation.legacyJsApi in _compileStylesheet
		if err := logger.WarnDeprecation(
			"The legacy JS API is deprecated and will be removed in "+
				"Dart Sass 2.0.0.\n\n"+
				"More info: https://sass-lang.com/d/legacy-js-api",
			nil,
			deprecation.LegacyJsApi,
			nil,
		); err != nil {
			return nil, err
		}
	}

	evaluateResult, err := eval.Evaluate(
		stylesheet,
		importCache,
		nodeImporter,
		importer,
		functions,
		logger,
		quietDeps,
		sourceMap,
	)
	if err != nil {
		// With emitErrorCss, only the four spanned error shapes render as
		// CSS (mirroring Dart's `on SassException`); unspanned script
		// errors have no span to render and propagate instead.
		if emitErrorCss {
			switch e := err.(type) {
			case *sasscommon.SassException:
				return NewCompileResult(&eval.EvaluateResult{}, &value.SerializeResult{CSS: errorToCssString(e, unicode)}), nil
			case *sasscommon.SassRuntimeException:
				return NewCompileResult(&eval.EvaluateResult{}, &value.SerializeResult{CSS: errorToCssString(e, unicode)}), nil
			case *sasscommon.MultiSpanSassException:
				return NewCompileResult(&eval.EvaluateResult{}, &value.SerializeResult{CSS: errorToCssString(e, unicode)}), nil
			case *sasscommon.MultiSpanSassRuntimeException:
				return NewCompileResult(&eval.EvaluateResult{}, &value.SerializeResult{CSS: errorToCssString(e, unicode)}), nil
			}
		}
		return nil, err
	}

	var smBldr *sourcemap.Builder
	if sourceMap {
		smBldr = sourcemap.NewBuilder("output.css")
	}

	serResult, err := value.SerializeWithSourceMap(
		evaluateResult.Stylesheet,
		&value.SerializeOptions{Style: value.OutputStyle(style), Charset: charset},
		smBldr,
	)
	if err != nil {
		return nil, err
	}

	// Rewrite source-map URLs in place, mirroring Dart's mapInPlace over
	// resultSourceMap.urls: entries that need an importer mapping go
	// through SourceMapURL, while empty ones stand for string/stdin input.
	// Matches Dart: post-serialization URL rewriting in _compileStylesheet
	if serResult.SourceMap != nil && importCache != nil {
		sspan, err := stylesheet.Span()
		if err != nil {
			return nil, err
		}
		for i, urlStr := range serResult.SourceMap.URLs {
			if urlStr == "" {
				// Empty URL (stdin/string input without explicit URL):
				// convert to data: URI containing the file contents.
				if file, ferr := sspan.File(); ferr == nil && file != nil {
					serResult.SourceMap.URLs[i] = "data:text/plain;charset=utf-8," + url.PathEscape(file.Text())
				}
			} else {
				// Rewrite using the importer's sourceMapUrl if available.
				parsed, perr := url.Parse(urlStr)
				if perr == nil {
					serResult.SourceMap.URLs[i] = importCache.SourceMapURL(parsed)
				}
			}
		}
	}

	return NewCompileResult(evaluateResult, serResult), nil
}

// Compile compiles the Sass file at path and returns the CSS result.
//
// It reads the file through io, anchors it at an absolute file: URL,
// defaults LoadPaths to the file's own directory when unset, infers the
// syntax from the extension unless opts already sets one, and delegates to
// CompileString on a copy of opts so the caller's struct is untouched.
//
// Like Dart's compile, but with the syntax-mismatch cache bypass folded in:
// the source is always parsed directly from the file contents.
//
// A nil opts gets default options; io must be set.
func Compile(path string, io sassio.IO, opts *CompileOptions) (*CompileResult, error) {
	if opts == nil {
		opts = &CompileOptions{Unicode: true, Charset: true}
	}
	if io == nil {
		return nil, fmt.Errorf("compile string: io must be set")
	}

	source, err := io.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Compute the absolute path and create a file URL. The URL path must
	// use forward slashes: relative resolution merges on "/" (Go's path
	// package), so a native Windows path with backslashes would lose its
	// directory and every relative import would fail to resolve.
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	fileURL := &url.URL{Scheme: "file", Path: filepath.ToSlash(absPath)}

	// Default LoadPaths to the directory containing the file being compiled,
	// so relative imports resolve next to the entrypoint.
	opts2 := *opts
	opts2.URL = fileURL
	if opts2.LoadPaths == nil {
		opts2.LoadPaths = []string{filepath.Dir(absPath)}
	}
	// Infer syntax from the file extension when the caller left it unset;
	// the zero value means SCSS.
	// Matches Dart: syntax ?? Syntax.forPath(path)
	if opts2.Syntax == 0 {
		opts2.Syntax = eval.SyntaxForPath(absPath)
	}

	return CompileString(string(source), io, &opts2)
}

// CompileStringToResult compiles a Sass source string from separately
// passed options, for callers that build the import cache inputs piecemeal.
//
// It folds every argument into a CompileOptions and delegates to
// CompileString; at most one of the importer list and the node importer
// applies, matching Dart's constraint.
//
// Matches Dart: sass.dart:compileStringToResult() lines 212-252
func CompileStringToResult(
	source string,
	io sassio.IO,
	syntax eval.Syntax,
	url *url.URL,
	importers []eval.Importer,
	importer eval.Importer,
	functions []sasscallable.Callable,
	logger sasslogger.Logger,
	style OutputStyle,
	quietDeps bool,
	verbose bool,
	sourceMap bool,
	charset bool,
	fatalDeprecations, silenceDeprecations, futureDeprecations []*deprecation.Deprecation,
	alertColor bool,
	alertAscii bool,
) (*CompileResult, error) {
	opts := &CompileOptions{
		Importers:           importers,
		Importer:            importer,
		Functions:           functions,
		Logger:              logger,
		Style:               style,
		QuietDeps:           quietDeps,
		Verbose:             verbose,
		SourceMap:           sourceMap,
		Charset:             charset,
		AlertColor:          alertColor,
		AlertAscii:          alertAscii,
		SilenceDeprecations: silenceDeprecations,
		FatalDeprecations:   fatalDeprecations,
		FutureDeprecations:  futureDeprecations,
		Syntax:              syntax,
		URL:                 url,
	}
	return CompileString(source, io, opts)
}
