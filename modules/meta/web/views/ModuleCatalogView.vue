<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="module-catalog-view flex h-full min-h-0 min-w-0 flex-1 flex-col" data-testid="module-catalog-view">
    <div
      v-if="showHeader"
      class="mb-3 flex flex-col gap-2"
      data-testid="module-catalog-toolbar"
    >
      <div class="flex flex-wrap items-center gap-2">
        <ChoySearchView class="min-w-0 flex-1" :store="store" />
        <button
          v-if="hasAction(moduleSyncAction)"
          type="button"
          class="inline-flex items-center gap-1.5 rounded-md border border-border px-2.5 py-1 text-xs hover:bg-muted"
          data-testid="module-catalog-sync"
          :disabled="syncLoading"
          @click="onSync"
        >
          <FolderSync class="size-3.5" :class="{ 'animate-spin': syncLoading }" aria-hidden="true" />
          {{ _t('Sync Modules') }}
        </button>
        <button
          v-if="hasAction(moduleLoadDemoAction) && hasPendingDemo"
          type="button"
          class="inline-flex items-center gap-1.5 rounded-md border border-border px-2.5 py-1 text-xs hover:bg-muted"
          data-testid="module-catalog-load-demo"
          @click="onLoadDemo"
        >
          {{ _t('Load Demo Data') }}
        </button>
        <span
          v-if="lastSyncAtLabel"
          class="text-[11px] text-muted-foreground"
          data-testid="module-catalog-last-sync"
        >
          {{ _t('Last sync %s', lastSyncAtLabel) }}
        </span>
        <ChoyViewSwitcher
          v-if="viewSwitcherOptions.length"
          :aria-label="_t('View switcher')"
          :active="viewMode"
          :options="viewSwitcherOptions"
          @select="onViewSelect"
        />
      </div>
      <!-- Filters apply to cards only; table mode uses ChoySearchView / store query. -->
      <div
        v-if="viewMode === 'cards'"
        class="flex flex-wrap gap-1"
        role="group"
        :aria-label="_t('Filter')"
      >
        <button
          v-for="opt in filterOptions"
          :key="opt.value"
          type="button"
          class="rounded-md border px-2.5 py-1 text-xs"
          :class="
            filter === opt.value
              ? 'border-foreground bg-foreground text-background'
              : 'border-border bg-background text-foreground hover:bg-muted'
          "
          :data-testid="`module-catalog-filter-${opt.value}`"
          :aria-pressed="filter === opt.value"
          @click="filter = opt.value"
        >
          {{ opt.label }}
        </button>
      </div>
    </div>

    <!-- ChoyVirtualField registers Installed for search without rendering a column. -->
    <ChoyListView
      v-show="viewMode === 'table'"
      ref="listRef"
      class="module-catalog-table min-h-0 flex-1"
      v-bind="$attrs"
      :store="store"
      :show-header="false"
      :has-action="hasAction"
      data-testid="module-catalog-table"
      @row-click="onRowClick"
    >
      <ChoyTableColumn type="index" :vColumnProps="{ align: 'right' }" />
      <ChoyVarcharField prop="ModuleName" :store="store" :vColumnProps="{ minWidth: 160 }" />
      <ChoyVarcharField prop="Summary" :store="store" :vColumnProps="{ minWidth: 220 }" />
      <ChoyVarcharField prop="InstalledVersion" :label="_t('Installed')" :store="store" :vColumnProps="{ minWidth: 110 }" />
      <ChoyVarcharField prop="LocalVersion" :label="_t('Local')" :store="store" :vColumnProps="{ minWidth: 110 }" />
      <ChoyVarcharField prop="RegistryVersion" :label="_t('Registry')" :store="store" :vColumnProps="{ minWidth: 110 }" />
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

    <div
      v-show="viewMode === 'cards'"
      class="module-catalog-cards grid min-h-0 flex-1 content-start gap-3 overflow-auto p-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
      data-testid="module-catalog-cards"
      data-anchor="choy.module-catalog-cards"
    >
      <div
        v-for="(row, index) in filteredCardRows"
        :key="cardKey(row, index)"
        class="module-catalog-card flex flex-col gap-1.5 rounded-lg border border-border bg-card p-3 text-start text-xs shadow-xs"
        data-testid="module-catalog-card"
      >
        <button
          type="button"
          class="flex flex-col gap-1.5 text-start hover:bg-muted/40"
          @click="toShow(row)"
        >
          <div class="module-catalog-card__title flex items-center justify-between gap-2">
            <span class="name truncate text-sm font-medium text-foreground">{{ field(row, 'ModuleName') }}</span>
            <ChoyBadge
              :variant="cardBadgeVariant(row)"
              class="module-catalog-card__status shrink-0 px-1.5 py-0 text-[10px] font-normal"
            >
              {{ statusLabel(row) }}
            </ChoyBadge>
          </div>
          <div class="truncate text-muted-foreground">
            {{ versionLine(row) }}
          </div>
          <div v-if="field(row, 'Summary')" class="line-clamp-2 text-muted-foreground">
            {{ field(row, 'Summary') }}
          </div>
          <div class="truncate text-muted-foreground/80">{{ field(row, 'Application') || '—' }}</div>
        </button>
        <div class="mt-auto flex flex-wrap items-center justify-end gap-1">
          <button
            v-if="canUninstall(row) && hasAction(moduleUninstallAction)"
            type="button"
            class="rounded-md border border-border px-2 py-0.5 text-[11px] text-muted-foreground hover:bg-muted"
            data-testid="module-catalog-card-uninstall"
            @click="onUninstall(row)"
          >
            {{ _t('Uninstall') }}
          </button>
          <button
            v-if="primaryActionFor(row)"
            type="button"
            class="rounded-md border border-border px-2 py-0.5 text-[11px] hover:bg-muted"
            :data-testid="`module-catalog-card-primary-${primaryActionFor(row)}`"
            @click="onPrimaryAction(row)"
          >
            {{ primaryActionLabel(primaryActionFor(row)) }}
          </button>
        </div>
      </div>
      <p
        v-if="filteredCardRows.length === 0"
        class="col-span-full p-6 text-center text-sm text-muted-foreground"
      >
        {{ _t('No modules match this filter.') }}
      </p>
    </div>

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
import { FolderSync } from 'lucide-vue-next';
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
  ChoyListView,
  ChoySearchView,
  ChoyTableColumn,
  ChoyVarcharField,
  ChoyViewSwitcher,
  ChoyVirtualField,
} from '@/web';

