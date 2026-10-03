<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <FieldBase
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
      <div class="choy-m2m-tags w-full" @keydown="handleKeydown">
        <div class="mb-1.5 flex min-h-0 flex-wrap items-center gap-1.5">
          <template v-for="item in editChipItems" :key="item.id">
            <span class="choy-m2m-tags__chip inline-flex items-center">
              <slot
                name="tag"
                :item="item.record"
                :label="item.label"
                :removable="tagClosable"
                :clickable="false"
              >
                <span class="choy-m2m-tags__tag inline-flex items-center gap-1 rounded-sm border border-border bg-muted px-2 py-0.5 text-sm text-foreground transition-colors">
                  {{ item.label }}
                  <button
                    v-if="tagClosable"
                    type="button"
                    class="cursor-pointer border-0 bg-transparent p-0 text-sm leading-none text-inherit"
                    :aria-label="_t('Remove')"
                    data-testid="choy-m2m-tag-remove"
                    @click.stop="removeChip(item.id)"
                  >
                    ×
                  </button>
                </span>
              </slot>
            </span>
          </template>
          <span v-if="hiddenCount > 0" class="choy-m2m-tags__tag inline-flex items-center gap-1 rounded-sm border border-border bg-muted px-2 py-0.5 text-sm text-foreground transition-colors">+{{ hiddenCount }}</span>
        </div>
        <RelationCombobox
          v-model="addModel"
          class="choy-m2m-tags__select w-full"
          :search="relationSearch"
          :search-key="relationSearchKey"
          :clearable="false"
          :placeholder="effectivePlaceholder"
          :page-size="suggestLimit"
          :disabled="!relationStore"
          :search-more="Boolean(searchList)"
          :style="{ width: width || '100%' }"
          @select="onComboboxSelect"
          @search-more="onSearchMore"
        />
        <div class="flex items-center justify-end gap-2 py-1.5">
          <slot name="suffix" />
          <div
            v-if="showNameCreateEntry"
            class="cursor-pointer text-xs text-primary"
            role="button"
            tabindex="0"
            data-testid="choy-m2m-name-create"
            @click.stop="onNameCreate"
            @keydown.enter.stop="onNameCreate"
            @keydown.space.prevent.stop="onNameCreate"
          >
            {{ nameCreateLabel }}
          </div>
        </div>
      </div>
    </template>

    <template #display>
      <div class="choy-m2m-tags choy-m2m-tags--display flex min-h-7 w-full flex-wrap items-center gap-1.5">
        <template v-if="displayItems.length">
          <template v-for="item in displayItems" :key="item.id">
            <span
              class="choy-m2m-tags__tag-hit inline-flex items-center"
              :class="{ 'choy-m2m-tags__tag-hit--clickable group cursor-pointer': isTagClickable }"
              :role="isTagClickable ? 'button' : undefined"
              :tabindex="isTagClickable ? 0 : undefined"
              @click="onDisplayTagClick(item, $event)"
              @keydown="onDisplayTagKeydown(item, $event)"
            >
              <slot name="tag" :item="item.record" :label="item.label" :removable="false" :clickable="isTagClickable">
                <span class="choy-m2m-tags__tag inline-flex items-center gap-1 rounded-sm border border-border bg-muted px-2 py-0.5 text-sm text-foreground transition-colors" :class="{ 'choy-m2m-tags__tag--primary border-primary bg-primary/10 text-primary group-hover:border-primary group-hover:bg-primary/15': isTagClickable }">{{ item.label }}</span>
              </slot>
            </span>
          </template>
          <span v-if="hiddenCount > 0" class="choy-m2m-tags__tag inline-flex items-center gap-1 rounded-sm border border-border bg-muted px-2 py-0.5 text-sm text-foreground transition-colors">+{{ hiddenCount }}</span>
        </template>
        <slot v-else name="empty">
          <span class="choy-m2m-tags__empty text-sm text-muted-foreground">{{ _t('None') }}</span>
        </slot>
      </div>
    </template>
  </FieldBase>

  <Dialog v-model:open="dialogVisible">
    <DialogContent class="choy-relation-picker-dialog" :style="{ width: typeof searchViewWidth === 'number' ? searchViewWidth + 'px' : searchViewWidth }">
      <DialogTitle>{{ effectiveSearchViewTitle }}</DialogTitle>
      <ChoyViewScope view-mode="display">
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
    </ChoyViewScope>
      <div class="dialog-footer">
        <ChoyButton @click="dialogVisible = false">{{ _t('Cancel') }}</ChoyButton>
        <ChoyButton @click="confirmPicker">{{ _t('OK') }}</ChoyButton>
      </div>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string[]>, V = FieldPathType<T, P>">
