<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyDialog v-model:open="visible">
    <ChoyDialogContent
      class="import-panel-dialog"
      :show-close-button="false"
      :dismissible="dialogDismissible"
      @open-auto-focus.prevent
    >
      <ChoyDialogHeader class="text-start">
        <ChoyDialogTitle>{{ title }}</ChoyDialogTitle>
      </ChoyDialogHeader>
      <ChoyStepper :model-value="stepValue" :steps="stepperSteps" readonly class="import-panel-stepper my-3" />

      <section v-if="step === 0" class="min-h-[180px]">
        <p class="mb-3 text-muted-foreground">{{ resolvedUploadHint }}</p>
        <p v-if="defaultFieldsHint" class="mb-3 text-muted-foreground" data-test="import-default-fields">{{ defaultFieldsHint }}</p>
        <div v-if="catalogError" role="alert" class="mt-3">{{ catalogError }}</div>
        <label class="block">
          <input type="file" accept=".csv,text/csv" @change="onNativeFile" />
          <div>{{ uploadDropText }}</div>
        </label>
      </section>

      <section v-else-if="step === 1" class="min-h-[180px]">
        <p class="mb-3 text-muted-foreground">{{ mappingHint }}</p>
        <ChoyFormView
          v-if="mappingFormStore"
          ref="mappingFormViewRef"
          class="import-panel-mapping-form flex flex-col gap-2"
          :store="mappingFormStore"
          view-mode="edit"
          v-bind="embeddedFormChrome"
          :cancel-handler="cancelHandler"
        >
          <template #default>
            <ChoySelectionField
              v-for="row in mappingRows"
              :key="row.header"
              :store="mappingFormStore"
              :prop="mappingPropForHeader(row.header)"
              orientation="vertical"
              buffer-strategy="live"
              :clearable="true"
              :data-testid="`import-map-${mappingPropForHeader(row.header)}`"
            />
          </template>
        </ChoyFormView>
        <div v-if="previewReport" role="alert" class="mt-3">{{ previewSummary }}</div>
        <table v-if="previewMessages.length" class="mt-3">
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

      <section v-else class="min-h-[180px]">
        <div v-if="importDone" class="text-success">
          <strong>{{ importSuccessTitle }}</strong>
          <p>{{ importSuccessSubtitle }}</p>
        </div>
        <div v-else-if="importError" role="alert" class="mt-3">{{ importError }}</div>
      </section>

      <ChoyDialogFooter>
        <ChoyButton size="sm" variant="outline" :disabled="busy" @click="onFooterCancel">{{ cancelLabel }}</ChoyButton>
        <ChoyActionButton
          v-if="step === 0"
          size="sm"
          variant="default"
          :disabled="!selectedFile"
          :busy="busy"
          @execute="uploadAndPreview"
        >
          {{ previewActionLabel }}
        </ChoyActionButton>
        <ChoyActionButton v-else-if="step === 1" size="sm" variant="default" :disabled="!canImport" :busy="busy" @execute="commitImport">
          {{ importActionLabel }}
        </ChoyActionButton>
        <ChoyButton v-else-if="importDone" size="sm" variant="default" @click="finish">{{ doneLabel }}</ChoyButton>
      </ChoyDialogFooter>
    </ChoyDialogContent>
  </ChoyDialog>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue';
import {
  ChoyDialog,
  ChoyDialogContent,
  ChoyDialogFooter,
  ChoyDialogHeader,
  ChoyDialogTitle,
} from '@/web/web/components/layout/choyDialog';
import ChoyButton from '@/web/web/components/button/ChoyButton.vue';
import ChoyActionButton from '@/web/web/components/button/ChoyActionButton.vue';
import ChoyStepper, { type ChoyStepperStep } from '@/web/web/components/layout/ChoyStepper.vue';
import ChoyFormView from '@/web/web/components/view/ChoyFormView.vue';
import ChoySelectionField from '@/web/web/components/field/ChoySelectionField.vue';
import {
  useChoyFormDialogShell,
  type ChoyFormDialogViewApi,
} from '@/web/web/composables/useChoyFormDialogShell';
import { createUiModelStore, type UiModelStore } from '@/web/web/stores/uiModelStore';
import type { SelectionItem } from '@/web/web/stores/modelStore';
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

function closeImportPanel(): void {
  visible.value = false;
}

const mappingFormViewRef = ref<ChoyFormDialogViewApi | null>(null);
const mappingFormStore = shallowRef<UiModelStore | null>(null);
const mappingPropByHeader = ref<Record<string, string>>({});

