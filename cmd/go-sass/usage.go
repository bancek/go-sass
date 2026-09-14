package main

// The Dart-style usage text, matching ExecutableOptions.usage as printed by
// bin/sass.dart's UsageException handler.

import (
	"strings"

	"github.com/bancek/go-sass/sassio"
)

// usageText returns the help text printed for --help and usage errors. It
// excludes the unimplemented --watch/--poll/--update/--interactive flags.
func usageText(sassIO sassio.IO) string {
	bar := "━"
	if sassIO.IsWindows() {
		bar = "="
	}
	bold, reset := "", ""
	if sassIO.HasTerminal() {
		bold = "\x1b[1m"
		reset = "\x1b[0m"
	}
	separator := func(text string) string {
		pad := 40 - 5 - len(text)
		if pad < 0 {
			pad = 0
		}
		return strings.Repeat(bar, 3) + " " + bold + text + reset + " " +
			strings.Repeat(bar, pad)
	}

	var b strings.Builder
	b.WriteString("Usage: sass <input.scss> [output.css]\n")
	b.WriteString("       sass <input.scss>:<output.css> <input/>:<output/> <dir/>\n")
	b.WriteString("\n")

	b.WriteString(separator("Input and Output") + "\n")
	b.WriteString("    --[no-]stdin               Read the stylesheet from stdin.\n")
	b.WriteString("    --[no-]indented            Use the indented syntax for input from stdin.\n")
	b.WriteString("-I, --load-path=<PATH>         A path to use when resolving imports.\n")
	b.WriteString("                               May be passed multiple times.\n")
	b.WriteString("-p, --pkg-importer=<TYPE>      Built-in importer(s) to use for pkg: URLs.\n")
	b.WriteString("\n")
	b.WriteString("          [node]               Load files like Node.js package resolution.\n")
	b.WriteString("\n")
	b.WriteString("-s, --style=<NAME>             Output style.\n")
	b.WriteString("                               [expanded (default), compressed]\n")
	b.WriteString("    --[no-]charset             Emit a @charset or BOM for CSS with non-ASCII characters.\n")
	b.WriteString("                               (defaults to on)\n")
	b.WriteString("    --[no-]error-css           When an error occurs, emit a stylesheet describing it.\n")
	b.WriteString("                               Defaults to true when compiling to a file.\n")
	b.WriteString("\n")

	b.WriteString(separator("Source Maps") + "\n")
	b.WriteString("    --[no-]source-map          Whether to generate source maps.\n")
	b.WriteString("                               (defaults to on)\n")
	b.WriteString("    --source-map-urls          How to link from source maps to source files.\n")
	b.WriteString("                               [relative (default), absolute]\n")
	b.WriteString("    --[no-]embed-sources       Embed source file contents in source maps.\n")
	b.WriteString("    --[no-]embed-source-map    Embed source map contents in CSS.\n")
	b.WriteString("\n")

	b.WriteString(separator("Warnings") + "\n")
	b.WriteString("-q, --[no-]quiet               Don't print warnings.\n")
	b.WriteString("    --[no-]quiet-deps          Don't print compiler warnings from dependencies.\n")
	b.WriteString("                               Stylesheets imported through load paths count as dependencies.\n")
	b.WriteString("    --[no-]verbose             Print all deprecation warnings even when they're repetitive.\n")
	b.WriteString("    --fatal-deprecation        Deprecations to treat as errors. You may also pass a Sass\n")
	b.WriteString("                               version to include any behavior deprecated in or before it.\n")
	b.WriteString("                               See https://sass-lang.com/documentation/breaking-changes for \n")
	b.WriteString("                               a complete list.\n")
	b.WriteString("    --silence-deprecation      Deprecations to ignore.\n")
	b.WriteString("    --future-deprecation       Opt in to a deprecation early.\n")
	b.WriteString("\n")

	b.WriteString(separator("Other") + "\n")
	b.WriteString("    --[no-]stop-on-error       Don't compile more files once an error is encountered.\n")
	b.WriteString("-c, --[no-]color               Whether to use terminal colors for messages.\n")
	b.WriteString("    --[no-]unicode             Whether to use Unicode characters for messages.\n")
	b.WriteString("    --[no-]trace               Print full Dart stack traces for exceptions.\n")
	b.WriteString("-h, --help                     Print this usage information.\n")
	b.WriteString("    --version                  Print the version of Dart Sass.\n")

	return b.String()
}
