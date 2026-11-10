// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, onMounted, ref, toValue, watch, type MaybeRefOrGetter, type Ref } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { createKanbanController } from '@/web/web/controllers/kanbanController';
import { awaitFieldSelection } from '@/web/web/query/utils/registry/fieldReady';
import type { ChoySearchQuery } from '@/web/web/components/view/searchViewHelpers';
import type { Lane } from '@/web/web/query/types';
import {
  CHOY_KANBAN_FLAT_LANE_KEY,
  sortChoyKanbanLanesByOrder,
  type ChoyKanbanCard,
  type ChoyKanbanLane,
  type ChoyKanbanLoadMore,
  type ChoyKanbanMove,
} from '@/web/web/components/view/kanbanViewHelpers';
import {
  createLaneSyncGate,
  defaultKanbanMapRowToCard,
  defaultKanbanMoveRecordId,
  shouldRecoverStaleKanbanSearch,
  shouldRestoreKanbanMove,
} from '@/web/web/components/view/kanbanStoreHelpers';
import { finishInitialSearchBoundLoad } from '@/web/web/components/view/viewFirstFrame';
import type { SearchFirstFrameGate } from '@/web/web/components/view/search/searchFirstFrame';

export type ChoyKanbanStoreEngineOptions = {
  store: WebModelStore<any>;
  keywordFields?: MaybeRefOrGetter<string[] | undefined>;
  titleField?: MaybeRefOrGetter<string | undefined>;
  flatLaneLabel?: MaybeRefOrGetter<string | undefined>;
  /** Preferred lane order tokens matched against lane keys/labels after sync. */
  laneOrder?: MaybeRefOrGetter<ReadonlyArray<string> | undefined>;
  /** Optional resolver for laneOrder tokens (defaults to lane.key). */
  laneOrderToken?: (lane: Pick<ChoyKanbanLane, 'key' | 'label'>) => string;
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
  /**
   * When true, skip empty first apply and wait for Search first-frame
   * query-update (legacy; prefer firstFrameGate).
   */
  deferFirstFrame?: MaybeRefOrGetter<boolean | undefined>;
  /** Preferred: gate from useSearchFirstFrameHost. */
  firstFrameGate?: SearchFirstFrameGate | null;
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
  /** Lane key currently appending pages; null when idle. */
  const loadMoreLaneKey = ref<string | null>(null);
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

  function orderLanes(next: ChoyKanbanLane[]): ChoyKanbanLane[] {
    const order = toValue(opts.laneOrder);
    if (!order?.length) return next;
    const tokenOf =
      opts.laneOrderToken ||
      ((lane: Pick<ChoyKanbanLane, 'key' | 'label'>) => String(lane.key || ''));
    return sortChoyKanbanLanesByOrder(next, order, tokenOf);
  }

  function materializeGroupedLane(lane: Lane): ChoyKanbanLane {
    const key = String(lane.key ?? '');
    return {
      key,
      label: labelFor(lane),
      remain: controller.getLaneRemain(lane),
      cards: (controller.laneRecords.value[key] || []).map((row, index) => mapRow(row, index, key)),
    };
  }

  /**
   * Refresh one lane's cards/remain after load-more without rematerializing siblings
   * (avoids virtualizer / drag-chrome flicker in untouched columns).
   */
  function patchLaneFromController(laneKey: string): boolean {
    // Preserve surrounding whitespace in lane keys; only reject blank-after-trim.
    const key = String(laneKey ?? '');
    if (!key.trim()) return false;
    const ctrlLane = controller.lanes.value.find((lane) => String(lane.key) === key);
    const idx = lanes.value.findIndex((lane) => lane.key === key);
    if (!ctrlLane || idx < 0) return false;
    const next = lanes.value.slice();
    next[idx] = materializeGroupedLane(ctrlLane);
    lanes.value = next;
    return true;
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
          const flatKey = CHOY_KANBAN_FLAT_LANE_KEY;
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
        lanes.value = orderLanes(laneList.map((lane) => materializeGroupedLane(lane)));
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

  /**
   * Sequenced search apply. `rethrow` is for refresh callers that must surface
   * failures; Search `query-update` handlers keep the default (no rethrow) so
   * unhandled Vue emit rejections do not escape.
   */
  async function runSearch(query: ChoySearchQuery, rethrow: boolean): Promise<void> {
    lastSearchQuery = query;
    const seq = ++searchSeq;
    searchInFlight++;
    searchPending.value = true;
    const keywordFields = toValue(opts.keywordFields);
    let applyError: unknown;
    try {
      await controller.apply({
        keyword: query.keyword,
        appliedFilters: (query.appliedFilters || []) as any,
        appliedGroups: query.appliedGroups as any,
        ...(keywordFields ? { keywordFields } : {}),
      });
    } catch (e) {
      applyError = e;
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
        if (recoverSeq === searchSeq) {
          opts.onSearchError?.(e);
          if (rethrow) throw e;
        }
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
    if (applyError) {
      searchPending.value = false;
      if (rethrow) throw applyError;
      return;
    }
    try {
      await syncLanesFromController();
    } catch (e) {
      // Lane sync failures are reported but not rethrown: a throwing map/preload
      // can also surface via the lanes watcher as an unhandled rejection.
      if (seq === searchSeq) opts.onSearchError?.(e);
    } finally {
      if (seq === searchSeq) searchPending.value = false;
    }
  }

  async function applySearch(query: ChoySearchQuery): Promise<void> {
    await runSearch(query, false);
  }

  async function onLaneLoadMore(payload: ChoyKanbanLoadMore): Promise<void> {
    // Keep the controller key as-is (may include padding); only reject blank keys.
    const laneKey = String(payload?.laneKey ?? '');
    if (
      loadMorePending.value ||
      movePending.value ||
      searchPending.value ||
      !laneKey.trim()
    ) {
      return;
    }
    loadMorePending.value = true;
    loadMoreLaneKey.value = laneKey;
    try {
      await controller.loadMoreLane(laneKey);
      if (!patchLaneFromController(laneKey)) {
        await syncLanesFromController();
      }
    } catch (e) {
      opts.onLoadMoreError?.(e);
    } finally {
      loadMorePending.value = false;
      loadMoreLaneKey.value = null;
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
      await finishInitialSearchBoundLoad({
        getLastSearchQuery: () => lastSearchQuery,
        gate: opts.firstFrameGate,
        deferToSearch: !!toValue(opts.deferFirstFrame),
        applyEmpty: () =>
          controller.apply(keywordFields?.length ? { keywordFields } : {}),
        afterEmpty: syncLanesFromController,
      });
    } catch (e) {
      // Failure must clear the latch so bootstrap()/host retry can run again.
      bootstrapped = false;
      opts.onLoadError?.(e);
      throw e;
    }
  }

  /**
   * Re-apply store queryState through searchSeq without latching lastSearchQuery
   * (avoids caching a synthetic empty Search query over real store filters).
   */
  async function refreshFromQueryState(rethrow: boolean): Promise<void> {
    const seq = ++searchSeq;
    searchInFlight++;
    searchPending.value = true;
    try {
      await applyCurrentQuery();
      if (seq !== searchSeq) return;
      await syncLanesFromController();
    } catch (e) {
      if (seq === searchSeq) {
        opts.onSearchError?.(e);
        if (rethrow) throw e;
      }
    } finally {
      searchInFlight--;
      if (seq === searchSeq) searchPending.value = false;
    }
  }

  /**
   * Re-apply the current search (or store queryState) through the sequenced
   * applySearch path so concurrent first-frame searches cannot be overwritten.
   */
  async function refresh(): Promise<void> {
    if (!bootstrapped) {
      await bootstrap();
      // Bootstrap already ran applyEmpty when not deferring; only re-run when a
      // search query landed during bootstrap (so refresh can surface errors).
      if (lastSearchQuery) {
        await runSearch(lastSearchQuery, true);
      }
      return;
    }
    if (lastSearchQuery) {
      await runSearch(lastSearchQuery, true);
      return;
    }
    // No search yet: do not apply while Search still owns the first frame.
    if (opts.firstFrameGate?.shouldDefer.value || !!toValue(opts.deferFirstFrame)) {
      return;
    }
    await refreshFromQueryState(true);
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

  // Full-board BusyHost + drag lock: search and move only.
  // Lane load-more uses loadMorePending so the overlay does not flash.
  // Action-bar New stays mounted while this is true (see ChoyKanbanView).
  const boardBusy = computed(() => movePending.value || searchPending.value);

  return {
    lanes,
    controller,
    movePending,
    loadMorePending,
    loadMoreLaneKey,
    searchPending,
    boardBusy,
    applySearch,
    onLaneLoadMore,
    persistMove,
    syncLanesFromController,
    bootstrap,
    refresh,
  };
}
