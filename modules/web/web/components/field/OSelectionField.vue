<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFieldBase
    v-bind="$attrs"
    :binding="binding"
    :label="label"
    :rules="mergedRules"
    :formItemProps="formItemProps"
    :vColumnProps="vColumnProps"
    :toView="toView"
    :fromView="fromView"
    :required="required"
    :readonly="readonly"
    :visible="visible"
    :cellVisible="cellVisible"
    :renderMode="renderMode"
    :showInlineError="showInlineError"
  >
    <!-- Shared form/row edit slot: derive options from fieldValue and record context -->
    <template #edit="{ fieldValue, record }">
      <el-select
        class="o-selection-field"
        :model-value="fieldValue().value"
        @update:model-value="(v: any) => onUpdate(fieldValue, v)"
        :clearable="clearable"
        :placeholder="effectivePlaceholder"
        :disabled="disabled || optionsLoading"
        :loading="optionsLoading"
        v-bind="selectProps"
        :style="{ width: width || '100%' }"
      >
        <el-option v-for="opt in optionsFor(record)" :key="opt.value" :label="opt.label" :value="opt.value" :disabled="opt.disabled" />
      </el-select>
    </template>

    <template #display="{ fieldValue, record }">
      <span class="o-field-display-text">{{ displayLabelFor(record, fieldValue().value) }}</span>
    </template>
  </OFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string | null | undefined>, V extends string = FieldPathType<T, P>">
import { computed, inject, onMounted, ref, type Ref } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ElSelect, ElOption, type FormItemProps } from 'element-plus';
import type { WritableComputedRef } from 'vue';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import type { NarrowAggProp, NonNumericAggFns } from '@/web/web/composables/useField';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/field/OSelectionField' });

defineOptions({ name: 'OSelectionField', inheritAttrs: false });

type IsAny<T> = 0 extends 1 & T ? true : false;

type MetaOption = { value: string; label: string };
type EffectiveOption = { value: string; label: string; disabled: boolean };

/** FieldsGet attributes used for selection option loading (D14). */
const SELECTION_FIELDS_GET_ATTRS = ['type', 'selection', 'selectionKind', 'string', 'stringText'];

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];

    clearable?: boolean;
    placeholder?: string;
    disabled?: boolean;

    formItemProps?: Partial<FormItemProps>;
    vColumnProps?: Record<string, any>;
    selectProps?: Partial<InstanceType<typeof ElSelect>['$props']>;
    width?: string;

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;
    agg?: NarrowAggProp<NonNumericAggFns>;
    /** Optional value whitelist / order filter. */
    selection?: string[];
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    rules: () => [],
    clearable: true,
    placeholder: '',
    disabled: false,
    formItemProps: () => ({}),
    vColumnProps: () => ({}),
    selectProps: () => ({}),
    width: '100%',
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    renderMode: 'auto',
    showInlineError: false,
  }
);

const effectivePlaceholder = computed(() => props.placeholder || _t('Please select...'));

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;

const toView = (raw: any): string | null => (raw == null ? null : String(raw));
const fromView = (v: string | null) => (v ?? null) as unknown as V;

const lastOnchangeResult = inject<Ref<any | null>>('lastOnchangeResult', ref(null));

const optionsLoading = ref(false);

const baseField = computed(() => String(binding.prop));
const pathSegs = computed(() => baseField.value.split('.').filter(Boolean));
const leafKey = computed(() => pathSegs.value[pathSegs.value.length - 1] || '');
const parentChain = computed(() => pathSegs.value.slice(0, pathSegs.value.length - 1));

const modelStore = computed(() => (binding.store || props.store) as WebModelStore<T> | undefined);

/**
 * Effective selection dictionary: FieldsGet overlay over static msgid labels.
 * Option labels are plain strings only — D5/D15 hard cut (no selection term refs on FE).
 */
const metaOptions = computed<MetaOption[]>(() => {
  const leaf = leafKey.value;
  const store = modelStore.value;
  const meta = (leaf && store?.getFieldMeta?.(leaf)) || binding.meta;
  const sel = meta?.selection;
  if (!Array.isArray(sel) || sel.length === 0) return [];
  return sel.map(item => ({
    value: String(item.value),
    label: String(item.label ?? item.value),
  }));
});

