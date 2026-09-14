# The compile pipeline, end to end

One page showing how a Sass string becomes CSS: which function runs at each
stage and where everything lives. Read this before tracing any compile through
the code. For stage internals, follow the links to the per-module refs.

## The three entries

| Entry                             | Input       | Extra work before `compileString`                                                                                              |
| --------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------ |
| `CompileString(source, io, opts)` | source text | none — the pipeline itself                                                                                                     |
| `Compile(path, io, opts)`         | file path   | `ReadFile` → `Canonicalize` → `file:` URL → UTF-8 decode → syntax from extension; `LoadPaths` defaults to the file's directory |
| `CompileStringToResult(…)`        | source text | convenience shim with default options                                                                                          |

## `CompileString`, in order (`compile/compile.go`)

1. **Parse** — `value.NewScssParser / NewSassParser / NewCssParser(source, url, …)`
   by `opts.Syntax`, then `parser.Parse()`. Parse errors return directly.
   See `ref/parse.md`.
2. **Import cache** — `eval.NewImportCacheWithOptions(importers, loadPaths,
SASS_PATH-env-or-opt, …, io, packageConfig)`. Ordering: user importers →
   load-paths → `SASS_PATH` → package config. See `ref/importer.md`.
3. **Logger wrap** — user logger is always wrapped in
   `sasslogger.NewDeprecationProcessingLogger` (silence/fatal/future lists,
   repetition limit); `Validate()` runs **before** evaluation, `Summarize()`
   is deferred to run on success _and_ failure. See `ref/logger.md`.
4. **`compileStylesheet`** — evaluate + serialize (below).

## `compileStylesheet` (`compile/compile.go`)

1. **LEGACY_JS_API** deprecation if a node-package importer is set.
2. **`eval.Evaluate(stylesheet, importCache, nodeImporter, importer,
functions, logger, quietDeps, sourceMap)`** — see below. Returns
   `*EvaluateResult { Stylesheet, … }`.
3. **Serialize** — `value.SerializeWithSourceMap(stylesheet,
&SerializeOptions{Style, Charset}, smBldr)`. See `ref/serialize.md` +
   `ref/source-maps.md`.
4. **Source-map URL rewrite** — post-serialization, mirroring Dart's
   `_compileStylesheet`.
5. **`NewCompileResult(evalResult, serResult)`** — exposes `CSS()`,
   `SourceMap()`, `LoadedUrls()`.
6. **Error path** — if `emitErrorCss` and the error is one of the four spanned
   variants (`SassException` / `SassRuntimeException` /
   `MultiSpanSassException` / `MultiSpanSassRuntimeException` — never the
   unspanned `SassScriptException` family), the error renders as CSS
   (`errorToCssString`) inside a synthetic result; otherwise the error
   propagates. See `ref/compile.md`.

## `Evaluate()` (`eval/evaluate.go`)

1. `NewEvaluateVisitor(importCache, logger)` — fresh visitor plus a fresh
   `compileContext` identity token (see `ref/compile-context.md`).
2. **User functions first** (`RegisterUserFunctions` — registered before
   built-ins so built-ins take priority), then built-ins registered
   (`RegisterBuiltInFunctions`), flags in (`SetQuietDeps`, `SetSourceMap`,
   `SetNodeImporter`).
3. `v.Run(importer, stylesheet)` evaluates inside the evaluation context —
   `EvaluateVisitor` methods + `switch` dispatch, **no visitor-trait impls**.
   See `ref/eval.md`.

## Tracing a compile (where to look)

| Question                             | Start here                                                                         |
| ------------------------------------ | ---------------------------------------------------------------------------------- |
| What did the parser produce?         | `parser.Parse()` return in `CompileString`; `value/parse_*` per `ref/parse.md`     |
| Which importer resolved a URL?       | `ImportCache.Canonicalize` (`eval/import_cache.go`); kinds in `ref/importer.md`    |
| Why is a variable/function missing?  | `sassenv.Environment` lookups; views in `ref/environment.md` + `ref/member-map.md` |
| What span/trace will an error carry? | `addExceptionSpan` + `stackTrace` (`eval/`); `ref/eval.md` error flow              |
| What CSS came out?                   | `EvaluateResult.Stylesheet`; serialization in `ref/serialize.md`                   |
| What will the source map contain?    | `sourcemap.Builder` output + rewrite step 4; `ref/source-maps.md`                  |

## File mapping

| Stage            | Dart                                         | Go                                                              |
| ---------------- | -------------------------------------------- | --------------------------------------------------------------- |
| entry + pipeline | `lib/src/compile.dart`                       | `compile/compile.go`, `compile_options.go`, `compile_result.go` |
| executable layer | `lib/src/executable/compile_stylesheet.dart` | `compile/compile_stylesheet.go`, `cmd/go-sass/` CLI             |
| evaluate         | `lib/src/visitor/evaluate.dart`              | `eval/evaluate*.go`                                             |
| serialize        | `lib/src/visitor/serialize.dart`             | `value/visitor_serialize.go`                                    |
