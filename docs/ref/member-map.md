# Module: `orderedmap` views (member maps)

Read-only ordered map views for module member filtering (`@forward ...
show/hide`, prefixes, shadowing). Ports Dart's
`util/{prefixed,unprefixed,limited,merged,public_member_map_view}.dart`.

## The `Map` interface

```go
// orderedmap/map.go
type Map[K comparable, V any] interface {
    Get(key K) (V, bool)
    Has(key K) bool
    Put(key K, val V)
    Delete(key K)
    Len() int
    Keys() iter.Seq[K]         // insertion order
    Entries() iter.Seq2[K, V]  // insertion order
}
```

`Module.Variables/Functions/Mixins` return this interface — never a concrete
`*LinkedMap`, never a builtin `map` (which loses insertion order). Concrete
storage is `LinkedMap` (= Dart's `LinkedHashMap`).

## Views

| Type                  | Dart counterpart              | Semantics                                                |
| --------------------- | ----------------------------- | -------------------------------------------------------- |
| `LinkedMap`           | owned map                     | The concrete map (built-in modules, environment frames). |
| `PrefixedMapView`     | `prefixed_map_view.dart`      | Strips/adds the `@forward` prefix on lookup.             |
| `LimitedMapView`      | `limited_map_view.dart`       | Safelist (show-list) / blocklist (hide-list).            |
| `MergedMapView`       | `merged_map_view.dart`        | First-appearance order, later maps win on conflicts.     |
| `PublicMemberMapView` | `public_member_map_view.dart` | Hides private (`-`/`_` prefixed) members.                |

Views compose: prefix → safelist/blocklist, returning the inner map unchanged
when no filtering applies; local members wrapped in a public view, then
merged with forwarded members (local last, so it shadows). Helpers
`needsBlocklist` / `mapKeyDiff` support the shadowed-module clone path.

## Working here

- Key **order** is observable (`meta.module-variables` iteration, `@each`
  serialization) — any new view must document whether it preserves inner
  order, safelist order, or first-appearance order, with a test locking it.
- Views are read-only by design; shadowing/mutation happens in the shadowed
  module view + `sassenv`, never here.
