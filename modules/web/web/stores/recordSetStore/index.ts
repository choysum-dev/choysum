// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref, watch } from 'vue';
import { defineStore } from 'pinia';
import { useMenuStore } from '@/web/web/stores/menuStore';
import {
  canMoveRecordSet,
  resolveRecordSetAtIndex,
  resolveRecordSetNeighbor,
} from './recordSetLogic';
import type { RecordSetEntry, SearchRecordSetIds } from './types';

export type { RecordSetEntry, SearchRecordSetIds } from './types';
export {
  RECORD_SET_NEIGHBOR_WINDOW,
  canMoveRecordSet,
  clampRecordSetIndex,
  idAtGlobalIndex,
  resolveRecordSetAtIndex,
  resolveRecordSetNeighbor,
} from './recordSetLogic';

/**
 * Session store for Form record pager: seeded by List/Kanban open-detail.
 */
export const useRecordSetStore = defineStore('recordSet', () => {
  const active = ref<RecordSetEntry | null>(null);
  const menuWatchInstalled = ref(false);
  // Monotonic id for the active seed; used so a late failed open cannot clear a newer session.
  const seedGeneration = ref(0);

  const hasActive = computed(() => active.value != null && active.value.total > 0);

  function assignActive(entry: RecordSetEntry): void {
    const ids = (entry.ids || []).map(id => String(id ?? '').trim()).filter(Boolean);
    const total = Math.max(0, Math.trunc(Number(entry.total) || 0));
    const index = Math.max(0, Math.min(Math.trunc(Number(entry.index) || 0), Math.max(0, total - 1)));
    const windowStart = Math.max(0, Math.trunc(Number(entry.windowStart) || 0));
    active.value = {
      key: String(entry.key || ''),
      model: String(entry.model || ''),
      condition: entry.condition,
      orderBy: entry.orderBy,
      total,
      index,
      ids,
      windowStart,
    };
  }

  function seed(entry: RecordSetEntry): void {
    seedGeneration.value += 1;
    assignActive(entry);
  }

  function clear(): void {
    active.value = null;
  }

  function getActiveForModel(model: string | null | undefined): RecordSetEntry | null {
    const m = String(model ?? '').trim();
    if (!m || !active.value) return null;
    return active.value.model === m ? active.value : null;
  }

  /** Update the active entry in place without bumping seedGeneration. */
  function replaceActive(entry: RecordSetEntry): void {
    assignActive(entry);
  }

  async function move(
    delta: -1 | 1,
    searchIds?: SearchRecordSetIds
  ): Promise<{ id: string; entry: RecordSetEntry } | null> {
    const cur = active.value;
    if (!cur) return null;
    const generation = seedGeneration.value;
    const next = await resolveRecordSetNeighbor(cur, delta, searchIds);
    if (!next) return null;
    // Drop the result if the session was cleared/re-seeded during the await.
    if (!active.value || seedGeneration.value !== generation) return null;
    active.value = next.entry;
    return next;
  }

  /** Jump to a 0-based absolute index (clamped); no-op when already there. */
  async function moveTo(
    index: number,
    searchIds?: SearchRecordSetIds
  ): Promise<{ id: string; entry: RecordSetEntry } | null> {
    const cur = active.value;
    if (!cur) return null;
    const generation = seedGeneration.value;
    const next = await resolveRecordSetAtIndex(cur, index, searchIds);
    if (!next) return null;
    if (!active.value || seedGeneration.value !== generation) return null;
    active.value = next.entry;
    return next;
  }

  function installMenuClearWatch(): void {
    if (menuWatchInstalled.value) return;
    try {
      const menuStore = useMenuStore();
      watch(
        () => menuStore.activeMenuId,
        (next, prev) => {
          if (prev && next !== prev) clear();
        }
      );
      menuWatchInstalled.value = true;
    } catch {
      // Pinia/menu may be unavailable in unit tests without app shell.
    }
  }

  return {
    active,
    hasActive,
    seedGeneration,
    seed,
    clear,
    getActiveForModel,
    replaceActive,
    move,
    moveTo,
    canMove: (delta: -1 | 1) => canMoveRecordSet(active.value, delta),
    installMenuClearWatch,
  };
});
