# Module: `sasscommon/`

Errors, source locations, and the span scanner. This is the foundation every
other module builds on.

## Errors

The exception family mirrors Dart's (all in `sasscommon/exception.go`):

```go
SassScriptException          // unspanned value/type error from built-ins
SassRuntimeException         // + span, trace, cause, loaded URLs
SassFormatException          // parse error (+ source context)
SassException                // generic spanned error
MultiSpanSassException / MultiSpanSassRuntimeException /
MultiSpanSassFormatException // + secondary spans and labels
MultiSpanSassScriptException // unspanned multi-location error (span attached later)
ScanError                    // scanner-level error (+ span, cause)
```

- `SassScriptException` is the unspanned error built-ins raise; the evaluator
  wraps it into a spanned `Runtime` (adding span + trace) at the boundary.
- Every spanned variant carries `Cause` (a chained error) and loaded URLs
  (the canonical URLs loaded so far — the embedded layer reports these even
  on failure). `Runtime` and `MultiSpan` also carry `Trace` (an empty
  `MultiSpan` trace means "derive the frame from the span").
- Loaded URLs are stamped at the evaluate boundary on **all** spanned
  variants (mirroring Dart's `error.withLoadedUrls` on every
  `SassException`), and additional-span rebuilds preserve them.
- Chaining uses `Cause`/`Unwrap()` + `ThrowWithTrace(new, cause)`,
  inspected with `errors.AsType[*T]` — never a direct type assertion.
- Rendering goes through `ErrorWithOptions(HighlightOptions)` (message +
  span highlight + indented trace); `core_errors.go` ports Dart's
  `ArgumentError`/`RangeError`/`StateError`/`UnsupportedError`.

### Frames and traces

```go
type Frame struct { /* member, span, URL */ }
type Trace struct { /* frames */ }
```

`Trace` is Dart's `package:stack_trace` `Trace`: an ordered (outermost →
innermost) list of frames. It owns its frames and is carried structurally
through the evaluator and the `Logger` seam; stringification happens only at
the output boundary, which needs the I/O seam for `prettyUri` because Go
never reads the ambient working directory. There is no string→trace parser
(Dart has none) — traces are never stringified mid-pipeline only to be
re-parsed.

### Error layering

The parser uses its own internal error types so that spans can be adjusted
before they become public errors:

```
ScanError → SourceSpanFormatException → SassFormatException → public error
```

The reason for the layering is **span adjustment**: a zero-length "expected
X" error must be repositioned to point at the preceding newline, which
requires the original `FileSpan` (and source text) to still be available.
See `parse.md`.

## Spans

```go
// sasscommon/source_span_file.go
type FileSpan interface {
    Span
    SourceURL() (*url.URL, error)
    File() (*FileSource, error)
    Context() (string, error)   // surrounding source, trimmed
    // … offsets, text, highlighting
}
```

- `FileSource` holds the URL, text, and precomputed line-starts for
  binary-search line lookups. Provenance checks (interpolation mapping,
  operator-span trimming, format-exception change detection) use allocation
  identity — two distinct allocations with equal content are equal but never
  identical, and only identity answers "already mapped here".
- `FileSpan` is an interface (`Simple` / `Lazy` / `MultiSpan`
  implementations); `nil` is the absent span, never an empty struct. Core
  accessors handle absence gracefully; utility methods (`Expand`, `Subspan`,
  `MapSpan`, …) return `(…, error)` — propagate, never substitute a bogus
  span.
- `LazyFileSpan` defers span resolution through a builder plus cache.
  `MultiSpan` pairs a primary span+label with secondary spans.
- The **highlighter** (`source_span_highlighter.go`) slices and counts in
  **characters, not bytes**: caret widths and line lengths count display
  characters, and source-map columns count characters/UTF-16 units — matching
  Dart. Byte-based slicing panics mid-codepoint and undercounts carets on
  non-ASCII lines.

## Scanner

`SpanScanner` is the character-level scanner used by the parser:
byte-offset `pos` (multi-byte UTF-8 advances bytes but line/column by one),
`peek` returning `-1` at EOF, `read/scan/expect` consuming,
`state/setState/setPosition` for backtracking, `spanFrom/spanFromTo/emptySpan`
for spans. Scanner errors are `ScanError` (not public errors), so the parser
can adjust spans.

## AstNode and CssValue

`AstNode` exists so heterogeneous node types can report their span
(`Span() (FileSpan, error)`); it is not used for visitor dispatch.
`CssValue[T]` pairs a value with a span; equality ignores the span.

## File mapping

| Dart                                          | Go                                          |
| --------------------------------------------- | ------------------------------------------- |
| `lib/src/exception.dart`                      | `sasscommon/exception.go`, `core_errors.go` |
| `(external) package:source_span`              | `sasscommon/source_span_*.go`               |
| `(external) package:string_scanner`           | `sasscommon/string_scanner_span_scanner.go` |
| `(external) package:path` (`prettyUri`)       | `sasscommon/pretty_uri.go`                  |
| `(external) dart:core` (`ArgumentError` etc.) | `sasscommon/core_errors.go`                 |
