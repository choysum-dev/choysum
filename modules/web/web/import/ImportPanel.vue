<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Dialog v-model:open="visible">
    <DialogContent class="import-panel-dialog" @open-auto-focus.prevent>
      <DialogTitle>{{ title }}</DialogTitle>
      <ol class="import-panel-steps">
        <li :class="{ active: step === 0 }">{{ uploadStepTitle }}</li>
        <li :class="{ active: step === 1 }">{{ previewStepTitle }}</li>
        <li :class="{ active: step === 2 }">{{ importStepTitle }}</li>
      </ol>

      <section v-if="step === 0" class="import-panel-section">
        <p class="import-panel-hint">{{ resolvedUploadHint }}</p>
        <p v-if="defaultFieldsHint" class="import-panel-hint" data-test="import-default-fields">{{ defaultFieldsHint }}</p>
        <div v-if="catalogError" role="alert" class="import-panel-alert">{{ catalogError }}</div>
        <label class="import-upload">
          <input type="file" accept=".csv,text/csv" @change="onNativeFile" />
          <div>{{ uploadDropText }}</div>
        </label>
      </section>

      <section v-else-if="step === 1" class="import-panel-section">
        <p class="import-panel-hint">{{ mappingHint }}</p>
        <div v-for="row in mappingRows" :key="row.header" class="import-panel-map-row">
          <span>{{ row.header }}</span>
          <select v-model="row.fieldPath" @change="onMappingChange">
            <option value="">{{ sameAsHeaderLabel }}</option>
            <option v-for="f in catalogOptions" :key="f.path" :value="f.path">{{ f.label }}</option>
          </select>
        </div>
        <div v-if="previewReport" role="alert" class="import-panel-alert">{{ previewSummary }}</div>
        <table v-if="previewMessages.length" class="import-panel-table">
          <thead>
            <tr>
              <th>{{ rowLabel }}</th>
              <th>{{ fieldLabel }}</th>
              <th>{{ codeLabel }}</th>
              <th>{{ messageLabel }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(msg, i) in previewMessages" :key="i">
              <td>{{ msg.row }}</td>
              <td>{{ msg.field }}</td>
              <td>{{ msg.code }}</td>
              <td>{{ msg.text }}</td>
            </tr>
          </tbody>
        </table>
      </section>

      <section v-else class="import-panel-section">
        <div v-if="importDone" class="import-panel-success">
          <strong>{{ importSuccessTitle }}</strong>
          <p>{{ importSuccessSubtitle }}</p>
        </div>
        <div v-else-if="importError" role="alert" class="import-panel-alert">{{ importError }}</div>
      </section>

      <div class="import-panel-footer">
        <ChoyButton size="sm" variant="outline" :disabled="busy" @click="visible = false">{{ cancelLabel }}</ChoyButton>
        <ChoyButton
          v-if="step === 0"
          size="sm"
          variant="default"
          :disabled="!selectedFile || busy"
          @click="uploadAndPreview"
        >
          {{ previewActionLabel }}
        </ChoyButton>
        <ChoyButton v-else-if="step === 1" size="sm" variant="default" :disabled="!canImport || busy" @click="commitImport">
          {{ importActionLabel }}
        </ChoyButton>
        <ChoyButton v-else-if="importDone" size="sm" variant="default" @click="finish">{{ doneLabel }}</ChoyButton>
      </div>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import Dialog from '@/web/web/components/vendor/ui/dialog/Dialog.vue';
import DialogContent from '@/web/web/components/vendor/ui/dialog/DialogContent.vue';
import DialogTitle from '@/web/web/components/vendor/ui/dialog/DialogTitle.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
type UploadFile = { raw?: File | null; name?: string };
import {
  describeImportFields,
  parseHeaders,
  previewImport,
  runImport,
  type ImportFieldNode,
  type ImportReport
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
    if (nodes == null) {
      return;
    }
    for (const node of nodes) {
      const path = String(node.path == null ? '' : node.path).trim();
      const children = node.children;
      if (children != null && children.length > 0) {
        walk(children);
        continue;
      }
      if (!path) continue;
      const label = String(node.label || path).trim() || path;
      out.push({ path, label: `${label} (${path})` });
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

function clearUploadDerivedState() {
  // New or cleared file must not reuse a prior upload sourceRef / mapping / preview.
  sourceRef.value = '';
  headers.value = [];
  mappingRows.value = [];
  previewReport.value = null;
}

function onFileSelected(uploadFile: UploadFile) {
  selectedFile.value = uploadFile.raw ?? null;
  // Drop in-flight upload/preview so a pending completion cannot write a stale sourceRef.
  // Only invalidate while busy: a first select must not abort the dialog-open catalog load.
  if (busy.value) {
    invalidateSession();
    busy.value = false;
  }
  clearUploadDerivedState();
}


function onNativeFile(ev: Event) {
  const input = ev.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) {
    onFileRemoved();
    return;
  }
  onFileSelected({ raw: file, name: file.name } as UploadFile);
}
function onFileRemoved() {
  selectedFile.value = null;
  if (busy.value) {
    invalidateSession();
    busy.value = false;
  }
  clearUploadDerivedState();
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
    // Re-preview after mapping edits reuses the uploaded source and keeps user row edits.
    let ref = sourceRef.value;
    if (!ref) {
      ref = await uploadImportCsv({
        ownerModel: props.model,
        file: selectedFile.value,
      });
      if (!isActiveSession(token)) return;
      sourceRef.value = ref;
      const headerResp = await parseHeaders(ref, signal);
      if (!isActiveSession(token)) return;
      headers.value = headerResp.headers ?? [];
      mappingRows.value = buildMappingRows(headers.value);
    }
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
  color: var(--choy-color-muted-foreground);
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
