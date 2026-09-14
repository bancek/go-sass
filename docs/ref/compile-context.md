# Module: per-compilation identity token

The identity token for a single compilation.

## `compileContext any`

```go
// eval/evaluate.go
compileContext any
compileContext: &struct{}{},   // fresh per compilation
```

Every compilation gets a fresh `&struct{}{}` stored on the `EvaluateVisitor`.
`SassFunction`/`SassMixin` values capture the context at creation
(`NewSassFunctionWithCompileContext` / `NewSassMixinWithCompileContext`);
`meta.call`/`meta.apply` assert it matches the current compilation via
`AssertCompileContext` before invoking the wrapped callable
(same-compilation guard). Comparison is interface identity — matching Dart's
`final Object _compileContext = Object()` (object identity;
`dart-source: lib/src/visitor/evaluate.dart`).

## Why `any`, not a struct type

The context must be comparable across compilations (a function value created
in one compilation is rejected in another) and must never be
confused with real data: an unexported `any` holding a unique allocation is
the smallest type that has identity and nothing else. Call sites never inspect
it — only `AssertCompileContext` compares it.

## Working here

- New cross-compilation guards go through `AssertCompileContext` on the value
  types, not through direct comparison at call sites.
- The token is created in exactly one place (`NewEvaluateVisitor`); never
  fabricate one elsewhere.
