# Go ↔ Dart review guide

Detailed per-file review of the Go port against Dart Sass. One file,
Go ↔ Dart only — the Rust port is a separate implementation, not a
porting path (see `porting.md`).

## 1. Goal and non-goals

**Goal:** find porting bugs where Go diverges from Dart in observable
behavior: CSS output, source maps, error messages, spans, traces,
warnings, import resolution, and evaluation order/scope effects.

**Non-goals:**

- No literal structural parity. Go's package system forces more
  restructuring than Dart needs (see §3). Review _boundary behavior_,
  not file counts.
- No cross-port check. Read Dart in `dart-sass/` directly (`porting.md`).
  The Rust port is provenance only.
- No style review. If behavior matches, structure is fine even when it
  looks nothing like Dart.

**Scope:** the whole module vs `dart-sass/lib/src`. Outlying surfaces map
elsewhere: `embedded/` ↔ `lib/src/embedded/*` +
`embedded_sass.proto`, `cmd/go-sass/` ↔ `lib/src/executable/*`. Review those
only when the finding lives at the seam (e.g. error → proto mapping).

Pin under review: `PORTED_FROM` (`e01e268c6f6826ae309bf3105765d4c93024ebbc`,
v1.104.0 per `upstream.md`) must agree with the `dart-sass/` submodule
pin. Record both hashes in every finding log (§7).

## 2. File map (Dart → Go)

Built from `// dart-source:` annotations. Read the Dart file in
`dart-sass/lib/src/...`, find Go via:

```sh
rg -l "dart-source:.*$(basename <dart-file>)" --glob '*.go' . | grep -v _test
```

Match on the **full `lib/src/...` path**, not the basename. Groups in
risk-first review order (§5). The evaluator group is first because most
porting bugs live there.

| #   | Dart                                                                                                                                                                                                                                       | Go                                                                                                              |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------- |
| E1  | `lib/src/visitor/evaluate.dart` (EvaluateVisitor, ~all visit methods; async twin ignored — Go is sync-only)                                                                                                                                | `eval/evaluate*.go` + `evalcontext/`                                                                            |
| E2  | `lib/src/environment.dart`                                                                                                                                                                                                                 | `sassenv/environment.go`                                                                                        |
| E3  | `lib/src/importer.dart`, `lib/src/importer/*.dart`, `lib/src/import_cache.dart`                                                                                                                                                            | `eval/import_cache.go`, `eval/importer*.go`, `eval/resolve_import_path.go`                                      |
| E4  | `lib/src/functions/*.dart`, `lib/src/functions.dart`                                                                                                                                                                                       | `functions/*.go` (registries) + `eval/` (evaluator meta)                                                        |
| E5  | `lib/src/compile.dart`, `lib/src/compile_result.dart`                                                                                                                                                                                      | `compile/*.go`                                                                                                  |
| E6  | `lib/src/callable*.dart`, `lib/src/callable/*.dart`, `lib/src/evaluation_context.dart`, `lib/src/configuration.dart`, `lib/src/configured_value.dart`, `lib/src/module*.dart`, `lib/src/module/*.dart`                                     | `sasscallable/`, `evalcontext/`, `configuration/`, `sassmodule/`                                                |
| A1  | `lib/src/ast/sass/*.dart`, `lib/src/ast/sass/**/*.dart`                                                                                                                                                                                    | `value/sass_*.go`                                                                                               |
| A2  | `lib/src/ast/css/*.dart`, `lib/src/ast/css/modifiable/*.dart`, `lib/src/visitor/clone_css.dart`, `lib/src/visitor/*css*.dart`                                                                                                              | `value/css_*.go`                                                                                                |
| A3  | `lib/src/value*.dart`, `lib/src/value/*.dart`                                                                                                                                                                                              | `value/value*.go`, `value/color*.go`, `value/number*.go`, `value/list.go`, `value/map.go`, …                    |
| P1  | `lib/src/parse/*.dart`                                                                                                                                                                                                                     | `value/parse_*.go`                                                                                              |
| S1  | `lib/src/visitor/serialize.dart`                                                                                                                                                                                                           | `value/visitor_serialize.go`                                                                                    |
| S2  | `lib/src/ast/selector/*.dart`, `lib/src/extend/*.dart`, `lib/src/visitor/*selector*.dart`, `lib/src/visitor/replace_expression.dart`, `lib/src/visitor/*plain*`, `lib/src/visitor/*calculation*`, `lib/src/visitor/find_dependencies.dart` | `value/selector_*.go`, `extend/`                                                                                |
| C1  | `lib/src/exception.dart`, `lib/src/util/span.dart`, `lib/src/util/lazy_file_span.dart`, `lib/src/util/multi_span.dart`, external `source_span`/`string_scanner`/`term_glyph`/`source_maps`/`path`/`dart:core`                              | `sasscommon/` (spans, errors, scanner), `termglyph/`, `sourcemap/`, `sourcemapbuffer/` (see `upstream.md` pins) |
| C2  | `lib/src/logger*.dart`, `lib/src/logger/*.dart`, `lib/src/deprecation.dart`, `lib/src/syntax.dart`, `lib/src/utils.dart`, `lib/src/util/*.dart`, `lib/src/color_names.dart`                                                                | `sasslogger/`, `deprecation/`, `eval/syntax.go`, `unvendor/`, `util/`, `sassmath/`                              |