const title = _t('Import');
const defaultUploadHint = _t('Upload a UTF-8 CSV. Map columns to importable fields from the catalog.');
const uploadDropText = _t('Drop CSV here or click to browse');
const defaultFieldsLabel = _t('Suggested columns');
const mappingHint = _t('Map each CSV column to an importable field. Leave blank to use the header as the field path.');
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
// Labels call _t inside computed so a locale switch updates the stepper, same as Setup.
const stepperSteps = computed<ChoyStepperStep[]>(() => [
  { value: 'upload', label: _t('Upload CSV') },
  { value: 'preview', label: _t('Preview') },
  { value: 'import', label: _t('Import') },
]);
const stepValue = computed(() => stepperSteps.value[step.value]?.value ?? 'upload');
const busy = ref(false);

const {
  embeddedFormChrome,
  cancelHandler,
  onFooterCancel,
  dismissible: dialogDismissible,
} = useChoyFormDialogShell({
  onClose: closeImportPanel,
  formViewRef: mappingFormViewRef,
  saving: busy,
  skipFormCancel: () => !mappingFormStore.value || step.value !== 1,
});

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
  mappingFormStore.value = null;
  mappingPropByHeader.value = {};
  previewReport.value = null;
}

function sanitizeHeaderFieldName(header: string, used: Set<string>): string {
  let base = String(header || '')
    .trim()
    .replace(/[^\w]+/g, '_')
    .replace(/^_+|_+$/g, '')
    .replace(/_+/g, '_');
  if (!base) {
    base = 'column';
  }
  if (/^\d/.test(base)) {
    base = `c_${base}`;
  }
  let name = base;
  let seq = 2;
  while (used.has(name)) {
    name = `${base}_${seq}`;
    seq += 1;
  }
  used.add(name);
  return name;
}

function mappingSelectionItems(): SelectionItem[] {
  const items: SelectionItem[] = [{ value: '', label: sameAsHeaderLabel }];
  for (const opt of catalogOptions.value) {
    items.push({ value: opt.path, label: opt.label });
  }
  return items;
}

function mappingPropForHeader(header: string): string {
  return mappingPropByHeader.value[header] ?? '';
}

function syncMappingRowsFromStore(): void {
  const store = mappingFormStore.value;
  if (!store) {
    return;
  }
  const values = store.getValues();
  for (const row of mappingRows.value) {
    const prop = mappingPropForHeader(row.header);
    if (!prop) {
      continue;
    }
    row.fieldPath = String(values[prop] ?? '').trim();
  }
}

function rebuildMappingFormStore(): void {
  if (!mappingRows.value.length) {
    mappingFormStore.value = null;
    mappingPropByHeader.value = {};
    return;
  }
  const used = new Set<string>();
  const propByHeader: Record<string, string> = {};
  const selection = mappingSelectionItems();
  const fields = mappingRows.value.map(row => {
    const prop = sanitizeHeaderFieldName(row.header, used);
    propByHeader[row.header] = prop;
    return {
      name: prop,
      label: row.header,
      type: 'selection' as const,
      selection,
    };
  });
  const initialValues: Record<string, unknown> = {};
  for (const row of mappingRows.value) {
    const prop = propByHeader[row.header];
    if (prop) {
      initialValues[prop] = row.fieldPath ?? '';
    }
  }
  mappingPropByHeader.value = propByHeader;
  mappingFormStore.value = createUiModelStore({
    storeId: `ui-model:import-mapping:${Date.now()}`,
    fields,
    initialValues,
  });
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

watch(
  () => {
    const store = mappingFormStore.value;
    if (!store) {
      return null;
    }
    return { store, values: store.fieldNames.map(name => store.getField(name)) };
  },
  (next, prev) => {
    if (!next) return;
    syncMappingRowsFromStore();
    // Store swap is a rebuild (new headers), not a user mapping edit.
    if (prev?.store === next.store) {
      onMappingChange();
    }
  },
);

watch(catalogOptions, () => {
  const store = mappingFormStore.value;
  if (!store) {
    return;
  }
  const selection = mappingSelectionItems();
  for (const name of store.fieldNames) {
    store.setFieldSelection(name, selection);
  }
});

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
  mappingFormStore.value = null;
  mappingPropByHeader.value = {};
  previewReport.value = null;
  importDone.value = false;
  importError.value = '';
}

async function uploadAndPreview() {
  if (!selectedFile.value) return;
  syncMappingRowsFromStore();
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
      rebuildMappingFormStore();
    }
    syncMappingRowsFromStore();
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
  syncMappingRowsFromStore();
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

