# Visitor interfaces

The generic visitor interfaces and the `Accept*` dispatch that ties them to
the AST.

## The interfaces

| #   | Interface                         | Methods | `T` instantiations in use                                                    |
| --- | --------------------------------- | ------- | ---------------------------------------------------------------------------- |
| 1   | `StatementVisitor[T]`             | 27      | `any` (general), `bool` (search)                                             |
| 2   | `ExpressionVisitor[T]`            | 18      | `any` (general), `bool` (IsPlainCss, IsCalcSafe), `Expression` (ReplaceExpr) |
| 3   | `ValueVisitor[T]`                 | 10+     | `struct{}` (serialize), `bool`, `any`                                        |
| 4   | `CssVisitor[T]`                   | 9       | `struct{}` (serialize), `bool` (EveryCss)                                    |
| 5   | `SelectorVisitor[T]`              | 11      | `struct{}` (serialize), `bool` (search)                                      |
| 6   | `IfConditionExpressionVisitor[T]` | 6       | `Value` (eval), `struct{}`, `bool`                                           |
| 7   | `InterpolatedSelectorVisitor[T]`  | 11      | `struct{}`                                                                   |
| 8   | `ModifiableCssVisitor`            | 9       | `struct{}` (serialize/eval)                                                  |
| 9   | `CloneCssVisitor`                 | 9       | modifiable node (clone)                                                      |

```go
type ValueVisitor[T any] interface {
    VisitBoolean(*SassBoolean) (T, error)
    VisitNumber(SassNumber) (T, error)
    // ... one method per value type
}
```

All methods return `(T, error)` — no visitor is infallible.

## `Accept*` dispatch

```go
func (m *SassMap) AcceptVoid(v ValueVisitor[struct{}]) (struct{}, error) { return v.VisitMap(m) }
```

One accept method per AST node per result shape used in practice:
`AcceptValue` (general `any` traversal), `AcceptBool` (predicates),
`AcceptVoid` (effect visitors, including all of serialization), `AcceptExpr`
(`ReplaceExpressionVisitor`). `T` selects the result type, so adding a
traversal never requires a new interface — only a new instantiation.

Cross-package assertions go through `any` first (`any(x).(*T)`); a direct
assertion across packages is a compile error.

## `VisitList` covers argument lists

`VisitList(ListValue)` accepts both `SassList` and `SassArgumentList`;
`SassArgumentList` embeds `*SassList` plus `Keywords`, and handlers recover
keywords via a `val.(*SassArgumentList)` assertion. There is no separate
`VisitArgumentList`.

## Concrete implementations

| Visitor                                        | Implements                                               | `T`          |
| ---------------------------------------------- | -------------------------------------------------------- | ------------ |
| `SerializeVisitor`                             | Css, Value, Selector                                     | `struct{}`   |
| `RecursiveAstVisitor`                          | Statement, Expression, IfCondition, InterpolatedSelector | `struct{}`   |
| `ReplaceExpressionVisitor`                     | Expression, IfCondition                                  | `Expression` |
| `IsPlainCssVisitor`                            | Expression, IfCondition                                  | `bool`       |
| `IsCalculationSafeVisitor`                     | Expression                                               | `bool`       |
| `FindDependenciesVisitor`                      | Statement                                                | `struct{}`   |
| `StatementSearchVisitor`                       | Statement                                                | `bool`       |
| `AnySelectorVisitor` / `SelectorSearchVisitor` | Selector                                                 | `bool`       |
| `EveryCssVisitor`                              | Css                                                      | `bool`       |

There is no `EvaluateVisitor` row: the evaluator implements none of these
interfaces and dispatches through `EvaluateVisitor` methods + `switch` (see
`architecture.md` §4, `ref/eval.md`). There is no
`SourceInterpolationVisitor` either: Dart's is inlined as per-node
`SourceInterpolation()` methods. Two Dart-exact edges:
`FindDependenciesVisitor` only records a `load-css` call with exactly one
positional argument (extra args are not statically analyzable), and
`ReplaceExpressionVisitor`'s unknown-`SupportsCondition` arm returns an
error, never panics.

## File mapping

| Dart                               | Go                                                                                                    |
| ---------------------------------- | ----------------------------------------------------------------------------------------------------- |
| `lib/src/visitor/interface/*.dart` | `value/visitor.go`, `value/sass_*_visitor.go`, `value/css_visitor.go`, `value/selector_visitor.go`, … |
