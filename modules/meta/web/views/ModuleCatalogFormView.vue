<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Module show is read-only: omit mutating action-ids so Edit/Copy/Delete stay hidden. -->
  <div class="module-catalog-form-view flex min-h-0 flex-1 flex-col" data-testid="module-catalog-form-view">
    <!--
      Installed / Available / UpdateAvailable are not shown in the sheet; register them as
      virtual fields so Form Browse still selects them for header CTAs.
    -->
    <div class="sr-only" aria-hidden="true">
      <ChoyVirtualField :store="store" prop="Installed" />
      <ChoyVirtualField :store="store" prop="Available" />
      <ChoyVirtualField :store="store" prop="UpdateAvailable" />
      <ChoyVirtualField :store="store" prop="ActiveOperationId" />
      <ChoyVirtualField :store="store" prop="RegistryNewerThanLocal" />
    </div>

    <ChoyFormView
      v-bind="{ store, recordId, viewMode: 'display' as const, showHeader }"
    >
      <!-- Install / Upgrade / Uninstall after the title/⋮ cluster (List/Kanban CTA parity). -->
      <template #secondary-actions>
        <ChoyButton
          v-if="primaryAction"
          type="button"
          size="default"
          :variant="primaryButtonVariant"
          :data-testid="`module-catalog-form-primary-${primaryAction}`"
          @click="onPrimaryClick"
        >
          {{ primaryActionLabel }}
        </ChoyButton>
        <ChoyButton
          v-if="canUninstallRow && hasAction(moduleUninstallAction)"
          type="button"
          size="default"
          variant="outline-destructive"
          data-testid="module-catalog-form-uninstall"
          @click="onUninstallClick"
        >
          {{ _t('Uninstall') }}
        </ChoyButton>
      </template>

      <div data-anchor="choy.form.sheet" class="flex min-h-full flex-col gap-0">
        <ChoySection :title="_t('Overview')" class="mt-0">
          <ChoyGrid :cols="12" class="gap-x-6 gap-y-3">
            <ChoyCol :span="4">
              <ChoyVarcharField :store="store" prop="ModuleName" orientation="vertical" />
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoyVarcharField :store="store" prop="Application" orientation="vertical" />
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoyVarcharField :store="store" prop="Category" orientation="vertical" />
            </ChoyCol>
            <ChoyCol v-if="showSummary" :span="12">
              <ChoyTextField :store="store" prop="Summary" orientation="vertical" />
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoyVarcharField :store="store" prop="Author" orientation="vertical" />
            </ChoyCol>
            <ChoyCol :span="8">
              <ChoyTextField :store="store" prop="License" orientation="vertical" />
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoyVarcharField :store="store" prop="InstalledVersion" orientation="vertical" />
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoyDatetimeField :store="store" prop="LastInstalledOrUpgradedAt" orientation="vertical" />
            </ChoyCol>
          </ChoyGrid>
        </ChoySection>

        <ChoyTabs v-model="detailTab" class="mt-4" default-value="versions">
          <ChoyTab value="versions" :label="_t('Versions & Source')" force-mount>
            <ChoyGrid :cols="12" class="gap-x-6 gap-y-3">
              <ChoyCol :span="6" class="flex min-w-0 flex-col gap-3">
                <div class="text-sm font-medium text-muted-foreground">{{ _t('Local') }}</div>
                <ChoyVarcharField :store="store" prop="LocalVersion" orientation="vertical" />
                <ChoyVarcharField :store="store" prop="LocalPath" orientation="vertical" />
                <ChoyDatetimeField :store="store" prop="LastSyncAt" orientation="vertical" />
              </ChoyCol>
              <ChoyCol :span="6" class="flex min-w-0 flex-col gap-3">
                <div class="text-sm font-medium text-muted-foreground">{{ _t('Registry') }}</div>
                <ChoyVarcharField :store="store" prop="RegistryVersion" orientation="vertical" />
                <ChoyVarcharField :store="store" prop="RegistryTarball" orientation="vertical" />
              </ChoyCol>
              <ChoyCol v-if="showActiveOperation" :span="4">
                <ChoyVarcharField :store="store" prop="ActiveOperationId" orientation="vertical" />
              </ChoyCol>
              <ChoyCol v-if="showUnavailableReason" :span="12">
                <ChoyTextField :store="store" prop="UnavailableReason" orientation="vertical" />
              </ChoyCol>
            </ChoyGrid>
          </ChoyTab>

          <ChoyTab value="dependencies" :label="_t('Dependencies')" force-mount>
            <ChoyGrid :cols="12" class="gap-x-6 gap-y-4">
              <ChoyCol :span="12">
                <div
                  class="module-catalog-depend-m2m"
                  data-testid="module-catalog-form-depends"
                >
                  <ChoyManyToManyField
                    :store="store"
                    prop="Depends"
                    orientation="vertical"
                    :readonly="true"
                    :show-index="false"
                    @row-click="onDependRowClick"
                  >
                    <!-- Nested fields register Browse paths; custom columns render cells. -->
                    <ChoyVarcharField :store="store" prop="Depends.ModuleName" :visible="false" />
                    <ChoyVarcharField :store="store" prop="Depends.Application" :visible="false" />
                    <ChoyVarcharField :store="store" prop="Depends.License" :visible="false" />
                    <ChoyVarcharField :store="store" prop="Depends.InstalledVersion" :visible="false" />
                    <ChoyVarcharField :store="store" prop="Depends.LocalVersion" :visible="false" />
                    <ChoyVarcharField :store="store" prop="Depends.RegistryVersion" :visible="false" />
                    <ChoyBooleanField :store="store" prop="Depends.Installed" :visible="false" />
                    <ChoyBooleanField :store="store" prop="Depends.UpdateAvailable" :visible="false" />
                    <ChoyBooleanField :store="store" prop="Depends.Available" :visible="false" />
                    <ChoyTableColumn
                      prop="ModuleName"
                      :label="_t('Module Name')"
                      :vColumnProps="{ minWidth: 140 }"
                    >
                      <template #default="{ row }">
                        <span class="truncate text-sm text-foreground">{{
                          dependFieldText(row, 'ModuleName')
                        }}</span>
                      </template>
                    </ChoyTableColumn>
                    <ChoyTableColumn
                      prop="InstalledVersion"
                      :label="_t('Version')"
                      :vColumnProps="{ minWidth: 220 }"
                    >
                      <template #default="{ row }">
                        <span
                          class="font-mono text-xs text-muted-foreground whitespace-nowrap"
                          data-testid="module-catalog-depend-version"
                          :title="dependVersionLine(row)"
                          >{{ dependVersionLine(row) }}</span
                        >
                      </template>
                    </ChoyTableColumn>
                    <ChoyTableColumn
                      prop="Installed"
                      :label="_t('Status')"
                      :vColumnProps="{ minWidth: 120, width: 140 }"
                    >
                      <template #default="{ row }">
                        <ChoyBadge
                          :variant="dependStatusBadgeVariant(row)"
                          class="h-5 px-1.5 text-[11px] font-normal"
                          data-testid="module-catalog-depend-status"
                        >
                          {{ dependStatusLabel(row) }}
                        </ChoyBadge>
                      </template>
                    </ChoyTableColumn>
                    <ChoyTableColumn
                      prop="Application"
                      :label="_t('Application')"
                      :vColumnProps="{ minWidth: 120 }"
                    >
                      <template #default="{ row }">
                        <span class="truncate text-sm text-muted-foreground">{{
                          dependFieldText(row, 'Application')
                        }}</span>
                      </template>
                    </ChoyTableColumn>
                    <ChoyTableColumn
                      prop="License"
                      :label="_t('License')"
                      :vColumnProps="{ minWidth: 120 }"
                    >
                      <template #default="{ row }">
                        <span class="truncate text-sm text-muted-foreground">{{
                          dependFieldText(row, 'License')
                        }}</span>
                      </template>
                    </ChoyTableColumn>
                  </ChoyManyToManyField>
                </div>
              </ChoyCol>
              <ChoyCol :span="12" class="mt-2">
                <div
                  class="module-catalog-depend-m2m"
                  data-testid="module-catalog-form-dependents"
                >
                  <ChoyManyToManyField
                    :store="store"
                    prop="Dependents"
                    orientation="vertical"
                    :readonly="true"
                    :show-index="false"
                    @row-click="onDependRowClick"
                  >
                    <ChoyVarcharField :store="store" prop="Dependents.ModuleName" :visible="false" />
                    <ChoyVarcharField :store="store" prop="Dependents.Application" :visible="false" />
                    <ChoyVarcharField :store="store" prop="Dependents.License" :visible="false" />
                    <ChoyVarcharField :store="store" prop="Dependents.InstalledVersion" :visible="false" />
                    <ChoyVarcharField :store="store" prop="Dependents.LocalVersion" :visible="false" />
                    <ChoyVarcharField :store="store" prop="Dependents.RegistryVersion" :visible="false" />
                    <ChoyBooleanField :store="store" prop="Dependents.Installed" :visible="false" />
                    <ChoyBooleanField :store="store" prop="Dependents.UpdateAvailable" :visible="false" />
                    <ChoyBooleanField :store="store" prop="Dependents.Available" :visible="false" />
                    <ChoyTableColumn
                      prop="ModuleName"
                      :label="_t('Module Name')"
                      :vColumnProps="{ minWidth: 140 }"
                    >
                      <template #default="{ row }">
                        <span class="truncate text-sm text-foreground">{{
                          dependFieldText(row, 'ModuleName')
                        }}</span>
                      </template>
                    </ChoyTableColumn>
                    <ChoyTableColumn
                      prop="InstalledVersion"
                      :label="_t('Version')"
                      :vColumnProps="{ minWidth: 220 }"
                    >
                      <template #default="{ row }">
                        <span
                          class="font-mono text-xs text-muted-foreground whitespace-nowrap"
                          data-testid="module-catalog-depend-version"
                          :title="dependVersionLine(row)"
                          >{{ dependVersionLine(row) }}</span
                        >
                      </template>
                    </ChoyTableColumn>
                    <ChoyTableColumn
                      prop="Installed"
                      :label="_t('Status')"
                      :vColumnProps="{ minWidth: 120, width: 140 }"
                    >
                      <template #default="{ row }">
                        <ChoyBadge
                          :variant="dependStatusBadgeVariant(row)"
                          class="h-5 px-1.5 text-[11px] font-normal"
                          data-testid="module-catalog-depend-status"
                        >
                          {{ dependStatusLabel(row) }}
                        </ChoyBadge>
                      </template>
                    </ChoyTableColumn>
                    <ChoyTableColumn
                      prop="Application"
                      :label="_t('Application')"
                      :vColumnProps="{ minWidth: 120 }"
                    >
                      <template #default="{ row }">
                        <span class="truncate text-sm text-muted-foreground">{{
                          dependFieldText(row, 'Application')
                        }}</span>
                      </template>
                    </ChoyTableColumn>
                    <ChoyTableColumn
                      prop="License"
                      :label="_t('License')"
                      :vColumnProps="{ minWidth: 120 }"
                    >
                      <template #default="{ row }">
                        <span class="truncate text-sm text-muted-foreground">{{
                          dependFieldText(row, 'License')
                        }}</span>
                      </template>
                    </ChoyTableColumn>
                  </ChoyManyToManyField>
                </div>
              </ChoyCol>
            </ChoyGrid>
          </ChoyTab>
        </ChoyTabs>
      </div>
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
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type MetaModuleCatalog from '@/meta/service/models/module_catalog';
import type MetaModuleOperation from '@/meta/service/models/module_operation';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { useModuleCatalogOpProgress } from '../composables/useModuleCatalogOpProgress';
import {
  catalogInstalledStatus,
  formatModuleCatalogVersionLine,
  moduleCatalogCardSummary,
  moduleStatusBadgeVariant,
  resolveModuleCatalogFormPrimaryAction,
} from './module_catalog_chrome';
import { moduleUninstallAction } from './module_catalog_actions';
import ModuleCatalogOpDialog from './ModuleCatalogOpDialog.vue';
import ModuleCatalogOpProgressStrip from './ModuleCatalogOpProgressStrip.vue';
import {
  ChoyBadge,
  ChoyBooleanField,
  ChoyButton,
  ChoyCol,
  ChoyDatetimeField,
  ChoyFormView,
  ChoyGrid,
  ChoyManyToManyField,
  ChoySection,
  ChoyTab,
  ChoyTableColumn,
  ChoyTabs,
  ChoyTextField,
  ChoyVarcharField,
  ChoyVirtualField,
} from '@/web';

