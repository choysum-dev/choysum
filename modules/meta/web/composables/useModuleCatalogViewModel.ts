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
  formatModuleCatalogDate,
  moduleStatusBadgeVariant,
  resolveModuleCatalogPrimaryAction,
  type CatalogSyncResultSummary,
} from '../views/module_catalog_chrome';

export type ModuleCatalogViewMsgs = {
  unknown: string;
  install: string;
  upgrade: string;
  progress: string;
  installed: string;
  updateAvailable: string;
  uninstalled: string;
  unavailable: string;
  syncFailed: (message: string) => string;
  syncSettledOk: (summary: string) => string;
  syncSettledPartial: (summary: string) => string;
  syncSettledFailed: (message: string) => string;
  syncJobStillRunning: string;
  syncServiceRestarting: string;
  syncFailedToGetStatus: string;
  formatSyncSummaryOk: (sync: CatalogSyncResultSummary) => string;
  formatSyncSummaryPartial: (sync: CatalogSyncResultSummary) => string;
};

export type ModuleCatalogDrawerAction =
  | ''
  | 'install'
  | 'upgrade'
  | 'uninstall'
  | 'progress'
  | 'sync'
  | 'load_demo';

function isOperationDeepLinkRoute(routeName: unknown): boolean {
  return String(routeName || '') === 'MetaModuleOperationDeepLink';
}

/**
 * Catalog view model: sync (button busy + toast), drawer/deep-link, and row action helpers.
 */
