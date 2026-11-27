<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyViewBase
    data-anchor="choy.kanban-view"
    :class="['choy-kanban-view', 'gap-2', props.class]"
    chrome-anchor="choy.kanban.view-chrome"
    :show-header="
      showHeader &&
      (!!slots.search ||
        !!slots['header-right'] ||
        showActions ||
        !!pageTitle ||
        showTitleOverflow)
    "
    :show-actions="showActions"
    :page-title="pageTitle"
    :page-title-id="pageTitleId"
    page-title-class="choy-kanban-view__page-title"
    :page-title-actions-io="pageTitleActionsIo"
    :show-title-overflow="showTitleOverflow"
    :store="boundStore"
    :busy="boardBusy"
    :mode="boardBusyMode"
    :sheet-surface="true"
    sheet-pad="board"
    busy-host-class="choy-kanban-view__body flex min-h-0 min-w-0 flex-1 flex-col gap-3 overflow-hidden"
    :error="boardSurfaceError"
    :has-frame="boardHasFrame"
    :has-settled="boardHasSettled"
    :error-fallback-title="_t('Failed to load')"
    @retry="onSurfaceRetry"
    @cancel="onSurfaceCancel"
  >
    <template v-if="showActions" #system-actions>
      <slot name="system-actions">
        <!--
          Keep New mounted during search/move busy. Hiding it behind
          boardBusy made the action bar flash when switching into Kanban.
        -->
        <ChoyActionButton
          v-if="!props.readonly"
          type="button"
          size="default"
          variant="soft-primary"
          :busy="boardBusy"
          :action-id="actionIds?.create"
          :has-action="hasAction"
          @execute="onCreate"
        >
          <Plus class="size-4" aria-hidden="true" />
          {{ resolvedCreateLabel }}
        </ChoyActionButton>
      </slot>
    </template>

    <template v-if="showRefreshInOverflow || !!slots['overflow-actions']" #overflow-actions>
      <ChoyViewOverflowRefreshItem
        v-if="showRefreshInOverflow"
        test-id="kanban-overflow-refresh"
        @select="onRefresh"
      />
      <slot name="overflow-actions" />
    </template>

    <template v-if="showActions && !!slots['secondary-actions']" #secondary-actions>
      <slot name="secondary-actions" />
    </template>

    <template
      v-if="slots.search || slots['contextual-actions'] || slots['user-actions']"
      #center
    >
      <div
        v-if="slots['contextual-actions'] || slots['user-actions']"
        data-anchor="choy.kanban.contextual-actions"
        data-testid="kanban-contextual-actions"
        class="flex min-w-0 w-full flex-nowrap items-center justify-center gap-2 overflow-x-auto"
        role="toolbar"
        :aria-label="_t('Contextual actions')"
      >
        <slot name="contextual-actions" />
        <slot name="user-actions" />
      </div>
      <ChoySearchFit v-if="slots.search">
        <slot name="search" :on-query-update="engine.applySearch" />
      </ChoySearchFit>
    </template>

    <template v-if="slots['header-right']" #header-right>
      <slot name="header-right" />
    </template>

      <div v-if="laneCount > 0 && !isFlatBoard" class="md:hidden">
        <Select :model-value="mobileLaneKey" @update:model-value="onMobileLaneChange">
          <SelectTrigger
            class="w-full"
            data-testid="choy-kanban-mobile-lane"
            :placeholder="_t('Select lane')"
            :aria-label="_t('Select lane')"
          />
          <SelectContent>
            <SelectItem v-for="lane in displayLanes" :key="lane.key" :value="lane.key">
              {{ lane.label }} · {{ choyKanbanLaneTotal(lane) }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>

      <!-- Ungrouped: tile cards across the board (Odoo-style flat kanban). -->
      <div
        v-if="isFlatBoard"
        class="choy-kanban-view__board choy-kanban-view__board--flat grid min-h-0 flex-1 content-start gap-3 overflow-y-auto pb-1 [grid-template-columns:repeat(auto-fill,minmax(18rem,1fr))]"
        role="list"
        data-testid="choy-kanban-flat-board"
        :aria-label="_t('Kanban board')"
      >
        <template v-for="lane in displayLanes" :key="lane.key">
          <article
            v-for="card in lane.cards"
            :key="card.id"
            class="choy-kanban-view__card flex flex-col cursor-pointer rounded-md border border-border bg-card p-3 shadow-xs outline-none transition-colors hover:bg-accent/40 focus-visible:ring-2 focus-visible:ring-ring"
            :data-card-id="card.id"
            tabindex="0"
            role="listitem"
            :aria-label="card.title"
            @click="onCardClick(card)"
            @keydown.enter.self.prevent="onCardClick(card)"
          >
            <slot name="card" :card="card" :lane="lane">
              <ChoyKanbanCardFallback :card="card" />
            </slot>
          </article>

          <div
            v-if="lane.cards.length === 0"
            class="col-span-full rounded-md border border-dashed border-border px-3 py-12 text-center text-sm text-foreground/50"
            role="status"
          >
            <slot name="card-empty" :lane="lane">{{ _t('No cards') }}</slot>
          </div>

          <ChoyKanbanLoadSentinel
            v-if="choyKanbanLaneRemain(lane) > 0"
            class="col-span-full"
            :disabled="loadMoreBusy"
            @visible="onLaneLoadMore(lane)"
          />

          <div
            v-if="$slots['lane-footer'] || choyKanbanLaneRemain(lane) > 0"
            class="choy-kanban-view__lane-footer col-span-full"
          >
            <slot
              name="lane-footer"
              :lane="lane"
              :remain="choyKanbanLaneRemain(lane)"
              :load-more="() => onLaneLoadMore(lane)"
              :busy="isLaneFetchBusy(lane.key)"
            >
              <ChoyButton
                v-if="choyKanbanLaneRemain(lane) > 0"
                type="button"
                variant="ghost"
                size="sm"
                class="w-full"
                data-testid="choy-kanban-load-more"
                :disabled="loadMoreBusy"
                @click="onLaneLoadMore(lane)"
              >
                {{ formatChoyKanbanLoadMoreLabel(choyKanbanLaneRemain(lane), (n) => _t('%s more', n)) }}
              </ChoyButton>
            </slot>
          </div>
        </template>
      </div>

      <div
        v-else
        class="choy-kanban-view__board flex min-h-0 flex-1 gap-3 overflow-x-auto overflow-y-hidden pb-1"
        role="list"
        :aria-label="_t('Kanban board')"
        :data-drag-mode="resolvedDragMode"
        data-testid="choy-kanban-board"
        @dragend="laneDropActiveKey = ''"
      >
        <section
          v-for="lane in displayLanes"
          :key="lane.key"
          class="choy-kanban-view__lane flex h-full min-h-0 w-72 shrink-0 flex-col rounded-md bg-muted/70"
          :class="{ 'ring-2 ring-ring/40': laneDropActiveKey === lane.key }"
          role="listitem"
          :aria-label="lane.label"
          :data-mobile-active="lane.key === mobileLaneKey ? 'true' : 'false'"
          :data-drag-mode="resolvedDragMode"
          @dragover="onLaneDragOver($event, lane)"
          @dragleave="onLaneDragLeave($event, lane)"
          @drop="onLaneDrop($event, lane)"
        >
          <!-- Soft lane wash only (no outer border); cards keep the bordered surface. -->
          <header class="flex items-center justify-between gap-2 px-2 pb-2 pt-2">
            <slot name="lane-header" :lane="lane">
              <span class="min-w-0 truncate text-sm font-medium text-foreground">{{ lane.label }}</span>
              <ChoyBadge
                variant="secondary"
                class="h-5 shrink-0 px-1.5 text-[11px] font-normal tabular-nums"
                data-testid="choy-kanban-lane-count"
              >
                {{ choyKanbanLaneTotal(lane) }}
              </ChoyBadge>
            </slot>
          </header>

          <ChoyKanbanVirtualLaneBody
            :cards="lane.cards"
            :lane="lane"
            :virtualize="virtualize"
            :estimate-card-size="estimateCardSize"
            :remain="choyKanbanLaneRemain(lane)"
            :load-more-busy="loadMoreBusy"
            :lane-drag="resolvedDragMode === 'lane' && !boardBusy && !loadMoreBusy"
            :other-lanes="otherLanes(lane.key)"
            :empty-label="_t('No cards')"
            :move-to-lane-label="_t('Move to lane')"
            :move-to-label-fn="(label: string) => _t('Move to %s', label)"
            @card-click="onCardClick"
            @load-more="onLaneLoadMore(lane)"
            @move-to-lane="(card, toKey) => moveCardToLane(card, lane.key, toKey)"
          >
            <template #card="slotProps">
              <slot name="card" v-bind="slotProps">
                <ChoyKanbanCardFallback :card="slotProps.card" />
              </slot>
            </template>
            <template #card-empty>
              <slot name="card-empty" :lane="lane">{{ _t('No cards') }}</slot>
            </template>
          </ChoyKanbanVirtualLaneBody>

          <div
            v-if="$slots['lane-footer'] || choyKanbanLaneRemain(lane) > 0"
            class="choy-kanban-view__lane-footer px-2 pb-2 pt-1"
          >
            <slot
              name="lane-footer"
              :lane="lane"
              :remain="choyKanbanLaneRemain(lane)"
              :load-more="() => onLaneLoadMore(lane)"
              :busy="isLaneFetchBusy(lane.key)"
            >
              <ChoyButton
                v-if="choyKanbanLaneRemain(lane) > 0"
                type="button"
                variant="ghost"
                size="sm"
                class="w-full"
                data-testid="choy-kanban-load-more"
                :disabled="loadMoreBusy"
                @click="onLaneLoadMore(lane)"
              >
                {{ formatChoyKanbanLoadMoreLabel(choyKanbanLaneRemain(lane), (n) => _t('%s more', n)) }}
              </ChoyButton>
            </slot>
          </div>
        </section>

        <div
          v-if="laneCount === 0"
          class="flex w-full items-center justify-center rounded-md border border-dashed border-border py-12 text-sm text-foreground/50"
          role="status"
        >
          {{ _t('No lanes') }}
        </div>
      </div>
  </ChoyViewBase>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, ref, useSlots, watch, type PropType } from 'vue';
