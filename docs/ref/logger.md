# Modules: `sasslogger/`, `deprecation/`

Logging and deprecation handling.

## The `Logger` interface

```go
// sasslogger/logger.go
type Logger interface {
    Warn(message string, span *sasscommon.FileSpan, trace *sasscommon.Trace)
    Debug(message string, span *sasscommon.FileSpan)
    WarnDeprecation(message string, span *sasscommon.FileSpan, deprecation *deprecation.Deprecation, trace *sasscommon.Trace) error
}
```

- Spans and traces are **pointers** (`nil` = absent), mirroring Dart's
  nullable parameters.
- The stack trace is carried **structurally** as `*sasscommon.Trace`
  (Dart's `Trace?`), not as a pre-formatted string. Loggers stringify only
  at the render boundary (`StderrLogger`, embedded).
- `WarnDeprecation` returns `error` — a fatal deprecation fails the
  compilation through the normal error path.
- `Logger` is one of several host-supplied seams (`UserImporter`,
  `sassio.IO`); it is the only _host-supplied_ one besides `UserImporter`.

### Implementations

- `QuietLogger` — a no-op (used by `--quiet` / silent loggers).
- `StderrLogger` — writes to an `io.Writer` (defaults to stderr);
  `NewStderrLogger(color, unicode)`, `NewStderrLoggerWithWriter(color,
unicode, w)` for tests.
- `TrackingLogger` — decorates a logger with emitted flags.
- `DeprecationProcessingLogger` — decorates a logger, applying the
  silence/fatal/future deprecation lists and a repetition limit; `Validate()`
  runs before evaluation, `Summarize()` after (even on failure).

`NewDefaultLogger(unicode)` returns a `StderrLogger` that detects ANSI color
support from `TERM`/`COLORTERM` plus terminal detection.

## Deprecation

`deprecation.Deprecation` carries id, deprecated-in/obsolete-in versions,
description, and the future flag (~31 instances: `CALL_STRING`,
`SLASH_DIV`, `IMPORT`, …). `FromID(id)` looks them up;
`ForVersion(version)` iterates the full set. Deprecations reach parsers as
explicit optional parameters (see `patterns.md` §9).

## File mapping

| Dart                       | Go                           |
| -------------------------- | ---------------------------- |
| `lib/src/logger/*.dart`    | `sasslogger/*.go`            |
| `lib/src/deprecation.dart` | `deprecation/deprecation.go` |
