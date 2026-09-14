# Porting new Dart Sass changes — Go port

How this port stays in sync with the Dart Sass reference implementation
(dart-sass). This is a **manual** procedure — there is no sync automation.
Run the commands below from the repository root.

## The model

- The port tracks one dart-sass commit at a time: `PORTED_FROM` (hash
  only, must agree with the `dart-sass/` submodule pin) + `docs/upstream.md`
  (human-readable: commit, version, external package pins).
- Porting is **dart → go directly** — read the Dart source in the
  `dart-sass/` submodule; never chain through another port. (Byte-identical
  output against Dart is verified via the oracles in the test battery — a
  cross-check, not a porting path.)
- Cadence: sync in **functional units** (see step 1). A release span
  contains mostly mechanical noise (Dart syntax modernization,
  analyzer/lint/reformats, CI/dependabot/version bumps) with zero behavior
  change. Each functional PR is ported in isolation from its own diff —
  never from the whole-span range diff (which buries behavior under
  churn), and never by walking every commit.
- Dart keeps separate sync/async twin files (e.g. `evaluate.dart` /
  `async_evaluate.dart`); the async twin is auto-generated from the sync one
  upstream. Go is sync-only: read the **sync** file and ignore the async
  twin's diff entirely.

## Sync procedure

1. **Fetch and locate the functional units.** `dart-sass/` is a submodule pinned at the
   tracked commit (`PORTED_FROM` must agree — see `docs/upstream.md`).

   ```sh
   git -C dart-sass fetch origin
   git -C dart-sass log --oneline $(cat PORTED_FROM)..<target> -- lib/
   ```

   dart-sass lands work as **squash commits on main with the PR number as a
   message suffix** (`<title> (#NNNN)`); release spans contain almost no
   merge commits, so merge topology carries no information — group by those
   squash commits, not by merges. Classify each commit by subject + stat;
   mechanical commits are recognizable by subject alone (`shorthand`,
   `constructor`, `lint`, `reformat`, `analyzer`, `Bump …`, CI/actions,
   release plumbing):

   ```sh
   git -C dart-sass show --stat --oneline <sha> | head -30
   git -C dart-sass show <sha> -- lib/ test/ | head -100
   ```

   A mechanical commit's `lib/` diff is syntax/style churn only; a
   functional one changes behavior and usually tests. Each surviving commit
   is one **functional unit**, ported in isolation from its own
   `git show <sha> -- lib/ test/` diff.

   Subject-based filtering is **not sufficient**: mechanical commits can
   smuggle functional changes (observed: a "prefer interpolation" lint
   commit that also swapped two deprecation-message branches and added an
   empty-args guard). The CHANGELOG cross-check below is the backstop —
   any user-visible entry without a matching functional unit means the
   filter missed something: find it with `git log -S"< distinctive
string>" -- lib/` on the entry's behavior, not by re-reading subjects.

   Cross-check against the `CHANGELOG.md` entries in the span: every
   user-visible entry must map to a unit. If
   `git -C dart-sass log --oneline $(cat PORTED_FROM)..<target> --grep="<keyword>"`
   for a changelog entry returns empty, it predates `PORTED_FROM` and is
   already ported — move on.

   CLI-only/tooling-only units with no Go counterpart are **recorded as
   skipped** in the bump ledger (step 7), not ported and not silently
   dropped. Standing skips: `--watch` / `--poll` / `--update` /
   `--interactive` handling — the Go CLI does not implement them, so watch
   fixes are always skipped. Re-evaluate only if the CLI ever gains them.

   To advance the pin: `git -C dart-sass checkout <new-commit>` (full clone —
   any commit is inspectable).

2. **Find the affected ported files (per unit).** Each ported file carries a
   `// dart-source: lib/src/...` annotation. Map the unit's changed Dart
   files to Go counterparts, matching on the **full `lib/src/...` path**,
   not the basename (basenames over-match — e.g. `color.dart` appears in
   many directories):

   ```sh
   git -C dart-sass show --name-only --format= <sha> -- lib/ \
     | while read d; do
         rg -l "dart-source:.*$d" --glob '*.go' . | grep -v _test || echo "NO GO PORT: $d"
       done
   ```

   A `NO GO PORT` hit is usually a **false alarm, not a missing file**.
   Common causes: the changed file is an async twin (Go ports sync only —
   both twins resolve to the same Go file, and only the sync diff matters),
   a barrel/export file (no Go counterpart by policy), or a skipped unit
   (step 1). Only if none of these apply is it a genuinely new file (add a
   Go file, honoring the package-layout rules in `architecture.md` §13) or a
   removed one (drop the Go file) — decide explicitly, never ignore. Never
   leave a stale `dart-source:` path: the header sweep in the release
   process relies on them resolving.

   Handle Dart renames/additions/deletions the same way: add or drop the Go
   file, and update the annotations.

