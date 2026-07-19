<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFieldBase
    v-bind="$attrs"
    :binding="binding"
    :label="label"
    :rules="rules"
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
    <template #edit="{ fieldValue, record }">
      <el-select-v2
        class="o-many-to-one-select"
        :model-value="fieldValue().value"
        @update:model-value="(v: any) => onUpdate(fieldValue, v)"
        value-key="Id"
        :options="optionsFor(fieldValue().value as any)"
        :remote="true"
        :filterable="true"
        :clearable="clearable"
        :placeholder="effectivePlaceholder"
        :loading="loading"
        :remote-method="(q: string) => handleRemoteSearch(q, record)"
        @visible-change="onDropdownVisibleChange"
        @blur="() => {}"
        v-bind="selectProps"
        :style="{ width: width || '100%' }"
      >
        <template #footer>
          <div
            v-if="searchView"
            class="o-m2o__more o-m2o__more--clickable"
            role="button"
            tabindex="0"
            @click.stop="openSearchDialog(fieldValue, record)"
            @keydown.enter.stop="openSearchDialog(fieldValue, record)"
            @keydown.space.prevent.stop="openSearchDialog(fieldValue, record)"
          >
            {{ _t('Search more') }}
          </div>
        </template>
      </el-select-v2>
    </template>
    <template #display="{ fieldValue }">
      <span
        class="o-field-display-text"
        :class="{ 'o-field-display-text--clickable': isValueClickable }"
        :role="isValueClickable ? 'button' : undefined"
        :tabindex="isValueClickable ? 0 : undefined"
        @click="onDisplayValueClick(fieldValue().value as any, $event)"
        @keydown="onDisplayValueKeydown(fieldValue().value as any, $event)"
      >
        {{ getDisplayLabel(fieldValue().value as any) }}
      </span>
    </template>
  </OFieldBase>

  <el-dialog v-model="dialogVisible" :title="effectiveSearchViewTitle" :width="searchViewWidth" append-to-body destroy-on-close>
    <OViewScope view-mode="display" :container="'List'">
      <component
        v-if="searchView && relationStore"
        :is="searchView"
        ref="searchViewRef"
        :store="relationStore"
        :show-actions="false"
        :selection-mode="'single'"
        :click-to-select="true"
        :height-mode="'viewport'"
        :viewportGap="250"
        :condition="dialogEffectiveConditions"
        style="margin-top: -10px"
      />
    </OViewScope>
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="dialogVisible = false">{{ _t('Cancel') }}</el-button>
        <el-button type="primary" @click="confirmPick">{{ _t('OK') }}</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, ClientModel<BaseModel> | null | undefined>, V = FieldPathType<T, P>">
import { ref, shallowRef, computed, onMounted, onBeforeUnmount, inject, Ref, getCurrentInstance } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType, ClientModel, QueryCondition } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ElSelectV2, ElDialog, ElButton, type FormItemProps } from 'element-plus';
import type { WritableComputedRef, Component } from 'vue';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
// Narrow aggregation types to count_distinct only.
import type { NarrowAggProp, NonNumericAggFns } from '@/web/web/composables/useField';
import OViewScope from '@/web/web/components/view/OViewScope.vue';
import type { SelectionExpose } from '@/web/web/components/view/listViewTypes';
import { useProvidedOnchange } from '@/web/web/composables/useOnchange';
import { buildKeywordCondition } from '@/web/web/query/utils/condition/builder';
import { createTranslate } from '@/web/web/i18n';
import type { ValueClickPayload } from '@/web/web/components/field/manyToOneTypes';

const { _t } = createTranslate('web', { scope: 'web/components/field/OManyToOneField' });

type RelationValue = Extract<V, { Id: any }>;
type OptionType = { value: RelationValue; label: string };

defineOptions({ name: 'OManyToOneField', inheritAttrs: false });

type IsAny<T> = 0 extends 1 & T ? true : false;

const emit = defineEmits<{
  (e: 'value-click', payload: ValueClickPayload<any>): void;
}>();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;
    label?: string;
    rules?: RuleItem[];
    clearable?: boolean;
    placeholder?: string;
    pageSize?: number;
    formItemProps?: Partial<FormItemProps>;
    vColumnProps?: Record<string, any>;
    selectProps?: Partial<InstanceType<typeof ElSelectV2>['$props']>;
    width?: string;
    searchView?: Component;
    searchViewTitle?: string;
    searchViewWidth?: string | number;

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;

    condition?: QueryCondition<any> | QueryCondition<any>[];

    strategy?: 'live' | 'idle' | 'blur';
    idleDelay?: number;
    commitOnBlur?: boolean;
    // Only count_distinct is supported for foreign-key Id deduplication.
    agg?: NarrowAggProp<NonNumericAggFns>;
    // Added render mode and inline error support.
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
    valueClickable?: boolean | 'auto';
  }>(),
  {
    label: '',
    rules: () => [],
    clearable: true,
    placeholder: '',
    pageSize: 20,
    formItemProps: () => ({}),
    vColumnProps: () => ({}),
    selectProps: () => ({}),
    width: '100%',
    searchViewTitle: '',
    searchViewWidth: '70%',
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    condition: undefined,
    strategy: 'idle',
    idleDelay: 180,
    commitOnBlur: true,
    renderMode: 'auto',
    showInlineError: false,
    valueClickable: 'auto',
  }
);

