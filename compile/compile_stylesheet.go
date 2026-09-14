// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package compile

// dart-source: lib/src/executable/compile_stylesheet.dart

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/bancek/go-sass/eval"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassio"
	"github.com/bancek/go-sass/sourcemap"
	"github.com/bancek/go-sass/termglyph"
	"github.com/bancek/go-sass/value"
)

// CompileStylesheet compiles the stylesheet at source to destination,
// the executable layer over Compile and CompileString.
//
// If source is empty, the stylesheet is read from stdin (via sassIO.ReadStdin).
// If destination is empty, the CSS is written to stdout.
//
// Failures map to exit codes: a Sass error becomes 65 after deleting a
// half-written destination (unless error CSS is enabled), and a filesystem
// failure becomes 66. The source-map comment is appended to the CSS before
// writing, and file output always trails a newline.
//
// Matches Dart: compileStylesheet (synchronous path; watch/update modes omitted)
func CompileStylesheet(sassIO sassio.IO, source, destination string, opts *CompileOptions) error {
	if opts == nil {
		opts = &CompileOptions{Unicode: true, Charset: true}
	}
	if opts.SassPath == "" {
		opts.SassPath = sassIO.GetEnvironmentVariable("SASS_PATH")
	}

	// Try the compile, then sort failures: Sass errors carry an exit code
	// of 65 (invalid data per sysexits), filesystem errors 66 (no input).
	// Matches Dart: try { _compileStylesheetWithoutErrorHandling(...) }
	result, err := compileStylesheetInternal(sassIO, source, destination, opts)
	if err != nil {
		// A Sass failure deletes a file destination so stale CSS never
		// survives, unless error CSS replaces it. The message renders with
		// the caller's color/unicode settings.
		// Matches Dart: on SassException catch (error, stackTrace) { ... }
		if isSassException(err) {
			if destination != "" && !opts.EmitErrorCss {
				tryDelete(sassIO, destination)
			}
			msg := err.Error()
			if e, ok := err.(interface {
				ErrorWithOptions(sasscommon.HighlightOptions) string
			}); ok {
				msg = e.ErrorWithOptions(buildHighlightOpts(opts))
			}
			return &StylesheetError{ExitCode: 65, Message: msg}
		}
		// A filesystem failure names the offending file's base name, since
		// the full path already appears in the underlying error.
		// Matches Dart: on FileSystemException catch (error, stackTrace) { ... }
		if fse, ok := errors.AsType[*sassio.FileSystemException](err); ok {
			path := fse.Path
			message := fse.Message
			if path != "" {
				message = fmt.Sprintf("Error reading %s: %s.", filepath.Base(path), fse.Message)
			}
			return &StylesheetError{ExitCode: 66, Message: message}
		}
		return err
	}

	// Matches Dart: css += _writeSourceMap(...); write to stdout or file
	css := result.CSS()
	smCSS, err := writeSourceMap(sassIO, opts, result.SourceMap(), destination)
	if err != nil {
		return err
	}
	css += smCSS

	if destination == "" {
		if len(css) > 0 {
			sassIO.PrintOutput(css + "\n")
		}
	} else {
		// Matches Dart: ensureDir(p.dirname(destination)) + writeFile(destination, css + "\n")
		dir := filepath.Dir(destination)
		if err := sassIO.EnsureDir(dir); err != nil {
			return err
		}
		if err := sassIO.WriteFile(destination, []byte(css+"\n")); err != nil {
			return err
		}
	}

	return nil
}

