# Patterns and conventions

This document records the translation patterns used across the port and the
conventions that every contributor should follow. It is written from the
perspective of the code as it exists — "what" the conventions are and "why"
they exist — not as a porting tutorial.

## 1. Design philosophy

Two principles govern the whole codebase:

- **Dart is the source of truth.** Where Dart and Go disagree, Dart wins.
  Never preserve a Go simplification that changes observable behavior; port the
  Dart logic exactly, including its quirks (see `divergences.md` for the
  deliberate exceptions).
- **Structural fidelity.** The Go code mirrors the Dart structure, not merely
  its output. If Dart serializes through `SerializeVisitor`, Go does too — no
  inlined simplifications. This keeps the port auditable against upstream and
  keeps `// dart-source:` annotations meaningful.

A corollary, RULE ZERO: never write a comment claiming Go is "equivalent" to
Dart. Either the code matches statement-for-statement (then no comment is
needed), or the difference is an intentional, documented divergence (then it
belongs in `divergences.md`).

## 2. Source annotations

Every ported file carries a header annotation naming its origin:

```go
// dart-source: lib/src/exception.dart
```

For external packages or multiple sources:

```go
// dart-source: (external) package:source_span/lib/src/file.dart + lib/src/util/span.dart
```

`// Matches Dart:` marks a specific symbol or block that was cross-checked
against Dart:

```go
// CompileString compiles a Sass source string and returns the CSS result.
//
// Matches Dart: compileString
```

One `// dart-source:` per file (enforced by audit: any file with two
annotations is a split waiting to happen). Where the mapping is not a bare 1:1
path, a **parenthetical qualifier** documents the relationship in place, e.g.
`(interface only)`, `(errors)`, `(external)`, `(test helper)`,
`(not present in Dart)`.

**Why:** these annotations are the reverse index used to back-port upstream
dart-sass changes. `porting.md` builds a changed-Dart-file → Go-file mapping
from them (match on the **full `lib/src/...` path**, never the basename), and
the license-header policy relies on every annotation resolving to a real Dart
source. They are a live, maintained part of the code — not archaeology.

## 3. Type-system translation

| Dart                                | Go                                                              |
| ----------------------------------- | --------------------------------------------------------------- |
| `abstract interface class X`        | `type X interface` + `IsXxx()` marker                           |
| `final class X extends Y`           | `type X struct` embedding `Y` anonymously                       |
| `sealed class` (same package)       | interface + unexported marker                                   |
| `sealed class` (cross-package)      | interface + **exported** `IsXxx()` marker                       |
| `enum X { a, b }`                   | `type X int` + `const` iota + `String()`                        |
| `enum X { a(v), b(w) }` with fields | `type X int` + `String()` switch                                |
| `factory X(...)` / constructors     | `func NewX(...) *X`                                             |
| named/optional params (`{this.ns}`) | explicit `*T` parameter, `nil` when absent                      |
| `T?` nullable                       | `*T` pointer (`*string`, `*Expression`, …)                      |
| `List<T>`                           | `[]T` (copied on construction for immutability)                 |
| `Map<K, V>` (ordered)               | `*orderedmap.LinkedMap[K, V]` / `orderedmap.Map` interface      |
| `Set<T>`                            | `map[T]struct{}` or `LinkedHashSet[T]`                          |
| `get prop`                          | `func (t T) Prop() ReturnType`                                  |
| `operator ==`                       | `Equal()` method (never `==`, never `reflect.DeepEqual`)        |
| `get hashCode`                      | cached `*int` hash where Dart caches (perf parity)              |
| `toString()`                        | `String() string`                                               |
| `Uri`                               | `*url.URL` from `net/url`                                       |
| `StringBuffer`                      | `strings.Builder` (`WriteRune`, not `WriteByte`, for non-ASCII) |
| `Box<T>` / `ModifiableBox<T>`       | `box.Box[T]` / `box.ModifiableBox[T]`                           |

**Marker methods must be exported across packages.** An interface satisfied from
another package needs an exported marker (`IsAstNode()`, `IsSelector()`,
`IsSimpleSelector()`); unexported markers work only when all implementations
live in the same package. Concrete structs embed bases **anonymously** so
getters and markers promote; a named field would need forwarding methods.

**Field vs. method naming:** if an interface requires `Foo()` and the struct
stores backing data, name the field `foo` (unexported) and implement `Foo()`
as a getter. Go forbids a field and method sharing a name.

**`nil` vs empty:** like Dart, distinguish absent from empty where it matters
(`nil []Statement` vs `[]Statement{}` is no-block vs empty `{}` block;
`Namespace *string` `nil` vs `""` is default vs none).

## 4. Dispatch

| Dart                                 | Go                                            |
| ------------------------------------ | --------------------------------------------- |
| `x.accept(v)` returning `T`          | `x.AcceptValue(v)` with `ValueVisitor[T]`     |
| predicate traversal                  | `x.AcceptBool(v)` with `Visitor[bool]`        |
| effect traversal                     | `x.AcceptVoid(v)` with `Visitor[struct{}]`    |
| `ReplaceExpressionVisitor`           | `x.AcceptExpr(v)` returning `Expression`      |
| `switch` with destructuring patterns | manual `if` chains on lengths/nil             |
| `is` / `as`                          | comma-ok assertion, via `any` across packages |

Visitor interfaces are generic (`ValueVisitor[T]`, `ExpressionVisitor[T]`,
`StatementVisitor[T]`, `CssVisitor[T]`, `SelectorVisitor[T]`); every method
returns `(T, error)`. Concrete result types in use: `struct{}` (serialize and
all effect visitors), `bool` (predicates), `Expression` (replace), `any`
(general traversal).

