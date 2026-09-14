# Architecture

This document describes how `go-sass` is structured and why the design
decisions were made. It is organized from the compiler's core data structures
outward.

## 1. The pipeline

Compilation is a three-stage pipeline, each stage a distinct set of modules:

```
source text ──parse──▶ Stylesheet ──eval──▶ modifiable CSS tree ──serialize──▶ CSS string
```

1. **Parse** (`value/parse_*.go`) turns SCSS, indented Sass, or plain CSS into an AST
   (`*value.Stylesheet`) of statements and expressions.
2. **Eval** (`eval/`) walks that AST, resolving variables, imports, mixins,
   functions, and `@extend`, and produces a mutable CSS output tree
   (`value.ModifiableCssParentNode` and friends).
3. **Serialize** (`value/visitor_serialize.go`) walks the output tree and writes the final CSS
   string (and an optional source map).

Memory is garbage-collected: AST nodes, spans, and Sass values are ordinary
heap objects shared by reference, exactly like Dart. There is no arena, no
reference counting, and no lifetime annotation anywhere in the port.

## 2. `Value`: an interface with concrete structs

Dart's `Value` is an abstract class with subclasses; Go models it as an
interface with one struct per Sass type:

```go
type Value interface {
    AstNode
    IsValue()
    // ... String(), Equal(), hash key, accept methods
}
```

Concrete types include `SassBoolean` (two singletons), `SassNull` (singleton),
`SassString`, `SassNumber` (itself an interface with `Unitless` / `SingleUnit` /
`ComplexUnit` implementations sharing `sassNumberBase`), `SassColor`,
`SassList`, `SassMap`, `SassArgumentList` (embeds `*SassList` plus `Keywords`),
`SassCalculation`, `SassFunction`, and `SassMixin`.

**Why an interface, not a struct:** the value set is open to host-defined
behavior in Dart (and the evaluator dispatches on dynamic type constantly), so
a Go interface with type switches mirrors Dart's `is`/`as` directly. **Value
equality is never `==`:** Dart overrides `operator==` per type (ignoring spans
and metadata), so every value type implements `Equal()` replicating that logic,
and `SassMap` is backed by a `LinkedHashMap[Value, Value]` keyed with the
`ValueEquals` comparator. Using `==` or `reflect.DeepEqual` on a value is a bug
(see `patterns.md` and `critical-invariants.md`).

## 3. AST: interfaces + embedding

| Dart                           | Go                                             |
| ------------------------------ | ---------------------------------------------- |
| `abstract interface class X`   | `type X interface` with an `IsXxx()` marker    |
| `final class X extends Y`      | `type X struct` embedding the base struct      |
| `enum X { a, b }`              | `type X int` + `const` iota block + `String()` |
| `sealed class` (same package)  | interface + unexported marker                  |
| `sealed class` (cross-package) | interface + **exported** `IsXxx()` marker      |

Marker methods must be exported when the interface is implemented from another
package: `ast.AstNode` requires `IsAstNode()`, `Selector` requires
`IsSelector()`, and so on. Concrete structs embed their bases anonymously
(`SelectorBase`, `SimpleSelectorBase`, `ParentStatement`, `callableBase`,
`expressionBase`) so getters and markers promote automatically; a named field
would need explicit forwarding methods.

Statements (27 visit methods), expressions (18), CSS nodes (9), and selectors
each have their own visitor interface in `value/` (`sass_statement_visitor.go`,
`sass_expression_visitor.go`, `css_visitor.go`, `selector_visitor.go`, …).

## 4. Visitors: generic interfaces, one accept per result type

Each AST family has a generic visitor interface whose type parameter selects
the result — the same traversal serves many result types:

```go
type ValueVisitor[T any] interface {
    VisitBoolean(*SassBoolean) (T, error)
    VisitNumber(SassNumber) (T, error)
    // ... one method per value type
}
type ExpressionVisitor[T any] interface { /* 18 methods */ }
type StatementVisitor[T any] interface { /* 27 methods */ }
type CssVisitor[T any] interface { /* 9 methods */ }
type SelectorVisitor[T any] interface { /* ... */ }
```

