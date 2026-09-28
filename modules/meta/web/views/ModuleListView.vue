<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="module-list-view">
    <div v-if="showHeader" class="mb-2 flex justify-end">
      <div class="flex items-center gap-1">
        <ChoyButton
          v-if="canRoute('meta.route.module_board')"
          variant="outline"
          size="sm"
          :title="_t('Board View')"
          @click="toKanban"
        >
          <LayoutGrid class="size-4" aria-hidden="true" />
        </ChoyButton>
        <ChoyButton size="sm" :title="_t('List View')" @click="toList">
          <List class="size-4" aria-hidden="true" />
        </ChoyButton>
        <ChoyButton
          v-if="canRoute('meta.route.module_history')"
          variant="outline"
          size="sm"
          :title="_t('Operation History')"
          @click="toHistory"
        >
          <History class="size-4" aria-hidden="true" />
        </ChoyButton>
        <ChoyButton
          v-if="hasAction(moduleSyncIndexAction)"
          variant="outline"
          size="sm"
          :title="_t('Sync Index')"
          :disabled="syncLoading"
          @click="onSyncIndex"
        >
          <RefreshCw class="size-4" :class="{ 'animate-spin': syncLoading }" aria-hidden="true" />
        </ChoyButton>
      </div>
    </div>
    <ChoyListView
      ref="listRef"
      v-bind="$attrs"
      :store="store"
      :searchView="ChoySearchView"
      :show-header="showHeader"
      :action-ids="{ delete: moduleIndexActions.delete }"
      :has-action="hasAction"
      @row-click="onRowClick"
    >
      <ChoyVColumn type="index" :vColumnProps="{ align: 'right' }" />
      <ChoyVarcharField prop="ModuleName" :store="store" :vColumnProps="{ minWidth: 180 }" />
      <ChoyVarcharField prop="LocalVersion" :store="store" :vColumnProps="{ minWidth: 120 }" />
      <ChoyVarcharField prop="RegistryVersion" :store="store" :vColumnProps="{ minWidth: 120 }" />
      <ChoyVarcharField prop="Version" :label="_t('Display Version')" :store="store" :vColumnProps="{ minWidth: 120 }" />
      <ChoyVarcharField prop="InstalledStatus" :store="store" :vColumnProps="{ minWidth: 120 }" />
      <ChoyVarcharField prop="InstalledVersion" :store="store" :vColumnProps="{ minWidth: 120 }" />
      <ChoyBooleanField prop="Available" :store="store" :vColumnProps="{ minWidth: 100 }" />
      <ChoyVarcharField prop="OriginTypes" :label="_t('Origin')" :store="store" :vColumnProps="{ minWidth: 140 }" />
      <ChoyVarcharField prop="LocalPath" :label="_t('Path')" :store="store" :vColumnProps="{ minWidth: 220 }" />
      <ChoyDatetimeField prop="LastSyncAt" :label="_t('Synced At')" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    </ChoyListView>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { History, LayoutGrid, List, RefreshCw } from 'lucide-vue-next';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type MetaModuleIndex from '@/meta/service/models/module_index';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineAction, defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyBooleanField,
  ChoyButton,
  ChoyDatetimeField,
  ChoyListView,
  ChoyMessage,
  ChoySearchView,
  ChoyVColumn,
  ChoyVarcharField,
} from '@/web';
import { resolveListRowRecordId } from './list_row_nav';

defineOptions({ name: 'ModuleListView', inheritAttrs: false });

const { _t, _lt } = createTranslate('meta', { scope: 'web/views/ModuleListView' });

const router = useRouter();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<MetaModuleIndex>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  },
);

const store = resolvePageStore(props.store, 'ModuleListView');
const { showHeader } = props;
const moduleSyncIndexAction = defineAction('meta.action.module_sync_index', {
  title: _lt('Sync Module Index'),
  requires: [{ model: 'meta.MetaModuleIndex', method: 'RequestSync' }],
});
const moduleIndexActions = defineModelActions('meta.MetaModuleIndex', {
  entityTitle: _lt('Module Index'),
});
const { canRoute, hasAction } = usePermission();

const syncLoading = ref(false);
const autoSyncTriggered = ref(false);

async function onSyncIndex() {
  if (syncLoading.value) return;
  syncLoading.value = true;
  try {
    const jobId = await (store as any).RequestSync({ Force: true, IfStale: false });
    ChoyMessage.success(jobId ? _t('Sync job triggered: all:%s', String(jobId)) : _t('Sync job triggered'));
  } catch (error: any) {
    ChoyMessage.warning(_t('Sync failed: %s', String(error?.message || 'request failed')));
  } finally {
    syncLoading.value = false;
  }
}

watch(
  () => (store as any)?.state?.result,
  result => {
    if (autoSyncTriggered.value) return;
    const rows = (result as any)?.rows as any[] | undefined;
    if (!Array.isArray(rows)) return;
    if (rows.length > 0) return;
    autoSyncTriggered.value = true;
    void onSyncIndex();
  },
);

function onRowClick(row: Record<string, unknown>) {
  const id = resolveListRowRecordId(row);
  if (id) router.push(`/meta/modules/${id}`);
}

function toKanban() {
  router.push('/meta/modules');
}
function toList() {
  router.push('/meta/modules/list');
}
function toHistory() {
  router.push('/meta/modules/history');
}

const { listRef, expose } = useListViewExpose<MetaModuleIndex>();
defineExpose(expose);
</script>