defineOptions({ name: 'ModuleCatalogView', inheritAttrs: false });

const { _t } = createTranslate('meta', { scope: 'web/views/ModuleCatalogView' });
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

const store = resolvePageStore(props.store, 'ModuleCatalogView');
const showHeader = toRef(props, 'showHeader');
const operationStore = computed(() => props.operationStore ?? null);

const {
  filter,
  viewMode,
  drawerOpen,
  drawerOperationId,
  drawerModuleName,
  drawerAction,
  drawerRegistryNewerThanLocal,
  filterOptions,
  viewSwitcherOptions,
  syncLoading,
  filteredCardRows,
  lastSyncAtLabel,
  onSync,
  onViewSelect,
  onRowClick,
  onPrimaryAction,
  canUninstall,
  onUninstall,
  hasPendingDemo,
  onLoadDemo,
  refreshCatalog,
  toShow,
  field,
  cardKey,
  statusBadgeVariant,
  cardBadgeVariant,
  statusLabel,
  versionLine,
  primaryActionFor,
  primaryActionLabel,
  openDrawer,
  closeDrawer,
  bootFromRoute,
} = useModuleCatalogViewModel({
  store,
  route,
  router,
  initialOperationId: () => String(props.initialOperationId || ''),
  canRequestSync: () => hasAction(moduleSyncAction),
  msgs: {
    all: _t('All'),
    installed: _t('Installed'),
    upgradeable: _t('Upgradeable'),
    uninstalled: _t('Not Installed'),
    unavailable: _t('Unavailable'),
    cards: _t('Cards'),
    table: _t('Table'),
    unknown: _t('Unknown'),
    install: _t('Install'),
    upgrade: _t('Upgrade'),
    progress: _t('View Progress'),
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