Notes:

- `evaluate.dart` is a 1:N split across `eval/evaluate*.go` plus helpers,
  meta, and init files. Never expect one Go file to mirror the whole Dart
  file.
- Package roots carry no `dart-source:` by policy (`patterns.md` §2).
  Qualifiers like `(interface only)`, `(errors)`, `(external)` disambiguate
  non-1:1 mappings — trust them, then verify.
- Stale/missing annotations are findings: a changed Dart file with no Go
  counterpart (`porting.md` step 2 recipe) means a missing port, not a pass.

## 3. Accepted Go adaptations (not bugs)

Check these first so review time goes to real bugs. Each is documented in
`patterns.md` / `architecture.md` / `critical-invariants.md` /
`divergences.md` / `ref/*` — the refs below are entry points, not proofs.

| Dart                          | Go                                                                                                            | Why                                                          |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| abstract/final/sealed classes | interfaces + `IsXxx()` markers, structs with anonymous embedding                                              | closed type set; `architecture.md` §3, `patterns.md` §3      |
| `accept<T>(v)`                | `AcceptValue/AcceptBool/AcceptVoid/AcceptExpr` over generic `Visitor[T]`                                      | one traversal, many results; `architecture.md` §4            |
| `throw` / `try on T catch`    | `(T, error)` + `errors.AsType[*T]`                                                                            | no exceptions; every fallible function returns `error`       |
| `null` returns                | `nil` pointers; `nil` slice vs empty slice preserved                                                          | `critical-invariants.md` nil rule                            |
| GC sharing                    | pointers and slices (same GC shape)                                                                           | `architecture.md` §5                                         |
| shared mutable frames         | `[]*LinkedMap` env stacks; `ModifiableCssXxx` wrappers + `Box`                                                | order-sensitive scopes; `architecture.md` §9–10              |
| `Map`/`Set` with value `==`   | `LinkedHashMap` + `ValueEquals`, `Equal()` methods                                                            | `patterns.md` §3, `critical-invariants.md`                   |
| class fields + zones          | `EvaluateVisitor` struct + explicit `*EvaluationContext`                                                      | `architecture.md` §7, `ref/eval.md`                          |
| `Future`/async twins          | not ported — sync only                                                                                        | `architecture.md` §6                                         |
| `buffer.write(x)` void        | `_, _ = sb.WriteString()`                                                                                     | identical output; `divergences.md` §2                        |
| `toString()` / inspect / css  | `String()` vs `SerializeValueInspect` vs `ToCssString(quote)` — never conflate                                | `patterns.md` §6                                             |
| `Module`/`Callable` identity  | reference identity                                                                                            | no ID counter in Dart; `architecture.md` §10                 |
| float formatting edge         | `float64()` per-product casts; `sassmathcgo` C-libm backend in tests/spec, opt-in `-tags cgomath` for the CLI | bit parity; `critical-invariants.md` FMA rule, `ref/math.md` |

If a difference is not in this table (or in `divergences.md`), treat it as
a suspect, not an adaptation.

## 4. Per-file review procedure

Apply to each map row in §2, in §5 order. Keep the Dart file open beside
the Go file(s); judge behavior, not shape.

1. **Locate and size.** Resolve Go counterpart(s) via the `rg` recipe in
   §2. Note splits/merges/qualifiers. If Dart changed since `PORTED_FROM`
   with no Go counterpart, file a missing-port finding and stop.
2. **Entry points and names.** For each public Dart function/method, find
   the Go func. Names convert mechanically (`isBogus` → `IsBogus`,
   `toSpace` → `ToSpace`); any rename beyond case/shape, moved logic
   between files, or dropped/added parameter is a finding unless §3 covers
   it (e.g. added `ec`/`io` params are expected).
3. **Control flow and guards.** Walk the Dart body in order: every guard,
   branch, loop bound, early return, and error site must have a Go
   counterpart producing the same outcome in the same order. A missing
   `if`, swapped branch order, or combined/split check that changes which
   error fires first is a bug — even if each branch looks right alone.
4. **Errors: type, message, span, trace.** For every Dart `throw`: the
   exception type (`SassScriptException` vs spanned `Runtime`/`Format`/
   `Sass`/`MultiSpan`), exact message text, span source node, and
   trace/cause/`loadedUrls`. Gruff rule: `Script` = unspanned value/type
   error inside a wrappable callback; anything escaping to the user must be
   spanned with trace.
5. **Wrapping, scope, and effects.** Check per-visit `addExceptionSpan` /
   `addErrorSpan` presence (must match Dart per-type, never a blanket
   wrapper in `accept()`); `Scope(..., when=HasDeclarations(children))`,
   not length checks or hardcoded bools; `at_root` handling,
   `!global` visibility, and CSS parent-chain behavior preserved.
