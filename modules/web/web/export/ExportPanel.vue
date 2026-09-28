<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Dialog v-model:open="visible">
    <DialogContent class="export-panel-dialog" @open-auto-focus.prevent>
      <DialogTitle>{{ title }}</DialogTitle>
      <div class="export-panel">
        <p class="export-panel-scope">{{ scopeSummary }}</p>

        <details class="export-panel-fields" :open="customFieldsOpen.includes('fields')" @toggle="onFieldsToggle">
          <summary>{{ customFieldsLabel }}</summary>
          <div v-if="templatesEnabled" class="export-panel-templates">
            <div class="export-panel-template-row">
              <select v-model="selectedTemplateId" class="export-panel-template-select">
                <option value="">{{ templateSelectLabel }}</option>
                <option v-for="item in exportTemplateItems" :key="item.Id" :value="item.Id">
                  {{ item.shared ? `${item.Name} (${sharedTemplateLabel})` : item.Name }}
                </option>
              </select>
              <ChoyButton size="sm" variant="outline" :disabled="!selectedTemplateId || busy" @click="applySelectedTemplate">{{ loadTemplateLabel }}</ChoyButton>
              <ChoyButton size="sm" variant="destructive" :disabled="!selectedTemplateCanDelete || busy" @click="deleteSelectedTemplate">{{ deleteTemplateLabel }}</ChoyButton>
            </div>
            <div class="export-panel-template-row">
              <input v-model="templateSaveName" :placeholder="templateNameLabel" class="export-panel-template-name" />
              <label class="inline-flex items-center gap-2">
                <input type="checkbox" v-model="templateSaveShared" />
                {{ sharedTemplateLabel }}
              </label>
              <ChoyButton size="sm" variant="outline" :disabled="!canSaveTemplate || busy" @click="saveCurrentTemplate">{{ saveTemplateLabel }}</ChoyButton>
            </div>
            <p v-if="exportTemplatesLoadError" class="export-panel-hint">{{ exportTemplatesLoadError }}</p>
          </div>
          <ul v-if="fieldTree.length" class="export-panel-field-list">
            <li v-for="node in flatFieldNodes" :key="node.path">
              <label class="inline-flex items-center gap-2">
                <input type="checkbox" :checked="selectedFieldPaths.includes(node.path)" @change="toggleFieldPath(node.path, ($event.target as HTMLInputElement).checked)" />
                <span>{{ node.label }}</span>
              </label>
            </li>
          </ul>
          <p v-else-if="fieldsLoading" class="export-panel-hint">{{ loadingFieldsLabel }}</p>
          <p v-else class="export-panel-hint">{{ noFieldsLabel }}</p>
        </details>

        <div v-if="previewReport" role="alert" class="export-panel-alert">{{ previewSummary }}</div>
        <div v-if="exportDone" class="export-panel-success">
          <strong>{{ exportSuccessTitle }}</strong>
          <p>{{ exportSuccessSubtitle }}</p>
        </div>
        <div v-else-if="exportError" role="alert" class="export-panel-alert">{{ exportError }}</div>
      </div>

      <div class="export-panel-footer">
        <ChoyButton size="sm" variant="outline" :disabled="busy" @click="visible = false">{{ cancelLabel }}</ChoyButton>
        <ChoyButton v-if="!exportDone" size="sm" variant="outline" :disabled="busy" @click="runPreview">{{ previewActionLabel }}</ChoyButton>
        <ChoyButton v-if="!exportDone" size="sm" variant="default" :disabled="busy" @click="commitExport">{{ exportActionLabel }}</ChoyButton>
        <ChoyButton v-else size="sm" variant="default" @click="visible = false">{{ doneLabel }}</ChoyButton>
      </div>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import Dialog from '@/web/web/components/vendor/ui/dialog/Dialog.vue';
import DialogContent from '@/web/web/components/vendor/ui/dialog/DialogContent.vue';
import DialogTitle from '@/web/web/components/vendor/ui/dialog/DialogTitle.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { describeExportFields, previewExport, runExport, ExportMode, type ExportFieldNode, type ExportReport } from '@/core/web/export/client';
import { downloadExportCsvBytes, suggestExportFileName } from '@/core/web/export/download_csv';
import { normalizeExportFieldPaths } from '@/core/web/export/field_paths';
import { exportReportErrorText, exportReportHasErrors, exportPreviewSummary } from '@/core/web/export/report';
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

const { _t } = createTranslate('web', { scope: 'web/export/ExportPanel' });

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
const fieldTreeRef = ref<HTMLElement | null>(null);
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
  const paths = selectedFieldPaths.value.map(String).filter(Boolean);
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


const flatFieldNodes = computed(() => {
  const out: Array<{ path: string; label: string }> = [];
  const walk = (nodes: any[], prefix = '') => {
    for (const n of nodes || []) {
      const label = String(n.label ?? n.path ?? '');
      if (n.path) out.push({ path: String(n.path), label: prefix ? `${prefix} / ${label}` : label });
      if (Array.isArray(n.children) && n.children.length) walk(n.children, prefix ? `${prefix} / ${label}` : label);
    }
  };
  walk(fieldTree.value as any[]);
  return out;
});

function toggleFieldPath(path: string, on: boolean) {
  const set = new Set(selectedFieldPaths.value);
  if (on) set.add(path);
  else set.delete(path);
  selectedFieldPaths.value = Array.from(set);
  onFieldCheck();
}

function onFieldsToggle(ev: Event) {
  const open = (ev.target as HTMLDetailsElement).open;
  customFieldsOpen.value = open ? ['fields'] : [];
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
