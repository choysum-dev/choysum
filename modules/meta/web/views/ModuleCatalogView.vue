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
        <ChoyViewSwitcher
          v-if="viewSwitcherOptions.length"
          :aria-label="_t('View switcher')"
          :active="viewMode"
          :options="viewSwitcherOptions"
          @select="onViewSelect"
        />
      </div>
      <div class="flex flex-wrap gap-1" role="tablist" :aria-label="_t('Filter')">
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
          :aria-selected="filter === opt.value"
          @click="filter = opt.value"
        >
          {{ opt.label }}
        </button>
      </div>
    </div>

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
      <ChoyTableColumn :label="_t('Action')" :vColumnProps="{ minWidth: 120 }">
        <template #default="{ row }">
          <button
            v-if="primaryActionFor(row)"
            type="button"
            class="rounded-md border border-border px-2 py-0.5 text-xs hover:bg-muted"
            :data-testid="`module-catalog-primary-${primaryActionFor(row)}`"
            @click.stop="onPrimaryAction(row)"
          >
            {{ primaryActionLabel(primaryActionFor(row)) }}
          </button>
        </template>
      </ChoyTableColumn>
    </ChoyListView>

    <div
      v-show="viewMode === 'cards'"
      class="module-catalog-cards grid min-h-0 flex-1 content-start gap-3 overflow-auto p-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
      data-testid="module-catalog-cards"
      data-anchor="choy.module-catalog-cards"
    >
      <button
        v-for="(row, index) in filteredCardRows"
        :key="cardKey(row, index)"
        type="button"
        class="module-catalog-card flex flex-col gap-1.5 rounded-lg border border-border bg-card p-3 text-start text-xs shadow-xs hover:bg-muted/40"
        data-testid="module-catalog-card"
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
        <div class="mt-auto flex items-center justify-between gap-2 pt-1">
          <span class="truncate text-muted-foreground/80">{{ field(row, 'Application') || '—' }}</span>
          <span
            v-if="primaryActionFor(row)"
            class="rounded-md border border-border px-2 py-0.5 text-[11px]"
            @click.stop="onPrimaryAction(row)"
          >
            {{ primaryActionLabel(primaryActionFor(row)) }}
          </span>
        </div>
      </button>
      <p
        v-if="filteredCardRows.length === 0"
        class="col-span-full p-6 text-center text-sm text-muted-foreground"
      >
        {{ _t('No modules match this filter.') }}
      </p>
    </div>

    <ModuleOpDrawer
      v-model:open="drawerOpen"
      :operation-id="drawerOperationId"
      :module-name="drawerModuleName"
      :action="drawerAction"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { FolderSync } from 'lucide-vue-next';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type MetaModuleCatalog from '@/meta/service/models/module_catalog';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { unwrapListRecord } from '@/web/web/composables/listRowEdit';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { useModuleCatalogSync } from '../composables/useModuleCatalogSync';
import { moduleSyncAction } from './module_actions';
import ModuleOpDrawer from './ModuleOpDrawer.vue';
import {
  catalogInstalledStatus,
  matchesModuleCatalogFilter,
  moduleCatalogCardBadgeVariant,
  moduleStatusBadgeVariant,
  resolveModuleCatalogPrimaryAction,
  type ModuleCatalogFilter,
} from './module_catalog_chrome';
import {
  ChoyBadge,
  ChoyBooleanField,
  ChoyListView,
  ChoySearchView,
  ChoyTableColumn,
  ChoyVarcharField,
  ChoyViewSwitcher,
  ChoyVirtualField,
  type ChoyViewSwitcherOption,
  type RowEventPayload,
} from '@/web';

defineOptions({ name: 'ModuleCatalogView', inheritAttrs: false });

const { _t } = createTranslate('meta', { scope: 'web/views/ModuleCatalogView' });
const router = useRouter();
const route = useRoute();
const { hasAction } = usePermission();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<MetaModuleCatalog>;
    showHeader?: boolean;
    initialOperationId?: string;
  }>(),
  { showHeader: true },
);

const store = resolvePageStore(props.store, 'ModuleCatalogView');
const { showHeader } = props;

const filter = ref<ModuleCatalogFilter>('all');
const viewMode = ref<'cards' | 'table'>('cards');
const drawerOpen = ref(false);
const drawerOperationId = ref('');
const drawerModuleName = ref('');
const drawerAction = ref<'' | 'install' | 'upgrade' | 'uninstall' | 'progress'>('');

const filterOptions = computed(() => [
  { value: 'all' as const, label: _t('All') },
  { value: 'installed' as const, label: _t('Installed') },
  { value: 'upgradeable' as const, label: _t('Upgradeable') },
  { value: 'uninstalled' as const, label: _t('Not Installed') },
  { value: 'unavailable' as const, label: _t('Unavailable') },
]);

const viewSwitcherOptions = computed<ChoyViewSwitcherOption[]>(() => [
  { value: 'cards', label: _t('Cards') },
  { value: 'table', label: _t('Table') },
]);

