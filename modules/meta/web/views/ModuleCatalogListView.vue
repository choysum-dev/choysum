<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    class="module-catalog-list-view flex h-full min-h-0 min-w-0 flex-1 flex-col"
    data-testid="module-catalog-view"
  >
    <ChoyListView
      ref="listRef"
      class="module-catalog-table min-h-0 flex-1"
      v-bind="$attrs"
      :store="store"
      :searchView="ChoySearchView"
      :show-header="showHeader"
      :show-actions="true"
      :has-action="hasAction"
      data-testid="module-catalog-table"
      @row-click="onRowClick"
    >
      <!-- Suppress default New; keep show-actions for Refresh in ⋮. -->
      <template v-if="showHeader" #system-actions />

      <!-- After title + ⋮ in the system tray (no ActionBar separator). -->
      <template v-if="showHeader" #secondary-actions>
        <ChoyButton
          v-if="hasAction(moduleSyncAction)"
          type="button"
          size="default"
          variant="outline"
          data-testid="module-catalog-sync"
          :disabled="syncBusy"
          :aria-busy="syncBusy || undefined"
          @click="onSync"
        >
          <RefreshCw
            class="size-4"
            :class="{ 'animate-spin': syncBusy }"
            aria-hidden="true"
          />
          {{ _t('Update Module List') }}
        </ChoyButton>
        <ChoyButton
          v-if="hasAction(moduleLoadDemoAction) && hasPendingDemo"
          type="button"
          size="default"
          variant="outline"
          data-testid="module-catalog-load-demo"
          @click="onLoadDemo"
        >
          {{ _t('Load Demo Data') }}
        </ChoyButton>
      </template>

      <template v-if="showHeader" #header-right>
        <span
          v-if="lastSyncAtLabel"
          class="text-[11px] text-muted-foreground"
          data-testid="module-catalog-last-sync"
        >
          {{ _t('Last sync %s', lastSyncAtLabel) }}
        </span>
        <ChoyViewSwitcher
          :aria-label="_t('View switcher')"
          :active="activeView"
          :options="viewSwitcherOptions"
          @select="onViewSelect"
        />
      </template>

      <ChoyTableColumn type="index" :vColumnProps="{ align: 'right' }" />
      <ChoyVarcharField prop="ModuleName" :store="store" :vColumnProps="{ minWidth: 160 }" />
      <ChoyVarcharField prop="Application" :store="store" :vColumnProps="{ minWidth: 120 }" />
      <ChoyVarcharField prop="Author" :store="store" :vColumnProps="{ minWidth: 120 }" />
      <ChoyVarcharField prop="License" :store="store" :vColumnProps="{ minWidth: 110 }" />
      <!-- Version fields stay registered for Browse/Search; UI shows one compressed line. -->
      <ChoyVirtualField :store="store" prop="InstalledVersion" />
      <ChoyVirtualField :store="store" prop="LocalVersion" />
      <ChoyVirtualField :store="store" prop="RegistryVersion" />
      <ChoyVirtualField :store="store" prop="UpdateAvailable" />
      <ChoyTableColumn :label="_t('Version')" :vColumnProps="{ minWidth: 160 }">
        <template #default="{ row }">
          <span class="truncate" :title="versionLineFor(row)">{{ versionLineFor(row) }}</span>
        </template>
      </ChoyTableColumn>
      <ChoyDatetimeField
        prop="LastInstalledOrUpgradedAt"
        :label="_t('Installed / Upgraded At')"
        :store="store"
        :vColumnProps="{ minWidth: 160 }"
      />
      <ChoyVirtualField :store="store" prop="Installed" />
      <ChoyTableColumn prop="Installed" :label="_t('Status')" :vColumnProps="{ minWidth: 120 }">
        <template #default="{ row }">
          <ChoyBadge :variant="statusBadgeVariant(row)" class="font-normal">
            {{ statusLabel(row) }}
          </ChoyBadge>
        </template>
      </ChoyTableColumn>
      <ChoyTableColumn :label="_t('Action')" :vColumnProps="{ minWidth: 160 }">
        <template #default="{ row }">
          <div class="flex flex-wrap items-center gap-1">
            <ChoyButton
              v-if="primaryActionFor(row)"
              type="button"
              size="xs"
              :variant="primaryActionFor(row) === 'progress' ? 'outline' : 'default'"
              :data-testid="`module-catalog-primary-${primaryActionFor(row)}`"
              @click.stop="onPrimaryAction(row)"
            >
              {{ primaryActionLabel(primaryActionFor(row)) }}
            </ChoyButton>
            <ChoyButton
              v-if="canUninstall(row) && hasAction(moduleUninstallAction)"
              type="button"
              size="xs"
              variant="outline-destructive"
              data-testid="module-catalog-uninstall"
              @click.stop="onUninstall(row)"
            >
              {{ _t('Uninstall') }}
            </ChoyButton>
          </div>
        </template>
      </ChoyTableColumn>
    </ChoyListView>

    <ModuleCatalogOpDialog
      v-model:open="drawerOpen"
      :operation-store="operationStore"
      :operation-id="drawerOperationId"
      :module-name="drawerModuleName"
      :action="drawerAction"
      :registry-newer-than-local="drawerRegistryNewerThanLocal"
      @applied="onOpApplied"
      @settled="onDrawerSettled"
    />
    <ModuleCatalogOpProgressStrip
      :busy="progressBusy"
      :title="progressTitle"
      :headline="progressHeadline"
      :value="progressValue"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, toRef } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { RefreshCw } from 'lucide-vue-next';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type MetaModuleCatalog from '@/meta/service/models/module_catalog';
