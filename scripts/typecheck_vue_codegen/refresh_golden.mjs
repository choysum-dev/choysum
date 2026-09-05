#!/usr/bin/env node
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * Refresh golden service scripts + mappings for testdata/vue/fixtures.
 *
 * Usage (from repo root):
 *   node scripts/typecheck_vue_codegen/refresh_golden.mjs
 *
 * Requires Node + @vue/language-core@3.3.7 (repo root node_modules or local install).
 */

import fs from "node:fs";
import path from "node:path";
import crypto from "node:crypto";
import { fileURLToPath } from "node:url";
import { createServiceScript } from "./create_service_script.mjs";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "../..");
const fixturesDir = path.join(
  repoRoot,
  "internal/typecheck/testdata/vue/fixtures",
);
const goldenDir = path.join(repoRoot, "internal/typecheck/testdata/vue/golden");

function main() {
  fs.mkdirSync(goldenDir, { recursive: true });
  const files = fs
    .readdirSync(fixturesDir)
    .filter((n) => n.endsWith(".vue"))
    .sort();
  if (files.length === 0) {
    console.error("no .vue fixtures in", fixturesDir);
    process.exit(1);
  }
  for (const name of files) {
    const vuePath = path.join(fixturesDir, name);
    const source = fs.readFileSync(vuePath, "utf8");
    // Stable absolute-looking path so mappings.sourceFile is predictable.
    const fileName = `/fixtures/${name}`;
    const result = createServiceScript(fileName, source, {
      currentDirectory: "/fixtures",
    });
    const base = path.join(goldenDir, name);
    // .service.txt (not .ts): content is TypeScript, but must not be scanned as
    // product JS/TS by Code Quality / CodeQL extractors.
    fs.writeFileSync(`${base}.service.txt`, result.content);
    fs.writeFileSync(
      `${base}.mappings.json`,
      JSON.stringify(
        {
          embeddedId: result.embeddedId,
          scriptKind: result.scriptKind,
          sourceSHA256: crypto.createHash("sha256").update(source).digest("hex"),
          mappings: result.mappings,
        },
        null,
        2,
      ) + "\n",
    );
    console.log("wrote", path.relative(repoRoot, `${base}.service.txt`));
  }
}

main();