const { syncLoading, requestSync, watchEmptyCatalogAutoSync } = useModuleCatalogSync(store, {
  triggered: id => _t('Sync job triggered: %s', id),
  triggeredEmpty: _t('Sync job triggered'),
  failed: msg => _t('Sync failed: %s', msg),
});
watchEmptyCatalogAutoSync();

function onSync() {
  void requestSync({ Force: true, IfStale: false });
}

function onViewSelect(value: string) {
  viewMode.value = value === 'table' ? 'table' : 'cards';
  const nextQuery = { ...route.query, view: viewMode.value };
  void router.replace({ query: nextQuery });
}

watch(
  () => String(route.query.view || ''),
  view => {
    if (view === 'table' || view === 'cards') viewMode.value = view;
  },
  { immediate: true },
);

function rowRecord(row: unknown): Record<string, unknown> {
  const rec = unwrapListRecord(row);
  return rec && typeof rec === 'object' ? (rec as Record<string, unknown>) : {};
}

function field(row: unknown, key: string): string {
  const val = rowRecord(row)[key];
  return val == null ? '' : String(val);
}

function cardKey(row: unknown, index: number): string {
  return field(row, 'Id') || field(row, 'ModuleName') || `row-${index}`;
}

function statusBadgeVariant(row: unknown) {
  const rec = rowRecord(row);
  return moduleStatusBadgeVariant(catalogInstalledStatus(rec), rec.Available as boolean | undefined);
}

function cardBadgeVariant(row: unknown) {
  const rec = rowRecord(row);
  return moduleCatalogCardBadgeVariant(catalogInstalledStatus(rec), rec.Available as boolean | undefined);
}

function statusLabel(row: unknown) {
  const rec = rowRecord(row);
  if (rec.Available === false) return _t('Unavailable');
  const val = catalogInstalledStatus(rec).toLowerCase();
  if (val === 'installed') return _t('Installed');
  if (val === 'uninstalled') return _t('Not Installed');
  return catalogInstalledStatus(rec) || _t('Unknown');
}

function versionLine(row: unknown): string {
  const rec = rowRecord(row);
  const installed = String(rec.InstalledVersion || '').trim();
  const catalog = String(rec.LocalVersion || rec.RegistryVersion || '').trim();
  if (rec.Installed === true && rec.UpdateAvailable === true && installed && catalog) {
    return `${installed} → ${catalog}`;
  }
  if (rec.Installed === true) return installed || catalog || '—';
  return catalog || '—';
}

function primaryActionFor(row: unknown) {
  return resolveModuleCatalogPrimaryAction(rowRecord(row));
}

function primaryActionLabel(action: 'install' | 'upgrade' | 'progress' | null) {
  if (action === 'install') return _t('Install');
  if (action === 'upgrade') return _t('Upgrade');
  if (action === 'progress') return _t('View Progress');
  return '';
}

function toShow(row: unknown) {
  const name = field(row, 'ModuleName').trim();
  if (name) void router.push(`/meta/modules/${encodeURIComponent(name)}`);
}

function onRowClick(payload: RowEventPayload) {
  toShow(payload.row);
}

function openDrawer(opts: {
  operationId?: string;
  moduleName?: string;
  action?: '' | 'install' | 'upgrade' | 'uninstall' | 'progress';
}) {
  drawerOperationId.value = String(opts.operationId || '').trim();
  drawerModuleName.value = String(opts.moduleName || '').trim();
  drawerAction.value = opts.action || (drawerOperationId.value ? 'progress' : '');
  drawerOpen.value = true;
}

function onPrimaryAction(row: unknown) {
  const rec = rowRecord(row);
  const action = resolveModuleCatalogPrimaryAction(rec);
  if (!action) return;
  if (action === 'progress') {
    openDrawer({
      operationId: String(rec.ActiveOperationId || '').trim(),
      moduleName: String(rec.ModuleName || ''),
      action: 'progress',
    });
    return;
  }
  // Plan+Apply wiring lands in W3-2; open the drawer shell with the intended action.
  openDrawer({
    moduleName: String(rec.ModuleName || ''),
    action,
  });
}

const allRows = computed(() => {
  const rows = ((store as any)?.state?.result?.rows as unknown[]) || [];
  return Array.isArray(rows) ? rows : [];
});

const filteredCardRows = computed(() =>
  allRows.value.filter(row => matchesModuleCatalogFilter(rowRecord(row), filter.value)),
);

onMounted(() => {
  const opId = String(props.initialOperationId || route.params.id || '').trim();
  if (opId) openDrawer({ operationId: opId, action: 'progress' });
});

watch(
  () => String(route.params.id || ''),
  id => {
    const opId = id.trim();
    if (opId && String(route.name || '').includes('Operation')) {
      openDrawer({ operationId: opId, action: 'progress' });
    }
  },
);

const { listRef, expose } = useListViewExpose<MetaModuleCatalog>();
defineExpose({
  ...expose,
  openDrawer,
});
</script>
