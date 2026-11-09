<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Module show is read-only: omit mutating action-ids so Edit/Copy/Delete stay hidden. -->
  <div class="module-catalog-form-view flex min-h-0 flex-1 flex-col gap-3" data-testid="module-catalog-form-view">
    <div v-if="showHeaderActions" class="flex flex-wrap items-center gap-2" data-testid="module-catalog-form-actions">
      <ChoyButton
        v-if="primaryAction"
        type="button"
        size="sm"
        :variant="primaryButtonVariant"
        :data-testid="`module-catalog-form-primary-${primaryAction}`"
        @click="onPrimaryClick"
      >
        {{ primaryActionLabel }}
      </ChoyButton>
      <ChoyButton
        v-if="canLoadDemo && hasAction(moduleLoadDemoAction)"
        type="button"
        size="sm"
        variant="outline"
        data-testid="module-catalog-form-load-demo"
        @click="onLoadDemoClick"
      >
        {{ _t('Load Demo Data') }}
      </ChoyButton>
      <ChoyButton
        v-if="canUninstallRow && hasAction(moduleUninstallAction)"
        type="button"
        size="sm"
        variant="outline-destructive"
        data-testid="module-catalog-form-uninstall"
        @click="onUninstallClick"
      >
        {{ _t('Uninstall') }}
      </ChoyButton>
    </div>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode: 'display' as const, showHeader }"
  >
    <ChoyCard :title="_t('Overview')" class="mb-3.5">
      <ChoyGrid :cols="12">
        <ChoyCol :span="4">
          <ChoyVarcharField :store="store" prop="ModuleName" />
        </ChoyCol>
        <ChoyCol :span="4">
          <ChoyVarcharField :store="store" prop="Application" />
        </ChoyCol>
        <ChoyCol :span="4">
          <ChoyVarcharField :store="store" prop="Category" />
        </ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="12">
          <ChoyTextField :store="store" prop="Summary" />
        </ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="Author" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyTextField :store="store" prop="License" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>

    <ChoyCard :title="_t('Dependencies')" class="mb-3.5">
      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyJsonField :store="store" prop="DependsJson" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyJsonField :store="store" prop="DependentsJson" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>

    <ChoyCard :title="_t('Versions & Source')" class="mb-3.5">
      <ChoyGrid :cols="12">
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="InstalledVersion" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="LocalVersion" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="RegistryVersion" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="LocalPath" />
        </ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="3">
          <ChoyBooleanField :store="store" prop="Installed" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyBooleanField :store="store" prop="UpdateAvailable" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyBooleanField :store="store" prop="Available" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyTextField :store="store" prop="UnavailableReason" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>

    <ChoyCard :title="_t('Sync')" class="mb-3.5">
      <ChoyGrid :cols="12">
        <ChoyCol :span="4">
          <ChoyDatetimeField :store="store" prop="LastSyncAt" />
        </ChoyCol>
        <ChoyCol :span="4">
          <ChoyVarcharField :store="store" prop="ActiveOperationId" />
        </ChoyCol>
        <ChoyCol :span="2">
          <ChoyBooleanField :store="store" prop="HasDemo" />
        </ChoyCol>
        <ChoyCol :span="2">
          <ChoyBooleanField :store="store" prop="DemoLoaded" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>
  </ChoyFormView>

  <ModuleCatalogOpDialog
    v-model:open="drawerOpen"
    :operation-store="operationStore"
    :operation-id="drawerOperationId"
    :module-name="drawerModuleName"
    :action="drawerAction"
    :registry-newer-than-local="drawerRegistryNewer"
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
import { computed, ref, toRef } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type MetaModuleCatalog from '@/meta/service/models/module_catalog';
import type MetaModuleOperation from '@/meta/service/models/module_operation';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { useModuleCatalogOpProgress } from '../composables/useModuleCatalogOpProgress';
import { resolveModuleCatalogFormPrimaryAction } from './module_catalog_chrome';
import {
  moduleInstallAction,
  moduleLoadDemoAction,
  moduleUpgradeAction,
  moduleUninstallAction,
} from './module_catalog_actions';
import ModuleCatalogOpDialog from './ModuleCatalogOpDialog.vue';
import ModuleCatalogOpProgressStrip from './ModuleCatalogOpProgressStrip.vue';
import {
  ChoyBooleanField,
  ChoyButton,
  ChoyCard,
  ChoyCol,
  ChoyDatetimeField,
  ChoyFormView,
  ChoyGrid,
  ChoyJsonField,
  ChoyTextField,
  ChoyVarcharField,
} from '@/web';

