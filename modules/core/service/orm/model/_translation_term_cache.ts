// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type ChoysumI18nBridge = {
  t?: (module: string, lang: string, scope: string, src: string, kind?: string) => string;
  invalidateModule?: (application: string, module: string) => boolean;
  upsertPackagedTerms?: (
    application: string,
    module: string,
    lang: string,
    poText: string | Uint8Array
  ) => Promise<{
    upserted: number;
    skippedOverride: number;
    rejectedNoCtxt: number;
    skippedObsolete: number;
    purgedRetired: number;
    lang: string;
  }>;
};

export function getChoysumI18nBridge(): ChoysumI18nBridge | undefined {
  const root = globalThis as { $choysum?: { i18n?: ChoysumI18nBridge } };
  return root.$choysum?.i18n;
}

/** Invalidate Go TermStore for one module (no-op when bridge is absent). */
export function invalidateTerminologyModule(application: string, module: string): void {
  const app = String(application || '').trim();
  const mod = String(module || '').trim();
  if (!app || app === 'core' || !mod) return;
  const bridge = getChoysumI18nBridge();
  if (!bridge || typeof bridge.invalidateModule !== 'function') return;
  try {
    bridge.invalidateModule(app, mod);
  } catch {
    /* best-effort: write already succeeded */
  }
}

/** Invalidate each distinct module name for the host application. */
export function invalidateTerminologyModules(application: string, modules: Iterable<string>): void {
  const seen = new Set<string>();
  for (const raw of modules) {
    const mod = String(raw || '').trim();
    if (!mod || seen.has(mod)) continue;
    seen.add(mod);
    invalidateTerminologyModule(application, mod);
  }
}

export function modulesFromRows(rows: unknown): string[] {
  const list = Array.isArray(rows) ? rows : rows != null ? [rows] : [];
  const out: string[] = [];
  for (const row of list) {
    const mod = String((row as any)?.Module ?? '').trim();
    if (mod) out.push(mod);
  }
  return out;
}

export function modulesFromPayloads(values: unknown): string[] {
  const list = Array.isArray(values) ? values : values != null ? [values] : [];
  const out: string[] = [];
  for (const row of list) {
    const mod = String((row as any)?.Module ?? '').trim();
    if (mod) out.push(mod);
  }
  return out;
}
