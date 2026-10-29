<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    ref="scrollRef"
    class="choy-kanban-view__lane-body flex min-h-0 flex-1 flex-col overflow-y-auto px-2 pb-2"
    data-testid="choy-kanban-virtual-lane-body"
    :data-virtualized="virtualized ? 'true' : 'false'"
  >
    <div
      v-if="virtualized"
      class="relative w-full shrink-0"
      data-testid="choy-kanban-virtual-spacer"
      :style="{ height: `${totalSize}px` }"
    >
      <article
        v-for="item in virtualItems"
        :key="String(item.key)"
        :ref="measureElement"
        :data-index="item.index"
        class="choy-kanban-view__card absolute left-0 top-0 w-full rounded-md border border-border bg-card p-3 shadow-xs outline-none transition-colors hover:bg-accent/40 focus-visible:ring-2 focus-visible:ring-ring"
        :class="laneDrag ? 'cursor-grab active:cursor-grabbing' : 'cursor-pointer'"
        :style="{ transform: `translateY(${item.start}px)` }"
        :data-card-id="cards[item.index]?.id"
        :draggable="laneDrag"
        tabindex="0"
        role="article"
        :aria-label="cards[item.index]?.title"
        @click="onCardClick(cards[item.index])"
        @keydown.enter.self.prevent="onCardClick(cards[item.index])"
        @dragstart="onCardDragStart($event, cards[item.index])"
      >
        <div class="flex items-start gap-1">
          <div class="min-w-0 flex-1">
            <slot name="card" :card="cards[item.index]!" :lane="lane">
              <div class="text-sm font-medium text-foreground">{{ cards[item.index]!.title }}</div>
              <div v-if="cards[item.index]!.subtitle" class="mt-1 text-xs text-muted-foreground">
                {{ cards[item.index]!.subtitle }}
              </div>
            </slot>
          </div>
          <DropdownMenu v-if="laneDrag && otherLanes.length">
            <DropdownMenuTrigger as-child>
              <ChoyButton
                type="button"
                variant="ghost"
                size="sm"
                class="h-7 w-7 shrink-0 px-0"
                data-testid="choy-kanban-move-to-lane"
                :aria-label="moveToLaneLabel"
                @click.stop
              >
                <MoreHorizontal class="size-4" aria-hidden="true" />
              </ChoyButton>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" class="min-w-[10rem]">
              <DropdownMenuItem
                v-for="target in otherLanes"
                :key="target.key"
                @select="emit('move-to-lane', cards[item.index]!, target.key)"
              >
                {{ moveToLabel(target.label) }}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </article>
    </div>

    <div v-else class="flex shrink-0 flex-col gap-2">
      <article
        v-for="card in cards"
        :key="card.id"
        class="choy-kanban-view__card rounded-md border border-border bg-card p-3 shadow-xs outline-none transition-colors hover:bg-accent/40 focus-visible:ring-2 focus-visible:ring-ring"
        :class="laneDrag ? 'cursor-grab active:cursor-grabbing' : 'cursor-pointer'"
        :data-card-id="card.id"
        :draggable="laneDrag"
        tabindex="0"
        role="article"
        :aria-label="card.title"
        @click="onCardClick(card)"
        @keydown.enter.self.prevent="onCardClick(card)"
        @dragstart="onCardDragStart($event, card)"
      >
        <div class="flex items-start gap-1">
          <div class="min-w-0 flex-1">
            <slot name="card" :card="card" :lane="lane">
              <div class="text-sm font-medium text-foreground">{{ card.title }}</div>
              <div v-if="card.subtitle" class="mt-1 text-xs text-muted-foreground">{{ card.subtitle }}</div>
            </slot>
          </div>
          <DropdownMenu v-if="laneDrag && otherLanes.length">
            <DropdownMenuTrigger as-child>
              <ChoyButton
                type="button"
                variant="ghost"
                size="sm"
                class="h-7 w-7 shrink-0 px-0"
                data-testid="choy-kanban-move-to-lane"
                :aria-label="moveToLaneLabel"
                @click.stop
              >
                <MoreHorizontal class="size-4" aria-hidden="true" />
              </ChoyButton>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" class="min-w-[10rem]">
              <DropdownMenuItem
                v-for="target in otherLanes"
                :key="target.key"
                @select="emit('move-to-lane', card, target.key)"
              >
                {{ moveToLabel(target.label) }}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </article>
    </div>

    <div
      v-if="cards.length === 0"
      class="shrink-0 rounded-md border border-dashed border-border/70 px-3 py-6 text-center text-xs text-muted-foreground"
    >
      <slot name="card-empty" :lane="lane">{{ emptyLabel }}</slot>
    </div>

    <ChoyKanbanLoadSentinel
      v-if="remain > 0"
      :disabled="loadMoreBusy"
      @visible="emit('load-more')"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useVirtualizer } from '@tanstack/vue-virtual';
