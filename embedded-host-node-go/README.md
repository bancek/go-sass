# sass-embedded-go

The [`sass-embedded`](https://www.npmjs.com/package/sass-embedded) host API
backed by the [go-sass](https://github.com/bancek/go-sass) embedded compiler —
a drop-in replacement that compiles with Go instead of Dart.

## Install

```sh
npm install sass-embedded-go
```

## Usage

```js
import { compileString } from "sass-embedded-go";

compileString("a {b: 1px + 2px}").css;
```

The API is identical to `sass-embedded` (same exports and types), only the
compiler binary differs. Both packages can be imported in one process without
interference.

## Versioning

Version tracks [dart-sass](https://github.com/sass/dart-sass) (`1.104.0`,
protocol `3.2.0`).

## Links

- [Repository](https://github.com/bancek/go-sass) (compiler source).
- [Architecture and build docs](https://github.com/bancek/go-sass/blob/main/docs/ref/sass-embedded-go.md).

## License

MIT. Host code is © Google LLC (MIT).
