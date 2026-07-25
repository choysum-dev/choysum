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

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string[]>, V = FieldPathType<T, P>">
import { computed, ref, shallowRef, type Component, inject, Ref, watch, onBeforeUnmount, getCurrentInstance } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType, QueryCondition } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { FormItemProps } from 'element-plus';
import { ElButton, ElDialog, ElMessage, ElTag, ElSelectV2 } from 'element-plus';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import OViewScope from '@/web/web/components/view/OViewScope.vue';
import type { SelectionExpose } from '@/web/web/components/view/listViewTypes';
import { createStoreByModel } from '@/web/web/stores/registry';
import { registerFieldPath, unregisterFieldPath, pathsToFieldSelection, ensureRootId } from '@/web/web/query/utils/registry/field';
import { createTranslate } from '@/web/web/i18n';
import type { TagClickPayload } from '@/web/web/components/field/manyToManyTagsTypes';

const { _t } = createTranslate('web', { scope: 'web/components/field/OManyToManyRefTagsField' });

defineOptions({ name: 'OManyToManyRefTagsField', inheritAttrs: false });

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
const { getItems, insertItem, clearItems } = binding.asMutableArray<any>();

const relationStore = computed<WebModelStore<any> | undefined>(() => {
  if (binding.relationStore) return binding.relationStore as WebModelStore<any>;
  const target = props.targetModel || binding.meta?.relationModel;
  if (!target) return undefined;
  try {
    return createStoreByModel(target);
  } catch (e) {
    console.warn(`[OManyToManyRefTagsField] Failed to create store for model '${target}'`, e);
    return undefined;
  }
});

const dialogVisible = ref(false);
const searchViewRef = ref<SelectionExpose<any> | null>(null);
const loading = ref(false);
const hydratedCache = ref<Record<string, any>>({});
const hydratingIds = ref<Set<string>>(new Set());
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
  if (v == null) return undefined;
  if (typeof v === 'object') return (v as any).Id ?? (v as any).id;
  return String(v);
}

function resolveTagLabel(row: any, fallback?: string): string {
  if (!row || typeof row !== 'object') return fallback || '';
  for (const key of labelFields.value) {
    const value = (row as any)?.[key];
    if (value != null && String(value).trim()) return String(value);
  }
  return fallback || String((row as any)?.Id ?? (row as any)?.id ?? '');
}

function upsertHydrated(row: any) {
  if (!row || typeof row !== 'object') return;
  const id = String(row.Id ?? row.id ?? '');
  if (!id) return;
  hydratedCache.value[id] = { ...(row as any) };
}

const selectedIds = computed<string[]>(() => {
  return (getItems() || []).map(extractId).filter(Boolean).map(String);
});