import { Plus } from 'lucide-vue-next';
import type { ClassValue } from '../../lib/utils';
import type { ModelStore } from '@/web/web/stores/modelStore';
import type { Lane } from '@/web/web/query/types';
import ChoyBadge from '../layout/ChoyBadge.vue';
import ChoyButton from '../button/ChoyButton.vue';
import ChoyActionButton from '../button/ChoyActionButton.vue';
import type { ActionIdMap } from '@/web/web/components/view/actionVisibility';
import ChoyViewBase from '@/web/web/components/view/ChoyViewBase.vue';
import ChoySearchFit from '@/web/web/components/view/ChoySearchFit.vue';
import ChoyViewOverflowRefreshItem from '@/web/web/components/view/ChoyViewOverflowRefreshItem.vue';
import ChoyKanbanLoadSentinel from '@/web/web/components/view/ChoyKanbanLoadSentinel.vue';
import ChoyKanbanVirtualLaneBody from '@/web/web/components/view/ChoyKanbanVirtualLaneBody.vue';
import { useProvideViewScope } from '@/web/web/components/view/viewScopeKeys';
import { useViewTitleOverflow } from '@/web/web/components/view/useViewTitleOverflow';
import { useCollectionBusyMode } from '@/web/web/components/view/useCollectionBusyMode';
import { useControllerSoftFailSurface } from '@/web/web/components/view/useControllerSoftFailSurface';
import { useDetailPathRecordSetHygiene } from '@/web/web/components/view/useDetailPathRecordSetHygiene';
import { useDetailPathCollectionSoftRefresh } from '@/web/web/components/view/useDetailPathCollectionSoftRefresh';
import Select from '../vendor/ui/select/Select.vue';
import SelectContent from '../vendor/ui/select/SelectContent.vue';
import SelectItem from '../vendor/ui/select/SelectItem.vue';
import SelectTrigger from '../vendor/ui/select/SelectTrigger.vue';
import {
  resolvePageStore,
  useHostPageTitleInView,
  useOptionalPageStore,
  useRegisterPageActionTarget,
} from '@/web/web/composables/usePageContext';
import { useChoyKanbanStoreEngine } from '@/web/web/composables/useChoyKanbanStoreEngine';
import { useSearchFirstFrameHost } from '@/web/web/components/view/search/searchFirstFrame';
import { ChoyMessage } from '../../composables/useChoyMessage';
import { createTranslate } from '@/web/web/i18n';
import {
  applyChoyKanbanMove,
  choyKanbanLaneRemain,
  choyKanbanLaneTotal,
  formatChoyKanbanLoadMoreLabel,
  isChoyKanbanFlatBoard,
  type ChoyKanbanCard,
  type ChoyKanbanLane,
  type ChoyKanbanLoadMore,
  type ChoyKanbanMove,
} from './kanbanViewHelpers';
import {
  CHOY_KANBAN_LANE_DRAG_MIME,
  decodeChoyKanbanLaneDragPayload,
  resolveChoyKanbanLaneDropMove,
  type ChoyKanbanDragMode,
  type ChoyKanbanVirtualizeMode,
} from './kanbanVirtualHelpers';
import {
  canOpenDetailRow,
  resolveDetailPath,
  type DetailPathProp,
} from '@/web/web/navigation/detailPath';
import { openRecordFromCollection } from '@/web/web/navigation/openRecordFromCollection';
import { snapshotFromListStore } from '@/web/web/navigation/snapshotFromListStore';
import { useRecordSetStore } from '@/web/web/stores/recordSetStore';
import { useRoute, useRouter } from 'vue-router';

