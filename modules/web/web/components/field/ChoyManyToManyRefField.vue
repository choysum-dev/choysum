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
    :hide-label="hideLabel"
    :required="required"
    :readonly="readonly"
    :visible="visible"
    :cellVisible="cellVisible"
    :renderMode="renderMode"
    :showInlineError="showInlineError"
    :label-for-control="false"
  >
    <!-- Edit mode. -->
    <template #edit="{ inputId, labelId, describedBy, invalid }">
      <span
        v-if="showCellSummary"
        class="truncate text-sm text-foreground"
        data-testid="choy-m2m-ref-summary"
      >{{ cellSummaryText }}</span>
      <ChoyViewScope
        v-else
        :view-mode="allowRowEdit ? binding.env.viewMode : 'display'"
        :kind="'List'"
        :field-prefix="String(prop)"
      >
        <div
          class="choy-m2m-list w-full min-w-0 overflow-hidden rounded-md border border-border"
          :id="inputId || undefined"
          tabindex="-1"
          :aria-labelledby="labelId || undefined"
          :aria-describedby="describedBy || undefined"
          :aria-invalid="invalid ? 'true' : undefined"
        >
          <!-- Mount with 0 rows too so nested columns register before Form Browse. -->
          <ChoyTableHost
            ref="ovTableRef"
            :data="tableRows"
            :row-key="'__rowKey'"
            :row-height="rowHeightEditRes"
            :header-height="headerHeight"
            :footer-height="addLineFooterHeight"
            :table-height="tableHeightEdit"
            :store="store"
            :base-index="1"
            :virtualize="'auto'"
            :frameless="true"
          >
            <ChoyTableColumn v-if="showIndex" type="index" label="#" :vColumnProps="{ align: 'end', width: 44 }" />
            <slot :ref-store="relationStore" />
            <ChoyTableColumn :label="_t('Actions')" :width="44" :vColumnProps="{ align: 'center', width: 44 }">
              <template #default="{ $index }">
                <ChoyButton
                  type="button"
                  size="icon"
                  variant="ghost"
                  class="size-7 text-muted-foreground hover:text-destructive"
                  :aria-label="_t('Delete')"
                  data-testid="choy-m2m-ref-row-remove"
                  @click="onRemove($index)"
                >
                  <Trash2 class="size-3.5" aria-hidden="true" />
                </ChoyButton>
              </template>
            </ChoyTableColumn>
            <template #footer>
              <div class="flex h-full items-center px-3">
                <ChoyButton
                  v-if="searchList"
                  type="button"
                  size="sm"
                  variant="link"
                  class="h-auto px-0"
                  data-testid="choy-m2m-ref-add-row"
                  @click="openPicker"
                >
                  {{ _t('Add a line…') }}
                </ChoyButton>
                <span
                  v-else
                  class="text-sm text-muted-foreground"
                  data-testid="choy-m2m-ref-add-unavailable"
                >{{ _t('Picker is not configured') }}</span>
              </div>
            </template>
          </ChoyTableHost>
        </div>
      </ChoyViewScope>
    </template>

    <!-- Display mode. -->
    <template #display>
      <span
        v-if="showCellSummary"
        class="truncate text-sm text-foreground"
        data-testid="choy-m2m-ref-summary"
      >{{ cellSummaryText }}</span>
      <ChoyViewScope v-else view-mode="display" :kind="'List'" :field-prefix="String(prop)">
        <!-- Always mount Host (incl. 0 rows) so nested columns register before Browse. -->
        <div class="choy-m2m-list w-full min-w-0 overflow-hidden rounded-md border border-border">
          <ChoyTableHost
            :data="tableRows"
            :row-key="'__rowKey'"
            :row-height="rowHeightDisplayRes"
            :header-height="headerHeight"
            :table-height="tableHeightDisplay"
            :store="store"
            :base-index="1"
            :virtualize="'auto'"
            :frameless="true"
          >
            <ChoyTableColumn v-if="showIndex" type="index" label="#" :vColumnProps="{ align: 'end', width: 44 }" />
            <slot :ref-store="relationStore" />
            <!-- Header-only empty: suppress host default "No data". -->
            <template #empty />
          </ChoyTableHost>
        </div>
      </ChoyViewScope>
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
        <ChoyButton type="button" @click="confirmAdd">{{ _t('OK') }}</ChoyButton>
      </div>
    </ChoyDialogContent>
  </ChoyDialog>
