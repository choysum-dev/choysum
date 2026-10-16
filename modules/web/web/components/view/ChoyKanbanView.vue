<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyViewContainer
    data-anchor="choy.kanban-view"
    :class="['choy-kanban-view', 'gap-4', props.class]"
  >
    <template
      v-if="
        showHeader &&
        ($slots.header ||
          $slots.search ||
          $slots['header-right'] ||
          showActions ||
          pageTitle ||
          showTitleOverflow)
      "
      #header
    >
      <div data-anchor="choy.kanban.view-chrome">
        <ChoyViewActionBar
          layout="thirds"
          class="gap-2 border-b-0 [border-bottom-width:0px]"
          :system-actions-label="_t('System actions')"
          :user-actions-label="_t('User actions')"
        >
          <template v-if="showActions || pageTitle || showTitleOverflow" #system-actions>
            <slot v-if="showActions" name="system-actions">
              <ChoyButton
                v-if="!effectiveReadonly"
                size="default"
                @click="onCreate"
              >
                {{ resolvedCreateLabel }}
              </ChoyButton>
            </slot>

            <!-- Page title + ⋮ live in view chrome, not a separate Page band (List parity). -->
            <h1
              v-if="pageTitle"
              :id="pageTitleId || undefined"
              data-slot="page-title"
              class="choy-kanban-view__page-title min-w-0 max-w-[10rem] truncate text-base font-semibold tracking-tight text-foreground sm:max-w-[14rem]"
            >
              {{ pageTitle }}
            </h1>
            <ChoyPageTitleActions
              v-if="showTitleOverflow"
              :has-io-menu="!!pageTitleActionsIo"
              :action-import="pageTitleActionsIo?.actionImport ?? false"
              :action-export="pageTitleActionsIo?.actionExport ?? false"
              :action-import-upload-hint="pageTitleActionsIo?.actionImportUploadHint"
              :action-import-column-mapping="pageTitleActionsIo?.actionImportColumnMapping"
              :action-list-ref="(pageTitleActionsIo?.actionListRef as any)"
              :action-company-id="pageTitleActionsIo?.actionCompanyId"
              :store="boundStore"
            >
              <template v-if="showRefreshInOverflow || $slots['overflow-actions']" #overflow>
                <DropdownMenuItem
                  v-if="showRefreshInOverflow"
                  data-testid="kanban-overflow-refresh"
                  @select="onRefresh"
                >
                  <RefreshCw class="size-4" aria-hidden="true" />
                  {{ _t('Refresh') }}
                </DropdownMenuItem>
                <slot name="overflow-actions" />
              </template>
            </ChoyPageTitleActions>
          </template>
          <!-- Only mount the tray when the product view actually fills it (List parity). -->
          <template v-if="$slots['user-actions'] || $slots.header" #user-actions>
            <slot name="user-actions" />
            <div v-if="$slots.header" class="min-w-0">
              <slot name="header" />
            </div>
          </template>
          <!-- Keep #center registered so thirds grid stays stable even without search. -->
          <template #center>
            <div
              v-if="$slots.search"
              class="flex min-w-0 w-full items-center max-md:order-2"
            >
              <slot name="search" :on-query-update="engine.applySearch" />
            </div>
          </template>

          <!-- View-mode toggles (and optional trailing chrome); mirrors ListView #header-right. -->
          <template v-if="$slots['header-right']" #header-right>
            <slot name="header-right" />
          </template>
        </ChoyViewActionBar>
      </div>
    </template>

    <ChoyViewBusyHost
      :busy="boardBusy"
      class="choy-kanban-view__body flex min-h-0 min-w-0 flex-1 flex-col gap-3 overflow-hidden"
    >
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
            class="choy-kanban-view__card cursor-pointer rounded-md border border-border bg-card p-3 shadow-xs outline-none transition-colors hover:bg-accent/40 focus-visible:ring-2 focus-visible:ring-ring"
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
            :disabled="boardBusy"
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
              :busy="boardBusy"
            >
              <ChoyButton
                v-if="choyKanbanLaneRemain(lane) > 0"
                type="button"
                variant="ghost"
                size="sm"
                class="w-full"
                data-testid="choy-kanban-load-more"
                :disabled="boardBusy"
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
      >
        <section
          v-for="lane in displayLanes"
          :key="lane.key"
          class="choy-kanban-view__lane flex h-full min-h-0 w-72 shrink-0 flex-col rounded-md bg-muted/40"
          role="listitem"
          :aria-label="lane.label"
          :data-mobile-active="lane.key === mobileLaneKey ? 'true' : 'false'"
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

          <div
            class="choy-kanban-view__lane-body flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto px-2 pb-2"
            :data-testid="readonly ? 'choy-kanban-lane-static' : undefined"
          >
            <template v-if="readonly">
              <article
                v-for="card in lane.cards"
                :key="card.id"
                class="choy-kanban-view__card cursor-pointer rounded-md border border-border bg-card p-3 shadow-xs outline-none transition-colors hover:bg-accent/40 focus-visible:ring-2 focus-visible:ring-ring"
                :data-card-id="card.id"
                tabindex="0"
                role="article"
                :aria-label="card.title"
                @click="onCardClick(card)"
                @keydown.enter.self.prevent="onCardClick(card)"
              >
                <div class="flex items-start gap-1">
                  <div class="min-w-0 flex-1">
                    <slot name="card" :card="card" :lane="lane">
                      <ChoyKanbanCardFallback :card="card" />
                    </slot>
                  </div>
                </div>
              </article>
            </template>
            <draggable
              v-else
              class="flex flex-col gap-2"
              :list="lane.cards"
              item-key="id"
              group="choy-kanban"
              :animation="150"
              :disabled="effectiveReadonly"
              ghost-class="choy-kanban-ghost"
              @change="(evt: DraggableChangeEvent) => onDragChange(lane.key, evt)"
            >
              <template #item="{ element: card }">
                <article
                  class="choy-kanban-view__card rounded-md border border-border bg-card p-3 shadow-xs outline-none transition-colors hover:bg-accent/40 focus-visible:ring-2 focus-visible:ring-ring"
                  :class="effectiveReadonly ? 'cursor-pointer' : 'cursor-grab active:cursor-grabbing'"
                  :data-card-id="card.id"
                  tabindex="0"
                  role="article"
                  :aria-label="card.title"
                  @click="onCardClick(card)"
                  @keydown.enter.self.prevent="onCardClick(card)"
                >
                  <div class="flex items-start gap-1">
                    <div class="min-w-0 flex-1">
                      <slot name="card" :card="card" :lane="lane">
                        <ChoyKanbanCardFallback :card="card" />
                      </slot>
                    </div>
                    <DropdownMenu v-if="!effectiveReadonly && otherLanes(lane.key).length">
                      <DropdownMenuTrigger as-child>
                        <ChoyButton
                          type="button"
                          variant="ghost"
                          size="sm"
                          class="h-7 w-7 shrink-0 px-0"
                          data-testid="choy-kanban-move-to-lane"
                          :aria-label="_t('Move to lane')"
                          @click.stop
                        >
                          <MoreHorizontal class="size-4" aria-hidden="true" />
                        </ChoyButton>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" class="min-w-[10rem]">
                        <DropdownMenuItem
                          v-for="target in otherLanes(lane.key)"
                          :key="target.key"
                          @select="moveCardToLane(card, lane.key, target.key)"
                        >
                          {{ _t('Move to %s', target.label) }}
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </article>
              </template>
            </draggable>

            <div
              v-if="lane.cards.length === 0"
              class="rounded-md border border-dashed border-border/70 px-3 py-6 text-center text-xs text-muted-foreground"
            >
              <slot name="card-empty" :lane="lane">{{ _t('No cards') }}</slot>
            </div>

            <ChoyKanbanLoadSentinel
              v-if="choyKanbanLaneRemain(lane) > 0"
              :disabled="boardBusy"
              @visible="onLaneLoadMore(lane)"
            />
          </div>

          <div
            v-if="$slots['lane-footer'] || choyKanbanLaneRemain(lane) > 0"
            class="choy-kanban-view__lane-footer px-2 pb-2 pt-1"
          >
            <slot
              name="lane-footer"
              :lane="lane"
              :remain="choyKanbanLaneRemain(lane)"
              :load-more="() => onLaneLoadMore(lane)"
              :busy="boardBusy"
            >
              <ChoyButton
                v-if="choyKanbanLaneRemain(lane) > 0"
                type="button"
                variant="ghost"
                size="sm"
                class="w-full"
                data-testid="choy-kanban-load-more"
                :disabled="boardBusy"
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
    </ChoyViewBusyHost>
  </ChoyViewContainer>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, ref, useSlots, watch, type PropType } from 'vue';
