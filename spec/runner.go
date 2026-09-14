package spec

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/bancek/go-sass/compile"
	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassio"
	"github.com/bancek/go-sass/sasslogger"
)

const implName = "dart-sass-go"

// implNames is the chain of implementation names to try when looking up
// implementation-specific expected files (output-{name}.css, error-{name},
// warning-{name}) and when checking :ignore_for/:todo/:warning_todo in
// options.yml. We try the most specific name first, then fall back to
// the reference implementation's name.
var implNames = []string{"dart-sass-go", "dart-sass"}

// specStats tracks the outcome of HRX tests that actually ran.
type specStats struct {
	passed int
	failed int
	todo   int
}

// RunSpecs walks the spec directory tree (or single HRX file) and runs all spec tests.
// specRoot is the top-level spec directory used as a load path for @use resolution.
func RunSpecs(t *testing.T, specPath string, specRoot string) *specStats {
	absPath, err := filepath.Abs(specPath)
	if err != nil {
		t.Fatalf("resolving spec path %s: %v", specPath, err)
	}
	absSpecRoot, err := filepath.Abs(specRoot)
	if err != nil {
		t.Fatalf("resolving spec root %s: %v", specRoot, err)
	}
	fi, err := os.Stat(absPath)
	if err != nil {
		t.Fatalf("spec path does not exist: %s", absPath)
	}

	stats := &specStats{}
	if fi.IsDir() {
		walkSpecDir(t, absPath, absPath, absSpecRoot, nil, stats)
	} else if strings.HasSuffix(absPath, ".hrx") {
		runHRXFile(t, absPath, filepath.Dir(absPath), absSpecRoot, nil, stats)
	} else {
		t.Fatalf("spec path is not a directory or .hrx file: %s", absPath)
	}
	return stats
}

// runHRXFile parses a single HRX file and runs all tests within it.
// root is used to compute relative test names; parentOpts are inherited options.
func runHRXFile(t *testing.T, hrxPath string, root string, specRoot string, parentOpts *SpecOptions, stats *specStats) {
	content, err := os.ReadFile(hrxPath)
	if err != nil {
		t.Errorf("reading HRX %s: %v", hrxPath, err)
		return
	}
	dir := filepath.Dir(hrxPath)
	archiveName := strings.TrimSuffix(filepath.Base(hrxPath), ".hrx")

	// Load options from the parent directory.
	opts, err := loadDirOptions(dir, parentOpts)
	if err != nil {
		t.Errorf("loading options from %s: %v", dir, err)
		opts = parentOpts
	}

	archive, err := ParseHRX(archiveName, string(content))
	if err != nil {
		t.Errorf("parsing HRX %s: %v", hrxPath, err)
		return
	}

	// Merge HRX-level options.
	hrxOpts := opts
	if optsYAML := archive.GetOptionsYAML(); optsYAML != "" {
		child, err := ParseOptions(optsYAML)
		if err != nil {
			t.Errorf("parsing options.yml in %s: %v", hrxPath, err)
		} else {
			hrxOpts = opts.Merge(child)
		}
	}

	hrxRoot := filepath.Join(dir, archiveName)
	for _, testArchive := range archive.ListTestDirs() {
		virtDir := filepath.Join(hrxRoot, testArchive.Path)
		inputFile := testArchive.InputFile()
		absInput := filepath.Join(virtDir, inputFile)

		files := make(map[string]string)
		for p, c := range testArchive.AllFiles() {
			files[filepath.Join(virtDir, p)] = c
		}
		for p, c := range archive.AllFiles() {
			absPath := filepath.Join(hrxRoot, p)
			if _, exists := files[absPath]; !exists {
				files[absPath] = c
			}
		}

		// Include real files from the HRX file's directory so that
		// HRX tests can reference supporting files (e.g. _test-hue.scss).
		entries, err := os.ReadDir(dir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
					continue
				}
				realPath := filepath.Join(dir, entry.Name())
				if _, exists := files[realPath]; !exists {
					content, readErr := os.ReadFile(realPath)
					if readErr == nil {
						files[realPath] = string(content)
					}
				}
			}
		}

		testOpts := hrxOpts
		if optsYAML := testArchive.GetOptionsYAML(); optsYAML != "" {
			child, err := ParseOptions(optsYAML)
			if err != nil {
				t.Errorf("parsing options.yml in %s/%s: %v", archiveName, testArchive.Path, err)
			} else {
				testOpts = hrxOpts.Merge(child)
			}
		}

		runSpecTest(t, root, virtDir, absInput, inputFile, files, specRoot, testOpts, stats, true)
	}
}