const effectivePlaceholder = computed(() => props.placeholder || _t('Please select...'));
const effectiveSearchViewTitle = computed(() => props.searchViewTitle || _t('Select record'));

const binding = (props.binding ??
  useField<T, P, V>({
    store: props.store as WebModelStore<T>,
    prop: props.prop as P,
    // Forward agg to useField.
    agg: props.agg,
  })) as UseField<T, V>;
const relationStore = binding.relationStore as WebModelStore<any> | undefined;
binding.registerFields(`${binding.prop}.DisplayName`);

const toView = (raw: any) => (raw ?? null) as V | null;
const fromView = (v: V | null) => (v ?? null) as any;

const options = shallowRef<OptionType[]>([]);
const loading = ref(false);
const searchQuery = ref('');
const isSearching = computed(() => (searchQuery.value?.trim()?.length ?? 0) > 0);
const vm = getCurrentInstance();

const hasValueClickListener = computed<boolean>(() => {
  const p = (vm?.vnode.props || {}) as Record<string, any>;
  return Boolean(p.onValueClick || p['onValue-click']);
});

const isValueClickable = computed<boolean>(() => {
  if (props.valueClickable === true) return true;
  if (props.valueClickable === false) return false;
  return hasValueClickListener.value;
});

function asRelationValue(item: any): RelationValue | null {
  if (!item || typeof item !== 'object') return null;
  if (!('Id' in item)) return null;
  return item as RelationValue;
}

function toOption(item: RelationValue): OptionType {
  return { value: item, label: ((item as any)?.DisplayName as string) || '' };
}
function upsertOption(item?: V | null) {
  const relation = asRelationValue(item);
  if (!relation) return;
  const idx = options.value.findIndex(o => o.value.Id === relation.Id);
  const opt = toOption(relation);
  if (idx >= 0) options.value.splice(idx, 1, opt);
  else options.value.unshift(opt);
}
function composeOptionsForValue(val?: V | null): OptionType[] {
  const relation = asRelationValue(val);
  if (!relation) return options.value;
  const head = toOption(relation);
  const exists = options.value.findIndex(o => o.value.Id === relation.Id);
  if (exists < 0) return [head, ...options.value];
  const merged = options.value.slice();
  merged.splice(exists, 1, head);
  return merged;
}
function optionsFor(val?: V | null): OptionType[] {
  return isSearching.value ? options.value : composeOptionsForValue(val);
}

function getDisplayId(val: any): string {
  const id = val?.Id ?? val?.id;
  if (id == null) return '';
  return String(id).trim();
}

function getDisplayLabel(val: any): string {
  if (!val || typeof val !== 'object') return '';
  if (val.DisplayName != null) return String(val.DisplayName);
  if (val.Name != null) return String(val.Name);
  if (val.Title != null) return String(val.Title);
  const label = val.Code ?? val.Id ?? val.id;
  return label == null ? '' : String(label);
}

function onDisplayValueClick(val: any, event: MouseEvent) {
  if (!isValueClickable.value) return;
  const id = getDisplayId(val);
  if (!id) return;
  emit('value-click', { id, item: val && typeof val === 'object' ? val : null, label: getDisplayLabel(val), source: 'display', event });
}

function onDisplayValueKeydown(val: any, event: KeyboardEvent) {
  if (!isValueClickable.value) return;
  if (event.key !== 'Enter' && event.key !== ' ') return;
  event.preventDefault();
  const id = getDisplayId(val);
  if (!id) return;
  emit('value-click', { id, item: val && typeof val === 'object' ? val : null, label: getDisplayLabel(val), source: 'display', event });
}

const onchangeCtrl = useProvidedOnchange();
// Use the cumulative onchange result injected by OFormView.
const lastOnchangeResult = inject<Ref<any | null>>('lastOnchangeResult', ref(null));

function toArray<T>(v: T | T[] | undefined | null): T[] {
  if (v == null) return [];
  return Array.isArray(v) ? v : [v];
}

const externalConditions = computed<QueryCondition<any>[]>(() => toArray(props.condition));

// Read the root record through the injected form-root only.
const rootRecord = computed<any>(() => binding.recordRef().value as any);

const baseField = computed(() => String(binding.prop));
const segs = computed(() => baseField.value.split('.').filter(Boolean));
const leafField = computed(() => segs.value[segs.value.length - 1]);
const chainBeforeLeaf = computed(() => segs.value.slice(0, segs.value.length - 1));

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
  const leaf = leafField.value;
  const chain = chainBeforeLeaf.value;
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
  const leaf = leafField.value;
  const chain = chainBeforeLeaf.value;
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