import type MetaModuleOperation from '@/meta/service/models/module_operation';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { useModuleCatalogViewModel } from '../composables/useModuleCatalogViewModel';
import { moduleLoadDemoAction, moduleSyncAction, moduleUninstallAction } from './module_catalog_actions';
import ModuleCatalogOpDialog from './ModuleCatalogOpDialog.vue';
import ModuleCatalogOpProgressStrip from './ModuleCatalogOpProgressStrip.vue';
import { formatModuleCatalogVersionLine } from './module_catalog_chrome';
import { unwrapListRecord } from '@/web/web/composables/listRowEdit';
import {
  ChoyBadge,
  ChoyButton,
  ChoyDatetimeField,
  ChoyListView,
  ChoySearchView,
  ChoyTableColumn,
  ChoyVarcharField,
  ChoyViewSwitcher,
  ChoyVirtualField,
  useChoyViewRouteSwitch,
  type ChoyViewSwitcherOption,
} from '@/web';

defineOptions({ name: 'ModuleCatalogListView', inheritAttrs: false });

const { _t } = createTranslate('meta', { scope: 'web/views/ModuleCatalogListView' });
const router = useRouter();
const route = useRoute();
const { hasAction } = usePermission();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<MetaModuleCatalog>;
    operationStore?: WebModelStore<MetaModuleOperation> | null;
    showHeader?: boolean;
    initialOperationId?: string;
  }>(),
  { showHeader: true, operationStore: null },
);

const store = resolvePageStore(props.store, 'ModuleCatalogListView');
const showHeader = toRef(props, 'showHeader');
// Prefer the prop binding in the template (`operationStore`) so we pass the
// concrete Pinia store — wrapping it in computed() can leave a ComputedRef
// unwrapped and break session.store.Plan().

const { active: activeView, selectView: onViewSelect } = useChoyViewRouteSwitch({ store });

const viewSwitcherOptions = computed<ChoyViewSwitcherOption[]>(() => [
  { value: 'kanban', label: _t('Kanban View') },
  { value: 'list', label: _t('List View') },
]);

function versionLineFor(row: unknown): string {
  const rec = unwrapListRecord(row);
  return formatModuleCatalogVersionLine(
    rec && typeof rec === 'object' ? (rec as Record<string, unknown>) : {},
  );
}

const {
  drawerOpen,
  drawerOperationId,
  drawerModuleName,
  drawerAction,
  drawerRegistryNewerThanLocal,
  syncBusy,
  lastSyncAtLabel,
  onSync,
  onRowClick,
  onPrimaryAction,
  canUninstall,
  onUninstall,
  hasPendingDemo,
  onLoadDemo,
  onOpApplied,
  onDrawerSettled,
  openDrawer,
  closeDrawer,
  progressBusy,
  progressTitle,
  progressHeadline,
  progressValue,
  statusBadgeVariant,
  statusLabel,
  primaryActionFor,
  primaryActionLabel,
  bootFromRoute,
} = useModuleCatalogViewModel({
  store,
  operationStore: toRef(props, 'operationStore'),
  route,
  router,
  initialOperationId: () => String(props.initialOperationId || ''),
  msgs: {
    unknown: _t('Unknown'),
    install: _t('Install'),
    upgrade: _t('Upgrade'),
    progress: _t('View Progress'),
    installed: _t('Installed'),
    updateAvailable: _t('Update available'),
    uninstalled: _t('Not Installed'),
    unavailable: _t('Unavailable'),
    syncFailed: msg => _t('Sync failed: %s', msg),
    // Prefer the sync summary as the toast body; fall back when summary is empty.
    syncSettledOk: summary => (summary ? summary : _t('Module list updated')),
    syncSettledPartial: summary =>
      summary ? summary : _t('Module list updated with warnings'),
    syncSettledFailed: msg => _t('Sync failed: %s', msg),
    syncJobStillRunning: _t('Job is still running in the background; refresh later'),
    syncServiceRestarting: _t('Service is restarting; status will retry automatically'),
    syncFailedToGetStatus: _t('Failed to get status'),
    formatSyncSummaryOk: sync =>
      _t('Module catalog updated, %s modules', String(sync.catalogRows)),
    formatSyncSummaryPartial: sync =>
      _t(
        'Module catalog updated, %s modules (%s failed)',
        String(sync.catalogRows),
        String(sync.failed),
      ),
    opSettledOk: _t('Operation Succeeded'),
    opSettledPartial: _t('Module list updated with warnings'),
    opSettledFailed: msg => _t('Operation Failed: %s', msg),
    opJobStillRunning: _t('Job is still running in the background; refresh later'),
    opServiceRestarting: _t('Service is restarting; status will retry automatically'),
    opFailedToGetStatus: _t('Failed to get status'),
    opReloadingService: () => _t('Almost done, refreshing…'),
    opExecuting: () => _t('Working on it, please wait…'),
    opQueued: () => _t('Getting ready…'),
    opFinishing: () => _t('Wrapping up…'),
    opInProgress: () => _t('Please wait…'),
    opProgressTitle: (action, moduleName) => {
      const name = String(moduleName || '').trim();
      if (action === 'install') return name ? _t('Installing %s', name) : _t('Installing…');
      if (action === 'upgrade') return name ? _t('Upgrading %s', name) : _t('Upgrading…');
      if (action === 'uninstall') return name ? _t('Uninstalling %s', name) : _t('Uninstalling…');
      if (action === 'load_demo') return _t('Loading demo data…');
      if (action === 'progress') return name ? _t('Working on %s', name) : _t('Working…');
      return name ? _t('Working on %s', name) : _t('Working…');
    },
  },
});

onMounted(() => {
  bootFromRoute();
});

const { listRef, expose } = useListViewExpose<MetaModuleCatalog>();
defineExpose({
  ...expose,
  openDrawer,
  closeDrawer,
  versionLineFor,
});
</script>