defineOptions({ name: 'ModuleCatalogFormView', inheritAttrs: true });

const { _t } = createTranslate('meta', { scope: 'web/views/ModuleCatalogFormView' });
const { hasAction } = usePermission();
const router = useRouter();

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

const drawerOpen = ref(false);
const drawerOperationId = ref('');
const drawerModuleName = ref('');
const drawerAction = ref<'install' | 'upgrade' | 'uninstall' | 'progress' | ''>('');
const drawerRegistryNewer = ref(false);
const detailTab = ref('versions');

/** Unwrap Form Browse RecordRow `{ key, payload }` or flat Search rows into a field map. */
function catalogRowFields(row: unknown): Record<string, unknown> | null {
  if (!row || typeof row !== 'object') return null;
  const r = row as Record<string, unknown>;
  if (r.payload && typeof r.payload === 'object') return r.payload as Record<string, unknown>;
  if (r.raw && typeof r.raw === 'object') return r.raw as Record<string, unknown>;
  return r;
}

function catalogRowMatchesId(row: unknown, id: string): boolean {
  if (!id || !row || typeof row !== 'object') return false;
  const r = row as Record<string, unknown>;
  if (String(r.key || '').trim() === id) return true;
  const fields = catalogRowFields(row);
  return !!fields && String(fields.Id || '').trim() === id;
}

