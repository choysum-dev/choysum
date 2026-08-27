<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <el-dialog
    v-model="visible"
    :title="title"
    width="640px"
    destroy-on-close
    :close-on-click-modal="!busy"
    :close-on-press-escape="!busy"
    @open="onOpen"
    @closed="resetState"
  >
    <div class="export-panel">
      <p class="export-panel-scope">{{ scopeSummary }}</p>

      <el-collapse v-model="customFieldsOpen">
        <el-collapse-item :title="customFieldsLabel" name="fields">
          <el-tree
            v-if="fieldTree.length"
            ref="fieldTreeRef"
            show-checkbox
            node-key="path"
            :data="fieldTree"
            :props="treeProps"
            :default-checked-keys="selectedFieldPaths"
            @check="onFieldCheck"
          />
          <p v-else-if="fieldsLoading" class="export-panel-hint">{{ loadingFieldsLabel }}</p>
          <p v-else class="export-panel-hint">{{ noFieldsLabel }}</p>
        </el-collapse-item>
      </el-collapse>

      <el-alert v-if="previewReport" :type="previewAlertType" :closable="false" show-icon class="export-panel-alert">
        <template #title>{{ previewSummary }}</template>
      </el-alert>

      <el-result v-if="exportDone" icon="success" :title="exportSuccessTitle" :sub-title="exportSuccessSubtitle" />
      <el-alert v-else-if="exportError" type="error" :title="exportError" show-icon :closable="false" />
    </div>

    <template #footer>
      <el-button :disabled="busy" @click="visible = false">{{ cancelLabel }}</el-button>
      <el-button v-if="!exportDone" plain :loading="busy" @click="runPreview">{{ previewActionLabel }}</el-button>
      <el-button v-if="!exportDone" type="primary" :loading="busy" @click="commitExport">{{ exportActionLabel }}</el-button>
      <el-button v-else type="primary" @click="visible = false">{{ doneLabel }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type ElTree from 'element-plus/es/components/tree/src/tree.vue';
import { describeExportFields, previewExport, runExport, ExportMode, type ExportFieldNode, type ExportReport } from './client';
import { downloadExportCsvBytes, suggestExportFileName } from './download_csv';
import { normalizeExportFieldPaths } from './field_paths';
import { exportReportErrorText, exportReportHasErrors, exportPreviewSummary } from './report';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'ExportPanel' });

const props = defineProps<{
  model: string;
  companyId?: string;
  ids?: string[];
  domain?: string;
  defaultFields?: string[];
  filteredCount?: number;
}>();

const { _t } = createTranslate('core', { scope: 'web/export/ExportPanel' });

const visible = defineModel<boolean>({ default: false });

const title = _t('Export');
const customFieldsLabel = _t('Customize fields…');
const cancelLabel = _t('Cancel');
const previewActionLabel = _t('Preview');
const exportActionLabel = _t('Export CSV');
const doneLabel = _t('Done');
const exportSuccessTitle = _t('Export completed');
const exportArtifactSubtitle = _t('CSV stored as document %ref.');
const loadingFieldsLabel = _t('Loading fields…');
const noFieldsLabel = _t('No exportable fields were returned.');
const selectedScopeLabel = _t('Export %count selected row(s).');
const filteredScopeLabel = _t('Export %count row(s) matching the current filters.');
const filteredScopeUnknownLabel = _t('Export rows matching the current filters.');

const busy = ref(false);
const customFieldsOpen = ref<string[]>([]);
const fieldsLoading = ref(false);
const fieldTree = ref<Array<{ path: string; label: string; children?: Array<{ path: string; label: string }> }>>([]);
const selectedFieldPaths = ref<string[]>([]);
const previewReport = ref<ExportReport | null>(null);
const exportDone = ref(false);
const exportError = ref('');
const exportSuccessSubtitle = ref('');
const fieldTreeRef = ref<InstanceType<typeof ElTree> | null>(null);

let sessionToken = 0;
let activeRpcAbort: AbortController | null = null;

function isAbortError(err: unknown): boolean {
  if (err instanceof DOMException && err.name === 'AbortError') {
    return true;
  }
  return err instanceof Error && err.name === 'AbortError';
}

function abortActiveRpc() {
  activeRpcAbort?.abort();
  activeRpcAbort = null;
}

function beginActiveRpc(): AbortSignal {
  abortActiveRpc();
  activeRpcAbort = new AbortController();
  return activeRpcAbort.signal;
}

function endActiveRpc(signal: AbortSignal) {
  if (activeRpcAbort?.signal === signal) {
    activeRpcAbort = null;
  }
}

function shouldIgnoreRpcError(token: number, err: unknown): boolean {
  return !isActiveSession(token) || isAbortError(err);
}

const treeProps = { label: 'label', children: 'children' };

const scopeSummary = computed(() => {
  const selected = (props.ids ?? []).filter(Boolean);
  if (selected.length > 0) {
    return selectedScopeLabel.replace('%count', String(selected.length));
  }
  const count = Number(props.filteredCount ?? 0);
  if (count > 0) {
    return filteredScopeLabel.replace('%count', String(count));
  }
  return filteredScopeUnknownLabel;
});

