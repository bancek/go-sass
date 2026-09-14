# go-sass CLI

The `go-sass` command-line compiler — a drop-in replacement for the Dart Sass
`sass` CLI. See the repository root for the project overview.

## Building

```sh
go build ./cmd/go-sass
```

## Usage

```sh
go-sass [options] [input.scss [output.css]]
go-sass [options] input.scss:output.css
go-sass [options] input.scss:output.css more.scss:more.css
go-sass [options] input-dir:output-dir
```

One or more sources may be passed; a source of `-` (or `--stdin`) reads from
stdin. A directory source compiles its non-partial `.scss`/`.sass`/`.css`
entrypoints to the corresponding destination directory.

### Options

| Flag                                           | Description                                                               |
| ---------------------------------------------- | ------------------------------------------------------------------------- |
| `--stdin` / `--no-stdin`                       | Read the stylesheet from stdin.                                           |
| `--indented` / `--no-indented`                 | Use the indented syntax for input from stdin.                             |
| `-I, --load-path PATH`                         | A path for resolving imports (repeatable).                                |
| `-p, --pkg-importer TYPE`                      | Built-in importer(s) for `pkg:` URLs (`node`).                            |
| `-s, --style NAME`                             | Output style: `expanded` (default) or `compressed`.                       |
| `--charset` / `--no-charset`                   | Emit a `@charset`/BOM for non-ASCII output (default on).                  |
| `--error-css` / `--no-error-css`               | Emit a stylesheet describing an error (default when writing to a file).   |
| `--source-map` / `--no-source-map`             | Generate source maps (default on).                                        |
| `--source-map-urls TYPE`                       | `relative` (default) or `absolute`.                                       |
| `--embed-sources` / `--no-embed-sources`       | Embed source contents in source maps.                                     |
| `--embed-source-map` / `--no-embed-source-map` | Embed source maps in CSS.                                                 |
| `-q, --quiet` / `--no-quiet`                   | Don't print warnings.                                                     |
| `--quiet-deps` / `--no-quiet-deps`             | Don't print warnings from dependencies.                                   |
| `--verbose` / `--no-verbose`                   | Print all deprecation warnings.                                           |
| `--fatal-deprecation DEP`                      | Treat a deprecation (or a Sass version) as an error.                      |
| `--silence-deprecation DEP`                    | Ignore a deprecation.                                                     |
| `--future-deprecation DEP`                     | Opt into a deprecation early.                                             |
| `--stop-on-error` / `--no-stop-on-error`       | Stop after the first error.                                               |
| `--trace` / `--no-trace`                       | Print full stack traces for exceptions (accepted; Go has no Dart traces). |
| `-c, --color` / `--no-color`                   | Use terminal colors for messages.                                         |
| `--unicode` / `--no-unicode`                   | Use Unicode glyphs in messages (default on).                              |
| `-h, --help`                                   | Print usage (exits 64, matching Dart).                                    |
| `--version`                                    | Print the version of Dart Sass.                                           |

Exit codes: `64` usage error, `65` Sass error, `66` filesystem error.

`--watch`, `--poll`, `--update`, and `--interactive` are not implemented.

### Embedded server mode

Starts the Sass embedded protocol server.

```sh
go-sass --embedded
```
