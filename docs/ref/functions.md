# Module: `functions/`

The built-in Sass function library: the `sass:color`, `sass:math`,
`sass:list`, `sass:map`, `sass:selector`, `sass:string`, and `sass:meta`
modules, their deprecated global aliases, and the special `if()` function.

## Layout

`functions/` holds **pure registries** — one file per module (`color*.go`,
`math.go`, `list.go`, `map.go`, `selector.go`, `string.go`, `meta.go`,
`package.go`) plus `*_init.go` registration files and `disallowed.go`.
Evaluator-owned behavior (the `meta.*` functions that re-enter evaluation:
`call`, `variable-exists`, `load-css`) lives in `eval/evaluate_meta.go`;
`functions/` depends only on `value`, `evalcontext`, `deprecation`,
`sasscommon`, and `extend` — never on the evaluator itself.

`if($condition, $if-true, $if-false)` is hand-authored as a legacy-if
expression in the parser — it is only reachable as a callable through
`meta.call()`.

## Global vs module functions

Global functions (`rgb`, `lighten`, `str-length`, …) are deprecated in favor
of their module forms. A global is built by wrapping the module function with
a deprecation warning and a renamed signature (`WithDeprecationWarning("map",
nil).WithName("map-get")`). The wrapping order matters:
`WithDeprecationWarning` runs **before** `WithName`, so the deprecation
message names the _original_ module function. Calling a wrapped global emits
a `global-builtin` deprecation warning.

## Behavior notes

- **Error-message shape.** Built-in argument errors are formatted as
  `$name: message` (Dart's `toString()`), asserted byte-identically.
- **`$min-number`** is `5e-324` — the smallest _subnormal_ (Go's
  `math.SmallestNonzeroFloat64`, Dart's `double.minPositive`), **not** the
  smallest normal (2.2e-308).
- **`disallowed.go`** is the set of global function names minus the
  CSS-compatible ones; it feeds the CSS parser's bare-call validation, which
  decides which function calls are valid in plain CSS.
- **`callable_*.go`** (`callable_built_in.go`, `callable_user_defined.go`,
  `callable_plain_css.go`) hold the concrete callable types used by the
  registries — see `callable.md`.
- The math builtins route through `sassmath` (see `math.md`); `ceil`/`floor`/
  `round` saturate at int64 extremes like Dart's `num.ceil()`, with the
  extreme sentinels printing as exact digit strings.

## File mapping

| Dart                       | Go               |
| -------------------------- | ---------------- |
| `lib/src/functions/*.dart` | `functions/*.go` |
