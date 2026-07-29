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
    :required="required"
    :readonly="readonly"
    :visible="visible"
    :cellVisible="cellVisible"
    :renderMode="renderMode"
    :showInlineError="showInlineError"
  >
    <template #edit>
      <div class="o-m2m-tags">
        <el-select-v2
          class="o-m2m-tags__select"
          multiple
          filterable
          remote
          :reserve-keyword="false"
          :model-value="selectedIds"
          :options="selectOptions"
          :placeholder="effectivePlaceholder"
          :loading="loading"
          :clearable="false"
          :disabled="!relationStore"
          :style="{ width: width || '100%' }"
          :collapse-tags="maxTagsVisible > 0"
          :max-collapse-tags="maxTagsVisible > 0 ? maxTagsVisible : 1"
          v-bind="selectProps"
          :remote-method="handleRemoteSearch"
          @visible-change="onDropdownVisibleChange"
          @keydown="handleKeydown"
          @update:model-value="onSelectedIdsChange"
        >
          <template #default="{ item }">
            <slot name="suggestion" :item="item.record" :label="item.label">
              <span class="o-m2m-tags__suggestion" v-html="highlightSuggestion(item.label)"></span>
            </slot>
          </template>

          <template #footer>
            <div class="o-m2m-tags__footer">
              <slot name="suffix" />
              <div
                v-if="searchList"
                class="o-m2m-tags__more o-m2m-tags__more--clickable"
                role="button"
                tabindex="0"
                @click.stop="openPicker"
                @keydown.enter.stop="openPicker"
                @keydown.space.prevent.stop="openPicker"
              >
                {{ _t('Search more') }}
              </div>
            </div>
          </template>
        </el-select-v2>
      </div>
    </template>

    <template #display>
      <div class="o-m2m-tags o-m2m-tags--display">
        <template v-if="displayItems.length">
          <template v-for="item in displayItems" :key="item.id">
            <span
              class="o-m2m-tags__tag-hit"
              :class="{ 'o-m2m-tags__tag-hit--clickable': isTagClickable }"
              :role="isTagClickable ? 'button' : undefined"
              :tabindex="isTagClickable ? 0 : undefined"
              @click="onDisplayTagClick(item, $event)"
              @keydown="onDisplayTagKeydown(item, $event)"
            >
              <slot name="tag" :item="item.record" :label="item.label" :removable="false" :clickable="isTagClickable">
                <el-tag size="small" :type="isTagClickable ? 'primary' : 'info'" effect="plain">{{ item.label }}</el-tag>
              </slot>
            </span>
          </template>
          <el-tag v-if="hiddenCount > 0" size="small" type="info" effect="plain">+{{ hiddenCount }}</el-tag>
        </template>
        <slot v-else name="empty">
          <span class="o-m2m-tags__empty">{{ _t('None') }}</span>
        </slot>
      </div>
    </template>
  </OFieldBase>

  <el-dialog v-model="dialogVisible" :title="effectiveSearchViewTitle" :width="searchViewWidth" append-to-body destroy-on-close>
    <OViewScope view-mode="display">
      <component
        v-if="searchList && relationStore"
        :is="searchList"
        ref="searchViewRef"
        :store="relationStore"
        :show-actions="false"
        :click-to-select="true"
        :height-mode="'auto'"
        :forced-condition="effectiveConditions"
        style="margin-top: -10px"
      />
    </OViewScope>
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="dialogVisible = false">{{ _t('Cancel') }}</el-button>
        <el-button type="primary" @click="confirmPicker">{{ _t('OK') }}</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, ClientModel<BaseModel>[]>, V = FieldPathType<T, P>">
import { computed, ref, shallowRef, type Component, inject, Ref, watch, onMounted, getCurrentInstance } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, ClientModel, FieldPath, FieldPathType, QueryCondition } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { FormItemProps } from 'element-plus';
import { ElButton, ElDialog, ElMessage, ElTag, ElSelectV2 } from 'element-plus';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import { buildRelationalForField } from '@/web/web/composables/relationalForField';
import OViewScope from '@/web/web/components/view/OViewScope.vue';
import type { SelectionExpose } from '@/web/web/components/view/listViewTypes';
import { createStoreByModel } from '@/web/web/stores/registry';
import { createTranslate } from '@/web/web/i18n';
import type { TagClickPayload } from '@/web/web/components/field/manyToManyTagsTypes';

