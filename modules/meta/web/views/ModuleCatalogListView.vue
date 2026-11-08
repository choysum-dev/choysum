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
          :disabled="syncLoading"
          :aria-busy="syncLoading || undefined"
          @click="onSync"
        >
          <RefreshCw
            class="size-4"
            :class="{ 'animate-spin': syncLoading }"
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
      <ChoyVarcharField prop="Summary" :store="store" :vColumnProps="{ minWidth: 220 }" />
      <ChoyVarcharField
        prop="InstalledVersion"
        :label="_t('Installed')"
        :store="store"
        :vColumnProps="{ minWidth: 110 }"
      />
      <ChoyVarcharField prop="LocalVersion" :label="_t('Local')" :store="store" :vColumnProps="{ minWidth: 110 }" />
      <ChoyVarcharField
        prop="RegistryVersion"
        :label="_t('Registry')"
        :store="store"
        :vColumnProps="{ minWidth: 110 }"
      />
      <ChoyVirtualField :store="store" prop="Installed" />
      <ChoyTableColumn prop="Installed" :label="_t('Status')" :vColumnProps="{ minWidth: 120 }">
        <template #default="{ row }">
          <ChoyBadge :variant="statusBadgeVariant(row)" class="font-normal">
            {{ statusLabel(row) }}
          </ChoyBadge>
        </template>
      </ChoyTableColumn>
      <ChoyBooleanField prop="UpdateAvailable" :store="store" :vColumnProps="{ minWidth: 110 }" />
      <ChoyTableColumn :label="_t('Action')" :vColumnProps="{ minWidth: 160 }">
        <template #default="{ row }">
          <div class="flex flex-wrap items-center gap-1">
            <button
              v-if="primaryActionFor(row)"
              type="button"
              class="rounded-md border border-border px-2 py-0.5 text-xs hover:bg-muted"
              :data-testid="`module-catalog-primary-${primaryActionFor(row)}`"
              @click.stop="onPrimaryAction(row)"
            >
              {{ primaryActionLabel(primaryActionFor(row)) }}
            </button>
            <button
              v-if="canUninstall(row) && hasAction(moduleUninstallAction)"
              type="button"
              class="rounded-md border border-border px-2 py-0.5 text-xs text-muted-foreground hover:bg-muted"
              data-testid="module-catalog-uninstall"
              @click.stop="onUninstall(row)"
            >
              {{ _t('Uninstall') }}
            </button>
          </div>
        </template>
      </ChoyTableColumn>
    </ChoyListView>

    <ModuleOpDrawer
      v-model:open="drawerOpen"
      :operation-store="operationStore"
      :operation-id="drawerOperationId"
      :module-name="drawerModuleName"
      :action="drawerAction"
      :registry-newer-than-local="drawerRegistryNewerThanLocal"
      @settled="onDrawerSettled"
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
import { moduleLoadDemoAction, moduleSyncAction, moduleUninstallAction } from './module_actions';
import ModuleOpDrawer from './ModuleOpDrawer.vue';
import {
  ChoyBadge,
  ChoyBooleanField,
  ChoyButton,
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

const {
  drawerOpen,
  drawerOperationId,
  drawerModuleName,
  drawerAction,
  drawerRegistryNewerThanLocal,
  syncLoading,
  lastSyncAtLabel,
  onSync,
  onRowClick,
  onPrimaryAction,
  canUninstall,
  onUninstall,
  hasPendingDemo,
  onLoadDemo,
  refreshCatalog,
  openDrawer,
  closeDrawer,
  statusBadgeVariant,
  statusLabel,
  primaryActionFor,
  primaryActionLabel,
  bootFromRoute,
} = useModuleCatalogViewModel({
  store,
  route,
  router,
  initialOperationId: () => String(props.initialOperationId || ''),
  canRequestSync: () => hasAction(moduleSyncAction),
  msgs: {
    unknown: _t('Unknown'),
    install: _t('Install'),
    upgrade: _t('Upgrade'),
    progress: _t('View Progress'),
    installed: _t('Installed'),
    uninstalled: _t('Not Installed'),
    unavailable: _t('Unavailable'),
    syncTriggered: id => _t('Sync job triggered: %s', id),
    syncTriggeredEmpty: _t('Sync job triggered'),
    syncFailed: msg => _t('Sync failed: %s', msg),
  },
});

function onDrawerSettled() {
  void refreshCatalog();
}

onMounted(() => {
  bootFromRoute();
});

const { listRef, expose } = useListViewExpose<MetaModuleCatalog>();
defineExpose({
  ...expose,
  openDrawer,
  closeDrawer,
});
</script>
