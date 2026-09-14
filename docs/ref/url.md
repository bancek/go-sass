# Module: `sassurl/`

Dart-compatible URL parsing over Go's `net/url`. Unlike stricter URL
libraries, `net/url` represents scheme-less (relative) URLs natively, so no
wrapper type is needed — this package adapts the two spots where `net/url`
semantics diverge from Dart's `Uri`.

## `Parse`: opaque-URI normalization

Go's `url.Parse` puts opaque-URI data (like `sass:color`) in `u.Opaque`,
while Dart's `Uri.parse` puts it in `path`. `Parse` normalizes this: move
`Opaque` to `RawPath` (preserving the original encoding for `String()`), set
`Path` to the decoded form, and set `OmitHost` — so `String()` still returns
`sass:color` rather than `sass://color`.

```go
// sassurl/sassurl.go
func Parse(raw string) (*url.URL, error)   // Matches Dart: Uri.parse
```

## `Resolve`: opaque-base resolution

`url.URL.ResolveReference` does not resolve relative URLs against opaque
bases (it treats the opaque part as non-hierarchical); Dart's `Uri.resolve`
treats the path as hierarchical. `Resolve` handles the opaque case by
resolving path components manually (absolute-path refs against the scheme,
relative refs against the base directory), falling back to
`ResolveReference` for hierarchical bases.

```go
func Resolve(base *url.URL, ref *url.URL) *url.URL   // Matches Dart: Uri.resolve
```

Related URL handling lives with its callers: Windows-absolute detection in
`ParseImportUrl` (`value/parse_import_url.go`), load-path fallback in the
filesystem importer, `data:`-URL construction at the importer-result /
source-map boundary.

## Working here

- Canonicalize before using URLs as map keys (see `eval/import_cache.go`):
  two URLs that display identically but were built differently can compare
  unequal.
- New URL-dependent behavior needs a differential test against
  `dart run bin/sass.dart` (URL edge cases are where `net/url` and Dart's
  `Uri` most often disagree: opaque bases, trailing slashes, empty segments,
  percent-encoding, Windows paths).
