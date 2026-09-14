# The `sass-spec` suite: layout, `.hrx` format, and debugging failures

The definitive correctness gate: thousands of small compilations with expected
outputs. There are **two ways to run it** (next section) — the native Go
runner is the per-change gate; the upstream harness cross-checks the built
binary. This page explains both, the spec layout, the `.hrx` archive format,
`options.yml`, how the runner compares, and the debug loop for a failing
test. For commands, see `CONTRIBUTING.md` Testing.

## Ways to run (and when)

| #   | Runner                                 | Command                                                                                                                                 | What it exercises                                                   | Run when                                                                                    |
| --- | -------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| 1   | Native Go (`spec/`, `//go:build spec`) | `SASS_SPEC_PATH=../sass-spec/spec go test -tags=spec -timeout=30s ./spec/` (repo root; `SASS_SPEC=<subpath>` for a subset)              | the compiler in-process (`VirtualIO`, spec root on the load path)   | after **every** core change — the per-change gate                                           |
| 2   | Upstream harness → built binary        | `npm run sass-spec -- --command <path-to>/go-sass --impl dart-sass [subpath]` (from `sass-spec/`; needs `go build ./cmd/go-sass` first) | the release CLI end to end, incl. arg handling and real process I/O | cross-check after larger changes, or when a change touches CLI behavior, I/O, or exit codes |

Runner 2 goes through the upstream `sass-spec` harness (`--command` +
`--impl` + optional positional subpath filter); runner 1 is ours.

`--impl dart-sass` matters: it selects the expectation files and the
`:todo:`/`:ignore_for:` matching used for Dart Sass. A trailing subpath
(e.g. `spec/core_functions/color/to_space`) scopes the run like
`SASS_SPEC=` does for the native runner.

Cleanup note: sass-spec tooling can materialize stray untracked dirs inside
`spec/`; remove them before full runs so they don't pollute results.

## Layout

`sass-spec/spec/` is a tree of _test directories_ and `.hrx` files:

- A **test directory** is any directory containing `input.scss` (or
  `input.sass`). Siblings provide expectations: `output.css` for success,
  `error` for failure, `warning` for expected warnings. Extra files
  (`_partial.scss`, subdirectories) are importable fixtures.
- An **`.hrx` file** packs several tests (or fixtures) into one file; each
  archived test dir inside it works exactly like a physical test directory.
- `options.yml` files at any level adjust behavior for everything below
  (inherited down, parent + child merged — see below).

Top-level groups (use as `SASS_SPEC=<path>`, which appends to
`sass-spec/spec/`):

| Path                                 | Contents                                                                                                                        |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------- |
| `core_functions/`                    | built-in modules (`color/`, `math/`, `map/`, `list`, …)                                                                         |
| `css/`                               | plain-CSS parsing, `@media`, `@supports`, comments, `plain/`                                                                    |
| `directives/`                        | `@use`, `@forward`, `@import`, `@extend`, `@media`, `@at-root`, … (single-case `.hrx` files like `each.hrx` live here directly) |
| `expressions/`, `operators/`         | expression semantics, arithmetic                                                                                                |
| `values/`                            | `calculation/`, `colors/`, `lists/`, `maps/`, `numbers/`                                                                        |
| `variables/`, `callable/`, `parser/` | scoping, callables, syntax edge cases                                                                                           |
| `libsass*`, `non_conformant/`        | legacy / known-divergent (mostly `ignore_for` / `todo`)                                                                         |

## The `.hrx` format

A multi-file archive: sections delimited by `<===> path`, one file per
section (parsed by `spec/hrx.go`):

```
<===> leading/input.scss
a {b: c}
> d {@extend a}

<===> leading/output.css
a, > d {
  b: c;
}

<===> leading/warning
DEPRECATION WARNING [bogus-combinators]: ...
```

Rules:

- `<===>` + whitespace + relative path starts a section; a trailing single
  newline of each section is stripped.
- A bare `<===>` line is a comment/separator and carries no file.
- Nested paths (`leading/input.scss`) become nested test dirs; each dir with
  an `input.scss`/`input.sass` is an independent test.
- An `options.yml` section inside the archive applies to that archive, and a
  per-test-dir `options.yml` section applies to just that test.
- Sibling files on disk next to the `.hrx` are visible to its tests (they do
  not override same-named archived files).

## `options.yml`

```yaml
:todo:
  - dart-sass
:warning_todo:
  - dart-sass
:ignore_for:
  - libsass
:precision: 10
```

| Key              | Meaning for our runner (`implNames = ["dart-sass-go", "dart-sass"]`) |
| ---------------- | -------------------------------------------------------------------- |
| `:todo:`         | test counts as **skipped** (known failure)                           |
| `:warning_todo:` | CSS is compared but the `warning` file is **not**                    |
| `:ignore_for:`   | test is **skipped** entirely                                         |
| `:precision:`    | noted for future evaluator handling                                  |

Matching checks the whole impl chain (`dart-sass-go` first, then
`dart-sass`). Options merge down the directory tree (and archive → test dir
inside `.hrx`).

## How our runner compares (`spec/runner.go`)

