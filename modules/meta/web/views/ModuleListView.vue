<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyListView
    ref="listRef"
    class="module-list-view"
    v-bind="$attrs"
    :store="store"
    :searchView="ChoySearchView"
    :show-header="showHeader"
    :action-ids="{ delete: moduleIndexActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <template v-if="showHeader" #header-right>
      <div class="flex items-center gap-1">
        <ChoyButton
          v-if="canRoute('meta.route.module_board')"
          type="button"
          size="icon"
          :variant="isListRoute ? 'outline' : 'default'"
          :aria-label="_t('Board View')"
          @click="toKanban"
        >
          <LayoutGrid class="size-4" aria-hidden="true" />
        </ChoyButton>
        <ChoyButton
          v-if="canRoute('meta.route.module_list')"
          type="button"
          size="icon"
          :variant="isListRoute ? 'default' : 'outline'"
          :aria-label="_t('List View')"
          @click="toList"
        >
          <List class="size-4" aria-hidden="true" />
        </ChoyButton>

        <ChoyTooltipProvider v-if="canRoute('meta.route.module_history')">
          <ChoyTooltip>
            <ChoyTooltipTrigger as-child>
              <ChoyButton
                type="button"
                size="icon"
                variant="ghost"
                :aria-label="_t('Operation History')"
                @click="toHistory"
              >
                <History class="size-4" aria-hidden="true" />
              </ChoyButton>
            </ChoyTooltipTrigger>
            <ChoyTooltipContent side="bottom">{{ _t('Operation History') }}</ChoyTooltipContent>
          </ChoyTooltip>
        </ChoyTooltipProvider>

        <ChoyTooltipProvider v-if="hasAction(moduleSyncIndexAction)">
          <ChoyTooltip>
            <ChoyTooltipTrigger as-child>
              <ChoyButton
                type="button"
                size="icon"
                variant="ghost"
                :aria-label="_t('Sync Index')"
                :disabled="syncLoading"
                @click="onSyncIndex"
              >
                <FolderSync class="size-4" :class="{ 'animate-spin': syncLoading }" aria-hidden="true" />
              </ChoyButton>
            </ChoyTooltipTrigger>
            <ChoyTooltipContent side="bottom">{{ _t('Sync Index') }}</ChoyTooltipContent>
          </ChoyTooltip>
        </ChoyTooltipProvider>
      </div>
    </template>

    <ChoyTableColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField prop="ModuleName" :store="store" :vColumnProps="{ minWidth: 180 }" />
    <ChoyVarcharField prop="LocalVersion" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <ChoyVarcharField prop="RegistryVersion" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <ChoyVarcharField prop="Version" :label="_t('Display Version')" :store="store" :vColumnProps="{ minWidth: 120 }" />

    <ChoyVirtualField :store="store" prop="InstalledStatus" />
    <ChoyTableColumn
      prop="InstalledStatus"
      :label="_t('Install Status')"
      :vColumnProps="{ minWidth: 120 }"
    >
      <template #default="{ row }">
        <ChoyBadge :variant="statusBadgeVariant(row)" class="font-normal">
          {{ statusLabel(row) }}
        </ChoyBadge>
      </template>
    </ChoyTableColumn>

    <ChoyVarcharField prop="InstalledVersion" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <ChoyBooleanField prop="Available" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ChoyVarcharField prop="OriginTypes" :label="_t('Origin')" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyVarcharField prop="LocalPath" :label="_t('Path')" :store="store" :vColumnProps="{ minWidth: 220 }" />
    <ChoyDatetimeField prop="LastSyncAt" :label="_t('Synced At')" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </ChoyListView>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { FolderSync, History, LayoutGrid, List } from 'lucide-vue-next';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type MetaModuleIndex from '@/meta/service/models/module_index';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { unwrapListRecord } from '@/web/web/composables/listRowEdit';
import { defineAction, defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyBadge,
  ChoyBooleanField,
  ChoyButton,
  ChoyDatetimeField,
  ChoyListView,
  ChoyMessage,
  ChoySearchView,
  ChoyTableColumn,
  ChoyTooltip,
  ChoyTooltipContent,
  ChoyTooltipProvider,
  ChoyTooltipTrigger,
  ChoyVarcharField,
  ChoyVirtualField,
} from '@/web';
import { resolveListRowRecordId } from './list_row_nav';
import { moduleStatusBadgeVariant } from './module_kanban_chrome';

defineOptions({ name: 'ModuleListView', inheritAttrs: false });

const { _t, _lt } = createTranslate('meta', { scope: 'web/views/ModuleListView' });

const router = useRouter();
const route = useRoute();

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

const isListRoute = computed(() => String(route.path || '').includes('/meta/modules/list'));

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

function rowRecord(row: unknown): Record<string, unknown> {
  const rec = unwrapListRecord(row);
  return rec && typeof rec === 'object' ? (rec as Record<string, unknown>) : {};
}

function statusBadgeVariant(row: unknown) {
  const rec = rowRecord(row);
  return moduleStatusBadgeVariant(
    String(rec.InstalledStatus ?? ''),
    rec.Available as boolean | undefined,
  );
}

function statusLabel(row: unknown) {
  const rec = rowRecord(row);
  if (rec.Available === false) return _t('Unavailable');
  const val = String(rec.InstalledStatus || '').toLowerCase();
  if (val === 'installed') return _t('Installed');
  if (val === 'uninstalled') return _t('Not Installed');
  if (val === 'disabled') return _t('Disabled');
  if (val === 'broken') return _t('Broken');
  if (val === 'succeeded') return _t('Succeeded');
  if (val === 'failed') return _t('Failed');
  if (val === 'dispatching') return _t('Dispatching');
  if (val === 'queued') return _t('Queued');
  return String(rec.InstalledStatus || '') || _t('Unknown');
}

const { listRef, expose } = useListViewExpose<MetaModuleIndex>();
defineExpose(expose);
</script>
