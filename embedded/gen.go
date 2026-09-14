// Copyright 2024 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package embedded

// To regenerate embedded_sass.pb.go after protocol changes:
//   1. Make sure protoc 3.15+ is installed (the proto uses optional fields)
//   2. Install protoc-gen-go: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
//   3. Check out the sass language-spec submodule: git submodule update --init sass
//   4. Run: go generate ./embedded/
//
// The bindings generate from the canonical ../sass/spec/embedded_sass.proto
// (never a vendored copy). paths=source_relative keeps the output in this
// directory; the M flag supplies the import path (the upstream proto carries
// no go_package option, and the submodule must stay unedited).
//go:generate protoc -I../sass/spec --go_out=. --go_opt=paths=source_relative --go_opt=Membedded_sass.proto=github.com/bancek/go-sass/embedded embedded_sass.proto
