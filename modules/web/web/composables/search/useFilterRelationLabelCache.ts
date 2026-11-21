// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { onUnmounted, shallowRef, watch, type ShallowRef } from 'vue';
import type { ConditionGroup } from '@/web/web/query/types';
import type { ModelStore } from '@/web/web/stores/modelStore';
import { createModelStore as defaultCreateModelStore } from '@/web/web/stores/registry';
import { batchLoadRefRecordsByModel } from '@/web/web/query/utils/relation/displayNameBatch';
import {
  collectRelationIdsFromGroup,
  relationIdBatchSignature,
  relationLabelCacheKey,
} from '@/web/web/query/utils/relation/filterRelationLabels';

export type FilterRelationLabelCacheDeps = {
  createModelStore?: typeof defaultCreateModelStore;
};

/**
 * Short-lived DisplayName cache for SearchFilter preview.
 * Never writes hydrated labels back into the draft / query condition tree.
 */
export function useFilterRelationLabelCache(
  store: ModelStore<any>,
  getRoot: () => ConditionGroup,
  deps: FilterRelationLabelCacheDeps = {}
): {
  labels: ShallowRef<Map<string, string>>;
  labelOf: (relationModel: string, id: string) => string;
} {
  const createModelStore = deps.createModelStore ?? defaultCreateModelStore;
  const labels = shallowRef(new Map<string, string>());
  let generation = 0;
  // Stash the batch computed in the watch source so the callback need not recollect.
  let pendingBatch = new Map<string, Set<string>>();

  async function refresh(batch: Map<string, Set<string>>, gen: number) {
    if (batch.size === 0) {
      if (gen === generation && labels.value.size > 0) {
        labels.value = new Map();
      }
      return;
    }
    const lookups = await batchLoadRefRecordsByModel(batch, createModelStore, '[SearchFilter]');
    if (gen !== generation) return;

    // Replace (do not merge forever): drop ids no longer in the current draft batch.
    const next = new Map<string, string>();
    for (const [model, ids] of batch) {
      const map = lookups.get(model);
      for (const id of ids) {
        const key = relationLabelCacheKey(model, id);
        const display = String(map?.get(id)?.DisplayName ?? map?.get(id)?.Id ?? id);
        next.set(key, display);
      }
    }
    labels.value = next;
  }

  watch(
    () => {
      const meta = (store as any)?.fieldsMetadata || {};
      pendingBatch = collectRelationIdsFromGroup(getRoot(), meta);
      return relationIdBatchSignature(pendingBatch);
    },
    async signature => {
      const gen = ++generation;
      const batch = pendingBatch;
      try {
        if (!signature) {
          await refresh(new Map(), gen);
          return;
        }
        await refresh(batch, gen);
      } catch (e) {
        console.warn('[SearchFilter] Failed to hydrate relation labels', e);
      }
    },
    { immediate: true }
  );

  onUnmounted(() => {
    generation += 1;
    labels.value = new Map();
  });

  function labelOf(relationModel: string, id: string): string {
    const key = relationLabelCacheKey(relationModel, id);
    return labels.value.get(key) ?? id;
  }

  return { labels, labelOf };
}
