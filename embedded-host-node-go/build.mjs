// Build-time assembly for the sass-embedded-go npm package.
//
// Assembles (never vendors) the genuine embedded-host-node source with a
// one-line `compiler-module.ts` template swap, so the stock resolution logic
// finds OUR platform packages (`sass-embedded-go-<platform>-<arch>`) while
// sharing zero modules with an app's own `sass-embedded` install.
//
// Steps: compiler binary (dev: `go build -tags cgomath` for the host; CI:
// prebuilt per-platform cgo binaries from GO_SASS_DIST, staged flat as
// go-sass-<triple>[.exe] by the release.yml build matrix) → npm install +
// compile in ../embedded-host-node (in place; its tree stays pristine —
// dist/ is gitignored) → copy dist/ here → apply the template swap →
// assemble the local platform package(s) + link the host one into
// node_modules (same mechanism as registry publish, unpublished locally).
//
// Usage:
//   npm run build                     (host platform, local `go build`)
//   npm run build -- --platforms=all  (all 8 triples, needs GO_SASS_DIST
//                                     pointed at the release.yml prebuilt/
//                                     staging dir — the release.yml flow)

import {execFileSync} from 'node:child_process';
import * as fs from 'node:fs';
import {createRequire} from 'node:module';
import * as p from 'node:path';
import {fileURLToPath} from 'node:url';

const require = createRequire(import.meta.url);

const pkgDir = p.dirname(fileURLToPath(import.meta.url));
const repoRoot = p.resolve(pkgDir, '..');
const hostDir = p.resolve(repoRoot, 'embedded-host-node');
const distDir = p.resolve(pkgDir, 'dist');

function sh(cmd, args, cwd) {
  execFileSync(cmd, args, {cwd, stdio: 'inherit'});
}

// No musl detection here: on a musl host the vendored compiler-module.js
// (upstream musl branch, preserved by the template swap) resolves the
// linux-musl-* package at runtime from the Node binary's ELF interpreter.
// This only assembles the local package and links the host platform into
// node_modules, where the gnu-named package is what CI (glibc) resolves.
function triple() {
  return `${process.platform}-${process.arch}`;
}

function platformPackageName() {
  return `sass-embedded-go-${triple()}`;
}

// The 8 shipped triples. The musl entries are genuine musl-linked static
// binaries (built with musl-gcc in release.yml), not relabeled gnu
// binaries. Linux manifests declare their libc (glibc/musl strings,
// mirroring upstream sass-embedded) so npm installs the right variant per
// host; darwin/win carry no libc field, as upstream. Android/riscv/armv7
// triples arrive with full upstream parity later.
const TRIPLES = [
  {triple: 'darwin-arm64', os: 'darwin', cpu: 'arm64'},
  {triple: 'darwin-x64', os: 'darwin', cpu: 'x64'},
  {triple: 'linux-x64', os: 'linux', cpu: 'x64', libc: 'glibc'},
  {triple: 'linux-arm64', os: 'linux', cpu: 'arm64', libc: 'glibc'},
  {triple: 'linux-musl-x64', os: 'linux', cpu: 'x64', libc: 'musl'},
  {triple: 'linux-musl-arm64', os: 'linux', cpu: 'arm64', libc: 'musl'},
  {triple: 'win32-x64', os: 'win32', cpu: 'x64'},
  {triple: 'win32-arm64', os: 'win32', cpu: 'arm64'},
];

const modeAll = process.argv.includes('--platforms=all');
const distDir0 = process.env.GO_SASS_DIST;
if (modeAll && !distDir0) {
  throw new Error('`--platforms=all` needs GO_SASS_DIST pointed at the release.yml prebuilt/ staging dir');
}

// Locate the compiler binary for one triple in the release.yml staging dir
// (flat `go-sass-<triple>[.exe]`, present for both snapshot and tag runs —
// one code path for tags and non-tags). The error lists the dir contents —
// no guessing when the layout surprises us.
function findBinary(t) {
  const file = `go-sass-${t.triple}${t.os === 'win32' ? '.exe' : ''}`;
  const exe = p.join(distDir0, file);
  if (!fs.existsSync(exe)) {
    throw new Error(`no prebuilt binary ${file} in ${distDir0} (dist: ${fs.readdirSync(distDir0).join(', ')})`);
  }
  return exe;
}

// 1. Compiler binaries: one per requested triple.
const binaries = new Map(); // triple -> binary path
if (modeAll) {
  for (const t of TRIPLES) {
    binaries.set(t.triple, findBinary(t));
  }
} else {
  // -tags cgomath, like the shipped binaries: dev tests what we release.
  sh('go', ['build', '-tags', 'cgomath', '-o', p.join(pkgDir, 'platform-tmp-sass'), './cmd/go-sass'], repoRoot);
  binaries.set(triple(), p.join(pkgDir, 'platform-tmp-sass'));
}
for (const [name, binPath] of binaries) {
  if (!fs.existsSync(binPath)) {
    throw new Error(`missing compiler binary for ${name}: ${binPath}`);
  }
}

// 2. Host source deps + build, in place (node_modules/ and dist/ are
// gitignored inside the submodule — its tracked tree stays pristine).
// The host tree carries no lockfile, so plain install (our own lockfile
// pins what we test).
sh('npm', ['install', '--no-audit', '--no-fund'], hostDir);
// Vendor sources (protobuf bindings via buf, JS API from the language repo).
// --skip-compiler: we never need the Dart binary; --language-path points at
// our pinned sass submodule (all local, no network, no Dart SDK).
sh('npx', [
  'ts-node',
  './tool/init.ts',
  '--skip-compiler',
  '--language-path',
  p.resolve(repoRoot, 'sass'),
], hostDir);
sh('npm', ['run', 'clean'], hostDir);
sh('npm', ['run', 'compile'], hostDir);

