# Module: `util/`

Small shared helpers with minimal dependencies.

| File                            | Contents                                                                                                                                          |
| ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `character.go`                  | Character-classification free functions (`isName`, `isNameStart`, `isHex`, `isWhitespace`, …) plus converters. `MaxAllowedCharacter = 0x10FFFF`.  |
| `css_identifier.go`             | `toCssIdentifier(text)` — CSS identifier escaping.                                                                                                |
| `fuzzy_equality.go`             | `FuzzyEquality` (equality/hash for floats).                                                                                                       |
| `number.go` + `number_write.go` | Fuzzy math (`fuzzyEquals`, `fuzzyAsInt`, `fuzzyRound`, `moduloLikeSass`, `clampLikeCss`, …) with `PRECISION = 10`, plus number-to-string writing. |
| `string.go`                     | Surrogate-pair helpers and string utilities.                                                                                                      |
| `trim_ascii.go`                 | ASCII trims plus `a(word)` (a/an). Trims only `0x20`; escape-aware trailing-space handling.                                                       |
| `utils.go`                      | `isPublic(name)`, `isPrivateMember(name)`, `pluralize(name, n, plural)`.                                                                          |

Edge cases: `fuzzyAsInt` rejects non-finite/overflowing values (mirroring
Dart's 64-bit `round()` bounds); `fuzzyRound` rounds `.5` up;
`moduloLikeSass` uses floored division (unlike Go's `%`); `clampLikeCss`
prefers the lower bound on NaN. Compressed number writing strips only a
leading `0` (`0.5 → .5` but `-0.5` keeps its sign).

## File mapping

| Dart                  | Go          |
| --------------------- | ----------- |
| `lib/src/util/*.dart` | `util/*.go` |
