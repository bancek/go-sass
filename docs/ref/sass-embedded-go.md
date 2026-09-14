# Module: `embedded-host-node-go`

The `sass-embedded-go` npm package: the genuine `sass-embedded` host API
backed by the go-sass embedded compiler instead of Dart Sass.

## Why this exists (and why not a patch)

`sass-embedded` resolves its compiler binary in `compiler-path.js` with no
public knob (no env var, no option). Patching the resolved path at runtime
mutates a process-global singleton: importing `sass-embedded-go` alongside
`sass-embedded` would silently rewire the app's host too. So the release
strategy assembles the host into **our own module namespace**: zero shared
modules with an app's `sass-embedded`, coexistence by construction. Rejected
alternatives: vendoring `dist/` (157 files committed), npm `overrides` for
the platform peer (install-time existence requirement plus a static 19-entry
list that can't condition on platform), relying on the stock `sass.js`
fallback (same-version false-green risk — hence the resolve gate).

## Build-time assembly (nothing vendored)

`embedded-host-node` submodule @ tag `1.104.0` (`191d6aa1`), full clone:

1. `go build -o <pkg>/platform-tmp-sass ./cmd/go-sass` (the compiler binary).
2. `npm install` + `npm run compile` inside `./embedded-host-node`, in place
   (`node_modules/` and `dist/` are gitignored there — the tracked tree stays
   pristine). The host tree carries no lockfile; our own lockfile pins what
   we test. Vendoring runs `tool/init.ts --skip-compiler --language-path
../sass` (all local: no network, no Dart SDK, no Dart binary).
3. Copy its `dist/` to `embedded-host-node-go/dist/` and apply the one-line
   template swap in `dist/lib/src/compiler-module.js` (`` `sass-embedded-${platform}-${arch}` ``
   → `` `sass-embedded-go-…` ``; asserted to hit exactly once). This is the
   ONLY fork point in host source (the sole functional `sass-embedded-`
   reference; musl/arch detection untouched).
   `compiler-path.ts` stays stock; unsupported platforms keep the upstream
   `sass.js` fallback (documented behavior, not relied upon — see the gate).
4. Assemble the local platform package
   `platform/<platform>-<arch>/` (`package.json` with `os`/`cpu` + the
   `dart-sass/sass` binary, exec bit preserved; Windows ships `sass.bat` +
   `sass.exe` per the resolution order) and link it into `node_modules/`
   (script state, never committed — same `require.resolve` path as a
   registry install).
5. `test/test.mjs` (harness copied from the old `sass-embedded-go` suite,
   repointed at `../dist` behind the resolve gate) + `test/coexistence.mjs`
   (both engines, one process) run against the assembly.

## Coexistence + resolve gate

`test/gate.mjs` runs first in every suite: it asserts the platform specifier
resolves inside our `platform/` tree, the binary exists, and the wrapper's
`compilerCommand` points at it — failing loudly with the binary path logged.
Without this gate, a missing platform package would fall back to Dart Sass
(same version, byte-identical output) and the suite would pass against the
wrong engine.

## Binary distribution

- Release binaries come from the `release.yml` build matrix: 8 native
  `go build -tags cgomath` jobs (linux-x64/arm64, linux-musl-x64/arm64,
  darwin-x64/arm64, win32-x64/arm64), staged flat as
  `go-sass-<triple>[.exe]` and assembled by `npm run build --
  --platforms=all` (`GO_SASS_DIST` pointing at the staging dir).
- Wrapper + 8 `sass-embedded-go-<platform>-<arch>` packages with
  exact-pinned `optionalDependencies` (host pattern). Linux manifests
  declare their `libc` (`glibc`/`musl` strings, mirroring upstream
  sass-embedded); musl entries are genuine musl-linked static binaries,
  never relabeled gnu binaries.
- Wrapper `package.json`: version tracks dart-sass from `1.104.0`, host runtime
  deps exact-pinned (proven with host 1.104.0), lockfile committed.
- **Unsigned binaries.** The compiler binaries ship unsigned (no Apple
  notarization) — the esbuild/swc/dart-sass precedent. macOS quarantine
  (`com.apple.quarantine`, the Gatekeeper prompt) is attached by browsers
  and download UX, never by installers: `npm install` sets no quarantine
  attribute, so npm-delivered binaries run without prompts. Only manual
  tarball downloads from the releases page may trip Gatekeeper (standard
  `xattr -d com.apple.quarantine` answer).

## Update procedure

Bump the submodule pin → rebuild → rerun harness + coexistence + tarball
validation (`npm pack`, scratch-dir install, smoke). Touch the template swap
only if upstream refactors the `compiler-module` consumers. Version bumps
ride the re-sync train (`upstream.md`, `compilerVersion`, CLI `sassVersion`,
wrapper + platform rebuild).

Rebuild hygiene: `embedded-host-node/node_modules/` predates the pin move
(the build script only installs when it is absent), so a stale tree breaks
the build with toolchain drift — delete `node_modules/` inside the submodule
before rebuilding after a pin move; the tracked tree stays pristine either
way.

## File mapping

| Dart/JS                                                                | Go                                                                       |
| ---------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| `embedded-host-node` submodule (`sass/embedded-host-node` @ `1.104.0`) | `embedded-host-node-go/` (build script, template swap, manifests, tests) |
| `sass-embedded` npm host (dev-only peer)                               | coexistence differential target                                          |
| old `sass-embedded-go` harness (`test.js`)                             | `test/test.mjs` (copied verbatim behind the gate)                        |
