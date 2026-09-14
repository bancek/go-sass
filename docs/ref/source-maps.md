# Modules: `sourcemap/`, `sourcemapbuffer/`

Source-map generation: the VLQ encoder, the V3 builder, and the buffer that
records mappings during serialization.

## VLQ

The V3 source-map variable-length quantity encoder (`sourcemap/vlq.go`, a
port of `package:source_maps` 0.10.13 — see `upstream.md`):

- The sign is folded into the least-significant bit.
- Values are emitted as 5-bit chunks with a continuation bit.
- Each chunk is mapped through the Base64 alphabet.

## Builder

`sourcemap.Builder` accumulates `(generated_line, generated_column,
source_line, source_column)` entries and emits V3 JSON. Two behaviors match
Dart exactly:

- **Entry dedup** — an entry is skipped when the last entry shares the same
  source line and target line (browsers don't care about position within a
  line).
- **charset/BOM** — compressed output prepends `\uFEFF`; expanded prepends
  `@charset "UTF-8";`.

`sourcesContent` is emitted only when source-map include-sources is set
(`CompileOptions` threads through serialization to the builder).

## SingleMapping

`SingleMapping` is the source-map value carried on `CompileResult`
(`CompileResult.SourceMap()`), serialized to V3 JSON bytes for output files
and embedded maps.

## SourceMapBuffer

`sourcemapbuffer/` (`DefaultSourceMapBuffer`, `NoSourceMapBuffer`) is the
serialization-time buffer (see `serialize.md`): `ForSpan(span, cb)` runs a
callback while mapping output positions to the span. During serialization,
`forNode` maps the current output position to a source span start, and each
`\n` while in-span auto-maps the output line. Columns count
characters/UTF-16 units, not bytes (see `common.md`).

## File mapping

| Dart                                            | Go                       |
| ----------------------------------------------- | ------------------------ |
| `lib/src/util/source_map_buffer.dart`           | `sourcemapbuffer/*.go`   |
| `(external) package:source_maps` (VLQ)          | `sourcemap/vlq.go`       |
| `lib/src/util/source_map_buffer.dart` (builder) | `sourcemap/sourcemap.go` |
