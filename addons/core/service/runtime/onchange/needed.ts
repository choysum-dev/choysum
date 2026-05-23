// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../orm/metadata/model';
import type { NeededResult } from './types';
import type { OnchangeHandlerMeta } from '../../orm/metadata/model';
import type { ObjectRecord } from '../../../utils/types';

export function buildNeededFields(meta: ModelMetadata, draft: ObjectRecord, changed: string[]): NeededResult {
  void draft;
  const normalizedChanged = changed.filter(Boolean).map(String);
  const fieldMap = meta.fields || new Map<string, unknown>();

  const needed = new Set<string>(normalizedChanged);

  const allHandlers: OnchangeHandlerMeta[] = meta.onchangeHandlers || [];
  const activeHandlers: OnchangeHandlerMeta[] = [];
  for (const h of allHandlers) {
    if (h.triggers.some((t: string) => needed.has(t))) {
      activeHandlers.push(h);
      h.triggers.forEach((t: string) => needed.add(t));
    }
  }

  for (const h of activeHandlers) {
    (h.reads || []).forEach((r: string) => {
      const root = r.split('.').filter(Boolean)[0];
      if (root) needed.add(root);
    });
  }

  const g = meta.computeGraph;
  if (g) {
    const queue: string[] = [...needed];
    const seen = new Set<string>(queue);

    while (queue.length) {
      const src = queue.shift()!;
      const affected = g.fastReverseDeps.get(src);
      if (!affected) continue;

      for (const cf of affected) {
        const deps = g.computeScalarDeps?.get(cf);
        if (!deps) continue;
        for (const depField of deps) {
          if (!seen.has(depField)) {
            seen.add(depField);
            queue.push(depField);
            needed.add(depField);
          }
        }
      }
    }
  }

  const filtered = new Set<string>();
  for (const f of needed) {
    if (fieldMap.has(f)) filtered.add(f);
  }

  return {
    needed: filtered,
    activeHandlers,
  };
}
