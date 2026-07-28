<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- ================= Render Mode Dispatch ================= -->
  <!-- FORM mode -->
  <el-form-item
    v-if="effectiveRenderMode === 'form'"
    v-show="visibleForm"
    class="o-field-base"
    v-bind="formItemProps"
    :label="resolvedLabel"
    :prop="String(binding.prop)"
    :rules="effectiveRules"
    :required="requiredForm"
    :error="serverError"
  >
    <template v-if="preserveModeSlotForm">
      <div v-show="effectiveEditForm" class="o-field-base__edit-wrap">
        <div class="o-field-base__edit-control">
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
        <el-tooltip v-if="showTranslateAction" :content="translateAriaLabel" placement="top" :show-after="200">
          <el-button
            class="o-field-base__translate-btn"
            text
            :aria-label="translateAriaLabel"
            @click="translationsOpen = true"
          >
            <el-icon :size="16">
              <component :is="TranslateOutlined" />
            </el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip v-if="showCompanyValuesAction" :content="companyValuesAriaLabel" placement="top" :show-after="200">
          <el-button
            class="o-field-base__company-values-btn"
            text
            :aria-label="companyValuesAriaLabel"
            @click="companyValuesOpen = true"
          >
            <el-icon :size="16">
              <component :is="BusinessOutlined" />
            </el-icon>
          </el-button>
        </el-tooltip>
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
        <div class="o-field-base__edit-wrap">
          <div class="o-field-base__edit-control">
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
          <el-tooltip v-if="showTranslateAction" :content="translateAriaLabel" placement="top" :show-after="200">
            <el-button
              class="o-field-base__translate-btn"
              text
              :aria-label="translateAriaLabel"
              @click="translationsOpen = true"
            >
              <el-icon :size="16">
                <component :is="TranslateOutlined" />
              </el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip v-if="showCompanyValuesAction" :content="companyValuesAriaLabel" placement="top" :show-after="200">
            <el-button
              class="o-field-base__company-values-btn"
              text
              :aria-label="companyValuesAriaLabel"
              @click="companyValuesOpen = true"
            >
              <el-icon :size="16">
                <component :is="BusinessOutlined" />
              </el-icon>
            </el-button>
          </el-tooltip>
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
    <OFieldTranslationsDialog
      v-if="showTranslateAction && translationRecordId"
      v-model="translationsOpen"
      :store="binding.store as any"
      :record-id="translationRecordId"
      :field-name="leafFieldName"
      :field-label="resolvedLabel"
      :max-length="translationMaxLength"
      :draft-value="translationDraftValue"
      @saved="onTranslationsSaved"
    />
    <OFieldCompanyValuesDialog
      v-if="showCompanyValuesAction && translationRecordId"
      v-model="companyValuesOpen"
      :store="binding.store as any"
      :record-id="translationRecordId"
      :field-name="leafFieldName"
      :field-label="resolvedLabel"
      :max-length="translationMaxLength"
      :draft-value="translationDraftValue"
      @saved="onCompanyValuesSaved"
    />
  </el-form-item>

  <!-- TABLE mode -->
  <OVColumn
    v-else-if="effectiveRenderMode === 'table' && columnVisible"
    :prop="String(binding.prop)"
    :label="resolvedLabel"
    :vColumnProps="vColumnProps"
    v-slot="{ row, $index }"
  >
    <div
      class="o-field-base__cell"
      v-show="cellVisibleForRow(row)"
      :data-field="inputName"
      :data-row-key="guessRowKey(row)"
      :id="`fld-${inputName}-${guessRowKey(row)}`"
    >
      <el-form-item
        class="o-field-base__cell-item"
        :prop="`__cell__:${inputName}:${guessRowKey(row)}`"
        :rules="[
          {
            validator: (_r: any, _v: any, cb: any) => {
              const msg = serverErrorForRow(row, $index);
              msg ? cb(new Error(msg)) : cb();
            },
            trigger: 'blur',
          } as any,
        ]"
        :error="serverErrorForRow(row, $index)"
        :show-message="true"
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
      </el-form-item>
    </div>
  </OVColumn>

  <!-- INLINE mode -->
  <div v-else-if="effectiveRenderMode === 'inline'" class="o-field-base__inline" v-show="visibleInline">
    <!-- When an error is present, wrap with Tooltip and show an error icon -->
    <el-tooltip v-if="showInlineError && serverError" :content="serverError" effect="dark" placement="top" :show-after="120">
      <div class="o-field-base__inline-wrap o-field-base__inline-wrap--has-error">
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
        <el-icon class="o-inline-err-icon" color="var(--el-color-error)">
          <WarningFilled />
        </el-icon>
      </div>
    </el-tooltip>

    <!-- Without an error, render as-is -->
    <template v-else>
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
    </template>
  </div>
</template>

<script setup lang="ts" generic="T extends BaseModel, V = unknown, View = V">
import type { RuleItem } from 'async-validator';
import type { BaseModel } from '@/core/rpc';
import { ElButton, ElFormItem, ElIcon, ElTooltip, type FormItemProps } from 'element-plus';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import type { UseField, FieldEnv } from '@/web/web/composables/useField';
import type { ComputedRef, WritableComputedRef, Ref } from 'vue';
import { computed, inject, onMounted, ref, watch } from 'vue';
import { useProvidedOnchange, getOnchangeController } from '@/web/web/composables/useOnchange';
import { WarningFilled } from '@element-plus/icons-vue';
import { TranslateOutlined, BusinessOutlined } from '@vicons/material';
import { createTranslate, getGlobalComposer } from '@/web/web/i18n/translate';
import { resolveFieldLabel } from '@/web/web/composables/resolveFieldLabel';
import { FIELD_PRESENTATION_FIELDS_GET_ATTRS } from '@/web/web/stores/fieldsGet';
import OFieldTranslationsDialog from './OFieldTranslationsDialog.vue';
import OFieldCompanyValuesDialog from './OFieldCompanyValuesDialog.vue';

