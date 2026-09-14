package main

// Command-line option parsing for the `sass` executable.
//
// Port of Dart Sass's ExecutableOptions (lib/src/executable/options.dart).
// pflag handles flag tokenization; this file mirrors the post-parse "resolver"
// semantics (positional classification and validation, source-map gating,
// deprecation resolution).

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bancek/go-sass/compile"
	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/eval"
	"github.com/bancek/go-sass/sassio"
	"github.com/bancek/go-sass/sasslogger"
	"github.com/spf13/pflag"
)

// sassVersion is the version reported by --version and used for
// --fatal-deprecation version comparisons. It tracks the dart-sass version
// implemented by this port and the embedded compiler version.
const sassVersion = "1.104.0"

// usageError mirrors Dart's UsageException (exit code 64).
type usageError struct{ message string }

func (e *usageError) Error() string { return e.message }

// sourceDest is a single compilation, keyed by source path (empty = stdin) to
// destination path (empty = stdout).
type sourceDest struct {
	source string
	dest   string
}

// cliOptions is the parsed and validated CLI options.
type cliOptions struct {
	sources []sourceDest

	indented       bool
	emitSourceMap  bool
	emitErrorCss   bool
	embedSources   bool
	embedSourceMap bool
	sourceMapURLs  string

	silent    bool
	verbose   bool
	quietDeps bool
	style     compile.OutputStyle
	charset   bool
	unicode   bool
	color     bool
	trace     bool

	stopOnError bool

	loadPaths []string
	importers []eval.Importer

	silenceDeprecations []*deprecation.Deprecation
	fatalDeprecations   []*deprecation.Deprecation
	futureDeprecations  []*deprecation.Deprecation

	// Go-specific profiling flags (hidden; not part of the Dart CLI).
	cpuProfile string
	memProfile string

	version bool
}

// negBoolValue is a pflag.Value backing a --name / --no-name pair. The two
// names share one target so the last one parsed wins, matching Dart's
// negatable-by-default flags.
type negBoolValue struct {
	target   *bool
	negative bool
}

func (v *negBoolValue) String() string {
	if v.target == nil {
		return "false"
	}
	return strconv.FormatBool(*v.target)
}

func (v *negBoolValue) Set(s string) error {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	if v.negative {
		*v.target = !b
	} else {
		*v.target = b
	}
	return nil
}

func (v *negBoolValue) Type() string { return "bool" }

// parsed reports whether name was explicitly provided, either as --name or as
// its --no-name negation.
func parsed(fs *pflag.FlagSet, name string) bool {
	return fs.Changed(name) || fs.Changed("no-"+name)
}

// dartParseError translates pflag's parse errors into the wording used by
// Dart's args package where possible.
func dartParseError(err error) string {
	msg := err.Error()
	if rest, ok := strings.CutPrefix(msg, "unknown flag: "); ok {
		return fmt.Sprintf("Could not find an option named %q.", rest)
	}
	if rest, ok := strings.CutPrefix(msg, "unknown shorthand flag: "); ok {
		// pflag: "unknown shorthand flag: 'z' in -z"
		if strings.HasPrefix(rest, "'") {
			if end := strings.Index(rest, "' in "); end > 0 {
				return fmt.Sprintf("Could not find an option or flag %q.", "-"+rest[1:end])
			}
		}
	}
	return msg
}

