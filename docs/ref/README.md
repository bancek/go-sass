# Module reference

Index of the per-module reference documents.

| Document                                     | Module(s)                                                                 |
| -------------------------------------------- | ------------------------------------------------------------------------- |
| [`common.md`](common.md)                     | `sasscommon/` — errors, spans, source locations, the scanner              |
| [`value.md`](value.md)                       | `value/` — the Sass value type and its concrete types                     |
| [`ast.md`](ast.md)                           | `value/sass_*`, `value/css_*` — the Sass and CSS abstract syntax trees    |
| [`selector.md`](selector.md)                 | `value/selector_*` — selector types (extend algorithms live in `extend/`) |
| [`parse.md`](parse.md)                       | `value/parse_*` — the parser                                              |
| [`extend.md`](extend.md)                     | `extend/` — the `@extend` store                                           |
| [`serialize.md`](serialize.md)               | `value/visitor_serialize.go` — the serializer                             |
| [`visitors.md`](visitors.md)                 | the generic visitor interfaces and `Accept*` dispatch                     |
| [`eval.md`](eval.md)                         | `eval/` — the evaluator                                                   |
| [`environment.md`](environment.md)           | `sassenv/`, `sassmodule/`, `configuration/`, `evalcontext/`               |
| [`callable.md`](callable.md)                 | `sasscallable/` — the function/mixin invocation machinery                 |
| [`functions.md`](functions.md)               | `functions/` — the built-in Sass function library                         |
| [`math.md`](math.md)                         | `sassmath/` (+ `sassmath/sassmathcgo/`) — math backend and float parity   |
| [`io.md`](io.md)                             | `sassio/` — the host I/O abstraction                                      |
| [`logger.md`](logger.md)                     | `sasslogger/`, `deprecation/`                                             |
| [`source-maps.md`](source-maps.md)           | `sourcemap/`, `sourcemapbuffer/`                                          |
| [`warn-logger.md`](warn-logger.md)           | deferred deprecation warnings                                             |
| [`compile.md`](compile.md)                   | `compile/` — the public compile API                                       |
| [`sass-spec.md`](sass-spec.md)               | the `sass-spec` suite: layout, `.hrx` format, debugging failures          |
| [`pipeline.md`](pipeline.md)                 | end-to-end walkthrough: entries, stages, ownership                        |
| [`compile-context.md`](compile-context.md)   | the per-compilation identity token (`compileContext any`)                 |
| [`member-map.md`](member-map.md)             | `orderedmap` views — lazy member maps for `@forward` filtering            |
| [`url.md`](url.md)                           | `sassurl/` — Dart-compatible URL resolution                               |
| [`importer.md`](importer.md)                 | `eval/import_cache.go`, `eval/importer*.go` — stylesheet resolution       |
| [`embedded.md`](embedded.md)                 | `embedded/` — the protocol server                                         |
| [`sass-embedded-go.md`](sass-embedded-go.md) | `embedded-host-node-go/` — the published host API package                 |
| [`util.md`](util.md)                         | `util/` — small shared helpers                                            |

The CLI is documented in [`cmd/go-sass/README.md`](../../cmd/go-sass/README.md).

See [`../architecture.md`](../architecture.md) for how these fit together and
[`../README.md`](../README.md) for the project overview.
