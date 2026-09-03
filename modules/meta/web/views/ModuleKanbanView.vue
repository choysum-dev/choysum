<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OKanbanView
    :store="store"
    :show-header="showHeader"
    :show-actions="false"
    :show-paginate="true"
    :searchView="OSearchView"
    :keyword-fields="keywordFields"
    @card-click="onCardClick"
  >
    <template #header-right>
      <el-button-group>
        <el-tooltip :content="_t('Board View')" placement="top">
          <el-button :icon="GridViewSharp" @click="toKanban" type="primary" />
        </el-tooltip>
        <el-tooltip v-if="canRoute('meta.route.module_list')" :content="_t('List View')" placement="top">
          <el-button :icon="FormatListBulletedOutlined" @click="toList" />
        </el-tooltip>
        <el-tooltip v-if="canRoute('meta.route.module_history')" :content="_t('Operation History')" placement="top">
          <el-button :icon="HistoryOutlined" @click="toHistory" />
        </el-tooltip>
        <el-tooltip v-if="hasAction(moduleSyncIndexAction)" :content="_t('Sync Index')" placement="top">
          <el-button :icon="Refresh" :loading="syncLoading" @click="onSyncIndex" />
        </el-tooltip>
      </el-button-group>
    </template>
    <template #fields>
      <OVirtualField :store="store" prop="ModuleName" />
      <OVirtualField :store="store" prop="Version" />
      <OVirtualField :store="store" prop="LocalVersion" />
      <OVirtualField :store="store" prop="RegistryVersion" />
      <OVirtualField :store="store" prop="InstalledStatus" />
      <OVirtualField :store="store" prop="InstalledVersion" />
      <OVirtualField :store="store" prop="Available" />
      <OVirtualField :store="store" prop="OriginTypes" />
      <OVirtualField :store="store" prop="LastSyncAt" />
      <OVirtualField :store="store" prop="ManifestJson" />
    </template>

    <template #card="{ record }">
      <div class="module-card">
        <div class="module-card__title">
          <span class="name">{{ record.ModuleName }}</span>
          <el-tag size="small" :type="statusTagType(record.InstalledStatus, record.Available)">{{
            statusLabel(record.InstalledStatus, record.Available)
          }}</el-tag>
        </div>
        <div class="module-card__meta">
          <span class="version">{{ _t('Local:') }} {{ record.LocalVersion || '—' }}</span>
          <span class="category">{{ _t('Registry:') }} {{ record.RegistryVersion || '—' }}</span>
        </div>
        <div class="module-card__meta">
          <span class="version">{{ _t('Display:') }} {{ record.Version || '—' }}</span>
          <span class="category">{{ _t('Installed:') }} {{ record.InstalledVersion || '—' }}</span>
        </div>
        <div class="module-card__meta">
          <span class="app">{{ _t('Origin:') }} {{ record.OriginTypes || record.OriginType || 'local' }}</span>
          <span class="updated">{{ _t('Synced:') }} {{ formatDate(record.LastSyncAt) || '—' }}</span>
        </div>
        <div class="module-card__desc">{{ manifestSummary(record.ManifestJson) || _t('No description') }}</div>
        <div class="module-card__actions">
          <el-button
            size="small"
            type="primary"
            plain
            @click.stop="onActionClick('install', record)"
            v-if="!isInstalled(record.InstalledStatus) && hasAction(moduleInstallAction)"
            :disabled="record.Available === false"
          >
            {{ _t('Install') }}
          </el-button>
          <el-button
            v-if="isInstalled(record.InstalledStatus) && hasAction(moduleUpgradeAction)"
            size="small"
            type="warning"
            plain
            @click.stop="onActionClick('upgrade', record)"
          >
            {{ _t('Upgrade') }}
          </el-button>
          <el-button
            v-if="isInstalled(record.InstalledStatus) && hasAction(moduleUninstallAction)"
            size="small"
            type="danger"
            plain
            @click.stop="onActionClick('uninstall', record)"
          >
            {{ _t('Uninstall') }}
          </el-button>
        </div>
      </div>
    </template>

    <template #card-empty>
      <div class="module-card__empty">{{ _t('No modules') }}</div>
    </template>
  </OKanbanView>

  <el-dialog v-model="dialogVisible" width="680px" :close-on-click-modal="false" @close="onDialogClose">
    <template #header>
      <span>{{ dialogTitle }}</span>
    </template>

    <div v-if="dialogStep === 'plan'" class="module-dialog">
      <el-skeleton v-if="planLoading" :rows="6" animated />
      <template v-else>
        <el-alert v-if="plan?.blockers?.length" type="error" :closable="false" show-icon :title="_t('Resolve blockers before continuing')" />
        <el-alert v-else type="info" :closable="false" show-icon :title="_t('Confirm the impact of this operation')" />

        <div class="module-dialog__section">
          <div class="section-title">{{ _t('Affected Modules') }}</div>
          <ul class="section-list">
            <li v-for="item in plan?.affectedModules || []" :key="item.moduleName">
              <span class="item-name">{{ item.moduleName }}</span>
              <span class="item-meta">{{ item.currentVersion || '—' }}</span>
              <span class="item-meta">{{ item.targetVersion || '' }}</span>
              <span class="item-reason">{{ item.reason || '' }}</span>
            </li>
          </ul>
        </div>

        <div class="module-dialog__section" v-if="plan?.risks?.length">
          <div class="section-title">{{ _t('Risks') }}</div>
          <ul class="section-list">
            <li v-for="risk in plan?.risks" :key="risk.code">
              <el-tag size="small" type="warning">{{ risk.code }}</el-tag>
              <span class="item-desc">{{ risk.message || '' }}</span>
            </li>
          </ul>
        </div>

        <div class="module-dialog__section" v-if="plan?.blockers?.length">
          <div class="section-title">{{ _t('Blockers') }}</div>
          <ul class="section-list">
            <li v-for="blocker in plan?.blockers" :key="blocker.code">
              <el-tag size="small" type="danger">{{ blocker.code }}</el-tag>
              <span class="item-desc">{{ blocker.message || '' }}</span>
            </li>
          </ul>
        </div>

        <div class="module-dialog__section" v-if="action === 'install'">
          <el-checkbox v-model="withDemo">{{ _t('Include demo data') }}</el-checkbox>
        </div>
      </template>
    </div>

    <div v-else class="module-dialog">
      <div class="module-dialog__section">
        <el-alert type="info" :closable="false" show-icon :title="_t('Operation in progress, please wait')" v-if="dialogStep === 'progress'" />
        <el-alert v-else :type="resultAlertType" :closable="false" show-icon :title="resultTitle" />
      </div>
      <div class="module-dialog__section">
        <div class="section-title">{{ _t('Execution Status') }}</div>
        <div class="status-row">
          <span class="label">{{ _t('Status:') }}</span>
          <el-tag size="small" :type="statusTagType(opStatus?.status)">{{ opStatus?.status || '—' }}</el-tag>
          <span class="label">{{ _t('Result:') }}</span>
          <el-tag size="small" :type="resultTagType(opStatus?.resultStatus)">{{ opStatus?.resultStatus || '—' }}</el-tag>
        </div>
        <div class="status-row" v-if="opStatus?.summary">
          <span class="label">{{ _t('Summary:') }}</span>
          <span class="value">{{ formatSummary(opStatus?.summary) }}</span>
        </div>
        <div class="status-row" v-if="opStatus?.failureKind && opStatus?.failureKind !== 'NONE'">
          <span class="label">{{ _t('Failure Kind:') }}</span>
          <span class="value">{{ opStatus?.failureKind }}</span>
        </div>
        <div class="status-row" v-if="opStatus?.errorDomain || opStatus?.errorCode">
          <span class="label">{{ _t('Error:') }}</span>
          <span class="value">{{ opStatus?.errorDomain || '—' }} / {{ opStatus?.errorCode || '—' }}</span>
        </div>
        <div class="status-row" v-if="opStatus?.reload_triggered">
          <span class="label">{{ _t('Reload:') }}</span>
          <span class="value">{{ opStatus?.reload_failed ? _t('Trigger Failed') : _t('Triggered') }}</span>
        </div>
      </div>
      <div class="module-dialog__section" v-if="dialogStep === 'progress'">
        <el-divider />
        <div class="progress-note">{{ _t('Do not refresh; status updates automatically.') }}</div>
      </div>
    </div>

    <template #footer>
      <el-button @click="dialogVisible = false" :disabled="planLoading">{{ _t('Cancel') }}</el-button>
      <el-button
        v-if="dialogStep === 'plan'"
        type="primary"
        :loading="planLoading || executeLoading"
        :disabled="!!plan?.blockers?.length"
        @click="submitOperation"
      >
        {{ _t('Confirm') }}
      </el-button>
      <el-button v-else type="primary" @click="dialogVisible = false">{{ _t('Done') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type MetaModule from '@/meta/service/models/module';
import type MetaModuleIndex from '@/meta/service/models/module_index';
import OKanbanView from '@/web/web/components/view/OKanbanView.vue';
import OVirtualField from '@/web/web/components/field/OVirtualField.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { ElTag, ElDialog, ElButton, ElAlert, ElDivider, ElCheckbox, ElSkeleton, ElMessage, ElTooltip, ElButtonGroup } from 'element-plus';
import { useRouter } from 'vue-router';
import type { ClientModelProps } from '@/core/rpc/types';
import { FormatListBulletedOutlined, GridViewSharp, HistoryOutlined } from '@vicons/material';
import { Refresh } from '@element-plus/icons-vue';
import { defineAction } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import {
  createModuleOpProgressSession,
  type ModuleOpStatusSnapshot,
} from '../composables/useModuleOpProgress';
import { createModuleKanbanOpProgressHooks } from '../composables/moduleKanbanOpProgress';

defineOptions({ name: 'ModuleKanbanView' });

const { _t, _lt } = createTranslate('meta', { scope: 'web/views/ModuleKanbanView' });

/**
 * Props consumed by the module kanban view and its backing stores.
 */
const props = withDefaults(defineProps<{ store?: WebModelStore<MetaModuleIndex>; moduleStore: WebModelStore<MetaModule>; showHeader?: boolean }>(), {
  showHeader: true,
});
const store = resolvePageStore(props.store, 'ModuleKanbanView');
const { showHeader } = props;
const moduleStore = props.moduleStore;

// Keep keyword search on persisted fields only; computed/derived fields require
// explicit backend @Search handlers and can break list queries.
const keywordFields = ['ModuleName', 'Version', 'OriginType', 'OriginRef'];

const router = useRouter();
const moduleInstallAction = defineAction('meta.action.module_install', {
  title: _lt('Install Module'),
  requires: [{ model: 'meta.MetaModule', method: 'RequestInstall' }],
});
const moduleUpgradeAction = defineAction('meta.action.module_upgrade', {
  title: _lt('Upgrade Module'),
  requires: [{ model: 'meta.MetaModule', method: 'RequestUpgrade' }],
});
const moduleUninstallAction = defineAction('meta.action.module_uninstall', {
  title: _lt('Uninstall Module'),
  requires: [{ model: 'meta.MetaModule', method: 'RequestUninstall' }],
});
const moduleSyncIndexAction = defineAction('meta.action.module_sync_index', {
  title: _lt('Sync Module Index'),
  requires: [{ model: 'meta.MetaModuleIndex', method: 'RequestSync' }],
});
const { canRoute, hasAction } = usePermission();

/**
 * Routes back to the module kanban board.
 */
function toKanban() {
  router.push('/meta/modules');
}

/**
 * Routes to the table-based module list.
 */
function toList() {
  router.push('/meta/modules/list');
}

/**
 * Routes to the module management history page.
 */
function toHistory() {
  router.push('/meta/modules/history');
}

/**
 * Module actions exposed by the kanban toolbar and cards.
 */
type ModuleAction = 'install' | 'uninstall' | 'upgrade';

/**
 * Plan response returned before a module action is executed.
 */
type PlanOperationResp = {
  baseRevision: string;
  affectedModules: Array<{ moduleName: string; reason?: string; currentVersion?: string; targetVersion?: string }>;
  risks: Array<{ code: string; level: string; message?: string; params?: Record<string, any> }>;
  blockers: Array<{ code: string; level: string; message?: string; params?: Record<string, any> }>;
};

/**
 * Operation status snapshot returned while a module action is executing.
 */
type OpStatusResp = ModuleOpStatusSnapshot;

const dialogVisible = ref(false);
const dialogStep = ref<'plan' | 'progress' | 'result'>('plan');
const planLoading = ref(false);
const executeLoading = ref(false);
const plan = ref<PlanOperationResp | null>(null);
const opStatus = ref<OpStatusResp | null>(null);
const action = ref<ModuleAction>('install');
const targetModule = ref<ClientModelProps<MetaModuleIndex> | null>(null);
const withDemo = ref(false);

const opProgress = createModuleOpProgressSession(
  createModuleKanbanOpProgressHooks({
    fetchStatus: async (jobId) => (await (moduleStore as any).GetOpStatus(jobId)) as OpStatusResp,
    isDialogOpen: () => dialogVisible.value,
    setOpStatus: (status) => {
      opStatus.value = status;
    },
    setDialogStep: (step) => {
      dialogStep.value = step;
    },
    warn: (message) => {
      ElMessage.warning(message);
    },
    error: (message) => {
      ElMessage.error(message);
    },
    messages: {
      jobStillRunning: () => _t('Job is still running in the background; refresh later'),
      serviceRestarting: () => _t('Service is restarting; status will retry automatically'),
      failedToGetStatus: () => _t('Failed to get status'),
    },
  })
);

/**
 * Builds the current dialog title from the selected module action.
 */
const dialogTitle = computed(() => {
  const actionLabel = action.value === 'install' ? _t('Install Module') : action.value === 'uninstall' ? _t('Uninstall Module') : _t('Upgrade Module');
  return `${actionLabel} · ${targetModule.value?.ModuleName || ''}`.trim();
});

/**
 * Maps the latest operation snapshot to the dialog headline.
 */
const resultTitle = computed(() => {
  if (!opStatus.value) return _t('Completed');
  if (opStatus.value.resultStatus === 'FAILED') return _t('Operation Failed');
  return opStatus.value.reload_failed ? _t('Succeeded but reload failed') : _t('Operation Succeeded');
});

/**
 * Maps the latest operation snapshot to the dialog alert tone.
 */
const resultAlertType = computed(() => {
  if (!opStatus.value) return 'info';
  if (opStatus.value.resultStatus === 'FAILED') return 'error';
  return opStatus.value.reload_failed ? 'warning' : 'success';
});

/**
 * Chooses the result tag style for the operation status summary.
 */
const resultTagType = (status?: string) => {
  if (status === 'SUCCEEDED') return 'success';
  if (status === 'FAILED') return 'danger';
  return 'info';
};

/**
 * Chooses the visual tag type for module and operation statuses.
 */
function statusTagType(status?: string, available?: boolean) {
  if (available === false) return 'danger';
  const val = String(status || '').toLowerCase();
  if (val === 'installed') return 'success';
  if (val === 'uninstalled') return 'info';
  if (val === 'disabled') return 'warning';
  if (val === 'broken') return 'danger';
  if (val === 'succeeded') return 'success';
  if (val === 'failed') return 'danger';
  if (val === 'dispatching' || val === 'queued') return 'warning';
  return 'info';
}

/**
 * Maps raw status values to the labels shown on the module board.
 */
function statusLabel(status?: string, available?: boolean) {
  if (available === false) return _t('Unavailable');
  const val = String(status || '').toLowerCase();
  if (val === 'installed') return _t('Installed');
  if (val === 'uninstalled') return _t('Not Installed');
  if (val === 'disabled') return _t('Disabled');
  if (val === 'broken') return _t('Broken');
  if (val === 'succeeded') return _t('Succeeded');
  if (val === 'failed') return _t('Failed');
  if (val === 'dispatching') return _t('Dispatching');
  if (val === 'queued') return _t('Queued');
  return status || _t('Unknown');
}

/**
 * Reports whether a module record is currently installed.
 */
function isInstalled(status?: string) {
  return String(status || '').toLowerCase() === 'installed';
}

/**
 * Formats module timestamps for card metadata display.
 */
function formatDate(dt?: any) {
  if (!dt) return '';
  try {
    const d = typeof dt === 'string' ? new Date(dt) : dt;
    if (!(d instanceof Date) || isNaN(d.getTime())) return String(dt).slice(0, 19);
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    const hh = String(d.getHours()).padStart(2, '0');
    const mm = String(d.getMinutes()).padStart(2, '0');
    return `${y}-${m}-${day} ${hh}:${mm}`;
  } catch {
    return String(dt).slice(0, 19);
  }
}

/**
 * Condenses the operation summary payload into display text.
 */
function formatSummary(summary: any) {
  if (!summary) return '';
  if (typeof summary === 'string') return summary;
  if (summary.message) return summary.message;
  if (summary.code) return summary.code;
  try {
    return JSON.stringify(summary);
  } catch {
    return String(summary);
  }
}

/**
 * Opens the module detail route for the selected kanban card.
 */
function onCardClick(payload: { row: ClientModelProps<MetaModuleIndex> }) {
  router.push(`/meta/modules/${payload.row.Id}`);
}

/**
 * Opens the operation dialog and loads the execution plan for a module action.
 */
async function onActionClick(nextAction: ModuleAction, record: ClientModelProps<MetaModuleIndex>) {
  resetDialog();
  action.value = nextAction;
  targetModule.value = record;
  withDemo.value = false;
  dialogVisible.value = true;
  dialogStep.value = 'plan';
  planLoading.value = true;
  try {
    plan.value = (await (moduleStore as any).PlanOperation({
      action: nextAction,
      moduleName: record.ModuleName,
      withDemo: nextAction === 'install' ? withDemo.value : false,
    })) as PlanOperationResp;
  } catch (error: any) {
    ElMessage.error(error?.message || _t('Failed to load plan'));
    dialogVisible.value = false;
  } finally {
    planLoading.value = false;
  }
}

/**
 * Dispatches the selected module action and watches its job status (C1 tip + fallback).
 */
async function submitOperation() {
  if (!targetModule.value) return;
  executeLoading.value = true;
  dialogStep.value = 'progress';
  try {
    let jobId = '';
    const moduleName = targetModule.value.ModuleName;
    if (action.value === 'install') jobId = await (moduleStore as any).RequestInstall(moduleName, withDemo.value);
    else if (action.value === 'uninstall') jobId = await (moduleStore as any).RequestUninstall(moduleName);
    else jobId = await (moduleStore as any).RequestUpgrade(moduleName);
    executeLoading.value = false;
    await opProgress.watch(jobId);
  } catch (error: any) {
    executeLoading.value = false;
    dialogStep.value = 'result';
    opStatus.value = {
      status: 'failed',
      resultStatus: 'FAILED',
      errorDomain: 'CLIENT',
      errorCode: 'REQUEST_FAILED',
    } as OpStatusResp;
    ElMessage.error(error?.message || _t('Operation request failed'));
  }
}

/**
 * Resets dialog state before a new module action starts.
 */
function resetDialog() {
  plan.value = null;
  opStatus.value = null;
  dialogStep.value = 'plan';
  planLoading.value = false;
  executeLoading.value = false;
  opProgress.stop();
}

/**
 * Stops tip/poll when the dialog is dismissed.
 */
function onDialogClose() {
  opProgress.stop();
}

/**
 * Extracts a short module summary from the manifest payload.
 */
function manifestSummary(raw: any) {
  if (!raw || typeof raw !== 'object') return '';
  const text = raw.short_desc || raw.shortDesc || raw.summary || raw.description || raw.name || '';
  return typeof text === 'string' ? text : '';
}

const syncLoading = ref(false);

/**
 * Triggers a forced module index sync from the kanban toolbar.
 */
async function onSyncIndex() {
  if (syncLoading.value) return;
  syncLoading.value = true;
  try {
    const jobId = await (store as any).RequestSync({ force: true, ifStale: false });
    ElMessage.success(jobId ? _t('Sync job triggered: all:%s', String(jobId)) : _t('Sync job triggered'));
  } catch (error: any) {
    ElMessage.warning(_t('Sync failed: %s', String(error?.message || 'request failed')));
  } finally {
    syncLoading.value = false;
  }
}

onBeforeUnmount(() => {
  opProgress.stop();
});

/**
 * Silently triggers a stale-aware index refresh on kanban entry.
 * Failures are suppressed to avoid blocking page usability.
 */
onMounted(async () => {
  try {
    await (store as any).RequestSync({ ifStale: true });
  } catch {
    // sync unavailable — silently skip, page remains usable
  }
});

defineExpose({
  onActionClick,
  submitOperation,
  resetDialog,
  onDialogClose,
  dialogVisible,
  dialogStep,
  opStatus,
  action,
  targetModule,
});
</script>

<style scoped lang="scss">
.module-card {
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  padding: 12px 14px 10px 14px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  background: var(--el-color-white);
  transition:
    box-shadow 0.18s ease,
    transform 0.18s ease;
}
.module-card:hover {
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.08);
}

.module-card__title {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
  font-weight: 600;
  font-size: 14px;
}

.module-card__meta {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.module-card__desc {
  font-size: 12px;
  color: var(--el-text-color-regular);
  min-height: 34px;
}

.module-card__actions {
  display: flex;
  gap: 8px;
}

.module-card__empty {
  opacity: 0.6;
  font-size: 12px;
  text-align: center;
  padding: 8px 0;
}

.module-dialog {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.module-dialog__section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.section-title {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.section-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 12px;
}

.section-list li {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--el-text-color-regular);
}

.item-name {
  font-weight: 600;
}

.item-meta {
  color: var(--el-text-color-secondary);
}

.item-reason {
  color: var(--el-color-primary);
}

.item-desc {
  color: var(--el-text-color-secondary);
}

.status-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--el-text-color-regular);
  flex-wrap: wrap;
}

.label {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.value {
  color: var(--el-text-color-regular);
}

.progress-note {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
