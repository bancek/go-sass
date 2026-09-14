# CI

`concurrency.cancel-in-progress` on every push, in every workflow.

## `ci.yml` (push to `main` + PRs)

Unit tests + full spec suite on all three consumer OSes:

```yaml
strategy:
  matrix: { os: [ubuntu-latest, macos-latest, windows-latest] }
```

```sh
go build ./...
go vet ./...
gofmt -l .  # must print nothing (Go sources pin eol=lf in .gitattributes)
go test ./...
go test -tags=spec -timeout=300s ./spec/  # SASS_SPEC_PATH via env:
                                          # ../sass-spec/spec
```

- The multi-OS matrix is deliberate: OS-gated code and OS-sensitive
  behavior (fs paths, symlinks, case, platform libm) are invisible to an
  ubuntu-only gate. No extra toolchains: gcc ships on ubuntu runners,
  clang on macOS runners, mingw gcc on Windows runners — the `Probe C
  toolchain` step fails loudly if that ever drifts (the test/spec
  binaries link C libm unconditionally, so a missing compiler would
  otherwise surface as a cryptic cgo error).
- The spec suite needs the `sass-spec` submodule (checkout uses
  `submodules: recursive`). Timeout is 300s: Windows/macOS runners are
  slower than ubuntu (~10s on Linux).

## `release.yml` (push to `main` + PRs + tags `v*`)

Two jobs: native per-OS builders, one ubuntu assembler. cgo has no
linux-hosted cross path — each OS needs its own SDK — so single-runner
cross-compilation (GoReleaser style) is out; there is no
`.goreleaser.yaml`.

### `build` matrix: 8 native `go build -tags cgomath` jobs

linux-x64/arm64, linux-musl-x64/arm64, darwin-x64/arm64, win32-x64/arm64
(runners ubuntu-24.04[/-arm], macos-latest/macos-15-intel,
windows-latest/windows-11-arm). `-trimpath` on every build.

- The shipped binaries link C libm (`sassmathcgo`) for exact Dart
  matching — the same backend the test/spec gates assert. musl jobs
  install `musl-tools` and build with `CC=musl-gcc` plus static ldflags:
  genuine musl-linked static binaries, never relabeled gnu binaries.
- Each job stages `go-sass-<triple>[.exe]` and uploads it as a
  single-file artifact (one file per zip, 14-day retention — any run's
  binaries are downloadable without a release). No submodules: the CLI
  build needs module sources only.
- A `file` listing shows arch/libc per binary. musl-arm64 gets a native
  + Alpine smoke inside its own build job (host arch, no QEMU needed);
  musl-x64 gets the Alpine smoke in `assemble` (an x64 runner cannot
  execute arm64 without emulation).

### `assemble` (ubuntu, `needs: [build]`)

Downloads the 8 binaries (merged flat into `prebuilt/` via
`merge-multiple`, `chmod +x` since
artifact downloads drop the exec bit) → Alpine musl-x64 smoke →
`npm ci` → `npm run build -- --platforms=all` with
`GO_SASS_DIST=<workspace>/prebuilt` (assembles the 8 platform packages —
genuine per-OS binaries, with `libc` declared on the linux manifests
exactly like upstream — plus the `dist/` publish tree; see
`embedded-host-node-go/build.mjs`) → `npm test` (resolve gate +
harness + coexistence) → pack wrapper + 8 platform tgzs (pack failure is
packaging failure, so this subsumes dry-run validation) →
scratch-install smoke (`compileString` asserting `3px`, the true consumer
path with zero registry involvement; it does not prove optionalDep
auto-install, only a real publish does that) → `Make release archives`
(`go-sass-<version>-<platform>-<arch>.tar.gz`/`.zip`, upstream-style
names, + checksums; runs every push so main rehearses archive naming) →
per-file artifacts (18 one-file zips, 14-day retention).

- Tags only: version guard (full tag vs `package.dist.json` — the
  manifests carry the FULL version on prereleases), then
  `gh release create --draft` (`--prerelease` on `-`-suffixed tags,
  `--latest` otherwise, `--generate-notes`), `actions/attest@v4` on the
  release archives, `gh release upload` of archives + tgzs, and
  `npm publish --provenance` of the 8 platform packages + `./dist` —
  under `next` when the tag carries a `-` suffix (derived, not gated),
  `latest` otherwise. Auth is npm trusted publishing (OIDC) — no token
  secret; attestation needs `id-token: write` + `attestations: write`.
- Plain `npm run build` (no flags) keeps the dev flow: local
  `go build -tags cgomath` for the host platform only.

Local testing without publishing: `gh release download <tag>` (or the
Actions-artifact download for non-tag runs), then `npm install` the
wrapper + platform tarballs in a scratch dir, exactly like the smoke
step. Verify archive provenance with
`gh attestation verify <archive> -R <owner/repo>`.

No `ldflags` version injection: `--version` reports the hardcoded
`sassVersion`, which tracks dart-sass via the version-bump checklist in
[`CONTRIBUTING.md`](CONTRIBUTING.md).

## Manifest model (`package.dist.json`)

`embedded-host-node-go/package.dist.json` is the single version that
matters: the assemble job copies it into `dist/` as the published
manifest, platform versions and the tag guard derive from it. Source
`package.json` stays `0.0.0` forever — npm ≥ 11 rejects `npm ci` on
lockfiles whose optionalDeps are unresolvable (our platform packages,
unpublished by design), while `0.0.0` without optionalDeps installs
cleanly on every npm version. The lockfile carries no platform entries
(they live only in the dist manifest); it re-syncs only for the
`sass-embedded` devDep (the coexistence target), which is why the bump
checklist runs `npm install --package-lock-only`.

## Tags and prereleases

Tags track the pinned dart-sass version: `v1.104.0`, etc. Pushing a `v*`
tag creates the draft GitHub release with binaries + npm tarballs, and
publishes registries: npm under `next` on `-`-suffixed tags, `latest`
otherwise; archives carry Sigstore attestations either way. The full
procedure, including the alpha/final distinction, lives in
[`release.md`](release.md); the bump checklist in
[`CONTRIBUTING.md`](CONTRIBUTING.md).

Test the release path with a prerelease suffix (never `latest`):

```sh
git tag v1.104.0-prerelease && git push origin v1.104.0-prerelease
# verify assets on the release page, then clean up:
gh release delete v1.104.0-prerelease --cleanup-tag
git tag -d v1.104.0-prerelease
```

## While the repo is private

- Module/build caches are commented out in all workflows (re-enable on
  publish day — one line each).
- The `id-token` + `attestations` permissions are commented out in
  `release.yml` (uncomment on publish day — one block; see
  [`release.md`](release.md) First release). Until then, do not push
  tags: the attest and npm-publish steps need those permissions.
- For npm trusted publishing, add the repo as a trusted publisher on the
  `sass-embedded-go*` packages at first publish; nothing is parked in
  secrets beforehand.

## Roadmap (full CI)

- [ ] Publish day: uncomment caches (`ci.yml`, `release.yml`), `id-token`
      + `attestations` (`release.yml`), register npm trusted publishers.
- [ ] Platform parity beyond the big 6 + musl (android/riscv/armv7 need
      native runners alongside the `GOOS`/`GOARCH` adds) — see
      [`ref/sass-embedded-go.md`](ref/sass-embedded-go.md) §Binary distribution.
- [ ] SBOM attestations for the release archives (the `attest` action
      takes an `sbom-path`; the SBOM itself needs a generation step,
      e.g. `anchore-sbom-action`).