// walkSpecDir recursively walks a physical directory and runs tests.
func walkSpecDir(t *testing.T, root string, currentDir string, specRoot string, parentOpts *SpecOptions, stats *specStats) {
	// Load options from this directory.
	dirOpts, err := loadDirOptions(currentDir, parentOpts)
	if err != nil {
		t.Errorf("loading options from %s: %v", currentDir, err)
		return
	}

	entries, err := os.ReadDir(currentDir)
	if err != nil {
		t.Errorf("reading dir %s: %v", currentDir, err)
		return
	}

	// Collect HRX files and subdirectories.
	var hrxFiles []string
	var subdirs []string
	hasInputFile := false

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if entry.IsDir() {
			subdirs = append(subdirs, name)
		} else if strings.HasSuffix(name, ".hrx") {
			hrxFiles = append(hrxFiles, name)
		} else if name == "input.scss" || name == "input.sass" {
			hasInputFile = true
		}
	}

	// Check if this directory itself is a physical test directory.
	if hasInputFile {
		inputFile := "input.scss"
		fullInputPath := filepath.Join(currentDir, "input.scss")
		if _, err := os.Stat(fullInputPath); os.IsNotExist(err) {
			inputFile = "input.sass"
			fullInputPath = filepath.Join(currentDir, "input.sass")
		}
		files := readPhysicalDir(t, currentDir)
		absInput := filepath.Join(currentDir, inputFile)
		runSpecTest(t, root, currentDir, absInput, inputFile, files, specRoot, dirOpts, stats, false)
	}

	// Process HRX files.
	for _, hrxFile := range hrxFiles {
		runHRXFile(t, filepath.Join(currentDir, hrxFile), root, specRoot, dirOpts, stats)
	}

	// Recurse into physical subdirectories.
	for _, subdir := range subdirs {
		walkSpecDir(t, root, filepath.Join(currentDir, subdir), specRoot, dirOpts, stats)
	}
}

// readPhysicalDir reads all files in a physical directory into a map.
func readPhysicalDir(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := make(map[string]string)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return files
	}
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		files[filepath.Join(dir, entry.Name())] = string(content)
	}
	return files
}

// loadDirOptions loads options from a physical directory's options.yml,
// merging with parent options.
func loadDirOptions(dir string, parent *SpecOptions) (*SpecOptions, error) {
	optsPath := filepath.Join(dir, "options.yml")
	content, err := os.ReadFile(optsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return parent, nil
		}
		return nil, err
	}
	child, err := ParseOptions(string(content))
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return child, nil
	}
	return parent.Merge(child), nil
}

// runSpecTest runs a single spec test.
func runSpecTest(t *testing.T, root, testDir, absInput, inputFile string, files map[string]string, specRoot string, opts *SpecOptions, stats *specStats, fromHRX bool) {
	if opts == nil {
		opts = &SpecOptions{}
	}

	// Compute the relative path for test naming.
	relPath, _ := filepath.Rel(root, testDir)

	// Skip based on options. Check against all implementation names in the
	// chain (dart-sass-go first, then dart-sass).
	if opts.IsIgnoredAny(implNames) {
		t.Run(relPath, func(t *testing.T) {
			t.Skip(":ignore_for " + strings.Join(implNames, "/"))
		})
		return
	}
	if opts.IsTodoAny(implNames) {
		t.Run(relPath, func(t *testing.T) {
			t.Skip(":todo " + strings.Join(implNames, "/"))
		})
		if fromHRX {
			stats.todo++
		}
		return
	}

	ok := t.Run(relPath, func(t *testing.T) {
		// Create VirtualIO with all files, falling back to real filesystem
		// for shared utility files (e.g. _utils.scss) that live outside the
		// HRX archive alongside the spec directory tree.
		io := NewVirtualIO(files)
		io.SetFallback(sassio.NewDefaultIO())

		// Create a test logger to capture warnings.
		logger := newTestLogger(new(bytes.Buffer), io)

		// Build compile options.
		// The test directory is not added as a load path — relative
		// imports resolve against the containing file's URL (baseURL
		// derived from the current stylesheet's source URL).
		// FilesystemImporters from load paths would incorrectly resolve
		// relative URLs globally (e.g. @import "sibling" from a
		// subdirectory finding a sibling at the test root).
		// Only the spec root is needed for shared utility files
		// (e.g. _test-hue.scss).
		loadPaths := []string{}
		if specRoot != "" && specRoot != filepath.Dir(absInput) {
			loadPaths = append(loadPaths, specRoot)
		}
		compileOpts := &compile.CompileOptions{
			LoadPaths: loadPaths,
			Logger:    logger,
			Verbose:   true,
			Unicode:   false,
			Charset:   true,
		}

		if opts.Precision > 0 {
			// Precision is handled by the evaluator; note it for future.
		}

		// Run CompileStylesheet.
		err := compile.CompileStylesheet(io, absInput, "", compileOpts)

		output := io.OutputBuffer()
		warnings := logger.buf.String()

		// Determine expected outcome, checking implementation-specific files
		// in order: dart-sass-go first, then dart-sass, then the generic file.
		outputContent, hasOutput := lookupImplFile(files, testDir, "output.css")
		errorContent, hasError := lookupImplFile(files, testDir, "error")

		if hasOutput {
			expected := strings.TrimSpace(outputContent)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			actual := strings.TrimSpace(output)
			if normalizeOutput(actual) != normalizeOutput(expected) {
				t.Errorf("CSS mismatch:\n--- expected:\n%s\n\n--- actual:\n%s", expected, actual)
			}
		}

		if hasError {
			if err == nil {
				t.Errorf("expected error but compilation succeeded")
				return
			}
			expected := strings.TrimSpace(errorContent)
			actual := strings.TrimSpace(err.Error())
			if normalizeError(actual) != normalizeError(expected) {
				t.Errorf("error mismatch:\n--- expected:\n%s\n\n--- actual:\n%s", expected, actual)
			}
		}

		// Check warnings only for success tests, matching sass-spec's
		// behavior: expected.ts only includes warning in the return
		// value when isSuccessCase is true.
		if hasOutput {
			warningContent, hasWarning := lookupImplFile(files, testDir, "warning")
			if hasWarning {
				if opts.IsWarningTodoAny(implNames) {
					// Skip warning comparison but still run the test.
				} else {
					expected := strings.TrimSpace(warningContent)
					actual := strings.TrimSpace(warnings)
					if normalizeOutput(actual) != normalizeOutput(expected) {
						t.Errorf("warning mismatch:\n--- expected:\n%s\n\n--- actual:\n%s", expected, actual)
					}
				}
			}
		}

		if !hasOutput && !hasError {
			t.Errorf("test has no expected output (missing output.css or error file)")
		}
	})
	if fromHRX {
		if ok {
			stats.passed++
		} else {
			stats.failed++
		}
	}
}