const effectiveFields = computed(() => {
  const checked = fieldTreeRef.value?.getCheckedKeys?.(false) as string[] | undefined;
  const paths = (checked?.length ? checked : selectedFieldPaths.value).map(String).filter(Boolean);
  if (paths.length > 0) {
    return paths;
  }
  return [...(props.defaultFields ?? [])];
});

const previewAlertType = computed(() => (exportReportHasErrors(previewReport.value) ? 'warning' : 'info'));
const previewSummary = computed(() => exportPreviewSummary(previewReport.value));

function buildRunInput() {
  const ids = (props.ids ?? []).filter(Boolean);
  return {
    model: props.model,
    companyId: props.companyId,
    fields: normalizeExportFieldPaths(effectiveFields.value),
    ids: ids.length > 0 ? ids : [],
    domain: ids.length > 0 ? '' : props.domain ?? '',
    mode: ExportMode.DATA,
  };
}

function mapFieldNodes(nodes: ExportFieldNode[]): Array<{ path: string; label: string; children?: Array<{ path: string; label: string }> }> {
  return (nodes ?? [])
    .filter(node => String(node.path ?? '').trim())
    .map(node => ({
      path: node.path,
      label: node.label || node.path,
      children: node.children?.length ? mapFieldNodes(node.children) : undefined,
    }));
}

function isActiveSession(token: number): boolean {
  return token === sessionToken;
}

function invalidateSession() {
  abortActiveRpc();
  sessionToken += 1;
  busy.value = false;
}

async function loadFields() {
  const token = sessionToken;
  const signal = beginActiveRpc();
  fieldsLoading.value = true;
  try {
    const resp = await describeExportFields(props.model, signal);
    if (!isActiveSession(token)) {
      return;
    }
    fieldTree.value = mapFieldNodes(resp.fields);
    const defaults = (props.defaultFields?.length ? props.defaultFields : resp.defaultFields) ?? [];
    selectedFieldPaths.value = normalizeExportFieldPaths(defaults);
  } catch (err) {
    if (shouldIgnoreRpcError(token, err)) {
      return;
    }
    exportError.value = err instanceof Error ? err.message : String(err);
  } finally {
    endActiveRpc(signal);
    if (isActiveSession(token)) {
      fieldsLoading.value = false;
    }
  }
}

function onOpen() {
  invalidateSession();
  busy.value = false;
  exportError.value = '';
  exportDone.value = false;
  previewReport.value = null;
  customFieldsOpen.value = [];
  void loadFields();
}

function resetState() {
  invalidateSession();
  busy.value = false;
  exportDone.value = false;
  exportError.value = '';
  previewReport.value = null;
  fieldTree.value = [];
  selectedFieldPaths.value = [];
}

function onFieldCheck() {
  invalidateSession();
  previewReport.value = null;
}

async function runPreview() {
  const token = sessionToken;
  const signal = beginActiveRpc();
  busy.value = true;
  exportError.value = '';
  try {
    const resp = await previewExport(buildRunInput(), signal);
    if (!isActiveSession(token)) {
      return;
    }
    previewReport.value = resp.report ?? null;
  } catch (err) {
    if (shouldIgnoreRpcError(token, err)) {
      return;
    }
    exportError.value = err instanceof Error ? err.message : String(err);
  } finally {
    endActiveRpc(signal);
    if (isActiveSession(token)) {
      busy.value = false;
    }
  }
}

async function commitExport() {
  const token = sessionToken;
  const signal = beginActiveRpc();
  busy.value = true;
  exportError.value = '';
  try {
    const resp = await runExport(buildRunInput(), signal);
    if (!isActiveSession(token)) {
      return;
    }
    const report = resp.report ?? null;
    if (exportReportHasErrors(report)) {
      exportError.value = exportReportErrorText(report);
      return;
    }
    const stats = report?.stats;
    exportSuccessSubtitle.value = stats ? `${stats.ok ?? 0} row(s) exported.` : '';
    if (resp.csvData?.length) {
      downloadExportCsvBytes(resp.csvData, suggestExportFileName(props.model));
    } else if (report?.artifactRef) {
      exportSuccessSubtitle.value = exportArtifactSubtitle.replace('%ref', report.artifactRef);
    }
    exportDone.value = true;
  } catch (err) {
    if (shouldIgnoreRpcError(token, err)) {
      return;
    }
    exportError.value = err instanceof Error ? err.message : String(err);
  } finally {
    endActiveRpc(signal);
    if (isActiveSession(token)) {
      busy.value = false;
    }
  }
}

watch(
  () => props.defaultFields,
  value => {
    if (Array.isArray(value) && value.length > 0 && selectedFieldPaths.value.length === 0) {
      selectedFieldPaths.value = normalizeExportFieldPaths(value);
    }
  },
  { immediate: true }
);
</script>

<style scoped>
.export-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.export-panel-scope {
  margin: 0;
  color: var(--el-text-color-regular);
}

.export-panel-hint {
  margin: 0;
  color: var(--el-text-color-secondary);
}

.export-panel-alert {
  margin-top: 4px;
}
</style>