const currentRow = computed(() => {
  const id = String(recordId.value || '').trim();
  const records = ((store as any)?.state?.records || {}) as Record<string, Record<string, unknown>>;
  if (id && records[id]) return records[id];
  const rows = ((store as any)?.state?.result?.rows || []) as unknown[];
  // Never fall back to an unrelated rows[0] — wrong-module Plan/Apply would be unsafe.
  // Form beginDisplay writes RecordRow `{ kind, key, payload }`; List Search may be flat.
  const matched = id ? rows.find(r => catalogRowMatchesId(r, id)) : undefined;
  return catalogRowFields(matched) || {};
});

const hasMatchedRow = computed(() => {
  const rec = currentRow.value;
  return !!(String(rec.ModuleName || '').trim() || String(rec.Id || '').trim());
});

const primaryAction = computed(() => {
  if (!hasMatchedRow.value) return null;
  // Match List/Kanban: row state decides Install/Upgrade/Progress visibility.
  // Method ACL still gates Plan/Apply on the server when the user confirms.
  return resolveModuleCatalogFormPrimaryAction(currentRow.value);
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

const showSummary = computed(() => !!moduleCatalogCardSummary(currentRow.value));
const showUnavailableReason = computed(() => {
  const rec = currentRow.value;
  return rec.Available === false && !!String(rec.UnavailableReason || '').trim();
});
const showActiveOperation = computed(() => !!String(currentRow.value.ActiveOperationId || '').trim());
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

function openCatalogModule(moduleName: unknown) {
  const name = String(moduleName || '').trim();
  if (!name) return;
  void router.push(`/meta/modules/${encodeURIComponent(name)}`);
}

function onDependRowClick(payload: { rowData?: unknown }) {
  const rec = dependRowRecord(payload?.rowData);
  openCatalogModule(rec.ModuleName);
}

function dependRowRecord(row: unknown): Record<string, unknown> {
  if (!row || typeof row !== 'object') return {};
  const r = row as Record<string, unknown>;
  if (r.payload && typeof r.payload === 'object') return r.payload as Record<string, unknown>;
  if (r.raw && typeof r.raw === 'object') return r.raw as Record<string, unknown>;
  return r;
}

function dependFieldText(row: unknown, key: string): string {
  const text = String(dependRowRecord(row)[key] || '').trim();
  return text || '—';
}

function dependVersionLine(row: unknown): string {
  return formatModuleCatalogVersionLine(dependRowRecord(row)) || '—';
}

function dependStatusLabel(row: unknown): string {
  const rec = dependRowRecord(row);
  if (rec.Available === false) return _t('Unavailable');
  const val = catalogInstalledStatus(rec).toLowerCase();
  if (val === 'installed') {
    if (rec.UpdateAvailable === true) return _t('Update available');
    return _t('Installed');
  }
  if (val === 'uninstalled') return _t('Not Installed');
  return catalogInstalledStatus(rec) || _t('Unknown');
}

function dependStatusBadgeVariant(row: unknown) {
  const rec = dependRowRecord(row);
  return moduleStatusBadgeVariant(
    catalogInstalledStatus(rec),
    rec.Available as boolean | undefined,
    rec.UpdateAvailable === true,
  );
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
  onOpApplied,
  openCatalogModule,
  onDependRowClick,
  dependFieldText,
  dependVersionLine,
  dependStatusLabel,
  dependStatusBadgeVariant,
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
  detailTab,
});
</script>

<style scoped>
.module-catalog-depend-m2m :deep([role='row'][tabindex='0']) {
  cursor: pointer;
}
</style>
