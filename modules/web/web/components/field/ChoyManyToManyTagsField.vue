<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    v-bind="$attrs"
    :binding="binding"
    :label="label"
    :rules="rules"
    :formItemProps="formItemProps"
    :orientation="orientation"
    :required="required"
    :readonly="readonly"
    :visible="visible"
    :cellVisible="cellVisible"
    :renderMode="renderMode"
    :showInlineError="showInlineError"
  >
    <template #edit="{ inputId, inputName, describedBy, invalid }">
      <div class="choy-m2m-tags w-full" @keydown="handleKeydown">
        <div
          class="choy-m2m-tags__shell flex min-h-control w-full flex-wrap items-center gap-1 rounded-md border border-input bg-transparent px-2 py-1 shadow-xs"
          :class="[
            controlFocusWithinRing,
            invalid ? 'border-destructive ring-destructive/20' : undefined,
            !relationStore ? 'opacity-50' : undefined,
          ]"
          :style="{ width: width || '100%' }"
          data-testid="choy-m2m-tags-shell"
          @click="onShellClick"
        >
          <template v-for="item in editChipItems" :key="item.id">
            <span class="choy-m2m-tags__chip inline-flex items-center">
              <slot
                name="tag"
                :item="item.record"
                :label="item.label"
                :removable="tagClosable"
                :clickable="false"
              >
                <ChoyBadge variant="secondary" class="choy-m2m-tags__tag gap-1 pr-1 font-normal">
                  {{ item.label }}
                  <button
                    v-if="tagClosable"
                    type="button"
                    class="inline-flex size-3.5 shrink-0 cursor-pointer items-center justify-center appearance-none border-0 bg-transparent p-0 text-inherit opacity-70 hover:opacity-100"
                    :aria-label="_t('Remove')"
                    data-testid="choy-m2m-tag-remove"
                    @click.stop="removeChip(item.id)"
                  >
                    <X class="size-3" />
                  </button>
                </ChoyBadge>
              </slot>
            </span>
          </template>
          <ChoyBadge v-if="hiddenCount > 0" variant="secondary" class="choy-m2m-tags__tag font-normal">
            +{{ hiddenCount }}
          </ChoyBadge>
          <RelationCombobox
            ref="comboboxRef"
            v-model="addModel"
            class="choy-m2m-tags__select min-w-[8rem] flex-1"
            chrome="plain"
            :id="inputId"
            :name="inputName"
            :search="relationSearch"
            :search-key="relationSearchKey"
            :clearable="false"
            :placeholder="effectivePlaceholder"
            :page-size="suggestLimit"
            :disabled="!relationStore"
            :search-more="Boolean(searchList)"
            :aria-describedby="describedBy || undefined"
            :aria-invalid="invalid ? 'true' : undefined"
            @click.stop
            @select="onComboboxSelect"
            @search-more="onSearchMore"
          />
        </div>
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
                <ChoyBadge
                  :variant="isTagClickable ? 'default' : 'secondary'"
                  class="choy-m2m-tags__tag font-normal"
                  :class="{
                    'choy-m2m-tags__tag--primary border-primary bg-primary/10 text-primary group-hover:border-primary group-hover:bg-primary/15':
                      isTagClickable,
                  }"
                >{{ item.label }}</ChoyBadge>
              </slot>
            </span>
          </template>
          <ChoyBadge v-if="hiddenCount > 0" variant="secondary" class="choy-m2m-tags__tag font-normal">
            +{{ hiddenCount }}
          </ChoyBadge>
        </template>
        <slot v-else name="empty" />
      </div>
    </template>
  </ChoyFieldBase>

  <ChoyDialog v-model:open="dialogVisible">
    <ChoyDialogContent
      class="choy-relation-picker-dialog flex max-h-[min(85vh,40rem)] w-[min(90vw,56rem)] max-w-[min(90vw,56rem)] flex-col gap-3 overflow-hidden"
      :style="{ width: typeof searchViewWidth === 'number' ? searchViewWidth + 'px' : searchViewWidth }"
    >
      <ChoyDialogTitle class="shrink-0">{{ effectiveSearchViewTitle }}</ChoyDialogTitle>
      <div class="choy-relation-picker-list min-h-[16rem] flex-1 overflow-hidden">
        <ChoyViewScope view-mode="display" :kind="'List'">
          <component
            v-if="dialogVisible && searchList && relationStore"
            :is="searchList"
            ref="searchViewRef"
            :store="relationStore"
            :show-actions="false"
            :host-page-title="false"
            :register-action-target="false"
            :selection-mode="'multiple'"
            :click-to-select="true"
            :height-mode="'container'"
            :container-selector="'.choy-relation-picker-list'"
            :forced-condition="effectiveConditions"
            :relation-condition-source="dialogRelationConditionSource"
          />
        </ChoyViewScope>
      </div>
      <div class="mt-1 flex shrink-0 justify-end gap-2">
        <ChoyButton type="button" variant="outline" @click="dialogVisible = false">{{ _t('Cancel') }}</ChoyButton>
        <ChoyButton type="button" @click="confirmPicker">{{ _t('OK') }}</ChoyButton>
      </div>
    </ChoyDialogContent>
  </ChoyDialog>
