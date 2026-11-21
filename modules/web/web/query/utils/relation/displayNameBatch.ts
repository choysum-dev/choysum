// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Batch-load target records for *Ref fields by model name and id set.
 * Used by list/browse hydrate and readGroup label hydrate (frontend-only for Ref).
 */

/**
 * For each target model, Search Id in ids with fields Id+DisplayName.
 * Failed targets are skipped (warned); callers keep raw ids as fallback.
 * `createModelStore` is intentionally loose so overload-heavy registry factories typecheck.
 */
/** Max ids per Search `in` clause to avoid oversized query payloads. */
export const REF_DISPLAY_NAME_IN_CHUNK = 200;

export async function batchLoadRefRecordsByModel(
  idsByModel: Map<string, Set<string>>,
  createModelStore: (modelName: string, ...args: any[]) => any,
  logPrefix = '[Executor]'
): Promise<Map<string, Map<string, any>>> {
  const lookups = new Map<string, Map<string, any>>();

  await Promise.all(
    Array.from(idsByModel.entries()).map(async ([model, ids]) => {
      if (ids.size === 0) return;
      try {
        const targetStore = createModelStore(model) as {
          Search?: (condition?: any, options?: any) => Promise<any>;
        };
        if (typeof targetStore.Search !== 'function') return;
        const map = new Map<string, any>();
        const idList = Array.from(ids);
        for (let i = 0; i < idList.length; i += REF_DISPLAY_NAME_IN_CHUNK) {
          const chunk = idList.slice(i, i + REF_DISPLAY_NAME_IN_CHUNK);
          try {
            const results = await targetStore.Search(['Id', 'in', chunk], {
              fields: ['Id', 'DisplayName'],
            });
            for (const r of (results as Array<{ Id?: unknown }>) || []) {
              const key = String(r.Id ?? '');
              if (!key || key === 'undefined') continue;
              map.set(key, r);
            }
          } catch (e) {
            // Keep earlier chunks; only this chunk falls back to raw ids at call sites.
            console.warn(`${logPrefix} Failed to hydrate chunk for target '${model}'`, e);
          }
        }
        lookups.set(model, map);
      } catch (e) {
        console.warn(`${logPrefix} Failed to hydrate ManyToOneRef for target '${model}'`, e);
      }
    })
  );

  return lookups;
}

/** Strip temporal groupby suffixes (e.g. CompanyId__month → CompanyId). */
export function baseFieldFromGroupAlias(alias: string): string {
  return String(alias || '').split('__')[0];
}
