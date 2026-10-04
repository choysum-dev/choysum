<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- ================= Render Mode Dispatch ================= -->
  <!-- FORM mode -->
  <div
    v-if="effectiveRenderMode === 'form'"
    v-show="visibleForm"
    class="choy-field-base p-0"
    v-bind="formItemProps"
  >
    <div class="choy-field-base__label mb-1 inline-flex max-w-full items-center gap-1 text-sm font-medium leading-none text-foreground">
      <label class="choy-field-base__label-text min-w-0" :for="inputIdForm || undefined">{{ resolvedLabel }}</label>
      <button
        v-if="effectiveHelp"
        type="button"
        class="choy-field-base__help-btn m-0 inline-flex cursor-help items-center justify-center border-0 bg-transparent p-0 leading-none text-inherit focus-visible:rounded-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
        :aria-label="helpAccessibleLabel"
        :title="effectiveHelp"
      >
        <CircleHelp class="size-3.5" />
      </button>
    </div>
    <template v-if="preserveModeSlotForm">
      <div v-show="effectiveEditForm" class="choy-field-base__edit-wrap flex w-full items-center gap-0.5">
        <div class="choy-field-base__edit-control min-w-0 flex-1">
          <slot
            name="edit"
            :fieldValue="valueForm"
            :record="recordForm"
            :readonly="false"
            :required="requiredForm"
            :visible="visibleForm"
            :inputName="inputName"
            :inputId="inputIdForm"
            :onFieldChange="onchangeHandlers.onChange"
            :triggerOnchange="onchangeHandlers.trigger"
            :onchangeRunning="onchangeHandlers.running?.value"
          />
        </div>
        <ChoyButton
          v-if="showTranslateAction"
          size="sm"
          variant="ghost"
          class="choy-field-base__translate-btn h-6 w-6 shrink-0 p-0 text-muted-foreground hover:text-primary focus:text-primary"
          :aria-label="translateAriaLabel"
          :title="translateAriaLabel"
          @click="translationsOpen = true"
        >
          <Languages class="size-4" />
        </ChoyButton>
        <ChoyButton
          v-if="showCompanyValuesAction"
          size="sm"
          variant="ghost"
          class="choy-field-base__company-values-btn h-6 w-6 shrink-0 p-0 text-muted-foreground hover:text-primary focus:text-primary"
          :aria-label="companyValuesAriaLabel"
          :title="companyValuesAriaLabel"
          @click="companyValuesOpen = true"
        >
          <Building2 class="size-4" />
        </ChoyButton>
      </div>
      <div v-show="!effectiveEditForm">
        <slot
          name="display"
          :fieldValue="valueForm"
          :record="recordForm"
          renderMode="form"
          :readonly="true"
          :required="false"
          :visible="visibleForm"
          :inputName="inputName"
          :inputId="inputIdForm"
          :triggerOnchange="onchangeHandlers.trigger"
          :onchangeRunning="onchangeHandlers.running?.value"
        />
      </div>
    </template>
    <template v-else>
      <template v-if="effectiveEditForm">
        <div class="choy-field-base__edit-wrap flex w-full items-center gap-0.5">
          <div class="choy-field-base__edit-control min-w-0 flex-1">
            <slot
              name="edit"
              :fieldValue="valueForm"
              :record="recordForm"
              :readonly="false"
              :required="requiredForm"
              :visible="visibleForm"
              :inputName="inputName"
              :inputId="inputIdForm"
              :onFieldChange="onchangeHandlers.onChange"
              :triggerOnchange="onchangeHandlers.trigger"
              :onchangeRunning="onchangeHandlers.running?.value"
            />
          </div>
          <ChoyButton
            v-if="showTranslateAction"
            size="sm"
            variant="ghost"
            class="choy-field-base__translate-btn h-6 w-6 shrink-0 p-0 text-muted-foreground hover:text-primary focus:text-primary"
            :aria-label="translateAriaLabel"
            :title="translateAriaLabel"
            @click="translationsOpen = true"
          >
            <Languages class="size-4" />
          </ChoyButton>
          <ChoyButton
            v-if="showCompanyValuesAction"
            size="sm"
            variant="ghost"
            class="choy-field-base__company-values-btn h-6 w-6 shrink-0 p-0 text-muted-foreground hover:text-primary focus:text-primary"
            :aria-label="companyValuesAriaLabel"
            :title="companyValuesAriaLabel"
            @click="companyValuesOpen = true"
          >
            <Building2 class="size-4" />
          </ChoyButton>
        </div>
      </template>
      <template v-else>
        <slot
          name="display"
          :fieldValue="valueForm"
          :record="recordForm"
          renderMode="form"
          :readonly="true"
          :required="false"
          :visible="visibleForm"
          :inputName="inputName"
          :inputId="inputIdForm"
          :triggerOnchange="onchangeHandlers.trigger"
          :onchangeRunning="onchangeHandlers.running?.value"
        />
      </template>
    </template>
    <p v-if="displayError" class="choy-field-base__error mt-1 whitespace-normal text-sm font-normal text-destructive" role="alert">{{ displayError }}</p>
    <FieldTranslationsDialog
      v-if="showTranslateAction"
      v-model="translationsOpen"
      :store="binding.store as any"
      :record-id="panelRecordId"
      :field-name="leafFieldName"
      :field-label="resolvedLabel"
      :max-length="panelMaxLength"
      :draft-value="panelDraftValue"
      @saved="onTranslationsSaved"
    />
    <FieldCompanyValuesDialog
      v-if="showCompanyValuesAction"
      v-model="companyValuesOpen"
      :store="binding.store as any"
      :record-id="panelRecordId"
      :field-name="leafFieldName"
      :field-label="resolvedLabel"
      :field-type="companyValuesFieldType"
      :max-length="panelMaxLength"
      :draft-value="panelDraftValue"
      @saved="onCompanyValuesSaved"
    />
  </div>

  <!-- TABLE mode -->
  <ChoyTableColumn
    v-else-if="effectiveRenderMode === 'table' && columnVisible"
    :prop="String(binding.prop)"
    :label="resolvedLabel"
    :vColumnProps="vColumnProps"
    v-slot="{ row, $index }"
  >
    <div
      class="choy-field-base__cell block w-full"
      v-show="cellVisibleForRow(row)"
      :data-field="inputName"
      :data-row-key="guessRowKey(row)"
      :id="`fld-${inputName}-${guessRowKey(row)}`"
    >
      <div
        class="choy-field-base__cell-item"
        :class="{ 'choy-field-base__cell-item--error': !!serverErrorForRow(row, $index) }"
        :data-error="serverErrorForRow(row, $index) || undefined"
      >
        <template v-if="effectiveEditForRow(row)">
          <slot
            name="edit"
            :fieldValue="valueForRow(row)"
            :record="recordForRow(row)"
            :readonly="false"
            :required="requiredForRow(row)"
            :visible="cellVisibleForRow(row)"
            :inputName="inputName"
            :inputId="inputIdForRow(row)"
            :onFieldChange="onchangeHandlers.onChange"
            :triggerOnchange="onchangeHandlers.trigger"
            :onchangeRunning="onchangeHandlers.running?.value"
          />
        </template>
        <template v-else>
          <slot
            name="display"
            :fieldValue="valueForRow(row)"
            :record="recordForRow(row)"
            renderMode="table"
            :readonly="true"
            :required="false"
            :visible="cellVisibleForRow(row)"
            :inputName="inputName"
            :inputId="inputIdForRow(row)"
            :triggerOnchange="onchangeHandlers.trigger"
            :onchangeRunning="onchangeHandlers.running?.value"
          />
        </template>
      </div>
    </div>
  </ChoyTableColumn>

  <!-- INLINE mode -->
  <div v-else-if="effectiveRenderMode === 'inline'" class="choy-field-base__inline inline-flex items-center gap-1" v-show="visibleInline">
    <div
      v-if="showInlineError && displayError"
      class="choy-field-base__inline-wrap choy-field-base__inline-wrap--has-error inline-flex items-center gap-1.5"
      :title="displayError"
    >
      <template v-if="effectiveEditInline">
        <slot
          name="edit"
          :fieldValue="valueForm"
          :record="recordForm"
          :readonly="false"
          :required="requiredInline"
          :visible="visibleInline"
          :inputName="inputName"
          :inputId="inputIdForm"
          :onFieldChange="onchangeHandlers.onChange"
          :triggerOnchange="onchangeHandlers.trigger"
          :onchangeRunning="onchangeHandlers.running?.value"
        />
      </template>
      <template v-else>
        <slot
          name="display"
          :fieldValue="valueForm"
          :record="recordForm"
          renderMode="inline"
          :readonly="true"
          :required="false"
          :visible="visibleInline"
          :inputName="inputName"
          :inputId="inputIdForm"
          :triggerOnchange="onchangeHandlers.trigger"
          :onchangeRunning="onchangeHandlers.running?.value"
        />
      </template>
      <CircleAlert class="choy-inline-err-icon size-4" />
    </div>

    <div
      v-else
      class="choy-field-base__inline-wrap inline-flex items-center gap-1.5"
      :class="{ 'choy-field-base__inline-wrap--has-help': !!effectiveHelp }"
    >
      <template v-if="effectiveEditInline">
        <slot
          name="edit"
          :fieldValue="valueForm"
          :record="recordForm"
          :readonly="false"
          :required="requiredInline"
          :visible="visibleInline"
          :inputName="inputName"
          :inputId="inputIdForm"
          :onFieldChange="onchangeHandlers.onChange"
          :triggerOnchange="onchangeHandlers.trigger"
          :onchangeRunning="onchangeHandlers.running?.value"
        />
      </template>
      <template v-else>
        <slot
          name="display"
          :fieldValue="valueForm"
          :record="recordForm"
          renderMode="inline"
          :readonly="true"
          :required="false"
          :visible="visibleInline"
          :inputName="inputName"
          :inputId="inputIdForm"
          :triggerOnchange="onchangeHandlers.trigger"
          :onchangeRunning="onchangeHandlers.running?.value"
        />
      </template>
      <button
        v-if="effectiveHelp"
        type="button"
        class="choy-field-base__help-btn m-0 inline-flex cursor-help items-center justify-center border-0 bg-transparent p-0 leading-none text-inherit focus-visible:rounded-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
        :aria-label="helpAccessibleLabel"
        :title="effectiveHelp"
      >
        <CircleHelp class="size-3.5" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts" generic="T extends BaseModel, V = unknown, View = V">
