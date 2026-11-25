<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyDialog v-model:open="visible">
    <ChoyDialogContent
      class="choy-field-company-values-dialog max-w-xl"
      :show-close-button="false"
      :dismissible="dismissible"
      @open-auto-focus.prevent
    >
      <ChoyDialogHeader class="text-start">
        <ChoyDialogTitle>{{ dialogTitle }}</ChoyDialogTitle>
      </ChoyDialogHeader>
      <div class="choy-field-company-values-dialog__body min-h-[120px]" :aria-busy="loading || undefined">
        <ChoyFormView
          v-if="formStore"
          ref="formViewRef"
          class="choy-field-company-values-dialog__form flex flex-col gap-3.5"
          :store="formStore"
          view-mode="create"
          v-bind="embeddedFormChrome"
          :initial-values="formInitialValues"
          :submit-handler="onSubmit"
          :cancel-handler="cancelHandler"
        >
          <template #default>
            <div
              v-for="companyId in fieldCodes"
              :key="companyId"
              class="choy-field-company-values-dialog__row"
            >
              <ChoyVarcharField
                :store="formStore"
                :prop="companyId"
                orientation="horizontal"
                buffer-strategy="live"
                class="choy-field-company-values-dialog__field"
              />
            </div>
          </template>
        </ChoyFormView>
      </div>
      <ChoyDialogFooter>
        <ChoyButton type="button" variant="outline" :disabled="saving" @click="onFooterCancel">{{
          _t('Cancel')
        }}</ChoyButton>
        <ChoyActionButton type="button" :busy="saving" @execute="onFooterSave">
          {{ _t('Save company values') }}
        </ChoyActionButton>
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
import ChoyFormView from '@/web/web/components/view/ChoyFormView.vue';
import ChoyVarcharField from '@/web/web/components/field/ChoyVarcharField.vue';
import { useChoyFormDialogShell, type ChoyFormDialogViewApi } from '@/web/web/composables/useChoyFormDialogShell';
import { ChoyMessage } from '../../composables/useChoyMessage';
import { useAuthStore } from '@/auth/web/stores/auth';
import { createTranslate } from '@/web/web/i18n';
import { createModelStore } from '@/web/web/stores/registry';
import { createUiModelStore, type UiModelStore } from '@/web/web/stores/uiModelStore';
import type { ModelStore } from '@/web/web/stores/modelStore';

defineOptions({ name: 'ChoyFieldCompanyValuesDialog' });

export type ChoyCompanyValueRow = {
  companyId: string;
  label: string;
  value: string;
  initial: string;
  existed: boolean;
};

type CompanyValueRowMeta = Pick<ChoyCompanyValueRow, 'initial' | 'existed'> & {
  label: string;
};

type CompanyValueRow = ChoyCompanyValueRow;

const props = defineProps<{
  modelValue: boolean;
  store: ModelStore<any>;
  recordId: string;
  fieldName: string;
  fieldLabel?: string;
  /** Logical field type from FieldsGet / binding meta (e.g. number, boolean, int). */
  fieldType?: string;
  maxLength?: number;
  /** Unsaved form value for the current company; applied on open. */
  draftValue?: string;
  /** Optional company override; defaults to auth activeCompanyId. */
  draftCompanyId?: string;
}>();

const emit = defineEmits<{
  'update:modelValue': [boolean];
  saved: [value: unknown];
  closed: [];
}>();

const { _t } = createTranslate('web', { scope: 'web/components/field/FieldCompanyValuesDialog' });
const companyStore = createModelStore('base.Company');

const formViewRef = ref<ChoyFormDialogViewApi | null>(null);
const formStore = shallowRef<UiModelStore | null>(null);
const formInitialValues = shallowRef<Record<string, unknown>>({});
const fieldCodes = ref<string[]>([]);
const rowMeta = ref(new Map<string, CompanyValueRowMeta>());

const visible = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v),
});

function closeDialog(): void {
  visible.value = false;
}

const saving = ref(false);
const { embeddedFormChrome, cancelHandler, onFooterSave, onFooterCancel, dismissible } = useChoyFormDialogShell({
  onClose: closeDialog,
  formViewRef,
  saving,
});

const dialogTitle = computed(() => {
  const label = asTrimmedText(props.fieldLabel) || asTrimmedText(props.fieldName);
  return label ? _t('Company values: %s', label) : _t('Company values');
});

const loading = ref(false);