const displayItems = computed(() => {
  const ids = selectedIds.value;
  const cap = Number(props.maxTagsVisible || 0);
  const visibleIds = cap > 0 ? ids.slice(0, cap) : ids;
  return visibleIds.map(id => {
    const rec = hydratedCache.value[id] || { Id: id };
    return { id, record: rec, label: resolveTagLabel(rec, id) };
  });
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
  for (const id of selectedIds.value) {
    const rec = hydratedCache.value[id] || { Id: id };
    map.set(id, { value: id, label: resolveTagLabel(rec, id), record: rec });
  }

  for (const rec of searchRows.value) {
    const id = extractId(rec);
    if (!id) continue;
    const key = String(id);
    if (picked.has(key)) continue;
    map.set(key, { value: key, label: resolveTagLabel(rec, key), record: rec });
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

const remoteFields = computed<string[]>(() => Array.from(new Set(['Id', ...labelFields.value])));

function pickHydrationFields() {
  const own = remoteFields.value && remoteFields.value.length ? remoteFields.value : [];
  const ensured = ensureRootId(pathsToFieldSelection(own) ?? own) || [];
  return ensured;
}

async function ensureHydrated(ids: string[]) {
  const store = relationStore.value;
  if (!store) return;
  const missing = ids.filter(id => !hydratedCache.value[id] && !hydratingIds.value.has(id));
  if (!missing.length) return;

  for (const id of missing) hydratingIds.value.add(id);
  try {
    const records = await store.Search(['Id', 'in', missing] as any, { fields: pickHydrationFields() as any } as any);
    const rows = Array.isArray(records) ? records : [];
    for (const row of rows) upsertHydrated(row);
  } catch (e) {
    console.warn('[OManyToManyRefTagsField] hydrate failed', e);
  } finally {
    for (const id of missing) hydratingIds.value.delete(id);
  }
}

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
        fields: pickHydrationFields() as any,
        limit: props.suggestLimit,
      } as any
    );
    const rows = Array.isArray(records) ? records : [];
    searchRows.value = rows.map(x => ({ ...(x || {}) }));
    for (const row of searchRows.value) upsertHydrated(row);
  } catch (e) {
    console.warn('[OManyToManyRefTagsField] search failed', e);
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

let registeredStoreId: string | null = null;
const registeredRefFields = ref<Set<string>>(new Set());

function syncRemoteFieldRegistration(store?: WebModelStore<any> | null, fields?: string[]) {
  const nextStoreId = store?.storeId ?? null;
  const nextSet = new Set(fields || []);
  const unchanged =
    nextStoreId === registeredStoreId && nextSet.size === registeredRefFields.value.size && Array.from(nextSet).every(f => registeredRefFields.value.has(f));
  if (unchanged) return;

  if (registeredStoreId && registeredRefFields.value.size) {
    for (const f of registeredRefFields.value) unregisterFieldPath(registeredStoreId, f);
  }
  registeredRefFields.value.clear();
  registeredStoreId = null;

  if (!store || !nextSet.size) return;
  for (const f of nextSet) registerFieldPath(store.storeId, f);
  registeredRefFields.value = nextSet;
  registeredStoreId = store.storeId;
}

watch(
  () => [relationStore.value, remoteFields.value] as const,
  ([store, fields]) => syncRemoteFieldRegistration(store, fields),
  { immediate: true }
);

onBeforeUnmount(() => {
  syncRemoteFieldRegistration(null, []);
});

watch(
  selectedIds,
  ids => {
    void ensureHydrated(ids);
  },
  { immediate: true }
);

function onSelectedIdsChange(values: any[]) {
  const prevIds = selectedIds.value;
  let nextIds = Array.from(new Set((Array.isArray(values) ? values : []).map(x => String(x || '')).filter(Boolean)));

  if (!props.tagClosable) {
    nextIds = Array.from(new Set([...prevIds, ...nextIds]));
  }

  const prev = new Set(prevIds);
  const next = new Set(nextIds);
  const removed = prevIds.some(id => !next.has(id));
  const added = nextIds.some(id => !prev.has(id));

  // Clear the keyword after selecting an item so stale input does not linger.
  if (added) searchKeyword.value = '';

  clearItems();
  for (const id of nextIds) insertItem(id as any);

  for (const id of nextIds) {
    if (!prev.has(id)) emit('tag-add', { id, item: hydratedCache.value[id] || { Id: id } });
  }
  for (const id of prevIds) {
    if (!next.has(id)) emit('tag-remove', { id, item: hydratedCache.value[id] || { Id: id } });
  }

  void ensureHydrated(nextIds);

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
    for (const row of selected) upsertHydrated(row);

    const ids = selected
      .map(x => extractId(x))
      .filter(Boolean)
      .map(String);
    if (!ids.length) {
      dialogVisible.value = false;
      return;
    }

    const prevIds = selectedIds.value;
    const prev = new Set(prevIds);
    const merged = Array.from(new Set([...prevIds, ...ids]));

    clearItems();
    for (const id of merged) insertItem(id as any);

    for (const id of merged) {
      if (!prev.has(id)) emit('tag-add', { id, item: hydratedCache.value[id] || { Id: id } });
    }

    emit('picker-confirm', { items: selected });
    await ensureHydrated(merged);
  } finally {
    dialogVisible.value = false;
  }
}

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