import type { RuleItem } from 'async-validator';
import type { BaseModel } from '@/core/rpc';
import type { TermReference } from '@/core/service/i18n';
import ChoyTableColumn from '@/web/web/components/table/ChoyTableColumn.vue';
import type { UseField, FieldEnv } from '@/web/web/composables/useField';
import type { ComputedRef, WritableComputedRef, Ref } from 'vue';
import { computed, inject, onMounted, onBeforeUnmount, ref, watch, getCurrentInstance } from 'vue';
import { useProvidedOnchange, getOnchangeController } from '@/web/web/composables/useOnchange';
import {
  FIELD_CLIENT_VALIDATORS_KEY,
  firstRuleError,
} from '@/web/web/composables/fieldClientValidation';
import { CircleAlert, CircleHelp, Languages, Building2 } from 'lucide-vue-next';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { createTranslate, getGlobalComposer } from '@/web/web/i18n/translate';
import { resolveFieldLabel } from '@/web/web/composables/resolveFieldLabel';
import { resolveFieldHelp } from '@/web/web/composables/resolveFieldHelp';
import { FIELD_PRESENTATION_FIELDS_GET_ATTRS } from '@/web/web/stores/fieldsGet';
import FieldTranslationsDialog from './ChoyFieldTranslationsDialog.vue';
import FieldCompanyValuesDialog from './ChoyFieldCompanyValuesDialog.vue';