export type FieldStatePredicate<T, V> = (args: { record: T; value: V | null; env: FieldEnv }) => boolean;
export type FieldStateExpr<T, V> = boolean | FieldStatePredicate<T, V>;

defineOptions({ name: 'OFieldBase' });

const { _t } = createTranslate('web', { scope: 'web/components/field/OFieldBase' });

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

const translationRecordId = computed(() => {
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
  if (!translationRecordId.value) return false;
  const meta = effectiveFieldMeta.value as { translate?: boolean } | undefined;
  return meta?.translate === true;
});

const showCompanyValuesAction = computed(() => {
  if (effectiveRenderMode.value !== 'form') return false;
  if (!binding.env.isEditMode) return false;
  if (!translationRecordId.value) return false;
  const meta = effectiveFieldMeta.value as { companyDependent?: boolean } | undefined;
  return meta?.companyDependent === true;
});

const translationMaxLength = computed(() => {
  const meta = effectiveFieldMeta.value as { size?: number } | undefined;
  const size = meta?.size;
  return typeof size === 'number' && Number.isInteger(size) && size > 0 ? size : undefined;
});

const translateAriaLabel = computed(() => {
  const label = String(resolvedLabel.value || leafFieldName.value || '').trim();
  return label ? _t('Translate: %s', label) : _t('Translate field');
});

const companyValuesAriaLabel = computed(() => {
  const label = String(resolvedLabel.value || leafFieldName.value || '').trim();
  return label ? _t('Company values: %s', label) : _t('Company values');
});

/** Current form draft (current UI lang / active company unwrap); seeded into dialogs on open. */
const translationDraftValue = computed(() => {
  try {
    const fieldRef = valueForm() as WritableComputedRef<View> | undefined;
    const v = fieldRef?.value;
    return v == null ? '' : String(v);
  } catch {
    return '';
  }
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
    const fieldRef = valueForm() as WritableComputedRef<View>;
    if (fieldRef) {
      fieldRef.value = nextValue as View;
    }
  } catch {
    // Ignore draft write failures; Browse already refreshed server state.
  }
}

/* ===================== Render Mode Dispatch ===================== */
// Inject a render mode override (for example, Kanban cards provide inline)
const injectedRenderOverride = inject<'inline' | 'form' | 'table' | 'auto' | null>('o-field-render-override', null);
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

// Inject the field error map from OFormView
const fieldErrors = inject<Ref<Map<string, string>> | null>('field-errors', null);

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
  const rowId = rec?.Id ?? rec?.id ?? null;
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

/* ===================== Merge server errors into rules (dual strategy) ===================== */
const effectiveRules = computed<RuleItem[]>(() => {
  const baseRules = props.rules || [];
  const rulesWithServerError: RuleItem[] = [];

  if (serverError.value && binding.env.isEditMode) {
    rulesWithServerError.push({
      validator: (_rule: unknown, _value: unknown, callback: (error?: Error) => void) => {
        const currentError = fieldErrors?.value?.get(String(binding.prop));
        if (currentError) {
          callback(new Error(currentError));
        } else {
          callback();
        }
      },
    } as RuleItem);
  }

  rulesWithServerError.push(...baseRules);

  // Normalize consistently by defaulting to a blur trigger
  return rulesWithServerError.map((r: any) => {
    if (r && typeof r === 'object' && !Array.isArray(r)) {
      const hasTrigger = Object.prototype.hasOwnProperty.call(r, 'trigger');
      const trg = r.trigger;
      if (!hasTrigger || (Array.isArray(trg) && trg.length === 0)) {
        return { ...r, trigger: 'blur' } as RuleItem;
      }
    }
    return r as RuleItem;
  });
});

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
  return String(row?.__rowKey ?? row?.key ?? rec?.Id ?? rec?.id ?? ++__autoRowKey);
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
const effectiveEditForRow = (row: T) => binding.env.isEditMode && cellVisibleForRow(row) && !readonlyForRow(row);

/* Clear server errors when the field value changes, after all variable definitions */
watch(
  () => rawValueForm().value,
  () => {
    if (serverError.value && fieldErrors?.value) {
      fieldErrors.value.delete(String(binding.prop));
    }
  }
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

<style scoped>
.o-field-base {
  padding: 0; /* keep wrapper neutral; satisfy linter */
}
.o-field-base__cell {
  display: block;
  width: 100%;
}
/* Compact error styling inside cells */
.o-field-base__cell-item :deep(.el-form-item__error) {
  white-space: normal;
}

/* Inline-mode error styles */
.o-field-base__inline {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.o-field-base__inline-wrap {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.o-inline-err-icon {
  color: var(--el-color-error);
}

.o-field-base__edit-wrap {
  display: flex;
  align-items: center;
  gap: 2px;
  width: 100%;
}
.o-field-base__edit-control {
  flex: 1 1 auto;
  min-width: 0;
}
.o-field-base__translate-btn {
  flex: 0 0 auto;
  height: 24px;
  width: 24px;
  padding: 0;
  color: var(--el-text-color-secondary);
}
.o-field-base__translate-btn:hover,
.o-field-base__translate-btn:focus {
  color: var(--el-color-primary);
}
.o-field-base__company-values-btn {
  flex: 0 0 auto;
  height: 24px;
  width: 24px;
  padding: 0;
  color: var(--el-text-color-secondary);
}
.o-field-base__company-values-btn:hover,
.o-field-base__company-values-btn:focus {
  color: var(--el-color-primary);
}
</style>