/** Shared default card body for lane cards. */
const ChoyKanbanCardFallback = defineComponent({
  name: 'ChoyKanbanCardFallback',
  props: {
    card: { type: Object as PropType<ChoyKanbanCard>, required: true },
  },
  setup(props) {
    return () => [
      h('div', { class: 'text-sm font-medium text-foreground' }, props.card.title),
      props.card.subtitle
        ? h('div', { class: 'mt-1 text-xs text-foreground/60' }, props.card.subtitle)
        : null,
    ];
  },
});

/**
 * Store-bound kanban board. Requires :store or a page-provided store.
 * Owns createKanbanController, lane sync, search apply, load-more, and move persist.
 * Lane bodies use VirtualLaneBody; cross-lane moves use HTML5 whole-lane drop.
 */
const { _t } = createTranslate('web', { scope: 'web/components/view/KanbanView' });

useProvideViewScope({ kind: 'Kanban', mode: 'display' });

const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    store?: ModelStore<any>;
    showHeader?: boolean;
    showActions?: boolean;
    createLabel?: string;
    actionIds?: ActionIdMap;
    hasAction?: (actionId: string | undefined) => boolean;
    readonly?: boolean;
    /**
     * Card drag behavior. `readonly` forces `none`.
     * - `lane`: HTML5 drag + whole-lane drop (append to target lane)
     * - `none`: no drag
     */
    dragMode?: ChoyKanbanDragMode;
    /** Lane-body virtualization: `auto` uses the loaded-card threshold. */
    virtualize?: ChoyKanbanVirtualizeMode;
    /** Estimated card height for the virtualizer (px). */
    estimateCardSize?: number;
    keywordFields?: string[];
    titleField?: string;
    flatLaneLabel?: string;
    /** Preferred lane key/token order after sync (e.g. installed before uninstalled). */
    laneOrder?: ReadonlyArray<string>;
    laneOrderToken?: (lane: Pick<ChoyKanbanLane, 'key' | 'label'>) => string;
    laneLabel?: (lane: Pick<ChoyKanbanLane, 'key' | 'label'> | Lane) => string;
    mapRowToCard?: (row: unknown, index: number, laneKey: string) => ChoyKanbanCard;
    resolveMoveRecordId?: (
      cards: ReadonlyArray<ChoyKanbanCard>,
      moveCardId: string,
    ) => string;
    beforeBootstrap?: () => Promise<void>;
    autoBootstrap?: boolean;
    onLoadError?: (error: unknown) => void;
    onSearchError?: (error: unknown) => void;
    onLoadMoreError?: (error: unknown) => void;
    onMoveError?: (error: unknown) => void;
    /** Opt out of page IO action-target registration (embedded boards). */
    registerActionTarget?: boolean;
    /**
     * When set, card open seeds the Form record pager and navigates.
     * Same contract as ChoyListView `detailPath`.
     */
    detailPath?: DetailPathProp;
    canOpenDetail?: boolean | ((row: Record<string, unknown>) => boolean);
  }>(),
  {
    showHeader: true,
    showActions: true,
    createLabel: undefined,
    readonly: false,
    dragMode: 'lane',
    virtualize: 'auto',
    autoBootstrap: true,
    // Keep omitted as undefined (not false) so matching page-store views auto-register.
    registerActionTarget: undefined,
    // Vue Boolean cast would otherwise default a missing canOpenDetail to false.
    canOpenDetail: true,
  },
);