Concrete visitors instantiate `T`: `SerializeVisitor` is `ValueVisitor[struct{}]`,
`IsPlainCssVisitor` is `ValueVisitor[bool]`, `ReplaceExpressionVisitor` is
`ExpressionVisitor[Expression]`. Nodes expose one accept method per result
shape used in practice:

```go
func (m *SassMap) AcceptVoid(v ValueVisitor[struct{}]) (struct{}, error) { return v.VisitMap(m) }
```

`AcceptValue` is the general traversal, `AcceptBool` serves predicate visitors
(`IsPlainCss`, `IsCalculationSafe`), `AcceptVoid` serves effect visitors
(including all of serialization), and `AcceptExpr` serves
`ReplaceExpressionVisitor`. The evaluator itself is **not** a visitor: it
dispatches through `EvaluateVisitor` methods and free functions with explicit
`switch` on dynamic type. See `ref/visitors.md`.

Cross-package assertions need an `any` hop: types in sub-packages may not
satisfy the parent interface at compile time, so `any(x).(*T)` is the house
pattern (a direct assertion is a compile error under Go ≥1.22 rules).

## 5. Ownership: GC references, no arena

Everything is an ordinary Go reference. Consequences that differ from broader
Go practice but mirror Dart:

- Nullable Dart fields become pointers (`Namespace *string`, `Rest *Expression`,
  `*ContentBlock`); absent optionals are `nil`.
- Shared mutable state (environment frames, the CSS parent chain, module
  caches) is shared by pointer. The CSS tree caches derived slices
  (`childCache`) and invalidates on mutation (`AddChild`, `ClearChildren`).
- `Box[T]` / `ModifiableBox[T]` (`box/`) ports Dart's pair: a `*Box[T]` field
  holds the read view while the owner mutates through `ModifiableBox.Value`,
  so (e.g.) the extension store can swap a selector in place.

There is intentionally no `sync.Pool`, no `unsafe`, and no manual memory
management: the compiler is single-threaded per compilation and shares Dart's
allocation shape, which the GC handles the way the Dart VM does.

## 6. Sync only (locked)

Dart ships twin sync/async implementations (`evaluate.dart` /
`async_evaluate.dart`, …) generated from the async source. Go ports **only the
synchronous behavior**: one evaluator, blocking I/O, no goroutines in the
compile path. The embedded server multiplexes compilations with one goroutine
per compilation instead (see `ref/embedded.md`). There is deliberately no
async port to maintain.

## 7. Evaluator: `EvaluateVisitor` + `EvaluationContext`

The evaluator is a single struct holding everything a compilation needs:

```go
type EvaluateVisitor struct {
    ec          *evalcontext.EvaluationContext
    env         *sassenv.Environment
    importCache *ImportCache
    logger      sasslogger.Logger
    modules     map[string]sassmodule.Module
    // ... built-ins, stack frames, parent chain, media/at-root state,
    // declaration flags, loaded URLs (see eval/evaluate.go)
}
```

Dart's zone-scoped state becomes explicit parameters: the evaluation context
(`evalcontext/`) is threaded through built-in callables instead of ambient
zones, and `@import`/`@use` state lives on the visitor. Import-cache ownership
flows through the visitor rather than by borrowing. `ref/eval.md` documents the
evaluation semantics (rest-argument dispatch, `!global` writes, rule-child
scoping, calculation arguments) that this structure serves.

## 8. Serializer: `SerializeVisitor` + buffer

Serialization walks the frozen CSS tree with a `SerializeVisitor` over a
source-map-aware buffer (`sourcemapbuffer/`). `AcceptVoid` is the traversal:
every `Visit*` method returns `(struct{}, error)` because no visitor method is
infallible (see `critical-invariants.md`). `ToCssString(quote)` on values routes
through the same serializer and errors on non-CSS values; `String()` is Dart's
`toString()` (with list parens); the two must never be conflated. See
`ref/serialize.md`.

## 9. CSS output tree: immutable nodes + modifiable wrappers

The CSS tree has two layers, mirroring Dart:

- **Immutable** `CssXxx` nodes (the serialized output).
- **Mutable** `ModifiableCssXxx{Inner *CssXxx}` wrappers implementing the same
  `CssNode` / `ModifiableCssParentNode` interfaces, used during evaluation and
  `@extend`.