// compileStylesheetInternal runs the actual compilation, throwing errors
// instead of mapping them to exit codes like CompileStylesheet does.
//
// Matches Dart: _compileStylesheetWithoutErrorHandling (synchronous path; no graph caching)
func compileStylesheetInternal(sassIO sassio.IO, source, destination string, opts *CompileOptions) (*CompileResult, error) {
	var result *CompileResult
	var err error

	// The library's emitErrorCss is the JS-API contract: on a Sass error it
	// returns a result holding the error CSS with a nil error. The CLI handles
	// --error-css itself (writing the CSS then returning the error/exit code),
	// so it must stay off for the library call.
	libOpts := *opts
	libOpts.EmitErrorCss = false

	if source == "" {
		// Stdin input compiles as a string; relative imports resolve
		// against the working directory, matching Dart's CLI importer.
		// Matches Dart: readStdin() → compileString(...)
		stdin, readErr := sassIO.ReadStdin()
		if readErr != nil {
			return nil, readErr
		}
		// Dart's CLI uses FilesystemImporter.cwd for stdin so relative imports
		// resolve against the current directory.
		if libOpts.Importer == nil {
			libOpts.Importer = eval.NewFilesystemImporterCwd(sassIO)
		}
		result, err = CompileString(string(stdin), sassIO, &libOpts)
	} else {
		// Matches Dart: compile(source, ...)
		result, err = Compile(source, sassIO, &libOpts)
	}
	if err != nil {
		// On a Sass failure with error CSS enabled, write the error
		// stylesheet to the destination (or stdout) and still return the
		// error so the caller exits non-zero — mirroring Dart's rethrow
		// after printing toCssString.
		// Matches Dart: on SassException catch — emitErrorCss handling, then rethrow
		if isSassException(err) && opts.EmitErrorCss {
			css := errorToCssString(err, opts.Unicode)
			if destination == "" {
				if len(css) > 0 {
					sassIO.PrintOutput(css + "\n")
				}
			} else {
				if dir := filepath.Dir(destination); dir != "" {
					_ = sassIO.EnsureDir(dir)
				}
				_ = sassIO.WriteFile(destination, []byte(css+"\n"))
			}
		}
		return nil, err
	}

	return result, nil
}

// writeSourceMap writes the source map given by sm to disk (if necessary)
// according to opts, returning the source map comment to append to the CSS.
//
// destination is the path the companion CSS will be written to; an empty
// destination means stdout, which requires an embedded map. The map's target
// becomes the destination's base name, its URLs are rebased per SourceMapURLs,
// and the comment is separated by a blank line except in compressed output.
//
// Matches Dart: _writeSourceMap (compile_stylesheet.dart)
func writeSourceMap(sassIO sassio.IO, opts *CompileOptions, sm *sourcemap.SingleMapping, destination string) (string, error) {
	if sm == nil {
		return "", nil
	}

	target := ""
	if destination != "" {
		target = pathToURI(filepath.Base(destination))
	}

	for i, u := range sm.URLs {
		sm.URLs[i] = sourceMapURL(opts, u, destination)
	}

	jsonBytes, err := sm.JSONWithTarget(target, opts.EmbedSources)
	if err != nil {
		return "", err
	}
	text := string(jsonBytes)

	var mapURL string
	if opts.EmbedSourceMap {
		mapURL = "data:application/json;charset=utf-8," + dartDataURIEncode(text)
	} else {
		// An empty destination means stdout, which is incompatible with a
		// sidecar file — so stdout output always embeds the map instead.
		// [destination] can't be empty here because --embed-source-map is
		// incompatible with writing to stdout.
		mapPath := destination + ".map"
		if dir := filepath.Dir(mapPath); dir != "" {
			if err := sassIO.EnsureDir(dir); err != nil {
				return "", err
			}
		}
		if err := sassIO.WriteFile(mapPath, []byte(text)); err != nil {
			return "", err
		}
		mapURL = relativeURI(filepath.Dir(destination), mapPath)
	}

	// Escape the comment terminator so the URL can't close the
	// sourceMappingURL comment early.
	escaped := strings.ReplaceAll(mapURL, "*/", "%2A/")
	prefix := "\n\n"
	if opts.Style == OutputStyleCompressed {
		prefix = ""
	}
	return prefix + "/*# sourceMappingURL=" + escaped + " */", nil
}