// parseOptions parses args (without the program name) and validates them.
func parseOptions(sassIO sassio.IO, args []string) (*cliOptions, *usageError) {
	fs := pflag.NewFlagSet("sass", pflag.ContinueOnError)
	fs.SetInterspersed(true)
	fs.SetOutput(io.Discard)

	addBool := func(name, shorthand, usage string, def bool) *bool {
		p := new(bool)
		*p = def
		if shorthand != "" {
			fs.VarP(&negBoolValue{target: p}, name, shorthand, usage)
		} else {
			fs.Var(&negBoolValue{target: p}, name, usage)
		}
		fs.Var(&negBoolValue{target: p, negative: true}, "no-"+name, "")
		fs.Lookup(name).NoOptDefVal = "true"
		no := fs.Lookup("no-" + name)
		no.NoOptDefVal = "true"
		no.Hidden = true
		return p
	}

	stdin := addBool("stdin", "", "Read the stylesheet from stdin.", false)
	indented := addBool("indented", "", "Use the indented syntax for input from stdin.", false)

	var loadPaths []string
	fs.StringArrayVarP(&loadPaths, "load-path", "I", nil,
		"A path to use when resolving imports. May be passed multiple times.")

	var pkgImporters []string
	fs.StringArrayVarP(&pkgImporters, "pkg-importer", "p", nil,
		"Built-in importer(s) to use for pkg: URLs.")

	var styleName string
	fs.StringVarP(&styleName, "style", "s", "expanded", "Output style.")

	charset := addBool("charset", "", "Emit a @charset or BOM for CSS with non-ASCII characters.", true)
	errorCSS := addBool("error-css", "", "When an error occurs, emit a stylesheet describing it.", false)

	sourceMap := addBool("source-map", "", "Whether to generate source maps.", true)
	var sourceMapURLs string
	fs.StringVar(&sourceMapURLs, "source-map-urls", "relative",
		"How to link from source maps to source files.")
	embedSources := addBool("embed-sources", "", "Embed source file contents in source maps.", false)
	embedSourceMap := addBool("embed-source-map", "", "Embed source map contents in CSS.", false)

	quiet := addBool("quiet", "q", "Don't print warnings.", false)
	quietDeps := addBool("quiet-deps", "", "Don't print compiler warnings from dependencies.", false)
	verbose := addBool("verbose", "", "Print all deprecation warnings even when they're repetitive.", false)

	var fatalDeprecation, silenceDeprecation, futureDeprecation []string
	fs.StringArrayVar(&fatalDeprecation, "fatal-deprecation", nil,
		"Deprecations to treat as errors. You may also pass a Sass version to include any behavior deprecated in or before it.")
	fs.StringArrayVar(&silenceDeprecation, "silence-deprecation", nil, "Deprecations to ignore.")
	fs.StringArrayVar(&futureDeprecation, "future-deprecation", nil, "Opt in to a deprecation early.")

	stopOnError := addBool("stop-on-error", "", "Don't compile more files once an error is encountered.", false)
	trace := addBool("trace", "", "Print full stack traces for exceptions.", false)
	color := addBool("color", "c", "Whether to use terminal colors for messages.", false)
	unicode := addBool("unicode", "", "Whether to use Unicode characters for messages.", true)

	var help, version bool
	fs.BoolVarP(&help, "help", "h", false, "Print this usage information.")
	fs.BoolVar(&version, "version", false, "Print the version of Dart Sass.")

	// Hidden no-ops, accepted for sass-spec compatibility.
	var precision int
	fs.IntVar(&precision, "precision", 0, "")
	_ = fs.MarkHidden("precision")
	var async bool
	fs.BoolVar(&async, "async", false, "")
	_ = fs.MarkHidden("async")

	// Go-specific profiling flags (hidden; not part of the Dart CLI).
	var cpuProfile, memProfile string
	fs.StringVar(&cpuProfile, "cpuprofile", "", "write CPU profile to file")
	_ = fs.MarkHidden("cpuprofile")
	fs.StringVar(&memProfile, "memprofile", "", "write memory profile to file")
	_ = fs.MarkHidden("memprofile")

	if err := fs.Parse(args); err != nil {
		return nil, &usageError{message: dartParseError(err)}
	}
	_, _ = precision, async

	// Dart routes --help through a UsageException, so it prints the message,
	// the usage text, and exits 64.
	if help {
		return nil, &usageError{message: "Compile Sass to CSS."}
	}

	// --version short-circuits before any source resolution.
	if version {
		return &cliOptions{version: true}, nil
	}

	var style compile.OutputStyle
	switch styleName {
	case "expanded":
		style = compile.OutputStyleExpanded
	case "compressed":
		style = compile.OutputStyleCompressed
	default:
		return nil, &usageError{message: fmt.Sprintf("%q is not an allowed value for option \"--style\".", styleName)}
	}

	if sourceMapURLs != "relative" && sourceMapURLs != "absolute" {
		return nil, &usageError{message: fmt.Sprintf("%q is not an allowed value for option \"--source-map-urls\".", sourceMapURLs)}
	}

	var importers []eval.Importer
	for _, kind := range pkgImporters {
		if kind != "node" {
			return nil, &usageError{message: fmt.Sprintf("%q is not an allowed value for option \"--pkg-importer\".", kind)}
		}
		importers = append(importers, eval.NewNodePackageImporter(".", sassIO))
	}

	silenceDeprecations, ue := resolveDeprecations(silenceDeprecation)
	if ue != nil {
		return nil, ue
	}
	futureDeprecations, ue := resolveDeprecations(futureDeprecation)
	if ue != nil {
		return nil, ue
	}
	fatalDeprecations, ue := resolveFatalDeprecations(fatalDeprecation)
	if ue != nil {
		return nil, ue
	}

	sources, ue := resolveSources(sassIO, fs.Args(), *stdin)
	if ue != nil {
		return nil, ue
	}

	emitSourceMap, ue := resolveEmitSourceMap(fs, *sourceMap, sourceMapURLs, sources)
	if ue != nil {
		return nil, ue
	}

	emitErrorCSS := *errorCSS
	if !parsed(fs, "error-css") {
		emitErrorCSS = false
		for _, sd := range sources {
			if sd.dest != "" {
				emitErrorCSS = true
				break
			}
		}
	}

	alertColor := *color
	if !parsed(fs, "color") {
		alertColor = sassIO.SupportsAnsiEscapes()
	}

	return &cliOptions{
		sources:             sources,
		indented:            *indented,
		emitSourceMap:       emitSourceMap,
		emitErrorCss:        emitErrorCSS,
		embedSources:        *embedSources,
		embedSourceMap:      *embedSourceMap,
		sourceMapURLs:       sourceMapURLs,
		silent:              *quiet,
		verbose:             *verbose,
		quietDeps:           *quietDeps,
		style:               style,
		charset:             *charset,
		unicode:             *unicode,
		color:               alertColor,
		trace:               *trace,
		stopOnError:         *stopOnError,
		loadPaths:           loadPaths,
		importers:           importers,
		silenceDeprecations: silenceDeprecations,
		fatalDeprecations:   fatalDeprecations,
		futureDeprecations:  futureDeprecations,
		cpuProfile:          cpuProfile,
		memProfile:          memProfile,
		version:             version,
	}, nil
}