const propOptions = computed<MetaOption[]>(() => {
  const input = props.selection;
  const meta = metaOptions.value;
  if (!Array.isArray(input) || input.length === 0) return [];

  if (meta.length === 0) {
    const out: MetaOption[] = [];
    const seen = new Set<string>();
    for (const valRaw of input) {
      const val = valRaw != null ? String(valRaw) : '';
      if (!val || seen.has(val)) continue;
      out.push({ value: val, label: val });
      seen.add(val);
    }
    return out;
  }

  const metaMap = new Map(meta.map(i => [i.value, i.label]));
  const out: MetaOption[] = [];
  for (const valRaw of input) {
    const val = valRaw != null ? String(valRaw) : '';
    if (!val) continue;
    if (metaMap.has(val)) {
      out.push({ value: val, label: metaMap.get(val)! });
    }
  }
  return out;
});

const baseOptions = computed<MetaOption[]>(() => {
  if (propOptions.value.length > 0) return propOptions.value;
  return metaOptions.value;
});

onMounted(() => {
  const store = modelStore.value;
  const leaf = leafKey.value;
  if (!store?.ensureFieldsGet || !leaf) return;

  // First ensure happens on mount (D14) — not on visible-change.
  const request = store.ensureFieldsGet([leaf], SELECTION_FIELDS_GET_ATTRS);
  if (binding.env.isEditMode) {
    optionsLoading.value = true;
    void request.finally(() => {
      optionsLoading.value = false;
    });
  } else {
    // Display / readonly: background ensure once; store cache dedupes multi-cell mounts.
    void request;
  }
});

const rootRecord = computed<any>(() => binding.recordRef().value as any);

function normalizeRowRef(rowRef: any): any | null {
  try {
    if (!rowRef) return null;
    if (typeof rowRef === 'function') {
      const v = rowRef();
      return v && typeof v === 'object' && 'value' in v ? (v as any).value : v;
    }
    if (typeof rowRef === 'object' && 'value' in rowRef) return (rowRef as any).value;
    return rowRef;
  } catch {
    return null;
  }
}

type LevelSel = { key: string; id?: string; idx?: number };
function findSelectorsChain(root: any, chain: string[], targetId: string | number | null | undefined): LevelSel[] | null {
  if (!root || !Array.isArray(chain) || !chain.length || targetId == null) return null;
  const sid = String(targetId);

  function dfs(node: any, depth: number, acc: LevelSel[]): LevelSel[] | null {
    if (depth >= chain.length) return null;
    const key = chain[depth];
    const slot = node?.[key];

    if (Array.isArray(slot)) {
      for (let i = 0; i < slot.length; i++) {
        const row = slot[i];
        const nextAcc = acc.concat([{ key, id: row?.Id != null ? String(row.Id) : undefined, idx: i }]);
        if (depth === chain.length - 1) {
          if (row && String(row?.Id ?? '') === sid) return nextAcc;
        } else {
          const hit = dfs(row, depth + 1, nextAcc);
          if (hit) return hit;
        }
      }
      return null;
    }

    if (slot && typeof slot === 'object') {
      return dfs(slot, depth + 1, acc);
    }

    return null;
  }

  return dfs(root, 0, []);
}

function buildFullChainKeys(row: any): string[] {
  const out: string[] = [];
  const leaf = leafKey.value;
  const chain = parentChain.value;
  if (!rootRecord.value || !chain.length) return out;

  const rowId = row?.Id ?? row?.id ?? null;
  const selChain = findSelectorsChain(rootRecord.value, chain, rowId);
  if (!selChain) return out;

  const idPath = selChain.map(s => (s.id != null ? `${s.key}(id=${s.id})` : s.idx != null ? `${s.key}[${s.idx}]` : s.key)).join('.') + `.${leaf}`;
  out.push(idPath);

  const idxPath = selChain.map(s => (s.idx != null ? `${s.key}[${s.idx}]` : s.id != null ? `${s.key}(id=${s.id})` : s.key)).join('.') + `.${leaf}`;
  if (idxPath !== idPath) out.push(idxPath);

  return out;
}

