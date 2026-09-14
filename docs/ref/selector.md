# Module: `value/selector_*` (selectors)

Selectors and the extend algorithm primitives. Selectors are interfaces with
struct implementations, not a class hierarchy.

## Types

```go
type Selector interface { AstNode; IsSelector() }
type SimpleSelector interface { Selector; IsSimpleSelector() }
// Attribute, Class, ID, Pseudo, Parent, Placeholder, Type, Universal  // 8 impls
type Combinator int   // NextSibling(+), Child(>), FollowingSibling(~)
```

All concrete types implement `Equal()` (logical equality, spans excluded),
`String()`, and the `Accept*` family over `SelectorVisitor[T]`.

## Equality and hashing

- **Span is excluded from equality and hash** everywhere — Dart never
  compares `span`.
- `QualifiedName{Name string, Namespace *string}`: `nil` = default, `""` =
  none, `"*"` = any; needs explicit `Equal()` (pointer comparison is
  insufficient).
- `IsSuperselector` returns `(bool, error)` and handles the pseudo-class
  strategies (`:is` / `:not` / `:where` / `:has` / `:nth-child`).
- `SelectorList` has **structural** equality (compares components, matching
  Dart's `==`); the extension store's identity map uses allocation identity
  (matching Dart's `Map.identity()`) for its clone translation.

## `PseudoSelector` boolean sense

`IsSyntacticClass` is the inverse of Dart's `isSyntacticElement`
(`IsSyntacticClass: !element` at construction): single-colon `:` vs
double-colon `::` sense. The naming is inverted but the serialized output is
identical (see `divergences.md` §1). `NormalizedName` unwendors the name.

## Extend algorithms

- **Weave** — LCS-based interleaving of N complex selectors, preserving
  relative order, with `mergeTrailingCombinators` handling the combinator
  merges (`value/selector_extend.go`, `selector_extend_functions.go`).
- **Unify** — complex unification extracts and unifies base compounds, weaves
  the results, and fails on ID conflicts; covers universal/element and
  namespace/name unifications.
- **Superselector** — list/complex/compound/simple levels, with pseudo-class
  strategies for `:is`/`:not`/`:where`/`:has`/`:nth-child`.
- **Recursive search** — `RecursiveSelectorVisitor`
  (`value/selector_recursive_selector.go`).

### Specificity (base-1000)

| Selector                                            | Value              |
| --------------------------------------------------- | ------------------ |
| Universal                                           | 0                  |
| Type, pseudo-element                                | 1                  |
| Attribute, class, placeholder, parent, pseudo-class | 1000               |
| ID                                                  | 1,000,000          |
| `:where()`                                          | 0                  |
| `:is()`, `:not()`, `:has()`, `:matches()`           | max of args        |
| `:nth-child()`, `:nth-last-child()`                 | 1000 + max of args |

## File mapping

| Dart                                                             | Go                                             |
| ---------------------------------------------------------------- | ---------------------------------------------- |
| `lib/src/ast/selector/*.dart`, `lib/src/visitor/*selector*.dart` | `value/selector_*.go`                          |
| `lib/src/visitor/recursive_selector.dart`                        | `value/selector_recursive_selector.go`         |
| `lib/src/visitor/replace_expression.dart`                        | `ReplaceExpressionVisitor` (see `visitors.md`) |
| `lib/src/extend/functions.dart` (selector part)                  | `value/selector_extend_functions.go`           |