// logger returns the logger used for a compilation.
func (o *cliOptions) logger() sasslogger.Logger {
	if o.silent {
		return sasslogger.Quiet
	}
	return sasslogger.NewStderrLogger(o.color, o.unicode)
}

// compileOptions builds the library CompileOptions for a compilation.
func (o *cliOptions) compileOptions() *compile.CompileOptions {
	opts := &compile.CompileOptions{
		Importers:           o.importers,
		LoadPaths:           append([]string{}, o.loadPaths...),
		Logger:              o.logger(),
		QuietDeps:           o.quietDeps,
		SourceMap:           o.emitSourceMap,
		EmbedSources:        o.embedSources,
		EmbedSourceMap:      o.embedSourceMap,
		SourceMapURLs:       o.sourceMapURLs,
		EmitErrorCss:        o.emitErrorCss,
		Style:               o.style,
		Charset:             o.charset,
		SilenceDeprecations: o.silenceDeprecations,
		FatalDeprecations:   o.fatalDeprecations,
		FutureDeprecations:  o.futureDeprecations,
		Verbose:             o.verbose,
		Unicode:             o.unicode,
		AlertColor:          o.color,
		AlertAscii:          !o.unicode,
	}
	if o.indented {
		opts.Syntax = eval.SyntaxSass
	}
	return opts
}

