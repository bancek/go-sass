# Upstream tracking — Go port

This port mirrors a specific revision of the Dart Sass reference
implementation. `PORTED_FROM` in the repository root holds the
machine-readable dart-sass commit the port currently matches (read by sync
tooling). This file records the human-readable picture.

## Tracked upstream

|                     |                                                                        |
| ------------------- | ---------------------------------------------------------------------- |
| dart-sass commit    | `811ecc67bc5ed16d410d76e8bde343d9770f08c0` (2026-10-21, `PORTED_FROM`) |
| dart-sass version   | 1.106.1                                                                |
| Embedded protocol   | 3.3.0                                                                  |
| Dart SDK constraint | `>=3.13.0 <4.0.0` (dart-sass `pubspec.yaml`)                           |

On every re-sync to current `origin/main` (procedure: `porting.md`),
update `PORTED_FROM` and the table above together.

## External-package-derived files

Most files are ports of dart-sass `lib/` sources (see each file's
`// dart-source:` annotation and the Google MIT header). A small set instead
ports third-party Dart packages that dart-sass depends on; their license
headers carry the Dart-project-authors BSD text. Because these packages are not
part of dart-sass's tree, the exact version ported is recorded here. All
versions below are compatible with the ranges in dart-sass `pubspec.yaml` at
the tracked commit (dart-sass carries no committed lockfile).

| Ported file(s)                                                                                                                                                                      | Source package         | Version         |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------- | --------------- |
| `sasscommon/source_span_file.go`, `source_span_file_source.go`, `source_span_span.go`, `source_span_span_with_context.go`, `source_span_highlighter.go`, `source_span_exception.go` | `source_span`          | 1.10.2          |
| `sasscommon/string_scanner_span_scanner.go`, `string_scanner_span_scanner_test.go`                                                                                                  | `string_scanner`       | 1.4.1           |
| `termglyph/termglyph.go`                                                                                                                                                            | `term_glyph`           | 1.2.2           |
| `sourcemap/vlq.go`                                                                                                                                                                  | `source_maps`          | 0.10.13         |
| `sasscommon/pretty_uri.go`                                                                                                                                                          | `path`                 | 1.9.1           |
| `sasscommon/core_errors.go`                                                                                                                                                         | Dart SDK (`dart:core`) | Dart `>=3.13.0` |
| `sassurl/url.go`, `data.go`, `file_path.go`, `ip.go`, `tables.go`                                                                                                                   | Dart SDK (`dart:core` Uri) + `path` | Dart 3.13.5 / 1.9.1 |

When syncing (`porting.md`), re-port any file whose package range changed in
dart-sass's `pubspec.yaml`, and copy the package's actual header from the
pinned release (pub cache / GitHub), not dart-sass.

## Submodule pins

| Submodule            | Pin                          | Governs                                                                                   |
| -------------------- | ---------------------------- | ----------------------------------------------------------------------------------------- |
| `dart-sass`          | `811ecc67` (= `PORTED_FROM`) | the ported behavior, porting reference + differential loop; must agree with `PORTED_FROM` |
| `sass-spec`          | `fd1b9497`                   | the spec suite expectations                                                               |
| `sass`               | `88142a47`                   | `embedded_sass.proto` (moves only on `PROTOCOL_VERSION` changes) + host build vendoring   |
| `bootstrap-main`     | `25aa8cc0`                   | the `bench/` bootstrap workload                                                           |
| `embedded-host-node` | `c60459d` (host `1.106.1`)  | the `sass-embedded-go` host build (moves with the wrapper version)                        |

Update this table together with `PORTED_FROM` on every sync.
