// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, onMounted, ref, toValue, watch, type MaybeRefOrGetter, type Ref } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { createKanbanController } from '@/web/web/controllers/kanbanController';
import { awaitFieldSelection } from '@/web/web/query/utils/registry/fieldReady';
import type { ChoySearchQuery } from '@/web/web/components/view/searchViewHelpers';
import type { Lane } from '@/web/web/query/types';
import {
  type ChoyKanbanCard,
  type ChoyKanbanLane,
  type ChoyKanbanLoadMore,
  type ChoyKanbanMove,
} from '@/web/web/components/view/kanbanViewHelpers';
import {
  createLaneSyncGate,
  defaultKanbanMapRowToCard,
  defaultKanbanMoveRecordId,
  finishInitialKanbanLoad,
  shouldRecoverStaleKanbanSearch,
  shouldRestoreKanbanMove,
} from '@/web/web/components/view/kanbanStoreHelpers';

export type ChoyKanbanStoreEngineOptions = {
  store: WebModelStore<any>;
  keywordFields?: MaybeRefOrGetter<string[] | undefined>;
  titleField?: MaybeRefOrGetter<string | undefined>;
  flatLaneLabel?: MaybeRefOrGetter<string | undefined>;
  laneLabel?: (lane: Pick<ChoyKanbanLane, 'key' | 'label'> | Lane) => string;
  mapRowToCard?: (row: unknown, index: number, laneKey: string) => ChoyKanbanCard;
  resolveMoveRecordId?: (
    cards: ReadonlyArray<ChoyKanbanCard>,
    moveCardId: string,
  ) => string;
  /** Runs before field selection / first apply (e.g. index sync). */
  beforeBootstrap?: () => Promise<void>;
  /** When false, host must call `bootstrap()` (default true). */
  autoBootstrap?: boolean;
  onLoadError?: (error: unknown) => void;
  onSearchError?: (error: unknown) => void;
  onLoadMoreError?: (error: unknown) => void;
  onMoveError?: (error: unknown) => void;
};

/**
 * Store-backed kanban engine: controller apply/sync, search, load-more, and move persist.
 * ChoyKanbanView chrome binds to `lanes` and the returned handlers.
 */