// 3. Copy the built dist/ here and apply the one-line template swap.
fs.rmSync(distDir, {recursive: true, force: true});
fs.cpSync(p.join(hostDir, 'dist'), distDir, {recursive: true});
const moduleJs = p.join(distDir, 'lib', 'src', 'compiler-module.js');
let moduleSrc = fs.readFileSync(moduleJs, 'utf8');
const from = 'sass-embedded-${platform}-${arch}';
const to = 'sass-embedded-go-${platform}-${arch}';
const hits = moduleSrc.split(from).length - 1;
if (hits !== 1) {
  throw new Error(
    `template swap assertion failed: expected 1 hit, found ${hits} in ${moduleJs}`,
  );
}
moduleSrc = moduleSrc.split(from).join(to);
fs.writeFileSync(moduleJs, moduleSrc);

// Public typings: dist/types/ mirrors lib/src/vendor/sass (minus README),
// plus index.m.d.ts for the .mjs entry — the same assembly upstream
// prepare-release.ts performs (tsc emits declarations to _types/, never
// into dist/, so without this the `types` fields dangle).
// NOTE: vendor/sass is a symlink (init.ts links the language repo's
// js-api-doc); dereference into real files, since npm will not pack
// symlinks escaping the package root.
const vendorTypes = p.join(hostDir, 'lib', 'src', 'vendor', 'sass');
const distTypes = p.join(distDir, 'types');
fs.rmSync(distTypes, {recursive: true, force: true});
fs.cpSync(vendorTypes, distTypes, {recursive: true, dereference: true});
if (fs.lstatSync(distTypes).isSymbolicLink()) {
  throw new Error(`dist/types is a symlink — npm would silently skip it`);
}
fs.rmSync(p.join(distTypes, 'README.md'), {force: true});
fs.copyFileSync(
  p.join(distTypes, 'index.d.ts'),
  p.join(distTypes, 'index.m.d.ts'),
);

// Publish manifest: package.dist.json (the version authority — source
// package.json stays 0.0.0) + LICENSE + README land in dist/, so `npm
// publish ./dist` ships exactly this tree.
fs.copyFileSync(p.join(pkgDir, 'package.dist.json'), p.join(distDir, 'package.json'));
fs.copyFileSync(p.join(repoRoot, 'LICENSE'), p.join(distDir, 'LICENSE'));
fs.copyFileSync(p.join(pkgDir, 'README.md'), p.join(distDir, 'README.md'));

// 4. Assemble the platform package(s) (mirrors the registry layout the
// wrapper resolves: <pkg>/dart-sass/sass).
const wrapperVersion = JSON.parse(fs.readFileSync(p.join(pkgDir, 'package.dist.json'), 'utf8')).version;
const assembledTriples = modeAll ? TRIPLES.map((t) => t.triple) : [triple()];
for (const name of assembledTriples) {
  const goBinaryPath = binaries.get(name);
  const t = TRIPLES.find((x) => x.triple === name);
  const platDir = p.join(pkgDir, 'platform', name);
  const dartSassDir = p.join(platDir, 'dart-sass');
  fs.mkdirSync(dartSassDir, {recursive: true});
  const platPkg = {
    name: `sass-embedded-go-${name}`,
    version: wrapperVersion,
    description: `go-sass embedded compiler binary (${name}).`,
    repository: {
      type: 'git',
      url: 'git+https://github.com/bancek/go-sass.git',
    },
    license: 'MIT',
    os: [t.os],
    cpu: [t.cpu],
    // Mirrors upstream: linux manifests declare their libc (glibc/musl
    // string); darwin/win carry no libc field.
    ...(t.libc ? {libc: t.libc} : {}),
  };
  fs.writeFileSync(p.join(platDir, 'package.json'), JSON.stringify(platPkg, null, 2) + '\n');
  if (name.startsWith('win32')) {
    const exe = p.join(dartSassDir, 'sass.exe');
    fs.copyFileSync(goBinaryPath, exe);
    // Stock resolution probes `dart-sass/sass.bat` on Windows.
    fs.writeFileSync(
      p.join(dartSassDir, 'sass.bat'),
      `@echo off\r\n"%~dp0sass.exe" %*\r\n`,
    );
  } else {
    const bin = p.join(dartSassDir, 'sass');
    fs.copyFileSync(goBinaryPath, bin);
    fs.chmodSync(bin, 0o755);
  }
}
fs.rmSync(p.join(pkgDir, 'platform-tmp-sass'), {force: true});

// 5. Link the host platform into node_modules (same require.resolve path
// as a registry install; script state, never committed).
const hostPlatDir = p.join(pkgDir, 'platform', triple());
const nmDir = p.join(pkgDir, 'node_modules');
fs.mkdirSync(nmDir, {recursive: true});
const link = p.join(nmDir, platformPackageName());
fs.rmSync(link, {recursive: true, force: true});
fs.symlinkSync(
  p.relative(nmDir, hostPlatDir),
  link,
  process.platform === 'win32' ? 'junction' : 'dir',
);

// 6. Report the resolved binary (also asserted by the test gate). Entry
// mirrors stock resolution (`dart-sass/sass`, + `.bat` on Windows).
const entry =
  process.platform === 'win32' ? 'dart-sass/sass.bat' : 'dart-sass/sass';
const resolved = require.resolve(`${platformPackageName()}/${entry}`);
console.log(`[build] platform package: ${platformPackageName()}`);
console.log(`[build] compiler binary:  ${resolved}`);
