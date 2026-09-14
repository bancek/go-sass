# go-sass

`go-sass` is a Go port of [Dart Sass](https://sass-lang.com/dart-sass), the
reference implementation of the Sass CSS preprocessor. It produces
**byte-identical output** to Dart Sass `1.104.0` — the same CSS, source maps,
error messages, and warnings — as a self-contained Go module, and a drop-in
replacement for the `sass` command line and the `sass-embedded` protocol server
(spawned with `--embedded`).

There is also a matching [Rust port](https://github.com/bancek/rust-sass).

The compiler is complete and passes the full spec suite against Dart Sass
`1.104.0` (14246/14252 passing specs).

## Packages

Entry points are [`compile/`](compile/) (public compile API),
[`cmd/go-sass/`](cmd/go-sass/) (the `go-sass` command-line binary), and
[`embedded/`](embedded/) (the `sass-embedded` protocol server).

The full package layout and the reason `value/` is monolithic live in
[`docs/architecture.md §13`](docs/architecture.md#13-package-layout-and-import-cycles).

## Usage

### Library

```go
package main

import (
	"fmt"

	"github.com/bancek/go-sass/compile"
	"github.com/bancek/go-sass/sassio"
)

func main() {
	io := sassio.NewDefaultIO()
	opts := &compile.CompileOptions{Unicode: true, Charset: true}
	result, err := compile.CompileString("$color: red; .foo { color: $color; }", io, opts)
	if err != nil {
		panic(err)
	}
	fmt.Println(result.CSS())
}
```

A custom `sassio.IO` implementation can be provided for customization.

### Command line

```sh
go install github.com/bancek/go-sass/cmd/go-sass@latest
go-sass input.scss
```

Or run from a checkout:

```sh
go run ./cmd/go-sass input.scss
```

Pass `--help` for the full flag reference. The CLI mirrors the `sass` CLI
(`--style`, `--load-path`, `--source-map`, `--fatal-deprecation`, …). See
[`cmd/go-sass/README.md`](cmd/go-sass/README.md).

### Embedded protocol

```sh
go-sass --embedded
```

Serves the Sass embedded protocol (version 3.2.0) over stdio, for use by the
`sass-embedded` host package. See [`docs/ref/embedded.md`](docs/ref/embedded.md).
The published host API is [`sass-embedded-go`](embedded-host-node-go/) (npm),
assembled from the pinned `embedded-host-node` submodule — see
[`docs/ref/sass-embedded-go.md`](docs/ref/sass-embedded-go.md).

## Building and testing

Prerequisites: Go 1.26, submodules
(`git clone --recurse-submodules`).

```sh
go build ./...
go vet ./...
# unit tests
go test ./...
# full sass-spec suite (takes ~10 seconds, ~14k tests)
go test -tags=spec -v ./spec
```

## Upstream tracking

This port mirrors a specific dart-sass commit:

- [`PORTED_FROM`](PORTED_FROM) — the machine-readable tracked commit.
- [`docs/upstream.md`](docs/upstream.md) — the human-readable picture (commit,
  version, external-package pins).
- [`docs/porting.md`](docs/porting.md) — the procedure for porting new
  dart-sass changes.

Every ported source file carries a `// dart-source: lib/src/...` annotation
naming the Dart file it ports. This is the reverse index used when back-porting
upstream changes.

## Documentation

| Document                                                     | Contents                                             |
| ------------------------------------------------------------ | ---------------------------------------------------- |
| [`docs/architecture.md`](docs/architecture.md)               | The pipeline and the design decisions behind it.     |
| [`docs/ref/pipeline.md`](docs/ref/pipeline.md)               | End-to-end walkthrough: entries, stages, ownership.  |
| [`docs/patterns.md`](docs/patterns.md)                       | Translation conventions and the annotation system.   |
| [`docs/critical-invariants.md`](docs/critical-invariants.md) | Rules that must not be violated.                     |
| [`docs/ref/`](docs/ref/README.md)                            | Per-module reference (evaluator, parser, values, …). |

Maintainer references:

| Document                                       | Contents                                                             |
| ---------------------------------------------- | -------------------------------------------------------------------- |
| [`docs/divergences.md`](docs/divergences.md)   | Intentional implementation differences (identical output); not bugs. |
| [`docs/porting.md`](docs/porting.md)           | How to port new dart-sass changes.                                   |
| [`docs/upstream.md`](docs/upstream.md)         | Tracked commit and external-package pins.                            |
| [`docs/CONTRIBUTING.md`](docs/CONTRIBUTING.md) | Build, test, and contribution guide.                                 |
| [`docs/ci.md`](docs/ci.md)                     | CI workflows, release tags, and roadmap.                             |
| [`docs/review.md`](docs/review.md)             | Go↔Dart review guide (how to check a port for bugs).                 |

## License

MIT. Copyright (c) 2026 Luka Zakrajšek <luka@bancek.net>.

Ported files carry the Dart source's own license header (Google MIT, or the BSD
"Dart project authors" header for files derived from third-party packages) plus
a `Ported and rearchitected for Go by Luka Zakrajsek.` attribution line. See
[`docs/upstream.md`](docs/upstream.md) for the external-package inventory.
