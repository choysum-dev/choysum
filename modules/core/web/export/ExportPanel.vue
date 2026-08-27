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
          <div v-if="templatesEnabled" class="export-panel-templates">
            <div class="export-panel-template-row">
              <el-select
                v-model="selectedTemplateId"
                clearable
                filterable
                :loading="exportTemplatesLoading"
                :placeholder="templateSelectLabel"
                class="export-panel-template-select"
              >
                <el-option
                  v-for="item in exportTemplateItems"
                  :key="item.Id"
                  :label="item.shared ? `${item.Name} (${sharedTemplateLabel})` : item.Name"
                  :value="item.Id"
                />
              </el-select>
              <el-button :disabled="!selectedTemplateId || busy" @click="applySelectedTemplate">{{ loadTemplateLabel }}</el-button>
              <el-button
                :disabled="!selectedTemplateCanDelete || busy"
                type="danger"
                plain
                @click="deleteSelectedTemplate"
              >
                {{ deleteTemplateLabel }}
              </el-button>
            </div>
            <div class="export-panel-template-row">
              <el-input v-model="templateSaveName" :placeholder="templateNameLabel" class="export-panel-template-name" />
              <el-checkbox v-model="templateSaveShared">{{ sharedTemplateLabel }}</el-checkbox>
              <el-button :disabled="!canSaveTemplate || busy" @click="saveCurrentTemplate">{{ saveTemplateLabel }}</el-button>
            </div>
            <p v-if="exportTemplatesLoadError" class="export-panel-hint">{{ exportTemplatesLoadError }}</p>
          </div>
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
import { computed, nextTick, ref, watch } from 'vue';
import type ElTree from 'element-plus/es/components/tree/src/tree.vue';
import { describeExportFields, previewExport, runExport, ExportMode, type ExportFieldNode, type ExportReport } from './client';
import { downloadExportCsvBytes, suggestExportFileName } from './download_csv';
import { normalizeExportFieldPaths } from './field_paths';
import { exportReportErrorText, exportReportHasErrors, exportPreviewSummary } from './report';
import { createTranslate } from '@/web/web/i18n';
import { useExportTemplates } from '@/web/web/composables/export/useExportTemplates';

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
const templateSelectLabel = _t('Saved templates');
const loadTemplateLabel = _t('Load template');
const saveTemplateLabel = _t('Save template');
const deleteTemplateLabel = _t('Delete');
const templateNameLabel = _t('Template name');
const sharedTemplateLabel = _t('Shared');
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
const exportTemplates = useExportTemplates(() => props.model);
const { templates: exportTemplateItems, loading: exportTemplatesLoading, loadError: exportTemplatesLoadError, load: loadExportTemplates, apply: applyExportTemplate, saveCurrent: saveExportTemplate, remove: removeExportTemplate } = exportTemplates;
const selectedTemplateId = ref('');
const templateSaveName = ref('');
const templateSaveShared = ref(false);
const pendingFieldPaths = ref<string[] | null>(null);

const templatesEnabled = computed(() => String(props.model || '').includes('.'));

const selectedTemplateCanDelete = computed(() => {
  const id = String(selectedTemplateId.value || '').trim();
  if (!id) return false;
  return exportTemplates.templates.value.some(item => item.Id === id && item.canDelete);
});

const canSaveTemplate = computed(() => {
  const name = String(templateSaveName.value || '').trim();
  return name.length > 0 && effectiveFields.value.length > 0;
});

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
    const paths = pendingFieldPaths.value ?? defaults;
    pendingFieldPaths.value = null;
    selectedFieldPaths.value = normalizeExportFieldPaths(paths);
    await nextTick();
    fieldTreeRef.value?.setCheckedKeys?.(selectedFieldPaths.value);
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

function applyFieldPaths(paths: string[]) {
  const normalized = normalizeExportFieldPaths(paths);
  selectedFieldPaths.value = normalized;
  fieldTreeRef.value?.setCheckedKeys?.(normalized);
  invalidateSession();
  previewReport.value = null;
}

function applySelectedTemplate() {
  const id = String(selectedTemplateId.value || '').trim();
  const template = exportTemplateItems.value.find(item => item.Id === id);
  if (!template) return;
  const paths = applyExportTemplate(template);
  if (fieldsLoading.value || fieldTree.value.length === 0) {
    pendingFieldPaths.value = paths;
    selectedFieldPaths.value = normalizeExportFieldPaths(paths);
    return;
  }
  applyFieldPaths(paths);
}

async function saveCurrentTemplate() {
  if (busy.value) return;
  busy.value = true;
  exportError.value = '';
  try {
    const saved = await saveExportTemplate({
      name: templateSaveName.value,
      shared: templateSaveShared.value,
      fields: effectiveFields.value,
    });
    if (saved) {
      selectedTemplateId.value = saved.Id;
      templateSaveName.value = '';
      templateSaveShared.value = false;
    }
  } catch (err) {
    exportError.value = err instanceof Error ? err.message : String(err);
  } finally {
    busy.value = false;
  }
}

async function deleteSelectedTemplate() {
  const id = String(selectedTemplateId.value || '').trim();
  if (!id || busy.value) return;
  busy.value = true;
  exportError.value = '';
  try {
    await removeExportTemplate(id);
    selectedTemplateId.value = '';
  } catch (err) {
    exportError.value = err instanceof Error ? err.message : String(err);
  } finally {
    busy.value = false;
  }
}

function onOpen() {
  invalidateSession();
  busy.value = false;
  exportError.value = '';
  exportDone.value = false;
  previewReport.value = null;
  customFieldsOpen.value = [];
  selectedTemplateId.value = '';
  templateSaveName.value = '';
  templateSaveShared.value = false;
  pendingFieldPaths.value = null;
  void loadFields();
}

watch(
  customFieldsOpen,
  value => {
    if (Array.isArray(value) && value.includes('fields') && templatesEnabled.value) {
      void loadExportTemplates();
    }
  },
  { deep: true }
);

function resetState() {
  invalidateSession();
  busy.value = false;
  exportDone.value = false;
  exportError.value = '';
  previewReport.value = null;
  fieldTree.value = [];
  selectedFieldPaths.value = [];
  selectedTemplateId.value = '';
  templateSaveName.value = '';
  templateSaveShared.value = false;
  pendingFieldPaths.value = null;
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

.export-panel-templates {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}

.export-panel-template-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.export-panel-template-select {
  min-width: 220px;
  flex: 1 1 220px;
}

.export-panel-template-name {
  min-width: 180px;
  flex: 1 1 180px;
}

.export-panel-alert {
  margin-top: 4px;
}
</style>