// testLogger records warnings and debug messages to a buffer.
// Delegates to StderrLogger with a bytes.Buffer writer to ensure
// the same output format as the real logger.
type testLogger struct {
	buf    *bytes.Buffer
	io     *VirtualIO
	stderr *sasslogger.StderrLogger
}

func newTestLogger(buf *bytes.Buffer, io *VirtualIO) *testLogger {
	stderr := sasslogger.NewStderrLoggerWithWriter(false, false, buf)
	return &testLogger{
		buf:    buf,
		io:     io,
		stderr: stderr,
	}
}

func (l *testLogger) Warn(msg string, span *sasscommon.FileSpan, trace *sasscommon.Trace) {
	l.stderr.Warn(msg, span, trace)
}

func (l *testLogger) Debug(msg string, span *sasscommon.FileSpan) {
	l.stderr.Debug(msg, span)
}

func (l *testLogger) WarnDeprecation(msg string, span *sasscommon.FileSpan, dep *deprecation.Deprecation, trace *sasscommon.Trace) error {
	return l.stderr.WarnDeprecation(msg, span, dep, trace)
}

var inputPathPattern = regexp.MustCompile(`\S*/([-_a-zA-Z0-9.]+\.s[ca]ss)`)

var consecutiveNewlines = regexp.MustCompile(`\n+`)

// normalizeOutput normalizes CSS output for comparison, matching Dart's
// normalizeOutput in lib/test-case/compare.ts.
func normalizeOutput(output string) string {
	output = strings.ReplaceAll(output, "\r\n", "\n")
	// Collapse consecutive newlines into one (matches TS: /(\r?\n)+/g -> '\n').
	output = consecutiveNewlines.ReplaceAllString(output, "\n")
	// Normalize paths: replace any path ending in input.scss/sass with just the basename.
	output = inputPathPattern.ReplaceAllString(output, "$1")
	return strings.TrimSpace(output)
}

// normalizeError normalizes an error message for comparison.
func normalizeError(msg string) string {
	msg = strings.ReplaceAll(msg, "\r\n", "\n")
	msg = inputPathPattern.ReplaceAllString(msg, "$1")
	lines := strings.Split(msg, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "Error:") {
			return strings.TrimSpace(line)
		}
	}
	return strings.TrimSpace(msg)
}

// lookupImplFile looks up an expected file in files, trying implementation-
// specific names in implNames order, then falling back to the base name.
// For example, lookupImplFile(files, testDir, "output.css") will try:
//
//	output-dart-sass-go.css, output-dart-sass.css, output.css
func lookupImplFile(files map[string]string, testDir, baseName string) (string, bool) {
	for _, name := range implNames {
		ext := filepath.Ext(baseName)
		base := strings.TrimSuffix(baseName, ext)
		implFile := filepath.Join(testDir, base+"-"+name+ext)
		if content, ok := files[implFile]; ok {
			return content, true
		}
	}
	content, ok := files[filepath.Join(testDir, baseName)]
	return content, ok
}
