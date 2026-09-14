# Release runbook

Tags track the dart-sass version (`v1.104.0`); rehearsals add a suffix
(`v1.104.0-alpha0`, `v1.104.0-alpha1`). Mechanics: [`ci.md`](ci.md). Bump
surfaces: [`CONTRIBUTING.md`](CONTRIBUTING.md#version-bump-checklist).

## Bump

`tools/bump-version.sh <version>` — patch releases and `-alpha`
rehearsals only (no behavior change behind them). Anything bigger rides
the [`porting.md`](porting.md) re-sync, whose step 7 ends here. The script
writes FULL version to the published identity
(`embedded-host-node-go/package.dist.json`: `version` + the 8
`optionalDependencies` pins), suffix-stripped BASE to reported strings
(`sassVersion`, `compilerVersion`) and the `sass-embedded` devDep.
Anchored, fails loudly on drift. Then: triage its sweep report,
`CHANGELOG.md` entry, lockfile re-sync
(`npm install --package-lock-only` in `embedded-host-node-go/` — on a
prerelease this fails if upstream has not published the matching
`sass-embedded` yet; hold the devDep at the last published upstream
instead), move `PORTED_FROM` + submodule pins, commit. Registry writes are
tag-gated — pushes without tags only build/test/pack/upload artifacts.

## First release (alpha0, manual registries)

Bootstraps everything before automation can use it. No tag at any point.

1. Publish-day setup: uncomment the `id-token` + `attestations`
   permissions and the caches in `release.yml` (one block each), re-enable
   the `cache` lines in `ci.yml`. Re-enable the parked workflow
   (`release.yml.disabled` back to `release.yml`) — need the build matrix
   for artifacts.
2. Bump to `1.104.0-alpha0`, CHANGELOG, lockfile re-sync, commit, push.
3. CI green → `gh run download <run-id> -D alpha-artifacts` (9 npm tgzs +
   8 release archives + checksums).
4. Classic token in env (`NODE_AUTH_TOKEN`; OIDC never serves hand runs).
   Rotate/delete after.
5. `npm publish --tag next` from the downloaded tarballs: 8 platform
   packages first, then the wrapper. Never omit `--tag` on a prerelease.
6. Optional paper trail: `gh release create v1.104.0-alpha0 --draft
   --prerelease --generate-notes` + upload.
7. Set up trusted publishing now (add the repo as a trusted publisher on
   the `sass-embedded-go*` packages; nothing is parked in secrets
   beforehand) — prerequisite for every flow below.

## CI prerelease (alpha1, `1.105.0-alpha0`, …)

Standing rehearsal mechanism, not a one-off: proves the full automation
before each final. Needs trusted publishing live (preceding step).

1. Bump, CHANGELOG, lockfile re-sync, commit, push.
2. CI green → tag (e.g. `v1.104.0-alpha1`), push tag.
3. CI creates the release as **draft** (`--prerelease`), attaches archives
   + tarballs, attests the archives (check with `gh attestation verify
   <archive> -R <owner/repo>`), and publishes npm under `next` (derived
   from the `-` in the tag). Edit notes, publish the draft when ready
   (email fires then).

## Final release

Same as CI prerelease, minus the suffix:

1. Bump (no suffix), CHANGELOG, lockfile re-sync, commit, push.
2. CI green → tag `v<version>`, push tag.
3. CI publishes under `latest` (npm default tag) + attaches + attests;
   draft → edit → publish.
