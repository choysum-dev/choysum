<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="token-kanban-view">
    <div class="sr-only" aria-hidden="true">
      <ChoyVirtualField :store="store" prop="TokenType" />
      <ChoyVirtualField :store="store" prop="ExpiresAt" />
      <ChoyVirtualField :store="store" prop="Revoked" />
      <ChoyVirtualField :store="store" prop="RevokedAt" />
      <ChoyVirtualField :store="store" prop="UserId.Username" />
      <ChoyVirtualField :store="store" prop="CreatedAt" />
    </div>

    <ChoyKanbanView
      v-model:lanes="choyLanes"
      :show-header="showHeader"
      :show-actions="true"
      :readonly="movePending || loadMorePending"
      :create-label="_t('New')"
      @card-click="onCardClick"
      @card-move="onCardMove"
      @lane-load-more="onLaneLoadMore"
      @create="onCreate"
    >
      <template #search>
        <ChoySearchView :store="store" @query-update="onSearch" />
      </template>

      <template #user-actions>
        <div class="flex items-center gap-1">
          <ChoyButton variant="outline" size="sm" :title="_t('List View')" @click="toList">
            <List class="size-4" aria-hidden="true" />
          </ChoyButton>
          <ChoyButton size="sm" :title="_t('Kanban View')" @click="toKanban">
            <LayoutGrid class="size-4" aria-hidden="true" />
          </ChoyButton>
          <ChoyButton variant="outline" size="sm" :title="_t('Icon View')" @click="toKanban">
            <BarChart3 class="size-4" aria-hidden="true" />
          </ChoyButton>
        </div>
      </template>

      <template #lane-header="{ lane }">
        <div class="token-lane-header flex items-center gap-1.5 text-sm font-semibold text-foreground">
          <span class="title">{{ laneLabel(lane) }}</span>
          <span class="count text-foreground/60">({{ lane.cards.length }})</span>
        </div>
      </template>

      <template #card="{ card }">
        <div
          class="token-card flex flex-col gap-1.5 text-xs"
          :class="{ revoked: isRevokedCard(card) }"
          @dblclick.stop="openDetailFromCard(card)"
        >
          <div class="top-row flex items-center justify-between gap-3">
            <span class="token-type rounded px-1.5 py-0.5 text-[11px] font-semibold uppercase tracking-wide" :class="tokenTypeClass(card)">
              {{ tokenTypeLabel(card) }}
            </span>
            <span class="user text-foreground/70">{{ usernameLabel(card) }}</span>
          </div>
          <div class="expires text-foreground/80" :title="formatDate(expiresAt(card))">
            {{ _t('Expires') }}: {{ formatDate(expiresAt(card)) }}
          </div>
          <div v-if="isRevokedCard(card)" class="revoked-info text-amber-700">
            {{ _t('Revoked At') }}: {{ formatDate(revokedAt(card)) || '—' }}
          </div>
        </div>
      </template>

      <template #card-empty>
        <div class="empty-lane text-xs opacity-60">{{ _t('No tokens in this lane') }}</div>
      </template>

      <template #lane-footer="{ remain, loadMore }">
        <ChoyButton
          v-if="remain > 0"
          type="button"
          variant="ghost"
          size="sm"
          class="w-full"
          :disabled="loadMorePending"
          @click="loadMore()"
        >
          {{ _t('Load more (%s remaining)', remain) }}
        </ChoyButton>
      </template>
    </ChoyKanbanView>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { BarChart3, LayoutGrid, List } from 'lucide-vue-next';
import type Token from '@/auth/service/models/token';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import {
  ChoyButton,
  ChoyKanbanView,
  ChoyMessage,
  ChoySearchView,
  ChoyVirtualField,
  type ChoyKanbanCard,
  type ChoyKanbanLane,
  type ChoyKanbanLoadMore,
  type ChoyKanbanMove,
} from '@/web';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { createKanbanController } from '@/web/web/controllers/kanbanController';
import { awaitFieldSelection } from '@/web/web/query/utils/registry/fieldReady';
import type { ChoySearchQuery } from '@/web/web/components/view/searchViewHelpers';
import type { Lane } from '@/web/web/query/types';
import {
  resolveTokenDetailId,
  resolveTokenKanbanCardId,
  resolveTokenKanbanRowPayload,
  resolveTokenMoveRecordId,
  resolveTokenUsernameLabel,
  type TokenKanbanRow,
} from './token_kanban_nav';

defineOptions({ name: 'TokenKanbanView' });
const { _t } = createTranslate('auth', { scope: 'web/views/TokenKanbanView' });