</template>

<style scoped>
  .choy-relation-picker-list :deep([data-anchor='choy.list.table']) {
    overflow-x: auto;
  }
</style>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string[]>, V = FieldPathType<T, P>">
import {
  computed,
  ref,
  type Component,
  nextTick,
  watch,
  onMounted,
  onBeforeUnmount,
  inject,
  provide,
  type Ref,
} from 'vue';
import { Trash2 } from 'lucide-vue-next';
import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from '@/web/web/components/layout/choyDialog';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { RELATION_PICKER_CLOSE_KEY } from '@/web/web/components/field/relationPickerContext';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType, QueryCondition } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import ChoyTableHost from '@/web/web/components/internal/ChoyTableHost.vue';
import ChoyTableColumn from '@/web/web/components/table/ChoyTableColumn.vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import ChoyViewScope from '@/web/web/components/view/ChoyViewScope.vue';
import type { SelectionExpose } from '@/web/web/components/view/listViewTypes';
import { createStoreByModel } from '@/web/web/stores/registry';
import { createTranslate } from '@/web/web/i18n';
import { formatRelationCountSummary } from './relationCountSummary';
import { buildRelationConditionSource } from '@/web/web/composables/relationalForField';
import { CHOY_LIST_HEADER_HEIGHT, CHOY_LIST_ROW_HEIGHT } from '@/web/web/composables/listDensity';
import {
  exportFieldSelection,
  registerFieldPath,
  unregisterFieldPath,
  pathsToFieldSelection,
  ensureRootId,
} from '@/web/web/query/utils/registry/field';
import {
  buildM2MEffectiveCondition,
  resetM2MPickerQueryState,
  unwrapListSelectionRecords,
} from './manyToManyRelationHelpers';

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyManyToManyRefField' });

defineOptions({ name: 'ChoyManyToManyRefField', inheritAttrs: false });

type IsAny<T> = 0 extends 1 & T ? true : false;

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];
    formItemProps?: Partial<FormItemProps>;
    /** Subtable default: label above the list (full width). */
    orientation?: 'horizontal' | 'vertical';
    /** Hide label row when Section/Tab already titles the block. Prefer over `label=""`. */
    hideLabel?: boolean;

    showIndex?: boolean;
    rowHeightEdit?: number;
    rowHeightDisplay?: number;
    headerHeight?: number;
    minTableHeight?: number;
    maxTableHeight?: number;

    allowRowEdit?: boolean;

    searchList?: Component;
    searchViewTitle?: string;
    searchViewWidth?: string | number;
    targetModel?: string;

    strategy?: 'live' | 'idle' | 'blur';
    idleDelay?: number;
    commitOnBlur?: boolean;

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;

    condition?: QueryCondition<any> | QueryCondition<any>[];
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    rules: () => [],
    formItemProps: () => ({}),
    orientation: 'vertical',
    hideLabel: false,
    showIndex: true,
    rowHeightEdit: CHOY_LIST_ROW_HEIGHT,
    rowHeightDisplay: CHOY_LIST_ROW_HEIGHT,
    headerHeight: CHOY_LIST_HEADER_HEIGHT,
    /** Floor only when set; default 0 = height follows row count (no empty padding). */
    minTableHeight: 0,
    maxTableHeight: 360,
    allowRowEdit: false,
    searchViewTitle: '',
    searchViewWidth: '75%',
    targetModel: '',

    strategy: 'live',
    idleDelay: 200,
    commitOnBlur: true,

    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,

    condition: undefined,
    renderMode: 'auto',
    showInlineError: false,
  }
);

const effectiveSearchViewTitle = computed(() => props.searchViewTitle || _t('Select related items'));
/** Footer strip for Add a line inside the table frame (edit mode). */
const addLineFooterHeight = 36;

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P })) as UseField<T, V>;