</template>

<style scoped>
  .choy-relation-picker-list :deep([data-anchor='choy.list.table']) {
    overflow-x: auto;
  }
</style>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, ClientModel<BaseModel>[]>, V = FieldPathType<T, P>">
import { computed, ref, shallowRef, type Component, inject, provide, Ref, watch, onBeforeUnmount, getCurrentInstance, nextTick } from 'vue';
import { X } from 'lucide-vue-next';
import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from '@/web/web/components/layout/choyDialog';
import ChoyBadge from '@/web/web/components/layout/ChoyBadge.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { ChoyMessage } from '../../composables/useChoyMessage';
import { RELATION_PICKER_CLOSE_KEY } from '@/web/web/components/field/relationPickerContext';
import type { RuleItem } from 'async-validator';
import type { BaseModel, ClientModel, FieldPath, FieldPathType, QueryCondition } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import { buildRelationConditionSource } from '@/web/web/composables/relationalForField';
import ChoyViewScope from '@/web/web/components/view/ChoyViewScope.vue';
import type { SelectionExpose } from '@/web/web/components/view/listViewTypes';
import { createStoreByModel } from '@/web/web/stores/registry';
import { registerFieldPath, unregisterFieldPath, pathsToFieldSelection, ensureRootId } from '@/web/web/query/utils/registry/field';
import { createTranslate } from '@/web/web/i18n';
import { controlFocusWithinRing } from '@/web/web/lib/utils';
import type { TagClickPayload } from '@/web/web/components/field/manyToManyTagsTypes';
import { shouldShowNameCreateEntry } from '@/web/web/components/field/nameCreateVisibility';
import { runNameCreateQuickCreate, trimSearchKeyword } from '@/web/web/components/field/nameCreateQuickCreate';
import { usePermission } from '@/auth/web/composables/usePermission';
import RelationCombobox from '@/web/web/components/internal/RelationCombobox.vue';
import {
  mapNameSearchRows,
  type RelationOption,
} from '@/web/web/components/internal/relationComboboxHelpers';
import {
  buildM2MEffectiveCondition,
  resetM2MPickerQueryState,
  resolveRelationRecordLabel,
  unwrapListSelectionRecords,
} from './manyToManyRelationHelpers';

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyManyToManyTagsField' });

defineOptions({ name: 'ChoyManyToManyTagsField', inheritAttrs: false });

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
    orientation?: 'horizontal' | 'vertical';

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
    console.warn(`[ChoyManyToManyTagsField] Failed to create store for model '${target}'`, e);
    return undefined;
  }
});

const dialogVisible = ref(false);
provide(RELATION_PICKER_CLOSE_KEY, () => {
  dialogVisible.value = false;
});
const searchViewRef = ref<SelectionExpose<any> | null>(null);
const loading = ref(false);
const hydratedCache = ref<Record<string, any>>({});
const hydratingIds = ref<Set<string>>(new Set());
const searchRows = shallowRef<any[]>([]);
const searchKeyword = ref('');
const dropdownVisible = ref(false);
const addModel = ref<string | null>(null);
const comboboxRef = ref<{ focusInput?: () => void } | null>(null);
const vm = getCurrentInstance();

function focusComboboxInput() {
  comboboxRef.value?.focusInput?.();
}

function onShellClick(event: MouseEvent) {
  const target = event.target as HTMLElement | null;
  if (target?.closest?.('[data-testid="choy-m2m-tag-remove"]')) return;
  focusComboboxInput();
}

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
        appendRecords([row && typeof row === 'object' ? row : { Id: id, DisplayName: id }]);
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
  return resolveRelationRecordLabel(row, fallback, labelFields.value);
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