import draggable from 'vuedraggable';
import { MoreHorizontal, RefreshCw } from 'lucide-vue-next';
import type { ClassValue } from '../../lib/utils';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { Lane } from '@/web/web/query/types';
import ChoyBadge from '../layout/ChoyBadge.vue';
import ChoyButton from '../layout/ChoyButton.vue';
import ChoyPageTitleActions from '@/web/web/components/layout/ChoyPageTitleActions.vue';
import ChoyViewContainer from '@/web/web/components/view/ChoyViewContainer.vue';
import ChoyViewActionBar from '@/web/web/components/view/ChoyViewActionBar.vue';
import ChoyViewBusyHost from '@/web/web/components/view/ChoyViewBusyHost.vue';
import ChoyKanbanLoadSentinel from '@/web/web/components/view/ChoyKanbanLoadSentinel.vue';
import { useProvideViewScope } from '@/web/web/components/view/viewScopeKeys';
import DropdownMenu from '../vendor/ui/dropdown-menu/DropdownMenu.vue';
import DropdownMenuContent from '../vendor/ui/dropdown-menu/DropdownMenuContent.vue';
import DropdownMenuItem from '../vendor/ui/dropdown-menu/DropdownMenuItem.vue';
import DropdownMenuTrigger from '../vendor/ui/dropdown-menu/DropdownMenuTrigger.vue';
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

