package main

import (
	"testing"

	"github.com/bancek/go-sass/compile"
	"github.com/bancek/go-sass/sassio"
)

func parseArgs(args ...string) (*cliOptions, *usageError) {
	return parseOptions(sassio.NewDefaultIO(), args)
}

func parseErr(t *testing.T, args ...string) string {
	t.Helper()
	_, err := parseArgs(args...)
	if err == nil {
		t.Fatalf("expected usage error for %v", args)
	}
	return err.message
}

func TestNoArgsIsUsageError(t *testing.T) {
	if got := parseErr(t); got != "Compile Sass to CSS." {
		t.Errorf("message = %q", got)
	}
}

func TestSingleFileToStdout(t *testing.T) {
	o, err := parseArgs("input.scss")
	if err != nil {
		t.Fatal(err)
	}
	if len(o.sources) != 1 || o.sources[0].source != "input.scss" || o.sources[0].dest != "" {
		t.Fatalf("sources = %+v", o.sources)
	}
	if o.emitSourceMap {
		t.Error("expected no source map for stdout without --embed-source-map")
	}
}

func TestTwoPositionals(t *testing.T) {
	o, err := parseArgs("in.scss", "out.css")
	if err != nil {
		t.Fatal(err)
	}
	if o.sources[0].source != "in.scss" || o.sources[0].dest != "out.css" {
		t.Fatalf("sources = %+v", o.sources)
	}
	if !o.emitErrorCss {
		t.Error("expected --error-css default true when writing to a file")
	}
}

func TestStdinFlagWithOutput(t *testing.T) {
	o, err := parseArgs("--stdin", "out.css")
	if err != nil {
		t.Fatal(err)
	}
	if o.sources[0].source != "" || o.sources[0].dest != "out.css" {
		t.Fatalf("sources = %+v", o.sources)
	}
}

func TestDashIsStdin(t *testing.T) {
	o, err := parseArgs("-", "out.css")
	if err != nil {
		t.Fatal(err)
	}
	if o.sources[0].source != "" || o.sources[0].dest != "out.css" {
		t.Fatalf("sources = %+v", o.sources)
	}
}

func TestColonForm(t *testing.T) {
	o, err := parseArgs("in.scss:out.css")
	if err != nil {
		t.Fatal(err)
	}
	if o.sources[0].source != "in.scss" || o.sources[0].dest != "out.css" {
		t.Fatalf("sources = %+v", o.sources)
	}
}

func TestTooManyPositionals(t *testing.T) {
	if got := parseErr(t, "a", "b", "c"); got != "Only two positional args may be passed." {
		t.Errorf("message = %q", got)
	}
}

func TestMixingPositionalAndColon(t *testing.T) {
	if got := parseErr(t, "a", "bc:d"); got != `Positional and ":" arguments may not both be used.` {
		t.Errorf("message = %q", got)
	}
}

func TestDoubleColon(t *testing.T) {
	if got := parseErr(t, "in.scss:out:css"); got != `"in.scss:out:css" may only contain one ":".` {
		t.Errorf("message = %q", got)
	}
}

func TestWindowsDriveLetterColonOK(t *testing.T) {
	o, err := parseArgs("C:foo.scss")
	if err != nil {
		t.Fatal(err)
	}
	if o.sources[0].source != "C:foo.scss" || o.sources[0].dest != "" {
		t.Fatalf("sources = %+v", o.sources)
	}
}

func TestStdinTooManyArgs(t *testing.T) {
	if got := parseErr(t, "--stdin", "a", "b"); got != "Only one argument is allowed with --stdin." {
		t.Errorf("message = %q", got)
	}
}

func TestStdinWithColonArg(t *testing.T) {
	if got := parseErr(t, "--stdin", "ab:c"); got != `--stdin may not be used with ":" arguments.` {
		t.Errorf("message = %q", got)
	}
}

func TestNoSourceMapBlocksMapFlags(t *testing.T) {
	if got := parseErr(t, "--no-source-map", "--embed-sources", "a.scss"); got != "--embed-sources isn't allowed with --no-source-map." {
		t.Errorf("message = %q", got)
	}
}

func TestStdoutRequiresEmbedSourceMap(t *testing.T) {
	if got := parseErr(t, "--source-map", "a.scss"); got != "When printing to stdout, --source-map requires --embed-source-map." {
		t.Errorf("message = %q", got)
	}
}

func TestStdoutRelativeURLsBlocked(t *testing.T) {
	if got := parseErr(t, "--source-map-urls=relative", "a.scss"); got != "--source-map-urls=relative isn't allowed when printing to stdout." {
		t.Errorf("message = %q", got)
	}
}

func TestInvalidDeprecation(t *testing.T) {
	if got := parseErr(t, "--silence-deprecation=nope", "a.scss"); got != `Invalid deprecation "nope".` {
		t.Errorf("message = %q", got)
	}
}

func TestStyleAndCharset(t *testing.T) {
	o, err := parseArgs("--style", "compressed", "--no-charset", "a.scs")
	if err != nil {
		t.Fatal(err)
	}
	if o.style != compile.OutputStyleCompressed {
		t.Errorf("style = %v", o.style)
	}
	if o.charset {
		t.Error("expected charset false")
	}
}

func TestInvalidStyle(t *testing.T) {
	if got := parseErr(t, "--style", "bogus", "a.scss"); got != `"bogus" is not an allowed value for option "--style".` {
		t.Errorf("message = %q", got)
	}
}

func TestSplitColonRespectsWindowsDrive(t *testing.T) {
	source, dest, err := splitSourceAndDestination(`C:\foo.scss:out.css`)
	if err != nil {
		t.Fatal(err)
	}
	if source != `C:\foo.scss` || dest != "out.css" {
		t.Fatalf("source=%q dest=%q", source, dest)
	}
}

func TestPrecisionAndAsyncAccepted(t *testing.T) {
	o, err := parseArgs("--precision", "10", "--async", "a.scss")
	if err != nil {
		t.Fatal(err)
	}
	if len(o.sources) != 1 {
		t.Fatalf("sources = %+v", o.sources)
	}
}

func TestNegatableFlagLastWins(t *testing.T) {
	o, err := parseArgs("--charset", "--no-charset", "--charset", "a.scss")
	if err != nil {
		t.Fatal(err)
	}
	if !o.charset {
		t.Error("expected charset true")
	}
}

func TestHelpIsUsageError64(t *testing.T) {
	if got := parseErr(t, "--help"); got != "Compile Sass to CSS." {
		t.Errorf("message = %q", got)
	}
}

func TestVersionShortCircuits(t *testing.T) {
	o, err := parseArgs("--version")
	if err != nil {
		t.Fatal(err)
	}
	if !o.version {
		t.Error("expected version true")
	}
}

func TestLoadPathRepeatable(t *testing.T) {
	o, err := parseArgs("-I", "a", "--load-path", "b", "in.scss")
	if err != nil {
		t.Fatal(err)
	}
	if len(o.loadPaths) != 2 || o.loadPaths[0] != "a" || o.loadPaths[1] != "b" {
		t.Fatalf("loadPaths = %v", o.loadPaths)
	}
}
