<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Module show is read-only: omit mutating action-ids so Edit/Copy/Delete stay hidden. -->
  <div class="module-show-view flex min-h-0 flex-1 flex-col gap-3" data-testid="module-show-view">
    <div v-if="showHeaderActions" class="flex flex-wrap items-center gap-2" data-testid="module-show-actions">
      <button
        v-if="canUninstallRow && hasAction(moduleUninstallAction)"
        type="button"
        class="rounded-md border border-border px-2.5 py-1 text-xs text-muted-foreground hover:bg-muted"
        data-testid="module-show-uninstall"
        @click="onUninstallClick"
      >
        {{ _t('Uninstall') }}
      </button>
      <button
        v-if="canLoadDemo && hasAction(moduleLoadDemoAction)"
        type="button"
        class="rounded-md border border-border px-2.5 py-1 text-xs hover:bg-muted"
        data-testid="module-show-load-demo"
        @click="onLoadDemoClick"
      >
        {{ _t('Load Demo Data') }}
      </button>
      <button
        v-if="primaryAction"
        type="button"
        class="rounded-md border border-border px-2.5 py-1 text-xs hover:bg-muted"
        :data-testid="`module-show-primary-${primaryAction}`"
        @click="onPrimaryClick"
      >
        {{ primaryActionLabel }}
      </button>
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

  <ModuleOpDrawer
    v-model:open="drawerOpen"
    :operation-store="operationStore"
    :operation-id="drawerOperationId"
    :module-name="drawerModuleName"
    :action="drawerAction"
    :registry-newer-than-local="drawerRegistryNewer"
    @settled="onDrawerSettled"
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
import { resolveModuleCatalogPrimaryAction } from './module_catalog_chrome';
import {
  moduleInstallAction,
  moduleLoadDemoAction,
  moduleUpgradeAction,
  moduleUninstallAction,
} from './module_actions';
import ModuleOpDrawer from './ModuleOpDrawer.vue';
import {
  ChoyBooleanField,
  ChoyCard,
  ChoyCol,
  ChoyDatetimeField,
  ChoyFormView,
  ChoyGrid,
  ChoyJsonField,
  ChoyTextField,
  ChoyVarcharField,
} from '@/web';

defineOptions({ name: 'ModuleShowView', inheritAttrs: true });

const { _t } = createTranslate('meta', { scope: 'web/views/ModuleShowView' });
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

const store = resolvePageStore(props.store, 'ModuleShowView');
const recordId = toRef(props, 'recordId');
const showHeader = toRef(props, 'showHeader');
const operationStore = computed(() => props.operationStore ?? null);

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
  const action = resolveModuleCatalogPrimaryAction(currentRow.value);
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

function onPrimaryClick() {
  const rec = currentRow.value;
  const action = primaryAction.value;
  if (!action) return;
  drawerModuleName.value = String(rec.ModuleName || '');
  drawerRegistryNewer.value = !!rec.RegistryNewerThanLocal;
  if (action === 'progress') {
    drawerOperationId.value = String(rec.ActiveOperationId || '').trim();
    drawerAction.value = 'progress';
  } else {
    drawerOperationId.value = '';
    drawerAction.value = action;
  }
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

async function onDrawerSettled() {
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
});
</script>