/** Native form-item chrome attrs (Element Plus FormItemProps removed). */
export type FormItemProps = Record<string, unknown>;

export type FieldStatePredicate<T, V> = (args: { record: T; value: V | null; env: FieldEnv }) => boolean;
export type FieldStateExpr<T, V> = boolean | FieldStatePredicate<T, V>;

defineOptions({ name: 'FieldBase' });

const { _t } = createTranslate('web', { scope: 'web/components/field/FieldBase' });

const props = withDefaults(
  defineProps<{
    binding: UseField<T, V>;
    label?: string;
    rules?: RuleItem[];
    formItemProps?: Partial<FormItemProps>;
    vColumnProps?: Record<string, unknown>;
    toView?: (raw: V) => View;
    fromView?: (v: View) => V;
    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    preserveModeSlot?: boolean;
    showInlineError?: boolean;
  }>(),
  {
    rules: () => [],
    formItemProps: () => ({}),
    vColumnProps: () => ({}),
    visible: true,
    cellVisible: true,
    required: false,
    readonly: false,
    renderMode: 'auto',
    preserveModeSlot: false,
    showInlineError: false,
  }
);

const binding = props.binding;

const leafFieldName = computed(() => {
  const prop = String(binding.prop || '');
  return prop.split('.').filter(Boolean).pop() || prop;
});

