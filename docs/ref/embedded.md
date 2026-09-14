# Module: `embedded/`

The Sass embedded protocol server: a protocol-buffer-based server over
stdin/stdout that the `sass-embedded` host (Node.js) spawns with
`sass --embedded`.

## What it is

One package plus generated bindings plus the CLI entry:

- `embedded/` — the server (`IsolateDispatcher`, compilation dispatchers,
  reusable isolates, host callables, importers, logger, protofier, packet
  I/O).
- `embedded/embedded_sass.pb.go` — the committed `protoc-gen-go` bindings
  for `embedded_sass.proto` (regenerate with `go generate ./embedded/`
  only when the `.proto` changes).
- `embedded/executable.go` + `sass/` — the `--embedded` entry that serves
  the protocol over stdio.

It reports protocol version `3.2.0`, compiler/implementation version
`1.104.0`, and implementation name `"dart-sass"` — the values the
`sass-embedded` host requires (`compilerVersion` / `protocolVersion` in
`embedded/isolate_dispatcher.go`).

## Wire protocol

Packets are **length-delimited**: `[varint length][varint compilation_id]
[protobuf message]` (`length_delimited_transformer.go`,
`varint_builder.go`, `packet_writer.go`).

- **compilation_id 0** → version request → version response.
- **compilation_id ≠ 0** → the compilation's isolate (`getIsolate` creates
  or reuses a `ReusableIsolate`): a compile request first (producing a
  compile response — success or failure — on the same compilation id), then
  canonicalize/import/file-import/function-call responses correlated by
  request id.
- **Host callbacks** are synchronous request/response within a compilation;
  the compiler sends canonicalize/import/file-import/function-call requests
  and blocks awaiting the response (`host_callable.go`, `importer_host.go`).
  Log events are fire-and-forget (`logger.go`).

Exit codes: `64` (extra CLI args), `70` (internal compiler error).
A compile _failure_ is a normal failure response and must **not** exit.

## Decisions

| #   | Decision                                                                                      |
| --- | --------------------------------------------------------------------------------------------- |
| D1  | `protoc-gen-go` with **committed** generated code — no build-time `protoc`.                   |
| D2  | Dispatcher + reusable isolates; one isolate per compilation, pooled and reused.               |
| D3  | `"dart-sass"` / `"1.104.0"` version reporting (host-required constants).                      |
| D4  | Host function signatures never panic — an invalid signature becomes a compile failure.        |
| D5  | Importer canonicalize contexts carry the containing-URL access tracking that affects caching. |

## Threading

A reader loop pushes raw packets; the dispatcher routes them by compilation
id; each isolate owns its compiler state, host importers/functions/logger,
and response channel. Outbound writes go through the concurrent packet
writer (one full packet per write).

## Protofier

`protofier.go` converts values to and from proto form with a per-compilation
context tracking argument-list ids (round-tripped by id) and opaque
function/mixin ids. Colors map through the missing-channel bitmask;
calculations validate names and arities. `proto_extensions.go` holds the
hand-written helpers alongside the generated code.

## Codegen

`embedded/embedded_sass.pb.go` is committed generated code — regenerate only
when the `.proto` changes (`go generate ./embedded/`, directive in
`embedded/gen.go`). The source is the canonical
`../sass/spec/embedded_sass.proto` (the `sass` language-spec submodule);
there is deliberately no vendored copy. `paths=source_relative` keeps the
output in `embedded/`; the `M` flag supplies the import path because the
upstream proto carries no `go_package` option (the submodule stays unedited).

## Gotchas

- `protoc-gen-go` flattens nested messages differently from other
  generators — use the generated Go names, not the proto nesting.
- Only `FileSpan`-derived owned spans reach the wire; internal span types
  never do.
- The file importer uses a no-load-path filesystem importer, so no spurious
  load-path behavior leaks into compilations.
- `--version` over the protocol reports the same versions as the constants
  above; keep them in lockstep via the `CONTRIBUTING.md` checklist.

## File mapping

| Dart                                            | Go                                                     |
| ----------------------------------------------- | ------------------------------------------------------ |
| `lib/src/embedded/*.dart`                       | `embedded/*.go`                                        |
| `sass/spec/embedded_sass.proto` (language spec) | `embedded/embedded_sass.pb.go` (via `embedded/gen.go`) |
