<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OKanbanView
    :store="store"
    :show-header="showHeader"
    :show-actions="true"
    :show-paginate="true"
    :searchView="OSearchView"
    create-action="/auth/tokens/new"
    @card-click="onCardClick"
    @card-move="onCardMove"
  >
    <template #header-right>
      <el-button-group>
        <el-tooltip :content="_t('List View')" placement="top"> <el-button :icon="FormatListBulletedOutlined" @click="toList" /></el-tooltip>
        <el-tooltip :content="_t('Kanban View')" placement="top"><el-button :icon="GridViewSharp" @click="toKanban" type="primary" /></el-tooltip>
        <el-tooltip :content="_t('Icon View')" placement="top"><el-button :icon="BarChartOutlined" @click="toKanban" /></el-tooltip>
      </el-button-group>
    </template>

    <template #fields>
      <!-- Register virtual fields so card slots can read record.X directly. -->
      <OVirtualField :store="store" prop="TokenType" />
      <OVirtualField :store="store" prop="ExpiresAt" />
      <OVirtualField :store="store" prop="Revoked" />
      <OVirtualField :store="store" prop="RevokedAt" />
      <!-- Register nested relation fields used by the card body. -->
      <OVirtualField :store="store" prop="UserId.Username" />
      <OVirtualField :store="store" prop="CreatedAt" />
    </template>

    <template #lane-header="{ lane }">
      <div class="token-lane-header">
        <span class="title">{{ laneLabel(lane) }}</span>
        <span class="count">({{ lane.count ?? 0 }})</span>
      </div>
    </template>

    <template #card="{ record }">
      <div class="token-card" :class="{ revoked: record.Revoked }" @dblclick="openDetail(record)">
        <div class="top-row">
          <span class="token-type" :class="record.TokenType">{{ record.TokenType }}</span>
          <OVarCharField :store="store" prop="UserId.Username" />
        </div>
        <div class="expires" :title="formatDate(record.ExpiresAt)">{{ _t('Expires') }}: {{ formatDate(record.ExpiresAt) }}</div>
        <div class="revoked-info" v-if="record.Revoked">{{ _t('Revoked At') }}: {{ formatDate(record.RevokedAt) || '—' }}</div>
      </div>
    </template>

    <template #card-empty="{ lane }">
      <div class="empty-lane">{{ _t('No tokens in this lane') }}</div>
    </template>
  </OKanbanView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type Token from '@/auth/service/models/token';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import OKanbanView from '@/web/web/components/view/OKanbanView.vue';
import { ElButton, ElTooltip, ElButtonGroup } from 'element-plus';
import { FormatListBulletedOutlined, GridViewSharp, BarChartOutlined } from '@vicons/material';
import type { ClientModelProps } from '@/core/rpc/types';
import OVirtualField from '@/web/web/components/field/OVirtualField.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'TokenKanbanView' });
const { _t } = createTranslate('auth', { scope: 'web/views/TokenKanbanView' });

const props = withDefaults(defineProps<{ store: WebModelStore<Token>; showHeader?: boolean }>(), { showHeader: true });
const { store, showHeader } = props;

const router = useRouter();

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

/**
 * Open the token detail page for the selected record.
 */
function openDetail(rec: ClientModelProps<Token>) {
  router.push(`/auth/tokens/${rec.Id}`);
}

/**
 * Handle kanban card clicks by opening the record detail view.
 */
function onCardClick(payload: { row: ClientModelProps<Token> }) {
  openDetail(payload.row);
}

/**
 * Reserve a hook for drag-and-drop side effects.
 */
function onCardMove(e: { cardId: string; fromLane: string; toLane: string; index: number }) {
  // The kanban component already persists Revoked changes.
  void e;
}

/**
 * Convert a kanban lane descriptor into a display label.
 */
function laneLabel(lane: any): string {
  // Convert Revoked=true/false lanes into human-readable labels.
  const key = String(lane.key);
  if (/Revoked=true/.test(key)) return _t('Revoked');
  if (/Revoked=false/.test(key)) return _t('Not Revoked');
  // Fall back to the lane label when the key was not normalized.
  if (lane.label === 'true') return _t('Revoked');
  if (lane.label === 'false') return _t('Not Revoked');
  return String(lane.label || lane.key);
}

/**
 * Format token timestamps for kanban card display.
 */
function formatDate(dt: any): string {
  if (!dt) return '';
  try {
    const d = typeof dt === 'string' ? new Date(dt) : dt;
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
/* Token kanban styling aligned with the shared board look and feel. */
.token-lane-header {
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--el-text-color-primary);
}

.token-card {
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  padding: 10px 12px 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 12px;
  background: var(--el-color-white);
  transition:
    box-shadow 0.18s ease,
    transform 0.18s ease;
  position: relative;
  cursor: grab; /* Match the shared drag wrapper affordance. */
}
.token-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}
.token-card:active {
  transform: scale(0.98);
}
.token-card.revoked {
  opacity: 0.9;
  background: #fff7f5;
}

.top-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.token-type {
  font-weight: 600;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.token-type.access {
  color: #409eff;
  background: rgba(64, 158, 255, 0.12);
}
.token-type.refresh {
  color: #67c23a;
  background: rgba(103, 194, 58, 0.12);
}

.user {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.expires {
  color: var(--el-text-color-regular);
  font-size: 12px;
}
.revoked-info {
  color: #e67e22;
  font-size: 12px;
}

.empty-lane {
  opacity: 0.6;
  font-size: 12px;
  padding: 6px 0;
  text-align: center;
}
</style>