const modelStore = computed(() => {
  return binding.store as
    | {
        getFieldMeta?: (name: string) => typeof binding.meta;
        getFieldsGetTranslatedString?: (name: string) => string | undefined;
        getFieldsGetTranslatedHelp?: (name: string) => string | undefined;
        ensureFieldsGet?: (fields?: string[], attributes?: string[]) => Promise<unknown>;
      }
    | undefined;
});

/** Effective meta: FieldsGet overlay over static binding.meta (D6 / P5). */
const effectiveFieldMeta = computed(() => {
  const leaf = leafFieldName.value;
  return modelStore.value?.getFieldMeta?.(leaf) ?? binding.meta;
});

const resolvedLabel = computed(() => {
  const prop = String(binding.prop || '');
  const leaf = leafFieldName.value;
  return resolveFieldLabel({
    label: props.label,
    prop,
    meta: effectiveFieldMeta.value,
    fieldsGetTranslatedString: modelStore.value?.getFieldsGetTranslatedString?.(leaf),
    composer: getGlobalComposer(),
  });
});

const effectiveHelp = computed(() => {
  const leaf = leafFieldName.value;
  return resolveFieldHelp({
    meta: effectiveFieldMeta.value as { help?: string; helpText?: TermReference } | undefined,
    fieldsGetTranslatedHelp: modelStore.value?.getFieldsGetTranslatedHelp?.(leaf),
    composer: getGlobalComposer(),
  });
});

const helpAccessibleLabel = computed(() => {
  const help = effectiveHelp.value;
  const label = String(resolvedLabel.value || leafFieldName.value || '').trim();
  // Tip only mounts when help is non-empty; keep label context when available.
  return label ? `${label}: ${help}` : help;
});

onMounted(() => {
  // Edit mode: ensure ACL/presentation overlay so deny-write → isReadonly is visible (T5.3).
  if (!binding.env.isEditMode) return;
  const store = modelStore.value;
  const leaf = leafFieldName.value;
  if (!store?.ensureFieldsGet || !leaf) return;
  void store.ensureFieldsGet([leaf], [...FIELD_PRESENTATION_FIELDS_GET_ATTRS]);
});

const translationsOpen = ref(false);
const companyValuesOpen = ref(false);

/** Record id shared by translate / company-values panels. */
const panelRecordId = computed(() => {
  try {
    const record = binding.recordRef?.()?.value as { Id?: unknown } | undefined;
    const id = String(record?.Id ?? '').trim();
    return id || '';
  } catch {
    return '';
  }
});

const showTranslateAction = computed(() => {
  if (effectiveRenderMode.value !== 'form') return false;
  if (!binding.env.isEditMode) return false;
  if (!panelRecordId.value) return false;
  if (!binding.store) return false;
  const meta = effectiveFieldMeta.value as { translate?: boolean } | undefined;
  return meta?.translate === true;
});

const showCompanyValuesAction = computed(() => {
  if (effectiveRenderMode.value !== 'form') return false;
  if (!binding.env.isEditMode) return false;
  if (!panelRecordId.value) return false;
  if (!binding.store) return false;
  const meta = effectiveFieldMeta.value as { companyDependent?: boolean } | undefined;
  return Boolean(meta && meta.companyDependent === true);
});

const companyValuesFieldType = computed(() => {
  const type = (effectiveFieldMeta.value as { type?: string } | undefined)?.type;
  if (type == null) return undefined;
  const trimmed = String(type).trim();
  return trimmed ? trimmed : undefined;
});

const panelMaxLength = computed(() => {
  const meta = effectiveFieldMeta.value as { size?: number } | undefined;
  const size = meta?.size;
  return typeof size === 'number' && Number.isInteger(size) && size > 0 ? size : undefined;
});

const translateAriaLabel = computed(() => {
  const label = String(resolvedLabel.value || leafFieldName.value || '').trim();
  return label ? _t('Translate: %s', label) : _t('Translate field');
});

