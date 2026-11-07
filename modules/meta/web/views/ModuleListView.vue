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
    :action-ids="{}"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <!-- Domain secondary actions share the title ⋮ with Refresh / import / export. -->
    <template
      v-if="showHeader && (canRoute('meta.route.module_history') || hasAction(moduleSyncAction))"
      #overflow-actions
    >
      <ChoyDropdownMenuItem
        v-if="canRoute('meta.route.module_history')"
        data-testid="module-overflow-history"
        @select="toHistory"
      >
        <History class="size-4" aria-hidden="true" />
        {{ _t('Operation History') }}
      </ChoyDropdownMenuItem>
      <ChoyDropdownMenuItem
        v-if="hasAction(moduleSyncAction)"
        data-testid="module-overflow-sync"
        :disabled="syncLoading"
        @select="onSyncIndex"
      >
        <FolderSync class="size-4" :class="{ 'animate-spin': syncLoading }" aria-hidden="true" />
        {{ _t('Sync Modules') }}
      </ChoyDropdownMenuItem>
    </template>

    <!-- View-mode toggles only; ListView keeps pagination at the trailing edge. -->
    <template v-if="showHeader && viewSwitcherOptions.length" #header-right>
      <ChoyViewSwitcher
        :aria-label="_t('View switcher')"
        :active="isListRoute ? 'list' : 'board'"
        :options="viewSwitcherOptions"
        @select="onViewSelect"
      />
    </template>

    <ChoyTableColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField prop="ModuleName" :store="store" :vColumnProps="{ minWidth: 180 }" />
    <ChoyVarcharField prop="LocalVersion" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <ChoyVarcharField prop="RegistryVersion" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <ChoyVarcharField prop="InstalledVersion" :label="_t('Installed Version')" :store="store" :vColumnProps="{ minWidth: 120 }" />

    <ChoyVirtualField :store="store" prop="Installed" />
    <ChoyTableColumn
      prop="Installed"
      :label="_t('Install Status')"
      :vColumnProps="{ minWidth: 120 }"
    >
      <template #default="{ row }">
        <ChoyBadge :variant="statusBadgeVariant(row)" class="font-normal">
          {{ statusLabel(row) }}
        </ChoyBadge>
      </template>
    </ChoyTableColumn>

    <ChoyBooleanField prop="UpdateAvailable" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <ChoyBooleanField prop="Available" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ChoyVarcharField prop="Application" :label="_t('Application')" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyVarcharField prop="LocalPath" :label="_t('Path')" :store="store" :vColumnProps="{ minWidth: 220 }" />
    <ChoyDatetimeField prop="LastSyncAt" :label="_t('Synced At')" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </ChoyListView>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { FolderSync, History } from 'lucide-vue-next';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type MetaModuleCatalog from '@/meta/service/models/module_catalog';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { unwrapListRecord } from '@/web/web/composables/listRowEdit';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { moduleSyncAction } from './module_actions';
import {
  ChoyBadge,
  ChoyBooleanField,
  ChoyDatetimeField,
  ChoyDropdownMenuItem,
  ChoyListView,
  ChoyMessage,
  ChoySearchView,
  ChoyTableColumn,
  ChoyVarcharField,
  ChoyViewSwitcher,
  ChoyVirtualField,
  type ChoyViewSwitcherOption,
} from '@/web';
import type { RowEventPayload } from '@/web';
import { moduleStatusBadgeVariant } from './module_kanban_chrome';

defineOptions({ name: 'ModuleListView', inheritAttrs: false });

const { _t } = createTranslate('meta', { scope: 'web/views/ModuleListView' });

const router = useRouter();
const route = useRoute();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<MetaModuleCatalog>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  },
);

const store = resolvePageStore(props.store, 'ModuleListView');
const { showHeader } = props;
const { canRoute, hasAction } = usePermission();

const isListRoute = computed(() => String(route.path || '').includes('/meta/modules/list'));

const syncLoading = ref(false);
const autoSyncTriggered = ref(false);

async function onSyncIndex() {
  if (syncLoading.value) return;
  syncLoading.value = true;
  try {
    const resp = (await (store as any).RequestSync({ Force: true, IfStale: false })) as { Id?: string } | string | null;
    const operationId = typeof resp === 'string' ? resp : String(resp?.Id || '').trim();
    ChoyMessage.success(operationId ? _t('Sync job triggered: all:%s', operationId) : _t('Sync job triggered'));
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

function onRowClick(payload: RowEventPayload) {
  const id = String(payload.row?.Id ?? '').trim();
  if (id) router.push(`/meta/modules/${id}`);
}

const viewSwitcherOptions = computed<ChoyViewSwitcherOption[]>(() => {
  const opts: ChoyViewSwitcherOption[] = [];
  if (canRoute('meta.route.module_board')) {
    opts.push({ value: 'board', label: _t('Board View') });
  }
  if (canRoute('meta.route.module_list')) {
    opts.push({ value: 'list', label: _t('List View') });
  }
  return opts;
});

function onViewSelect(value: string) {
  if (value === 'list') router.push('/meta/modules/list');
  else router.push('/meta/modules');
}

function toHistory() {
  router.push('/meta/modules/history');
}

function rowRecord(row: unknown): Record<string, unknown> {
  const rec = unwrapListRecord(row);
  return rec && typeof rec === 'object' ? (rec as Record<string, unknown>) : {};
}

function catalogInstalledStatus(rec: Record<string, unknown>): string {
  if (rec.InstalledStatus) return String(rec.InstalledStatus);
  if (rec.Installed === true) return 'installed';
  if (rec.Installed === false) return 'uninstalled';
  return '';
}

function statusBadgeVariant(row: unknown) {
  const rec = rowRecord(row);
  return moduleStatusBadgeVariant(catalogInstalledStatus(rec), rec.Available as boolean | undefined);
}

function statusLabel(row: unknown) {
  const rec = rowRecord(row);
  if (rec.Available === false) return _t('Unavailable');
  const val = catalogInstalledStatus(rec).toLowerCase();
  if (val === 'installed') return _t('Installed');
  if (val === 'uninstalled') return _t('Not Installed');
  if (val === 'disabled') return _t('Disabled');
  if (val === 'broken') return _t('Broken');
  if (val === 'succeeded') return _t('Succeeded');
  if (val === 'failed') return _t('Failed');
  if (val === 'dispatching') return _t('Dispatching');
  if (val === 'queued') return _t('Queued');
  return catalogInstalledStatus(rec) || _t('Unknown');
}

const { listRef, expose } = useListViewExpose<MetaModuleCatalog>();
defineExpose(expose);
</script>
