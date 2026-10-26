<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    ref="boardRef"
    class="choy-o2m-kanban-virtual-board w-full min-w-0 overflow-auto p-0.5"
    data-testid="choy-o2m-kanban-board"
    :data-virtualized="virtualized ? 'true' : 'false'"
    :style="boardRootStyle"
  >
    <template v-if="virtualized">
      <div
        class="relative w-full"
        data-testid="choy-o2m-kanban-virtual-spacer"
        :style="{ height: `${totalSize}px` }"
      >
        <div
          v-for="row in virtualRows"
          :key="String(row.key)"
          :ref="measureElement"
          :data-index="row.index"
          class="absolute left-0 grid w-full"
          :style="{
            transform: `translateY(${row.start}px)`,
            gap: `${gapPx}px`,
            gridTemplateColumns: `repeat(${cols}, minmax(0, 1fr))`,
          }"
        >
          <div
            v-for="cardIndex in o2mKanbanRowCardIndices(row.index, cols, items.length)"
            :key="cardKey(cardIndex)"
          >
            <slot name="card" :item="items[cardIndex]" :index="cardIndex" />
          </div>
        </div>
      </div>
      <div v-if="$slots.trailing" class="mt-[var(--kanban-gap,10px)]">
        <slot name="trailing" />
      </div>
    </template>

    <div
      v-else
      class="grid w-full min-w-0 [grid-template-columns:repeat(auto-fill,minmax(var(--kanban-min-card-width),1fr))] gap-[var(--kanban-gap)]"
    >
      <template v-if="items.length > 0">
        <div v-for="(item, index) in items" :key="cardKey(index)">
          <slot name="card" :item="item" :index="index" />
        </div>
      </template>
      <slot v-else name="empty" />
      <slot name="trailing" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useVirtualizer } from '@tanstack/vue-virtual';
import {
  o2mKanbanColumnCount,
  o2mKanbanRowCardIndices,
  o2mKanbanRowCount,
  o2mKanbanShouldVirtualize,
  resolveO2MKanbanEstimateCardSize,
  type O2MKanbanVirtualizeMode,
} from './o2mKanbanVirtualHelpers';

const props = withDefaults(
  defineProps<{
    items: readonly unknown[];
    minCardWidth?: number;
    gap?: number;
    maxHeight?: number;
    virtualize?: O2MKanbanVirtualizeMode;
    estimateCardSize?: number;
    /** Stable key for a card index (defaults to index). */
    itemKey?: (item: unknown, index: number) => string | number;
  }>(),
  {
    minCardWidth: 220,
    gap: 10,
    maxHeight: 480,
    virtualize: 'auto',
    estimateCardSize: undefined,
  },
);

const boardRef = ref<HTMLElement | null>(null);
const boardWidth = ref(0);
let resizeObserver: ResizeObserver | null = null;

const gapPx = computed(() => Math.max(0, Number(props.gap) || 0));
const minCardWidthPx = computed(() => Math.max(1, Number(props.minCardWidth) || 220));

const cols = computed(() =>
  o2mKanbanColumnCount({
    boardWidth: boardWidth.value,
    minCardWidth: minCardWidthPx.value,
    gap: gapPx.value,
  }),
);

const virtualized = computed(() =>
  o2mKanbanShouldVirtualize(props.virtualize, props.items.length),
);

const estimateSize = computed(() => {
  const card = resolveO2MKanbanEstimateCardSize(props.estimateCardSize);
  return card + gapPx.value;
});

const rowCount = computed(() => o2mKanbanRowCount(props.items.length, cols.value));

const virtualizer = useVirtualizer({
  get count() {
    return virtualized.value ? rowCount.value : 0;
  },
  getScrollElement: () => (virtualized.value ? (boardRef.value as Element | null) : null),
  estimateSize: () => estimateSize.value,
  getItemKey: (index) => `row-${index}-${cols.value}`,
  overscan: 4,
});

const virtualRows = computed(() => virtualizer.value.getVirtualItems());
const totalSize = computed(() => virtualizer.value.getTotalSize());

const boardRootStyle = computed(() => ({
  '--kanban-min-card-width': `${minCardWidthPx.value}px`,
  '--kanban-gap': `${gapPx.value}px`,
  maxHeight: `${props.maxHeight}px`,
}));

function cardKey(index: number): string {
  const item = props.items[index];
  if (props.itemKey) return String(props.itemKey(item, index));
  return String(index);
}

function measureElement(el: Element | null): void {
  if (el && virtualized.value) {
    virtualizer.value.measureElement(el);
  }
}

function syncBoardWidth(): void {
  const el = boardRef.value;
  boardWidth.value = el ? el.clientWidth : 0;
}

onMounted(() => {
  syncBoardWidth();
  const el = boardRef.value;
  if (typeof ResizeObserver !== 'undefined' && el) {
    resizeObserver = new ResizeObserver(() => syncBoardWidth());
    resizeObserver.observe(el);
  }
});

onBeforeUnmount(() => {
  resizeObserver?.disconnect();
  resizeObserver = null;
});

watch(
  () => [props.items.length, cols.value, virtualized.value] as const,
  () => {
    if (virtualized.value) {
      virtualizer.value.measure();
    }
  },
);

defineExpose({
  measureElement,
  cols,
  virtualized,
});
</script>