const relationStore = computed<WebModelStore<any> | undefined>(() => {
  if (binding.relationStore) return binding.relationStore as WebModelStore<any>;
  const target = props.targetModel || binding.meta?.relationModel;
  if (!target) return undefined;
  try {
    return createStoreByModel(target);
  } catch (e) {
    console.warn(`[ChoyManyToManyRefField] Failed to create store for model '${target}'`, e);
    return undefined;
  }
});

// Nested slot columns register on the relation store (field value is string[], not ClientModel[]).
provide('suppress-field-registration', true);
provide('alt-field-registration-store', relationStore);

const { getItems, insertItem, removeItemAt } = binding.asMutableArray();

/** Cache hydrated target records keyed by Id (field value is string[]). */
const hydratedCache = ref<Record<string, any>>({});
const hydratingIds = ref<Set<string>>(new Set());

function extractId(v: any): string | undefined {
  if (v == null) return undefined;
  if (typeof v === 'object') return (v as any).Id != null ? String((v as any).Id) : undefined;
  return String(v);
}

const selectedIds = computed(() =>
  (getItems() || []).map(extractId).filter(Boolean).map(String)
);

const itemCount = computed(() => selectedIds.value.length);

const showCellSummary = computed(() => binding.env?.isForm === false);
const cellSummaryText = computed(() => formatRelationCountSummary(itemCount.value, _t));

const rowHeightEditRes = computed(() => props.rowHeightEdit!);
const rowHeightDisplayRes = computed(() => props.rowHeightDisplay!);

/**
 * Cap height only when content would exceed maxTableHeight.
 * Returning undefined lets the host size to content (no one-row scrollbar).
 * Optional minTableHeight pads short tables when set.
 */
function computeTableHeight(
  rowCount: number,
  rowHeight: number,
  headerHeight: number,
  footerHeight = 0,
): number | undefined {
  const rows = Math.max(0, rowCount);
  const rh = rowHeight || CHOY_LIST_ROW_HEIGHT;
  const hh = headerHeight || CHOY_LIST_HEADER_HEIGHT;
  const fh = Math.max(0, footerHeight);
  const total = hh + rows * rh + fh;
  const floor = Number(props.minTableHeight || 0);
  const max = props.maxTableHeight!;
  if (floor > 0 && rows > 0) return Math.max(floor, Math.min(max, total));
  if (total <= max) return undefined;
  return max;
}

const tableHeightEdit = computed(() =>
  computeTableHeight(itemCount.value, rowHeightEditRes.value, props.headerHeight!, addLineFooterHeight)
);
const tableHeightDisplay = computed(() =>
  computeTableHeight(itemCount.value, rowHeightDisplayRes.value, props.headerHeight!)
);

const dialogVisible = ref(false);
provide(RELATION_PICKER_CLOSE_KEY, () => {
  dialogVisible.value = false;
});
const searchViewRef = ref<SelectionExpose<any> | null>(null);
const ovTableRef = ref<InstanceType<typeof ChoyTableHost> | null>(null);