function asTrimmedText(value: unknown): string {
  if (value == null) return '';
  return String(value).trim();
}

function uniqIds(ids: string[]): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  for (const raw of ids) {
    const id = asTrimmedText(raw);
    if (!id) continue;
    if (seen.has(id)) continue;
    seen.add(id);
    out.push(id);
  }
  return out;
}

function readAuthCompanyMeta(): {
  allowedCompanyIds: string[];
  enabledCompanyIds: string[];
  activeCompanyId: string;
} {
  try {
    const authStore = useAuthStore();
    const identity = (authStore as { identity?: { metadata?: unknown } | null }).identity;
    const metaRaw = identity == null ? undefined : identity.metadata;
    const meta = metaRaw != null && typeof metaRaw === 'object' ? (metaRaw as Record<string, unknown>) : {};
    const allowedCompanyIds = Array.isArray(meta.allowedCompanyIds)
      ? meta.allowedCompanyIds.map((x: unknown) => asTrimmedText(x))
      : [];
    const enabledCompanyIds = Array.isArray(meta.enabledCompanyIds)
      ? meta.enabledCompanyIds.map((x: unknown) => asTrimmedText(x))
      : [];
    const activeCompanyId = asTrimmedText(meta.activeCompanyId);
    return { allowedCompanyIds, enabledCompanyIds, activeCompanyId };
  } catch {
    return { allowedCompanyIds: [], enabledCompanyIds: [], activeCompanyId: '' };
  }
}

/** Prefer company display name; fall back to id. */
function formatCompanyLabel(name: unknown, companyId: string): string {
  const display = asTrimmedText(name);
  return display ? display : companyId;
}

function resolveDraftCompanyId(): string {
  const fromProp = asTrimmedText(props.draftCompanyId);
  if (fromProp) return fromProp;
  return readAuthCompanyMeta().activeCompanyId;
}

/**
 * Overlay the form draft onto the current-company row only.
 * Keep `initial` as the server value so Save still detects the draft as dirty.
 */
function applyDraftValue(byId: Map<string, CompanyValueRow>) {
  if (props.draftValue === undefined) return;
  const companyId = resolveDraftCompanyId();
  if (!companyId) return;
  const row = byId.get(companyId);
  if (!row) return;
  row.value = props.draftValue == null ? '' : String(props.draftValue);
}

/**
 * Row set = allowedCompanyIds, or enabled ∪ map keys when allowlist is empty (D8).
 */
function resolveRowCompanyIds(map: Record<string, unknown>, auth: ReturnType<typeof readAuthCompanyMeta>): string[] {
  const mapKeys = Object.keys(map).map(k => asTrimmedText(k)).filter(Boolean);
  if (auth.allowedCompanyIds.length > 0) {
    return uniqIds(auth.allowedCompanyIds);
  }
  return uniqIds([...auth.enabledCompanyIds, ...mapKeys]);
}

function clearForm(): void {
  formStore.value = null;
  formInitialValues.value = {};
  fieldCodes.value = [];
  rowMeta.value = new Map();
}

function openFormFromRows(ordered: CompanyValueRow[]): void {
  const initialValues: Record<string, string> = {};
  const meta = new Map<string, CompanyValueRowMeta>();
  const codes: string[] = [];
  const fields: Array<{ name: string; label: string; type: 'varchar'; size?: number }> = [];

  for (const row of ordered) {
    codes.push(row.companyId);
    initialValues[row.companyId] = row.value;
    meta.set(row.companyId, { initial: row.initial, existed: row.existed, label: row.label });
    fields.push({
      name: row.companyId,
      label: row.label,
      type: 'varchar',
      ...(props.maxLength != null ? { size: props.maxLength } : {}),
    });
  }

  fieldCodes.value = codes;
  rowMeta.value = meta;
  formInitialValues.value = { ...initialValues };
  formStore.value = createUiModelStore({
    storeId: `ui-model:field-company-values:${props.recordId}:${props.fieldName}`,
    fields,
    initialValues,
  });
}

