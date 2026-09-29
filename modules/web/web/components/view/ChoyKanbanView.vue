<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    data-anchor="choy.kanban-view"
    :class="['choy-kanban-view flex w-full flex-col gap-3', props.class]"
  >
    <div
      v-if="showHeader && ($slots.header || $slots.search || showActions)"
      class="choy-kanban-view__toolbar flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between"
    >
      <div class="flex min-w-0 flex-1 flex-wrap items-center gap-2">
        <div v-if="showActions" class="flex flex-wrap gap-2">
          <slot name="system-actions">
            <ChoyButton v-if="!effectiveReadonly" size="sm" @click="onCreate">{{ createLabel }}</ChoyButton>
          </slot>
          <slot name="user-actions" />
        </div>
        <div v-if="$slots.header" class="min-w-0">
          <slot name="header" />
        </div>
      </div>
      <div v-if="$slots.search" class="shrink-0">
        <slot name="search" :on-query-update="engine.applySearch" />
      </div>
    </div>

    <div
      class="choy-kanban-view__board flex gap-3 overflow-x-auto pb-1"
      :style="{ minHeight: '12rem' }"
    >
      <div
        v-for="lane in displayLanes"
        :key="lane.key"
        class="choy-kanban-view__lane flex w-64 shrink-0 flex-col rounded-md border border-border bg-muted/30"
        @dragover="onLaneDragOver"
        @drop="dropOnLane(lane.key, lane.cards.length, $event)"
      >
        <div class="flex items-center justify-between gap-2 border-b border-border px-3 py-2">
          <slot name="lane-header" :lane="lane">
            <span class="text-sm font-medium text-foreground">{{ lane.label }}</span>
            <span class="text-xs text-foreground/60">({{ lane.cards.length }})</span>
          </slot>
        </div>
        <div class="flex flex-1 flex-col gap-2 p-2">
          <div
            v-for="(card, index) in lane.cards"
            :key="card.id"
            class="choy-kanban-view__card cursor-pointer rounded-md border border-border bg-background p-3 shadow-sm"
            :class="{ 'opacity-60': dragCardId === card.id }"
            :draggable="!effectiveReadonly"
            :data-card-id="card.id"
            @click="onCardClick(card)"
            @dragstart="onDragStart(card, $event)"
            @dragend="onDragEnd"
            @dragover="onLaneDragOver"
            @drop.stop="dropOnLane(lane.key, index, $event)"
          >
            <slot name="card" :card="card" :lane="lane">
              <div class="text-sm font-medium text-foreground">{{ card.title }}</div>
              <div v-if="card.subtitle" class="mt-1 text-xs text-foreground/60">
                {{ card.subtitle }}
              </div>
            </slot>
          </div>
          <div
            v-if="lane.cards.length === 0"
            class="rounded-md border border-dashed border-border px-3 py-6 text-center text-xs text-foreground/50"
          >
            <slot name="card-empty" :lane="lane">No cards</slot>
          </div>
          <div
            v-if="$slots['lane-footer'] || choyKanbanLaneRemain(lane) > 0"
            class="choy-kanban-view__lane-footer pt-1"
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
                {{ formatChoyKanbanLoadMoreLabel(choyKanbanLaneRemain(lane)) }}
              </ChoyButton>
            </slot>
          </div>
        </div>
      </div>
      <div
        v-if="laneCount === 0"
        class="flex w-full items-center justify-center rounded-md border border-dashed border-border py-12 text-sm text-foreground/50"
      >
        No lanes
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import type { ClassValue } from '../../lib/utils';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { Lane } from '@/web/web/query/types';
import ChoyButton from '../layout/ChoyButton.vue';
import { resolvePageStore, useOptionalPageStore } from '@/web/web/composables/usePageContext';
import { useChoyKanbanStoreEngine } from '@/web/web/composables/useChoyKanbanStoreEngine';
import {
  applyChoyKanbanMove,
  choyKanbanLaneRemain,
  formatChoyKanbanLoadMoreLabel,
  type ChoyKanbanCard,
  type ChoyKanbanLane,
  type ChoyKanbanLoadMore,
  type ChoyKanbanMove
} from './kanbanViewHelpers';

/**
 * Store-bound kanban board. Requires :store or a page-provided store.
 * Owns createKanbanController, lane sync, search apply, load-more, and move persist.
 */
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
    createLabel: 'New',
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
  onMoveError: props.onMoveError,
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

const dragCardId = ref<string | null>(null);
const dragFromLane = ref<string | null>(null);

const laneCount = computed(() => displayLanes.value.length);

function onCardClick(card: ChoyKanbanCard): void {
  emit('card-click', card);
}

function onDragStart(card: ChoyKanbanCard, event: DragEvent): void {
  if (effectiveReadonly.value) {
    event.preventDefault();
    return;
  }
  dragCardId.value = card.id;
  dragFromLane.value = card.laneKey;
  event.dataTransfer?.setData('text/plain', card.id);
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move';
  }
}

function onDragEnd(): void {
  dragCardId.value = null;
  dragFromLane.value = null;
}

function onLaneDragOver(event: DragEvent): void {
  if (effectiveReadonly.value || !dragCardId.value) return;
  event.preventDefault();
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'move';
  }
}

function dropOnLane(toLaneKey: string, toIndex: number, event: DragEvent): void {
  event.preventDefault();
  if (effectiveReadonly.value) return;
  const cardId = dragCardId.value || event.dataTransfer?.getData('text/plain');
  const fromLaneKey = dragFromLane.value;
  if (!cardId || !fromLaneKey) return;

  const move: ChoyKanbanMove = { cardId, fromLaneKey, toLaneKey, toIndex };
  const next = applyChoyKanbanMove(displayLanes.value, move);
  if (!next) return;
  displayLanes.value = next;
  const finalLane = next.find((l) => l.key === toLaneKey);
  const finalIndex = finalLane ? finalLane.cards.findIndex((c) => c.id === cardId) : toIndex;
  const finalized: ChoyKanbanMove = {
    ...move,
    toIndex: finalIndex < 0 ? toIndex : finalIndex,
  };
  emit('card-move', finalized);
  void engine.persistMove(finalized);
  onDragEnd();
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

defineExpose({
  bootstrap: () => engine.bootstrap(),
  applySearch: engine.applySearch,
  syncLanes: () => engine.syncLanesFromController(),
  boardBusy,
});
</script>