import { computed, ref, shallowRef, type Component, inject, Ref, watch, onBeforeUnmount, getCurrentInstance, nextTick } from 'vue';
import { ChoyDialog as Dialog, ChoyDialogContent as DialogContent, ChoyDialogTitle as DialogTitle } from '@/web/web/components/layout/choyDialog';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { ChoyMessage } from '../../composables/useChoyMessage';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType, QueryCondition } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import FieldBase, { type FieldStateExpr, type FormItemProps } from './FieldBase.vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import { buildRelationConditionSource } from '@/web/web/composables/relationalForField';
import ChoyViewScope from '@/web/web/components/view/ChoyViewScope.vue';
import type { SelectionExpose } from '@/web/web/components/view/listViewTypes';
import { createStoreByModel } from '@/web/web/stores/registry';
import { registerFieldPath, unregisterFieldPath, pathsToFieldSelection, ensureRootId } from '@/web/web/query/utils/registry/field';
import { createTranslate } from '@/web/web/i18n';
import type { TagClickPayload } from '@/web/web/components/field/manyToManyTagsTypes';
import { shouldShowNameCreateEntry } from '@/web/web/components/field/nameCreateVisibility';
import { runNameCreateQuickCreate, trimSearchKeyword } from '@/web/web/components/field/nameCreateQuickCreate';
import { usePermission } from '@/auth/web/composables/usePermission';
import RelationCombobox from '@/web/web/components/internal/RelationCombobox.vue';
import {
  mapNameSearchRows,
  type RelationOption,
} from '@/web/web/components/internal/relationComboboxHelpers';

const { _t } = createTranslate('web', { scope: 'web/components/field/ManyToManyRefTagsField' });

defineOptions({ name: 'ManyToManyRefTagsField', inheritAttrs: false });

type IsAny<T> = 0 extends 1 & T ? true : false;

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

    /** Quick-create via NameCreate (PR-P2-M1). Default false (opt-in); gated by create UI action. */
    allowCreate?: boolean;
    /** Target model write field for NameCreate; omit → BE uses Name. */
    nameField?: string;
    /** Override create UI action id; '' skips ACL (see namecreate-design D6). */
    createActionId?: string;

    tagLabelField?: string | string[];
    tagClickable?: boolean | 'auto';
    placeholder?: string;
    maxTagsVisible?: number;
    tagClosable?: boolean;
    suggestLimit?: number;
    width?: string;
    selectProps?: Record<string, unknown>;
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
    allowCreate: false,
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
const addModel = ref<string | null>(null);
const vm = getCurrentInstance();
const relationSearchKey = computed(
  () => `${relationStore.value?.storeId || relationStore.value?.fullModelName || ''}:${String(binding.prop)}`
);

const hasKeyword = computed(() => trimSearchKeyword(searchKeyword.value).length > 0);
const { hasAction } = usePermission();
const showNameCreateEntry = computed(() =>
  shouldShowNameCreateEntry({
    allowCreate: props.allowCreate === true,
    hasKeyword: hasKeyword.value,
    relationQualifiedName: relationStore.value?.fullModelName,
    createActionId: props.createActionId,
    hasAction,
  })
);
const nameCreateLabel = computed(() => _t('Create "%s"', trimSearchKeyword(searchKeyword.value)));
const creatingName = ref(false);

