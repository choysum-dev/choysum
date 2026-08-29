<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <el-dialog
    v-model="visible"
    :title="title"
    width="780px"
    destroy-on-close
    :close-on-click-modal="!busy"
    :close-on-press-escape="!busy"
    :before-close="handleBeforeClose"
    @closed="resetState"
    @open="loadCatalog"
  >
    <div class="import-panel">
      <el-steps :active="step" finish-status="success" align-center>
        <el-step :title="uploadStepTitle" />
        <el-step :title="previewStepTitle" />
        <el-step :title="importStepTitle" />
      </el-steps>

      <section v-if="step === 0" class="import-panel-section">
        <p class="import-panel-hint">{{ resolvedUploadHint }}</p>
        <p v-if="defaultFieldsHint" class="import-panel-hint" data-test="import-default-fields">
          {{ defaultFieldsLabel }}: {{ defaultFieldsHint }}
        </p>
        <el-alert v-if="catalogError" type="warning" :title="catalogError" show-icon :closable="false" class="import-panel-alert" />
        <el-upload drag accept=".csv,text/csv" :auto-upload="false" :limit="1" :on-change="onFileSelected" :on-remove="onFileRemoved">
          <div class="el-upload__text">{{ uploadDropText }}</div>
        </el-upload>
      </section>

      <section v-else-if="step === 1" class="import-panel-section">
        <p class="import-panel-hint">{{ mappingHint }}</p>
        <div v-if="mappingRows.length" class="import-mapping" data-test="import-mapping-table">
          <div class="import-mapping__row import-mapping__row--head">
            <div>{{ csvColumnLabel }}</div>
            <div>{{ importFieldLabel }}</div>
          </div>
          <div v-for="row in mappingRows" :key="row.header" class="import-mapping__row">
            <div class="import-mapping__header">{{ row.header }}</div>
            <el-select
              v-model="row.fieldPath"
              filterable
              clearable
              :placeholder="sameAsHeaderLabel"
              class="import-mapping__select"
              @change="onMappingChange"
            >
              <el-option
                v-for="opt in catalogOptions"
                :key="opt.path"
                :label="opt.label"
                :value="opt.path"
              />
            </el-select>
          </div>
        </div>
        <el-alert v-if="previewReport" :type="previewAlertType" :closable="false" show-icon class="import-panel-alert">
          <template #title>{{ previewSummary }}</template>
        </el-alert>
        <el-table v-if="previewMessages.length" :data="previewMessages" size="small" max-height="200" class="import-panel-table">
          <el-table-column prop="row" :label="rowLabel" width="72" />
          <el-table-column prop="field" :label="fieldLabel" width="120" />
          <el-table-column prop="code" :label="codeLabel" width="140" />
          <el-table-column prop="text" :label="messageLabel" min-width="220" />
        </el-table>
      </section>

      <section v-else class="import-panel-section">
        <el-result v-if="importDone" icon="success" :title="importSuccessTitle" :sub-title="importSuccessSubtitle" />
        <el-alert v-else-if="importError" type="error" :title="importError" show-icon :closable="false" />
      </section>
    </div>

    <template #footer>
      <el-button :disabled="busy" @click="visible = false">{{ cancelLabel }}</el-button>
      <el-button
        v-if="step === 0 || (step === 1 && !canImport && !!sourceRef)"
        type="primary"
        :loading="busy"
        :disabled="step === 0 ? !selectedFile : busy"
        @click="uploadAndPreview"
      >
        {{ previewActionLabel }}
      </el-button>
      <el-button v-else-if="step === 1" type="primary" :loading="busy" :disabled="!canImport" @click="commitImport">
        {{ importActionLabel }}
      </el-button>
      <el-button v-else-if="importDone" type="primary" @click="finish">{{ doneLabel }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { UploadFile } from 'element-plus';
import {
  describeImportFields,
  parseHeaders,
  previewImport,
  runImport,
  type ImportFieldNode,
  type ImportReport,
} from '@/core/web/import';
import { uploadImportCsv } from '@/core/web/import/upload_csv';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'ImportPanel' });

type MappingRow = { header: string; fieldPath: string };

const props = defineProps<{
  model: string;
  companyId?: string;
  columnMapping?: Record<string, string>;
  uploadHint?: string;
}>();

const emit = defineEmits<{
  (e: 'imported'): void;
}>();

const { _t } = createTranslate('web', { scope: 'web/import/ImportPanel' });

const visible = defineModel<boolean>({ default: false });