// resolveSources classifies positional arguments into source → destination
// pairs, mirroring ExecutableOptions._ensureSources (options.dart:319-428).
func resolveSources(sassIO sassio.IO, rest []string, stdin bool) ([]sourceDest, *usageError) {
	if len(rest) == 0 && !stdin {
		return nil, &usageError{message: "Compile Sass to CSS."}
	}

	directories := map[string]bool{}
	colonArgs := false
	positionalArgs := false
	for _, argument := range rest {
		if argument == "" {
			return nil, &usageError{message: `Invalid argument "".`}
		}
		if containsColon(argument) {
			colonArgs = true
		} else if sassIO.DirExists(argument) {
			directories[argument] = true
		} else {
			positionalArgs = true
		}
	}

	if positionalArgs || len(rest) == 0 {
		if colonArgs {
			return nil, &usageError{message: `Positional and ":" arguments may not both be used.`}
		}
		if stdin {
			if len(rest) > 1 {
				return nil, &usageError{message: "Only one argument is allowed with --stdin."}
			}
			dest := ""
			if len(rest) == 1 {
				dest = rest[0]
			}
			return []sourceDest{{source: "", dest: dest}}, nil
		}
		if len(rest) > 2 {
			return nil, &usageError{message: "Only two positional args may be passed."}
		}
		if len(directories) != 0 {
			firstDir := ""
			for _, a := range rest {
				if directories[a] {
					firstDir = a
					break
				}
			}
			target := rest[len(rest)-1]
			message := fmt.Sprintf("Directory %q may not be a positional arg.", firstDir)
			if firstDir == rest[0] && !sassIO.FileExists(target) {
				message += fmt.Sprintf(
					"\nTo compile all CSS in %q to %q, use `sass %s:%s`.",
					firstDir, target, firstDir, target)
			}
			return nil, &usageError{message: message}
		}
		source := rest[0]
		if source == "-" {
			source = ""
		}
		dest := ""
		if len(rest) == 2 {
			dest = rest[1]
		}
		return []sourceDest{{source: source, dest: dest}}, nil
	}

	if stdin {
		return nil, &usageError{message: `--stdin may not be used with ":" arguments.`}
	}

	seen := map[string]bool{}
	var sources []sourceDest
	for _, argument := range rest {
		if directories[argument] {
			if seen[argument] {
				return nil, &usageError{message: fmt.Sprintf("Duplicate source %q.", argument)}
			}
			seen[argument] = true
			listed, ue := listSourceDirectory(sassIO, argument, argument)
			if ue != nil {
				return nil, ue
			}
			for _, sd := range listed {
				if !seen[sd.source] {
					seen[sd.source] = true
					sources = append(sources, sd)
				}
			}
			continue
		}

		source, destination, ue := splitSourceAndDestination(argument)
		if ue != nil {
			return nil, ue
		}
		if seen[source] {
			return nil, &usageError{message: fmt.Sprintf("Duplicate source %q.", source)}
		}
		seen[source] = true

		if source == "-" {
			sources = append(sources, sourceDest{source: "", dest: destination})
		} else if sassIO.DirExists(source) {
			listed, ue := listSourceDirectory(sassIO, source, destination)
			if ue != nil {
				return nil, ue
			}
			for _, sd := range listed {
				if !seen[sd.source] {
					seen[sd.source] = true
					sources = append(sources, sd)
				}
			}
		} else {
			sources = append(sources, sourceDest{source: source, dest: destination})
		}
	}
	return sources, nil
}

// containsColon reports whether argument contains a source/destination
// separator colon (ignoring a Windows drive letter).
func containsColon(argument string) bool {
	if !strings.Contains(argument, ":") {
		return false
	}
	if !isWindowsPath(argument, 0) {
		return true
	}
	return len(argument) > 2 && strings.Contains(argument[2:], ":")
}

// isWindowsPath reports whether s contains an absolute Windows path at index.
func isWindowsPath(s string, index int) bool {
	return len(s) > index+2 && isAlphabetic(s[index]) && s[index+1] == ':'
}