const slots = useSlots();
const { pageTitle, pageTitleId, pageTitleActionsIo } = useHostPageTitleInView(() => props.showHeader);
const { showRefreshInOverflow, showTitleOverflow } = useViewTitleOverflow({
  pageTitleActionsIo,
  showRefresh: () => props.showActions && props.showHeader,
  hasOverflowSlot: () => !!slots['overflow-actions'],
});

const emit = defineEmits<{
  'card-click': [card: ChoyKanbanCard];
  'card-move': [move: ChoyKanbanMove];
  'lane-load-more': [payload: ChoyKanbanLoadMore];
  create: [];
}>();

const pageStore = useOptionalPageStore<ModelStore<any>>();
const boundStore = resolvePageStore(props.store ?? pageStore.value, 'ChoyKanbanView');
// Chrome unit mounts use mountApp without vue-router; keep setup resilient.
let router: ReturnType<typeof useRouter> | null = null;
let route: ReturnType<typeof useRoute> | null = null;
try {
  router = useRouter();
  route = useRoute();
} catch {
  router = null;
  route = null;
}
useDetailPathRecordSetHygiene({
  detailPath: () => props.detailPath,
  modelName: () => (boundStore as { modelName?: string }).modelName,
});

function handleMoveError(error: unknown): void {
  if (props.onMoveError) {
    props.onMoveError(error);
    return;
  }
  const message =
    error instanceof Error && error.message
      ? error.message
      : _t('Failed to move card');
  ChoyMessage.error(message);
}

