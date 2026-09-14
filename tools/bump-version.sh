#!/bin/bash
# tools/bump-version.sh <new-version> — set the release version repo-wide.
#
# <new-version> is full semver with an optional prerelease suffix, e.g.
# `1.104.1` or `1.105.0-alpha0`. Run from anywhere; operates on the repo root.
#
# Scope: patch releases and -alpha rehearsals only — version-string moves
# with no behavior change behind them. Anything crossing a minor/major
# boundary with ported behavior rides the porting.md re-sync instead (its
# step 7 ends at the version-bump checklist); this script only moves the
# strings, so a boundary-crossing invocation prints a reminder and the
# operator confirms the sync train owns the behavior side.
#
# Split model (see docs/release.md):
# - FULL version (as given): the published identity —
#   embedded-host-node-go/package.dist.json (`version` + the 8
#   `optionalDependencies` pins, which build.mjs derives platform manifests
#   from, and which the release.yml version guard compares against the tag).
# - BASE version (suffix stripped): reported strings (`sassVersion`,
#   `compilerVersion`) and the `sass-embedded` devDep — what Dart would
#   report / what upstream publishes, so hosts and --version parsers keep
#   seeing exact values.
# Everything else (docs prose, PORTED_FROM, submodule pins, historical
# markers, 0.0.0 sources, lockfiles) is triaged by hand from the sweep
# report below.
set -eu

die() {
  echo "bump-version: $*" >&2
  exit 1
}

[ $# -eq 1 ] || die "usage: tools/bump-version.sh <new-version> (e.g. 1.105.0-alpha0, 1.105.0)"
NEW="$1"
BASE="${NEW%%-*}"

[[ "$NEW" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]] \
  || die "'$NEW' is not valid semver (want X.Y.Z[-prerelease])"

cd "$(dirname "$0")/.."

OLD=$(node -p "require('./embedded-host-node-go/package.dist.json').version")
[ -n "$OLD" ] || die "could not detect current version from package.dist.json"
BASE_OLD="${OLD%%-*}"
[ "$NEW" != "$OLD" ] || die "already at $OLD (nothing to do)"

# Scope guard (non-blocking): a minor/major crossing means ported behavior
# is almost certainly involved — that bump belongs to the porting.md
# re-sync, and this script only moves the strings. Proceeds anyway: the
# sanctioned exception is the -alpha rehearsal of the next version.
mm() { # 1.105.0-alpha0 -> 1.105
  local v="${1%%-*}"
  echo "${v%.*}"
}
if [ "$(mm "$NEW")" != "$(mm "$OLD")" ]; then
  echo "NOTE: $OLD -> $NEW crosses a minor/major boundary."
  echo "Behavior-changing bumps ride the porting.md re-sync (step 7); this"
  echo "script only moves version strings — patch releases and -alpha"
  echo "rehearsals. Continuing with the strings."
fi

# Portable in-place sed (GNU vs BSD/macOS).
if sed --version >/dev/null 2>&1; then
  sedi() { sed -i "$@"; }
else
  sedi() { sed -i '' "$@"; }
fi

# Replace, failing loudly if the anchor pattern stopped matching (drift).
rep() {
  file="$1"
  pattern="$2"
  replacement="$3"
  if ! grep -q "$pattern" "$file"; then
    die "$file: pattern not found (drift?): $pattern"
  fi
  # Escape `&` (whole-match in sed replacements); backslashes pass through.
  replacement="${replacement//&/\\&}"
  sedi "s/$pattern/$replacement/g" "$file"
  echo "  updated $file"
}

echo "bump-version: $OLD -> $NEW (base $BASE_OLD -> $BASE)"
echo "== full version (published identity) =="
rep embedded-host-node-go/package.dist.json "\"$OLD\"" "\"$NEW\""
echo "== base version (reported strings, upstream devDep) =="
rep embedded-host-node-go/package.json "\"sass-embedded\": \"$BASE_OLD\"" "\"sass-embedded\": \"$BASE\""
rep cmd/go-sass/options.go "const sassVersion = \"$BASE_OLD\"" "const sassVersion = \"$BASE\""
rep embedded/isolate_dispatcher.go "var compilerVersion = \"$BASE_OLD\"" "var compilerVersion = \"$BASE\""

echo "== sweep report (triage by hand — code must be clean, historical prose stays) =="
git grep -F -l "$OLD" -- . ':!dart-sass' ':!sass-spec' ':!sass' ':!bootstrap-main' ':!embedded-host-node' || true

echo "== next steps =="
echo "  1. Add a CHANGELOG.md entry (human prose, not scripted)."
echo "  2. Re-sync the lockfile: (cd embedded-host-node-go && npm install --package-lock-only)."
echo "     On a prerelease this fails if upstream has not published sass-embedded@$BASE"
echo "     yet — then hold the devDep at the last published upstream and re-sync;"
echo "     see docs/release.md."
echo "  3. Leave PORTED_FROM and the submodule pins alone: pins move only via"
echo "     the porting.md re-sync (step 7), never for a patch/-alpha bump. If"
echo "     this version rides a re-sync, they move there (see CONTRIBUTING.md"
echo "     checklist)."
if [[ "$NEW" == *-* ]]; then
  echo "  4. Prerelease: commit, push, wait for CI, then tag v$NEW — CI"
  echo "     publishes registries under \`next\` (needs trusted publishing"
  echo "     live). The very first release is manual instead: no tag, pull"
  echo "     CI artifacts, publish by hand — see docs/release.md."
else
  echo "  4. Final: commit, push, wait for CI, then tag v$NEW and push the tag;"
  echo "     see the release runbook in docs/release.md."
fi