const companyValuesAriaLabel = computed(() => {
  const fromLabel = String(resolvedLabel.value || '').trim();
  const fromLeaf = String(leafFieldName.value || '').trim();
  const label = fromLabel || fromLeaf;
  return label ? _t('Company values: %s', label) : _t('Company values');
});

/** Current form draft (current UI lang / active company unwrap); seeded into dialogs on open. */
const panelDraftValue = computed(() => {
  const fieldRef = valueForm() as WritableComputedRef<View>;
  const v = fieldRef.value;
  return v == null ? '' : String(v);
});

function onTranslationsSaved(nextValue: string | null) {
  try {
    const fieldRef = valueForm() as WritableComputedRef<View>;
    if (fieldRef) {
      fieldRef.value = nextValue as View;
    }
  } catch {
    // Ignore draft write failures; Browse already refreshed server state.
  }
}

function onCompanyValuesSaved(nextValue: unknown) {
  try {
    const fieldRef = valueForm() as WritableComputedRef<View> | null | undefined;
    if (fieldRef == null) return;
    fieldRef.value = nextValue as View;
  } catch (_err) {
    // Ignore draft write failures; Browse already refreshed server state.
  }
}

/* ===================== Render Mode Dispatch ===================== */
// Inject a render mode override (for example, Kanban cards provide inline)
const injectedRenderOverride = inject<'inline' | 'form' | 'table' | 'auto' | null>('choy-field-render-override', null);
const effectiveRenderMode = computed(() => {
  const rm = props.renderMode || 'auto';
  if (rm !== 'auto') return rm;
  if (injectedRenderOverride) return injectedRenderOverride;
  return binding.env.isForm ? 'form' : 'table';
});
const showInlineError = computed(() => props.showInlineError === true);
const preserveModeSlotForm = computed(() => props.preserveModeSlot === true);

/* Derive INLINE-mode state from form logic without rendering ElFormItem */
const visibleInline = computed(() => visibleForm.value); // Reuse visibleForm as the visibility gate
const readonlyInline = computed(() => readonlyForm.value);
const requiredInline = computed(() => requiredForm.value);
const effectiveEditInline = computed(() => binding.env.isEditMode && visibleInline.value && !readonlyInline.value);

// Unwrap row records to the actual business record (row-level detection, not snapshot-level QueryKind):
// - Legacy grouped-tree detail row: { type:'record', record:{...} }
// - New controller RecordRow: { kind:'record', payload:{...} } // DataSetSnapshot.kind is now 'search' | 'group'
function unwrapRecord(row: any): any {
  if (!row) return row;
  if (row.type === 'record' && row.record) return row.record;
  if (row.kind === 'record' && row.payload) return row.payload;
  if (row.payload && typeof row.payload === 'object') return row.payload; // Fallback
  return row;
}

// Inject the field error map from the form view
const fieldErrors = inject<Ref<Map<string, string>> | null>('field-errors', null);

// Standalone List S2: only the active editing row may enter table edit mode.
const listEditingRowId = inject<Ref<string | null>>('list-editing-row-id', ref(null));

// Compute the server error for the current field in the form container
const serverError = computed(() => {
  if (!fieldErrors?.value) return undefined;
  return fieldErrors.value.get(String(binding.prop)) || undefined;
});

/* Server error for a list cell, matched only by row Id or index */
function serverErrorForRow(row: any, rowIndex?: number): string | undefined {
  const map = fieldErrors?.value;
  if (!map) return undefined;

  const base = String(inputName.value || '');
  if (!base) return undefined;

  const segs = base.split('.').filter(Boolean);
  if (segs.length < 2) return undefined;

  const lastCollectionIdx = segs.length - 2;
  const lastCollection = segs[lastCollectionIdx];

  // Read Id from the real record
  const rec = unwrapRecord(row);
  const rowId = rec?.Id ?? null;
  const tryKeys: string[] = [];

  // Use an Id selector for the last segment
  if (rowId != null) {
    const withLastId = [...segs.slice(0, lastCollectionIdx), `${lastCollection}(id=${String(rowId)})`, ...segs.slice(lastCollectionIdx + 1)].join('.');
    tryKeys.push(withLastId);
  }

  // Use an index selector for the last segment
  if (typeof rowIndex === 'number' && rowIndex >= 0) {
    const withLastIdx = [...segs.slice(0, lastCollectionIdx), `${lastCollection}[${rowIndex}]`, ...segs.slice(lastCollectionIdx + 1)].join('.');
    tryKeys.push(withLastIdx);
  }

  for (const k of tryKeys) {
    const msg = map.get(k);
    if (msg) return msg;
  }
  return undefined;
}