defineOptions({ name: 'ModuleCatalogFormView', inheritAttrs: true });

const { _t } = createTranslate('meta', { scope: 'web/views/ModuleCatalogFormView' });
const { hasAction } = usePermission();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<MetaModuleCatalog>;
    operationStore?: WebModelStore<MetaModuleOperation> | null;
    recordId?: string;
    showHeader?: boolean;
  }>(),
  { showHeader: true, operationStore: null },
);

const store = resolvePageStore(props.store, 'ModuleCatalogFormView');
const recordId = toRef(props, 'recordId');
const showHeader = toRef(props, 'showHeader');
// Prefer the prop binding in the template (`operationStore`) so we pass the
// concrete Pinia store — wrapping it in computed() can leave a ComputedRef
// unwrapped and break session.store.Plan().

const drawerOpen = ref(false);
const drawerOperationId = ref('');
const drawerModuleName = ref('');
const drawerAction = ref<'install' | 'upgrade' | 'uninstall' | 'progress' | 'load_demo' | ''>('');
const drawerRegistryNewer = ref(false);

const currentRow = computed(() => {
  const id = String(recordId.value || '').trim();
  const records = ((store as any)?.state?.records || {}) as Record<string, Record<string, unknown>>;
  if (id && records[id]) return records[id];
  const rows = ((store as any)?.state?.result?.rows || []) as Record<string, unknown>[];
  // Never fall back to an unrelated rows[0] — wrong-module Plan/Apply would be unsafe.
  const matched = id ? rows.find(r => String(r?.Id || '') === id) : undefined;
  return matched || {};
});

const hasMatchedRow = computed(() => {
  const rec = currentRow.value;
  return !!(String(rec.ModuleName || '').trim() || String(rec.Id || '').trim());
});
const primaryAction = computed(() => {
  if (!hasMatchedRow.value) return null;
  const action = resolveModuleCatalogFormPrimaryAction(currentRow.value);
  if (action === 'install' && !hasAction(moduleInstallAction)) return null;
  if (action === 'upgrade' && !hasAction(moduleUpgradeAction)) return null;
  return action;
});
const primaryActionLabel = computed(() => {
  if (primaryAction.value === 'install') return _t('Install');
  if (primaryAction.value === 'upgrade') return _t('Upgrade');
  if (primaryAction.value === 'progress') return _t('View Progress');
  return '';
});
/** Emphasize Upgrade only when catalog detected a newer source version. */
const primaryButtonVariant = computed(() => {
  if (primaryAction.value === 'upgrade' && currentRow.value.UpdateAvailable !== true) {
    return 'outline' as const;
  }
  if (primaryAction.value === 'progress') return 'outline' as const;
  return 'default' as const;
});
const canUninstallRow = computed(() => {
  if (!hasMatchedRow.value) return false;
  const rec = currentRow.value;
  if (String(rec.ActiveOperationId || '').trim()) return false;
  return rec.Installed === true && rec.Available !== false;
});
/** Same visibility gate as catalog load-demo, scoped to this module row. */
const canLoadDemo = computed(() => {
  if (!hasMatchedRow.value) return false;
  const rec = currentRow.value;
  if (String(rec.ActiveOperationId || '').trim()) return false;
  return rec.Installed === true && !!rec.HasDemo && !rec.DemoLoaded;
});
const showHeaderActions = computed(
  () =>
    !!primaryAction.value ||
    (canUninstallRow.value && hasAction(moduleUninstallAction)) ||
    (canLoadDemo.value && hasAction(moduleLoadDemoAction)),
);

