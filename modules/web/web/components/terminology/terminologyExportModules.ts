// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Status value for installed modules (matches pkg/meta.Installed / PO gateway). */
export const META_MODULE_STATUS_INSTALLED = 'installed';

/** Collect distinct sorted module names from meta.MetaModule Search rows. */
export function collectModuleNamesFromMetaModuleRows(rows: unknown): string[] {
  if (!Array.isArray(rows)) return [];
  const seen = new Set<string>();
  const out: string[] = [];
  for (const row of rows) {
    if (!row || typeof row !== 'object') continue;
    const name = String((row as { Name?: unknown }).Name ?? '').trim();
    if (!name || seen.has(name)) continue;
    seen.add(name);
    out.push(name);
  }
  out.sort((a, b) => a.localeCompare(b));
  return out;
}

/**
 * Search condition aligned with internal/i18n/gateway installedModulesByApp:
 * ApplicationStr + Status=installed.
 */
export function installedModulesSearchCondition(application: string) {
  const app = String(application || '').trim();
  return {
    And: [
      ['ApplicationStr', '=', app],
      ['Status', '=', META_MODULE_STATUS_INSTALLED],
    ],
  } as const;
}