const firstFrame = useSearchFirstFrameHost(boundStore, {
  // Header (and search slot outlet) only mounts when showHeader is true.
  expectFirstFrame: () => !!props.showHeader && !!slots.search,
});

const engine = useChoyKanbanStoreEngine({
  store: boundStore,
  keywordFields: () => props.keywordFields,
  titleField: () => props.titleField,
  flatLaneLabel: () => props.flatLaneLabel,
  laneOrder: () => props.laneOrder,
  laneOrderToken: props.laneOrderToken,
  laneLabel: props.laneLabel,
  mapRowToCard: props.mapRowToCard,
  resolveMoveRecordId: props.resolveMoveRecordId,
  beforeBootstrap: props.beforeBootstrap,
  autoBootstrap: props.autoBootstrap,
  firstFrameGate: firstFrame,
  onLoadError: props.onLoadError,
  onSearchError: props.onSearchError,
  onLoadMoreError: props.onLoadMoreError,
  onMoveError: handleMoveError,
});

// KeepAlive return from Form: silent board refresh; skips mount activate.
useDetailPathCollectionSoftRefresh({
  detailPath: () => props.detailPath,
  hasSettled: () => !!engine.controller.vm.hasAppliedOnce,
  refresh: () => engine.refresh(),
});

useRegisterPageActionTarget({
  store: boundStore,
  enabled: () => props.registerActionTarget,
  target: {
    get selectedItems() {
      // Kanban has no row selection chrome; IO export uses an empty selection.
      return [] as Array<{ Id?: string }>;
    },
    refresh: () => engine.refresh(),
  },
});

const displayLanes = computed({
  get(): ChoyKanbanLane[] {
    return engine.lanes.value;
  },
  set(next: ChoyKanbanLane[]) {
    engine.lanes.value = next;
  },
});

