# Modules: `eval/import_cache.go`, `eval/importer*.go`

Stylesheet resolution: turning `@use`/`@forward`/`@import` URLs into loaded
stylesheets. Ports `lib/src/import_cache.dart`, `lib/src/importer.dart`, and
`lib/src/importer/*.dart` (Go is sync-only; the async twins have no
counterpart).

## `ImportCache`

Created by `NewImportCacheWithOptions(importers, loadPaths, sassPath, …)`
with ordering **user importers → load-paths → `SASS_PATH` → package config**
(matching Dart). It holds per-importer canonicalize caches, load caches, and
the shortest-original-URL data for stack-frame humanization (`Humanize`,
`SourceMapURL`).

Ownership, not borrowing: created in `CompileString`, passed into
`Evaluate()`, held by the visitor for the run. Relative (scheme-less) URLs
try the base importer first with the URL resolved against the base URL.

## `Importer` interface

```go
// eval/importer.go
type Importer interface {
    Canonicalize(u *url.URL, ctx *CanonicalizeContext) (*url.URL, error)
    Load(canonicalURL *url.URL) (*ImporterResult, error)
    // … CouldCanonicalize, display
}
```

Implementations, one file each:

| File                                     | Dart counterpart                            | Notes                                                                                                                             |
| ---------------------------------------- | ------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `importer.go`                            | `importer.dart`                             | interface + `CouldCanonicalize` defaults (`NoOp → false`); `pkg` is a non-canonical scheme                                        |
| `importer_no_op.go`                      | `importer/no_op.dart`                       | all-nil, `"(unknown)"` display                                                                                                    |
| `import_cache.go` (`FilesystemImporter`) | `importer/filesystem.dart`                  | wrapped-relative-only load-path fallback; relative-URL CWD deprecation warn                                                       |
| `importer_package.go`                    | `importer/package.dart`                     | `package:` scheme only; "Unknown package." / "Unsupported URL"                                                                    |
| `node_package_importer.go`               | `importer/node_package.dart`                | `pkg:` scheme; exports-map resolution in insertion order; missing package.json surfaces as a read error                           |
| `importer_result.go`                     | `importer/result.dart`                      | `{contents, syntax, sourceMapUrl}`; missing source-map URL falls back to a `data:` URL                                            |
| `resolve_import_path.go`                 | `importer/utils.dart` (`resolveImportPath`) | `.import`-first, sass→scss→css, partial-before-full, `index`                                                                      |
| `importer_utils.go`                      | `importer/utils.dart`                       | URL-scheme validity, syntax-for-path                                                                                              |
| `importer_canonicalize_context.go`       | `importer/canonicalize_context.dart`        | explicit `CanonicalizeContext{fromImport, containingURL}` replacing Dart's zone-ambient context; containing-URL access is tracked |

## Error discipline

Importer failures in Dart are plain values (not exceptions); the Go port
returns unspanned errors which the load site wraps with span + trace. Do not
"upgrade" these to spanned errors at the throw site — the span must come
from the `@use`/`@import` rule, not the importer internals.

## Working here

- New importer kinds add an implementation + registration; the interface
  keeps every match site explicit.
- Resolution-order changes are high-blast-radius (they shadow later
  importers): verify with the `directives/use`, `directives/import`, and
  `directives/forward` spec groups plus a differential CLI run.
- `modificationTime` has no Go counterpart anywhere (a watch-mode/staleness
  concern only) — accepted gap, do not port it as part of behavior work.