const title = _t('Import');
const uploadStepTitle = _t('Upload CSV');
const previewStepTitle = _t('Preview');
const importStepTitle = _t('Import');
const defaultUploadHint = _t('Upload a UTF-8 CSV. Map columns to importable fields from the catalog.');
const uploadDropText = _t('Drop CSV here or click to browse');
const defaultFieldsLabel = _t('Suggested columns');
const mappingHint = _t('Map each CSV column to an importable field. Leave blank to use the header as the field path.');
const csvColumnLabel = _t('CSV column');
const importFieldLabel = _t('Import field');
const sameAsHeaderLabel = _t('Same as header');
const rowLabel = _t('Row');
const fieldLabel = _t('Field');
const codeLabel = _t('Code');
const messageLabel = _t('Message');
const cancelLabel = _t('Cancel');
const previewActionLabel = _t('Preview');
const importActionLabel = _t('Import');
const doneLabel = _t('Done');
const importSuccessTitle = _t('Import completed');
const importSuccessSubtitle = _t('Rows were imported successfully.');

const resolvedUploadHint = computed(() => String(props.uploadHint || '').trim() || defaultUploadHint);

const step = ref(0);
const busy = ref(false);
const selectedFile = ref<File | null>(null);
const sourceRef = ref('');
const headers = ref<string[]>([]);
const mappingRows = ref<MappingRow[]>([]);
const catalogFields = ref<ImportFieldNode[]>([]);
const catalogDefaults = ref<string[]>([]);
const catalogError = ref('');
const previewReport = ref<ImportReport | null>(null);
const importDone = ref(false);
const importError = ref('');

let sessionToken = 0;
let previewAbort: AbortController | null = null;
let catalogAbort: AbortController | null = null;

const previewMessages = computed(() => previewReport.value?.messages ?? []);
const previewAlertType = computed(() => ((previewReport.value?.stats?.error ?? 0) > 0 ? 'warning' : 'success'));
const previewSummary = computed(() => {
  const stats = previewReport.value?.stats;
  if (!stats) return '';
  return `Preview: ${stats.ok ?? 0} ok, ${stats.error ?? 0} errors, ${stats.total ?? 0} total`;
});
const canImport = computed(
  () => !!sourceRef.value && previewReport.value != null && (previewReport.value.stats?.error ?? 0) === 0,
);
const defaultFieldsHint = computed(() => catalogDefaults.value.filter(Boolean).join(', '));

const catalogOptions = computed(() => {
  const out: Array<{ path: string; label: string }> = [];
  const walk = (nodes: ImportFieldNode[] | null | undefined) => {
    for (const node of nodes || []) {
      const path = String(node.path || '').trim();
      if (!path) continue;
      const label = String(node.label || path).trim() || path;
      const children = node.children;
      if (!children || children.length === 0) {
        out.push({ path, label: `${label} (${path})` });
      } else {
        for (const child of children) {
          const childPath = String(child.path || '').trim();
          if (!childPath) continue;
          const childLabel = String(child.label || childPath).trim() || childPath;
          out.push({ path: childPath, label: `${childLabel} (${childPath})` });
        }
      }
    }
  };
  walk(catalogFields.value);
  return out;
});

const resolvedMapping = computed(() => {
  const mapping: Record<string, string> = {};
  for (const row of mappingRows.value) {
    const header = String(row.header ?? '').trim();
    const fieldPath = String(row.fieldPath ?? '').trim();
    if (!header || !fieldPath) continue;
    mapping[header] = fieldPath;
  }
  return mapping;
});

function isActiveSession(token: number): boolean {
  return token === sessionToken;
}

function invalidateSession() {
  sessionToken += 1;
  previewAbort?.abort();
  previewAbort = null;
  catalogAbort?.abort();
  catalogAbort = null;
}

function onFileSelected(uploadFile: UploadFile) {
  selectedFile.value = uploadFile.raw ?? null;
}

function onFileRemoved() {
  selectedFile.value = null;
}

function flattenPaths(nodes: ImportFieldNode[]): string[] {
  const out: string[] = [];
  const walk = (list: ImportFieldNode[] | null | undefined) => {
    if (list == null) {
      return;
    }
    for (let i = 0; i < list.length; i += 1) {
      const node = list[i];
      const rawPath = node.path;
      const path = String(rawPath == null ? '' : rawPath).trim();
      const children = node.children;
      if (children != null && children.length > 0) {
        walk(children);
        continue;
      }
      if (path === '') {
        continue;
      }
      out.push(path);
    }
  };
  walk(nodes);
  return out;
}

function buildMappingRows(csvHeaders: string[]): MappingRow[] {
  const propMap = props.columnMapping ?? {};
  const paths = new Set(flattenPaths(catalogFields.value));
  return csvHeaders.map(header => {
    const fromProp = String(propMap[header] ?? '').trim();
    if (fromProp) {
      return { header, fieldPath: fromProp };
    }
    if (paths.has(header)) {
      return { header, fieldPath: header };
    }
    return { header, fieldPath: '' };
  });
}

