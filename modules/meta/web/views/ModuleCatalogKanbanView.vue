<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    class="module-catalog-kanban-view flex h-full min-h-0 min-w-0 flex-1 flex-col"
    data-testid="module-catalog-view"
  >
    <div class="sr-only" aria-hidden="true">
      <ChoyVirtualField :store="store" prop="ModuleName" />
      <ChoyVirtualField :store="store" prop="Installed" />
      <ChoyVirtualField :store="store" prop="InstalledVersion" />
      <ChoyVirtualField :store="store" prop="LocalVersion" />
      <ChoyVirtualField :store="store" prop="RegistryVersion" />
      <ChoyVirtualField :store="store" prop="UpdateAvailable" />
      <ChoyVirtualField :store="store" prop="Application" />
      <ChoyVirtualField :store="store" prop="Author" />
      <ChoyVirtualField :store="store" prop="License" />
      <ChoyVirtualField :store="store" prop="Summary" />
    </div>

    <ChoyKanbanView
      :store="store"
      :show-header="showHeader"
      :show-actions="true"
      readonly
      :flat-lane-label="_t('All')"
      :map-row-to-card="rowToCard"
      :resolve-move-record-id="resolveMoveRecordId"
      :on-load-error="onLoadError"
      :on-search-error="onSearchError"
      :on-load-more-error="onLoadMoreError"
      @card-click="onCardClick"
    >
      <template #search="{ onQueryUpdate }">
        <ChoySearchView
          class="min-w-0 w-full"
          :store="store"
          :default-groups="defaultGroups"
          @query-update="onQueryUpdate"
        />
      </template>

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

      <template #card="{ card }">
        <!--
          Intrinsic height (no h-full / min-h-0): flat board scrolls the grid;
          card body must not shrink-clip meta rows when the viewport is short.
        -->
        <ModuleCatalogKanbanCard
          :card="card"
          :status-label="cardStatusLabel(card)"
          :status-variant="statusBadgeVariant(cardPayload(card))"
          :primary-action="primaryActionFor(cardPayload(card))"
          :primary-label="primaryActionLabel(primaryActionFor(cardPayload(card)))"
          :show-uninstall="canUninstall(cardPayload(card)) && hasAction(moduleUninstallAction)"
          @primary="onPrimaryAction(cardPayload(card))"
          @uninstall="onUninstall(cardPayload(card))"
        />
      </template>

      <template #card-empty>
        <div class="text-xs opacity-60">{{ _t('No modules match this filter.') }}</div>
      </template>
    </ChoyKanbanView>

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
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { useModuleCatalogViewModel } from '../composables/useModuleCatalogViewModel';
import { moduleLoadDemoAction, moduleSyncAction, moduleUninstallAction } from './module_catalog_actions';
import ModuleCatalogKanbanCard from './ModuleCatalogKanbanCard.vue';
import ModuleCatalogOpDialog from './ModuleCatalogOpDialog.vue';
import ModuleCatalogOpProgressStrip from './ModuleCatalogOpProgressStrip.vue';
import {
  createModuleCatalogKanbanErrorHandlers,
  moduleCatalogKanbanCardPayload as cardPayload,
  moduleCatalogKanbanResolveMoveRecordId as resolveMoveRecordId,
  moduleCatalogKanbanRowToCard as rowToCard,
} from './module_catalog_kanban_view';
import {
  ChoyButton,
  ChoyKanbanView,
  ChoyMessage,
  ChoySearchView,
  ChoyViewSwitcher,
  ChoyVirtualField,
  useChoyViewRouteSwitch,
  type ChoyKanbanCard,
  type ChoyViewSwitcherOption,
} from '@/web';
import type { GroupBySpec } from '@/web/web/query/types';

defineOptions({ name: 'ModuleCatalogKanbanView', inheritAttrs: false });

const { _t } = createTranslate('meta', { scope: 'web/views/ModuleCatalogKanbanView' });
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

const store = resolvePageStore(props.store, 'ModuleCatalogKanbanView');
const showHeader = toRef(props, 'showHeader');
// Prefer the prop binding in the template (`operationStore`) so we pass the
// concrete Pinia store — wrapping it in computed() can leave a ComputedRef
// unwrapped and break session.store.Plan().

const { active: activeView, defaultGroups: routeGroups, selectView: onViewSelect } =
  useChoyViewRouteSwitch({ store });
const defaultGroups = routeGroups.value as GroupBySpec<MetaModuleCatalog>[];

const viewSwitcherOptions = computed<ChoyViewSwitcherOption[]>(() => [
  { value: 'kanban', label: _t('Kanban View') },
  { value: 'list', label: _t('List View') },
]);

const {
  drawerOpen,
  drawerOperationId,
  drawerModuleName,
  drawerAction,
  drawerRegistryNewerThanLocal,
  syncBusy,
  lastSyncAtLabel,
  onSync,
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
  toShow,
  rowRecord,
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

function cardStatusLabel(card: ChoyKanbanCard): string {
  return statusLabel(cardPayload(card));
}

function onCardClick(card: ChoyKanbanCard) {
  toShow(cardPayload(card));
}

const { onLoadError, onSearchError, onLoadMoreError } = createModuleCatalogKanbanErrorHandlers(
  {
    loadFailed: _t('Failed to load modules'),
    searchFailed: _t('Failed to load modules'),
    loadMoreFailed: _t('Failed to load more'),
  },
  message => ChoyMessage.error(message),
);

onMounted(() => {
  bootFromRoute();
});

defineExpose({
  openDrawer,
  closeDrawer,
});
</script>