const props = withDefaults(defineProps<{ store?: WebModelStore<Token>; showHeader?: boolean }>(), { showHeader: true });
const store = resolvePageStore(props.store, 'TokenKanbanView');
const { showHeader } = props;

const router = useRouter();
const controller = createKanbanController(store as any);
const choyLanes = ref<ChoyKanbanLane[]>([]);
const movePending = ref(false);
const loadMorePending = ref(false);
let syncingLanes = false;
let resyncPending = false;
let searchSeq = 0;
let lastSearchQuery: ChoySearchQuery | null = null;

function rowToCard(row: TokenKanbanRow, index: number, laneKey: string): ChoyKanbanCard {
  const payload = resolveTokenKanbanRowPayload(row);
  return {
    id: resolveTokenKanbanCardId(row, index, laneKey),
    title: String(payload.TokenType ?? payload.Id ?? ''),
    laneKey,
    payload,
  };
}

/**
 * Map controller lane rows into ChoyKanbanView lane/card models.
 * When no group is applied, surface a single flat lane from search results.
 */
async function syncLanesFromController(): Promise<void> {
  if (syncingLanes) {
    resyncPending = true;
    return;
  }
  syncingLanes = true;
  try {
    do {
      resyncPending = false;
      const laneList = controller.lanes.value;
      if (!laneList.length) {
        const rows =
          controller.vm.result?.kind === 'search' ? ((controller.vm.result.rows as any[]) || []) : [];
        choyLanes.value = [
          {
            key: 'all',
            label: _t('All'),
            cards: rows.map((row, index) => rowToCard(row, index, 'all')),
          },
        ];
        continue;
      }
      await Promise.all(laneList.map(l => controller.preloadLane(l.key).catch(() => undefined)));
      choyLanes.value = laneList.map(lane => ({
        key: lane.key,
        label: laneLabel(lane),
        remain: controller.getLaneRemain(lane),
        cards: (controller.laneRecords.value[lane.key] || []).map((row, index) =>
          rowToCard(row as any, index, lane.key)
        ),
      }));
    } while (resyncPending);
  } finally {
    syncingLanes = false;
  }
}

watch(
  () => controller.lanes.value,
  () => {
    void syncLanesFromController();
  },
  { deep: true }
);

/**
 * Re-apply the last search payload (or current store query) after a failed move.
 */
async function applyCurrentQuery(): Promise<void> {
  if (lastSearchQuery) {
    await controller.apply({
      keyword: lastSearchQuery.keyword,
      appliedFilters: (lastSearchQuery.appliedFilters || []) as any,
      appliedGroups: lastSearchQuery.appliedGroups as any,
    });
    return;
  }
  const qs = ((store.state as any)?.queryState ?? {}) as Record<string, unknown>;
  await controller.apply({
    keyword: qs.keyword as string | undefined,
    appliedFilters: (qs.appliedFilters || []) as any,
    appliedGroups: qs.appliedGroups as any,
  });
}

onMounted(async () => {
  try {
    await awaitFieldSelection(store, { requireNonEmpty: true });
    // Prefer the search view's first-frame emit when it arrives; otherwise load once.
    if (!lastSearchQuery) {
      await controller.apply({});
      await syncLanesFromController();
    }
  } catch (e) {
    ChoyMessage.error(_t('Failed to load kanban'));
    console.error('Token kanban load failed:', e);
  }
});

/**
 * Apply the emitted search query from ChoySearchView (store-bound payload).
 */
async function onSearch(query: ChoySearchQuery) {
  lastSearchQuery = query;
  const seq = ++searchSeq;
  try {
    await controller.apply({
      keyword: query.keyword,
      appliedFilters: (query.appliedFilters || []) as any,
      appliedGroups: query.appliedGroups as any,
    });
    if (seq !== searchSeq) return;
    await syncLanesFromController();
  } catch (e) {
    if (seq !== searchSeq) return;
    ChoyMessage.error(_t('Failed to load kanban'));
    console.error('Token kanban search failed:', e);
  }
}

/**
 * Navigate from kanban view back to the token list.
 */
function toList() {
  router.push('/auth/tokens');
}

/**
 * Keep navigation on the token kanban route.
 */
function toKanban() {
  router.push('/auth/tokens/kanban');
}

function onCreate() {
  router.push('/auth/tokens/new');
}

function openDetailFromCard(card: ChoyKanbanCard) {
  // `rowToCard` may fall back to row key/index for Vue keys; only route on a real record Id.
  const id = resolveTokenDetailId(card.payload as Record<string, unknown> | undefined);
  if (id) router.push(`/auth/tokens/${id}`);
}