export function useChoyKanbanStoreEngine(opts: ChoyKanbanStoreEngineOptions) {
  const controller = createKanbanController(opts.store);
  const lanes: Ref<ChoyKanbanLane[]> = ref([]);
  const laneSyncGate = createLaneSyncGate();
  const movePending = ref(false);
  const loadMorePending = ref(false);
  const searchPending = ref(false);
  let searchSeq = 0;
  let searchInFlight = 0;
  let lastSearchQuery: ChoySearchQuery | null = null;
  let bootstrapped = false;

  const mapRow =
    opts.mapRowToCard ??
    ((row: unknown, index: number, laneKey: string) =>
      defaultKanbanMapRowToCard(row, index, laneKey, toValue(opts.titleField) || 'Title'));

  const resolveMoveId = opts.resolveMoveRecordId ?? defaultKanbanMoveRecordId;

  function labelFor(lane: Pick<ChoyKanbanLane, 'key' | 'label'> | Lane): string {
    if (opts.laneLabel) return opts.laneLabel(lane);
    return String((lane as ChoyKanbanLane).label ?? (lane as Lane).label ?? (lane as ChoyKanbanLane).key ?? '');
  }

  async function syncLanesFromController(): Promise<void> {
    if ((await laneSyncGate.enter()) === 'waited') return;
    try {
      do {
        laneSyncGate.beginPass();
        const laneList = controller.lanes.value;
        if (!laneList.length) {
          const rows =
            controller.vm.result?.kind === 'search'
              ? ((controller.vm.result.rows as any[]) || [])
              : [];
          const flatKey = 'all';
          lanes.value = [
            {
              key: flatKey,
              label: toValue(opts.flatLaneLabel) || 'All',
              cards: rows.map((row, index) => mapRow(row, index, flatKey)),
            },
          ];
          continue;
        }
        await Promise.all(
          laneList.map((l) =>
            controller.preloadLane(l.key).catch((error) => {
              console.error(`Kanban lane preload failed: ${l.key}`, error);
              return undefined;
            }),
          ),
        );
        lanes.value = laneList.map((lane) => ({
          key: lane.key,
          label: labelFor(lane),
          remain: controller.getLaneRemain(lane),
          cards: (controller.laneRecords.value[lane.key] || []).map((row, index) =>
            mapRow(row, index, lane.key),
          ),
        }));
      } while (laneSyncGate.shouldResync());
    } finally {
      laneSyncGate.leave();
    }
  }

  async function applyCurrentQuery(): Promise<void> {
    const keywordFields = toValue(opts.keywordFields);
    if (lastSearchQuery) {
      await controller.apply({
        keyword: lastSearchQuery.keyword,
        appliedFilters: (lastSearchQuery.appliedFilters || []) as any,
        appliedGroups: lastSearchQuery.appliedGroups as any,
        ...(keywordFields ? { keywordFields } : {}),
      });
      return;
    }
    const qs = ((opts.store.state as any)?.queryState ?? {}) as Record<string, unknown>;
    await controller.apply({
      keyword: qs.keyword as string | undefined,
      appliedFilters: (qs.appliedFilters || []) as any,
      appliedGroups: qs.appliedGroups as any,
      ...(keywordFields ? { keywordFields } : {}),
    });
  }

  async function applySearch(query: ChoySearchQuery): Promise<void> {
    lastSearchQuery = query;
    const seq = ++searchSeq;
    searchInFlight++;
    searchPending.value = true;
    const keywordFields = toValue(opts.keywordFields);
    try {
      await controller.apply({
        keyword: query.keyword,
        appliedFilters: (query.appliedFilters || []) as any,
        appliedGroups: query.appliedGroups as any,
        ...(keywordFields ? { keywordFields } : {}),
      });
    } catch (e) {
      if (seq === searchSeq) opts.onSearchError?.(e);
    } finally {
      searchInFlight--;
    }

    if (
      shouldRecoverStaleKanbanSearch({
        completedSeq: seq,
        latestSeq: searchSeq,
        inFlight: searchInFlight,
      }) &&
      lastSearchQuery
    ) {
      const recoverSeq = searchSeq;
      searchInFlight++;
      try {
        await controller.apply({
          keyword: lastSearchQuery.keyword,
          appliedFilters: (lastSearchQuery.appliedFilters || []) as any,
          appliedGroups: lastSearchQuery.appliedGroups as any,
          ...(keywordFields ? { keywordFields } : {}),
        });
        if (recoverSeq === searchSeq) await syncLanesFromController();
      } catch (e) {
        if (recoverSeq === searchSeq) opts.onSearchError?.(e);
      } finally {
        searchInFlight--;
        if (searchInFlight === 0) searchPending.value = false;
      }
      return;
    }

    if (seq !== searchSeq) {
      if (searchInFlight === 0) searchPending.value = false;
      return;
    }
    try {
      await syncLanesFromController();
    } catch (e) {
      if (seq === searchSeq) opts.onSearchError?.(e);
    } finally {
      if (seq === searchSeq) searchPending.value = false;
    }
  }

  async function onLaneLoadMore(payload: ChoyKanbanLoadMore): Promise<void> {
    if (
      loadMorePending.value ||
      movePending.value ||
      searchPending.value ||
      !payload?.laneKey
    ) {
      return;
    }
    loadMorePending.value = true;
    try {
      await controller.loadMoreLane(payload.laneKey);
      await syncLanesFromController();
    } catch (e) {
      opts.onLoadMoreError?.(e);
    } finally {
      loadMorePending.value = false;
    }
  }

  async function persistMove(move: ChoyKanbanMove): Promise<void> {
    const recordId = resolveMoveId(
      lanes.value.flatMap((lane) => lane.cards),
      move.cardId,
    );
    if (
      shouldRestoreKanbanMove({
        movePending: movePending.value,
        searchPending: searchPending.value,
        recordId,
        fromLaneKey: move.fromLaneKey,
        controllerLaneKeys: controller.lanes.value.map((lane) => lane.key),
      })
    ) {
      await syncLanesFromController();
      return;
    }
    movePending.value = true;
    try {
      await controller.moveCard(recordId, move.fromLaneKey, move.toLaneKey, move.toIndex);
      await syncLanesFromController();
    } catch (e) {
      opts.onMoveError?.(e);
      try {
        await applyCurrentQuery();
      } catch (reloadError) {
        console.error('Kanban reload after move failure failed:', reloadError);
      }
      await syncLanesFromController();
    } finally {
      movePending.value = false;
    }
  }

  async function bootstrap(): Promise<void> {
    if (bootstrapped) return;
    bootstrapped = true;
    try {
      if (opts.beforeBootstrap) await opts.beforeBootstrap();
      await awaitFieldSelection(opts.store, { requireNonEmpty: true });
      const keywordFields = toValue(opts.keywordFields);
      if (keywordFields?.length) {
        await controller.setKeywordFields(keywordFields);
      }
      await finishInitialKanbanLoad({
        getLastSearchQuery: () => lastSearchQuery,
        applyEmpty: () =>
          controller.apply(keywordFields?.length ? { keywordFields } : {}),
        onSearch: applySearch,
        syncLanes: syncLanesFromController,
      });
    } catch (e) {
      opts.onLoadError?.(e);
      throw e;
    }
  }

  watch(
    () => controller.lanes.value,
    () => {
      void syncLanesFromController();
    },
    { deep: true },
  );

  if (opts.autoBootstrap !== false) {
    onMounted(() => {
      void bootstrap().catch(() => undefined);
    });
  }

  const boardBusy = computed(
    () => movePending.value || loadMorePending.value || searchPending.value,
  );

  return {
    lanes,
    controller,
    movePending,
    loadMorePending,
    searchPending,
    boardBusy,
    applySearch,
    onLaneLoadMore,
    persistMove,
    syncLanesFromController,
    bootstrap,
  };
}
