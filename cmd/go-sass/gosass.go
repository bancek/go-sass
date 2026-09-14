package main

// This is the Go CLI entrypoint tool. It mirrors Dart Sass's `sass` executable
// (bin/sass.dart + lib/src/executable/).
//
// Matches Dart: bin/sass.dart

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/bancek/go-sass/compile"
	"github.com/bancek/go-sass/embedded"
	"github.com/bancek/go-sass/sassio"
)

func main() {
	// Matches Dart: bin/sass.dart lines 26-29
	// if (args case ['--embedded', ...var rest]) { embedded.main(rest); return; }
	if len(os.Args) >= 2 && os.Args[1] == "--embedded" {
		embedded.Run(os.Args[2:])
		return
	}

	sassIO := sassio.NewDefaultIO()

	opts, ue := parseOptions(sassIO, os.Args[1:])
	if ue != nil {
		// Matches Dart: UsageException handler (bin/sass.dart:69-75)
		sassIO.PrintOutput(ue.message + "\n\n" + usageText(sassIO) + "\n")
		os.Exit(64)
	}

	if opts.version {
		sassIO.PrintOutput(sassVersion + "\n")
		os.Exit(0)
	}

	stopProfiles := startProfiling(sassIO, opts)
	exitCode := run(sassIO, opts)
	stopProfiles()
	os.Exit(exitCode)
}

// run compiles every source, returning the maximum exit code.
func run(sassIO sassio.IO, opts *cliOptions) int {
	exitCode := 0
	printedError := false
	for _, sd := range opts.sources {
		err := compile.CompileStylesheet(sassIO, sd.source, sd.dest, opts.compileOptions())
		if err == nil {
			continue
		}

		code, message := errorExit(err)
		if code > exitCode {
			exitCode = code
		}
		prefix := ""
		if printedError {
			prefix = "\n"
		}
		sassIO.PrintError(prefix + message)
		printedError = true

		if opts.stopOnError {
			break
		}
	}
	return exitCode
}

// errorExit maps an error to its exit code and printed message, mirroring
// Dart's compileStylesheet error handling.
func errorExit(err error) (int, string) {
	if se, ok := errors.AsType[*compile.StylesheetError](err); ok {
		return se.ExitCode, se.Message
	}
	if fse, ok := errors.AsType[*sassio.FileSystemException](err); ok {
		message := fse.Message
		if fse.Path != "" {
			message = fmt.Sprintf("Error reading %s: %s.", fse.Path, fse.Message)
		}
		return 66, message
	}
	return 255, "Unexpected exception:\n" + err.Error()
}

// startProfiling starts the Go CPU/heap profilers if requested, returning a
// function that flushes them.
func startProfiling(sassIO sassio.IO, opts *cliOptions) func() {
	var stops []func()
	if opts.cpuProfile != "" {
		f, err := os.Create(opts.cpuProfile)
		if err != nil {
			sassIO.PrintError(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}
		_ = pprof.StartCPUProfile(f)
		stops = append(stops, func() {
			pprof.StopCPUProfile()
			_ = f.Close()
		})
	}
	if opts.memProfile != "" {
		stops = append(stops, func() {
			f, err := os.Create(opts.memProfile)
			if err != nil {
				sassIO.PrintError(fmt.Sprintf("Error: %v", err))
				return
			}
			runtime.GC()
			_ = pprof.WriteHeapProfile(f)
			_ = f.Close()
		})
	}
	return func() {
		for _, stop := range stops {
			stop()
		}
	}
}
