# Changelog

Release history. Tags track the pinned dart-sass version (`v1.108.2`); the
procedure lives in [`docs/release.md`](docs/release.md).

## Unreleased

## 1.108.2

Covers dart-sass 1.108.0 through 1.108.2. The intermediate 1.108.0 and 1.108.1
tags are not released separately: 1.108.1 fixed only upstream's embedded-host
packaging, and 1.108.2 has no user-visible changes.

- Undo the unintentional breaking changes added in 1.106.0 in how escaped
  identifiers interact with `sass:meta` and named rest arguments. Reflection
  lookups (`global-variable-exists`, `variable-exists`, `function-exists`,
  `mixin-exists`, `get-function`, `get-mixin`, `get-module`,
  `module-variables`/`functions`/`mixins`) and `@use … with` configuration
  resolve identifiers in canonical form again.
- Fix a bug where a namespace declared with an escaped identifier
  (`@use "sass:math" as \$`) was unreachable through `sass:meta`.
- Fix `meta.call()` with a string name falling back to a plain-CSS callable, and
  escape `get-function()`/`get-mixin()`/`get-module()` names when serializing.
- Fix argument verification to report unknown named arguments in the order they
  were passed and without an Oxford comma, and match `meta` overloads against
  parameter names as written.
- Byte-identical output to Dart Sass 1.108.2 (14496/14502 spec tests passing,
  6 todo).

## 1.107.1

- Advertise embedded protocol 3.4.0, the version dart-sass 1.107.1 reports for
  `fatal_deprecation_except` (the field itself shipped in 1.107.0).
- Match dart-sass 1.107.1's embedded host release.
- Byte-identical output to Dart Sass 1.107.1 (14383/14389 spec tests passing,
  6 todo).

## 1.107.0

- Support fatal deprecation selectors matching dart-sass 1.107.0:
  `--fatal-deprecation` accepts `-{id}` exclusions and `dart-sass-2`, and the
  embedded protocol accepts `fatal_deprecation_except`.
- Selecting deprecations from a version later than the running compiler now
  emits the new `deprecation-version` warning.
- Byte-identical output to Dart Sass 1.107.0 (14383/14389 spec tests passing,
  6 todo).

## 1.106.1

- Fix a crash when serializing an attribute selector whose value is the empty
  string, such as `[class=""]`; it is now quoted like any other non-identifier
  value.
- Fix a crash when serializing a number whose unit is the empty string. This is
  only reachable through the embedded protocol and the JavaScript/Dart APIs.
- Convert invalid-argument errors raised while decoding host-supplied values
  into `PARAMS` protocol errors instead of compile failures.
- Byte-identical output to Dart Sass 1.106.1 (14383/14389 spec tests passing,
  6 todo).

## 1.106.0

- Preserve CSS escapes as written in statement contexts and normalize them only
  where SassScript observes them. Selectors, identifiers, custom properties,
  unknown at-rules, and `@keyframes` names keep their original escape spelling;
  invalid escapes normalize to U+FFFD, while `\0` alone keeps the historical
  `\0 ` form. Attribute values containing backslashes stay quoted, and
  exponent-like units serialize with an escaped `\65`.
- Fix crashes and inaccurate source spans in interpolations containing patterns
  like `// */`, including span expansion and warning/error spans.
- Fix an indented-syntax crash when the first statement is indented and appears
  after the first line; the error now highlights only that line's indentation.
- Throw a multi-span `Duplicate parameter.` error when a rest parameter reuses a
  normal parameter name, instead of silently shadowing it.
- `FileSpan.trim()`, `trimLeft()`, and `trimRight()` now skip comments as well
  as whitespace, so diagnostics no longer highlight trailing comments.
- Match dart-sass 1.106.0's embedded host release.
- Byte-identical output to Dart Sass 1.106.0 (14382/14388 spec tests passing,
  6 todo).

## 1.105.2

- Match dart-sass 1.105.2's embedded-host release, fixing a deadlock when the
  async compiler fails to start. The compiler is unchanged from 1.105.1.

## 1.105.1

- Match dart-sass 1.105.1's multi-span `@extend` errors across media
  queries, with `extension`, `extension @media`, `extended selector`, and
  `extended selector @media` labels.
- Trim trailing whitespace from selector and media-query spans during
  parsing, and make span trimming lazy, so diagnostics no longer highlight
  trailing whitespace.
- Fix interpolation span expansion at adjacent non-whitespace characters.
- Bind mixin rest/named/default parameters to Dart's declaration nodes, so
  source maps point at the call site or default expression instead of the
  mixin declaration. Bootstrap source maps are byte-identical to Dart.
- Byte-identical output to Dart Sass 1.105.1 (14338/14344 spec tests
  passing, 6 todo).

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
