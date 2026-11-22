// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref, watch, type ComputedRef, type Ref } from 'vue';
import { toRpcOrderBy } from '@/web/web/query/orderBy';
import { useBreadcrumbStore } from '@/web/web/stores/breadcrumbStore';
import {
  useRecordSetStore,
  type SearchRecordSetIds,
} from '@/web/web/stores/recordSetStore';

type RouterLike = {
  currentRoute: { value: { path: string; name?: unknown; params?: Record<string, unknown>; query?: unknown } };
  push: (to: string | Record<string, unknown>) => unknown;
};

type SearchStoreLike = {
  Search?: (condition?: unknown, options?: Record<string, unknown>) => Promise<unknown[]>;
};

export type UseFormRecordPagerOpts = {
  model: ComputedRef<string | undefined> | Ref<string | undefined> | (() => string | undefined);
  recordId: ComputedRef<string | undefined> | Ref<string | undefined> | (() => string | undefined);
  isCreate: ComputedRef<boolean> | Ref<boolean> | (() => boolean);
  router: RouterLike;
  store: SearchStoreLike;
  confirmDiscardIfDirty: () => Promise<boolean>;
};

function read<T>(src: ComputedRef<T> | Ref<T> | (() => T)): T {
  return typeof src === 'function' ? (src as () => T)() : (src as Ref<T>).value;
}

function siblingLocation(router: RouterLike, nextId: string): string | Record<string, unknown> {
  const route = router.currentRoute.value;
  const params = route.params || {};
  if (route.name && (Object.prototype.hasOwnProperty.call(params, 'id') || Object.prototype.hasOwnProperty.call(params, 'recordId'))) {
    const nextParams = { ...params };
    if (Object.prototype.hasOwnProperty.call(nextParams, 'id')) nextParams.id = nextId;
    if (Object.prototype.hasOwnProperty.call(nextParams, 'recordId')) nextParams.recordId = nextId;
    return { name: route.name as string, params: nextParams, query: route.query as any };
  }
  const parts = String(route.path || '')
    .replace(/\/+$/, '')
    .split('/');
  if (parts.length) parts[parts.length - 1] = nextId;
  return parts.join('/') || `/${nextId}`;
}

function buildSearchIds(store: SearchStoreLike): SearchRecordSetIds {
  return async ({ condition, orderBy, offset, limit }) => {
    if (typeof store.Search !== 'function') return [];
    const rows = await store.Search(condition as any, {
      fields: ['Id'],
      limit,
      offset,
      orderBy: toRpcOrderBy(orderBy),
    });
    return (Array.isArray(rows) ? rows : [])
      .map((row: any) => String(row?.Id ?? row?.id ?? '').trim())
      .filter(Boolean);
  };
}

/**
 * Form toolbar record pager: visible when a matching recordSet session exists.
 */
export function useFormRecordPager(opts: UseFormRecordPagerOpts) {
  const recordSet = useRecordSetStore();
  recordSet.installMenuClearWatch();
  const busy = ref(false);

  const entry = computed(() => {
    if (read(opts.isCreate)) return null;
    return recordSet.getActiveForModel(read(opts.model));
  });

  const visible = computed(() => {
    const e = entry.value;
    return !!(e && e.total > 0 && e.ids.length >= 0);
  });

  const current = computed(() => (entry.value ? entry.value.index + 1 : 0));
  const total = computed(() => entry.value?.total ?? 0);
  const canPrev = computed(() => recordSet.canMove(-1) && !!entry.value);
  const canNext = computed(() => recordSet.canMove(1) && !!entry.value);

  // Create mode is a new single-record session — drop the collection pager.
  watch(
    () => read(opts.isCreate),
    isCreate => {
      if (isCreate) recordSet.clear();
    }
  );

  async function navigateToMoved(
    moved: { id: string } | null
  ): Promise<void> {
    if (!moved) return;
    // Sibling id under the same menu leaf: replace the current page crumb, do not deepen.
    useBreadcrumbStore().setPendingPageMode('replace');
    await opts.router.push(siblingLocation(opts.router, moved.id) as any);
  }

  async function navigate(delta: -1 | 1): Promise<void> {
    if (busy.value) return;
    if (!(await opts.confirmDiscardIfDirty())) return;
    busy.value = true;
    try {
      const moved = await recordSet.move(delta, buildSearchIds(opts.store));
      await navigateToMoved(moved);
    } finally {
      busy.value = false;
    }
  }

  /** Jump to a 1-based position in the active collection. */
  async function goTo(position: number): Promise<void> {
    if (busy.value) return;
    const e = entry.value;
    if (!e || e.total <= 0) return;
    const target = Math.trunc(Number(position) || 0);
    if (target <= 0) return;
    const index = Math.max(0, Math.min(target - 1, e.total - 1));
    if (index === e.index) return;
    if (!(await opts.confirmDiscardIfDirty())) return;
    busy.value = true;
    try {
      const moved = await recordSet.moveTo(index, buildSearchIds(opts.store));
      await navigateToMoved(moved);
    } finally {
      busy.value = false;
    }
  }

  return {
    visible,
    current,
    total,
    canPrev,
    canNext,
    busy,
    goPrev: () => navigate(-1),
    goNext: () => navigate(1),
    goTo,
  };
}
