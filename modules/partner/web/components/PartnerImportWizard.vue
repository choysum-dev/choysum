<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <el-dialog
    v-model="visible"
    :title="title"
    width="720px"
    destroy-on-close
    :close-on-click-modal="!busy"
    :close-on-press-escape="!busy"
    :before-close="handleBeforeClose"
    @closed="resetState"
  >
    <div class="partner-import-wizard">
      <el-steps :active="step" finish-status="success" align-center>
        <el-step :title="uploadStepTitle" />
        <el-step :title="previewStepTitle" />
        <el-step :title="importStepTitle" />
      </el-steps>

      <section v-if="step === 0" class="partner-import-section">
        <p class="partner-import-hint">{{ uploadHint }}</p>
        <el-upload drag accept=".csv,text/csv" :auto-upload="false" :limit="1" :on-change="onFileSelected" :on-remove="onFileRemoved">
          <div class="el-upload__text">{{ uploadDropText }}</div>
        </el-upload>
      </section>

      <section v-else-if="step === 1" class="partner-import-section">
        <p v-if="headers.length" class="partner-import-hint">{{ headersLabel }}: {{ headers.join(', ') }}</p>
        <el-alert v-if="previewReport" :type="previewAlertType" :closable="false" show-icon>
          <template #title>{{ previewSummary }}</template>
        </el-alert>
        <el-table v-if="previewMessages.length" :data="previewMessages" size="small" max-height="240" class="partner-import-table">
          <el-table-column prop="row" :label="rowLabel" width="72" />
          <el-table-column prop="field" :label="fieldLabel" width="120" />
          <el-table-column prop="code" :label="codeLabel" width="140" />
          <el-table-column prop="text" :label="messageLabel" min-width="220" />
        </el-table>
      </section>

      <section v-else class="partner-import-section">
        <el-result v-if="importDone" icon="success" :title="importSuccessTitle" :sub-title="importSuccessSubtitle" />
        <el-alert v-else-if="importError" type="error" :title="importError" show-icon :closable="false" />
      </section>
    </div>

    <template #footer>
      <el-button :disabled="busy" @click="visible = false">{{ cancelLabel }}</el-button>
      <el-button v-if="step === 0" type="primary" :loading="busy" :disabled="!selectedFile" @click="uploadAndPreview">
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
import { parseHeaders, previewImport, runImport, type ImportReport } from '@/core/web/import';
import { uploadImportCsv } from '@/core/web/import/upload_csv';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'PartnerImportWizard' });

const props = defineProps<{
  companyId?: string;
}>();

const emit = defineEmits<{
  (e: 'imported'): void;
}>();

const { _t } = createTranslate('partner', { scope: 'web/components/PartnerImportWizard' });

const visible = defineModel<boolean>({ default: false });

const title = _t('Import Partners');
const uploadStepTitle = _t('Upload CSV');
const previewStepTitle = _t('Preview');
const importStepTitle = _t('Import');
const uploadHint = _t('Upload a UTF-8 CSV with columns Name, Code, IsActive, CustomerRank, SupplierRank.');
const uploadDropText = _t('Drop CSV here or click to browse');
const headersLabel = _t('Detected headers');
const rowLabel = _t('Row');
const fieldLabel = _t('Field');
const codeLabel = _t('Code');
const messageLabel = _t('Message');
const cancelLabel = _t('Cancel');
const previewActionLabel = _t('Preview');
const importActionLabel = _t('Import');
const doneLabel = _t('Done');
const importSuccessTitle = _t('Import completed');
const importSuccessSubtitle = _t('Partner rows were imported successfully.');

const step = ref(0);
const busy = ref(false);
const selectedFile = ref<File | null>(null);
const sourceRef = ref('');
const headers = ref<string[]>([]);
const previewReport = ref<ImportReport | null>(null);
const importDone = ref(false);
const importError = ref('');

let sessionToken = 0;
let previewAbort: AbortController | null = null;

const previewMessages = computed(() => previewReport.value?.messages ?? []);
const previewAlertType = computed(() => ((previewReport.value?.stats?.error ?? 0) > 0 ? 'warning' : 'success'));
const previewSummary = computed(() => {
  const stats = previewReport.value?.stats;
  if (!stats) return '';
  return `Preview: ${stats.ok ?? 0} ok, ${stats.error ?? 0} errors, ${stats.total ?? 0} total`;
});
const canImport = computed(() => !!sourceRef.value && (previewReport.value?.stats?.error ?? 0) === 0);

function isActiveSession(token: number): boolean {
  return token === sessionToken;
}

function invalidateSession() {
  sessionToken += 1;
  previewAbort?.abort();
  previewAbort = null;
}

function onFileSelected(uploadFile: UploadFile) {
  selectedFile.value = uploadFile.raw ?? null;
}

function onFileRemoved() {
  selectedFile.value = null;
}

function resetState() {
  invalidateSession();
  step.value = 0;
  busy.value = false;
  selectedFile.value = null;
  sourceRef.value = '';
  headers.value = [];
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
    const ref = await uploadImportCsv({
      ownerModel: 'partner.Partner',
      file: selectedFile.value,
    });
    if (!isActiveSession(token)) return;
    sourceRef.value = ref;
    const headerResp = await parseHeaders(ref, signal);
    if (!isActiveSession(token)) return;
    headers.value = headerResp.headers ?? [];
    const previewResp = await previewImport(
      {
        targetModel: 'partner.Partner',
        sourceRef: ref,
        companyId: props.companyId ?? '',
        columnMapping: {},
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
      targetModel: 'partner.Partner',
      sourceRef: sourceRef.value,
      companyId: props.companyId ?? '',
      columnMapping: {},
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
.partner-import-wizard {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-top: 8px;
}

.partner-import-section {
  min-height: 180px;
}

.partner-import-hint {
  margin: 0 0 12px;
  color: var(--el-text-color-secondary);
}

.partner-import-table {
  margin-top: 12px;
}
</style>
