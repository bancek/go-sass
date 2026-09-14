# Module: `value/visitor_serialize.go` (serializer)

The serializer: CSS output tree (or a value) → a CSS string, with an optional
source map.

## Entries

```go
func Serialize(node CssNode, opts *SerializeOptions) (*SerializeResult, error)
func SerializeWithSourceMap(node CssNode, opts *SerializeOptions, bldr *sourcemap.Builder) (*SerializeResult, error)
func SerializeValue(v Value, quote bool) (string, error)
func SerializeValueFull(v Value, quote bool, inspect bool) (string, error)
func SerializeValueInspect(v Value) (string, error)
```

`SerializeOptions` uses nil-means-default pointers (`UseSpaces *bool` nil =
true, `IndentWidth *int` nil = 2, `LineFeed *LineFeed` nil = LF); a nil
options pointer takes all defaults (expanded, spaces, width 2, LF, charset
on, no source map). `SerializeResult{CSS, SourceMap}` carries the output.

## Structure and dispatch

`SerializeVisitor` holds the buffer plus a `SerializeState` (indentation,
style, inspect, quote, line feed, indent char/width). `forNode(node, cb)`
runs a callback and associates every byte it writes with the node's span
(mirroring Dart's `_for`), routing through the source-map buffer's
`ForSpan(span, cb)` — a no-op mapping on `NoSourceMapBuffer`.

All visitor methods return `(struct{}, error)` via `AcceptVoid` — no
serializer method is infallible. The `Value` / `Selector` / `Css` visitor
impls cover every variant of their families.

## Number formatting (Dragon4 emulation)

Numbers serialize shortest-round-trip decimals with Dart's switch points:
plain decimal up to `|v| ≥ 1e21` (not Go's `strconv` e-notation threshold),
rounded to 10 fractional digits with ripple-carry through the integer part,
trailing zeros trimmed. A fuzzy-integer fast path writes integers without a
`.0` suffix; compressed output strips leading zeros (`0.5 → .5`, keeping the
sign so `-0.5` stays `-0.5`). See `value/visitor_number.go`.

## Color

The color serializer is a decision tree over `ColorSpace` and `ColorFormat`:

```
legacy space (rgb/hsl/hwb) with all channels present
  → named color, hex, or rgb()/hsl()/hwb() (compressed picks the shortest)
legacy out-of-gamut (inspect=false) → writeHsl()
inspect && hwb → writeHwb(); format==RgbFunction → writeRgb(); Preserved → the stored text
Lab/Lch/Oklab/Oklch → lab-style function, or color-mix() when out of gamut
other modern spaces → color(space c0 c1 c2 / α)
```

Compressed `tryIntegerRgb` shortens hexable colors (comparing named-vs-hex
lengths); the compressed rgb-vs-hsl tiebreak compares **compressed** number
spellings, never uncompressed ones. `RgbFunction` format forces `rgb()`;
`Preserved` re-emits the original source text. See `value.md`.

## Strings, lists, calc

- **Quoting:** auto-detect a quote character; if both `'` and `"` appear,
  escape `"` and keep `'` (force-double mode). Escape backslashes, control
  chars, and private-use chars (compressed passes private-use through).
- **Lists:** separator and bracket handling; an empty unbracketed list in CSS
  mode errors (`() isn't a valid CSS value.`).
- **Calc:** serializes calculation arguments (number / string /
  interpolation / operation / nested calculation).

## `inspect`

`inspect` controls the error surface: in CSS mode (`inspect = false`) maps,
functions, mixins, and empty unbracketed lists error; `inspect = true`
serializes all types without error.

## CSS output formatting

| Feature                    | Expanded       | Compressed   |
| -------------------------- | -------------- | ------------ |
| Semicolon after last child | yes            | no           |
| Spaces around `{`/`}`      | yes            | no           |
| `rgb(1, 2, 3)`             | `rgb(1, 2, 3)` | `rgb(1,2,3)` |
| `0.5`                      | `0.5`          | `.5`         |
| `@media` query spaces      | spaces         | minimal      |
| Top-level blank lines      | `isGroupEnd`   | none         |
| Charset                    | `@charset`     | BOM          |

Only Expanded and Compressed exist (Dart has no others). `isGroupEnd` on the
last child of a root-level group emits the blank line between groups.
`writeWithIndent` (custom-property values, loud comments) never splits a code
point — Dart's UTF-16 scanner doesn't either.

## Source-map integration

`forNode` maps the current output position to the source span start. While in
a span, each `\n` auto-maps the output line to the source span; BOM/charset
are handled via prefix offsets. See `source-maps.md` for the builder.

## File mapping

| Dart                                          | Go                           |
| --------------------------------------------- | ---------------------------- |
| `lib/src/visitor/serialize.dart`              | `value/visitor_serialize.go` |
| `lib/src/visitor/*` (value, color, number, …) | `value/visitor_*.go`         |