/** Search / move: drives BusyHost and blocks drag; does not hide New. */
const boardBusy = computed(() => engine.boardBusy.value);
const boardSurfaceError = computed(() => engine.controller.vm.error);
const boardHasSettled = computed(() => !!engine.controller.vm.hasAppliedOnce);
const boardHasFrame = computed(() => {
  const lanes = engine.lanes.value;
  if (Array.isArray(lanes) && lanes.length > 0) return true;
  const rows = (engine.controller.vm.result as { rows?: unknown[] } | null)?.rows;
  return Array.isArray(rows) && rows.length > 0;
});
// Busy mode keys off rendered lanes only: group rows can land before
// syncLanesFromController assigns lanes; keep block overlay until then.
const boardBusyMode = useCollectionBusyMode({
  hasContent: () => {
    const lanes = engine.lanes.value;
    return Array.isArray(lanes) && lanes.length > 0;
  },
});

const { onSurfaceRetry, onSurfaceCancel } = useControllerSoftFailSurface({
  getController: () => engine.controller as any,
  reload: () => engine.refresh(),
});
/** Any lane append in flight (blocks drops board-wide so pages are not overwritten). */
const loadMoreBusy = computed(() => engine.loadMorePending.value);
/** Lane key currently appending; busy chrome is scoped to this lane only. */
const loadMoreLaneKey = computed(() => engine.loadMoreLaneKey.value);

/** True while this lane should disable its own load-more control / sentinel. */
function isLaneFetchBusy(laneKey: string): boolean {
  if (boardBusy.value) return true;
  return loadMoreBusy.value && loadMoreLaneKey.value === laneKey;
}

const effectiveReadonly = computed(() => props.readonly || boardBusy.value);
const resolvedDragMode = computed<ChoyKanbanDragMode>(() => {
  if (props.readonly) return 'none';
  return props.dragMode ?? 'lane';
});
const resolvedCreateLabel = computed(() => props.createLabel || _t('New'));
const laneCount = computed(() => displayLanes.value.length);
const isFlatBoard = computed(() => isChoyKanbanFlatBoard(displayLanes.value));

const mobileLaneKey = ref('');
/** Lane key currently highlighted as an HTML5 drop target. */
const laneDropActiveKey = ref('');

watch(
  displayLanes,
  (lanes) => {
    if (!lanes.length) {
      mobileLaneKey.value = '';
      return;
    }
    if (!lanes.some((l) => l.key === mobileLaneKey.value)) {
      mobileLaneKey.value = lanes[0]!.key;
    }
  },
  { immediate: true },
);

function onMobileLaneChange(value: unknown): void {
  if (typeof value === 'string' && value) {
    mobileLaneKey.value = value;
  }
}

function otherLanes(fromKey: string): ChoyKanbanLane[] {
  return displayLanes.value.filter((l) => l.key !== fromKey);
}

function onCardClick(card: ChoyKanbanCard): void {
  if (props.detailPath != null) {
    if (!router) {
      emit('card-click', card);
      return;
    }
    const row = (card.payload || {}) as Record<string, unknown>;
    let to: ReturnType<typeof resolveDetailPath> = null;
    try {
      if (!canOpenDetailRow(props.canOpenDetail as any, row)) {
        emit('card-click', card);
        return;
      }
      to = resolveDetailPath(props.detailPath, row);
    } catch {
      emit('card-click', card);
      return;
    }
    if (to == null) {
      emit('card-click', card);
      return;
    }
    const model = String((boundStore as { modelName?: string }).modelName || '').trim();
    const id = String(row.Id ?? row.id ?? card.id ?? '').trim();
    if (!id) {
      emit('card-click', card);
      return;
    }
    // Grouped lane cards are not Search offsets — open without a pager session.
    // Flat boards keep Search pagination offset via snapshotFromListStore.
    const isGrouped =
      (boundStore as { state?: { result?: { kind?: string } } }).state?.result?.kind === 'group';
    if (isGrouped) {
      try {
        useRecordSetStore().clear();
      } catch {
        // Pinia may be unavailable in unit mounts without an app shell.
      }
      void router.push(to as any);
      return;
    }
    const entry = snapshotFromListStore(boundStore as any, {
      model: model || 'unknown',
      key: `${model || 'unknown'}|${String(route?.path || 'kanban')}`,
      rowIndex: 0,
      recordId: id,
    });
    // Seed only when the opened id is in the known Search window.
    if (entry && entry.ids.length > 0 && entry.ids.includes(id)) {
      openRecordFromCollection({ router, to, entry });
    } else {
      try {
        useRecordSetStore().clear();
      } catch {
        // Pinia may be unavailable in unit mounts without an app shell.
      }
      void router.push(to as any);
    }
    return;
  }
  emit('card-click', card);
}

