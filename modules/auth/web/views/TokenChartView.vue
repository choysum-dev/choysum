<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="token-chart-view flex h-full min-h-0 min-w-0 flex-1 flex-col">
    <div class="sr-only" aria-hidden="true">
      <ChoyVirtualField :store="store" prop="TokenType" />
      <ChoyVirtualField :store="store" prop="Revoked" />
      <ChoyVirtualField :store="store" prop="UserId.Username" />
      <ChoyVirtualField :store="store" prop="ExpiresAt" />
      <ChoyVirtualField :store="store" prop="CreatedAt" />
    </div>

    <ChoyChartView
      class="min-h-0 flex-1"
      :store="store"
      :show-header="showHeader"
      :show-actions="true"
      :show-create="hasAction('auth.action.token_create')"
      :create-label="_t('New')"
      :default-groups="defaultGroups"
      :format-label="formatChartLabel"
      :metrics="chartMetrics"
      :chart-types="['bar', 'line', 'pie']"
      chart-type="bar"
      :stacked="true"
      metric-alias="count"
      :on-load-error="onLoadError"
      :on-search-error="onSearchError"
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
          active="chart"
          :options="viewSwitcherOptions"
          @select="onViewSelect"
        />
      </template>
    </ChoyChartView>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import type Token from '@/auth/service/models/token';
import { usePermission } from '@/auth/web/composables/usePermission';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import {
  ChoyChartView,
  ChoyMessage,
  ChoySearchView,
  ChoyViewSwitcher,
  ChoyVirtualField,
  type ChoyViewSwitcherOption,
} from '@/web';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import type { GroupBySpec } from '@/web/web/query/types';
import {
  ensureTokenViewGroups,
  formatTokenRevokedChartLabel,
  tokenViewDefaultGroups,
  tokenViewPath,
  type TokenViewKind,
} from './token_view_nav';

defineOptions({ name: 'TokenChartView' });
const { _t } = createTranslate('auth', { scope: 'web/views/TokenChartView' });

const props = withDefaults(defineProps<{ store?: WebModelStore<Token>; showHeader?: boolean }>(), {
  showHeader: true,
});
const store = resolvePageStore(props.store, 'TokenChartView');
const { showHeader } = props;
const { hasAction } = usePermission();

// Correct shared-store groups before ChoyChartView bootstraps (parent setup first).
ensureTokenViewGroups(store, 'chart');

/**
 * Category = TokenType, series = Revoked (nested children via fullGroupHierarchy).
 * Demonstrates stackable multi-series ChartView on real Token data.
 */
const defaultGroups = tokenViewDefaultGroups('chart') as GroupBySpec<Token>[];

const chartMetrics = computed(() => [{ alias: 'count', label: _t('Count') }]);

const router = useRouter();

const viewSwitcherOptions = computed<ChoyViewSwitcherOption[]>(() => [
  { value: 'list', label: _t('List View') },
  { value: 'kanban', label: _t('Kanban View') },
  { value: 'chart', label: _t('Chart View') },
]);

function formatChartLabel(label: string): string {
  return formatTokenRevokedChartLabel(label, {
    revoked: _t('Revoked'),
    notRevoked: _t('Not Revoked'),
  });
}

function onViewSelect(value: string) {
  router.push(tokenViewPath(value as TokenViewKind));
}

function onCreate() {
  router.push('/auth/tokens/new');
}

function onLoadError(e: unknown) {
  ChoyMessage.error(_t('Failed to load chart'));
  console.error('Token chart load failed:', e);
}

function onSearchError(e: unknown) {
  ChoyMessage.error(_t('Failed to load chart'));
  console.error('Token chart search failed:', e);
}
</script>