/* ===================== Unified onchange handling (automatic mode) ===================== */
function createOnchangeHandlers() {
  const usedStore: any = binding.store || binding.relationStore;
  if (!usedStore) {
    return {
      onChange: async () => {},
      trigger: async (_?: string | string[]) => {},
      running: undefined as any,
    };
  }
  const injected = useProvidedOnchange();
  const ctrl = injected || getOnchangeController(usedStore);

  async function onChange() {
    await ctrl.flush();
  }

  async function trigger(fieldPath?: string | string[]) {
    if (!fieldPath) return ctrl.flush();
    await ctrl.force(fieldPath);
  }

  return { onChange, trigger, running: ctrl.running };
}
const onchangeHandlers = createOnchangeHandlers();
/* ================== /Unified onchange handling ======================= */

/* Input control identifiers */
const inputName = computed(() => String(binding.prop));
let __autoRowKey = 0;
function guessRowKey(row: any): string {
  const rec = unwrapRecord(row);
  return String(row?.__rowKey ?? row?.key ?? rec?.Id ?? ++__autoRowKey);
}
const inputIdForm = computed(() => `fld-${inputName.value}`);
const inputIdForRow = (row: T) => `fld-${inputName.value}-${guessRowKey(row)}`;

/* View mapping for slot rendering */
const viewBinding =
  props.toView || props.fromView
    ? binding.asView<View, V>({
        toView: props.toView ?? ((r: V) => r as unknown as View),
        fromView: props.fromView ?? ((v: View) => v as unknown as V),
      })
    : null;

/* Raw values */
const rawValueForm = (() => binding.fieldRef()) as () => WritableComputedRef<V>;
const rawValueForRow = ((row: T) => () => binding.fieldRefOf(unwrapRecord(row) as any)) as (row: T) => () => WritableComputedRef<V>;

/* View values */
const valueForm = (() =>
  (viewBinding ? viewBinding.fieldValue() : (binding.fieldRef() as unknown)) as WritableComputedRef<View>) as () => WritableComputedRef<View>;
const valueForRow = ((row: T) => () =>
  (viewBinding
    ? viewBinding.fieldValueOfRow(unwrapRecord(row) as any)
    : (binding.fieldRefOf(unwrapRecord(row) as any) as unknown)) as WritableComputedRef<View>) as (row: T) => () => WritableComputedRef<View>;

/* Record refs */
const recordFormRef = binding.recordRef();
const recordForm = (() => recordFormRef) as () => ComputedRef<T>;
const recordForRow = ((row: T) => {
  const c = computed(() => unwrapRecord(row) as T) as ComputedRef<T>;
  return () => c;
}) as (row: T) => () => ComputedRef<T>;

/* Metadata flags (effective meta includes FieldsGet ACL overlay) */
const metaRequired = computed(() => effectiveFieldMeta.value?.notNull === true);
const metaReadonly = computed(() => effectiveFieldMeta.value?.isReadonly === true);

/* Evaluation helper */
function evalFlag(flag: FieldStateExpr<T, V> | undefined, rec: T, v: V | undefined, def = false) {
  if (typeof flag === 'function') {
    return !!flag({
      record: rec,
      value: (v ?? null) as V | null,
      env: binding.env,
    });
  }
  if (typeof flag === 'boolean') return flag;
  return def;
}

/* Form-level visible/readonly/required state */
const visibleForm = computed(() => evalFlag(props.visible, recordForm().value, rawValueForm().value, true));
const readonlyForm = computed(() => {
  if (!binding.env.isEditMode) return true;
  if (metaReadonly.value) return true;
  return evalFlag(props.readonly, recordForm().value, rawValueForm().value, false);
});
const requiredForm = computed(() => {
  if (!binding.env.isEditMode) return false;
  if (readonlyForm.value) return false;
  if (metaRequired.value) return true;
  return evalFlag(props.required, recordForm().value, rawValueForm().value, false);
});
const effectiveEditForm = computed(() => binding.env.isEditMode && visibleForm.value && !readonlyForm.value);

