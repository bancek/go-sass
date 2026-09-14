// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package embedded

// dart-source: lib/src/embedded/executable.dart

import (
	"fmt"
	"os"
	"slices"

	"google.golang.org/protobuf/encoding/protojson"
)

// Run starts the embedded compiler serving the Sass protocol over
// stdin/stdout, matching Dart's main entry for sass --embedded.
//
// A --version flag anywhere in args prints the indented JSON version
// response; any other argument is a usage error (exit 64) since the embedded
// binary takes no CLI compilation flags; otherwise an isolate dispatcher
// serves length-delimited packets until stdin closes, with an internal
// failure exiting 70. Dart pattern-matches args and builds the channel
// explicitly; Go branches on slices.Contains/len and lets
// NewIsolateDispatcher own the stdio wiring.
//
// Matches Dart: main in embedded/executable.dart.
func Run(args []string) {
	switch {
	case slices.Contains(args, "--version"):
		// Matches Dart: executable.dart lines 15-21
		fmt.Println(VersionResponseJSON())

	case len(args) > 0:
		// Matches Dart: executable.dart lines 23-32
		fmt.Fprintln(os.Stderr,
			"sass --embedded is not intended to be executed with additional "+
				"arguments.\n"+
				"See https://github.com/sass/dart-sass#embedded-dart-sass for "+
				"details.")
		os.Exit(64)

	default:
		// Matches Dart: executable.dart lines 35-41
		dispatcher := NewIsolateDispatcher(os.Stdin, os.Stdout, os.Stderr)
		if err := dispatcher.Listen(); err != nil {
			fmt.Fprintf(os.Stderr, "Internal compiler error: %v\n", err)
			os.Exit(70)
		}
	}
}

// VersionResponseJSON returns the version response as indented JSON for the
// --embedded --version case.
//
// The response carries ID 0 with the protocol, compiler, and implementation
// versions plus the "dart-sass" implementation name the host requires,
// mirroring Dart's versionResponse with the ID stamped after construction.
//
// Matches Dart: versionResponse plus JsonEncoder.withIndent("  ") in
// embedded/executable.dart.
func VersionResponseJSON() string {
	response := &OutboundMessage{
		Message: &OutboundMessage_VersionResponse_{
			VersionResponse: &OutboundMessage_VersionResponse{
				Id:                    0,
				ProtocolVersion:       protocolVersion,
				CompilerVersion:       compilerVersion,
				ImplementationVersion: compilerVersion,
				ImplementationName:    "dart-sass",
			},
		},
	}
	marshaler := protojson.MarshalOptions{Indent: "  "}
	jsonBytes, _ := marshaler.Marshal(response)
	return string(jsonBytes)
}