6. **Types and data.** `nil` vs empty, `LinkedMap` ordering, set
   membership, string-form choice (§3 row), number/color-space handling.
   `==` or `reflect.DeepEqual` on a value type is a bug — must be `Equal()`.
7. **No panics.** Any `panic` on a reachable path is a finding
   (`critical-invariants.md`).

Record the outcome per file using §7 before moving on.

## 5. Suggested order (risk-first)

1. E1 evaluator core (`evaluate.go`, statement/expression visitors, helpers).
2. E2–E3 environment/scope + importer/import-cache.
3. E4 built-in functions (argument handling + error variants).
4. E5–E6 compile entry points, callables, evaluation context, configuration, modules.
5. A1–A3 AST + values (usually faithful; focus on new/changed nodes).
6. P1 parser, S1 serializer, S2 selectors/extend.
7. C1–C2 spans/errors/source-maps/logger/deprecation/utils/math.

Within a group, start with files touched since `PORTED_FROM`
(`git -C dart-sass diff --name-only $(cat PORTED_FROM)..origin/main -- lib/`
then §2 recipe per `porting.md` step 2).

## 6. Pattern scans (`rg` recipes)

Use after each file (or group) to catch what line-by-line reading misses.
Every hit needs a Dart-side check — scans suggest, files decide.

```sh
# error-type suspects: Script escaping where Dart raises a spanned error
rg -n 'NewSassScriptException' eval compile functions value --glob '!*_test.go'
# wrapping: visits missing addExceptionSpan / addErrorSpan
rg -n 'func \(v \*EvaluateVisitor\) visit' eval
rg -n 'addExceptionSpan|addErrorSpan|withEvaluationContext' eval
# scope: hardcoded when or len-based checks instead of HasDeclarations
rg -n 'HasDeclarations' eval sassenv
rg -n 'Scope\(|semiGlobal|atRoot' eval sassenv
# string forms conflated
rg -n 'ToCssString|SerializeValueInspect|\.String\(\)' value eval
# equality used on value types
rg -n 'reflect\.DeepEqual' --glob '*.go' . | grep -v _test
# float parity hazards
rg -n 'math\.Pow\(|math\.Sin\(|math\.Cos\(|math\.Atan2\(' value functions sassmath
# forbidden in compiler code
rg -n 'panic\(' --glob '!*_test.go' .
# annotation hygiene
rg -n 'dart-source:' --glob '*.go' . | grep -v _test | wc -l
```

`panic(` hits are allowed only in provably unreachable branches and
type-switch defaults; any other `panic` in non-test compiler code is a
finding. `DeepEqual` on a ported type is a finding.

## 7. Verification and finding log

Verify after every fix, not at the end. Per fix:

```sh
go build ./... && go vet ./...
go test ./...
SASS_SPEC=<real/path/under/sass-spec/spec> go test -tags=spec -timeout=30s ./spec/
```

Per plan-file exit the bar rises to the full battery (see
`CONTRIBUTING.md`): whole `sass-spec`, embedded harness, byte-identity
spot-check.

Byte-identity (`CONTRIBUTING.md`): bootstrap / `huge` / `huge10` Go == Dart.
Spot-check CLI parity while reviewing eval changes:

```sh
echo '<scss>' | dart run bin/sass.dart --stdin   # from dart-sass/
echo '<scss>' | go run ./cmd/go-sass --stdin
```

Log each file (or finding) in the same shape so later passes can skip
clean files without re-reading them:

```md
### `<dart-path>` ↔ `<go-path(s)>`

- Pins: dart `PORTED_FROM` + submodule short hash, go commit.
- Verdict: clean | bug | accepted-adaptation (§3 row + why).
- Bugs: Dart line(s) vs Go line(s), expected vs actual (message/span/trace/order/scope), repro (`SASS_SPEC=` group or stdin snippet).
- Gates: unit, spec subset — pass/fail.
```

Finding severity: (a) wrong output/message/span/trace, (b) wrong scope/
ordering/import effect, (c) missing port, (d) annotation drift. Cosmetic
structure with identical behavior is not a finding — if tempted to file
one, re-read §1 and §3 first.

## 8. Reference files

- `porting.md` — delta → file mapping, header/license rules, full battery.
- `patterns.md` — translation conventions (§3–7 error/dispatch/ownership).
- `architecture.md` — pipeline, package layout, eval/serialize splits.
- `critical-invariants.md` — never-violate rules (FMA, nil-slice, `Equal()`,
  no panics, accept families, byte-identity).
- `divergences.md` — intentional differences; do not re-file these.
- `ref/` — per-module detail (`eval.md` semantics, `common.md` errors,
  `visitors.md` dispatch, `environment.md`, `math.md`).
- `upstream.md` + `PORTED_FROM` — tracked commit/version/package pins.
- `CONTRIBUTING.md` — build/test/bench procedure.
