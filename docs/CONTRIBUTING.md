# Contributing

How to build, test, and maintain `go-sass`. This is a maintenance and
contribution guide; the procedure for porting upstream dart-sass changes lives
in [`porting.md`](porting.md).

## Prerequisites

- Go 1.26.3 (the `new("lit")` pointer shorthand and `iter.Seq` iterators are
  used throughout; check with `go version`).
- Dart SDK (for the differential research loop: piping the same SCSS through
  `dart run bin/sass.dart --stdin` from `dart-sass/` vs the Go CLI).
  Constraint from [`upstream.md`](upstream.md): Dart `>=3.13.0 <4.0.0`;
  check with `dart --version`.
- `protoc` plus `protoc-gen-go` (to regenerate the embedded-protocol Go
  bindings — see below). The generator is a tool dependency of the module
  itself (`google.golang.org/protobuf/cmd/protoc-gen-go` in `go.mod`):
  ```sh
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  ```
  Ensure `$(go env GOPATH)/bin` is on `PATH`.
- Submodules: clone with `git clone --recurse-submodules` (or run
  `git submodule update --init` afterwards). Present: `dart-sass`
  (porting reference + differential loop), `sass-spec` (spec suite),
  `sass` (language spec: the `.proto` source + host-build vendoring),
  `bootstrap-main` (benchmark workload),
  `embedded-host-node` (host source for the `sass-embedded-go` build). See
  [`upstream.md`](upstream.md) for pins and what each feeds.

## Building

```sh
go build ./...                  # whole module (stdlib math)
go build ./cmd/go-sass          # the production `go-sass` binary
go build -tags cgomath ./cmd/go-sass   # with the C-libm math backend (exact Dart matching)
```

CI, the release matrix, and `npm run build` always pass `-tags cgomath` —
dev tests what we release. A plain `go build` (no tags) is hermetic
stdlib math for local iteration only; on libm-sensitive inputs it may
differ from Dart in the last ULP (see `ref/math.md`).

The CLI binary reads `input.scss [output.css]`, `input.scss:output.css`,
multiple pairs, or directories; `-` (or `--stdin`) reads stdin. Exit codes:
`64` usage error (including `--help`, matching Dart), `65` Sass error, `66`
filesystem error. See [`../cmd/go-sass/README.md`](../cmd/go-sass/README.md).

## Testing

Every code change must add or update tests covering it: a bug fix needs a
regression test that fails without the fix, a behavior change needs tests
locking the new behavior. Tests assert **exact** type/message/span (never bare
`err != nil`); golden values are hardcoded Dart output, never recomputed from
the code under test (see `patterns.md` §8 and `critical-invariants.md` Test
standards).

```sh
go build ./... && go vet ./...
gofmt -l .                 # must print nothing (see below)

go test ./...              # unit tests (C-libm backend wired automatically)
```

Unit-test binaries link C libm unconditionally (via `math_cgo_test.go` in
`value/`, `util/`, `functions/` and `spec/math_cgo.go`), so the suite always
asserts exact Dart parity — no build tags needed.

### The official sass-spec suite (definitive correctness gate)

```sh
SASS_SPEC_PATH=../sass-spec/spec go test -tags=spec -timeout=30s ./spec/
```

The runner requires the `sass-spec` submodule checkout (`git clone
--recurse-submodules`) and the `spec` build tag. To scope a run, set
`SASS_SPEC` to a **relative path under `sass-spec/spec/`** (appended verbatim —
use a real directory or `.hrx` file, e.g. `directives/use`, `core_functions/color`;
a nonexistent path runs 0 tests):

```sh
SASS_SPEC=directives/use go test -tags=spec -timeout=30s ./spec/
```

`SASS_SPEC_PATH` is different: an absolute path used _instead of_ the default
`../sass-spec/spec` root. How tests are laid out, what `.hrx` files and
`options.yml` mean, how the runner compares output, and the debug loop for a
failure: see [`ref/sass-spec.md`](ref/sass-spec.md).

### Embedded protocol

```sh
go test ./embedded/   # protocol server tests (included in go test ./...)
```

The round-trip harness is the `sass-embedded-go` package
(`embedded-host-node-go/`): `npm install`, `npm run build` (assembles the
host against the local binary — see `docs/ref/sass-embedded-go.md`), then
`npm test` (resolve gate + protocol harness + coexistence with genuine
`sass-embedded`). Re-run it after any change to `embedded/`, `compile/`,
or the CLI. Never point a host at a binary by patching `compilerCommand` —
the old monkey-patch approach is retired (process-global, breaks
co-imported `sass-embedded`).

### Byte-identity

After any core change, verify byte-identical output vs Dart Sass on the
bootstrap (`bootstrap-main/scss/bootstrap.scss`), `huge`
(`bench/huge.scss`), and `huge10` workloads (CSS, source maps, warnings, and
errors). Compare the CLI output directly:

