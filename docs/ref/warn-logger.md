# Deferred deprecation warnings

The pattern for code that cannot reach the evaluator but must emit a
deprecation warning.

## Why the pattern exists

Three implementations face the same problem — getting a deprecation warning
from utility code to the logger, with dedup, quiet-deps, and stack trace:

| Lang        | Mechanism                                                                                         |
| ----------- | ------------------------------------------------------------------------------------------------- |
| Dart        | Zone-based ambient `EvaluationContext.current`; utility code calls `warnForDeprecation()`.        |
| Go          | An explicit `*EvaluationContext` threaded everywhere, or a stored `warnFn` field on the importer. |
| Other ports | Buffer the warning, return, flush through the eval pipeline.                                      |

Go uses Dart's spirit without its mechanism: instead of an ambient zone,
callers that have an evaluation context pass it (or a narrow warning
callback derived from it) explicitly. Importers that outlive a single call
store a `warnFn` field set at construction. Code with genuinely no path to
the evaluator buffers warnings and lets the caller flush them afterward.

## Tiers

1. **Direct** — evaluator code warns through the evaluation context
   (`Warn` / `WarnDeprecation`), which applies dedup, quiet-deps, and trace
   handling.
2. **Threaded context** — built-in callables receive `*EvaluationContext`
   as a parameter (the zone replacement).
3. **Stored callback** — importers keep a `warnFn` set by their owner.
4. **Buffered** — parser helpers and `value/` utilities without any context
   buffer warnings for the caller to flush.

## File mapping

| Dart                                     | Go                                                              |
| ---------------------------------------- | --------------------------------------------------------------- |
| `lib/src/evaluation_context.dart` (zone) | `evalcontext/evaluation_context.go` (explicit param / `warnFn`) |
