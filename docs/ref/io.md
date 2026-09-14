# Module: `sassio/`

The host I/O abstraction: the seam through which the compiler reads files,
canonicalizes paths, and reports filesystem state. It is the extension point
a host implements.

## The `IO` interface

```go
// sassio/io_interface.go
type IO interface {
    ReadFile(name string) ([]byte, error)      // Matches Dart: readFile
    WriteFile(path string, contents []byte) error
    DeleteFile(path string) error
    // … dirExists, canonicalize, currentDir, environment, stdin/stdout …
}
```

Methods mirror Dart's `io.dart` one-to-one (`// Matches Dart:` on each).

## `IOError`

I/O failures carry message, kind, and path — no `os` types leak into the
interface, so alternative hosts (in-memory, virtual) implement it freely.

## Implementations

- **`DefaultIO`** — real filesystem. `canonicalize` is **lexical**
  (`default_io_canonicalize.go` plus the `default_io_real_case_path.go`
  case-corrector on case-insensitive filesystems), **preserving symlink
  names** (matching Dart).
- **`VirtualIO`** (`spec/virtual_io.go`) — in-memory filesystem for tests
  and the spec runner, with a real-filesystem fallback for unknown paths.

## Threading principles

Four rules govern how I/O flows through the code:

1. Use `io.CurrentDir()` — never `os.Getwd()`.
2. Build `file:` URLs via path helpers — never string formatting.
3. Thread `io` through the compile options and evaluator (not per-function
   globals) — errors are constructed in eval but formatted in compile.
4. Spans/traces do **not** store I/O — rendering functions receive it from
   the caller (there is no ambient working directory).

Two further invariants:

- **Absolute-path invariant:** every path passed to file-URL construction
  must already be absolute; all call sites route through `Canonicalize()` or
  an absolute CWD-joined path first.
- **`parse_import_url.go` is off-limits:** it parses SCSS _source_ URL
  strings (Dart-compatible `Uri.parse` handling), not filesystem paths.

## File mapping

| Dart                                     | Go                                                                  |
| ---------------------------------------- | ------------------------------------------------------------------- |
| `lib/src/io.dart`                        | `sassio/io_interface.go`, `default_io.go`                           |
| `(external) package:path` (canonicalize) | `sassio/default_io_canonicalize.go`, `default_io_real_case_path.go` |