async function loadRows() {
  loading.value = true;
  try {
    const auth = readAuthCompanyMeta();
    const map = (await props.store.GetFieldCompanyValues(
      props.recordId,
      props.fieldName
    )) as Record<string, unknown>;
    const current = map && typeof map === 'object' ? map : {};
    const companyIds = resolveRowCompanyIds(current, auth);

    let companies: Array<{ Id?: string; DisplayName?: string }> = [];
    if (companyIds.length) {
      const rowsRaw = (await (companyStore as any).Search(['Id', 'in', companyIds] as any, {
        fields: ['Id', 'DisplayName'],
        limit: 1000,
      } as any)) as any[];
      companies = Array.isArray(rowsRaw) ? rowsRaw : [];
    }

    const nameById = new Map<string, string>();
    for (const c of companies) {
      const id = asTrimmedText(c?.Id);
      if (!id) continue;
      nameById.set(id, asTrimmedText(c?.DisplayName));
    }

    const byId = new Map<string, CompanyValueRow>();
    for (const companyId of companyIds) {
      const existed = Object.prototype.hasOwnProperty.call(current, companyId);
      const raw = existed ? current[companyId] : undefined;
      const asText = raw == null ? '' : String(raw);
      byId.set(companyId, {
        companyId,
        label: formatCompanyLabel(nameById.get(companyId), companyId),
        value: asText,
        initial: asText,
        existed,
      });
    }

    applyDraftValue(byId);

    const ordered = Array.from(byId.values());
    ordered.sort((a, b) => a.label.localeCompare(b.label));
    openFormFromRows(ordered);
  } catch (err: unknown) {
    ChoyMessage.error(formatCaughtError(err, _t('Failed to load company values')));
    clearForm();
  } finally {
    loading.value = false;
  }
}

function onOpened() {
  void loadRows();
}

watch(
  () => props.modelValue,
  (open, wasOpen) => {
    if (open && !wasOpen) void onOpened();
    if (!open && wasOpen) {
      clearForm();
      emit('closed');
    }
  },
  { immediate: true },
);

function coercePatchValue(raw: string): unknown {
  const type = asTrimmedText(props.fieldType).toLowerCase();
  const text = String(raw);
  if (type === 'boolean') {
    const lower = text.trim().toLowerCase();
    if (lower === 'true' || lower === '1' || lower === 'yes') return true;
    if (lower === 'false' || lower === '0' || lower === 'no') return false;
    return text.trim() === '';
  }
  if (type === 'int' || type === 'integer' || type === 'bigint') {
    const n = Number.parseInt(text.trim(), 10);
    return Number.isFinite(n) ? n : text;
  }
  if (type === 'number' || type === 'float' || type === 'decimal' || type === 'monetary') {
    const n = Number(text.trim());
    return Number.isFinite(n) ? n : text;
  }
  return text;
}

function formatCaughtError(err: unknown, fallback: string): string {
  if (err == null) return fallback;
  if (typeof err === 'string') {
    const text = err.trim();
    return text ? text : fallback;
  }
  if (typeof err === 'object' && 'message' in err) {
    const text = asTrimmedText((err as { message?: unknown }).message);
    if (text) return text;
  }
  return fallback;
}

async function performSave(formData: Record<string, unknown>): Promise<boolean> {
  saving.value = true;
  try {
    const patch: Record<string, unknown | false> = {};
    for (const companyId of fieldCodes.value) {
      const meta = rowMeta.value.get(companyId);
      if (!meta) continue;
      const next = String(formData[companyId] ?? '');
      if (meta.existed && next === meta.initial) continue;
      if (!meta.existed && next === '') continue;
      if (meta.existed && next === '') {
        patch[companyId] = false;
        continue;
      }
      patch[companyId] = coercePatchValue(next);
    }

    if (Object.keys(patch).length) {
      await props.store.UpdateFieldCompanyValues(props.recordId, props.fieldName, patch);
    }

    const refreshed = (await props.store.Browse(props.recordId, [props.fieldName])) as Record<string, unknown>;
    const nextValue = refreshed == null ? undefined : refreshed[props.fieldName];
    emit('saved', nextValue == null ? null : nextValue);
    ChoyMessage.success(_t('Company values saved'));
    closeDialog();
    return true;
  } catch (err: unknown) {
    ChoyMessage.error(formatCaughtError(err, _t('Failed to save company values')));
    return false;
  } finally {
    saving.value = false;
  }
}

async function onSubmit(ctx: { formData: Record<string, unknown> }) {
  const ok = await performSave(ctx.formData || {});
  if (!ok) {
    throw new Error(_t('Failed to save company values'));
  }
  return { handled: true, skipSuccessMessage: true };
}

defineExpose({
  formatCaughtError,
  coercePatchValue,
  resolveDraftCompanyId,
  readAuthCompanyMeta,
  fieldCodes,
  rowMeta,
  formStore,
});
</script>