Cross-package assertions go through `any` first — types in sub-packages may
not satisfy the parent interface at compile time, so a direct assertion is a
compile error:

```go
if _, ok := any(ifFalse).(*NullExpression); ok {
```

## 5. Ownership and structural patterns

| Dart (GC)                         | Go                                                                                                                        |
| --------------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| shared references                 | pointers and slices (also GC — same shape)                                                                                |
| `Map` with insertion order        | `*orderedmap.LinkedMap[K, V]`                                                                                             |
| lazy member views                 | `MergedMapView` / `PublicMemberMapView` / `LimitedMapView` / `PrefixedMapView` (lazy, insertion-ordered, no eager copies) |
| `Box<T>` shared-mutable reference | `*Box[T]` read view + `ModifiableBox.Value` writes                                                                        |
| `[]Concrete` → `[]Interface`      | convert with a cached slice, invalidate on mutation                                                                       |
| generic method (e.g. `scope<T>`)  | free generic function `Scope[T](e, cb, semiGlobal, when)` — Go forbids generic methods                                    |
| `Future`/async twins              | not ported — sync only                                                                                                    |
| zones / ambient context           | explicit `*EvaluationContext` parameter                                                                                   |

`Scope` callbacks return `(T, error)` — never smuggle the error as `any`.
`when` conditions are computed (`HasDeclarations(children)`), never hardcoded.

**No embedded-base polymorphism:** when embedding keeps the wrong receiver
(a method on the base that must dispatch to the outer type), use a named field
plus copy methods and free functions instead of anonymous embedding.

## 6. Value serialization naming

| Dart / intent                         | Go                         |
| ------------------------------------- | -------------------------- |
| `toString()` (display, list parens)   | `String()`                 |
| `meta.inspect` (raw serializer)       | `SerializeValueInspect(v)` |
| CSS output (errors on non-CSS values) | `ToCssString(quote)`       |

These three are distinct and must not be conflated. `ToCssString` returns
`(string, error)` — in CSS mode maps, functions, mixins, and empty unbracketed
lists error.

## 7. Error handling

| Dart                              | Go                                                            |
| --------------------------------- | ------------------------------------------------------------- |
| `throw`                           | `return ..., err` (errors are values, never panics)           |
| `try on T catch`                  | `errors.AsType[*T](err)` (never a direct assertion)           |
| `SassScriptException` (unspanned) | `NewSassScriptException(...)`                                 |
| `SassException` + span/trace      | `SassRuntimeException` / `SassFormatException` / `MultiSpan*` |
| `throwWithTrace(err, cause)`      | `ThrowWithTrace(new, cause)` preserving `Cause`/`LoadedUrls`  |
| `on SassException` boundary       | the four spanned variants only — never unspanned `Script`     |

Rules: never extract `wrapX`/`formatX` error helpers — inline
`if err != nil` + `AsType` + `ThrowWithTrace` at each site so the span source
stays visible. `WithAdditionalSpan` / `WithLoadedUrls` must preserve
`Cause`/`LoadedUrls` when rebuilding. Parser errors chain
`ScanError → SourceSpanFormatException → SassFormatException`, and backtracking
is an `err != nil` check. `panic` is only for unreachable code and type
switches; everything else is `ArgumentError`/`StateError`/`RangeError` with an
`error` return. Span-library constructors are all fallible
(`Expand`/`Subspan`/`MapSpan` return `(…, error)`); propagate, never substitute
a bogus span.

Errors are asserted **exactly** — never bare `err != nil` without checking the
message or type. Tests match on the exception type and message text.

## 8. Testing

### Golden values

For behavior whose correctness is a specific string or number, capture the
real Dart output and hardcode it — use Dart as the oracle, one value per line,
including NaN/Infinity/empty edge cases. Never recompute the expected value
from the same formula as the code under test.

### Coverage

Implementation and tests should have comparable line counts — a 1000-line
implementation needs roughly 1000 lines of test code. Every happy path, error
path, and syntax variant needs explicit coverage. Constructor validation
(`NewSassList`, `NewSassArgumentList`, `AsList` returning `error`) is tested
through its error returns.

### Optional scalars (Go 1.26)

`new("lit")` yields `*string` directly — use it for optional scalar arguments
instead of a temporary variable. Literals only.

## 9. Logger and deprecation

`Deprecation` constants are defined once in `deprecation/`; every package that
can warn carries its own narrow API surfacing through `EvaluationContext.Warn`
/ `WarnDeprecationFromApi` (the JS-outside-eval path) and the `WarnLogger`
buffering used where the evaluator is unreachable. The stack trace is carried
**structurally** as `*Trace`, never as a pre-formatted string; stringification
happens only at the render boundary (`StderrLogger`, embedded). See
`ref/logger.md` and `ref/warn-logger.md`.

## 10. Importer and module shape

| Dart                                       | Go                                                                                  |
| ------------------------------------------ | ----------------------------------------------------------------------------------- |
| `Importer` interface                       | `Importer` interface (`Filesystem`, `NoOp`, `User`, `Package`, `NodePackage` impls) |
| `Module` interface (identity by reference) | `sassmodule.Module` interface (`URL`, `Upstream`, member views, `CSS`, `CloneCss`)  |
| `Environment` scopes                       | `[]*orderedmap.LinkedMap` parallel stacks                                           |

`Module.Variables/Functions/Mixins` return the `orderedmap.Map` interface (not
a concrete type, not a builtin map) so forwarded/shadowed views compose.
`SetVariable` writes into the environment's frame 0 for `!global` visibility.