func isAlphabetic(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// splitSourceAndDestination splits an argument on its separator colon.
//
// Matches Dart: ExecutableOptions._splitSourceAndDestination (options.dart:432-452)
func splitSourceAndDestination(argument string) (string, string, *usageError) {
	for i := 0; i < len(argument); i++ {
		// A colon at position 1 may be a Windows drive letter.
		if i == 1 && isWindowsPath(argument, i-1) {
			continue
		}
		if argument[i] != ':' {
			continue
		}
		nextColon := strings.IndexByte(argument[i+1:], ':')
		if nextColon != -1 {
			nextColon += i + 1
		}
		// A colon 2 characters after the separator may also be a Windows
		// drive letter.
		if nextColon == i+2 && isWindowsPath(argument, i+1) {
			nextColon = strings.IndexByte(argument[nextColon+1:], ':')
			if nextColon != -1 {
				nextColon += i + 3
			}
		}
		if nextColon != -1 {
			return "", "", &usageError{message: fmt.Sprintf("%q may only contain one \":\".", argument)}
		}
		return argument[:i], argument[i+1:], nil
	}
	return "", "", &usageError{message: fmt.Sprintf("Expected %q to contain a colon.", argument)}
}

// listSourceDirectory lists the entrypoint sources in source, mapped to
// destinations under destination.
//
// Matches Dart: ExecutableOptions._listSourceDirectory (options.dart:462-473)
func listSourceDirectory(sassIO sassio.IO, source, destination string) ([]sourceDest, *usageError) {
	files, err := sassIO.ListDir(source, true)
	if err != nil {
		return nil, &usageError{message: err.Error()}
	}
	var out []sourceDest
	for _, path := range files {
		if !isEntrypoint(path) {
			continue
		}
		// Don't compile a CSS file to its own location.
		if source == destination && filepath.Ext(path) == ".css" {
			continue
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			rel = path
		}
		rel = strings.TrimSuffix(rel, filepath.Ext(rel)) + ".css"
		out = append(out, sourceDest{source: path, dest: filepath.Join(destination, rel)})
	}
	return out, nil
}

// isEntrypoint reports whether path is a Sass entrypoint (not a partial).
func isEntrypoint(path string) bool {
	if strings.HasPrefix(filepath.Base(path), "_") {
		return false
	}
	switch filepath.Ext(path) {
	case ".scss", ".sass", ".css":
		return true
	}
	return false
}

// resolveEmitSourceMap resolves the emitSourceMap decision, mirroring
// ExecutableOptions.emitSourceMap (options.dart:488-525).
func resolveEmitSourceMap(fs *pflag.FlagSet, sourceMap bool, sourceMapURLs string, sources []sourceDest) (bool, *usageError) {
	if !sourceMap {
		if fs.Changed("source-map-urls") {
			return false, &usageError{message: "--source-map-urls isn't allowed with --no-source-map."}
		}
		if parsed(fs, "embed-sources") {
			return false, &usageError{message: "--embed-sources isn't allowed with --no-source-map."}
		}
		if parsed(fs, "embed-source-map") {
			return false, &usageError{message: "--embed-source-map isn't allowed with --no-source-map."}
		}
	}

	writeToStdout := len(sources) == 1 && sources[0].dest == ""
	if !writeToStdout {
		return sourceMap, nil
	}

	if fs.Changed("source-map-urls") && sourceMapURLs == "relative" {
		return false, &usageError{message: "--source-map-urls=relative isn't allowed when printing to stdout."}
	}
	if parsed(fs, "embed-source-map") {
		return sourceMap, nil
	}
	if parsed(fs, "source-map") && sourceMap {
		return false, &usageError{message: "When printing to stdout, --source-map requires --embed-source-map."}
	}
	if fs.Changed("source-map-urls") {
		return false, &usageError{message: "When printing to stdout, --source-map-urls requires --embed-source-map."}
	}
	if parsed(fs, "embed-sources") {
		return false, &usageError{message: "When printing to stdout, --embed-sources requires --embed-source-map."}
	}
	return false, nil
}

// resolveDeprecations resolves deprecation ids, rejecting unknown ones.
func resolveDeprecations(ids []string) ([]*deprecation.Deprecation, *usageError) {
	var out []*deprecation.Deprecation
	for _, id := range ids {
		d := deprecation.FromID(id)
		if d == nil {
			return nil, &usageError{message: fmt.Sprintf("Invalid deprecation %q.", id)}
		}
		out = append(out, d)
	}
	return out, nil
}

// resolveFatalDeprecations resolves --fatal-deprecation values, which may be
// either a deprecation id or a Sass version.
func resolveFatalDeprecations(ids []string) ([]*deprecation.Deprecation, *usageError) {
	var out []*deprecation.Deprecation
	seen := map[*deprecation.Deprecation]bool{}
	add := func(d *deprecation.Deprecation) {
		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	for _, id := range ids {
		if d := deprecation.FromID(id); d != nil {
			add(d)
			continue
		}
		v, ok := parseVersion(id)
		if !ok {
			return nil, &usageError{message: fmt.Sprintf("Invalid deprecation %q.", id)}
		}
		if cmp, ok := compareVersions(v, mustParseVersion(sassVersion)); ok && cmp > 0 {
			return nil, &usageError{message: fmt.Sprintf(
				"Invalid version %s. --fatal-deprecation requires a version less than or equal to the current Dart Sass version.",
				id)}
		}
		for _, d := range deprecation.ForVersion(id) {
			add(d)
		}
	}
	return out, nil
}

type semver struct{ major, minor, patch int }

func parseVersion(s string) (semver, bool) {
	parts := strings.Split(s, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return semver{}, false
	}
	var v semver
	var err error
	v.major, err = strconv.Atoi(parts[0])
	if err != nil {
		return semver{}, false
	}
	v.minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return semver{}, false
	}
	if len(parts) == 3 {
		v.patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return semver{}, false
		}
	}
	return v, true
}

func mustParseVersion(s string) semver {
	v, _ := parseVersion(s)
	return v
}

// compareVersions returns -1, 0, or 1 comparing a to b.
func compareVersions(a, b semver) (int, bool) {
	switch {
	case a.major != b.major:
		if a.major < b.major {
			return -1, true
		}
		return 1, true
	case a.minor != b.minor:
		if a.minor < b.minor {
			return -1, true
		}
		return 1, true
	case a.patch != b.patch:
		if a.patch < b.patch {
			return -1, true
		}
		return 1, true
	}
	return 0, true
}
