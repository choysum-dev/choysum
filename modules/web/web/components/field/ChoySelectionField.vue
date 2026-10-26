<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    v-bind="$attrs"
    :binding="binding"
    :label="label"
    :rules="mergedRules"
    :formItemProps="formItemProps"
    :orientation="orientation"
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
    <!-- shadcn-vue Combobox: button trigger + search in list + check on selected. -->
    <template #edit="{ fieldValue, record, inputId, inputName, describedBy, invalid}">
      <div
        class="choy-selection-field w-full min-w-0"
        :data-loading="optionsLoading ? 'true' : 'false'"
        :style="{ width: width || '100%' }"
      >
        <Combobox
          class="w-full"
          :model-value="fieldValue().value ?? undefined"
          :disabled="disabled || optionsLoading"
          @update:model-value="(v: unknown) => onUpdate(fieldValue, v ? String(v) : null)"
        >
          <ComboboxAnchor class="w-full">
            <ComboboxTrigger
              :id="inputId"
              :name="inputName"
              class="w-full"
              :disabled="disabled || optionsLoading"
              :aria-describedby="describedBy || undefined"
              :aria-invalid="invalid ? 'true' : undefined"
            >
              <span
                class="min-w-0 flex-1 truncate text-start"
                :class="fieldValue().value == null ? 'text-muted-foreground' : undefined"
              >
                {{ displayLabelFor(record, fieldValue().value) || effectivePlaceholder }}
              </span>
              <span class="inline-flex shrink-0 items-center gap-1">
                <button
                  v-if="clearable && fieldValue().value != null && !(disabled || optionsLoading)"
                  type="button"
                  class="choy-selection-field__clear inline-flex text-foreground/50 hover:text-foreground"
                  :aria-label="_t('Clear')"
                  data-testid="choy-selection-clear"
                  @click.stop.prevent="onUpdate(fieldValue, null)"
                >
                  <X class="size-3.5" aria-hidden="true" />
                </button>
                <ChevronsUpDown class="size-4 opacity-50" aria-hidden="true" />
              </span>
            </ComboboxTrigger>
          </ComboboxAnchor>
          <ComboboxContent class="p-0">
            <!-- Short lists: skip panel search (ERP Selection). Longer lists: bare filter row. -->
            <div
              v-if="optionsFor(record).length > 8"
              data-slot="combobox-input-wrapper"
              class="flex h-control items-center gap-2 border-b border-border px-3"
            >
              <Search class="size-4 shrink-0 opacity-50" aria-hidden="true" />
              <ComboboxInput
                variant="bare"
                :placeholder="_t('Search…')"
                :disabled="disabled || optionsLoading"
              />
            </div>
            <ComboboxViewport>
              <div
                v-if="optionsLoading"
                class="px-2 py-1.5 text-sm text-muted-foreground"
              >
                {{ _t('Loading…') }}
              </div>
              <template v-else>
                <ComboboxEmpty>{{ _t('No options') }}</ComboboxEmpty>
                <ComboboxItem
                  v-for="opt in optionsFor(record)"
                  :key="opt.value"
                  :value="opt.value"
                  :disabled="opt.disabled"
                >
                  {{ opt.label }}
                </ComboboxItem>
              </template>
            </ComboboxViewport>
          </ComboboxContent>
        </Combobox>
      </div>
    </template>

    <template #display="{ fieldValue, record }">
      <span class="choy-field-display-text inline-block max-w-full truncate whitespace-nowrap text-foreground">{{ displayLabelFor(record, fieldValue().value) }}</span>
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string | null | undefined>, V extends string = FieldPathType<T, P>">
import { computed, inject, onMounted, ref, type Ref } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { WritableComputedRef } from 'vue';
import { ChevronsUpDown, Search, X } from 'lucide-vue-next';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import Combobox from '../vendor/ui/combobox/Combobox.vue';
import ComboboxAnchor from '../vendor/ui/combobox/ComboboxAnchor.vue';
import ComboboxContent from '../vendor/ui/combobox/ComboboxContent.vue';
import ComboboxEmpty from '../vendor/ui/combobox/ComboboxEmpty.vue';
import ComboboxInput from '../vendor/ui/combobox/ComboboxInput.vue';
import ComboboxItem from '../vendor/ui/combobox/ComboboxItem.vue';
import ComboboxTrigger from '../vendor/ui/combobox/ComboboxTrigger.vue';
import ComboboxViewport from '../vendor/ui/combobox/ComboboxViewport.vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import type { NarrowAggProp, NonNumericAggFns } from '@/web/web/composables/useField';
import { createTranslate } from '@/web/web/i18n';
import { FIELD_PRESENTATION_FIELDS_GET_ATTRS } from '@/web/web/stores/fieldsGet';

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoySelectionField' });

defineOptions({ name: 'ChoySelectionField', inheritAttrs: false });

type IsAny<T> = 0 extends 1 & T ? true : false;

type MetaOption = { value: string; label: string };
type EffectiveOption = { value: string; label: string; disabled: boolean };

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
    orientation?: 'horizontal' | 'vertical';
    vColumnProps?: Record<string, any>;
    /** Kept for call-site compat; Combobox filtering is always on. */
    selectProps?: Record<string, unknown>;
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
  const request = store.ensureFieldsGet([leaf], [...FIELD_PRESENTATION_FIELDS_GET_ATTRS]);
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

  const rowId = row?.Id ?? null;
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

  const rowId = row?.Id ?? null;
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

