# Module: `extend/`

The `@extend` store: how extensions are registered and retroactively applied to
selectors.

## Types

```go
type ExtendMode int   // Normal, Replace, AllTargets (extend_mode.go)

type Extender struct { /* selector, specificity, isOriginal */ }
type Extension struct { /* extender, target, mediaContext, isOptional, span */ }

type ExtensionStore interface {
    IsEmpty() bool
    AddSelector(sel *value.SelectorList, mediaContext …) (*box.Box[*value.SelectorList], error)
    AddExtension(extender …, target …, extend …, mediaContext …) error
    // … queries, media scoping, cloning
}
// Empty + Default implementations (empty_extension_store.go, default_extension_store.go)
```

`ExtendMode.AllTargets` bypasses superselector checking when choosing
extenders. `MergedExtension` (`extension_merged.go`, a binary tree) records
the merge history of an extension for debug and error messages.

`CssMediaQuery` compares **without** `conjunction` (modifier, type,
conditions only), matching Dart's `==` — so the `Eq`/hash contract holds.

## Identity

`box.ModifiableBox` (shared mutable reference) and `box.Box` (sealed,
read-only handle) use reference identity — there is no `id` counter (Dart's
`ModifiableBox` uses default reference equality). `SelectorList` itself has
**structural** equality (compares components), while the store's clone map
uses **allocation identity** — matching Dart's `Map.identity()`.
`cloneStore()` translates old style-rule selectors to new store boxes at a
`@use` boundary; a selector absent from the map errors (tree and store must
come from the same compilation).

## Add-selector flow

1. Mark originals.
2. `extendList` → `extendComplex` → per compound, `extendSimple` → look up
   the extensions map.
3. `unifyExtenders` — weave the matching extenders.
4. `Trim` — superselector dedup, **skipped if the result exceeds 100** (a
   performance guard).
5. Seal the box (prevents further modification).

## Add-extension flow

1. Register in `extensions` (target → extender → `Extension`).
2. Register in `extensionsByExtender` (the reverse index).
3. **Retroactive propagation** — extend all previously-registered selectors
   that contain the target.
4. **Chain propagation** — walk `extensionsByExtender` so an extender of an
   extender's target is also propagated; previously extended selectors must
   **not** be double-extended.

## Scoping

`assertCompatibleMediaContext` prevents an extension from crossing
media-query boundaries. Private placeholders (prefixed `-`/`_`) are filtered
out of the selectors index.

## File mapping

| Dart                            | Go            |
| ------------------------------- | ------------- |
| `lib/src/extend/*.dart`         | `extend/*.go` |
| `lib/src/ast/selector/box.dart` | `box/box.go`  |
