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
        <el-tooltip content="看板视图" placement="top">
          <el-button :icon="GridViewSharp" @click="toKanban" type="primary" />
        </el-tooltip>
        <el-tooltip v-if="canRoute('meta.route.module_list')" content="列表视图" placement="top">
          <el-button :icon="FormatListBulletedOutlined" @click="toList" />
        </el-tooltip>
        <el-tooltip v-if="canRoute('meta.route.module_history')" content="操作历史" placement="top">
          <el-button :icon="HistoryOutlined" @click="toHistory" />
        </el-tooltip>
        <el-tooltip v-if="hasAction(moduleSyncIndexAction)" content="同步索引" placement="top">
          <el-button :icon="Refresh" :loading="syncLoading" @click="onSyncIndex" />
        </el-tooltip>
      </el-button-group>
    </template>
    <template #fields>
      <OVirtualField :store="store" prop="ModuleName" />
      <OVirtualField :store="store" prop="Version" />
      <OVirtualField :store="store" prop="InstalledStatus" />
      <OVirtualField :store="store" prop="InstalledVersion" />
      <OVirtualField :store="store" prop="Available" />
      <OVirtualField :store="store" prop="OriginType" />
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
          <span class="version">版本：{{ record.Version || '—' }}</span>
          <span class="category">已安装：{{ record.InstalledVersion || '—' }}</span>
        </div>
        <div class="module-card__meta">
          <span class="app">来源：{{ record.OriginType || 'local' }}</span>
          <span class="updated">同步：{{ formatDate(record.LastSyncAt) || '—' }}</span>
        </div>
        <div class="module-card__desc">{{ manifestSummary(record.ManifestJson) || '暂无描述' }}</div>
        <div class="module-card__actions">
          <el-button
            size="small"
            type="primary"
            plain
            @click.stop="onActionClick('install', record)"
            v-if="!isInstalled(record.InstalledStatus) && hasAction(moduleInstallAction)"
            :disabled="record.Available === false"
          >
            安装
          </el-button>
          <el-button
            v-if="isInstalled(record.InstalledStatus) && hasAction(moduleUpgradeAction)"
            size="small"
            type="warning"
            plain
            @click.stop="onActionClick('upgrade', record)"
          >
            升级
          </el-button>
          <el-button
            v-if="isInstalled(record.InstalledStatus) && hasAction(moduleUninstallAction)"
            size="small"
            type="danger"
            plain
            @click.stop="onActionClick('uninstall', record)"
          >
            卸载
          </el-button>
        </div>
      </div>
    </template>

    <template #card-empty>
      <div class="module-card__empty">暂无模块</div>
    </template>
  </OKanbanView>

  <el-dialog v-model="dialogVisible" width="680px" :close-on-click-modal="false" @close="onDialogClose">
    <template #header>
      <span>{{ dialogTitle }}</span>
    </template>

    <div v-if="dialogStep === 'plan'" class="module-dialog">
      <el-skeleton v-if="planLoading" :rows="6" animated />
      <template v-else>
        <el-alert v-if="plan?.blockers?.length" type="error" :closable="false" show-icon title="存在阻断项，需先处理后再继续" />
        <el-alert v-else type="info" :closable="false" show-icon title="请确认本次操作影响范围" />

        <div class="module-dialog__section">
          <div class="section-title">影响模块</div>
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
          <div class="section-title">风险提示</div>
          <ul class="section-list">
            <li v-for="risk in plan?.risks" :key="risk.code">
              <el-tag size="small" type="warning">{{ risk.code }}</el-tag>
              <span class="item-desc">{{ risk.message || '' }}</span>
            </li>
          </ul>
        </div>

        <div class="module-dialog__section" v-if="plan?.blockers?.length">
          <div class="section-title">阻断原因</div>
          <ul class="section-list">
            <li v-for="blocker in plan?.blockers" :key="blocker.code">
              <el-tag size="small" type="danger">{{ blocker.code }}</el-tag>
              <span class="item-desc">{{ blocker.message || '' }}</span>
            </li>
          </ul>
        </div>

        <div class="module-dialog__section" v-if="action === 'install'">
          <el-checkbox v-model="withDemo">包含 demo 数据</el-checkbox>
        </div>
      </template>
    </div>

    <div v-else class="module-dialog">
      <div class="module-dialog__section">
        <el-alert type="info" :closable="false" show-icon title="任务执行中，请稍候" v-if="dialogStep === 'progress'" />
        <el-alert v-else :type="resultAlertType" :closable="false" show-icon :title="resultTitle" />
      </div>
      <div class="module-dialog__section">
        <div class="section-title">执行状态</div>
        <div class="status-row">
          <span class="label">状态：</span>
          <el-tag size="small" :type="statusTagType(opStatus?.status)">{{ opStatus?.status || '—' }}</el-tag>
          <span class="label">结果：</span>
          <el-tag size="small" :type="resultTagType(opStatus?.resultStatus)">{{ opStatus?.resultStatus || '—' }}</el-tag>
        </div>
        <div class="status-row" v-if="opStatus?.summary">
          <span class="label">摘要：</span>
          <span class="value">{{ formatSummary(opStatus?.summary) }}</span>
        </div>
        <div class="status-row" v-if="opStatus?.failureKind && opStatus?.failureKind !== 'NONE'">
          <span class="label">失败类型：</span>
          <span class="value">{{ opStatus?.failureKind }}</span>
        </div>
        <div class="status-row" v-if="opStatus?.errorDomain || opStatus?.errorCode">
          <span class="label">错误：</span>
          <span class="value">{{ opStatus?.errorDomain || '—' }} / {{ opStatus?.errorCode || '—' }}</span>
        </div>
        <div class="status-row" v-if="opStatus?.reload_triggered">
          <span class="label">Reload：</span>
          <span class="value">{{ opStatus?.reload_failed ? '触发失败' : '已触发' }}</span>
        </div>
      </div>
      <div class="module-dialog__section" v-if="dialogStep === 'progress'">
        <el-divider />
        <div class="progress-note">请勿刷新页面，系统将自动更新状态。</div>
      </div>
    </div>

    <template #footer>
      <el-button @click="dialogVisible = false" :disabled="planLoading">取消</el-button>
      <el-button
        v-if="dialogStep === 'plan'"
        type="primary"
        :loading="planLoading || executeLoading"
        :disabled="!!plan?.blockers?.length"
        @click="submitOperation"
      >
        确认执行
      </el-button>
      <el-button v-else type="primary" @click="dialogVisible = false">完成</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type IrModule from '@/meta/service/models/ir_module';
