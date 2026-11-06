# Module: `sassurl/`

Dart-compatible URI implementation, ported from `dart:core`'s `Uri`
(Dart 3.13.5) and `package:path`'s `toUri`/`fromUri`. It exists because
`net/url` normalizes, escapes, and resolves differently from Dart in ways that
are observable in Sass output and error spans.

## `URL`

```go
type URL struct {
    Scheme       string
    UserInfo     string
    Host         string   // Dart's Uri.host getter (IPv6 brackets stripped)
    Port         int
    HasPort      bool
    Path         string
    Query        string
    HasQuery     bool
    Fragment     string
    HasFragment  bool
    HasAuthority bool
}

func Parse(raw string) (*URL, error)          // Matches Dart: Uri.parse
func (u *URL) String() string                 // Matches Dart: Uri.toString
func (u *URL) Resolve(ref *URL) *URL          // Matches Dart: Uri.resolveUri
func Resolve(base, ref *URL) *URL
func (u *URL) ResolveString(ref string) (*URL, error)
func (u *URL) IsAbs() bool
func (u *URL) Equal(other *URL) bool
func (u *URL) HashCode() int
```

`Host` follows Dart's getter: an IPv6 literal loses its brackets, while an
IPvFuture literal (`[v...]`) keeps them. The serialized host is kept internally
so `String()` still emits the original bracketed form.

Parsing operates on UTF-16 code units internally, matching Dart's code-unit
semantics for surrogate pairs and percent escapes. The internal `kind` field
preserves Dart's `_SimpleUri`/`_Uri` distinction because it changes relative
merge behavior for values that compare equal, such as `..` vs `../`.

## Resolution

`Resolve` ports Dart's `Uri.resolveUri`, including:

- opaque bases like `sass:color` and `u:foo/bar` (Dart treats the path as
  hierarchical);
- Dart's `_SimpleUri._simpleMerge` fast path for two relative references;
- dot-segment removal, query/fragment inheritance, and authority handling.

Use `Equal`, never pointer equality, when comparing URLs. When a map is keyed
by URL identity, key it by `String()` (see `eval/import_cache.go`) or by a
normalized component tuple.

## File paths and `data:` URLs

```go
func FileURLFromAbsPath(path string) (*URL, error) // package:path toUri
func PathToURI(path string) (*URL, error)          // absolute or relative
func (u *URL) ToFilePathString() (string, bool)    // Dart Uri.toFilePath
func (u *URL) FSPath() string
func ResolveFilePath(url, base *URL) (*URL, bool)
func DataURLFromString(contents string) string     // Uri.dataFromString
```

File-path conversion handles POSIX and Windows paths, including drive letters
and UNC shares. `DataURLFromString` matches
`Uri.dataFromString(contents, encoding: utf8)` byte-for-byte.

## Working here

- New URL-dependent behavior needs a differential test against Dart
  (`mise x dart@3.13.5 -- dart ...`): URL edge cases are where implementations
  most often disagree (opaque bases, trailing slashes, empty segments,
  percent-encoding, Windows paths).
- Keep the public `URL` fields consistent with Dart's component getters; do not
  reintroduce `net/url` field semantics such as `Opaque` or `OmitHost`.
