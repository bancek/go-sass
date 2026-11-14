// Copyright 2024 Google LLC. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package deprecation

// dart-source: lib/src/deprecation.dart

import (
	"fmt"
	"strconv"
	"strings"
)

// Deprecation is a deprecated feature in the language.
//
// Dart models this as an enum with one variant per deprecation (plus a
// future constructor for deprecations not yet tied to a version). Go uses a
// struct with one package-level value per deprecation instead, since the set
// stays open to host-defined entries such as UserAuthored. Look values up by
// kebab-case ID with FromID, and list the deprecations live in or before a
// release with ForVersion.
type Deprecation struct {
	// ID is the unique kebab-case identifier, used to refer to the
	// deprecation on the command line.
	ID string
	// DeprecatedIn is the Dart Sass version that first deprecated the
	// feature ("0.0.0" for deprecations predating versioning), or empty
	// when tied to no release. Dart exposes this as a parsed Version; Go
	// keeps the underlying version string.
	DeprecatedIn string // version string, empty if none
	// Description sketches the deprecated feature for CLI usage. An empty
	// description hides the deprecation from the listing.
	Description string
	// ObsoleteIn is the version that fully removed the feature, or empty
	// while it is still only deprecated.
	ObsoleteIn string // version string, empty if none
	// IsFuture marks deprecations that will occur in the future (Dart's
	// future constructor): DeprecatedIn is then empty since the live
	// version is not yet known.
	IsFuture bool
}

func (d *Deprecation) String() string { return d.ID }

// allDeprecations is populated via the variable declarations below.
var allDeprecations []*Deprecation

// FromID returns the deprecation with the given kebab-case ID, or nil when
// no deprecation uses it.
//
// Matches Dart: Deprecation.fromId
func FromID(id string) *Deprecation {
	for _, d := range allDeprecations {
		if d.ID == id {
			return d
		}
	}
	return nil
}

// DartSass2 returns the set of deprecations that will become fatal errors in
// Dart Sass 2.0.0.
//
// It is the set deprecated in or before 1.105.0 minus the four that are
// staying warnings: color-module-compat, import, global-builtin, and
// if-function.
//
// Matches Dart: Deprecation.dartSass2
func DartSass2() []*Deprecation {
	var result []*Deprecation
	for _, d := range ForVersion("1.105.0") {
		switch d {
		case ColorModuleCompat, Import, GlobalBuiltin, IfFunction:
			continue
		}
		result = append(result, d)
	}
	return result
}

// ForVersion returns all deprecations deprecated in or before the given version,
// excluding those that are obsolete. A version beyond the current Dart Sass
// version also emits the deprecation-version warning through the default
// logger.
//
// A deprecation counts when its DeprecatedIn parses at or below version and
// ObsoleteIn is empty; deprecations tied to no release never count.
//
// Matches Dart: Deprecation.forVersion
func ForVersion(version string) []*Deprecation {
	return ForVersionInternal(version, func(*Deprecation, string) error { return nil })
}

// DeprecationVersionWarning reports a deprecation-version warning for a
// version beyond the current Dart Sass version. It returns false when no
// warning is due.
//
// Go callers pass this warning to their logger explicitly: deprecation cannot
// import sasslogger (which imports this package), so the default logger is
// supplied by the caller exactly as Dart reaches it through
// EvaluationContext.currentOrNull.
//
// Matches Dart: the warning inside Deprecation.forVersionInternal
func DeprecationVersionWarning(version string) (string, bool) {
	major, minor, patch := parseVersion(version)
	currentMajor, currentMinor, currentPatch := parseVersion(CurrentVersion)
	if !versionLessThan(currentMajor, currentMinor, currentPatch, major, minor, patch) {
		return "", false
	}
	return fmt.Sprintf(
		"Version \"%s\" is beyond the current version of Dart Sass. "+
			"Selecting deprecations from later versions is deprecated and will be "+
			"an error in Dart Sass 3.0.\n\n"+
			"More info: http://sass-lang.com/d/deprecation-version",
		version), true
}

// ForVersionInternal is ForVersion with an explicit warning callback for the
// deprecation-version warning.
//
// Matches Dart: Deprecation.forVersionInternal
func ForVersionInternal(version string, warn func(*Deprecation, string) error) []*Deprecation {
	major, minor, patch := parseVersion(version)
	if message, ok := DeprecationVersionWarning(version); ok {
		_ = warn(DeprecationVersion, message)
	}

	var result []*Deprecation
	for _, d := range allDeprecations {
		if d.DeprecatedIn == "" {
			continue
		}
		dMajor, dMinor, dPatch := parseVersion(d.DeprecatedIn)
		if d.ObsoleteIn == "" && !versionLessThan(major, minor, patch, dMajor, dMinor, dPatch) {
			result = append(result, d)
		}
	}
	return result
}

// CurrentVersion is the current version of Dart Sass, which this port
// implements. It mirrors Dart's dartSassVersion.
const CurrentVersion = "1.107.1"

