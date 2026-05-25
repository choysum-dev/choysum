// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

'use strict';

const fs = require('fs');
const path = require('path');
const { createRequire } = require('module');

const requireFromCwd = createRequire(path.resolve(process.cwd(), '__choysum_coverage_require__.cjs'));

function argValue(name) {
  const idx = process.argv.indexOf(name);
  if (idx === -1) return null;
  const v = process.argv[idx + 1];
  if (!v || v.startsWith('--')) return null;
  return v;
}

function stripSourceMappingURL(code) {
  // Remove trailing sourceMappingURL comments (inline or external).
  return code.replace(/\n\/\/[#@]\s*sourceMappingURL=.*\s*$/m, '');
}

function detectSourceMap(code, inFilePath) {
  const re = /\/\/[#@]\s*sourceMappingURL=([^\s]+)\s*$/m;
  const m = code.match(re);
  if (!m) return null;
  const url = m[1].trim();

  // inline data url
  const prefix = 'data:application/json;base64,';
  if (url.startsWith(prefix)) {
    const b64 = url.slice(prefix.length);
    try {
      const json = Buffer.from(b64, 'base64').toString('utf8');
      return JSON.parse(json);
    } catch {
      return null;
    }
  }

  // external map file
  try {
    const mapPath = path.resolve(path.dirname(inFilePath), url);
    if (!fs.existsSync(mapPath)) return null;
    const raw = fs.readFileSync(mapPath, 'utf8');
    return JSON.parse(raw);
  } catch {
    return null;
  }
}

async function main() {
  const inPath = argValue('--in');
  const outPath = argValue('--out') || inPath;
  const outMapPath = argValue('--out-map');
  if (!inPath) {
    console.error('usage: node instrument.cjs --in <file> [--out <file>] [--out-map <file>]');
    process.exit(2);
  }

  const instrumentLib = requireFromCwd('istanbul-lib-instrument');
  const instrumenter = instrumentLib.createInstrumenter({
    produceSourceMap: true,
    esModules: false,
    compact: false,
    coverageGlobalScope: 'globalThis',
    coverageGlobalScopeFunc: false,
  });

  const original = fs.readFileSync(inPath, 'utf8');
  const inputMap = detectSourceMap(original, inPath);
  const code = stripSourceMappingURL(original);

  const instrumented = instrumenter.instrumentSync(code, inPath, inputMap || undefined);
  const map = instrumenter.lastSourceMap();

  // In QuickJS, the host must free the result value of script evaluation.
  // Some runtimes may leak the last evaluated value; make it definitely `undefined`.
  let finalCode = instrumented + '\n;void 0;\n';
  if (outMapPath && map) {
    fs.writeFileSync(outMapPath, JSON.stringify(map), 'utf8');
    const rel = path.basename(outMapPath);
    finalCode += `\n//# sourceMappingURL=${rel}\n`;
  }

  fs.writeFileSync(outPath, finalCode, 'utf8');
}

main().catch(err => {
  console.error(err && err.stack ? err.stack : String(err));
  process.exit(1);
});