Parents own children (`[]ModifiableCssNode`); identity for module/callable
tracking is by reference. `EqualsIgnoringChildren` / `CopyWithoutChildren` live
on the parent-node interface (with panic stubs on the embedded base) because
Dart declares them abstract. At the end of evaluation the tree freezes to
immutable nodes for serialization. See `ref/ast.md`.

## 10. Module system and environment

```go
// sassenv/environment.go
variables []*orderedmap.LinkedMap[string, value.Value]
// ... variableNodes, functions, mixins: parallel scope stacks

// sassmodule/module.go
type Module interface {
    Variables() orderedmap.Map[string, value.Value]
    Functions() orderedmap.Map[string, sasscallable.Callable]
    // ... mixins, extension store, CSS, configuration
}
```

`Environment` keeps parallel scope stacks as `[]*LinkedMap` (insertion-ordered:
Sass semantics are order-sensitive), which gives `!global` shared-frame
semantics — frame 0 is shared with the caller. `Scope[T]` is a free generic
function (`eval` + `when` conditions like `HasDeclarations(children)`), because
Go forbids generic methods. Module member views (`MergedMapView`,
`PublicMemberMapView`, `LimitedMapView`, `PrefixedMapView` in `orderedmap/`)
are lazy and insertion-ordered; there are no eager copies. Callable and module
identity is by reference. See `ref/environment.md` and `ref/member-map.md`.

## 11. Error system

Errors are values, never panics. The taxonomy mirrors Dart:

- `SassScriptException` — unspanned value/type errors raised inside callbacks.
- `SassRuntimeException` / `SassFormatException` / `SassException` /
  `MultiSpan*` — spanned errors carrying message, span, `Trace`, cause, and
  loaded URLs.
- Wrapping is per-visit and callback-based: `addExceptionSpan` turns a `Script`
  error into a spanned `Runtime` with stack trace; `addErrorSpan` handles
  `@error`; `invokeCallable` catches non-Sass errors. There is deliberately no
  blanket wrapper in `accept()` — each visit site must match Dart.
- Chaining uses `Cause`/`Unwrap()` + `ThrowWithTrace`, inspected with
  `errors.AsType[*T]` (never a direct type assertion).

The parser layers its own errors — `ScanError` → `SourceSpanFormatException` →
`SassFormatException` — so zero-length spans can be adjusted before conversion.
`Trace` is a structured frame list (`sasscommon`), stringified only at the
output boundary. See `ref/common.md` and `ref/parse.md`.

## 12. I/O and importers

`sassio.IO` is the host-abstraction seam (filesystem reads, existence checks,
canonicalization, working directory). Four rules govern its use: never read the
ambient working directory (always `io`); build `file:` URLs via path helpers,
never string formatting; thread `io` through the compile options and evaluator
rather than per-function globals; and spans render through the `io`-aware
highlighter because there is no ambient cwd. `UserImporter` is the extension
point a host implements; `FilesystemImporter`, `PackageImporter`, and
`NodePackageImporter` are built in. See `ref/io.md` and `ref/importer.md`.

## 13. Package layout and import cycles