3. **Check dependency bumps.** `pubspec.yaml` pins the external packages
   (`source_span`, `string_scanner`, `source_maps`, `path`, the Dart SDK). If a
   runtime pin moved, re-port the files listed in `docs/upstream.md` from the new
   release and update that table. Dev-only moves (analyzer, lints, grinder,
   CI actions) and SDK-floor moves are ignored unless they change language
   semantics the port relies on.

4. **Regenerate protocol bindings if the proto changed.**

   The bindings generate from the canonical `sass/spec/embedded_sass.proto`
   (the `sass` language-spec submodule — never a vendored copy):

   ```sh
   git -C sass diff --name-only <old-pin>..<new-pin> -- spec/embedded_sass.proto
   # if non-empty (or the dart-sass span touched the protocol):
   go generate ./embedded/
   ```

   Move the `sass` pin with the protocol change (it otherwise stays put).
   Bump `compilerVersion`/`protocolVersion` in
   `embedded/isolate_dispatcher.go` in lockstep with the dart-sass version.
   (Purely mechanical `protofier` churn with an unchanged `.proto` needs no
   regen.)

5. **Port one unit at a time.** Read the unit's Dart diff; apply the equivalent change to the Go files
   from step 2, statement-for-statement: same names, same check order, same
   error variants and spans. Keep `// Matches Dart:` / `// dart-source:` annotations honest. New files get the standard header: the Dart source's
   license lines plus `// Ported and rearchitected for Go by Luka Zakrajsek.`
   Repro first (differential A/B vs `dart-sass/` on the unit's behavior),
   fix narrowly, add the regression test before moving to the next unit.

6. **Verify per unit, then globally.** Per unit: the touched package's tests
   plus the relevant spec subset (`SASS_SPEC=<subpath> go test -tags=spec
-timeout=30s ./spec/`), and the differential A/B for that behavior.
   After all units: the full battery —

   ```sh
   go build ./... && go vet ./...
   gofmt -l .
   go test ./...                                                     # unit
   SASS_SPEC_PATH=../sass-spec/spec go test -tags=spec -timeout=30s ./spec/   # spec suite
   # byte-identity go==dart on bootstrap/huge/huge10 (see CONTRIBUTING.md)
   # embedded round-trip harness (drive the npm host against ./sass --embedded)
   ```

   Plus a re-run of the embedded harness if `embedded/`, `compile/`, or the
   CLI is affected.

7. **Record and commit.** Update `PORTED_FROM` to the new commit hash and
   `docs/upstream.md` (commit, version, any external package pins). The bump
   commit message is the **ledger**: list each functional unit ported
   (`<sha> <subject>`), each unit explicitly skipped and why, and the
   submodule pins moved (`dart-sass`, `sass-spec`, `sass`). Keep it free-form —
   a plain list is enough, no template. Commit the port changes and the tracking
   update together. Then follow the version-bump checklist in `CONTRIBUTING.md`
   (`compilerVersion`, CLI `sassVersion`, spec pin) — its mechanical
   version-string step is `tools/bump-version.sh <version>` (patch and
   `-alpha` bumps run the script directly; a re-sync ends here).

## Ecosystem pins (checked in step 1, moved per unit — not once at the end)

dart-sass does **not** pin sass-spec: its CI clones floating `main` HEAD.
So there are no upstream pins to copy — define per-unit pins instead. Each
behavior in the span has a matching spec-side commit, and every dart-sass
behavior commit predates (or is same-day-earlier than) its spec commit, so
chronological lockstep keeps every step green:

- Work the units in **chronological** order (this satisfies code
  dependencies like "serialize change A before serialize change B" as long
  as the order respects them — check before committing to it).
- After porting each unit, advance the `sass-spec` submodule pin **to**
  that unit's spec commit — never past an unported unit's spec commit, or
  the suite carries expectations for behavior you haven't ported yet.
- Then the in-scope suite must be green before moving on.
- `sass` (language spec): moves when a unit is spec-backed.

## License/header rules when porting

- Header = the ported Dart file's exact 3-line license header + `//\n// Ported
and rearchitected for Go by Luka Zakrajsek.`
- Files ported from external packages carry that package's BSD header (see
  `docs/upstream.md`), never a Google MIT header.
- Original Go code (no Dart source) gets no Google header.