/** Shared default card body for readonly and draggable lane cards. */
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
 * Dense Admin chrome: ActionTray, vuedraggable, mobile lane select, move-to-lane menu.
 */
const { _t } = createTranslate('web', { scope: 'web/components/view/KanbanView' });

useProvideViewScope({ kind: 'Kanban', mode: 'display' });

const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    store?: WebModelStore<any>;
    showHeader?: boolean;
    showActions?: boolean;
    createLabel?: string;
    readonly?: boolean;
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
  }>(),
  {
    showHeader: true,
    showActions: true,
    createLabel: undefined,
    readonly: false,
    autoBootstrap: true,
    // Keep omitted as undefined (not false) so matching page-store views auto-register.
    registerActionTarget: undefined,
  },
);

const slots = useSlots();
const { pageTitle, pageTitleId, pageTitleActionsIo } = useHostPageTitleInView(() => props.showHeader);
const showRefreshInOverflow = computed(() => props.showActions && props.showHeader);
const showTitleOverflow = computed(
  () =>
    !!pageTitleActionsIo.value ||
    showRefreshInOverflow.value ||
    !!slots['overflow-actions'],
);

const emit = defineEmits<{
  'card-click': [card: ChoyKanbanCard];
  'card-move': [move: ChoyKanbanMove];
  'lane-load-more': [payload: ChoyKanbanLoadMore];
  create: [];
}>();

const pageStore = useOptionalPageStore<WebModelStore<any>>();
const boundStore = resolvePageStore(props.store ?? pageStore.value, 'ChoyKanbanView');

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
  onLoadError: props.onLoadError,
  onSearchError: props.onSearchError,
  onLoadMoreError: props.onLoadMoreError,
  onMoveError: handleMoveError,
});

