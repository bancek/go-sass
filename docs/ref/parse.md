# Module: `value/parse_*` (parser)

The parser: source text in, `*Stylesheet` out. All parser code lives in the
`value/` package under the `parse_` prefix (see `architecture.md` §13 for why
it is not a separate package).

## Structure

One `SpanScanner` over `[]byte`, a `Parser` base, and per-syntax parsers
composed by **anonymous embedding**:

```
Parser (base: scanner, errors, helpers)
├── StylesheetParser (embeds Parser: stylesheet-level rules)
│   ├── ScssParser (embeds StylesheetParser)
│   └── SassParser (embeds StylesheetParser, indented syntax)
├── CssParser (embeds Parser directly)
└── SelectorParser (embeds Parser directly; media/keyframe/at-root variants alongside)
```

There is no virtual dispatch: subclass variance comes from
constructor-assigned `func` fields (statement separators, child detection,
indentation handling) and plain `bool` flags (`indented`, `plainCss`).

## The scanner

The scanner reads ASCII byte-at-a-time; only `int` return values exist, with
`-1` as the EOF sentinel (`PeekChar(offset)` / `ReadChar` return `int`;
`ch < 0` means EOF, mirroring Dart's nullable `int?`). UTF-8 bodies are
detected via `>= 0x80` and decoded with `utf8.DecodeRune`; columns advance
per rune (see `ref/common.md` for the character-column model).

- `ScanChar` / `ExpectChar` / `Expect` / `ExpectDone` for consuming input.
- `State` / `SetState` snapshots for backtracking; `SetPosition` seeks with
  cached line-starts (binary search).
- `SpanFrom` / `SpanFromTo` / `EmptySpan` / `Location` build spans;
  `Error(msg, pos, len)` constructs a `ScanError`.

Character helpers are pure free functions (`isWhitespace`, `isNewline`,
`isDigit`, `isHex`, `isName`, `isNameStart`, `asHex`, `characterEqualsIgnoreCase`,
surrogate helpers, `MaxAllowedCharacter = 0x10FFFF`). Dart `$lf`/`$cr`/`$space`
constants become rune literals (`'\n'`, `'\r'`, `' '`, …);
`StringBuffer` becomes `strings.Builder` (with `WriteRune`, never `WriteByte`,
for non-ASCII text); interpolation accumulation goes through
`InterpolationBuffer`.

`consumeEscapedCharacter`: `\` + EOF → U+FFFD; newline → error; up to 6 hex
digits plus optional whitespace skip; `0` / surrogates / out-of-range →
U+FFFD.

## Errors as values

```go
ScanError{Message, Span} → wrapSpanFormatException → SassFormatException{Message, Span}
```

Scanner methods return `*ScanError`; the parser's `wrapSpanFormatException`
(`parse_parser.go`) converts to `*SassFormatException`, remapping through the
`InterpolationMap` where present. Backtracking is an `err != nil` check — no
panics, no exceptions. The chain mirrors Dart's
`StringScannerException → SourceSpanFormatException → SassFormatException`,
and zero-length "expected" spans are adjusted before conversion. The parser
keeps a sticky error (`Parser.error`, `scannerErrorAt`) once raised. See
`ref/common.md`.

## Syntax specifics

- **SCSS / indented Sass / CSS** share `StylesheetParser`; the indented
  parser tracks `currentIndentation`, the CSS parser parses plain CSS with
  Sass interpolation.
- **Selectors** parse to interpolated AST (`SelectorList`, compounds,
  combinators, simple selectors) via `trySelector` / `parseSelectorList`; a
  style rule builds a real selector only when `parseSelectors` is set.
- **`if()`**: legacy `if($c, $t, $f)` is recognized in `identifierLike`;
  modern `if` goes through `ifExpression` / `ifConditionExpression`, with
  `tryArbitrarySubstitution` detecting `#{}` / `if()` / `var()` / `attr()` /
  custom properties before slurping raw conditions.
- **Import URLs**: `ParseImportUrl(rawURL string) string` is standalone;
  Windows-absolute paths (`len ≥ 3`, letter + `:` + slash) become `file://`
  via slash conversion, everything else passes through URL parsing
  (`looksLikeWindowsAbsolutePath`).

## Deprecation plumbing

Parsers that can trigger deprecations take explicit optional logger /
warning-function parameters threaded through the free functions; nested calls
that introduce no new deprecation pass nils. The evaluator supplies
stack-trace-aware callbacks. See `patterns.md` §9 and `ref/warn-logger.md`.

## Working here

- New syntax support goes in the matching per-syntax file; shared rules go
  on `StylesheetParser` or `Parser` by generality.
- Every new `Parse*` path needs both success tests and exact-error tests
  (span included).
- Never import the evaluator from the parser — the dependency runs one way
  (`eval` → `value`), and a cycle would break the package layout.

## File mapping

| Dart                                    | Go                                                             |
| --------------------------------------- | -------------------------------------------------------------- |
| `lib/src/parse/*.dart`                  | `value/parse_*.go`                                             |
| `lib/src/util/character.dart`           | character helpers in `value/parse_*` + `util/`                 |
| `lib/src/exception.dart` (parse errors) | `sasscommon/exception.go` (`ScanError`, `SassFormatException`) |
| `lib/src/interpolation_buffer.dart`     | `InterpolationBuffer` in `value/`                              |