export function useModuleCatalogViewModel(opts: {
  store: WebModelStore<any>;
  operationStore?: WebModelStore<any> | Ref<WebModelStore<any> | null | undefined> | null;
  route: RouteLocationNormalizedLoaded;
  router: Router;
  initialOperationId?: Ref<string> | (() => string);
  msgs: ModuleCatalogViewMsgs;
}) {
  const drawerOpen = ref(false);
  const drawerOperationId = ref('');
  const drawerModuleName = ref('');
  const drawerAction = ref<ModuleCatalogDrawerAction>('');
  const drawerRegistryNewerThanLocal = ref(false);
  // True when the open drawer was bound from the operation deep-link (or boot prop).
  const drawerBoundToRoute = ref(false);

  async function refreshCatalog() {
    try {
      const list = (opts.store as any)?.state?.list;
      if (list && typeof list.refresh === 'function') {
        await list.refresh();
        return;
      }
      if (typeof (opts.store as any)?.Search === 'function') {
        await (opts.store as any).Search();
      }
    } catch {
      // Refresh is best-effort after an operation settles.
    }
  }

  const { syncBusy, syncLoading, onSync, stop: stopSync } = useModuleCatalogSync(
    opts.store,
    opts.operationStore,
    {
      failed: opts.msgs.syncFailed,
      settledOk: opts.msgs.syncSettledOk,
      settledPartial: opts.msgs.syncSettledPartial,
      settledFailed: opts.msgs.syncSettledFailed,
      jobStillRunning: opts.msgs.syncJobStillRunning,
      serviceRestarting: opts.msgs.syncServiceRestarting,
      failedToGetStatus: opts.msgs.syncFailedToGetStatus,
      formatSummaryOk: opts.msgs.formatSyncSummaryOk,
      formatSummaryPartial: opts.msgs.formatSyncSummaryPartial,
    },
    { refreshCatalog },
  );

  function rowRecord(row: unknown): Record<string, unknown> {
    const rec = unwrapListRecord(row);
    return rec && typeof rec === 'object' ? (rec as Record<string, unknown>) : {};
  }

  function field(row: unknown, key: string): string {
    const val = rowRecord(row)[key];
    return val == null ? '' : String(val);
  }

  function statusBadgeVariant(row: unknown) {
    const rec = rowRecord(row);
    return moduleStatusBadgeVariant(
      catalogInstalledStatus(rec),
      rec.Available as boolean | undefined,
      rec.UpdateAvailable === true,
    );
  }

  function statusLabel(row: unknown) {
    const rec = rowRecord(row);
    if (rec.Available === false) return opts.msgs.unavailable;
    const val = catalogInstalledStatus(rec).toLowerCase();
    if (val === 'installed') {
      if (rec.UpdateAvailable === true) return opts.msgs.updateAvailable;
      return opts.msgs.installed;
    }
    if (val === 'uninstalled') return opts.msgs.uninstalled;
    return catalogInstalledStatus(rec) || opts.msgs.unknown;
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

  function openDrawer(
    next: {
      operationId?: string;
      moduleName?: string;
      action?: ModuleCatalogDrawerAction;
      registryNewerThanLocal?: boolean;
    },
    boundToRoute = false,
  ) {
    drawerOperationId.value = String(next.operationId || '').trim();
    drawerModuleName.value = String(next.moduleName || '').trim();
    drawerAction.value = next.action || (drawerOperationId.value ? 'progress' : '');
    drawerRegistryNewerThanLocal.value = !!next.registryNewerThanLocal;
    drawerBoundToRoute.value = boundToRoute;
    drawerOpen.value = true;
  }

  function closeDrawer() {
    drawerOpen.value = false;
    drawerOperationId.value = '';
    drawerModuleName.value = '';
    drawerAction.value = '';
    drawerRegistryNewerThanLocal.value = false;
    drawerBoundToRoute.value = false;
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
      registryNewerThanLocal: !!rec.RegistryNewerThanLocal,
    });
  }

  function canUninstall(row: unknown): boolean {
    const rec = rowRecord(row);
    if (String(rec.ActiveOperationId || '').trim()) return false;
    if (rec.Available === false) return false;
    return rec.Installed === true;
  }

  function onUninstall(row: unknown) {
    if (!canUninstall(row)) return;
    const rec = rowRecord(row);
    openDrawer({
      moduleName: String(rec.ModuleName || ''),
      action: 'uninstall',
    });
  }

  const allRows = computed(() => {
    const rows = ((opts.store as any)?.state?.result?.rows as unknown[]) || [];
    return Array.isArray(rows) ? rows : [];
  });

  /** Catalog rows that still need demo import. */
  const pendingDemoRows = computed(() =>
    allRows.value.filter(row => {
      const rec = rowRecord(row);
      return rec.Installed === true && !!rec.HasDemo && !rec.DemoLoaded;
    }),
  );

  const hasPendingDemo = computed(() => pendingDemoRows.value.length > 0);

  function onLoadDemo() {
    if (!hasPendingDemo.value) return;
    openDrawer({ action: 'load_demo', moduleName: '' });
  }

  const lastSyncAtLabel = computed(() => {
    let bestRaw: unknown = null;
    let bestTime = -Infinity;
    for (const row of allRows.value) {
      const raw = rowRecord(row).LastSyncAt;
      if (raw == null || raw === '') continue;
      const time =
        typeof raw === 'number'
          ? raw
          : raw instanceof Date
            ? raw.getTime()
            : new Date(raw as any).getTime();
      if (!Number.isFinite(time) || time <= bestTime) continue;
      bestTime = time;
      bestRaw = raw;
    }
    return bestRaw != null ? formatModuleCatalogDate(bestRaw) : '';
  });

  /** Install/upgrade/uninstall drawer settle — refresh only (sync uses its own toast path). */
  function onDrawerSettled() {
    void refreshCatalog();
  }

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
      // Only clear a drawer that was opened from the deep-link; user-opened shells stay put.
      if (drawerOpen.value && drawerBoundToRoute.value) closeDrawer();
      return;
    }
    const opId = String(opts.route.params.id || '').trim();
    if (opId) openDrawer({ operationId: opId, action: 'progress' }, true);
    else closeDrawer();
  }

  function bootFromRoute() {
    const opId = resolveInitialOperationId();
    if (opId) openDrawer({ operationId: opId, action: 'progress' }, true);
  }

  watch(
    () => [String(opts.route.name || ''), String(opts.route.params.id || '')] as const,
    () => {
      syncDrawerFromRoute();
    },
  );

  return {
    drawerOpen,
    drawerOperationId,
    drawerModuleName,
    drawerAction,
    drawerRegistryNewerThanLocal,
    syncLoading,
    syncBusy,
    lastSyncAtLabel,
    onSync,
    stopSync,
    onRowClick,
    onPrimaryAction,
    canUninstall,
    onUninstall,
    hasPendingDemo,
    pendingDemoRows,
    onLoadDemo,
    refreshCatalog,
    onDrawerSettled,
    openDrawer,
    closeDrawer,
    toShow,
    field,
    rowRecord,
    statusBadgeVariant,
    statusLabel,
    primaryActionFor,
    primaryActionLabel,
    bootFromRoute,
    syncDrawerFromRoute,
    resolveInitialOperationId,
  };
}