useRegisterPageActionTarget({
  store: boundStore,
  enabled: () => props.registerActionTarget,
  target: {
    get selectedItems() {
      // Kanban has no row selection chrome; IO export uses an empty selection.
      return [] as Array<{ Id?: string }>;
    },
    refresh: () => engine.bootstrap(),
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

const boardBusy = computed(() => engine.boardBusy.value);
const effectiveReadonly = computed(() => props.readonly || boardBusy.value);
const resolvedCreateLabel = computed(() => props.createLabel || _t('New'));
const laneCount = computed(() => displayLanes.value.length);
const isFlatBoard = computed(() => isChoyKanbanFlatBoard(displayLanes.value));

const mobileLaneKey = ref('');

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
  emit('card-click', card);
}

function onCreate(): void {
  emit('create');
}

function onRefresh(): void {
  void engine.bootstrap();
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

type DraggableAdded = { element: ChoyKanbanCard; newIndex: number };
type DraggableRemoved = { element: ChoyKanbanCard; oldIndex: number };
type DraggableMoved = { element: ChoyKanbanCard; newIndex: number; oldIndex: number };
type DraggableChangeEvent = {
  added?: DraggableAdded;
  removed?: DraggableRemoved;
  moved?: DraggableMoved;
};

/** Cross-lane moves fire removed then added; persist only from the added side. */
const pendingRemoval = ref<{ cardId: string; fromLaneKey: string } | null>(null);

function onDragChange(laneKey: string, evt: DraggableChangeEvent): void {
  if (effectiveReadonly.value || !evt) return;

  if (evt.removed) {
    pendingRemoval.value = {
      cardId: evt.removed.element.id,
      fromLaneKey: laneKey,
    };
    return;
  }

  if (evt.added) {
    const from =
      pendingRemoval.value?.cardId === evt.added.element.id
        ? pendingRemoval.value.fromLaneKey
        : evt.added.element.laneKey;
    pendingRemoval.value = null;
    // Cards were already moved in-place by vuedraggable; update laneKey and persist.
    evt.added.element.laneKey = laneKey;
    const finalized: ChoyKanbanMove = {
      cardId: evt.added.element.id,
      fromLaneKey: from,
      toLaneKey: laneKey,
      toIndex: evt.added.newIndex,
    };
    emit('card-move', finalized);
    void engine.persistMove(finalized);
    return;
  }

  if (evt.moved) {
    const finalized: ChoyKanbanMove = {
      cardId: evt.moved.element.id,
      fromLaneKey: laneKey,
      toLaneKey: laneKey,
      toIndex: evt.moved.newIndex,
    };
    emit('card-move', finalized);
    void engine.persistMove(finalized);
  }
}

defineExpose({
  bootstrap: () => engine.bootstrap(),
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

.choy-kanban-ghost {
  opacity: 0.45;
}

/*
 * Classic OS/browser scrollbars paint a track gutter that reads as a lane
 * border. Prefer a thin thumb on a transparent track (same idea as Search).
 */
.choy-kanban-view__lane-body,
.choy-kanban-view__board--flat {
  scrollbar-width: thin;
  scrollbar-color: color-mix(in oklab, var(--muted-foreground) 35%, transparent) transparent;
}

.choy-kanban-view__lane-body::-webkit-scrollbar,
.choy-kanban-view__board--flat::-webkit-scrollbar {
  width: 6px;
}

.choy-kanban-view__lane-body::-webkit-scrollbar-track,
.choy-kanban-view__board--flat::-webkit-scrollbar-track {
  background: transparent;
}

.choy-kanban-view__lane-body::-webkit-scrollbar-thumb,
.choy-kanban-view__board--flat::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: color-mix(in oklab, var(--muted-foreground) 35%, transparent);
}
</style>