function createRowRecord(v: any) {
  if (!v || typeof v !== 'object') return v;
  const copy = { ...(v as any) };
  const seed = copy.__rowKey ?? copy.Id ?? Math.random().toString(36).slice(2);
  defineHiddenRowKey(copy, '__rowKey', seed);
  return copy;
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

/** Include selected ids so RelationCombobox does not reuse a page that still lists picks. */
const selectedIdsFingerprint = computed(() =>
  [...selectedIds.value].map(String).sort().join(','),
);
const relationSearchKey = computed(
  () =>
    `${relationStore.value?.storeId || relationStore.value?.fullModelName || ''}:${String(binding.prop)}:${selectedIdsFingerprint.value}`,
);

const recordsById = computed<Record<string, any>>(() => {
  const map: Record<string, any> = { ...hydratedCache.value };
  for (const row of getItems() || []) {
    const id = extractId(row);
    if (!id) continue;
    if (row && typeof row === 'object') map[id] = { ...(map[id] || {}), ...(row as any) };
  }
  return map;
});

const displayItems = computed(() => {
  const ids = selectedIds.value;
  const cap = Number(props.maxTagsVisible || 0);
  const visibleIds = cap > 0 ? ids.slice(0, cap) : ids;
  return visibleIds.map(id => {
    const rec = recordsById.value[id] || { Id: id };
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
  replaceSelectedIds(selectedIds.value.filter(x => x !== id));
}

const lastOnchangeResult = inject<Ref<any | null>>('lastOnchangeResult', ref(null));

const fieldName = computed(() => String(binding.prop));
const onchangeConditions = computed<QueryCondition<any>[]>(() => {
  const raw = lastOnchangeResult.value?.condition || [];
  return raw
    .filter((c: any) => c?.field === fieldName.value)
    .map((c: any) => c?.condition)
    .filter(Boolean);
});

const effectiveConditions = computed<QueryCondition<any> | []>(() =>
  buildM2MEffectiveCondition({
    selectedIds: selectedIds.value,
    external: props.condition as any,
    onchangeParts: onchangeConditions.value,
  })
);

const dialogRelationConditionSource = computed(() => {
  return buildRelationConditionSource(props.store as any, binding.prop).relationConditionSource;
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
    console.warn('[ChoyManyToManyTagsField] hydrate failed', e);
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
    const picked = new Set(selectedIds.value.map(String));
    return mapNameSearchRows(searchRows.value).filter(opt => !picked.has(String(opt.id)));
  } catch (e) {
    console.warn('[ChoyManyToManyTagsField] search failed', e);
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
    appendRecords([opt.raw && typeof opt.raw === 'object' ? opt.raw : { Id: opt.id, DisplayName: opt.label }]);
  }
  void nextTick(() => {
    addModel.value = null;
    focusComboboxInput();
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
      replaceSelectedIds(selectedIds.value.slice(0, -1));
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
  if (!first) return;

  event.preventDefault();
  appendRecords([first]);
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

function recordForId(id: string): any {
  return recordsById.value[id] || hydratedCache.value[id] || { Id: id };
}

function replaceSelectedIds(nextIdsInput: string[]) {
  const prevIds = selectedIds.value;
  let nextIds = Array.from(new Set((nextIdsInput || []).map(x => String(x || '')).filter(Boolean)));
  if (!props.tagClosable) {
    nextIds = Array.from(new Set([...prevIds, ...nextIds]));
  }
  const prev = new Set(prevIds);
  const next = new Set(nextIds);
  const removed = prevIds.some(id => !next.has(id));
  const added = nextIds.some(id => !prev.has(id));
  if (added) searchKeyword.value = '';

  clearItems();
  for (const id of nextIds) {
    insertItem(createRowRecord(recordForId(id)) as any);
  }

  for (const id of nextIds) {
    if (!prev.has(id)) emit('tag-add', { id, item: recordForId(id) });
  }
  for (const id of prevIds) {
    if (!next.has(id)) emit('tag-remove', { id, item: recordForId(id) });
  }

  void ensureHydrated(nextIds);
  if (dropdownVisible.value && removed) {
    void handleRemoteSearch(searchKeyword.value);
  }
}

function appendRecords(rows: any[]) {
  const prevIds = selectedIds.value;
  const prev = new Set(prevIds);
  const nextIds = [...prevIds];
  for (const row of rows || []) {
    const id = extractId(row);
    if (!id || prev.has(id)) continue;
    if (row && typeof row === 'object') upsertHydrated(row);
    prev.add(id);
    nextIds.push(id);
    insertItem(createRowRecord(row && typeof row === 'object' ? row : { Id: id }) as any);
    emit('tag-add', { id, item: recordForId(id) });
  }
  if (nextIds.length > prevIds.length) searchKeyword.value = '';
  void ensureHydrated(nextIds);
}

function openPicker() {
  emit('picker-open');
  // Search-more gated by searchList; missing picker config stays silent (no Toast).
  if (!props.searchList || !relationStore.value) return;
  resetM2MPickerQueryState(relationStore.value as any);
  dialogVisible.value = true;
}

async function confirmPicker() {
  // Snapshot before closing — dialog v-if unmounts the list and drops the expose.
  const expose = searchViewRef.value as any;
  const selected = unwrapListSelectionRecords(expose?.selectedItems).slice();
  try {
    for (const row of selected) upsertHydrated(row);
    if (!selected.length) return;
    appendRecords(selected);
    emit('picker-confirm', { items: selected });
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