import { MoreHorizontal } from 'lucide-vue-next';
import { setHtml5DragImage } from '@/web/web/composables/useListHandleReorder';
import ChoyButton from '../layout/ChoyButton.vue';
import DropdownMenu from '../vendor/ui/dropdown-menu/DropdownMenu.vue';
import DropdownMenuContent from '../vendor/ui/dropdown-menu/DropdownMenuContent.vue';
import DropdownMenuItem from '../vendor/ui/dropdown-menu/DropdownMenuItem.vue';
import DropdownMenuTrigger from '../vendor/ui/dropdown-menu/DropdownMenuTrigger.vue';
import ChoyKanbanLoadSentinel from './ChoyKanbanLoadSentinel.vue';
import {
  CHOY_KANBAN_CARD_GAP,
  CHOY_KANBAN_LANE_DRAG_MIME,
  choyKanbanShouldVirtualize,
  encodeChoyKanbanLaneDragPayload,
  resolveChoyKanbanEstimateCardSize,
  type ChoyKanbanVirtualizeMode,
} from './kanbanVirtualHelpers';
import type { ChoyKanbanCard, ChoyKanbanLane } from './kanbanViewHelpers';

const props = withDefaults(
  defineProps<{
    cards: ChoyKanbanCard[];
    lane: ChoyKanbanLane;
    virtualize?: ChoyKanbanVirtualizeMode;
    estimateCardSize?: number;
    remain?: number;
    loadMoreBusy?: boolean;
    /** Enable HTML5 drag for whole-lane drop targets. */
    laneDrag?: boolean;
    otherLanes?: ReadonlyArray<Pick<ChoyKanbanLane, 'key' | 'label'>>;
    emptyLabel?: string;
    moveToLaneLabel?: string;
    moveToLabelFn?: (label: string) => string;
  }>(),
  {
    virtualize: 'auto',
    remain: 0,
    loadMoreBusy: false,
    laneDrag: false,
    otherLanes: () => [],
    emptyLabel: 'No cards',
    moveToLaneLabel: 'Move to lane',
  },
);

const emit = defineEmits<{
  'card-click': [card: ChoyKanbanCard];
  'load-more': [];
  'move-to-lane': [card: ChoyKanbanCard, toLaneKey: string];
}>();

const scrollRef = ref<HTMLElement | null>(null);

const virtualized = computed(() =>
  choyKanbanShouldVirtualize(props.virtualize, props.cards.length),
);

const estimateSize = computed(() => resolveChoyKanbanEstimateCardSize(props.estimateCardSize));

const virtualizer = useVirtualizer({
  get count() {
    return virtualized.value ? props.cards.length : 0;
  },
  getScrollElement: () => (virtualized.value ? (scrollRef.value as Element | null) : null),
  estimateSize: () => estimateSize.value,
  // Same rhythm as the non-virtual `gap-2` column so dense/sparse lanes match.
  gap: CHOY_KANBAN_CARD_GAP,
  getItemKey: (index) => props.cards[index]?.id ?? index,
  overscan: 8,
});

const virtualItems = computed(() => virtualizer.value.getVirtualItems());
const totalSize = computed(() => virtualizer.value.getTotalSize());

function measureElement(el: Element | null): void {
  if (el && virtualized.value) {
    virtualizer.value.measureElement(el);
  }
}

function onCardClick(card: ChoyKanbanCard | undefined): void {
  if (card) emit('card-click', card);
}

function onCardDragStart(event: DragEvent, card: ChoyKanbanCard | undefined): void {
  if (!card || !props.laneDrag || !event.dataTransfer) return;
  const encoded = encodeChoyKanbanLaneDragPayload({
    cardId: card.id,
    fromLaneKey: props.lane.key,
  });
  if (!encoded) {
    event.preventDefault();
    return;
  }
  event.dataTransfer.setData(CHOY_KANBAN_LANE_DRAG_MIME, encoded);
  event.dataTransfer.setData('text/plain', card.id);
  event.dataTransfer.effectAllowed = 'move';
  // Card node itself: off-screen ghosts often fail in Chromium; browser snapshots at dragstart.
  const el = event.currentTarget;
  if (el instanceof HTMLElement) setHtml5DragImage(el, event);
}

function moveToLabel(label: string): string {
  if (props.moveToLabelFn) return props.moveToLabelFn(label);
  return `Move to ${label}`;
}

defineExpose({
  /** Test hook: run the virtualizer measure callback. */
  measureElement,
});
</script>