// sourceMapURL makes rawURL absolute or relative (to the directory containing
// destination) according to the SourceMapURLs option.
//
// Non-file URLs pass through untouched; file: URLs resolve to absolute paths
// and, in "relative" mode with a destination, shrink to a path relative to
// the CSS output.
//
// Matches Dart: ExecutableOptions.sourceMapUrl (options.dart:556-567)
func sourceMapURL(opts *CompileOptions, rawURL, destination string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	// If it isn't a `file:` URL, return it as-is.
	if u.Scheme != "" && u.Scheme != "file" {
		return rawURL
	}

	path := rawURL
	if u.Scheme == "file" {
		path = u.Path
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}

	if opts.SourceMapURLs == "relative" && destination != "" {
		destDir, err := filepath.Abs(filepath.Dir(destination))
		if err != nil {
			destDir = filepath.Dir(destination)
		}
		rel, err := filepath.Rel(destDir, abs)
		if err != nil {
			rel = abs
		}
		return pathToURI(filepath.ToSlash(rel))
	}

	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}).String()
}

// relativeURI returns the from-relative URI for to, used for the source map link.
//
// Both ends resolve to absolute paths first so the relative hop is stable
// regardless of the caller's working directory.
func relativeURI(from, to string) string {
	absFrom, err := filepath.Abs(from)
	if err != nil {
		absFrom = from
	}
	absTo, err := filepath.Abs(to)
	if err != nil {
		absTo = to
	}
	rel, err := filepath.Rel(absFrom, absTo)
	if err != nil {
		rel = absTo
	}
	return pathToURI(filepath.ToSlash(rel))
}

// pathToURI converts a relative or absolute path to its URI string form,
// slash-separated so Windows paths still form valid URIs.
func pathToURI(path string) string {
	return (&url.URL{Path: path}).String()
}

// dartDataURIEncode percent-encodes content for a data: URI exactly like
// Dart's Uri.dataFromString (the allowed set is alphanumerics plus
// ! $ & ' ( ) * + , - . / : ; = ? @ _ ~; everything else is
// percent-encoded byte-by-byte).
func dartDataURIEncode(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if dartDataURISafe(c) {
			b.WriteByte(c)
		} else {
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&0x0f])
		}
	}
	return b.String()
}

func dartDataURISafe(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	}
	switch c {
	case '!', '$', '&', '\'', '(', ')', '*', '+', ',', '-', '.', '/', ':', ';', '=', '?', '@', '_', '~':
		return true
	}
	return false
}

// errorToCssString renders err as the error stylesheet Dart produces from
// SassException.toCssString(): the comment is rendered with ASCII glyphs (since
// the user's encoding may not be UTF-8), while the `content` declaration uses
// the error message with the caller's glyph setting, serialized as a quoted
// Sass string with non-ASCII runes hex-escaped.
//
// The comment's closing sequence is neutralized so the message can't break
// out of the comment, and line breaks align on continuation asterisks.
//
// Matches Dart: SassException.toCssString (lib/src/exception.dart:71-113)
func errorToCssString(err error, unicode bool) string {
	// The comment always uses ASCII glyphs, regardless of --unicode.
	commentMessage := renderError(err, sasscommon.HighlightOptions{Glyphs: termglyph.AsciiGlyphs})
	commentMessage = strings.ReplaceAll(commentMessage, "*/", "*∕")
	commentMessage = strings.ReplaceAll(commentMessage, "\r\n", "\n")

	// The string form uses the normal glyphs (no color).
	contentOpts := sasscommon.HighlightOptions{}
	if !unicode {
		contentOpts.Glyphs = termglyph.AsciiGlyphs
	}
	contentMessage := renderError(err, contentOpts)
	quoted, serr := value.SerializeValueInspect(&value.SassString{Text: contentMessage, HasQuotes: true})
	if serr != nil {
		quoted = contentMessage
	}

	// Render non-US-ASCII characters as escape sequences so they display even
	// when the HTTP headers specify the wrong encoding.
	var stringMessage strings.Builder
	for _, r := range quoted {
		if r > 0x7F {
			fmt.Fprintf(&stringMessage, "\\%x ", r)
		} else {
			stringMessage.WriteRune(r)
		}
	}

	comment := strings.Join(strings.Split(commentMessage, "\n"), "\n * ")
	return fmt.Sprintf(
		"/* %s */\n\n"+
			"body::before {\n"+
			"  font-family: \"Source Code Pro\", \"SF Mono\", Monaco, Inconsolata, \"Fira Mono\",\n"+
			"      \"Droid Sans Mono\", monospace, monospace;\n"+
			"  white-space: pre;\n"+
			"  display: block;\n"+
			"  padding: 1em;\n"+
			"  margin-bottom: 1em;\n"+
			"  border-bottom: 2px solid black;\n"+
			"  content: %s;\n"+
			"}",
		comment, stringMessage.String())
}