function onMappingChange() {
  // Mapping edits invalidate the last dry-run; Import stays disabled until Preview again.
  previewReport.value = null;
}

async function loadCatalog() {
  catalogError.value = '';
  catalogFields.value = [];
  catalogDefaults.value = [];
  if (!String(props.model || '').trim()) {
    return;
  }
  const token = sessionToken;
  catalogAbort?.abort();
  const request = new AbortController();
  catalogAbort = request;
  try {
    const resp = await describeImportFields(props.model, request.signal);
    if (!isActiveSession(token) || catalogAbort !== request) {
      return;
    }
    catalogFields.value = resp.fields ?? [];
    catalogDefaults.value = resp.defaultFields ?? [];
  } catch (err) {
    if (!isActiveSession(token) || catalogAbort !== request) {
      return;
    }
    if (err instanceof DOMException && err.name === 'AbortError') {
      return;
    }
    catalogError.value = err instanceof Error ? err.message : String(err);
  }
}

function resetState() {
  invalidateSession();
  step.value = 0;
  busy.value = false;
  selectedFile.value = null;
  sourceRef.value = '';
  headers.value = [];
  mappingRows.value = [];
  previewReport.value = null;
  importDone.value = false;
  importError.value = '';
}

function handleBeforeClose(done: () => void) {
  if (busy.value) {
    return;
  }
  done();
}

async function uploadAndPreview() {
  if (!selectedFile.value) return;
  const token = sessionToken;
  busy.value = true;
  importError.value = '';
  previewAbort?.abort();
  previewAbort = new AbortController();
  const signal = previewAbort.signal;
  try {
    if (!catalogFields.value.length && !catalogError.value) {
      await loadCatalog();
      if (!isActiveSession(token)) return;
    }
    const ref = await uploadImportCsv({
      ownerModel: props.model,
      file: selectedFile.value,
    });
    if (!isActiveSession(token)) return;
    sourceRef.value = ref;
    const headerResp = await parseHeaders(ref, signal);
    if (!isActiveSession(token)) return;
    headers.value = headerResp.headers ?? [];
    mappingRows.value = buildMappingRows(headers.value);
    const previewResp = await previewImport(
      {
        targetModel: props.model,
        sourceRef: ref,
        companyId: props.companyId ?? '',
        columnMapping: resolvedMapping.value,
      },
      signal,
    );
    if (!isActiveSession(token)) return;
    previewReport.value = previewResp.report ?? null;
    step.value = 1;
  } catch (err) {
    if (!isActiveSession(token)) return;
    if (err instanceof DOMException && err.name === 'AbortError') {
      return;
    }
    importError.value = err instanceof Error ? err.message : String(err);
    step.value = 2;
  } finally {
    if (isActiveSession(token)) {
      busy.value = false;
    }
  }
}

async function commitImport() {
  if (!sourceRef.value) return;
  const token = sessionToken;
  busy.value = true;
  importError.value = '';
  try {
    const res = await runImport({
      targetModel: props.model,
      sourceRef: sourceRef.value,
      companyId: props.companyId ?? '',
      columnMapping: resolvedMapping.value,
    });
    if (!isActiveSession(token)) return;
    const errCount = res.report?.stats?.error ?? 0;
    if (errCount > 0) {
      const firstMsg = res.report?.messages?.find(m => m?.text)?.text;
      importError.value = firstMsg || `Import failed with ${errCount} error(s).`;
      step.value = 2;
      return;
    }
    importDone.value = true;
    step.value = 2;
    emit('imported');
  } catch (err) {
    if (!isActiveSession(token)) return;
    importError.value = err instanceof Error ? err.message : String(err);
    step.value = 2;
  } finally {
    if (isActiveSession(token)) {
      busy.value = false;
    }
  }
}

function finish() {
  visible.value = false;
}

watch(visible, value => {
  if (!value) {
    resetState();
  }
});
</script>

<style scoped>
.import-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-top: 8px;
}

.import-panel-section {
  min-height: 180px;
}

.import-panel-hint {
  margin: 0 0 12px;
  color: var(--el-text-color-secondary);
}

.import-panel-table {
  margin-top: 12px;
}

.import-panel-alert {
  margin-top: 12px;
}

.import-mapping {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}

.import-mapping__row {
  display: grid;
  grid-template-columns: minmax(120px, 1fr) minmax(180px, 1.4fr);
  gap: 8px;
  align-items: center;
}

.import-mapping__header {
  font-size: 13px;
  word-break: break-all;
}

.import-mapping__select {
  width: 100%;
}
</style>
