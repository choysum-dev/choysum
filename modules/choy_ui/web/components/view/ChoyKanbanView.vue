<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, ref } from 'vue';
import type { ClassValue } from '../../lib/utils';
import ChoyButton from '../layout/ChoyButton.vue';
import {
  applyChoyKanbanMove,
  type ChoyKanbanCard,
  type ChoyKanbanLane,
  type ChoyKanbanMove,
} from './kanbanViewHelpers';

/**
 * Kanban board chrome: lanes + cards with HTML5 drag-and-drop.
 * Host owns lane state via v-model:lanes (no page store).
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    showHeader?: boolean;
    showActions?: boolean;
    createLabel?: string;
    readonly?: boolean;
  }>(),
  {
    showHeader: true,
    showActions: true,
    createLabel: 'New',
    readonly: false,
  },
);

const lanes = defineModel<ChoyKanbanLane[]>('lanes', { default: () => [] });

const emit = defineEmits<{
  'card-click': [card: ChoyKanbanCard];
  'card-move': [move: ChoyKanbanMove];
  create: [];
}>();

const dragCardId = ref<string | null>(null);
const dragFromLane = ref<string | null>(null);

const laneCount = computed(() => lanes.value.length);

function onCardClick(card: ChoyKanbanCard): void {
  emit('card-click', card);
}

function onDragStart(card: ChoyKanbanCard, event: DragEvent): void {
  if (props.readonly) {
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
  if (props.readonly || !dragCardId.value) return;
  event.preventDefault();
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'move';
  }
}

function dropOnLane(toLaneKey: string, toIndex: number, event: DragEvent): void {
  event.preventDefault();
  if (props.readonly) return;
  const cardId = dragCardId.value || event.dataTransfer?.getData('text/plain');
  const fromLaneKey = dragFromLane.value;
  if (!cardId || !fromLaneKey) return;

  const move: ChoyKanbanMove = { cardId, fromLaneKey, toLaneKey, toIndex };
  const next = applyChoyKanbanMove(lanes.value, move);
  if (!next) return;
  lanes.value = next;
  emit('card-move', move);
  onDragEnd();
}

function onCreate(): void {
  emit('create');
}
</script>

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
            <ChoyButton size="sm" @click="onCreate">{{ createLabel }}</ChoyButton>
          </slot>
          <slot name="user-actions" />
        </div>
        <div v-if="$slots.header" class="min-w-0">
          <slot name="header" />
        </div>
      </div>
      <div v-if="$slots.search" class="shrink-0">
        <slot name="search" />
      </div>
    </div>

    <div
      class="choy-kanban-view__board flex gap-3 overflow-x-auto pb-1"
      :style="{ minHeight: '12rem' }"
    >
      <div
        v-for="lane in lanes"
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
            :draggable="!readonly"
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
