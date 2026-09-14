# Module: `eval/`

The evaluator: AST in, CSS output tree out. It resolves variables, imports,
modules, mixins, functions, and `@extend`.

## `EvaluateVisitor`

All evaluation state lives on one struct (`eval/evaluate.go`), with logic in
its methods and in free generic helpers:

```go
type EvaluateVisitor struct {
    ec          *evalcontext.EvaluationContext
    env         *sassenv.Environment
    importCache *ImportCache
    logger      sasslogger.Logger
    modules     map[string]sassmodule.Module
    builtInFunctions map[string]sasscallable.Callable
    stack       *stackFrame
    parent      value.ModifiableCssParentNode
    // ... media-query + style-rule context, declaration flags,
    // import spans, loaded URLs, quiet/source-map flags (see evaluate.go)
}
```

Entry: `Evaluate(...)` builds the visitor (`NewEvaluateVisitor`), registers
user functions **before** built-ins (so built-ins take priority), sets flags,
and calls `v.Run(importer, stylesheet)` inside `withEvaluationContext`.

## Files

| File                                                                                    | Contents                                                                                       |
| --------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `evaluate.go`                                                                           | visitor struct, `Evaluate()`, `Run()`, `withEvaluationContext`, scope/save-restore combinators |
| `evaluate_statement.go`                                                                 | statement visitors (27 methods)                                                                |
| `evaluate_expression.go`                                                                | expression visitors (18 methods) + `invokeCallable`                                            |
| `evaluate_css.go`                                                                       | the CSS-output phase + `CssVisitor` bubbling                                                   |
| `evaluate_helpers.go`                                                                   | `addExceptionSpan`, `addErrorSpan`, `stackTrace`, span helpers                                 |
| `evaluate_meta.go`                                                                      | `meta.*` re-entrant functions (`call`, `apply`, `load-css`)                                    |
| `evaluate_result.go`                                                                    | `EvaluateResult`                                                                               |
| `eval_init.go`                                                                          | built-in registration                                                                          |
| `compat.go`, `syntax.go`                                                                | compat shims, syntax enum                                                                      |
| `import_cache.go`, `importer*.go`, `resolve_import_path.go`, `node_package_importer.go` | stylesheet resolution (see `importer.md`)                                                      |
| `imported_css_visitor.go`                                                               | the imported-CSS path                                                                          |

## Error flow

1. Type-check failures → `*sasscommon.SassScriptException` (unspanned).
2. `addExceptionSpan(v, node, callback, addStackFrame)` wraps a `Script`
   error with span + trace → spanned `Runtime`
   (`eval/evaluate_helpers.go:694`). It is generic over the callback result
   (`func addExceptionSpan[T any](...) (T, error)`).
3. `addErrorSpan` handles `@error` (respans a `Runtime` whose span text has
   the `@error` prefix); `invokeCallable` catches non-Sass errors from
   callbacks and wraps them as `Runtime`.
4. All wrapping is callback-based and **per-visit**: there is no helper that
   wraps an already-built error, and no blanket wrapper in `accept()`.
5. At the API boundary, `EmitErrorCss` decides whether to render the error
   as CSS (the four spanned variants only — never unspanned `Script`).

`stackTrace(span)` builds the `*sasscommon.Trace` from the visitor's
`stackFrame` chain; spans fall back to the stack top when absent.

## Evaluation semantics

- **`@return` short-circuit:** evaluating a block stops once a child returns
  a value (`handleReturn`); no `return_value` field on state.
- **`@at-root` scoping** saves/restores the parent chain, style-rule and
  media-query context, and keyframe/unknown-at-rule flags, mirroring Dart's
  `_scopeForAtRoot`.
- **`!global`** writes to frame 0 of the shared `[]*LinkedMap` stacks, so the
  write is visible to the caller.
- **Rest-argument dispatch:** the evaluated rest value is type-dispatched —
  maps → named args, argument lists → positional + keywords, lists →
  positional, anything else → a single positional value. Excess positionals
  and unmatched named args pack into a trailing `SassArgumentList`; unused
  keywords error (checked via keyword-access tracking).
- **Parameter verification** reports source spellings (`$foo_bar`, not
  `$foo-bar`) for missing/arity errors.
- **Namespaced callables** resolve inside `addExceptionSpan(node span)`, so a
  missing namespace reports Dart's `There is no module with the namespace
"ns".` at the call-site span.
- **Calculation arguments:** unary operations are rejected; unquoted strings
  pass the `is_calculation_safe` gate; `calc()` incompatibilities report the
  whole-operation span while multi-arg calls re-verify per-operand `MultiSpan`
  labels (e.g. `min(1px, 0s)`).
- **`withoutSlash`:** a `/`-as-division number emits a deprecation warning
  and strips the slash; applies to `@each` items and variable values, but not
  to modern `if()` branch values.
- **Rule-child scoping:** declaration/media/style-rule/keyframe/at-rule
  children scope with `HasDeclarations(children)` (never a hardcoded bool).
- **`loadModule`** owns the builtin check and wraps the callback in
  `addExceptionSpan`; `@import` takes a separate dedicated path
  (`_visitDynamicImport` equivalent) with `combineCss` and the
  `ImportedCssVisitor`, bypassing module caching.
- **Reentrant meta functions:** `meta.call` / `meta.apply` / `meta.load-css`
  re-enter the evaluator (`evaluate_meta.go`); `load-css` maps
  `$with: null` to an empty configuration, never the ambient one.

## CssVisitor bubbling

The CSS phase walks the parent chain with a `through` predicate: style rules
always bubble; media rules pass through only when their queries were merged.
When a target parent has a following sibling, the node is cloned first
(`CopyWithoutChildren`) so the sibling is not corrupted. See
`evaluate_css.go`.

## Function taxonomy

1. **Visitor methods** — the real logic: `func (v *EvaluateVisitor) visit*(…,
…) (T, error)`, dispatched via `switch` on dynamic type (the evaluator
   implements no Sass-AST visitor interfaces).
2. **Free generic helpers** — `addExceptionSpan[T]`, `addErrorSpan[T]`,
   `Scope[T]` (in `sassenv/`): shared combinators that need no visitor.
3. **Constructors and entry points** — `NewEvaluateVisitor`, `Evaluate`,
   `Run`, `RegisterUserFunctions`, `SetQuietDeps`, `SetSourceMap`,
   `SetNodeImporter`.

## File mapping

| Dart                                                                        | Go                                                                                                                               |
| --------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| `lib/src/visitor/evaluate.dart`, `async_evaluate.dart` (sync behavior only) | `eval/evaluate.go`, `evaluate_statement.go`, `evaluate_expression.go`, `evaluate_helpers.go`, `evaluate_meta.go`, `eval_init.go` |
| `lib/src/import_cache.dart`                                                 | `eval/import_cache.go`                                                                                                           |
| `lib/src/visitor/evaluate.dart` (CSS)                                       | `eval/evaluate_css.go`, `eval/imported_css_visitor.go`                                                                           |