/**
 * Handle kanban card clicks by opening the record detail view.
 */
function onCardClick(card: ChoyKanbanCard) {
  openDetailFromCard(card);
}

/**
 * Fetch the next batch for a lane, then remap cards / remain counts.
 */
async function onLaneLoadMore(payload: ChoyKanbanLoadMore) {
  if (loadMorePending.value || !payload?.laneKey) return;
  loadMorePending.value = true;
  try {
    await controller.loadMoreLane(payload.laneKey);
    await syncLanesFromController();
  } catch (e) {
    ChoyMessage.error(_t('Failed to load more tokens'));
    console.error('Token kanban load-more failed:', e);
  } finally {
    loadMorePending.value = false;
  }
}

/**
 * Persist lane moves (Revoked field) via the kanban controller.
 * Blocks overlapping moves while a write is in flight.
 */
async function onCardMove(move: ChoyKanbanMove) {
  if (movePending.value) return;
  // Synthetic Vue keys (lane-index fallbacks) must not drive UpdateById.
  const recordId = resolveTokenMoveRecordId(
    choyLanes.value.flatMap(lane => lane.cards),
    move.cardId,
  );
  if (!recordId) {
    // Optimistic v-model move already mutated lanes; restore controller state.
    await syncLanesFromController();
    return;
  }
  movePending.value = true;
  try {
    await controller.moveCard(recordId, move.fromLaneKey, move.toLaneKey, move.toIndex);
    await syncLanesFromController();
  } catch (e) {
    ChoyMessage.error(_t('Move failed; refreshed to recover'));
    console.error('Token kanban move failed:', e);
    try {
      await applyCurrentQuery();
    } catch (reloadError) {
      console.error('Token kanban reload failed:', reloadError);
    }
    await syncLanesFromController();
  } finally {
    movePending.value = false;
  }
}

function laneLabel(lane: Lane | ChoyKanbanLane): string {
  const key = String((lane as Lane).key ?? (lane as ChoyKanbanLane).key);
  const label = String((lane as Lane).label ?? (lane as ChoyKanbanLane).label ?? '');
  if (/Revoked=true/.test(key)) return _t('Revoked');
  if (/Revoked=false/.test(key)) return _t('Not Revoked');
  if (label === 'true') return _t('Revoked');
  if (label === 'false') return _t('Not Revoked');
  return label || key;
}

function payloadOf(card: ChoyKanbanCard): Record<string, unknown> {
  return (card.payload ?? {}) as Record<string, unknown>;
}

function tokenTypeLabel(card: ChoyKanbanCard): string {
  return String(payloadOf(card).TokenType ?? '');
}

function tokenTypeClass(card: ChoyKanbanCard): string {
  return String(payloadOf(card).TokenType ?? '').toLowerCase();
}

function usernameLabel(card: ChoyKanbanCard): string {
  return resolveTokenUsernameLabel(payloadOf(card));
}

function isRevokedCard(card: ChoyKanbanCard): boolean {
  return Boolean(payloadOf(card).Revoked);
}

function expiresAt(card: ChoyKanbanCard): unknown {
  return payloadOf(card).ExpiresAt;
}

function revokedAt(card: ChoyKanbanCard): unknown {
  return payloadOf(card).RevokedAt;
}

/**
 * Format token timestamps for kanban card display.
 */
function formatDate(dt: unknown): string {
  if (!dt) return '';
  try {
    const d = typeof dt === 'string' ? new Date(dt) : dt instanceof Date ? dt : new Date(String(dt));
    if (!(d instanceof Date) || isNaN(d.getTime())) return String(dt).slice(0, 19);
    const y = d.getFullYear();
    const m = (d.getMonth() + 1).toString().padStart(2, '0');
    const day = d.getDate().toString().padStart(2, '0');
    const hh = d.getHours().toString().padStart(2, '0');
    const mm = d.getMinutes().toString().padStart(2, '0');
    return `${y}-${m}-${day} ${hh}:${mm}`;
  } catch {
    return String(dt).slice(0, 19);
  }
}
</script>

<style scoped lang="scss">
.token-card {
  cursor: grab;
}

.token-card.revoked {
  opacity: 0.9;
}

.token-type.access {
  color: #2563eb;
  background: rgba(37, 99, 235, 0.12);
}

.token-type.refresh {
  color: #16a34a;
  background: rgba(22, 163, 74, 0.12);
}

.token-type:not(.access):not(.refresh) {
  background: color-mix(in oklab, var(--foreground, currentColor) 8%, transparent);
}
</style>