1. **Isolation** — each test compiles in a `VirtualIO` preloaded with the
   test dir's files (real-filesystem fallback for shared fixtures outside
   the archive), with only the spec root on the load path — the test
   directory itself is deliberately **not** a load path, so relative imports
   resolve against the containing file. Logger is a capturing `testLogger`
   (a `StderrLogger` over a buffer, same format as the real logger);
   options are `verbose: true, unicode: false, charset: true`.
2. **Expectation lookup** — impl-specific overrides first:
   `output-dart-sass-go.css`, then `output-dart-sass.css`, then
   `output.css` (same scheme for `error`/`warning`).
3. **Normalization** (`normalizeOutput`, mirroring Dart's compare helper):
   `\r\n` → `\n`, consecutive newlines collapsed, input paths replaced with
   basenames, surrounding whitespace trimmed. Both sides go through it.
4. **CSS path** — compile must succeed (`CompileStylesheet(io, absInput,
"", opts)`); normalized actual vs expected must match byte-for-byte, else
   `CSS mismatch:` with both texts. Then, if a `warning` file exists (and no
   `:warning_todo:`), normalized captured warnings must match — warnings are
   only compared for success tests, matching upstream behavior.
5. **Error path** — compile must fail; compared text is the error message via
   `normalizeError`. Mismatch → `error mismatch:`; success → `expected error
but compilation succeeded`.
6. **Neither file** — `test has no expected output` failure (the test itself
   is malformed).

## Debugging a failure

### Step 1: identify the failure type

| Runner output                              | Meaning                              |
| ------------------------------------------ | ------------------------------------ |
| `unexpected error: …`                      | threw an error; test expected CSS    |
| `CSS mismatch:`                            | both produced CSS but differ         |
| `warning mismatch:`                        | CSS matches, warnings differ         |
| `error mismatch:`                          | both errored but the text differs    |
| `expected error but compilation succeeded` | threw nothing; test expected `error` |

### Step 2: reproduce in isolation

Run the single test, then shrink to stdin on both compilers:

```sh
SASS_SPEC=<path/to/test-dir-or.hrx> go test -tags=spec -timeout=30s ./spec/
echo '<minimal scss>' | go run ./cmd/go-sass --stdin 2>&1
echo '<minimal scss>' | dart run bin/sass.dart --stdin 2>&1   # from dart-sass/
```

The CLI reads the same code path as the runner except for the harness
(`VirtualIO`, load paths, `unicode: false`, `verbose`); if stdin and spec
disagree, suspect `cwd`-relative paths, load paths, or warning capture —
not the compiler.

### Step 3: find the code

| Failure smells like…                   | Look in                                                                                   |
| -------------------------------------- | ----------------------------------------------------------------------------------------- |
| wrong error text/span, missing trace   | `eval/evaluate_helpers.go` (exception, `addExceptionSpan`)                                |
| wrong CSS for a construct              | `eval/evaluate_statement.go` / `evaluate_expression.go` / `evaluate_css.go` per construct |
| warning text/span/dedup                | `eval/`, `sasslogger/`                                                                    |
| `@use`/`@forward`/`@import` resolution | `eval/import_cache.go`, `eval/importer*.go`, `docs/ref/importer.md`                       |
| built-in function behavior             | `functions/`, `docs/ref/functions.md`                                                     |
| parse error or shape                   | `value/parse_*.go`, `docs/ref/parse.md`                                                   |
| number/color output                    | `value/`, `docs/ref/value.md`                                                             |
| selector/extend output                 | `value/selector_*.go`, `extend/`, `docs/ref/selector.md`, `docs/ref/extend.md`            |

Every Go file carries a `// dart-source:` annotation — open the Dart file
and compare. For the full port-review procedure, see `docs/review.md`.

### Step 4: fix and verify

Fix narrowly against Dart's behavior, add a unit regression test
(`CONTRIBUTING.md` Testing), then re-run the single spec test, the group,
and — at the end of the change — the full suite. A spec failure that
requires changing an expectation file (`output.css`/`error`/`warning`
upstream) is almost certainly a wrong fix: expectations are Dart's output.

## `:todo:` vs `:ignore_for:` vs `warning_todo`

- `:todo:` containing our impl name → skipped. Used for known open bugs.
  Fixing the bug means the test starts running — no bookkeeping needed.
- `:ignore_for:` → skipped before counting. Used for out-of-scope
  implementations (mostly `libsass`).
- `:warning_todo:` → CSS compared, warnings not. Used while warning text
  lags behind correct output.

## File mapping

| Concern                     | Code                                                                    |
| --------------------------- | ----------------------------------------------------------------------- |
| test walk + comparison      | `spec/runner.go`                                                        |
| `.hrx` parsing              | `spec/hrx.go`                                                           |
| `options.yml`               | `spec/options.go`                                                       |
| test entry                  | `spec/runner_test.go` (`SASS_SPEC`/`SASS_SPEC_PATH`, `//go:build spec`) |
| virtual filesystem + logger | `spec/virtual_io.go`, `testLogger` in `runner.go`                       |
| fixtures                    | `sass-spec/spec/` (submodule)                                           |
