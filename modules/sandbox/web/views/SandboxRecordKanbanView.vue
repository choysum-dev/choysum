<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="sandbox-record-kanban-view flex h-full min-h-0 min-w-0 flex-1 flex-col">
    <div class="sr-only" aria-hidden="true">
      <ChoyVirtualField :store="store" prop="Title" />
      <ChoyVirtualField :store="store" prop="Code" />
      <ChoyVirtualField :store="store" prop="State" />
      <ChoyVirtualField :store="store" prop="IsActive" />
      <ChoyVirtualField :store="store" prop="Sequence" />
      <ChoyVirtualField :store="store" prop="CategoryId.Name" />
    </div>

    <ChoyKanbanView
      :store="store"
      :show-header="showHeader"
      :show-actions="true"
      :create-label="_t('New')"
      :flat-lane-label="_t('All')"
      :lane-label="laneLabel"
      :map-row-to-card="rowToCard"
      :resolve-move-record-id="resolveMoveRecordId"
      :on-load-error="onLoadError"
      :on-search-error="onSearchError"
      :on-load-more-error="onLoadMoreError"
      :on-move-error="onMoveError"
      @card-click="onCardClick"
      @create="onCreate"
    >
      <template #search="{ onQueryUpdate }">
        <ChoySearchView
          class="min-w-0 w-full"
          :store="store"
          :default-groups="defaultGroups"
          @query-update="onQueryUpdate"
        />
      </template>

      <template v-if="showHeader" #header-right>
        <ChoyViewSwitcher
          :aria-label="_t('View switcher')"
          :active="activeView"
          :options="viewSwitcherOptions"
          @select="onViewSelect"
        />
      </template>

      <template #card="{ card }">
        <div
          class="flex cursor-grab flex-col gap-1.5 text-xs"
          @dblclick.stop="openDetailFromCard(card)"
        >
          <div class="flex items-center justify-between gap-3">
            <span class="truncate text-sm font-medium text-foreground">{{ cardTitle(card) }}</span>
            <span class="shrink-0 text-foreground/60">{{ cardCode(card) }}</span>
          </div>
          <div class="text-foreground/70">
            {{ _t('Category') }}: {{ cardCategory(card) || '—' }}
          </div>
          <div class="text-foreground/60">
            {{ _t('Seq') }} {{ cardSequence(card) }}
            <span v-if="!cardIsActive(card)" class="ml-2 text-amber-700">{{ _t('Inactive') }}</span>
          </div>
        </div>
      </template>

      <template #card-empty>
        <div class="text-xs opacity-60">{{ _t('No records in this lane') }}</div>
      </template>

      <template #lane-footer="{ remain, loadMore, busy }">
        <ChoyButton
          v-if="remain > 0"
          type="button"
          variant="ghost"
          size="sm"
          class="w-full"
          :disabled="busy"
          @click="loadMore()"
        >
          {{ _t('%s more', remain) }}
        </ChoyButton>
      </template>
    </ChoyKanbanView>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import type SandboxRecord from '@/sandbox/service/models/sandbox_record';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import {
  ChoyButton,
  ChoyKanbanView,
  ChoyMessage,
  ChoySearchView,
  ChoyViewSwitcher,
  ChoyVirtualField,
  useChoyViewRouteSwitch,
  type ChoyKanbanCard,
  type ChoyKanbanLane,
  type ChoyViewSwitcherOption,
} from '@/web';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import type { GroupBySpec, Lane } from '@/web/web/query/types';

defineOptions({ name: 'SandboxRecordKanbanView' });
const { _t } = createTranslate('sandbox', { scope: 'web/views/SandboxRecordKanbanView' });

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<SandboxRecord>;
    showHeader?: boolean;
  }>(),
  { showHeader: true },
);
const store = resolvePageStore(props.store, 'SandboxRecordKanbanView');
const { showHeader } = props;
const { active: activeView, defaultGroups: routeGroups, selectView: onViewSelect } =
  useChoyViewRouteSwitch({ store });

