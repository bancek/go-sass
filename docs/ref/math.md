# Module: `sassmath/` (+ `sassmath/sassmathcgo/`)

The math backend used by the color pipeline, trig functions, and
`math.pow()` — and the floating-point parity model that keeps output
bit-identical to Dart.

## Why the wrappers exist

Go's `math.Sin`/`Cos`/`Atan2`/`Pow` are Go's own implementations, diverging
from C libm by up to 7 ULP (measured: `sin(pi)`); Dart uses C libm via the
VM. Every libm-sensitive call in the port therefore routes through
`sassmath`, which exposes hook variables defaulting to stdlib:

```go
// sassmath/math.go
var CustomPow func(x, y float64) float64
var CustomSin func(x float64) float64
// … Atan2, Cos, Tan, Atan, Asin, Acos, Log, Sqrt, Abs

func Pow(x, y float64) float64 {
    if CustomPow != nil { return CustomPow(x, y) }
    return math.Pow(x, y)
}
```

`sassmath.Pow` (and siblings) are the single entry points for the whole port
(Sass `math.pow()`, calc `pow`, and every color-space conversion).

## The `sassmathcgo` backend

`sassmath/sassmathcgo/math_cgo.go` overrides every hook with C libm
(`math.h` via cgo: `pow`, `atan2`, `sin`, `cos`, `tan`, `atan`, `asin`,
`acos`, `log`, `sqrt`, `fabs`). Wiring is by `init()`:

- **Tests and spec runner always use C libm** (`value`/`util`/`functions`
  `math_cgo_test.go`, `spec/math_cgo.go` — no build tags), so all gates
  assert exact Dart parity. Without this, Go's native `math.Tan` diverges
  from C libm by 1 ULP (`tan(45deg)` = `...998` vs `...999`).
- **Shipped binaries always use C libm.** `cmd/go-sass/math_cgo.go`
  (`//go:build cgomath`) wires `sassmathcgo` into the CLI, and every
  production build — the release.yml matrix, `npm run build`, CI — passes
  `-tags cgomath`. A plain `go build` (no tags) is hermetic stdlib math
  for local iteration only; on libm-sensitive inputs it may differ from
  Dart in the last ULP (e.g. Go's native `math.Tan` diverges from C libm
  by 1 ULP: `tan(45deg)` = `...998` vs `...999`). Tests and the spec
  runner always use C libm (`value`/`util`/`functions`
  `math_cgo_test.go`, `spec/math_cgo.go` — no build tags), so all gates
  assert exact Dart parity.

## Floating-point parity model

Spec golden CSS depends on IEEE-754 double semantics of every operation.
Four conditions must hold:

1. **C-libm identity** (above) for all transcendental calls.
2. **FMA prevention.** Matrix multiplications round each product explicitly
   (`float64(m[i]*v) + …`) — a fused multiply-add rounds once instead of
   twice and diverges at the last ULP.
3. **Verbatim Dart formulas where Dart is lossy.** `normalizeHue` uses
   `(h % 360 + 360) % 360` exactly (the intermediate `+360` loses ~1 ULP for
   positive hues — replicated, not "fixed"); cubing uses `x*x*x`, not
   `Pow(x, 3)` (Dart's integer-`pow` path uses repeated multiplication, while
   C's `pow(x, 3.0)` takes the exp/log path — 1 ULP apart on large inputs).
4. **Golden `to_bits`-style tests.** Tests assert raw captured values from
   Dart/C libm, so accidental libm/FMA drift fails loudly rather than as a
   1-ULP CSS diff.

## Integer-valued builtins (`ceil`/`floor`/`round`)

Dart's `num.ceil()`/`floor()`/`round()` return 64-bit ints: out-of-range
finite values saturate at ±2⁶³∓1 and non-finite input throws. The Sass
builtins replicate this, returning the `i64` extreme as an f64 sentinel. No
f64 holds 2⁶³−1 exactly, so the serializer special-cases the sentinel f64s
back to the exact digit strings `9223372036854775807` /
`-9223372036854775808`, matching Dart's `integer.toString()`.

## Debugging a last-digit mismatch

Build standalone Go and Dart reproducers that inline the conversion pipeline
with hardcoded inputs, printing every intermediate value in full-precision
hex; compare to find the exact divergence point, then minimize to the single
operation. Guessing from the output digit alone almost never identifies the
stage.

## File mapping

| Dart                              | Go                      |
| --------------------------------- | ----------------------- |
| `lib/src/util/number.dart` (math) | `sassmath/*.go`         |
| (C libm, `math.h`)                | `sassmath/sassmathcgo/` |