/* Column-level visibility control */
const columnVisible = computed(() => (typeof props.visible === 'boolean' ? !!props.visible : true));

/* Row-level state, unwrapping grouped detail rows */
const cellVisibleForRow = (row: T) => {
  const rec = unwrapRecord(row) as T;
  const raw = rawValueForRow(row)().value;
  if (props.cellVisible !== undefined) return evalFlag(props.cellVisible, rec, raw, true);
  if (typeof props.visible === 'function') return evalFlag(props.visible, rec, raw, true);
  return true;
};
const readonlyForRow = (row: T) => {
  if (!binding.env.isEditMode) return true;
  if (metaReadonly.value) return true;
  const rec = unwrapRecord(row) as T;
  return evalFlag(props.readonly, rec, rawValueForRow(row)().value, false);
};
const requiredForRow = (row: T) => {
  if (!binding.env.isEditMode) return false;
  if (readonlyForRow(row)) return false;
  if (metaRequired.value) return true;
  const rec = unwrapRecord(row) as T;
  return evalFlag(props.required, rec, rawValueForRow(row)().value, false);
};
const effectiveEditForRow = (row: T) => {
  if (!binding.env.isEditMode || !cellVisibleForRow(row) || readonlyForRow(row)) return false;
  // Top-level list S2 only: nested relation tables (field-prefix / O2M lines) keep their own row ids.
  if (listEditingRowId.value != null && !binding.env.fieldPrefix) {
    const rec = unwrapRecord(row);
    const id = rec?.Id;
    if (id == null || String(id) !== String(listEditingRowId.value)) return false;
  }
  return true;
};

/* Clear server errors when the field value changes, after all variable definitions */
watch(
  () => rawValueForm().value,
  () => {
    if (serverError.value && fieldErrors?.value) {
      fieldErrors.value.delete(String(binding.prop));
    }
  }
);

/* Server errors are shown via dedicated alert nodes; client rules also gate submit. */
const fieldClientValidators = inject(FIELD_CLIENT_VALIDATORS_KEY, null);
const clientRuleError = ref('');
const displayError = computed(() => serverError.value || clientRuleError.value || undefined);

async function evaluateClientRules(): Promise<string> {
  if (!binding.env.isEditMode || readonlyForm.value || !visibleForm.value) {
    clientRuleError.value = '';
    return '';
  }
  const message = await firstRuleError(props.rules, rawValueForm().value);
  clientRuleError.value = message;
  return message;
}

/** Per-instance key so list/table mounts of the same prop do not clobber each other. */
const clientValidatorKey = `field-client-validator:${getCurrentInstance()?.uid ?? Math.random().toString(36).slice(2)}`;

onMounted(() => {
  const map = fieldClientValidators;
  if (map) {
    map.set(clientValidatorKey, evaluateClientRules);
  }
});

onBeforeUnmount(() => {
  const map = fieldClientValidators;
  if (map && map.get(clientValidatorKey) === evaluateClientRules) {
    map.delete(clientValidatorKey);
  }
});

watch(
  () => [rawValueForm().value, props.rules, binding.env.isEditMode, readonlyForm.value, visibleForm.value] as const,
  () => {
    void evaluateClientRules();
  },
  { immediate: true, deep: true }
);

/* Slot type declarations */
defineSlots<{
  edit(args: {
    fieldValue: () => WritableComputedRef<View>;
    record: () => ComputedRef<T>;
    readonly: boolean;
    required: boolean;
    visible: boolean;
    inputName: string | undefined;
    inputId: string | undefined;
    onFieldChange: () => Promise<void>;
    triggerOnchange: (fp?: string | string[]) => Promise<void>;
    onchangeRunning: boolean | undefined;
  }): any;
  display(args: {
    fieldValue: () => WritableComputedRef<View>;
    record: () => ComputedRef<T>;
    renderMode?: 'form' | 'table' | 'inline';
    readonly: boolean;
    required: boolean;
    visible: boolean;
    inputName: string | undefined;
    inputId: string | undefined;
    triggerOnchange: (fp?: string | string[]) => Promise<void>;
    onchangeRunning: boolean | undefined;
  }): any;
}>();
</script>