const { _t } = createTranslate('web', { scope: 'web/components/field/OManyToManyTagsField' });

defineOptions({ name: 'OManyToManyTagsField', inheritAttrs: false });

type IsAny<T> = 0 extends 1 & T ? true : false;

type SelectOption = {
  value: string;
  label: string;
  record: any;
};

const emit = defineEmits<{
  (e: 'tag-add', payload: { id: string; item: any }): void;
  (e: 'tag-remove', payload: { id: string; item: any }): void;
  (e: 'tag-click', payload: TagClickPayload<any>): void;
  (e: 'picker-open'): void;
  (e: 'picker-confirm', payload: { items: any[] }): void;
  (e: 'search', payload: { keyword: string }): void;
}>();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];
    formItemProps?: Partial<FormItemProps>;

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;

    condition?: QueryCondition<any> | QueryCondition<any>[];
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;

    searchList?: Component;
    searchViewTitle?: string;
    searchViewWidth?: string | number;
    targetModel?: string;

    tagLabelField?: string | string[];
    tagClickable?: boolean | 'auto';
    placeholder?: string;
    maxTagsVisible?: number;
    tagClosable?: boolean;
    suggestLimit?: number;
    width?: string;
    selectProps?: Partial<InstanceType<typeof ElSelectV2>['$props']>;
  }>(),
  {
    rules: () => [],
    formItemProps: () => ({}),
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    condition: undefined,
    renderMode: 'auto',
    showInlineError: false,
    searchViewTitle: '',
    searchViewWidth: '75%',
    tagLabelField: () => ['DisplayName', 'Name', 'Title', 'Code', 'Id'],
    tagClickable: 'auto',
    placeholder: '',
    maxTagsVisible: 0,
    tagClosable: true,
    suggestLimit: 20,
    width: '100%',
    selectProps: () => ({}),
  }
);

const effectivePlaceholder = computed(() => props.placeholder || _t('Please select...'));
const effectiveSearchViewTitle = computed(() => props.searchViewTitle || _t('Select related items'));

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P })) as UseField<T, V>;
binding.registerFields(`${binding.prop}.DisplayName`);
const { getItems, insertItem, clearItems } = binding.asMutableArray<any>();

const relationStore = computed<WebModelStore<any> | undefined>(() => {
  if (binding.relationStore) return binding.relationStore as WebModelStore<any>;
  const target = props.targetModel || binding.meta?.relationModel;
  if (!target) return undefined;
  try {
    return createStoreByModel(target);
  } catch (e) {
    console.warn(`[OManyToManyTagsField] Failed to create store for model '${target}'`, e);
    return undefined;
  }
});

const dialogVisible = ref(false);
const searchViewRef = ref<SelectionExpose<any> | null>(null);
const loading = ref(false);
const localCache = ref<Record<string, any>>({});
const searchRows = shallowRef<any[]>([]);
const searchKeyword = ref('');
const dropdownVisible = ref(false);
const vm = getCurrentInstance();

const hasTagClickListener = computed<boolean>(() => {
  const p = (vm?.vnode.props || {}) as Record<string, any>;
  return Boolean(p.onTagClick || p['onTag-click']);
});

const isTagClickable = computed<boolean>(() => {
  if (props.tagClickable === true) return true;
  if (props.tagClickable === false) return false;
  return hasTagClickListener.value;
});

const labelFields = computed<string[]>(() => {
  const raw = props.tagLabelField;
  const list = Array.isArray(raw) ? raw : [raw];
  const base = ['DisplayName', 'Name', 'Title', 'Code', 'Id'];
  return Array.from(new Set([...list.map(x => String(x || '').trim()).filter(Boolean), ...base]));
});

function extractId(v: any): string | undefined {
  if (v == null || typeof v !== 'object') return undefined;
  return (v as any).Id ?? (v as any).id;
}

function resolveTagLabel(row: any, fallback?: string): string {
  if (!row || typeof row !== 'object') return fallback || '';
  for (const key of labelFields.value) {
    const value = (row as any)?.[key];
    if (value != null && String(value).trim()) return String(value);
  }
  return fallback || String((row as any)?.Id ?? (row as any)?.id ?? '');
}

function readRowKeySeed(row: unknown): string | number | undefined {
  if (!row || typeof row !== 'object') return undefined;
  const r = row as Record<string, any>;
  return r.__rowKey ?? r.Id ?? r.id;
}

