# Critical invariants

Rules that must not be violated anywhere in the port. Each rule states what it
is and why it exists.

## FMA prevention

Every `a*b + c` in color-conversion matrices must round each multiply
explicitly:

```go
// CORRECT — each product rounds before the addition
float64(m[0]*v0) + float64(m[1]*v1) + float64(m[2]*v2)

// WRONG — the compiler/CPU may fuse into one FMA and diverge by 1 ULP
m[0]*v0 + m[1]*v1 + m[2]*v2
```

**Why:** fused multiply-add rounds once instead of twice, diverging from Dart
(and the spec goldens) at the last ULP. `ref/math.md` documents the full
floating-point parity model, including the `sassmathcgo` C-libm backend, the
verbatim `normalizeHue` formula, and the `x*x*x` cubing rule.

## Nil vs empty

Go distinguishes `nil` slices from empty ones, and the port relies on it:
`nil []Statement` (no block, ends with `;`) is not `[]Statement{}` (an empty
`{}` block). The same applies to `*string` namespaces (`nil` = default,
`""` = none) and optional sub-ASTs (`nil` = absent). Never normalize `nil` to
empty (or vice versa) at an API boundary. This affects declarations, at-rules,
style/media/supports rules, and include-rule content.

## Value equality is `Equal()`, never `==`

Dart overrides `operator==` per value type (comparing logical value, ignoring
spans and metadata). Go struct `==` and `reflect.DeepEqual` compare every
field, including the ones Dart ignores — so they are **never** the right
comparison for ported types:

```go
// WRONG — compares spans and metadata too
if v1 == v2 { ... }

// RIGHT — replicates Dart's operator==
if v1.Equal(v2) { ... }
```

`SassMap` is backed by a `LinkedHashMap[Value, Value]` keyed with the
`ValueEquals` comparator for the same reason. Selector equality
(`QualifiedName`, compound/complex selectors) likewise goes through `Equal()`;
pointer comparison is insufficient (namespace pointers).

## No silent error discarding

All visitor methods return `(T, error)`. No visitor method is infallible.
`ToCssString()` and `SerializeValueInspect()` return `(string, error)` — in
CSS mode (`inspect = false`) maps, functions, mixins, and empty unbracketed
lists error. Never ignore an `error` return; never substitute a bogus span for
a propagated one (every span-library constructor is fallible for this reason).

## No panics in compiler code

Errors are values. `panic` is allowed only for provably unreachable code and
exhaustive type switches; every reachable failure — including constructor
validation (`NewSassList`, `NewSassArgumentList`, `AsList`), span operations,
and parser backtracking — returns `error`. Tests must never rely on a panic.

## The accept families

AST roots expose `AcceptValue` / `AcceptBool` / `AcceptVoid` / `AcceptExpr`
over the generic `ValueVisitor[T]` / `ExpressionVisitor[T]` /
`StatementVisitor[T]` / `CssVisitor[T]` / `SelectorVisitor[T]` interfaces. Do
not add a new accept shape when a new traversal is needed — instantiate `T`
(`struct{}` for effects, `bool` for predicates, `Expression` for replace,
`any` for general traversal). Do not delete the closed-world switches that
remain.

## Sealed types stay sealed

Interfaces that mirror Dart `@sealed` classes keep their `IsXxx()` marker
methods (exported cross-package, unexported within one package). Never remove
a marker to "simplify" implementation — the marker is what keeps the type set
closed and auditable.

## `Module` members are interfaces, not concrete maps

`Module.Variables/Functions/Mixins` return `orderedmap.Map` (interface), and
`Environment` stacks are `[]*LinkedMap`. Never replace these with builtin
`map` (loses insertion order — Sass semantics are order-sensitive) or with
concrete `*LinkedMap` in an interface position (breaks the lazy forwarded /
shadowed / limited / prefixed views). No eager copies of member maps.

## Structural `Trace`, never strings

`Trace` is a structured frame list carried through the evaluator and the
`Logger` seam; stringification happens only at the output boundary. There is
no string→trace parser — traces are never stringified mid-pipeline only to
be re-parsed.

## Exact Dart fidelity, statement-for-statement

- Match Dart's names, order of checks, and branch structure. A missing guard,
  swapped branch order, or combined check that changes which error fires first
  is a bug — even if each branch looks right alone.
- Preserve Dart's bugs exactly (e.g. lossy `normalizeHue`, `x*x*x` cubing).
  A "fix" that changes output is a divergence, not an improvement.
- Wrapping is per-visit (`addExceptionSpan` / `addErrorSpan` /
  `invokeCallable`), never a blanket wrapper in `accept()`. Each site must
  match Dart's span source, trace, and error variant.
- `Scope` `when` conditions are computed (`HasDeclarations(children)`), never
  hardcoded; `@each`/`@for` loop variables are defined at the documented
  expression nodes.

## Byte-identity, spec-green, protocol alignment

- Output is **byte-identical** to Dart Sass on the bootstrap,
  `huge`, and `huge10` workloads — CSS, source maps, warnings, and errors.
- The `sass-spec` suite must stay green: 14246/14252 (6 todo).
- `compilerVersion`/`protocolVersion` must track the pinned dart-sass version
  (`1.104.0` / `3.2.0`) in lockstep — see `CONTRIBUTING.md` version-bump
  checklist.

## Test standards

- **Exact error assertions.** Never assert bare `err != nil` without checking
  the exact type/message/span.
- **Golden values.** Hardcode captured Dart output; never recompute from the
  SUT formula (see `patterns.md` §8).
- **Coverage.** Implementation and test line counts should be comparable.
