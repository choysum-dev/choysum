// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref, watch, type Ref } from 'vue';
import type { Router, RouteLocationNormalizedLoaded } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { unwrapListRecord } from '@/web/web/composables/listRowEdit';
import type { RowEventPayload } from '@/web';
import { useModuleCatalogSync } from './useModuleCatalogSync';
import {
  catalogInstalledStatus,
  matchesModuleCatalogFilter,
  moduleCatalogCardBadgeVariant,
  moduleStatusBadgeVariant,
  resolveModuleCatalogPrimaryAction,
  type ModuleCatalogFilter,
} from '../views/module_catalog_chrome';

export type ModuleCatalogViewMsgs = {
  all: string;
  installed: string;
  upgradeable: string;
  uninstalled: string;
  unavailable: string;
  cards: string;
  table: string;
  unknown: string;
  install: string;
  upgrade: string;
  progress: string;
  syncTriggered: (operationId: string) => string;
  syncTriggeredEmpty: string;
  syncFailed: (message: string) => string;
};

export type ModuleCatalogDrawerAction = '' | 'install' | 'upgrade' | 'uninstall' | 'progress';

function isOperationDeepLinkRoute(routeName: unknown): boolean {
  return String(routeName || '') === 'MetaModuleOperationDeepLink';
}

/**
 * Catalog view model: filters, cards/table mode, sync, and operation drawer shell state.
 */
export function useModuleCatalogViewModel(opts: {
  store: WebModelStore<any>;
  route: RouteLocationNormalizedLoaded;
  router: Router;
  initialOperationId?: Ref<string> | (() => string);
  msgs: ModuleCatalogViewMsgs;
}) {
  const filter = ref<ModuleCatalogFilter>('all');
  const viewMode = ref<'cards' | 'table'>('cards');
  const drawerOpen = ref(false);
  const drawerOperationId = ref('');
  const drawerModuleName = ref('');
  const drawerAction = ref<ModuleCatalogDrawerAction>('');

  const filterOptions = computed(() => [
    { value: 'all' as const, label: opts.msgs.all },
    { value: 'installed' as const, label: opts.msgs.installed },
    { value: 'upgradeable' as const, label: opts.msgs.upgradeable },
    { value: 'uninstalled' as const, label: opts.msgs.uninstalled },
    { value: 'unavailable' as const, label: opts.msgs.unavailable },
  ]);

  const viewSwitcherOptions = computed(() => [
    { value: 'cards', label: opts.msgs.cards },
    { value: 'table', label: opts.msgs.table },
  ]);

  const { syncLoading, requestSync, watchEmptyCatalogAutoSync } = useModuleCatalogSync(opts.store, {
    triggered: opts.msgs.syncTriggered,
    triggeredEmpty: opts.msgs.syncTriggeredEmpty,
    failed: opts.msgs.syncFailed,
  });
  watchEmptyCatalogAutoSync();

  function onSync() {
    void requestSync({ Force: true, IfStale: false });
  }

  function onViewSelect(value: string) {
    viewMode.value = value === 'table' ? 'table' : 'cards';
    const nextQuery = { ...opts.route.query, view: viewMode.value };
    void opts.router.replace({ query: nextQuery });
  }

  watch(
    () => String(opts.route.query.view || ''),
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
    if (rec.Available === false) return opts.msgs.unavailable;
    const val = catalogInstalledStatus(rec).toLowerCase();
    if (val === 'installed') return opts.msgs.installed;
    if (val === 'uninstalled') return opts.msgs.uninstalled;
    return catalogInstalledStatus(rec) || opts.msgs.unknown;
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
    if (action === 'install') return opts.msgs.install;
    if (action === 'upgrade') return opts.msgs.upgrade;
    if (action === 'progress') return opts.msgs.progress;
    return '';
  }

  function toShow(row: unknown) {
    const name = field(row, 'ModuleName').trim();
    if (name) void opts.router.push(`/meta/modules/${encodeURIComponent(name)}`);
  }

  function onRowClick(payload: RowEventPayload) {
    toShow(payload.row);
  }

  function openDrawer(next: {
    operationId?: string;
    moduleName?: string;
    action?: ModuleCatalogDrawerAction;
  }) {
    drawerOperationId.value = String(next.operationId || '').trim();
    drawerModuleName.value = String(next.moduleName || '').trim();
    drawerAction.value = next.action || (drawerOperationId.value ? 'progress' : '');
    drawerOpen.value = true;
  }

  function closeDrawer() {
    drawerOpen.value = false;
    drawerOperationId.value = '';
    drawerModuleName.value = '';
    drawerAction.value = '';
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
    openDrawer({
      moduleName: String(rec.ModuleName || ''),
      action,
    });
  }

  const allRows = computed(() => {
    const rows = ((opts.store as any)?.state?.result?.rows as unknown[]) || [];
    return Array.isArray(rows) ? rows : [];
  });

  const filteredCardRows = computed(() =>
    allRows.value.filter(row => matchesModuleCatalogFilter(rowRecord(row), filter.value)),
  );

  function resolveInitialOperationId(): string {
    const fromProp =
      typeof opts.initialOperationId === 'function'
        ? opts.initialOperationId()
        : opts.initialOperationId?.value;
    const fromPropTrimmed = String(fromProp || '').trim();
    if (fromPropTrimmed) return fromPropTrimmed;
    if (!isOperationDeepLinkRoute(opts.route.name)) return '';
    return String(opts.route.params.id || '').trim();
  }

  function syncDrawerFromRoute() {
    if (!isOperationDeepLinkRoute(opts.route.name)) {
      if (drawerOpen.value && drawerAction.value === 'progress' && drawerOperationId.value) {
        closeDrawer();
      }
      return;
    }
    const opId = String(opts.route.params.id || '').trim();
    if (opId) openDrawer({ operationId: opId, action: 'progress' });
    else closeDrawer();
  }

  function bootFromRoute() {
    const opId = resolveInitialOperationId();
    if (opId) openDrawer({ operationId: opId, action: 'progress' });
  }

  watch(
    () => [String(opts.route.name || ''), String(opts.route.params.id || '')] as const,
    () => {
      syncDrawerFromRoute();
    },
  );

  return {
    filter,
    viewMode,
    drawerOpen,
    drawerOperationId,
    drawerModuleName,
    drawerAction,
    filterOptions,
    viewSwitcherOptions,
    syncLoading,
    filteredCardRows,
    onSync,
    onViewSelect,
    onRowClick,
    onPrimaryAction,
    openDrawer,
    closeDrawer,
    toShow,
    field,
    cardKey,
    statusBadgeVariant,
    cardBadgeVariant,
    statusLabel,
    versionLine,
    primaryActionFor,
    primaryActionLabel,
    bootFromRoute,
    syncDrawerFromRoute,
    resolveInitialOperationId,
  };
}