import type IrModuleIndex from '@/meta/service/models/ir_module_index';
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

defineOptions({ name: 'ModuleKanbanView' });

/**
 * Props consumed by the module kanban view and its backing stores.
 */
const props = withDefaults(defineProps<{ store: WebModelStore<IrModuleIndex>; moduleStore: WebModelStore<IrModule>; showHeader?: boolean }>(), {
  showHeader: true,
});
const { store, showHeader } = props;
const moduleStore = props.moduleStore;

const keywordFields = ['ModuleName', 'Version', 'InstalledStatus', 'InstalledVersion'];

const router = useRouter();
const moduleInstallAction = defineAction('meta.action.module_install', {
  title: '安装模块',
  requires: [{ model: 'meta.IrModule', method: 'RequestInstall' }],
});
const moduleUpgradeAction = defineAction('meta.action.module_upgrade', {
  title: '升级模块',
  requires: [{ model: 'meta.IrModule', method: 'RequestUpgrade' }],
});
const moduleUninstallAction = defineAction('meta.action.module_uninstall', {
  title: '卸载模块',
  requires: [{ model: 'meta.IrModule', method: 'RequestUninstall' }],
});
const moduleSyncIndexAction = defineAction('meta.action.module_sync_index', {
  title: '同步模块索引',
  requires: [{ model: 'meta.IrModuleIndex', method: 'RequestSync' }],
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
 * Polling snapshot returned while a module action is executing.
 */
type OpStatusResp = {
  status: string;
  summary?: any;
  resultStatus?: 'SUCCEEDED' | 'FAILED';
  failureKind?: 'RETRYABLE' | 'NON_RETRYABLE' | 'NONE';
  reload_triggered?: boolean;
  reload_failed?: boolean;
  reload_web?: boolean;
  retryAfterMs?: number;
  errorDomain?: string;
  errorCode?: string;
};

const dialogVisible = ref(false);
const dialogStep = ref<'plan' | 'progress' | 'result'>('plan');
const planLoading = ref(false);
const executeLoading = ref(false);
const plan = ref<PlanOperationResp | null>(null);
const opStatus = ref<OpStatusResp | null>(null);
const action = ref<ModuleAction>('install');
const targetModule = ref<ClientModelProps<IrModuleIndex> | null>(null);
const withDemo = ref(false);
const pollTimer = ref<number | undefined>(undefined);
const pollIntervalMs = ref(1000);

/**
 * Builds the current dialog title from the selected module action.
 */
const dialogTitle = computed(() => {
  const actionLabel = action.value === 'install' ? '安装模块' : action.value === 'uninstall' ? '卸载模块' : '升级模块';
  return `${actionLabel} · ${targetModule.value?.ModuleName || ''}`.trim();
});

/**
 * Maps the latest operation snapshot to the dialog headline.
 */
const resultTitle = computed(() => {
  if (!opStatus.value) return '执行完成';
  if (opStatus.value.resultStatus === 'FAILED') return '操作失败';
  return opStatus.value.reload_failed ? '操作成功但重载失败' : '操作成功';
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
  if (available === false) return '不可用';
  const val = String(status || '').toLowerCase();
  if (val === 'installed') return '已安装';
  if (val === 'uninstalled') return '未安装';
  if (val === 'disabled') return '已禁用';
  if (val === 'broken') return '异常';
  if (val === 'succeeded') return '成功';
  if (val === 'failed') return '失败';
  if (val === 'dispatching') return '执行中';
  if (val === 'queued') return '排队中';
  return status || '未知';
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
function onCardClick(payload: { row: ClientModelProps<IrModuleIndex> }) {
  router.push(`/meta/modules/${payload.row.Id}`);
}

/**
 * Opens the operation dialog and loads the execution plan for a module action.
 */
async function onActionClick(nextAction: ModuleAction, record: ClientModelProps<IrModuleIndex>) {
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
    ElMessage.error(error?.message || '获取计划失败');
    dialogVisible.value = false;
  } finally {
    planLoading.value = false;
  }
}

/**
 * Dispatches the selected module action and starts polling its job status.
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
    await pollOpStatus(jobId);
  } catch (error: any) {
    executeLoading.value = false;
    dialogStep.value = 'result';
    opStatus.value = {
      status: 'failed',
      resultStatus: 'FAILED',
      errorDomain: 'CLIENT',
      errorCode: 'REQUEST_FAILED',
    } as OpStatusResp;
    ElMessage.error(error?.message || '操作请求失败');
  }
}

/**
 * Polls the backend operation job until a terminal state or reload request is observed.
 */
async function pollOpStatus(jobId: string) {
  clearPolling();
  pollIntervalMs.value = 1000;
  const startAt = Date.now();
  const maxDuration = 10 * 60 * 1000;
  let transientErrorNotified = false;

  const tick = async () => {
    if (!dialogVisible.value) return;
    try {
      const status = (await (moduleStore as any).GetOpStatus(jobId)) as OpStatusResp;
      opStatus.value = status;
      if (status?.status === 'succeeded' || status?.status === 'failed' || status?.status === 'cancelled') {
        dialogStep.value = 'result';
        if (status?.reload_web) {
          window.location.reload();
          return;
        }
        return;
      }
    } catch (error: any) {
      const message = String(error?.message || error || '').trim();
      const isTransient =
        message.includes('Failed to fetch') ||
        message.includes('NetworkError') ||
        message.includes('ERR_CONNECTION_REFUSED') ||
        message.includes('Load failed');
      if (isTransient) {
        if (!transientErrorNotified) {
          ElMessage.warning('服务正在重启，状态查询会自动重试');
          transientErrorNotified = true;
        }
      } else {
        ElMessage.error(message || '获取状态失败');
      }
    }

    if (Date.now() - startAt > maxDuration) {
      dialogStep.value = 'result';
      opStatus.value = {
        status: 'dispatching',
        resultStatus: undefined,
      } as OpStatusResp;
      ElMessage.warning('任务仍在后台执行，可稍后刷新查看');
      return;
    }

    const nextDelay = opStatus.value?.retryAfterMs ? Math.min(5000, Math.max(1000, opStatus.value.retryAfterMs)) : pollIntervalMs.value;
    pollIntervalMs.value = Math.min(5000, pollIntervalMs.value + 500);
    pollTimer.value = window.setTimeout(tick, nextDelay);
  };

  await tick();
}

/**
 * Clears the pending operation-status polling timer.
 */
function clearPolling() {
  if (pollTimer.value) {
    window.clearTimeout(pollTimer.value);
    pollTimer.value = undefined;
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
  clearPolling();
}

/**
 * Stops polling when the dialog is dismissed.
 */
function onDialogClose() {
  clearPolling();
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
    const jobId = await (store as any).RequestSync({ originType: 'local', force: true, ifStale: false });
    ElMessage.success(jobId ? `已触发同步任务：${jobId}` : '已触发同步任务');
  } catch (error: any) {
    ElMessage.error(error?.message || '触发同步失败');
  } finally {
    syncLoading.value = false;
  }
}

onBeforeUnmount(() => {
  clearPolling();
});

/**
 * Silently triggers a stale-aware index refresh on kanban entry.
 * Failures are suppressed to avoid blocking page usability.
 */
onMounted(async () => {
  try {
    await (store as any).RequestSync({ originType: 'registry', ifStale: true });
  } catch {
    // registry unavailable — silently skip, page remains usable
  }
  try {
    await (store as any).RequestSync({ originType: 'local', ifStale: true });
  } catch {
    // local scan failed — silently skip
  }
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
