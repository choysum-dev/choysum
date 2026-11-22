// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref, watch, type ComputedRef, type Ref } from 'vue';
import { toRpcOrderBy } from '@/web/web/query/orderBy';
import { useBreadcrumbStore } from '@/web/web/stores/breadcrumbStore';
import {
  useRecordSetStore,
  type RecordSetEntry,
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
  if (parts.length) parts[parts.length - 1] = encodeURIComponent(nextId);
  const targetPath = parts.join('/') || `/${encodeURIComponent(nextId)}`;
  const query = route.query;
  if (query && typeof query === 'object' && Object.keys(query as object).length > 0) {
    return { path: targetPath, query: query as any };
  }
  return targetPath;
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
    if (!e || e.total <= 0 || e.ids.length <= 0) return false;
    // Hide when the open id is outside the known window (stale / deep-link mismatch).
    const id = String(read(opts.recordId) || '').trim();
    return !!id && e.ids.includes(id);
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
    moved: { id: string } | null,
    previous: RecordSetEntry | null
  ): Promise<void> {
    if (!moved) return;
    const breadcrumb = useBreadcrumbStore();
    // Sibling id under the same menu leaf: replace the current page crumb, do not deepen.
    breadcrumb.setPendingPageMode('replace');
    try {
      await opts.router.push(siblingLocation(opts.router, moved.id) as any);
    } catch {
      // Failed / aborted navigations must not leave a sticky replace intent
      // or an advanced pager index while the Form route stayed put.
      breadcrumb.setPendingPageMode(null);
      const cur = recordSet.active;
      if (previous && cur && cur.key === previous.key) {
        recordSet.replaceActive(previous);
      }
    }
  }

  // Browser Back can change the route id without updating the stored index.
  watch(
    () => String(read(opts.recordId) || '').trim(),
    id => {
      const e = entry.value;
      if (!e || !id) return;
      const local = e.ids.indexOf(id);
      if (local < 0) return;
      const nextIndex = e.windowStart + local;
      if (nextIndex === e.index) return;
      recordSet.replaceActive({ ...e, index: nextIndex });
    }
  );

  async function navigate(delta: -1 | 1): Promise<void> {
    if (busy.value) return;
    if (!(await opts.confirmDiscardIfDirty())) return;
    busy.value = true;
    try {
      const previous = entry.value ? { ...entry.value } : null;
      const moved = await recordSet.move(delta, buildSearchIds(opts.store));
      await navigateToMoved(moved, previous);
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
      const previous = entry.value ? { ...entry.value } : null;
      const moved = await recordSet.moveTo(index, buildSearchIds(opts.store));
      await navigateToMoved(moved, previous);
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
