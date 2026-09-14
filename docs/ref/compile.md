# Module: `compile/`

The public compile API: `CompileString`, `Compile`, and the CLI pipeline.

## Types

`CompileOptions` carries every compilation setting: `Importers`,
`Importer` (the entrypoint importer), `LoadPaths`, `SassPath` (from
`SASS_PATH`), `URL`, `PackageConfig`, `NodePackageImporter`, `Functions`,
`Logger`, `QuietDeps`, `SourceMap`, `EmbedSourceMap`,
`IncludeSourceMapSources`, `EmitErrorCSS`, `Style`, `Charset`, the
`SilenceDeprecations`/`FatalDeprecations`/`FutureDeprecations` lists,
`Verbose`, `Unicode`, `AlertColor`, `AlertAscii`, and `Syntax`
(`eval.SyntaxSass` / `SyntaxCSS` / default SCSS). A `nil` options pointer gets
sane defaults (`Unicode: true, Charset: true`).

```go
func CompileString(source string, io sassio.IO, opts *CompileOptions) (*CompileResult, error)
func Compile(path string, io sassio.IO, opts *CompileOptions) (*CompileResult, error)
func CompileStringToResult( /* … */ ) (*CompileResult, error)

func (r *CompileResult) CSS() string
func (r *CompileResult) SourceMap() *sourcemap.SingleMapping
func (r *CompileResult) LoadedUrls() *orderedset.LinkedSet[string]
```

## The compile flow

```
CompileString(source, io, opts)
├── parse source → *value.Stylesheet   (syntax: scss/sass/css)
├── build ImportCache from importers + load_paths (+ SASS_PATH env)
├── wrap logger → DeprecationProcessingLogger (validate + deferred summarize)
├── compileStylesheet:
│   ├── LEGACY_JS_API deprecation if nodePackageImporter is set
│   ├── eval.Evaluate() → *EvaluateResult   (on error, emitErrorCss renders the error as CSS)
│   └── value.SerializeWithSourceMap() → CSS + SingleMapping
└── return *CompileResult
```

- `Compile(path, ...)` reads the file, resolves the absolute `file:` URL,
  defaults `LoadPaths` to the file's directory, and infers syntax from the
  extension (`.sass` → Sass, `.css` → CSS, else SCSS).
- **Deprecation triggers:** `COMPILE_STRING_RELATIVE_URL` fires only for a
  relative entry URL without a node-package importer; `LEGACY_JS_API`
  fires when `nodePackageImporter != nil`.

## ImportCache ownership

The `ImportCache` is created in `CompileString`, passed into `Evaluate()`,
and held by the visitor for the duration of the run. There is no borrowing —
ownership flows with the call.

## Error CSS

`errorToCssString(err, unicode)` renders a spanned error as CSS:

1. Replace `*/` → `*∕`.
2. Escape non-ASCII runes as hex for the `content:` property (the quoted
   inspect-form string).
3. Wrap the message in a `/* ... */` comment.
4. Emit `body::before { ...; content: <escaped>; }`.

`EmitErrorCSS` applies only to the four spanned variants
(`SassException` / `SassRuntimeException` / `MultiSpanSassException` /
`MultiSpanSassRuntimeException`) — **never the unspanned
`SassScriptException` family**. This mirrors Dart's `on SassException`: a
parse error with `--error-css` renders CSS and exits 65, while a missing
input file exits 66.

## Source-map URL rewriting

After serialization, source-map URLs are rewritten (empty entry URLs become
`data:` URLs from the entry source text), mirroring Dart's
`_compileStylesheet`.

## CLI

The CLI (`cmd/go-sass/`) resolves `input.scss [output.css]`,
`input.scss:output.css`, multiple pairs, and directory compilation, handles
stdin/stdout, and exits `64` on usage errors (including `--help`, matching
Dart), `65` on a Sass error, or `66` on a filesystem error. See
[`cmd/go-sass/README.md`](../../cmd/go-sass/README.md) for the flag reference
(`--style`, `--load-path`, `--source-map`, `--fatal-deprecation`, …).
`--watch`, `--poll`, `--update`, and `--interactive` are not implemented.

## File mapping

| Dart                                        | Go                                 |
| ------------------------------------------- | ---------------------------------- |
| `lib/src/compile.dart`, `executable/*.dart` | `compile/*.go`, `cmd/go-sass/` CLI |
