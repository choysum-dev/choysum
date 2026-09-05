// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * Choysum facade over @vue/language-core createVirtualCode.
 * Produces the service script (script_ts|tsx|js|jsx) + flattened mappings.
 */

import * as ts from "typescript";
import {
  createVueLanguagePlugin,
  createParsedCommandLineByJson,
  forEachEmbeddedCode,
} from "@vue/language-core";
import path from "node:path";

const SERVICE_SCRIPT_RE = /^script_(js|jsx|ts|tsx)$/;

/** Stable virtual helper paths used by Go overlay VFS. */
export const VUE_HELPER_TEMPLATE =
  "/choysum-vue-virtual/types/template-helpers.d.ts";
export const VUE_HELPER_PROPS =
  "/choysum-vue-virtual/types/props-fallback.d.ts";

const minimalSys = {
  ...ts.sys,
  getCurrentDirectory: () => "/",
  readDirectory: () => [],
  fileExists: () => false,
  readFile: () => undefined,
  realpath: (p) => p,
};

/**
 * @param {string} fileName absolute or workspace path ending in .vue
 * @param {string} source SFC source text
 * @param {{ currentDirectory?: string, vueCompilerOptions?: object, compilerOptions?: object }} [options]
 */
export function createServiceScript(fileName, source, options = {}) {
  const snapshot = {
    getText: (s, e) => source.slice(s, e),
    getLength: () => source.length,
    getChangeRange: () => undefined,
  };

  const vueOptions =
    options.vueCompilerOptions ??
    createParsedCommandLineByJson(
      ts,
      minimalSys,
      options.currentDirectory ?? "/",
      {},
    ).vueOptions;

  const compilerOptions = options.compilerOptions ?? {
    strict: true,
    target: ts.ScriptTarget.ES2020,
    module: ts.ModuleKind.ESNext,
    moduleResolution: ts.ModuleResolutionKind.Bundler,
    jsx: ts.JsxEmit.Preserve,
  };

  const plugin = createVueLanguagePlugin(
    ts,
    compilerOptions,
    vueOptions || {},
    (id) => id,
  );

  const root = plugin.createVirtualCode(fileName, "vue", snapshot, {
    getAssociatedScript: () => undefined,
  });
  if (!root) {
    throw new Error(`createVirtualCode returned undefined for ${fileName}`);
  }

  let service = null;
  for (const code of forEachEmbeddedCode(root)) {
    if (SERVICE_SCRIPT_RE.test(code.id)) {
      service = code;
      break;
    }
  }
  if (!service) {
    throw new Error(`no service script embedded for ${fileName}`);
  }

  const rawContent = service.snapshot.getText(0, service.snapshot.getLength());
  const { content, edits } = rewriteVueHelperReferences(rawContent);
  const lang = service.id.slice("script_".length);
  const mappings = flattenCodeMappings(fileName, service.mappings ?? []).map(
    (m) => {
      const delta = cumulativeDeltaAfter(edits, m.generatedStart);
      return {
        ...m,
        generatedStart: m.generatedStart + delta,
        generatedEnd: m.generatedEnd + cumulativeDeltaAfter(edits, m.generatedEnd),
      };
    },
  );

  return {
    embeddedId: service.id,
    scriptKind: lang,
    content,
    mappings,
  };
}

/** Sum of length deltas for edits that start at or before pos. */
export function cumulativeDeltaAfter(edits, pos) {
  let delta = 0;
  for (const e of edits) {
    if (e.index <= pos) {
      delta += e.delta;
    }
  }
  return delta;
}

export function flattenCodeMappings(sourceFile, mappings) {
  const out = [];
  for (const m of mappings) {
    const n = Math.min(
      m.sourceOffsets?.length ?? 0,
      m.generatedOffsets?.length ?? 0,
      m.lengths?.length ?? 0,
    );
    for (let i = 0; i < n; i++) {
      const data = m.data ?? {};
      if (data.verification === false) continue;
      out.push({
        sourceFile,
        sourceStart: m.sourceOffsets[i],
        sourceEnd: m.sourceOffsets[i] + m.lengths[i],
        generatedStart: m.generatedOffsets[i],
        generatedEnd: m.generatedOffsets[i] + m.lengths[i],
        verification: data.verification,
      });
    }
  }
  return out;
}

export function rewriteVueHelperReferences(content) {
  const replacements = [
    {
      re: /\/\/\/\s*<reference\s+types="[^"]*template-helpers\.d\.ts"\s*\/>/g,
      to: `/// <reference path="${VUE_HELPER_TEMPLATE}" />`,
    },
    {
      re: /\/\/\/\s*<reference\s+types="[^"]*props-fallback\.d\.ts"\s*\/>/g,
      to: `/// <reference path="${VUE_HELPER_PROPS}" />`,
    },
  ];
  /** @type {{ index: number, delta: number }[]} */
  const edits = [];
  let out = content;
  // Apply left-to-right so later match indices stay valid in the original string;
  // track deltas in original coordinates for mapping adjustment.
  const matches = [];
  for (const { re, to } of replacements) {
    re.lastIndex = 0;
    let m;
    while ((m = re.exec(content)) !== null) {
      matches.push({ index: m.index, from: m[0], to });
    }
  }
  matches.sort((a, b) => a.index - b.index);
  let shift = 0;
  out = content;
  for (const m of matches) {
    const at = m.index + shift;
    out = out.slice(0, at) + m.to + out.slice(at + m.from.length);
    const delta = m.to.length - m.from.length;
    edits.push({ index: m.index, delta });
    shift += delta;
  }
  return { content: out, edits };
}

/** CLI helper: resolve module from local or repo root node_modules. */
export function resolveImportMetaDir() {
  return path.dirname(new URL(import.meta.url).pathname);
}
