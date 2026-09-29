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
        <ChoySearchView :store="store" @query-update="onQueryUpdate" />
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
          <span class="title">{{ lane.label }}</span>
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
          {{ _t('Load more (%s remaining)', remain) }}
        </ChoyButton>
      </template>
    </ChoyKanbanView>
  </div>
</template>

<script setup lang="ts">
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
  type ChoyKanbanLane
} from '@/web';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import type { Lane } from '@/web/web/query/types';
import {
  resolveTokenDetailId,
  resolveTokenKanbanCardId,
  resolveTokenKanbanRowPayload,
  resolveTokenMoveRecordId,
  resolveTokenUsernameLabel,
  type TokenKanbanRow
} from './token_kanban_nav';

defineOptions({ name: 'TokenKanbanView' });
const { _t } = createTranslate('auth', { scope: 'web/views/TokenKanbanView' });

const props = withDefaults(defineProps<{ store?: WebModelStore<Token>; showHeader?: boolean }>(), { showHeader: true });
const store = resolvePageStore(props.store, 'TokenKanbanView');
const { showHeader } = props;

const router = useRouter();

function rowToCard(row: unknown, index: number, laneKey: string): ChoyKanbanCard {
  const payload = resolveTokenKanbanRowPayload(row as TokenKanbanRow);
  return {
    id: resolveTokenKanbanCardId(row as TokenKanbanRow, index, laneKey),
    title: String(payload.TokenType ?? payload.Id ?? ''),
    laneKey,
    payload,
  };
}

function resolveMoveRecordId(
  cards: ReadonlyArray<ChoyKanbanCard>,
  moveCardId: string,
): string {
  return resolveTokenMoveRecordId(cards, moveCardId);
}

function laneLabel(lane: Pick<ChoyKanbanLane, 'key' | 'label'> | Lane): string {
  const key = String((lane as Lane).key ?? (lane as ChoyKanbanLane).key);
  const label = String((lane as Lane).label ?? (lane as ChoyKanbanLane).label ?? '');
  if (/Revoked=true/.test(key)) return _t('Revoked');
  if (/Revoked=false/.test(key)) return _t('Not Revoked');
  if (label === 'true') return _t('Revoked');
  if (label === 'false') return _t('Not Revoked');
  return label || key;
}

function onLoadError(e: unknown) {
  ChoyMessage.error(_t('Failed to load kanban'));
  console.error('Token kanban load failed:', e);
}

function onSearchError(e: unknown) {
  ChoyMessage.error(_t('Failed to load kanban'));
  console.error('Token kanban search failed:', e);
}

function onLoadMoreError(e: unknown) {
  ChoyMessage.error(_t('Failed to load more tokens'));
  console.error('Token kanban load-more failed:', e);
}

function onMoveError(e: unknown) {
  ChoyMessage.error(_t('Move failed; refreshed to recover'));
  console.error('Token kanban move failed:', e);
}

function toList() {
  router.push('/auth/tokens');
}

function toKanban() {
  router.push('/auth/tokens/kanban');
}

function onCreate() {
  router.push('/auth/tokens/new');
}

function openDetailFromCard(card: ChoyKanbanCard) {
  const id = resolveTokenDetailId(card.payload as Record<string, unknown> | undefined);
  if (id) router.push(`/auth/tokens/${id}`);
}

function onCardClick(card: ChoyKanbanCard) {
  openDetailFromCard(card);
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

<style scoped>
.token-card {
  cursor: grab;
}
.token-card.revoked {
  opacity: 0.9;
}
.token-type.access {
  color: #2563eb;background: rgba(37, 99, 235, 0.12);
}
.token-type.refresh {
  color: #16a34a;background: rgba(22, 163, 74, 0.12);
}
.token-type:not(.access):not(.refresh) {
  background: color-mix(in oklab, var(--foreground, currentColor) 8%, transparent);
}
</style>
