# Module: `sasscallable/`

The runtime representation of Sass functions and mixins: the `Callable`
interface, its implementations, overload resolution, and invocation.

## `Callable`

```go
// sasscallable/callable.go
type Callable interface {
    Name() string
    // … parameters, invocation, content acceptance
}
```

Callables are pure data — invocation is evaluator-driven (`invokeCallable`
switches on `BuiltIn` / `UserDefined` / `PlainCss`). Identity is by reference
(there is no ID counter in Dart either). Callables are stored in string-keyed
maps (by name) in the environment and the built-in registry, never as map
keys themselves. Identity is compared exactly once: `assertNoConflicts`
during `@forward` deduplication.

## The three implementations

### `BuiltInCallable` (`functions/callable_built_in.go`)

```go
// name, overloads (params + callback), AcceptsContent, deprecation warning
```

Overloads pair a `ParameterList` with a callback taking
`(ec *EvaluationContext, args …)` — the evaluation context, not the whole
visitor, so `meta.call`/`meta.apply`/`meta.load-css` can re-enter the
evaluator through the narrow seam. `WithName` / `WithDeprecationWarning` are
copy-on-write: they return a modified copy, never mutating the original
(this is what makes the global-alias wrapping in `functions/` safe).

### `UserDefinedCallable` (`functions/callable_user_defined.go`)

Declaration (mixin/function rule or content block) + captured environment +
dependency flag. `Name()` and parameters derive from the declaration
(Dart-exact). There is no `isMixin` field — call sites switch on the
declaration kind.

### `PlainCssCallable` (`functions/callable_plain_css.go`)

A bare name constructed on the fly when no user-defined or built-in function
matches, serialized as plain CSS `name(args...)`. It is never stored
persistently. Positional expressions evaluate on the fly to CSS strings; only
the "isn't a valid CSS value" failure wraps as a labeled `MultiSpan`.

## Overload resolution and argument flow

`CallbackFor(positional, named)` resolves overloads:

1. **Exact match** — the first overload whose parameter list matches the
   positional count and named keys wins.
2. **Fuzzy fallback** — pick the overload with the smallest parameter-count
   distance, preferring more parameters on ties.
3. **No match** — error.

The full argument flow is five steps: parse the `ArgumentList` → evaluate
positional and named expressions → resolve the overload → verify the
parameters → resolve defaults and pack the rest argument into a
`SassArgumentList`.

## Lookup priority

When the evaluator resolves a function or mixin name:

1. Environment scope chain → `UserDefinedCallable`.
2. Built-in registry (`builtInFunctions[normalized_name]`) → `BuiltInCallable`.
3. Plain-CSS context → a fresh `PlainCssCallable`.
