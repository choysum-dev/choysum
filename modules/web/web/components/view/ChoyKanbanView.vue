<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ViewContainer
    :show-header="showHeader"
    data-anchor="choy.kanban-view"
    :class="['choy-kanban-view', props.class]"
  >
    <template v-if="showHeader && ($slots.header || $slots.search || showActions)" #header>
      <div
        class="choy-kanban-view__toolbar grid grid-cols-[auto_1fr_auto] items-center gap-3 border-b border-border pb-1 min-h-control max-md:grid-cols-1"
        data-anchor="choy.kanban.view-chrome"
      >
        <div class="choy-kanban-view__actions flex items-center gap-4">
          <ChoyActionTray
            v-if="showActions"
            class="choy-kanban__system-actions"
            :aria-label="_t('System actions')"
          >
            <slot name="system-actions">
              <ChoyButton
                v-if="!effectiveReadonly"
                size="sm"
                @click="onCreate"
              >
                {{ resolvedCreateLabel }}
              </ChoyButton>
            </slot>
          </ChoyActionTray>
          <ChoyActionTray
            v-if="showActions"
            class="choy-kanban__user-actions"
            :aria-label="_t('User actions')"
          >
            <slot name="user-actions" />
          </ChoyActionTray>
          <div v-if="$slots.header" class="min-w-0">
            <slot name="header" />
          </div>
        </div>

        <div
          v-if="$slots.search"
          class="choy-kanban-view__search flex min-w-60 items-center justify-center max-md:order-2"
        >
          <slot name="search" :on-query-update="engine.applySearch" />
        </div>

        <div class="choy-kanban-view__header-right" />
      </div>
    </template>

    <div class="choy-kanban-view__body flex min-h-0 flex-1 flex-col gap-3 p-1">
      <div v-if="laneCount > 0" class="md:hidden">
        <Select :model-value="mobileLaneKey" @update:model-value="onMobileLaneChange">
          <SelectTrigger
            class="w-full"
            data-testid="choy-kanban-mobile-lane"
            :placeholder="_t('Select lane')"
            :aria-label="_t('Select lane')"
          />
          <SelectContent>
            <SelectItem v-for="lane in displayLanes" :key="lane.key" :value="lane.key">
              {{ lane.label }} ({{ lane.cards.length }})
            </SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div
        class="choy-kanban-view__board flex gap-3 overflow-x-auto pb-1"
        :style="{ minHeight: '12rem' }"
        role="list"
        :aria-label="_t('Kanban board')"
      >
        <section
          v-for="lane in displayLanes"
          :key="lane.key"
          class="choy-kanban-view__lane flex w-64 shrink-0 flex-col rounded-md border border-border bg-muted/30"
          role="listitem"
          :aria-label="lane.label"
          :data-mobile-active="lane.key === mobileLaneKey ? 'true' : 'false'"
        >
          <header class="flex items-center justify-between gap-2 border-b border-border px-3 py-2">
            <slot name="lane-header" :lane="lane">
              <span class="text-sm font-medium text-foreground">{{ lane.label }}</span>
              <span class="text-xs text-foreground/60">({{ lane.cards.length }})</span>
            </slot>
          </header>

          <div
            v-if="readonly"
            class="choy-kanban-view__lane-body flex flex-1 flex-col gap-2 p-2"
            data-testid="choy-kanban-lane-static"
            :style="{ minHeight: '8rem' }"
          >
            <article
              v-for="card in lane.cards"
              :key="card.id"
              class="choy-kanban-view__card cursor-pointer rounded-md border border-border bg-background p-3 shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
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
          </div>
          <draggable
            v-else
            class="choy-kanban-view__lane-body flex flex-1 flex-col gap-2 p-2"
            :list="lane.cards"
            item-key="id"
            group="choy-kanban"
            :animation="150"
            :disabled="effectiveReadonly"
            ghost-class="choy-kanban-ghost"
            :style="{ minHeight: '8rem' }"
            @change="(evt: DraggableChangeEvent) => onDragChange(lane.key, evt)"
          >
            <template #item="{ element: card }">
              <article
                class="choy-kanban-view__card rounded-md border border-border bg-background p-3 shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
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
            class="mx-2 mb-2 rounded-md border border-dashed border-border px-3 py-6 text-center text-xs text-foreground/50"
          >
            <slot name="card-empty" :lane="lane">{{ _t('No cards') }}</slot>
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
                {{ formatChoyKanbanLoadMoreLabel(choyKanbanLaneRemain(lane), (n) => _t('Load more (%s remaining)', n)) }}
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
    </div>
  </ViewContainer>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, ref, watch, type PropType } from 'vue';
import draggable from 'vuedraggable';
import { MoreHorizontal } from 'lucide-vue-next';
import type { ClassValue } from '../../lib/utils';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { Lane } from '@/web/web/query/types';
import ChoyButton from '../layout/ChoyButton.vue';
import ChoyActionTray from '@/web/web/components/layout/ChoyActionTray.vue';
import ViewContainer from '@/web/web/components/view/ViewContainer.vue';
import DropdownMenu from '../vendor/ui/dropdown-menu/DropdownMenu.vue';
import DropdownMenuContent from '../vendor/ui/dropdown-menu/DropdownMenuContent.vue';
import DropdownMenuItem from '../vendor/ui/dropdown-menu/DropdownMenuItem.vue';
import DropdownMenuTrigger from '../vendor/ui/dropdown-menu/DropdownMenuTrigger.vue';
import Select from '../vendor/ui/select/Select.vue';
import SelectContent from '../vendor/ui/select/SelectContent.vue';
import SelectItem from '../vendor/ui/select/SelectItem.vue';
import SelectTrigger from '../vendor/ui/select/SelectTrigger.vue';
import { resolvePageStore, useOptionalPageStore } from '@/web/web/composables/usePageContext';
import { useChoyKanbanStoreEngine } from '@/web/web/composables/useChoyKanbanStoreEngine';
import { ChoyMessage } from '../../composables/useChoyMessage';
import { createTranslate } from '@/web/web/i18n';
import {
  applyChoyKanbanMove,
  choyKanbanLaneRemain,
  formatChoyKanbanLoadMoreLabel,
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
  }>(),
  {
    showHeader: true,
    showActions: true,
    createLabel: undefined,
    readonly: false,
    autoBootstrap: true,
  },
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
</style>