```sh
echo '<scss>' | dart run bin/sass.dart --stdin   # from dart-sass/
echo '<scss>' | ./sass --stdin
```

### Benchmarking

Micro-benchmarks live next to the code as `*_bench_test.go` (e.g.
`value/map_bench_test.go`) plus the workload suite in `bench/`
(`BenchmarkBootstrap`, `BenchmarkHuge`, `BenchmarkHuge10`):

```sh
go test -bench=. -benchmem ./bench/
```

Workloads: `bootstrap-main/scss/bootstrap.scss` (submodule) and the tracked
`bench/huge.scss` (`huge10` is a 10× in-memory concatenation built at bench
time). For workload timing, compare **user CPU** (not wall — background
threads inflate wall time): warm up once, interleave A/B runs, report the
median of several runs. A missing `bootstrap-main` checkout skips that
benchmark instead of failing — a skip is not a pass.

### Formatting and vet

`gofmt` must be clean — `gofmt -l .` prints nothing — and `go vet ./...` must
pass. Run both before every commit.

## Protocol codegen

The embedded-protocol bindings (`embedded/embedded_sass.pb.go`) are committed
generated code. Regenerate them only when the `.proto` changes:

```sh
go generate ./embedded/
```

This regenerates from the canonical `sass/spec/embedded_sass.proto`
(the `sass` language-spec submodule — never a vendored copy) via
`protoc-gen-go`. Prerequisites: `protoc` 3.15+, `protoc-gen-go`, and the
submodule checkout (`git submodule update --init sass`). See
`embedded/gen.go` for the exact directive.

## Version-bump checklist

The compiler version tracks the pinned dart-sass version; the protocol
version tracks the embedded protocol (it moves only on `PROTOCOL_VERSION`
changes, never on a dart-sass bump). Two entry points: patch releases and
`-alpha` rehearsals run `tools/bump-version.sh` directly (no behavior
change involved); anything bigger arrives via the [`porting.md`](porting.md)
re-sync, whose step 7 ends at this checklist. The mechanical edits are
scripted — the judgment calls are not:

1. Run `tools/bump-version.sh <version>` (`1.104.1` or `1.105.0-alpha0`):
   FULL version to `embedded-host-node-go/package.dist.json` (the
   published identity: `version` + 8 `optionalDependencies` pins), BASE to
   `sassVersion` (`cmd/go-sass/options.go`), `compilerVersion`
   (`embedded/isolate_dispatcher.go`), and the `sass-embedded` devDep.
   Triage its sweep report: our docs and code move to the new version,
   historical changelog sections and real past-release markers stay,
   submodule contents are upstream's (verify, don't hand-edit).
   Generated files (`embedded-host-node-go/platform/*/package.json`,
   `dist/`) are rebuilt, never hand-edited.
2. (Re-sync only — skip on a direct patch/`-alpha` bump.) Update
   [`../PORTED_FROM`](../PORTED_FROM) and [`upstream.md`](upstream.md)
   (commit, version, any external-package pins) — see [`porting.md`](porting.md).
   The `dart-sass/` submodule pin moves to the same commit (both must agree).
3. (Re-sync only — skip on a direct patch/`-alpha` bump.) Bump the
   `embedded-host-node` submodule pin to the matching host tag,
   rebuild `embedded-host-node-go` (new local platform binary), and re-run
   its harness + coexistence + tarball validation.
4. Re-sync the lockfile (`npm install --package-lock-only` in
   `embedded-host-node-go/`; `npm ci` fails when the lockfile drifts).
   On a prerelease the re-sync fails if upstream has not published the
   matching `sass-embedded` yet — hold the devDep at the last published
   upstream instead (see `release.md`).
5. Add a `CHANGELOG.md` entry and re-run the full battery above plus the
   embedded round-trip and byte-identity checks.
6. Release: commit, push, wait for CI green, then push tag `v<version>` —
   the `release.yml` build matrix compiles per-OS cgo binaries and
   `assemble` publishes wrapper + platforms to npm (`next` on `-`-suffixed
   prereleases, `latest` on finals), attaches archives with Sigstore
   attestation to a draft release; see [`release.md`](release.md).

## Synchronizing with dart-sass

See [`porting.md`](porting.md). The short version: fetch upstream, group new
commits into functional units, map changed Dart files to Go files via the
`// dart-source:` annotations, port each unit, then re-run the full battery.

## Tooling

### Searching the port

`rg` with the annotation index is the primary navigation tool:

```sh
# which Go file ports a Dart file?
rg -l "dart-source:.*evaluate.dart" --glob '*.go' . | grep -v _test

# where is a behavior implemented?
rg -n 'func (v \*EvaluateVisitor) visit' eval/
```

### Debugging a last-digit mismatch

Build standalone Go and Dart reproducers that inline the conversion pipeline
with hardcoded inputs, printing every intermediate value in full-precision
hex; compare to find the exact divergence point. See `ref/math.md` for the
worked methodology.
