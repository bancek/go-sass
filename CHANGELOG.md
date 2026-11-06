# Changelog

Release history. Tags track the pinned dart-sass version (`v1.105.0`); the
procedure lives in [`docs/release.md`](docs/release.md).

## Unreleased

## 1.105.0

- First-class modules, matching dart-sass 1.105.0: `meta.load()`,
  `meta.get-module()`, `meta.css()`, module references in the `$module`
  argument of `meta`'s introspection functions, and `meta.type-of()` returning
  `module`.
- Port the 1.104.1 parser fixes (phantom newline in indented selectors,
  `expected end of rule.` / `unrecognized syntax`) and the duplicate
  pre-module loud-comment fix.
- Port Dart's `Uri` semantics into `sassurl` (parse/resolve, file paths,
  `data:` URLs) and migrate every URL call site.
- Match Dart's module identity equality for forwarded and shadowed views.
- Embedded protocol 3.3.0 (`CompilerModule`) and module values over the
  protocol.
- Byte-identical output to Dart Sass 1.105.0 (14333/14339 spec tests passing,
  6 todo).

## 1.104.0

- Baseline: port complete against dart-sass `1.104.0` — `go-sass` CLI,
  `compile` API, embedded protocol server (`compilerVersion` 1.104.0,
  protocol 3.2.0), and the `sass-embedded-go` npm host. Byte-identical
  output to Dart Sass 1.104.0 (14246/14252 spec tests passing).