function defineHiddenRowKey(obj: any, key: string, val?: any) {
  if (!obj || typeof obj !== 'object') return;
  try {
    const hasOwn = Object.prototype.hasOwnProperty.call(obj, key);
    const enumerable = hasOwn ? Object.prototype.propertyIsEnumerable.call(obj, key) : false;
    if (!hasOwn) {
      Object.defineProperty(obj, key, {
        value: String(val ?? Math.random().toString(36).slice(2)),
        enumerable: false,
        configurable: false,
        writable: true,
      });
      return;
    }
    if (enumerable) {
      const v = val ?? obj[key];
      delete obj[key];
      Object.defineProperty(obj, key, {
        value: String(v ?? Math.random().toString(36).slice(2)),
        enumerable: false,
        configurable: false,
        writable: true,
      });
    }
  } catch {}
}

function createRowKey(v: any) {
  if (!v) return v;
  const seed = readRowKeySeed(v) ?? Math.random().toString(36).slice(2);
  defineHiddenRowKey(v, '__rowKey', seed);
  return v;
}

function hydrateRowKeys() {
  const arr = getItems() || [];
  for (const row of arr) {
    if (!row || typeof row !== 'object') continue;
    const seed = readRowKeySeed(row);
    defineHiddenRowKey(row, '__rowKey', seed);
    const id = extractId(row);
    if (id) localCache.value[String(id)] = row;
  }
}

const selectedIds = computed<string[]>(() => {
  return (getItems() || []).map(extractId).filter(Boolean).map(String);
});

const displayItems = computed(() => {
  const current = getItems() || [];
  const cap = Number(props.maxTagsVisible || 0);
  const visibleItems = cap > 0 ? current.slice(0, cap) : current;
  return visibleItems
    .map(row => {
      const id = extractId(row);
      if (!id) return null;
      const rec = row || localCache.value[String(id)] || { Id: id };
      return { id: String(id), record: rec, label: resolveTagLabel(rec, String(id)) };
    })
    .filter(Boolean) as Array<{ id: string; record: any; label: string }>;
});

const hiddenCount = computed(() => {
  const cap = Number(props.maxTagsVisible || 0);
  if (cap <= 0) return 0;
  return Math.max(0, selectedIds.value.length - cap);
});

function onDisplayTagClick(item: { id: string; record: any; label: string }, event: MouseEvent) {
  if (!isTagClickable.value) return;
  emit('tag-click', { id: item.id, item: item.record, label: item.label, source: 'display', event });
}

function onDisplayTagKeydown(item: { id: string; record: any; label: string }, event: KeyboardEvent) {
  if (!isTagClickable.value) return;
  if (event.key !== 'Enter' && event.key !== ' ') return;
  event.preventDefault();
  emit('tag-click', { id: item.id, item: item.record, label: item.label, source: 'display', event });
}

const selectOptions = computed<SelectOption[]>(() => {
  const map = new Map<string, SelectOption>();
  const picked = new Set(selectedIds.value);

  // Selected values must always remain in options so el-select-v2 can render tags.
  // Hiding them only while the dropdown is open leaves model-value without matching
  // options and can crash the renderer on the next selection.
  for (const row of getItems() || []) {
    const id = extractId(row);
    if (!id) continue;
    const rec = row || localCache.value[String(id)] || { Id: id };
    map.set(String(id), { value: String(id), label: resolveTagLabel(rec, String(id)), record: rec });
  }

  for (const row of searchRows.value) {
    const id = extractId(row);
    if (!id) continue;
    const key = String(id);
    if (picked.has(key)) continue;
    map.set(key, { value: key, label: resolveTagLabel(row, key), record: row });
  }
  return Array.from(map.values());
});

function onDropdownVisibleChange(visible: boolean) {
  dropdownVisible.value = Boolean(visible);
}

const lastOnchangeResult = inject<Ref<any | null>>('lastOnchangeResult', ref(null));

function toArray<T>(v: T | T[] | undefined | null): T[] {
  if (v == null) return [];
  return Array.isArray(v) ? v : [v];
}

const excludePicked = computed<QueryCondition<any> | undefined>(() => {
  const ids = selectedIds.value;
  if (!ids.length) return undefined;
  return ['Id', 'not in', ids] as unknown as QueryCondition<any>;
});

const externalConditions = computed<QueryCondition<any>[]>(() => toArray(props.condition));