function onCreate(): void {
  emit('create');
}

function onRefresh(): void {
  void engine.refresh().catch(() => undefined);
}

function onLaneLoadMore(lane: ChoyKanbanLane): void {
  if (choyKanbanLaneRemain(lane) <= 0) return;
  const payload = { laneKey: lane.key };
  emit('lane-load-more', payload);
  void engine.onLaneLoadMore(payload);
}

function commitMove(move: ChoyKanbanMove): void {
  const next = applyChoyKanbanMove(displayLanes.value, move);
  if (!next) return;
  displayLanes.value = next;
  const finalLane = next.find((l) => l.key === move.toLaneKey);
  const finalIndex = finalLane
    ? finalLane.cards.findIndex((c) => c.id === move.cardId)
    : move.toIndex;
  const finalized: ChoyKanbanMove = {
    ...move,
    toIndex: finalIndex < 0 ? move.toIndex : finalIndex,
  };
  emit('card-move', finalized);
  void engine.persistMove(finalized);
}

function moveCardToLane(card: ChoyKanbanCard, fromLaneKey: string, toLaneKey: string): void {
  if (effectiveReadonly.value) return;
  const toLane = displayLanes.value.find((l) => l.key === toLaneKey);
  if (!toLane) return;
  commitMove({
    cardId: card.id,
    fromLaneKey,
    toLaneKey,
    toIndex: toLane.cards.length,
  });
}

function laneDropEnabled(): boolean {
  // load-more replaces the lane array from the server, so a drop in that
  // window would be overwritten by the arriving page.
  return resolvedDragMode.value === 'lane' && !effectiveReadonly.value && !loadMoreBusy.value;
}

function readLaneDragPayload(event: DragEvent) {
  const dt = event.dataTransfer;
  if (!dt) return null;
  return decodeChoyKanbanLaneDragPayload(dt.getData(CHOY_KANBAN_LANE_DRAG_MIME));
}

function onLaneDragOver(event: DragEvent, lane: ChoyKanbanLane): void {
  if (!laneDropEnabled()) return;
  const types = event.dataTransfer?.types;
  const accepts =
    !!types &&
    (Array.from(types as ArrayLike<string>).includes(CHOY_KANBAN_LANE_DRAG_MIME) ||
      Array.from(types as ArrayLike<string>).includes('text/plain'));
  if (!accepts) return;
  event.preventDefault();
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'move';
  laneDropActiveKey.value = lane.key;
}

function onLaneDragLeave(event: DragEvent, lane: ChoyKanbanLane): void {
  if (!laneDropEnabled()) return;
  const related = event.relatedTarget as Node | null;
  const current = event.currentTarget as Node | null;
  if (related && current && current.contains(related)) return;
  if (laneDropActiveKey.value === lane.key) {
    laneDropActiveKey.value = '';
  }
}

function onLaneDrop(event: DragEvent, lane: ChoyKanbanLane): void {
  if (!laneDropEnabled()) return;
  event.preventDefault();
  laneDropActiveKey.value = '';
  const payload = readLaneDragPayload(event);
  const move = resolveChoyKanbanLaneDropMove({
    payload,
    toLaneKey: lane.key,
    toLaneCardCount: lane.cards.length,
  });
  if (!move) return;
  commitMove(move);
}

defineExpose({
  bootstrap: () => engine.bootstrap(),
  refresh: () => engine.refresh(),
  /** BusyHost surface Retry. */
  retrySurface: onSurfaceRetry,
  applySearch: engine.applySearch,
  syncLanes: () => engine.syncLanesFromController(),
  boardBusy,
});
</script>

<style scoped>
@media (max-width: 767px) {
  .choy-kanban-view__board > .choy-kanban-view__lane:not([data-mobile-active='true']) {
    display: none;
  }
}

</style>