async function onNameCreate() {
  await runNameCreateQuickCreate({
    busy: creatingName,
    store: relationStore.value,
    keyword: searchKeyword.value,
    nameField: props.nameField,
    failedMessage: _t('Create failed'),
    onError: message => ChoyMessage.error(message),
    onSuccess: (row, id) => {
      upsertHydrated(row);
      if (!selectedIds.value.includes(id)) {
        onSelectedIdsChange([...selectedIds.value, id]);
      }
      searchKeyword.value = '';
    },
  });
}

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
  if (typeof v === 'object') return (v as any).Id;
  return String(v);
}

function resolveTagLabel(row: any, fallback?: string): string {
  if (!row || typeof row !== 'object') return fallback || '';
  for (const key of labelFields.value) {
    const value = (row as any)?.[key];
    if (value != null && String(value).trim()) return String(value);
  }
  return fallback || String((row as any)?.Id ?? '');
}

function upsertHydrated(row: any) {
  if (!row || typeof row !== 'object') return;
  const id = String(row.Id ?? '');
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

/** Edit-mode chips use the same visibility cap as display. */
const editChipItems = computed(() => displayItems.value);

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

function onDropdownVisibleChange(visible: boolean) {
  dropdownVisible.value = Boolean(visible);
}

function removeChip(id: string) {
  if (!props.tagClosable) return;
  onSelectedIdsChange(selectedIds.value.filter(x => x !== id));
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
    out += `<span class="choy-m2m-tags__suggestion-hit text-primary font-semibold">${escapeHtml(raw.slice(pos, pos + needle.length))}</span>`;
    cursor = pos + needle.length;
    pos = lower.indexOf(needle, cursor);
  }
  out += escapeHtml(raw.slice(cursor));
  return out;
}

async function relationSearch(keyword: string, opts: { limit: number }): Promise<RelationOption[]> {
  searchKeyword.value = String(keyword || '');
  emit('search', { keyword: String(keyword || '') });
  const store = relationStore.value;
  if (!store) {
    searchRows.value = [];
    return [];
  }

  loading.value = true;
  try {
    const records = await store.NameSearch(
      String(keyword || '').trim(),
      effectiveConditions.value as any,
      {
        fields: pickHydrationFields() as any,
        limit: opts.limit ?? props.suggestLimit,
        ...buildRelationConditionSource(props.store as any, binding.prop),
      } as any
    );
    const rows = Array.isArray(records) ? records : [];
    searchRows.value = rows.map(x => ({ ...(x || {}) }));
    for (const row of searchRows.value) upsertHydrated(row);
    return mapNameSearchRows(searchRows.value);
  } catch (e) {
    console.warn('[OManyToManyRefTagsField] search failed', e);
    searchRows.value = [];
    return [];
  } finally {
    loading.value = false;
  }
}

/** Kept for mount tests and Backspace/Enter helpers that still call NameSearch directly. */
async function handleRemoteSearch(keyword: string) {
  await relationSearch(keyword, { limit: props.suggestLimit });
}

function onComboboxSelect(opt: RelationOption | null) {
  if (!opt) return;
  if (opt.raw && typeof opt.raw === 'object') upsertHydrated(opt.raw);
  if (!selectedIds.value.includes(opt.id)) {
    onSelectedIdsChange([...selectedIds.value, opt.id]);
  }
  void nextTick(() => {
    addModel.value = null;
  });
}

function onSearchMore(query: string) {
  searchKeyword.value = query ?? '';
  openPicker();
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
    ChoyMessage.warning(_t('searchList is not configured; cannot open picker'));
    return;
  }
  if (!relationStore.value) {
    ChoyMessage.warning(_t('relationStore is unresolved; cannot open picker'));
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