const fieldName = computed(() => String(binding.prop));
const onchangeConditions = computed<QueryCondition<any>[]>(() => {
  const raw = lastOnchangeResult.value?.condition || [];
  return raw
    .filter((c: any) => c?.field === fieldName.value)
    .map((c: any) => c?.condition)
    .filter(Boolean);
});

const effectiveConditions = computed<QueryCondition<any> | []>(() => {
  const parts: QueryCondition<any>[] = [];
  if (excludePicked.value) parts.push(excludePicked.value);
  parts.push(...externalConditions.value, ...onchangeConditions.value);
  if (parts.length === 0) return [] as any;
  if (parts.length === 1) return parts[0];
  return { And: parts } as any;
});

function escapeHtml(text: string): string {
  return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;');
}

function highlightSuggestion(label: string): string {
  const raw = String(label || '');
  const keyword = String(searchKeyword.value || '').trim();
  if (!keyword) return escapeHtml(raw);

  const lower = raw.toLowerCase();
  const needle = keyword.toLowerCase();
  if (!needle) return escapeHtml(raw);

  let cursor = 0;
  let pos = lower.indexOf(needle, cursor);
  if (pos < 0) return escapeHtml(raw);

  let out = '';
  while (pos >= 0) {
    out += escapeHtml(raw.slice(cursor, pos));
    out += `<span class="o-m2m-tags__suggestion-hit">${escapeHtml(raw.slice(pos, pos + needle.length))}</span>`;
    cursor = pos + needle.length;
    pos = lower.indexOf(needle, cursor);
  }
  out += escapeHtml(raw.slice(cursor));
  return out;
}

async function handleRemoteSearch(keyword: string) {
  searchKeyword.value = String(keyword || '');
  emit('search', { keyword: String(keyword || '') });
  const store = relationStore.value;
  if (!store) {
    searchRows.value = [];
    return;
  }

  loading.value = true;
  try {
    const records = await store.NameSearch(
      String(keyword || '').trim(),
      effectiveConditions.value as any,
      {
        fields: Array.from(new Set(['Id', ...labelFields.value])) as any,
        limit: props.suggestLimit,
        ...buildRelationalForField(props.store as any, String(binding.prop)),
      } as any
    );
    const rows = Array.isArray(records) ? records : [];
    searchRows.value = rows.map(x => ({ ...(x || {}) }));
    for (const row of searchRows.value) {
      const id = extractId(row);
      if (id) localCache.value[String(id)] = row;
    }
  } catch (e) {
    console.warn('[OManyToManyTagsField] search failed', e);
    searchRows.value = [];
  } finally {
    loading.value = false;
  }
}

function handleKeydown(event: KeyboardEvent) {
  if ((event as any)?.isComposing) return;

  if (event.key === 'Backspace') {
    const keyword = String(searchKeyword.value || '').trim();
    if (!keyword && props.tagClosable && selectedIds.value.length) {
      event.preventDefault();
      onSelectedIdsChange(selectedIds.value.slice(0, -1));
    }
    return;
  }

  if (event.key !== 'Enter') return;
  if (loading.value) return;

  const keyword = String(searchKeyword.value || '').trim();
  if (!keyword) return;

  const selectedSet = new Set(selectedIds.value);
  const first = searchRows.value.find(row => {
    const id = extractId(row);
    return id && !selectedSet.has(String(id));
  });
  const id = extractId(first);
  if (!id) return;

  event.preventDefault();
  onSelectedIdsChange([...selectedIds.value, String(id)]);
}

function onSelectedIdsChange(values: any[]) {
  const currentRows = getItems() || [];
  const currentIds = currentRows.map(extractId).filter(Boolean).map(String);
  let nextIds = Array.from(new Set((Array.isArray(values) ? values : []).map(x => String(x || '')).filter(Boolean)));

  if (!props.tagClosable) {
    nextIds = Array.from(new Set([...currentIds, ...nextIds]));
  }

  const currentMap = new Map<string, any>();
  for (const row of currentRows) {
    const id = extractId(row);
    if (!id) continue;
    currentMap.set(String(id), row);
  }
  for (const row of searchRows.value) {
    const id = extractId(row);
    if (!id) continue;
    currentMap.set(String(id), row);
  }

  const prev = new Set(currentIds);
  const next = new Set(nextIds);
  const removed = currentIds.some(id => !next.has(id));
  const added = nextIds.some(id => !prev.has(id));

  // Clear the keyword after selecting an item so stale input does not linger.
  if (added) searchKeyword.value = '';

  clearItems();
  for (const id of nextIds) {
    const row = currentMap.get(id) || localCache.value[id] || { Id: id };
    insertItem(createRowKey({ ...(row || {}) }) as any);
  }

  for (const id of nextIds) {
    if (!prev.has(id)) emit('tag-add', { id, item: currentMap.get(id) || localCache.value[id] || { Id: id } });
  }
  for (const id of currentIds) {
    if (!next.has(id)) emit('tag-remove', { id, item: currentMap.get(id) || localCache.value[id] || { Id: id } });
  }

  // Requery immediately after tag removal so the open dropdown does not keep stale results.
  if (dropdownVisible.value && removed) {
    void handleRemoteSearch(searchKeyword.value);
  }
}

