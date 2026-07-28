<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <el-dialog
    v-model="visible"
    :title="dialogTitle"
    width="560px"
    append-to-body
    destroy-on-close
    class="o-field-company-values-dialog"
    @opened="onOpened"
    @closed="emit('closed')"
  >
    <div v-loading="loading" class="o-field-company-values-dialog__body">
      <el-form label-position="left" label-width="168px" @submit.prevent>
        <el-form-item v-for="row in rows" :key="row.companyId" :label="row.label">
          <el-input
            v-model="row.value"
            :maxlength="maxLength"
            :show-word-limit="maxLength != null"
            clearable
          />
        </el-form-item>
      </el-form>
    </div>

    <template #footer>
      <el-button @click="visible = false">{{ _t('Cancel') }}</el-button>
      <el-button type="primary" :loading="saving" native-type="button" @click="handleSave">
        {{ _t('Save company values') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { ElButton, ElDialog, ElForm, ElFormItem, ElInput, ElMessage } from 'element-plus';
import { useAuthStore } from '@/auth/web/stores/auth';
import { createTranslate } from '@/web/web/i18n';
import { createStoreByModel } from '@/web/web/stores/registry';
import type { WebModelStore } from '@/web/web/stores/modelStore';

defineOptions({ name: 'OFieldCompanyValuesDialog' });

type CompanyValueRow = {
  companyId: string;
  label: string;
  value: string;
  initial: string;
  existed: boolean;
};

const props = defineProps<{
  modelValue: boolean;
  store: WebModelStore<any>;
  recordId: string;
  fieldName: string;
  fieldLabel?: string;
  /** Logical field type from FieldsGet / binding meta (e.g. number, boolean, int). */
  fieldType?: string;
  maxLength?: number;
  /** Unsaved form value for the current company; applied on open. */
  draftValue?: string | null;
  /** Optional company override; defaults to auth activeCompanyId. */
  draftCompanyId?: string;
}>();

const emit = defineEmits<{
  'update:modelValue': [boolean];
  saved: [value: unknown];
  closed: [];
}>();

const { _t } = createTranslate('web', { scope: 'web/components/field/OFieldCompanyValuesDialog' });
const companyStore = createStoreByModel('base.Company');

const visible = computed({
  get: () => props.modelValue,
  set: v => emit('update:modelValue', v),
});

const dialogTitle = computed(() => {
  const label = asTrimmedText(props.fieldLabel) || asTrimmedText(props.fieldName);
  return label ? _t('Company values: %s', label) : _t('Company values');
});

const loading = ref(false);
const saving = ref(false);
const rows = ref<CompanyValueRow[]>([]);

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
    rows.value = ordered;
  } catch (err: unknown) {
    ElMessage.error(formatCaughtError(err, _t('Failed to load company values')));
    rows.value = [];
  } finally {
    loading.value = false;
  }
}

function onOpened() {
  void loadRows();
}

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

async function handleSave() {
  saving.value = true;
  try {
    const patch: Record<string, unknown | false> = {};
    for (const row of rows.value) {
      const next = row.value;
      if (row.existed && next === row.initial) continue;
      if (!row.existed && next === '') continue;
      // Clearing any company key removes it (no undeletable base; D12 uses false for delete).
      if (row.existed && next === '') {
        patch[row.companyId] = false;
        continue;
      }
      patch[row.companyId] = coercePatchValue(next);
    }

    if (Object.keys(patch).length) {
      await props.store.UpdateFieldCompanyValues(props.recordId, props.fieldName, patch);
    }

    const refreshed = (await props.store.Browse(props.recordId, [props.fieldName])) as Record<string, unknown>;
    const nextValue = refreshed == null ? undefined : refreshed[props.fieldName];
    emit('saved', nextValue == null ? null : nextValue);
    ElMessage.success(_t('Company values saved'));
    visible.value = false;
  } catch (err: unknown) {
    ElMessage.error(formatCaughtError(err, _t('Failed to save company values')));
  } finally {
    saving.value = false;
  }
}
</script>

<style scoped>
.o-field-company-values-dialog__body {
  min-height: 120px;
}
.o-field-company-values-dialog__body :deep(.el-form-item) {
  margin-bottom: 14px;
  align-items: flex-start;
}
.o-field-company-values-dialog__body :deep(.el-form-item__label) {
  line-height: 32px;
  color: var(--el-text-color-regular);
  justify-content: flex-start;
  text-align: left;
  padding-right: 12px;
}
.o-field-company-values-dialog__body :deep(.el-form-item__content) {
  flex: 1 1 auto;
  min-width: 0;
}
</style>