| Package                                                                                                   | Purpose                                                                                       |
| --------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| [`compile/`](../compile/)                                                                                 | The public compile API (`CompileString`, `Compile`, `CompileStylesheet`).                     |
| [`cmd/go-sass/`](../cmd/go-sass/)                                                                         | The `go-sass` command-line binary (flags, usage, multi-file runner).                          |
| [`value/`](../value/)                                                                                     | Values, AST (Sass/CSS/selectors), parser, and visitors — monolithic by necessity (see below). |
| [`eval/`](../eval/)                                                                                       | The evaluator, import cache, and importers.                                                   |
| [`evalcontext/`](../evalcontext/)                                                                         | The evaluation context threaded through built-in callables.                                   |
| [`extend/`](../extend/)                                                                                   | The `@extend` store.                                                                          |
| [`functions/`](../functions/)                                                                             | Built-in Sass function registries.                                                            |
| [`sasscommon/`](../sasscommon/)                                                                           | Errors, spans, source locations, the scanner.                                                 |
| [`sassio/`](../sassio/)                                                                                   | The host I/O abstraction.                                                                     |
| [`sasslogger/`](../sasslogger/)                                                                           | Loggers and deprecation processing.                                                           |
| [`sassmodule/`](../sassmodule/) + [`sassenv/`](../sassenv/)                                               | Modules and environments.                                                                     |
| [`sasscallable/`](../sasscallable/) + [`sassclonecss/`](../sassclonecss/)                                 | Callables and CSS cloning.                                                                    |
| [`sassmath/`](../sassmath/) (+ `sassmath/sassmathcgo/`)                                                   | Math wrappers with optional C-libm backend (`-tags cgomath`).                                 |
| [`sassurl/`](../sassurl/)                                                                                 | Dart-compatible URL resolution.                                                               |
| [`sourcemap/`](../sourcemap/) + [`sourcemapbuffer/`](../sourcemapbuffer/)                                 | Source maps.                                                                                  |
| [`embedded/`](../embedded/)                                                                               | The `sass-embedded` protocol server (spawned with `--embedded`).                              |
| [`embedded-host-node-go/`](../embedded-host-node-go/)                                                     | The published `sass-embedded-go` npm host package.                                            |
| [`configuration/`](../configuration/) + [`deprecation/`](../deprecation/)                                 | `@use` configuration and deprecation catalog.                                                 |
| [`orderedmap/`](../orderedmap/) + [`linkedhashmap/`](../linkedhashmap/) + [`orderedset/`](../orderedset/) | Insertion-ordered collections.                                                                |
| [`util/`](../util/) + [`unvendor/`](../unvendor/) + [`termglyph/`](../termglyph/) + [`box/`](../box/)     | Small shared helpers.                                                                         |
| [`spec/`](../spec/)                                                                                       | The runner for the official `sass-spec` test suite.                                           |

Dart's `lib/src/` is a single library — any file may reference any other. Go
enforces a DAG between packages, and Dart's mutually referential patterns
(visitor interfaces ↔ concrete types, serializer ↔ value types, parser ↔ Sass
AST) would form import cycles as separate packages. The layout resolves this
by **co-location**:

- **`value/` is monolithic by necessity** (228 non-test files): all of Dart's
  `ast/sass/`, `ast/css/`, `ast/selector/`, `value/`, `parse/`, and `visitor/`.
  Domains are distinguished by filename prefix (`parse_*`, `sass_*`,
  `selector_*`, `visitor_*`, `css_*`, `value*`, `color*`, `number*`).
- Everything else maps 1:1 to its Dart directory: `eval/`, `compile/`,
  `configuration/`, `evalcontext/`, `sasslogger/`, `sassio/`, `sourcemap/`,
  `extend/`, `functions/`, `util/`, `unvendor/`.
- Cycle-break extractions: `sasscommon/` (spans, errors, scanner — merged from
  several Dart utils), `sassmodule/`, `sassenv/`, `sasscallable/`,
  `sassclonecss/`, `orderedmap/` (+ `linkedhashmap/`, `orderedset/`), `box/`,
  `sassmath/` (+ `sassmath/sassmathcgo/`), `sassurl/`, `termglyph/`,
  `sourcemapbuffer/`, `deprecation/`, `configuration/`.

The rule for new code: put it in the existing package that keeps the import
graph acyclic. Never split `value/` apart and never merge cycle-split packages
back together — either reintroduces the cycle. See `patterns.md`.

## 14. Embedded and CLI

Two consumer-facing surfaces are built on the compiler:

- **`embedded/`** serves the Sass embedded protocol (version 3.2.0,
  compiler version 1.104.0, implementation name `"dart-sass"`) over stdio. See
  `ref/embedded.md`.
- **`cmd/go-sass/`** is the `go-sass` command-line compiler (options, usage, multi-file
  and directory runner) with its own [README](../cmd/go-sass/README.md). It carries
  every Dart CLI flag except the unimplemented `--watch` / `--poll` /
  `--update` / `--interactive`, and an opt-in `-tags cgomath` build that swaps
  the math backend to C libm for exact Dart matching (see `ref/math.md`).
