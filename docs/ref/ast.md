# Module: `value/` (AST)

The Sass abstract syntax tree (statements and expressions) and the CSS output
tree. All AST code lives in the `value/` package under domain prefixes
(`sass_*`, `css_*`, `selector_*`, `sass_interpolated_selector_*`,
`sass_supports_condition_*`); visitors are documented in `visitors.md`.

## Design: interfaces, not hierarchies

One interface per hierarchy, structs embedding bases anonymously:

```
SassNode → SassDeclaration / Expression / Statement / …
Expression (18 impls) → ExpressionVisitor[T]
Statement (27 impls) → StatementVisitor[T]
IfConditionExpression (6 impls) → IfConditionExpressionVisitor
SupportsCondition (6 impls) → methods, no visitor
CssNode (9 impls) → CssVisitor[T]
ModifiableCssNode (9 impls) → ModifiableCssVisitor
```

**Why interfaces over structs:** Dart's sealed classes give exhaustiveness;
Go recovers it through the visitor interfaces (adding a node type forces every
visitor to grow a method) plus the `IsXxx()` markers that keep each set
closed. Recursive positions are pointers (`BinaryOperation` operands,
`Interpolation` parts) with child slices.

## Sass AST

- **Statements (27)** span the whole language: `Stylesheet`, `StyleRule`,
  `AtRule`, `AtRootRule`, `Declaration`, `VariableDeclaration`, the
  import/module rules (`ImportRule`, `UseRule`, `ForwardRule`), callable rules
  (`IncludeRule`, `FunctionRule`, `MixinRule`, `ContentRule`, `ContentBlock`),
  control flow (`IfRule`, `EachRule`, `ForRule`, `WhileRule`), `MediaRule`,
  `SupportsRule`, `ExtendRule`, `ErrorRule`/`WarnRule`/`DebugRule`/`ReturnRule`,
  and comments. `Stylesheet` is a `SassNode`, not a `Statement`.
- **Expressions (18)** include `BinaryOperation`, `Function`,
  `IfExpression`, `LegacyIfExpression`, `List`, `Map`, `String`, `Number`,
  `Color`, `Boolean`, `Null`, `Value`, `Variable`, `Selector`,
  `Parenthesized`, `UnaryOperation`, `SupportsExpression`,
  `InterpolatedFunction`.
- **If conditions (6)**: `Parenthesized`, `Negation`, `Operation`, `Function`,
  `Sass`, `Raw`.
- **Supports conditions (6)**: `Anything`, `Declaration`, `Function`,
  `Interpolation`, `Negation`, `Operation` (binary: `Left`/`Right`).
- **Wrapper interfaces** (`SassDeclaration`, `CallableInvocation`,
  `SassReference`, `SassDependency`) mirror Dart's sealed interfaces for
  structural fidelity.
- `CallableDeclaration` embeds in `FunctionRule`/`MixinRule`;
  `ParentStatement` (children list) is the base for loops, media, supports,
  at-root, style, and at-rules.
- `Interpolation{Contents []any(string|Expression), Spans}` carries
  constructor validation (length match, no adjacent strings); `IsPlain` /
  `AsPlain` / `InitialPlain` expose the plain-text fast path.
- `AtRootQuery{Include, Name *string}` is pure data (no span).
  `ParameterList` tracks rest arguments; `ParseTimeWarning` carries optional
  labels beyond Dart's record.
- `VariableDeclaration` rejects `namespace + global` (you cannot declare
  another module's member with `!global`).

### The nil-slice invariant

`nil []Statement` (no block, ends with `;`) is not `[]Statement{}` (an empty
`{}` block). This affects `Declaration`, `AtRule`, `StyleRule`, `MediaRule`,
`SupportsRule`, and `IncludeRule`'s content argument.

### `InterpolationMap` (span back-mapping)

`InterpolationMap` maps generated-output offsets back to the interpolation's
source spans. Provenance is allocation identity — never structural equality
and never URL comparison. See `parse.md` and `common.md`.

### Display conventions

`String()` on AST nodes mirrors Dart's `toString` exactly, including
`UseRule` URL quoting/`as`-elision and quote re-adding for quoted strings.

## CSS AST

Two layers, mirroring Dart's mutable/immutable split:

- **Frozen** `CssXxx` nodes (serialization): stylesheet, style rule, at-rule,
  comment, declaration, import, keyframe block, media rule, supports rule.
- **Mutable** `ModifiableCssXxx{Inner *CssXxx}` wrappers (evaluation)
  implementing the same `CssNode` / `ModifiableCssParentNode` interfaces.

### Parent chain

Parents own children (`[]ModifiableCssNode`); `Children()` converts to
`[]CssNode` through a cache invalidated on `AddChild` / `ClearChildren` /
`removeChildAt`. Identity is by reference. Tree guards mirror Dart:
`AddChild` errors on leaf types and childless at-rules; `remove` errors
without a parent; `ClearChildren` detaches both directions.

- `EqualsIgnoringChildren` is Dart-exact per variant (stylesheet → true;
  style rule → selector value equality, never allocation identity; at-rule →
  name+value+childless; keyframe → selector value; media → queries; supports
  → condition; else false).
- `CopyWithoutChildren` clones distinguishing fields except
  `StyleRule.fromPlainCSS` (reset to false, Dart's constructor default) and
  empties children — used when a target parent has a following sibling and
  when copying style rules into `@media`/at-rule bodies.
- Freezing to immutable nodes happens once at the end of evaluation.

### `clone_css`

`sassclonecss` deep-copies a module's CSS tree and its extension store for
`@use` boundaries. The clone map is keyed by selector allocation identity; a
selector absent from the map errors — the tree and the store must come from
the same compilation. Cloned style rules drop `fromPlainCSS`, like
`CopyWithoutChildren`.

## Interpolated selectors

`InterpolatedSelector` variants and `InterpolatedSimpleSelector` (8 variants)
represent selectors still containing `#{}` at parse time, with an
`InterpolatedSelectorVisitor`.

## File mapping

| Dart                                            | Go                                      |
| ----------------------------------------------- | --------------------------------------- |
| `lib/src/ast/sass/*.dart`, `ast/css/*.dart`     | `value/sass_*.go`, `value/css_*.go`     |
| `lib/src/ast/sass/expression/*.dart`            | `value/sass_expression_*.go`            |
| `lib/src/ast/sass/statement/*.dart`             | `value/sass_statement_*.go`             |
| `lib/src/ast/sass/interpolated_selector/*.dart` | `value/sass_interpolated_selector_*.go` |
| `lib/src/ast/sass/supports_condition/*.dart`    | `value/sass_supports_condition_*.go`    |
| `lib/src/ast/css/modifiable/*.dart`             | `value/css_modifiable_*.go`             |
| `lib/src/visitor/clone_css.dart`                | `sassclonecss/`                         |