function readRowKeySeed(row: unknown): string | number | undefined {
  if (!row || typeof row !== 'object') return undefined;
  const r = row as Record<string, any>;
  return r.__rowKey ?? r.Id;
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

/** Build display rows from string ids (or legacy object rows) via hydrate cache. No mutations here. */
const tableRows = computed(() => {
  const items = getItems() || [];
  void items.length;
  const cache = hydratedCache.value;
  return items.map((item, idx) => {
    if (item && typeof item === 'object') {
      return item as any;
    }
    const id = String(item ?? '');
    const cached = cache[id];
    if (cached) return cached;
    return { Id: id, __rowKey: String(id || idx) } as any;
  });
});

function hydrateRowKeys() {
  for (const row of tableRows.value) {
    if (!row || typeof row !== 'object') continue;
    const seed = readRowKeySeed(row);
    defineHiddenRowKey(row, '__rowKey', seed);
  }
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

/**
 * Hydration / NameSearch field set.
 * Prefer parent-store selections under the prop prefix (nested columns); else Id + DisplayName.
 * Do not read relation-store exportFieldSelection here — registering those fields bumps
 * registryVersion and would feedback into this computed.
 */
const remoteFields = computed<string[]>(() => {
  const base = ['DisplayName'];
  const storeId = binding.store?.storeId;
  if (storeId) {
    const prefix = `${binding.prop}.`;
    const selection = exportFieldSelection(storeId) || [];
    const picked = selection
      .filter(p => p.startsWith(prefix))
      .map(p => p.slice(prefix.length))
      .filter(Boolean);
    if (picked.length) {
      return Array.from(new Set(['Id', ...picked, ...base]));
    }
  }
  return Array.from(new Set(['Id', ...base]));
});

function pickHydrationFields() {
  const own = remoteFields.value && remoteFields.value.length ? remoteFields.value : [];
  return ensureRootId(pathsToFieldSelection(own) ?? own) || [];
}

async function ensureHydrated(ids: string[]) {
  const store = relationStore.value;
  if (!store) return;
  const fields = pickHydrationFields();
  const missing = ids.filter(id => !hydratedCache.value[id] && !hydratingIds.value.has(id));
  if (!missing.length) return;

  for (const id of missing) hydratingIds.value.add(id);
  try {
    const records = await store.Search(['Id', 'in', missing] as any, { fields } as any);
    const arr = Array.isArray(records) ? records : [];
    const next = { ...hydratedCache.value };
    for (const rec of arr) {
      const row = rec ? { ...(rec as any) } : null;
      if (!row) continue;
      const id = String(row.Id ?? '');
      defineHiddenRowKey(row, '__rowKey', readRowKeySeed(row) ?? id);
      next[id] = row;
    }
    hydratedCache.value = next;
  } catch (e) {
    console.warn('[ChoyManyToManyRefField] hydrate failed', e);
  } finally {
    for (const id of missing) hydratingIds.value.delete(id);
  }
}

watch(
  selectedIds,
  ids => {
    void ensureHydrated(ids);
  },
  { immediate: true }
);

const registeredRefFields = ref<Set<string>>(new Set());
let registeredStoreId: string | null = null;

function syncRemoteFieldRegistration(store?: WebModelStore<any> | null, fields?: string[]) {
  const nextStoreId = store?.storeId ?? null;
  const nextSet = new Set(fields || []);
  const unchanged =
    nextStoreId === registeredStoreId &&
    nextSet.size === registeredRefFields.value.size &&
    Array.from(nextSet).every(f => registeredRefFields.value.has(f));
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

function openPicker() {
  // Footer already shows "Picker is not configured"; stay silent (no Toast).
  if (!props.searchList || !relationStore.value) return;
  resetM2MPickerQueryState(relationStore.value as any);
  dialogVisible.value = true;
}

function createRowKey(v: any) {
  if (!v) return v;
  const seed = readRowKeySeed(v) ?? Math.random().toString(36).slice(2);
  defineHiddenRowKey(v, '__rowKey', seed);
  return v;
}

function onRemove(i: number) {
  removeItemAt(i);
}

async function confirmAdd() {
  // Snapshot selection before closing the dialog (v-if tears down the list).
  const expose = searchViewRef.value as any;
  const selected = unwrapListSelectionRecords(expose?.selectedItems).slice();
  try {
    const ids = selected.map(x => extractId(x)).filter(Boolean).map(String);
    if (!ids.length) return;
    const existIds = new Set(selectedIds.value);
    const newIds = ids.filter(id => !existIds.has(id));
    if (!newIds.length) return;
    const records = selected.filter(x => newIds.includes(String(extractId(x) || '')));
    for (const rec of records) {
      const row = createRowKey({ ...(rec || {}) });
      const id = extractId(row);
      if (!id) continue;
      hydratedCache.value = { ...hydratedCache.value, [id]: row };
      // Keep field value as string[] (ManyToManyRef); display uses hydrate cache.
      insertItem(id as any);
    }
  } finally {
    dialogVisible.value = false;
    await nextTick();
    ovTableRef.value?.scrollToRow?.(getItems().length - 1, 'end');
  }
}

onMounted(() => {
  hydrateRowKeys();
});
watch(
  () => getItems().length,
  () => {
    hydrateRowKeys();
  },
);

const store = props.store;

defineSlots<{
  default(props: { refStore: WebModelStore<any> | undefined }): any;
}>();
</script>