const defaultGroups = routeGroups.value as GroupBySpec<SandboxRecord>[];
const router = useRouter();

const viewSwitcherOptions = computed<ChoyViewSwitcherOption[]>(() => [
  { value: 'list', label: _t('List View') },
  { value: 'kanban', label: _t('Kanban View') },
  { value: 'chart', label: _t('Chart View') },
]);

function laneLabel(lane: Pick<ChoyKanbanLane, 'key' | 'label'> | Lane): string {
  const key = String((lane as Lane).key ?? (lane as ChoyKanbanLane).key ?? '');
  const label = String((lane as Lane).label ?? (lane as ChoyKanbanLane).label ?? '');
  const raw = key || label;
  if (/^State\s*=\s*draft$/i.test(raw) || raw === 'draft') return _t('Draft');
  if (/^State\s*=\s*open$/i.test(raw) || raw === 'open') return _t('Open');
  if (/^State\s*=\s*done$/i.test(raw) || raw === 'done') return _t('Done');
  return label || raw || _t('All');
}

function rowToCard(row: unknown, index: number, laneKey: string): ChoyKanbanCard {
  const payload =
    row && typeof row === 'object' ? (row as Record<string, unknown>) : ({} as Record<string, unknown>);
  const id = String(payload.Id ?? `${laneKey}-${index}`);
  return {
    id,
    title: String(payload.Title || payload.NameChar || payload.Code || id),
    laneKey,
    payload,
  };
}

function resolveMoveRecordId(cards: ReadonlyArray<ChoyKanbanCard>, moveCardId: string): string {
  const hit = cards.find(c => String(c.id) === String(moveCardId));
  const payload = (hit?.payload || {}) as Record<string, unknown>;
  return String(payload.Id ?? moveCardId ?? '').trim();
}

function cardPayload(card: ChoyKanbanCard | Record<string, unknown>): Record<string, unknown> {
  if (card && typeof card === 'object' && 'payload' in card) {
    const p = (card as ChoyKanbanCard).payload;
    return p && typeof p === 'object' ? (p as Record<string, unknown>) : {};
  }
  return card && typeof card === 'object' ? (card as Record<string, unknown>) : {};
}

function cardTitle(card: ChoyKanbanCard): string {
  const r = cardPayload(card);
  return String(r.Title || r.NameChar || r.Code || card.title || '').trim() || _t('Untitled');
}

function cardCode(card: ChoyKanbanCard): string {
  return String(cardPayload(card).Code || '').trim();
}

function cardSequence(card: ChoyKanbanCard): string {
  return String(cardPayload(card).Sequence ?? '');
}

function cardIsActive(card: ChoyKanbanCard): boolean {
  return cardPayload(card).IsActive !== false;
}

function cardCategory(card: ChoyKanbanCard): string {
  const cat = cardPayload(card).CategoryId as { Name?: string } | undefined;
  return String(cat?.Name || '').trim();
}

function openDetailFromCard(card: ChoyKanbanCard) {
  const id = String(card.payload?.Id ?? '').trim();
  if (id) router.push(`/sandbox/records/${id}`);
}

function onCardClick(card: ChoyKanbanCard) {
  openDetailFromCard(card);
}

function onCreate() {
  router.push('/sandbox/records/new');
}

function onLoadError(e: unknown) {
  ChoyMessage.error(_t('Failed to load kanban'));
  console.error('Sandbox kanban load failed:', e);
}

function onSearchError(e: unknown) {
  ChoyMessage.error(_t('Failed to load kanban'));
  console.error('Sandbox kanban search failed:', e);
}

function onLoadMoreError(e: unknown) {
  ChoyMessage.error(_t('Failed to load more'));
  console.error('Sandbox kanban load-more failed:', e);
}

function onMoveError(e: unknown) {
  ChoyMessage.error(_t('Failed to move card'));
  console.error('Sandbox kanban move failed:', e);
}
</script>