async function refreshDetailRow() {
  const id = String(recordId.value || '').trim();
  if (!id) return;
  try {
    if (typeof (store as any)?.Browse === 'function') {
      await (store as any).Browse(id);
      return;
    }
    if (typeof (store as any)?.Search === 'function') {
      await (store as any).Search(['Id', '=', id], { limit: 1 });
    }
  } catch {
    // Refresh is best-effort after an operation settles.
  }
}

const {
  progressBusy,
  progressTitle,
  progressHeadline,
  progressValue,
  startOpProgress,
} = useModuleCatalogOpProgress(toRef(props, 'operationStore'), {
  jobStillRunning: _t('Job is still running in the background; refresh later'),
  serviceRestarting: _t('Service is restarting; status will retry automatically'),
  failedToGetStatus: _t('Failed to get status'),
  settledOk: _t('Operation Succeeded'),
  settledFailed: msg => _t('Operation Failed: %s', msg),
  reloadingService: () => _t('Almost done, refreshing…'),
  executing: () => _t('Working on it, please wait…'),
  queued: () => _t('Getting ready…'),
  finishing: () => _t('Wrapping up…'),
  inProgress: () => _t('Please wait…'),
  titleFor: (action, moduleName) => {
    const name = String(moduleName || '').trim();
    if (action === 'install') return name ? _t('Installing %s', name) : _t('Installing…');
    if (action === 'upgrade') return name ? _t('Upgrading %s', name) : _t('Upgrading…');
    if (action === 'uninstall') return name ? _t('Uninstalling %s', name) : _t('Uninstalling…');
    if (action === 'load_demo') return _t('Loading demo data…');
    if (action === 'progress') return name ? _t('Working on %s', name) : _t('Working…');
    return name ? _t('Working on %s', name) : _t('Working…');
  },
}, { refreshCatalog: refreshDetailRow });

function onPrimaryClick() {
  const rec = currentRow.value;
  const action = primaryAction.value;
  if (!action) return;
  drawerModuleName.value = String(rec.ModuleName || '');
  drawerRegistryNewer.value = !!rec.RegistryNewerThanLocal;
  if (action === 'progress') {
    void startOpProgress(String(rec.ActiveOperationId || '').trim(), {
      moduleName: drawerModuleName.value,
      action: 'progress',
    });
    return;
  }
  drawerOperationId.value = '';
  drawerAction.value = action;
  drawerOpen.value = true;
}

function onUninstallClick() {
  const rec = currentRow.value;
  drawerModuleName.value = String(rec.ModuleName || '');
  drawerOperationId.value = '';
  drawerAction.value = 'uninstall';
  drawerRegistryNewer.value = false;
  drawerOpen.value = true;
}

function onLoadDemoClick() {
  if (!canLoadDemo.value) return;
  drawerModuleName.value = '';
  drawerOperationId.value = '';
  drawerAction.value = 'load_demo';
  drawerRegistryNewer.value = false;
  drawerOpen.value = true;
}

function onOpApplied(operationId?: string) {
  const id = String(operationId || '').trim();
  void refreshDetailRow();
  if (!id) return;
  drawerOpen.value = false;
  void startOpProgress(id, {
    moduleName: drawerModuleName.value,
    action: drawerAction.value,
  });
}

async function onDrawerSettled() {
  await refreshDetailRow();
}

defineExpose({
  onDrawerSettled,
  onPrimaryClick,
  onUninstallClick,
  onLoadDemoClick,
  canLoadDemo,
  drawerOpen,
  drawerAction,
  drawerOperationId,
  drawerModuleName,
  drawerRegistryNewer,
  primaryAction,
  primaryActionLabel,
  primaryButtonVariant,
  progressBusy,
  progressTitle,
  progressHeadline,
  progressValue,
});
</script>