function openPicker() {
  emit('picker-open');
  if (!props.searchList) {
    ElMessage.warning(_t('searchList is not configured; cannot open picker'));
    return;
  }
  if (!relationStore.value) {
    ElMessage.warning(_t('relationStore is unresolved; cannot open picker'));
    return;
  }
  dialogVisible.value = true;
}

async function confirmPicker() {
  try {
    const expose = searchViewRef.value as any;
    const unwrap = (v: any) => (v && typeof v === 'object' && 'value' in v ? v.value : v);
    const picked = unwrap(expose?.selectedItems) as any[] | undefined;
    const toRecord = (x: any) =>
      x && typeof x === 'object' && x.kind === 'record' && x.payload ? x.payload : x && typeof x === 'object' && x.type === 'record' && x.record ? x.record : x;
    const selected: any[] = Array.isArray(picked) ? picked.map(toRecord) : [];

    const ids = selected
      .map(x => extractId(x))
      .filter(Boolean)
      .map(String);
    if (!ids.length) {
      dialogVisible.value = false;
      return;
    }

    const currentRows = getItems() || [];
    const currentIds = currentRows.map(extractId).filter(Boolean).map(String);
    const currentMap = new Map<string, any>();
    for (const row of currentRows) {
      const id = extractId(row);
      if (!id) continue;
      currentMap.set(String(id), row);
    }
    for (const row of selected) {
      const id = extractId(row);
      if (!id) continue;
      currentMap.set(String(id), row);
      localCache.value[String(id)] = row;
    }

    const mergedIds = Array.from(new Set([...currentIds, ...ids]));
    const prev = new Set(currentIds);

    clearItems();
    for (const id of mergedIds) {
      const row = currentMap.get(id) || localCache.value[id] || { Id: id };
      insertItem(createRowKey({ ...(row || {}) }) as any);
    }

    for (const id of mergedIds) {
      if (!prev.has(id)) emit('tag-add', { id, item: currentMap.get(id) || localCache.value[id] || { Id: id } });
    }

    emit('picker-confirm', { items: selected });
  } finally {
    dialogVisible.value = false;
  }
}

onMounted(hydrateRowKeys);
watch(
  () => getItems().length,
  () => hydrateRowKeys(),
  { immediate: true }
);

defineSlots<{
  tag(props: { item: any; label: string; removable: boolean; clickable: boolean }): any;
  suggestion(props: { item: any; label: string }): any;
  empty(): any;
  suffix(): any;
}>();
</script>

<style scoped>
.o-m2m-tags {
  width: 100%;
}

.o-m2m-tags__select {
  width: 100%;
}

.o-m2m-tags--display {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-height: 28px;
  align-items: center;
}

.o-m2m-tags__tag-hit {
  display: inline-flex;
  align-items: center;
}

.o-m2m-tags__tag-hit--clickable {
  cursor: pointer;
}

.o-m2m-tags__tag-hit--clickable :deep(.el-tag) {
  transition:
    color 0.16s ease,
    border-color 0.16s ease,
    background-color 0.16s ease;
}

.o-m2m-tags__tag-hit--clickable:hover :deep(.el-tag) {
  color: var(--el-color-primary-dark-2);
  border-color: var(--el-color-primary);
  background-color: var(--el-color-primary-light-8);
}

.o-m2m-tags__empty {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

.o-m2m-tags__footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 6px 8px;
}

.o-m2m-tags__more {
  font-size: 12px;
  color: var(--el-color-primary);
}

.o-m2m-tags__more--clickable {
  cursor: pointer;
}

.o-m2m-tags__suggestion-hit {
  color: var(--el-color-primary);
  font-weight: 600;
}
</style>
