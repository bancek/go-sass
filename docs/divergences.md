# Intentional divergences

Every _behavioral_ divergence between this port and Dart Sass has been resolved:
the Go port produces byte-identical output to Dart Sass `1.104.0` on the
gate workloads.

This document records the differences that remain **on purpose**: places where
Dart and Go differ in implementation while producing identical results, plus
deliberate configuration leniencies. Do not "fix" any of these — they are
not bugs.

## Implementation differences with identical output

### 1. `PseudoSelector` boolean sense

`pseudo.IsSyntacticClass` is the inverse of Dart's
`pseudo.isSyntacticElement` (single-colon `:` vs double-colon `::` sense). The
naming is inverted but the serialized output is identical.

### 2. `WriteString` return discarded

`sb.WriteString()` returns `(int, error)`, discarded with `_, _ =`; Dart's
`buffer.write()` is void. A mechanical difference with identical output.

### 3. `utf8Decode`

Go strings are byte-indexed, so the scanner carries a manual `utf8Decode`
helper. Dart iterates code points natively (`string.codeUnitAt`), so no such
helper exists there. No output difference.

### 4. `hashCode` caching

Dart computes value hash codes on demand; Go caches them in a lazy field
(`cachedHash` on `SassMap`, and the equivalent on other value types) for map
performance. The cached value equals Dart's computation; only the memoization
is Go-side. Do not remove the caches.

### 5. Stdlib math without `-tags cgomath`

Unit tests, the spec runner, and every shipped binary (the release.yml
matrix `-tags cgomath` builds, `npm run build`) exercise the C-libm math
backend (`sassmathcgo`), so all gates and all releases assert exact Dart
parity. A plain `go build` without the tag yields Go stdlib math: on
libm-sensitive inputs (see `ref/math.md`) it may differ from Dart in the
last ULP. That untagged build is a local-iteration convenience only —
nothing ships it. Do not rewire the test/spec binaries to stdlib math.

## Not implemented (gaps, not divergences)

`--watch`, `--poll`, `--update`, and `--interactive` have no Go counterpart.
They are absent, not divergent — do not document their behavior here; the CLI
reference ([`cmd/go-sass/README.md`](../cmd/go-sass/README.md)) lists them as
unimplemented.

## Adding an entry

Conscious accepts land here instead of conforming to
Dart. Append a `### <short title>` section in this shape:

```md
### <Title>

Dart: <file:line> does X.
Go: <file:line> does Y instead.
Why accepted: <one-line reason — unreachable input, internal-only,
defensive leniency, or documented quirk preserved deliberately>.
Pins: dart <short>, go <short>.
```

Do not file message/span/trace/output divergences here — those are bugs.
This file is for behavior-neutral (or deliberately lenient) differences only.
