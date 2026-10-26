<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="sandbox-record-chart-view flex h-full min-h-0 min-w-0 flex-1 flex-col">
    <div class="sr-only" aria-hidden="true">
      <ChoyVirtualField :store="store" prop="State" />
      <ChoyVirtualField :store="store" prop="IsActive" />
      <ChoyVirtualField :store="store" prop="Title" />
      <ChoyVirtualField :store="store" prop="Code" />
    </div>

    <ChoyChartView
      class="min-h-0 flex-1"
      :store="store"
      :show-header="showHeader"
      :show-actions="true"
      :show-create="hasAction('sandbox.action.sandbox_record_create')"
      :create-label="_t('New')"
      :default-groups="defaultGroups"
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
          :active="activeView"
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
import type SandboxRecord from '@/sandbox/service/models/sandbox_record';
import { usePermission } from '@/auth/web/composables/usePermission';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import {
  ChoyChartView,
  ChoyMessage,
  ChoySearchView,
  ChoyViewSwitcher,
  ChoyVirtualField,
  useChoyViewRouteSwitch,
  type ChoyViewSwitcherOption,
} from '@/web';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import type { GroupBySpec } from '@/web/web/query/types';

defineOptions({ name: 'SandboxRecordChartView' });
const { _t } = createTranslate('sandbox', { scope: 'web/views/SandboxRecordChartView' });

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<SandboxRecord>;
    showHeader?: boolean;
  }>(),
  { showHeader: true },
);
const store = resolvePageStore(props.store, 'SandboxRecordChartView');
const { showHeader } = props;
const { hasAction } = usePermission();

const { active: activeView, defaultGroups: routeGroups, selectView: onViewSelect } =
  useChoyViewRouteSwitch({ store });
const defaultGroups = routeGroups.value as GroupBySpec<SandboxRecord>[];
const chartMetrics = computed(() => [{ alias: 'count', label: _t('Count') }]);
const router = useRouter();

const viewSwitcherOptions = computed<ChoyViewSwitcherOption[]>(() => [
  { value: 'list', label: _t('List View') },
  { value: 'kanban', label: _t('Kanban View') },
  { value: 'chart', label: _t('Chart View') },
]);

function onCreate() {
  router.push('/sandbox/records/new');
}

function onLoadError(e: unknown) {
  ChoyMessage.error(_t('Failed to load chart'));
  console.error('Sandbox chart load failed:', e);
}

function onSearchError(e: unknown) {
  ChoyMessage.error(_t('Failed to load chart'));
  console.error('Sandbox chart search failed:', e);
}
</script>
