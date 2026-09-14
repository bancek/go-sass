# AGENTS.md — read this first, then read the docs it points to

This codebase is large (~150k lines of Go in one module) and tightly
constrained: it must produce **byte-identical output to Dart Sass**, so most
"obvious" simplifications are bugs. Do **not** dive into code head-first.
Follow the reading map below before touching anything.

## Mandatory starting reads (in order)

1. **`docs/CONTRIBUTING.md`** — how to build, test, and verify. Non-negotiable.
2. **`docs/architecture.md`** — the pipeline and why the design looks the way
   it does (GC references, sync-only evaluator, interface dispatch, the
   `value/` monolith).
3. **`docs/ref/pipeline.md`** — end-to-end walkthrough of one compilation:
   entries, stages, ownership.

## Before ANY code change, also read

- **`docs/critical-invariants.md`** — rules that must never be violated
  (`Equal()`, never `==`; no panics in compiler code; no silent error
  discarding; exact Dart fidelity). Violating one means revert.
- **`docs/patterns.md`** — translation conventions (Dart → Go) and the
  `// dart-source:` annotation system.
- **The relevant `docs/ref/` page** for the module you are touching
  (index at `docs/ref/README.md`) — it documents the exact struct shapes,
  invariants, and gotchas. Doc snippets drift; verify signatures against
  code with `rg` before copying them.

## Rules for every change

- **Tests are mandatory.** Every fix needs a regression test that fails
  without it, asserting exact type/message/span (never bare `err != nil`);
  golden values are hardcoded Dart output, never recomputed from the code
  under test. See `CONTRIBUTING.md` Testing and `critical-invariants.md`
  Test standards.
- **Gates must be green.** `go build ./... && go vet ./...` passes and
  `gofmt -l .` prints nothing.
- **Iterate locally, verify globally.** Write fast unit tests first and
  iterate on those; at the end of every change the full spec suite must
  also pass:
  `SASS_SPEC_PATH=../sass-spec/spec go test -tags=spec -timeout=30s ./spec/`
  (use `SASS_SPEC=<subpath>` for a focused subset during development —
  it must be a real path under `sass-spec/spec/`).
- **Match Dart's observable behavior exactly** — messages, spans, traces,
  ordering, scope effects (see `docs/review.md` for the checklist). Never
  "improve" Dart; record deliberate divergences in `docs/divergences.md`
  using its entry template.
- **Keep annotations honest.** Every ported file carries `// dart-source:` —
  update them when code moves; never leave a stale path.
- **Verify per change, not at the end.** Repro first (differential A/B vs
  `dart-sass/`), fix narrowly, run the gates in `CONTRIBUTING.md`.

## Task-specific entry points

| Task                         | Start with                                                                           |
| ---------------------------- | ------------------------------------------------------------------------------------ |
| Porting a dart-sass change   | `docs/porting.md` + `docs/upstream.md`                                               |
| Evaluator/importer/functions | `docs/ref/eval.md`, `docs/ref/importer.md`, `docs/ref/environment.md`                |
| Parser/serializer/values     | `docs/ref/parse.md`, `docs/ref/serialize.md`, `docs/ref/value.md`, `docs/ref/ast.md` |
| Visitors/dispatch            | `docs/ref/visitors.md`                                                               |
| Float/math parity            | `docs/ref/math.md`                                                                   |
| Embedded protocol / host API | `docs/ref/embedded.md` / `docs/ref/sass-embedded-go.md`                              |