// parseVersion splits a dotted version into major/minor/patch numbers,
// treating missing or non-numeric parts as zero.
func parseVersion(v string) (major, minor, patch int) {
	parts := strings.Split(v, ".")
	if len(parts) >= 1 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) >= 3 {
		patch, _ = strconv.Atoi(parts[2])
	}
	return
}

// versionLessThan compares two semantic versions by major, then minor,
// then patch.
func versionLessThan(m1, n1, p1, m2, n2, p2 int) bool {
	if m1 != m2 {
		return m1 < m2
	}
	if n1 != n2 {
		return n1 < n2
	}
	return p1 < p2
}

// START AUTOGENERATED CODE
//
// DO NOT EDIT. This section was generated from the language repo.
// See tool/grind/generate_deprecations.dart for details.
//
// Checksum: 2c5800a683ec3b36c9e8d7f6eb427466b7439461

var (
	// CallString deprecates passing a string directly to meta.call().
	CallString = &Deprecation{
		ID:           "call-string",
		DeprecatedIn: "0.0.0",
		Description:  "Passing a string directly to meta.call().",
	}
	// Elseif deprecates @elseif in favor of @else if.
	Elseif = &Deprecation{
		ID:           "elseif",
		DeprecatedIn: "1.3.2",
		Description:  "@elseif.",
	}
	// MozDocument deprecates @-moz-document.
	MozDocument = &Deprecation{
		ID:           "moz-document",
		DeprecatedIn: "1.7.2",
		Description:  "@-moz-document.",
	}
	// RelativeCanonical deprecates imports using relative canonical URLs.
	RelativeCanonical = &Deprecation{
		ID:           "relative-canonical",
		DeprecatedIn: "1.14.2",
		Description:  "Imports using relative canonical URLs.",
	}
	// NewGlobal deprecates declaring new variables with !global.
	NewGlobal = &Deprecation{
		ID:           "new-global",
		DeprecatedIn: "1.17.2",
		Description:  "Declaring new variables with !global.",
	}
	// ColorModuleCompat deprecates using color module functions in place of
	// plain CSS functions.
	ColorModuleCompat = &Deprecation{
		ID:           "color-module-compat",
		DeprecatedIn: "1.23.0",
		Description:  "Using color module functions in place of plain CSS functions.",
	}
	// SlashDiv deprecates the / operator for division.
	SlashDiv = &Deprecation{
		ID:           "slash-div",
		DeprecatedIn: "1.33.0",
		Description:  "/ operator for division.",
	}
	// BogusCombinators deprecates leading, trailing, and repeated combinators.
	BogusCombinators = &Deprecation{
		ID:           "bogus-combinators",
		DeprecatedIn: "1.54.0",
		Description:  "Leading, trailing, and repeated combinators.",
	}
	// StrictUnary deprecates ambiguous + and - operators.
	StrictUnary = &Deprecation{
		ID:           "strict-unary",
		DeprecatedIn: "1.55.0",
		Description:  "Ambiguous + and - operators.",
	}
	// FunctionUnits deprecates passing invalid units to built-in functions.
	FunctionUnits = &Deprecation{
		ID:           "function-units",
		DeprecatedIn: "1.56.0",
		Description:  "Passing invalid units to built-in functions.",
	}
	// DuplicateVarFlags deprecates using !default or !global multiple times
	// for one variable.
	DuplicateVarFlags = &Deprecation{
		ID:           "duplicate-var-flags",
		DeprecatedIn: "1.62.0",
		Description:  "Using !default or !global multiple times for one variable.",
	}
	// NullAlpha deprecates passing null as alpha in the host API.
	NullAlpha = &Deprecation{
		ID:           "null-alpha",
		DeprecatedIn: "1.62.3",
		Description:  "Passing null as alpha in the Dart API.",
	}
	// AbsPercent deprecates passing percentages to the Sass abs() function.
	AbsPercent = &Deprecation{
		ID:           "abs-percent",
		DeprecatedIn: "1.65.0",
		Description:  "Passing percentages to the Sass abs() function.",
	}
	// FsImporterCwd deprecates using the current working directory as an
	// implicit load path.
	FsImporterCwd = &Deprecation{
		ID:           "fs-importer-cwd",
		DeprecatedIn: "1.73.0",
		Description:  "Using the current working directory as an implicit load path.",
	}
	// CssFunctionMixin deprecates function and mixin names beginning with
	// -- (fully removed in 1.94.0).
	CssFunctionMixin = &Deprecation{
		ID:           "css-function-mixin",
		DeprecatedIn: "1.76.0",
		ObsoleteIn:   "1.94.0",
		Description:  "Function and mixin names beginning with --.",
	}
	// MixedDecls deprecates declarations after or between nested rules
	// (fully removed in 1.92.0).
	MixedDecls = &Deprecation{
		ID:           "mixed-decls",
		DeprecatedIn: "1.77.7",
		ObsoleteIn:   "1.92.0",
		Description:  "Declarations after or between nested rules.",
	}
	// FeatureExists deprecates meta.feature-exists.
	FeatureExists = &Deprecation{
		ID:           "feature-exists",
		DeprecatedIn: "1.78.0",
		Description:  "meta.feature-exists",
	}
	// Color4Api deprecates certain uses of built-in sass:color functions.
	Color4Api = &Deprecation{
		ID:           "color-4-api",
		DeprecatedIn: "1.79.0",
		Description:  "Certain uses of built-in sass:color functions.",
	}
	// ColorFunctions deprecates using global color functions instead of
	// sass:color.
	ColorFunctions = &Deprecation{
		ID:           "color-functions",
		DeprecatedIn: "1.79.0",
		Description:  "Using global color functions instead of sass:color.",
	}
	// LegacyJsApi deprecates the legacy JS API.
	LegacyJsApi = &Deprecation{
		ID:           "legacy-js-api",
		DeprecatedIn: "1.79.0",
		Description:  "Legacy JS API.",
	}
	// Import deprecates @import rules.
	Import = &Deprecation{
		ID:           "import",
		DeprecatedIn: "1.80.0",
		Description:  "@import rules.",
	}
	// GlobalBuiltin deprecates global built-in functions that are available
	// in sass: modules.
	GlobalBuiltin = &Deprecation{
		ID:           "global-builtin",
		DeprecatedIn: "1.80.0",
		Description:  "Global built-in functions that are available in sass: modules.",
	}
	// TypeFunction deprecates functions named "type" (fully removed in 1.92.0).
	TypeFunction = &Deprecation{
		ID:           "type-function",
		DeprecatedIn: "1.86.0",
		ObsoleteIn:   "1.92.0",
		Description:  "Functions named \"type\".",
	}
	// CompileStringRelativeUrl deprecates passing a relative url to
	// compileString().
	CompileStringRelativeUrl = &Deprecation{
		ID:           "compile-string-relative-url",
		DeprecatedIn: "1.88.0",
		Description:  "Passing a relative url to compileString().",
	}
	// MisplacedRest deprecates a rest parameter before a positional or named
	// parameter.
	MisplacedRest = &Deprecation{
		ID:           "misplaced-rest",
		DeprecatedIn: "1.91.0",
		Description:  "A rest parameter before a positional or named parameter.",
	}
	// WithPrivate deprecates configuring private variables in @use, @forward,
	// or load-css().
	WithPrivate = &Deprecation{
		ID:           "with-private",
		DeprecatedIn: "1.92.0",
		Description:  "Configuring private variables in @use, @forward, or load-css().",
	}
	// IfFunction deprecates the Sass if($condition, $if-true, $if-false)
	// function.
	IfFunction = &Deprecation{
		ID:           "if-function",
		DeprecatedIn: "1.95.0",
		Description:  "The Sass if($condition, $if-true, $if-false) function.",
	}
	// FunctionName deprecates uppercase reserved function names.
	FunctionName = &Deprecation{
		ID:           "function-name",
		DeprecatedIn: "1.98.0",
		Description:  "Uppercase reserved function names.",
	}
	// AdjacentCompounds deprecates adjacent compound selectors like [class]a.
	AdjacentCompounds = &Deprecation{
		ID:           "adjacent-compounds",
		DeprecatedIn: "1.100.0",
		Description:  "Adjacent compound selectors like `[class]a`.",
	}

	// DeprecationVersion deprecates selecting versions beyond the current
	// one for fatalDeprecations.
	DeprecationVersion = &Deprecation{
		ID:           "deprecation-version",
		DeprecatedIn: "1.107.0",
		Description:  "Selecting versions beyond the current one for `fatalDeprecations`.",
	}

	// END AUTOGENERATED CODE

	// UserAuthored covers deprecations coming from user-authored code rather
	// than the language itself. It carries no version, so ForVersion skips it.
	UserAuthored = &Deprecation{
		ID: "user-authored",
	}
	// CalcInterp is a retired deprecation name that was never actually used.
	// It stays registered so old IDs still resolve, but nothing warns for it.
	CalcInterp = &Deprecation{
		ID: "calc-interp",
	}
)

func init() {
	allDeprecations = []*Deprecation{
		CallString,
		Elseif,
		MozDocument,
		RelativeCanonical,
		NewGlobal,
		ColorModuleCompat,
		SlashDiv,
		BogusCombinators,
		StrictUnary,
		FunctionUnits,
		DuplicateVarFlags,
		NullAlpha,
		AbsPercent,
		FsImporterCwd,
		CssFunctionMixin,
		MixedDecls,
		FeatureExists,
		Color4Api,
		ColorFunctions,
		LegacyJsApi,
		Import,
		GlobalBuiltin,
		TypeFunction,
		CompileStringRelativeUrl,
		MisplacedRest,
		WithPrivate,
		IfFunction,
		FunctionName,
		AdjacentCompounds,
		DeprecationVersion,
		UserAuthored,
		CalcInterp,
	}
}
