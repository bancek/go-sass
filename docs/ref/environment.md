# Modules: `sassenv/`, `sassmodule/`, `configuration/`, `evalcontext/`

The scoping and module system: `Environment` (variable/function/mixin scopes),
`Module` (the compiled result of a stylesheet), `Configuration` (`@use ...
with` values), and `EvaluationContext` (the warning-span chain threaded
through built-in callables).

## Environment

```go
// sassenv/environment.go
variables     []*orderedmap.LinkedMap[string, value.Value]
variableNodes []*orderedmap.LinkedMap[string, sasscommon.AstNode]
functions     []*orderedmap.LinkedMap[string, sasscallable.Callable]
mixins        []*orderedmap.LinkedMap[string, sasscallable.Callable]
// ... modules, namespace nodes, global/imported/forwarded modules,
// nested-forwarded, content, inMixin, semi-global state
```

Parallel scope stacks of insertion-ordered maps (Sass semantics are
order-sensitive).

- **`closure()`** shares frame 0 with the parent — a `!global` write to frame
  0 is visible to the caller. Module maps travel by reference (like Dart's
  shared maps); the configurable set is detached in nested contexts; `inMixin`
  resets.
- **`forImport()`** shares the scope chains but isolates the module maps for
  the legacy-`@import` world.
- Every stored AST node is only ever `.Span()`-ed, so node tables store spans
  (cheap to copy) directly.
- **`Scope[T]`** is a free generic function — `Scope(e, cb, semiGlobal, when)`
  — because Go forbids generic methods. It saves/restores semi-global state,
  optionally pushes/pops a frame, and invalidates lookup caches on exit.
- **`content`** holds the `@content` block, set and restored around mixin
  application.

## Module

```go
// sassmodule/module.go
type Module interface {
    URL() (string, error)
    Variables() orderedmap.Map[string, value.Value]
    Functions() / Mixins() orderedmap.Map[string, sasscallable.Callable]
    ExtensionStore() extend.ExtensionStore
    CSS() (*value.CssStylesheet, error)
    CloneCss() (Module, error)
    // … upstream, variable nodes, comments, configuration queries
}
```

Identity is by reference — there is no ID counter in Dart either. `Module`
and `Callable` are immutable value objects. `ForwardedModuleView` /
`ShadowedModuleView` (`module_forwarded_view.go`, `module_shadowed_view.go`)
hold an inner module with precomputed, prefix/show/hide-filtered member
views; `variableIdentity` delegates through the view to the originating inner
module. The environment-module assembly merges frame-0 variables with
forwarded members, mapping each variable to its originating module.

## Configuration

```go
// configuration/configuration.go
type Configuration struct {
    store                 valueStore
    originalConfiguration *Configuration
    isExplicit            bool
    NodeWithSpan          sasscommon.AstNode
}
```

`isExplicit` marks `@use ... with` configs (vs ambient implicit ones);
`NodeWithSpan` carries the rule node for error attribution. `SameOriginal`
is identity on the shared inner — configs derived from one base share it,
independently created ones do not. `ConfiguredValue`
(`configured_value.go`) pairs a value with its configuration and assignment
spans.

## Evaluation context (warning-span resolution)

Unlike the warning-span fields that other ports inline into evaluator state,
Go has an explicit struct (`evalcontext/evaluation_context.go`):

```go
type EvaluationContext struct {
    Logger       sasslogger.Logger
    callableNode sasscommon.AstNode
    importSpan   sasscommon.FileSpan
    defaultWarnSpan sasscommon.FileSpan   // fallback (stylesheet span)
    // …
}
```

`warnSpan()` falls back through `importSpan → callableNode →
defaultWarnSpan`, matching Dart's `_EvaluationContext` chain. It is threaded
as an explicit parameter through built-in callables (the replacement for
Dart's zones — see `architecture.md` §7). Deprecation warnings route through
dedup/quiet-deps handling (see `eval/`); otherwise they fall back to
`Logger.WarnDeprecation`.

## `fromOneModule`

Variable and callable lookup across `nestedForwarded → imported → global`
modules, with identity-based conflict detection in the global phase (a
differing identity produces a `MultiSpan` "available from multiple global
modules" error).

## `importForwards`

At root, `importForwards` shadows conflicting members and copies forwarded
modules into imported/forwarded sets; when nested, it appends to nested
forwarded modules; in both cases it removes now-shadowed locals.

## File mapping

| Dart                                                  | Go                                  |
| ----------------------------------------------------- | ----------------------------------- |
| `lib/src/environment.dart`                            | `sassenv/environment.go`            |
| `lib/src/module.dart`, `module/*.dart`                | `sassmodule/*.go`                   |
| `lib/src/configuration.dart`, `configured_value.dart` | `configuration/*.go`                |
| `lib/src/evaluation_context.dart`                     | `evalcontext/evaluation_context.go` |
