// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * Choysum facade over @vue/language-core createVirtualCode for QuickJS.
 * Produces the service script (script_ts|tsx|js|jsx) + flattened mappings.
 */

import * as ts from "typescript";
import {
  createVueLanguagePlugin,
  createParsedCommandLineByJson,
  forEachEmbeddedCode,
} from "@vue/language-core";
import { createMinimalSys } from "./minimal_sys";
import {
  cumulativeDeltaAfter,
  rewriteVueHelperReferences,
} from "./rewrite_refs";

const SERVICE_SCRIPT_RE = /^script_(js|jsx|ts|tsx)$/;

export type CodegenOptions = {
  currentDirectory?: string;
  vueCompilerOptions?: object;
  compilerOptions?: ts.CompilerOptions;
};

export type SpanMapping = {
  sourceFile: string;
  sourceStart: number;
  sourceEnd: number;
  generatedStart: number;
  generatedEnd: number;
  verification?: unknown;
};

export type ServiceScriptResult = {
  embeddedId: string;
  scriptKind: string;
  content: string;
  mappings: SpanMapping[];
};

type CodeMappingLike = {
  sourceOffsets?: number[];
  generatedOffsets?: number[];
  lengths?: number[];
  data?: { verification?: unknown };
};

export function flattenCodeMappings(
  sourceFile: string,
  mappings: CodeMappingLike[],
): SpanMapping[] {
  const out: SpanMapping[] = [];
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
        sourceStart: m.sourceOffsets![i],
        sourceEnd: m.sourceOffsets![i] + m.lengths![i],
        generatedStart: m.generatedOffsets![i],
        generatedEnd: m.generatedOffsets![i] + m.lengths![i],
        verification: data.verification,
      });
    }
  }
  return out;
}

/**
 * Create the language-core service script for one .vue SFC.
 * Exported on the IIFE global `vuevirtual.createServiceScript`.
 */
export function createServiceScript(
  fileName: string,
  source: string,
  options: CodegenOptions = {},
): ServiceScriptResult {
  const snapshot = {
    getText: (s: number, e: number) => source.slice(s, e),
    getLength: () => source.length,
    getChangeRange: () => undefined,
  };

  const cwd = options.currentDirectory ?? "/";
  const minimalSys = createMinimalSys(cwd);

  const vueOptions =
    options.vueCompilerOptions ??
    createParsedCommandLineByJson(ts, minimalSys as never, cwd, {}).vueOptions;

  const compilerOptions: ts.CompilerOptions = options.compilerOptions ?? {
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
    (id: string) => id,
  );

  const root = plugin.createVirtualCode(fileName, "vue", snapshot, {
    getAssociatedScript: () => undefined,
  });
  if (!root) {
    throw new Error(`createVirtualCode returned undefined for ${fileName}`);
  }

  let service: { id: string; snapshot: typeof snapshot; mappings?: CodeMappingLike[] } | null =
    null;
  for (const code of forEachEmbeddedCode(root)) {
    if (SERVICE_SCRIPT_RE.test(code.id)) {
      service = code as typeof service;
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
    (m) => ({
      ...m,
      generatedStart: m.generatedStart + cumulativeDeltaAfter(edits, m.generatedStart),
      generatedEnd: m.generatedEnd + cumulativeDeltaAfter(edits, m.generatedEnd),
    }),
  );

  return {
    embeddedId: service.id,
    scriptKind: lang,
    content,
    mappings,
  };
}
