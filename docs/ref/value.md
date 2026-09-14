# Module: `value/` (values)

The Sass value type and its concrete types. This page covers the runtime
values only; the AST, parser, and visitors that share the `value/` package
are documented in `ast.md`, `parse.md`, and `visitors.md`. See also
`callable.md` for `SassFunction`/`SassMixin`'s callable machinery.

## The `Value` interface

```go
type Value interface {
    AstNode
    IsValue()
    // String(), Equal(), accept methods, assertions, operators…
}
```

Every Sass value is a pointer to a concrete struct implementing `Value`.
Singletons (`SassTrue`/`SassFalse`, the null value) are package-level
instances — comparison is still `Equal()`, never `==`.

### Equality and hashing

`Equal()` performs **deep value equality** per Dart's `operator==` — not
pointer identity. The only identity-compared values are `SassFunction` and
`SassMixin` (whose equality is their callable's identity). `SassMap` is
backed by `linkedhashmap.LinkedHashMap[Value, Value]` with the `ValueEquals`
comparator, so map keys work by value equality. A lazy hash cache avoids
re-walking deep lists and maps on repeated lookup.

## Concrete types

### `SassNumber`

An **interface with three implementations** sharing `sassNumberBase`:

- `Unitless` — no units.
- `SingleUnit` — one numerator unit.
- `ComplexUnit` — multiple/composed units.

```go
type SassNumber interface {
    Value
    IsInt() bool
    AsInt() (int64, bool)
    UnitString() string
    HasUnits() bool
    // … coercion, arithmetic, math functions
}
```

Arithmetic dispatches on the operand implementations. The convertible unit
families are: lengths (`in`, `cm`, `pc`, `mm`, `q`, `pt`, `px`), angles
(`deg`, `grad`, `rad`, `turn`), time (`s`, `ms`), frequency (`Hz`, `kHz`),
and pixel density (`dpi`, `dpcm`, `dppx`). Font-relative units (`em`, `rem`,
`vw`, …) are _compatible_ but **not convertible**. Math-function unit
semantics: `sqrt` is unitless; `sin`/`cos`/`tan` coerce to radians →
unitless; `atan`/`asin`/`acos` take unitless → `deg`; `abs` preserves units.

`parseNumberOrString` tries `parseNumber()` first and falls back to an
unquoted `SassString` on `SassFormatException`.

### `SassColor`

Colors carry a space, channels, alpha, missing-channel mask, and format.
`ColorSpace` is an **interface with one singleton struct per space** (three
legacy `rgb`/`hsl`/`hwb`, four Lab-like `lab`/`lch`/`oklab`/`oklch`, and the
modern spaces):

```go
type ColorSpace interface {
    Name() string
    Channels() [3]LinearChannel
    IsBounded() / IsLegacy() / IsPolar() bool
    ToLinear() / FromLinear() / Convert() …
}
```

- Missing channels are `0.0` with a bitmask; the bitmask propagates through
  every conversion.
- Channel metadata lives in `color_channel*.go` (`ColorChannel`,
  `LinearChannel`, `AlphaChannel`); conversions and gamut mapping in
  `color_conversions*.go`.
- Two Dart-replicated behaviors: negative saturation/chroma shifts the hue
  180° and negates the value; `normalizeHue` uses Dart's lossy
  `(h % 360 + 360) % 360` formula verbatim (see `math.md`); cubing uses
  `x*x*x`, not `Pow` (Dart's integer-`pow` path).
- Callable defaults must be real Dart defaults, not null: `hwb $alpha`
  defaults to the number `1`, `join`/`append` `$separator`/`$bracketed` to
  `auto`.

### `SassCalculation`

`{ name, arguments }` with arguments of number, calculation, string,
interpolation, or nested operation. Thirty-plus constructors (`NewCalc`,
`NewMin`, `NewMax`, `NewClamp`, …) and a construct-time `simplify()` engine.
Error layering mirrors Dart's throw-site discipline:
`verifyCompatibleNumbers` raises the **unspanned** script error and the eval
caller attaches spans — the `BinaryOperation` arm wraps in
`addExceptionSpan` (whole-operation span) while multi-arg calls re-verify
against the original nodes for per-operand `MultiSpan` labels.

### `SassString`, `SassList`, `SassArgumentList`, `SassMap`

- `SassString { text string, hasQuotes bool }`. Only `StringExpression`
  overrides the default nil source-interpolation.
- `SassList { contents []Value, separator, hasBrackets }`; `ListSeparator`
  is `Space | Comma | Slash | Undecided`.
- `SassArgumentList` embeds `*SassList` plus `Keywords map` and a
  keywords-accessed flag; excess positionals and unmatched named args pack
  into a trailing argument list at call sites. Constructors validate and
  return `(*T, error)`.
- `SassMap` is entries + the `LinkedHashMap` above. `AsList() ([]Value,
error)` converts (maps only via fallible construction).

The empty value `()` parses to an empty unbracketed `SassList` (never null);
`map.get` traverses an empty list as an empty map. An empty argument list
counts as an empty map too — this is what lets `load-css` treat `$with: ()`
as an explicitly empty configuration.

### `SassFunction` / `SassMixin`

Opaque references carrying a `sasscallable.Callable` plus the compilation's
`compileContext` token (`NewSassFunctionWithCompileContext` /
`NewSassMixinWithCompileContext`). Equality is callable identity; the
serializer prints the callable's name; cross-compilation use errors via
`AssertCompileContext`. See `callable.md` and `compile-context.md`.

### Display vs CSS forms

`String()` (Dart `toString`, inspect serialization) vs `ToCssString(quote)`
(Dart `toCssString`, shared serializer, errors on non-CSS values: maps,
functions, mixins, empty unbracketed lists). Callers must not conflate the
two (see `patterns.md` §6).

## Operators, assertions, and helpers

Binary (`plus`, `minus`, `times`, `divided_by`, `modulo`, `single_equals`,
`greater_than`, …) and unary operators dispatch on operand types with
Dart-exact edges: `Number + Color` raises `Undefined operation`, while
non-number/non-color operands concatenate (`1 + true → 1true`).

Assertion methods return `(T, error)` — no panics. `assertMap` on an empty
list constructs an empty map (there is no inner map to borrow).

## Golden-value testing

Color and math values are pinned with **golden tests**: print the real Dart
output at full precision, hardcode it, assert within tolerance, and delete
the printer. Never compute the expected value from the same formula as the
code under test. See `patterns.md` §8.

## File mapping

| Dart                             | Go                                                                  |
| -------------------------------- | ------------------------------------------------------------------- |
| `lib/src/value/*.dart`           | `value/value.go`, `number*.go`, `color*.go`, `list.go`, `map.go`, … |
| `lib/src/value/calculation.dart` | `value/calculation.go`                                              |
| `lib/src/value/color/` (spaces)  | `value/color_space_*.go`, `value/color_conversions*.go`             |