// renderError renders err with the given highlight options, falling back to
// err.Error() for errors that don't support highlighting.
//
// The interface hop keeps this decoupled from the concrete error taxonomy:
// any error carrying span-aware rendering opts in automatically.
func renderError(err error, opts sasscommon.HighlightOptions) string {
	if e, ok := err.(interface {
		ErrorWithOptions(sasscommon.HighlightOptions) string
	}); ok {
		return e.ErrorWithOptions(opts)
	}
	return err.Error()
}

// isSassException reports whether err is a Sass exception: any of the
// spanned SassException shapes (single- and multi-span, runtime and
// format). Unspanned script errors are excluded — they never render as CSS.
//
// Matches Dart: on SassException catch
func isSassException(err error) bool {
	if _, ok := errors.AsType[*sasscommon.SassException](err); ok {
		return true
	}
	if _, ok := errors.AsType[*sasscommon.SassRuntimeException](err); ok {
		return true
	}
	if _, ok := errors.AsType[*sasscommon.SassFormatException](err); ok {
		return true
	}
	if _, ok := errors.AsType[*sasscommon.MultiSpanSassException](err); ok {
		return true
	}
	if _, ok := errors.AsType[*sasscommon.MultiSpanSassRuntimeException](err); ok {
		return true
	}
	return false
}

// tryDelete deletes path if it exists, silently ignoring errors if the file
// doesn't exist.
//
// A missing file is the expected case (nothing was written yet), so only
// that outcome is swallowed; other deletion failures propagate.
//
// Matches Dart: _tryDelete
func tryDelete(sassIO sassio.IO, path string) {
	if err := sassIO.DeleteFile(path); err != nil {
		if fse, ok := errors.AsType[*sassio.FileSystemException](err); ok && fse.IsNotExist() {
			return
		}
	}
}

// buildHighlightOpts builds HighlightOptions from CompileOptions for error formatting.
//
// Color follows AlertColor; glyphs fall back to ASCII when Unicode output is
// off. A nil opts yields zero options (no color, default glyphs).
func buildHighlightOpts(opts *CompileOptions) sasscommon.HighlightOptions {
	hl := sasscommon.HighlightOptions{}
	if opts != nil {
		hl.Color = opts.AlertColor
		if !opts.Unicode {
			hl.Glyphs = termglyph.AsciiGlyphs
		}
	}
	return hl
}

// StylesheetError is an error with an associated exit code: 65 for a Sass
// failure, 66 for a filesystem failure, mirroring Dart's sysexits usage.
//
// Matches Dart: (int exitCode, String error, String? stackTrace) record
// (stack traces omitted — Go surfaces the message only).
type StylesheetError struct {
	ExitCode int
	Message  string
}

func (e *StylesheetError) Error() string { return e.Message }