function pickOnchangeConditions(rowRef?: any): QueryCondition<any>[] {
  const raw = lastOnchangeResult.value?.condition || [];
  if (!Array.isArray(raw) || !raw.length) return [];

  const chain = chainBeforeLeaf.value;
  if (!chain.length) {
    const key = baseField.value;
    return raw
      .filter((c: any) => c && c.field === key)
      .map((c: any) => c.condition)
      .filter(Boolean);
  }

  const row = normalizeRowRef(rowRef);
  if (!row) return [];
  const keys = new Set<string>([...buildFullChainKeys(row), ...buildLastLevelKeys(row)]);
  if (!keys.size) return [];
  return raw
    .filter((c: any) => c && typeof c.field === 'string' && keys.has(c.field))
    .map((c: any) => c.condition)
    .filter(Boolean);
}

const dialogVisible = ref(false);
const searchViewRef = ref<SelectionExpose<any> | null>(null);
const pendingTarget = ref<null | (() => WritableComputedRef<V | null>)>(null);
const dialogRowRef = ref<any | null>(null);

const dialogEffectiveConditions = computed<QueryCondition<any> | []>(() => {
  const parts: QueryCondition<any>[] = [...externalConditions.value, ...pickOnchangeConditions(dialogRowRef.value)];
  if (parts.length === 0) return [] as any;
  if (parts.length === 1) return parts[0] as any;
  return { And: parts } as any;
});

async function handleRemoteSearch(query: string, recordRef?: any) {
  searchQuery.value = query ?? '';
  loading.value = true;
  try {
    const parts: QueryCondition<any>[] = [];
    parts.push(...externalConditions.value);
    parts.push(...pickOnchangeConditions(recordRef));

    const kw = (query ?? '').trim();
    if (kw) {
      const kwExpr = buildKeywordCondition(kw, ['DisplayName'], {
        fieldsMeta: relationStore?.fieldsMetadata,
        operator: 'like',
        normalizeLike: true,
        mapDisplayNameToName: true,
        fallbackTextTypes: ['char', 'varchar'],
        includeIdInFallback: false,
        fallbackWhenFilteredEmpty: true,
      });
      if (kwExpr) parts.push(kwExpr);
    }

    const final: QueryCondition<any> | [] = parts.length === 0 ? ([] as any) : parts.length === 1 ? parts[0] : ({ And: parts } as any);

    const result = (await relationStore?.Search(final as any, {
      fields: ['Id', 'DisplayName'],
      limit: props.pageSize ?? 20,
    })) as V[] | undefined;

    options.value = (result ?? [])
      .map(asRelationValue)
      .filter((item): item is RelationValue => Boolean(item))
      .map(toOption);
  } finally {
    loading.value = false;
  }
}

function onUpdate(getter: () => WritableComputedRef<V | null>, v: V | null) {
  getter().value = (v ?? null) as any;
  if (v) upsertOption(v);
}

function onDropdownVisibleChange(_visible: boolean) {
  searchQuery.value = '';
}

function openSearchDialog(target?: () => WritableComputedRef<V | null>, recordRef?: any) {
  if (!relationStore) return;
  pendingTarget.value = target ?? null;
  dialogRowRef.value = recordRef ?? null;
  dialogVisible.value = true;
}

async function confirmPick() {
  try {
    const expose = searchViewRef.value as any;
    const unwrap = (v: any) => (v && typeof v === 'object' && 'value' in v ? v.value : v);
    const selFromSingle = unwrap(expose?.selectedItem);
    const selFromMulti = unwrap(expose?.selectedItems);
    const sel = selFromSingle ?? (Array.isArray(selFromMulti) ? selFromMulti[0] : null);
    const selId = sel?.Id;
    if (!selId) {
      dialogVisible.value = false;
      return;
    }
    const record = sel as V;
    const setter = pendingTarget.value;
    if (setter) setter().value = record as any;
    upsertOption(record);
  } finally {
    dialogVisible.value = false;
  }
}
</script>

<style scoped>
.o-form-field {
  margin-bottom: var(--o-form-field-margin-bottom, 18px);
}
.o-many-to-one-select {
  width: 100%;
}
.o-field-display-text {
  color: var(--el-text-color-regular);
  word-break: break-all;
}
.o-field-display-text--clickable {
  cursor: pointer;
  color: var(--el-color-primary);
  transition: color 0.16s ease;
}
.o-field-display-text--clickable:hover {
  color: var(--el-color-primary-dark-2);
}
.o-field-display-text--clickable:focus-visible {
  outline: 2px solid var(--el-color-primary-light-7);
  outline-offset: 2px;
  border-radius: 2px;
}
.o-m2o__more {
  text-align: center;
}
.o-m2o__more--clickable {
  display: block;
  width: 100%;
  padding: 5px 12px;
  cursor: pointer;
  user-select: none;
  color: var(--el-color-primary);
  background: transparent;
  transition:
    background-color 0.15s ease,
    color 0.15s ease;
}
.o-m2o__more--clickable:hover {
  background-color: var(--el-fill-color-light);
}
.o-m2o__more--clickable:active {
  background-color: var(--el-fill-color-lighter);
}
.o-m2o__more--clickable:focus,
.o-m2o__more--clickable:focus-visible {
  outline: none;
  box-shadow: 0 0 0 2px var(--el-color-primary-light-7) inset;
  border-radius: 2px;
}
</style>