function buildLastLevelKeys(row: any): string[] {
  const out: string[] = [];
  const leaf = leafKey.value;
  const chain = parentChain.value;
  if (!chain.length) return out;

  const lastIdx = chain.length - 1;
  const lastKey = chain[lastIdx];
  const head = chain.slice(0, lastIdx).join('.');
  const headDot = head ? head + '.' : '';

  const rowId = row?.Id ?? row?.id ?? null;
  if (rowId != null) out.push(`${headDot}${lastKey}(id=${String(rowId)}).${leaf}`);

  let idx: number | null = null;
  try {
    let node: any = rootRecord.value;
    for (let i = 0; i < lastIdx; i++) {
      node = node?.[chain[i]];
      if (Array.isArray(node)) {
        node = null;
        break;
      }
    }
    const arr = Array.isArray(node?.[lastKey]) ? (node[lastKey] as any[]) : Array.isArray(node) ? (node as any[]) : null;
    if (arr) {
      const idStr = rowId != null ? String(rowId) : null;
      const pos = idStr != null ? arr.findIndex(x => String(x?.Id ?? '') === idStr) : arr.findIndex(x => x === row);
      if (pos >= 0) idx = pos;
    }
  } catch {}
  if (idx != null) out.push(`${headDot}${lastKey}[${idx}].${leaf}`);

  return out;
}

function pickOnchangeSelection(rowRef?: any): { values: string[]; disabled?: string[] } | null {
  const raw = lastOnchangeResult.value?.selection || [];
  if (!Array.isArray(raw) || !raw.length) return null;

  const chain = parentChain.value;

  if (!chain.length) {
    const key = baseField.value;
    const m = raw.find((s: any) => s && s.field === key);
    if (!m) return null;
    return {
      values: Array.isArray(m.selection) ? m.selection : [],
      disabled: Array.isArray(m.disabled) ? m.disabled : undefined,
    };
  }

  const row = normalizeRowRef(rowRef);
  if (!row) return null;

  const keys = new Set<string>([...buildFullChainKeys(row), ...buildLastLevelKeys(row)]);
  if (!keys.size) return null;

  const m = raw.find((s: any) => s && typeof s.field === 'string' && keys.has(s.field));
  if (!m) return null;
  return {
    values: Array.isArray(m.selection) ? m.selection : [],
    disabled: Array.isArray(m.disabled) ? m.disabled : undefined,
  };
}

/**
 * options = FieldsGet.selection ∩ onchange.selection/disabled ∩ props.selection
 */
function optionsFor(rowRef?: any): EffectiveOption[] {
  const meta = metaOptions.value;
  const base = baseOptions.value;

  if (!meta.length) {
    return base.map(i => ({ value: i.value, label: i.label, disabled: false }));
  }

  const metaMap = new Map(meta.map(i => [i.value, i.label]));

  const filt = pickOnchangeSelection(rowRef);
  if (filt && Array.isArray(filt.values) && filt.values.length > 0) {
    const disabledSet = new Set(filt.disabled || []);
    return filt.values.filter(v => metaMap.has(v)).map(v => ({ value: v, label: metaMap.get(v)!, disabled: disabledSet.has(v) }));
  }

  return base.map(i => ({ value: i.value, label: i.label, disabled: false }));
}

function displayLabelFor(rowRef: any | undefined, value: string | null): string {
  if (value == null) return '';
  const meta = metaOptions.value;
  if (meta && meta.length) {
    const hit = meta.find(o => o.value === String(value));
    if (hit) return hit.label;
  }
  const opt = optionsFor(rowRef).find(o => o.value === String(value));
  return opt?.label || String(value);
}

function onUpdate(getter: () => WritableComputedRef<string | null>, v: string | null) {
  getter().value = (v ?? null) as any;
}

const internalRule = {
  type: 'string',
  validator: (_r: unknown, value: unknown, cb: (error?: Error) => void) => {
    if (value == null) return cb();
    if (typeof value !== 'string') return cb(new Error(_t('Value must be a string')));
    const ok = optionsFor().some(o => o.value === value);
    if (!ok) return cb(new Error(_t('Invalid option value: %s', value)));
    cb();
  },
} as RuleItem;
const mergedRules = computed<RuleItem[]>(() => [...(props.rules || []), internalRule]);
</script>

<style lang="scss" scoped>
.o-selection-field {
  width: 100%;
}
.o-field-display-text {
  line-height: var(--el-component-size-base, 32px);
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 0 11px;
}
</style>
