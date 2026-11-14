# Porting new Dart Sass changes — Go port

How this port stays in sync with the Dart Sass reference implementation
(dart-sass). This is a **manual** procedure — there is no sync automation.
Run the commands below from the repository root.

## Start here: an `upgrade to <version>` request

A request like **“upgrade to 1.108.2”** is the complete instruction. It means:
discover the release span, port every functional unit, commit each unit,
create one Go commit per rust-sass commit (including its release commits),
push the feature branch, wait for branch CI, fast-forward and push `main`,
wait for main CI, tag the release, push the tag, and wait for tag CI.

No plan file is needed. This document plus the repository state are
authoritative; do not wait for a separate plan. “Stop before main” or “port
the units only” in the request withholds the corresponding release steps.

The first commands are:

```sh
cd /home/luka/projects/go-sass
git fetch origin
git status --short --branch
git switch main && git pull --ff-only
git switch -c upgrade/sass-<version> origin/main
```

Then follow [Target discovery and pin resolution](#target-discovery-and-pin-resolution),
[Commit-by-commit sync and release checkpoints](#commit-by-commit-sync-and-release-checkpoints-mandatory),
and [Review protocol](#review-protocol). Every commit — unit or release —
stops for review first.

## The model

- The port tracks one dart-sass commit at a time: `PORTED_FROM` (hash only,
  must agree with the `dart-sass/` submodule pin) + `docs/upstream.md`
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
  `async_evaluate.dart`); the sync twin is auto-generated from the async one
  (`tool/grind/synchronize.dart`: `async_evaluate→evaluate`,
  `async_compile→compile`, `async_environment→environment`,
  `async_import_cache→import_cache`). Go is sync-only: read the **async**
  file and ignore the sync twin's diff entirely. The sync file is generated
  output; the two twins resolve to the same Go file.

## No hacks, no shortcuts

Go code must match Dart **line by line**. Port the same branches, checks,
helper calls, messages, spans, ordering, and error paths. Do not combine
conditions, skip guards, replace a helper with a Go idiom, or “simplify”
control flow. Tests passing is not evidence of parity.

The only allowed differences are language mechanics: Dart exceptions become
Go `error` returns, nullable types become pointers, and names follow Go
conventions. Any behavior or structure that cannot match Dart is a
divergence and belongs in `docs/divergences.md` — never preserved silently.

If a unit exposes a pre-existing Go shortcut in the code it touches, fix it in
that same unit; do not leave it for a later cleanup.

## Porting standard

Ported code reproduces the Dart unit's algorithm. Go's types, error returns,
and package layout force some reshaping — that is expected. The rules below
are what stays fixed.

### Logic fidelity

For every Dart function or method the unit touches, the Go counterpart makes
the same decisions and performs the same effects, in the same order, on the
same values:

- every guard, branch, loop bound, early return, and error site has a Go
  counterpart producing the same outcome in the same order;
- the same value and span reach each call — a fallback where Dart re-evaluates
  an expression is a change;
- a step that can fail or allocate observably runs the same number of times;
- unreachable branches still match Dart's computation ("verify already
  guarantees it" is not a reason to drop one);
- a Dart callback parameter stays a Go parameter (or an explicit callback
  argument); it is not folded into the callee.

### Naming fidelity

Identifiers follow Dart wherever Go permits:

- **Functions/methods** convert mechanically (`lowerCamelCase` →
  `ExportedCamelCase` for public API, `lowerCamelCase` for package-private
  code). A leading `_` is dropped: Go package privacy replaces it. No semantic
  renames. The conversions are listed in `patterns.md` §11.
- **Arguments** keep their names after conversion (`nodeWithSpan` →
  `nodeWithSpan`). The only permitted additions are the receiver that was
  `this` in Dart and context parameters a Go signature needs (for example a
  `*EvaluateVisitor` where Dart uses an implicit `this`).
- **Locals** keep their names after conversion (`nameIdent` stays `nameIdent`).
  Rename only to avoid a Go keyword or a shadow that will not compile.

### Reshaping that is allowed

Method → function with an explicit receiver parameter; block → helper called
exactly where the block was, with the same inputs and the same result (pure
extraction); merged tightly-coupled steps when each branch still has a 1:1
Dart counterpart. When one Dart method becomes several Go functions, name the
primary part after it and note the split in the file's `// dart-source:`
header.

Re-implementing a Dart helper, inlining one into several call sites, or
extracting logic that changes what runs where is not reshaping — it is a port
bug even when output matches. A green `sass-spec` run does not establish
fidelity: the specs exercise behavior, and drift hides in paths they do not
reach. Duplicate implementations of one Dart declaration (two `verify`-style
helpers, say) are a finding under this rule.

## Target discovery and pin resolution

`TARGET` below is the upstream release tag, for example `1.108.2`. Run from
the repository root.

```sh
git -C dart-sass fetch origin --tags
git -C sass-spec fetch origin
git -C embedded-host-node fetch --tags origin
git -C sass fetch --tags origin

BASE=$(cat PORTED_FROM)
git submodule status dart-sass embedded-host-node sass sass-spec
git -C dart-sass rev-parse "$TARGET^{commit}"
git -C dart-sass log --oneline "$BASE..$TARGET" -- lib/ CHANGELOG.md
```

Classify the commits with [Sync procedure](#sync-procedure) step 1. Every
user-visible `CHANGELOG.md` entry must map to a functional unit or to an
explicitly skipped unit. A release span with no `lib/` changes is a patch
release with no port units.

Resolve the release pins as follows:

- **dart-sass**: the commit the release tag points at
  (`git -C dart-sass rev-parse "$TARGET^{commit}"`). `PORTED_FROM` must equal
  it.
- **embedded-host-node**: the **peeled commit** of the matching host tag
  (`git -C embedded-host-node rev-parse "$TARGET^{commit}"`), not the
  annotated tag object. Host tags are `Update Dart Sass version and release`
  commits.
- **sass**: the commit for the embedded protocol version. Read the version
  from the published `sass-embedded@$TARGET` package (`protocol-version`) or
  from `sass/spec/EMBEDDED_PROTOCOL_VERSION`, then
  `git -C sass rev-parse "embedded-protocol-<version>"`. Move this pin only
  when the protocol version changes.
- **sass-spec**: the spec commit belonging to the ported units. Inspect
  `git -C sass-spec log --oneline <current-pin>..origin/main -- spec/` and
  match commits to the units' tests. Advance per unit and never past an
  unported unit; a release has no single upstream “spec tag”.
- **version surfaces**: `cmd/go-sass/options.go` (`sassVersion`),
  `embedded/isolate_dispatcher.go` (`compilerVersion`/`protocolVersion`),
  `embedded-host-node-go/package.dist.json` (wrapper + platform pins), and
  `embedded-host-node-go/package.json` (the `sass-embedded` devDep).

Record the resolved pins in the release ledger. If a pin already matches,
leave it untouched.

## Commit-by-commit sync and release checkpoints (mandatory)

Do not batch or re-split the rust-sass commit stack. Create one Go commit per
rust-sass commit, in the same order, including its release commits. Do not
invent a release commit that rust-sass does not have.

### Per-unit commit

1. Check out the upstream commit and port only its `lib/` diff.
2. For Dart sync/async twin files, read the **async** file only (the sync
   file is generated).
3. Port line-by-line against the [Porting standard](#porting-standard): same
   logic, helpers, messages, ordering, spans, and error paths; same function,
   argument, and local names where Go permits. A Go-only guard, fallback, or
   simplification is a bug even if the tests pass.
4. Add failing-first internal tests in the touched package, covering the new
   behavior and its exact messages and error spans.
5. Advance the `sass-spec` pin to that unit's spec commit, never past an
   unported unit. If the unit has no spec commit, leave the pin unchanged.
6. Verify locally:
   - `go build ./... && go vet ./...`
   - `gofmt -l .` (must print nothing)
   - the touched package's tests
   - the affected `sass-spec` directories
   - differential A/B against the target dart-sass binary
7. Fold any pre-existing port bug or shortcut exposed by this unit into this
   same commit. Do not create a standalone fix commit before the branch has
   been pushed.
8. Advance `PORTED_FROM` and the `dart-sass` submodule to the ported commit.
9. Commit only that unit.

### Release checkpoint (every rust-sass release commit)

A rust-sass patch release still gets its own Go release commit and tag.

1. Resolve and check out the release pins from
   [Target discovery and pin resolution](#target-discovery-and-pin-resolution):
   `dart-sass`/`PORTED_FROM` to the tag commit, `embedded-host-node` to the
   peeled host tag commit, `sass` to the protocol tag commit (usually
   unchanged), and `sass-spec` to the last spec commit for the ported units.
2. Run `tools/bump-version.sh <version>`.
3. Update `protocolVersion` in `embedded/isolate_dispatcher.go` when the
   embedded protocol changed, and regenerate the bindings (see
   [Sync procedure](#sync-procedure) step 4).
4. Sweep the current version and update the release surfaces:
   - `git grep -n -F '<old-version>'`; current references move, historical
     CHANGELOG sections stay.
   - `README.md`, `docs/upstream.md` (commit **and date**),
     `docs/review.md` (`PORTED_FROM` hash), `docs/ref/sass-embedded-go.md`
     (host tag/hash), `docs/ci.md`, `docs/release.md`,
     `docs/architecture.md`, `docs/divergences.md`,
     `docs/critical-invariants.md`, and the remaining `docs/ref/*` version
     references.
   - npm oracle pins: `embedded-host-node-go/package.json`
     (`sass-embedded`) and `embedded-host-node-go/package.dist.json`
     (wrapper version + the exact optional platform dependencies), then
     refresh the lockfile with
     `npm install --package-lock-only --ignore-scripts` in that directory.
   - `CHANGELOG.md`: add the section from upstream. Keep historical
     sections unchanged.
5. Rebuild and re-test the Node host:
   ```sh
   cd embedded-host-node-go
   npm install
   npm run build
   npm test
   ```
6. Run the full local battery:
   ```sh
   go build ./... && go vet ./...
   gofmt -l .
   go test ./...
   SASS_SPEC_PATH=../sass-spec/spec go test -tags=spec -timeout=300s ./spec/
   # byte-identity go==dart on bootstrap/huge/huge10 (see CONTRIBUTING.md)
   ```
   Re-run the embedded round-trip harness if `embedded/`, `compile/`, or the
   CLI is affected.
7. Stage the release surfaces explicitly, run the
   [Review protocol](#review-protocol), and commit only after approval.
8. Push the short-lived feature branch (for example
   `upgrade/sass-<version>`) and wait for branch CI. `ci.yml` runs on every
   branch push.
9. After branch CI is green, fast-forward `main` to the feature branch tip
   and push it unless the request said “stop before main”. Wait for main CI.
   If it fails:
   - never amend a pushed commit,
   - add fix commits, re-run the gates, and wait for the next CI run.
10. After main CI is green, create the local annotated tag `v<version>` and
    push it.
11. Wait for the tag/release CI. Only then start the next upstream unit.

### Push and tag ownership

An explicit **“upgrade to `<version>`”** authorizes the full release flow:
feature-branch push, branch CI, fast-forwarding `main`, main push, main CI,
tag push, and tag CI. A request that says “stop before main” or “port the
units only” withholds the corresponding steps.

The agent may push short-lived feature branches. The agent **never pushes
`main`**. Every release checkpoint ends with a review stop before the main
push; the user reviews and pushes. Once main CI is green, the agent creates
and pushes the tag, then waits for tag CI. Tags are created only after main
CI is green, so they include any CI fix commits.

While the repository is private, tag pushes do not publish: the `id-token`
and `attestations` permissions and the npm trusted publishers are disabled
(see [`ci.md`](ci.md)). Stop before the tag step in that state.

### Branch policy

`ci.yml` runs on every branch push. Use a short-lived branch (for example
`upgrade/sass-<version>`) for each release checkpoint, fast-forward `main` to
it after branch CI is green, and delete the branch when the upgrade is
complete. `release.yml` stays `main`/tag-only; it has packaging and release
side effects.

### Rebuilding a commit stack

If the history must be rebuilt before release:

1. Keep the old stack on a backup branch.
2. Create the new branch from `origin/main`.
3. Recreate the commits, folding fixes into the relevant commit.
4. Delete and recreate local `v1.10X.*` tags on the rebuilt release commits.
5. Follow the normal review/push/tag checkpoints above.

## Review protocol

Every unit commit and every release commit is reviewed before it is created.

1. Stage explicit paths (`git add <paths>`, never `git add -A`).
2. Run the unit or release gates: fmt, vet, tests, focused `sass-spec`
   directories, and differential A/B where the behavior is observable. Then
   diff the Go against the Dart function line-by-line; tests passing is not
   proof of parity, and hacks or shortcuts are rejected.
3. Show the staged summary (`git diff --cached --stat`), the pins being moved,
   the new/changed internal tests, the focused spec results, and any
   differential output. For units that touch ported declarations, also list
   each touched Dart function/method and its Go counterpart — or the
   `review.md` §3 adaptation that permits the difference.
4. Stop and wait for approval. Do not `git commit` until the user approves.
5. After the commit, continue with the next unit or the release steps.

For release commits, the same review applies after the full battery, before
the branch push.

## Sync procedure

1. **Fetch and locate the functional units.** `dart-sass/` is a submodule
   pinned at the tracked commit (`PORTED_FROM` must agree — see
   `docs/upstream.md`).

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
   smuggle functional changes (observed: a “prefer interpolation” lint
   commit that also swapped two deprecation-message branches and added an
   empty-args guard). The CHANGELOG cross-check below is the backstop — any
   user-visible entry without a matching functional unit means the filter
   missed something: find it with `git log -S"<distinctive string>" -- lib/`
   on the entry's behavior, not by re-reading subjects.

   Cross-check against the `CHANGELOG.md` entries in the span: every
   user-visible entry must map to a unit. If
   `git -C dart-sass log --oneline $(cat PORTED_FROM)..<target> --grep="<keyword>"`
   for a changelog entry returns empty, it predates `PORTED_FROM` and is
   already ported — move on.

   CLI-only/tooling-only units with no Go counterpart are **not ported**.
   Standing skips: `--watch` / `--poll` / `--update` / `--interactive`
   handling — the Go CLI does not implement them, so watch fixes are always
   skipped. Re-evaluate only if the CLI ever gains them.

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
   Common causes: the changed file is the generated sync twin of an async
   file (read the async diff; both resolve to the same Go file), a
   barrel/export file (no Go counterpart by policy), a JS/wasm-only file,
   or a skipped unit (step 1). Only if none of these apply is it a genuinely
   new file (add a Go file, honoring the package-layout rules in
   `architecture.md` §13) or a removed one (drop the Go file) — decide
   explicitly, never ignore. Never leave a stale `dart-source:` path: the
   header sweep in the release process relies on them resolving.

   Handle Dart renames/additions/deletions the same way: add or drop the Go
   file, and update the annotations.

3. **Check dependency bumps.** `pubspec.yaml` pins the external packages
   (`source_span`, `string_scanner`, `source_maps`, `path`, the Dart SDK). If
   a runtime pin moved, re-port the files listed in `docs/upstream.md` from the
   new release and update that table. Dev-only moves (analyzer, lints,
   grinder, CI actions) and SDK-floor moves are ignored unless they change
   language semantics the port relies on.

4. **Regenerate protocol bindings if the proto changed.**

   The bindings generate from the canonical `sass/spec/embedded_sass.proto`
   (the `sass` language-spec submodule — never a vendored copy):

   ```sh
   git -C sass diff --name-only <old-pin>..<new-pin> -- spec/embedded_sass.proto
   # if non-empty (or the dart-sass span touched the protocol):
   go generate ./embedded/
   ```

   Move the `sass` pin with the protocol change (it otherwise stays put).
   Bump `protocolVersion` in `embedded/isolate_dispatcher.go` in lockstep
   with the protocol version. (Purely mechanical `protofier` churn with an
   unchanged `.proto` needs no regen.)

5. **Port one unit at a time.** Read the unit's Dart diff; apply the
   equivalent change to the Go files from step 2, statement-for-statement:
   same names, same check order, same error variants and spans. Keep
   `// Matches Dart:` / `// dart-source:` annotations honest. New files get
   the standard header: the Dart source's license lines plus
   `// Ported and rearchitected for Go by Luka Zakrajsek.` Repro first
   (differential A/B vs `dart-sass/` on the unit's behavior), fix narrowly,
   add the regression test before moving to the next unit.

6. **Verify per unit, then globally.** Per unit: the touched package's tests
   plus the relevant spec subset
   (`SASS_SPEC=<subpath> go test -tags=spec -timeout=30s ./spec/`), and the
   differential A/B for that behavior. After all units: the full battery —

   ```sh
   go build ./... && go vet ./...
   gofmt -l .
   go test ./...                                                     # unit
   SASS_SPEC_PATH=../sass-spec/spec go test -tags=spec -timeout=300s ./spec/   # spec suite
   # byte-identity go==dart on bootstrap/huge/huge10 (see CONTRIBUTING.md)
   # embedded round-trip harness (drive the npm host against ./sass --embedded)
   ```

   Plus a re-run of the embedded harness if `embedded/`, `compile/`, or the
   CLI is affected.

7. **Record and commit.** Update `PORTED_FROM` to the new commit hash and
   `docs/upstream.md` (commit, version, any external package pins). The
   release commit message is the **ledger**: list each functional unit ported
   (`<sha> <subject>`) and the submodule pins moved (`dart-sass`,
   `sass-spec`, `sass`, `embedded-host-node`). Keep it free-form — a plain
   list is enough, no template. Commit each functional unit separately; commit the release
   changes at the release checkpoint. Then follow the version-bump checklist
   in `CONTRIBUTING.md`.

## Ecosystem pins (checked in step 1, moved per unit — not once at the end)

dart-sass does **not** pin sass-spec: its CI clones floating `main` HEAD. So
there are no upstream pins to copy — define per-unit pins instead. Each
behavior in the span has a matching spec-side commit, and every dart-sass
behavior commit predates (or is same-day-earlier than) its spec commit, so
chronological lockstep keeps every step green:

- Work the units in **chronological** order (this satisfies code
  dependencies like “serialize change A before serialize change B” as long
  as the order respects them — check before committing to it).
- After porting each unit, advance the `sass-spec` submodule pin **to**
  that unit's spec commit — never past an unported unit's spec commit, or
  the suite carries expectations for behavior you haven't ported yet.
- Then the full suite must be green before moving on. Shared files touched
  by several spec commits (e.g. `mix/missing.hrx`) resolve themselves:
  each pin's version was authored against dart-sass main at that date,
  which contained exactly the behaviors released to date — the same set
  you have ported.
- Units whose spec delta is js-api-spec-only (no `.hrx` change) still
  advance the pin for monotonicity, but are verified by fixtures +
  in-module tests + differential A/B.
- `sass` (language spec): moves when a unit is spec-backed (check for
  flushed proposals in the span), and when the embedded protocol version
  changes.
- `embedded-host-node`: moves to the host tag matching the new dart-sass
  version (once per release, not per unit — host releases track versions).

## License/header rules when porting

- Header = the ported Dart file's exact 3-line license header + `//\n// Ported
and rearchitected for Go by Luka Zakrajsek.`
- Files ported from external packages carry that package's BSD header (see
  `docs/upstream.md`), never a Google MIT header.
- Original Go code (no Dart source) gets no Google header.
