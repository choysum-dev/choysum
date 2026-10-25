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
    <template #edit="{ inputId, labelId, describedBy }">
      <span
        v-if="showCellSummary"
        class="truncate text-sm text-foreground"
        data-testid="choy-m2m-summary"
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
            :virtualize="false"
            :frameless="true"
          >
            <ChoyTableColumn v-if="showIndex" type="index" label="#" :vColumnProps="{ align: 'end', width: 44 }" />
            <slot />
            <ChoyTableColumn :label="_t('Actions')" :width="44" :vColumnProps="{ align: 'center', width: 44 }">
              <template #default="{ $index }">
                <ChoyButton
                  type="button"
                  size="icon"
                  variant="ghost"
                  class="size-7 text-muted-foreground hover:text-destructive"
                  :aria-label="_t('Delete')"
                  data-testid="choy-m2m-row-remove"
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
                  data-testid="choy-m2m-add-row"
                  @click="openPicker"
                >
                  {{ _t('Add a line…') }}
                </ChoyButton>
                <span
                  v-else
                  class="text-sm text-muted-foreground"
                  data-testid="choy-m2m-add-unavailable"
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
        data-testid="choy-m2m-summary"
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
            :virtualize="false"
            :frameless="true"
          >
            <ChoyTableColumn v-if="showIndex" type="index" label="#" :vColumnProps="{ align: 'end', width: 44 }" />
            <slot />
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

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, ClientModel<BaseModel>[]>, V = FieldPathType<T, P>">
import { computed, ref, type Component, nextTick, watch, onMounted, inject, provide, Ref } from 'vue';
import { Trash2 } from 'lucide-vue-next';
import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from '@/web/web/components/layout/choyDialog';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { ChoyMessage } from '../../composables/useChoyMessage';
import { RELATION_PICKER_CLOSE_KEY } from '@/web/web/components/field/relationPickerContext';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType, ClientModel, QueryCondition } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import ChoyTableHost from '@/web/web/components/internal/ChoyTableHost.vue';
import ChoyTableColumn from '@/web/web/components/table/ChoyTableColumn.vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import ChoyViewScope from '@/web/web/components/view/ChoyViewScope.vue';
import type { SelectionExpose } from '@/web/web/components/view/listViewTypes';
import { createStoreByModel } from '@/web/web/stores/registry';
import { useProvidedOnchange } from '@/web/web/composables/useOnchange';
import { createTranslate } from '@/web/web/i18n';
import { formatRelationCountSummary } from './relationCountSummary';
import { buildRelationConditionSource } from '@/web/web/composables/relationalForField';
import { CHOY_LIST_HEADER_HEIGHT, CHOY_LIST_ROW_HEIGHT } from '@/web/web/composables/listDensity';
import {
  buildM2MEffectiveCondition,
  resetM2MPickerQueryState,
  unwrapListSelectionRecords,
} from './manyToManyRelationHelpers';

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyManyToManyField' });

defineOptions({ name: 'ChoyManyToManyField', inheritAttrs: false });

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
    rowHeightEdit?: number; // Row height in edit mode.
    rowHeightDisplay?: number; // Row height in display mode.
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

    /* External static constraints merged with onchange conditions. */
    condition?: QueryCondition<any> | QueryCondition<any>[];
    // Added render mode and inline error support.
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

// Field binding.
const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P })) as UseField<T, V>;
binding.registerFields(`${binding.prop}.DisplayName`);

const relationStore = computed<WebModelStore<any> | undefined>(() => {
  if (binding.relationStore) return binding.relationStore as WebModelStore<any>;
  const target = props.targetModel || binding.meta?.relationModel;
  if (!target) return undefined;
  try {
    return createStoreByModel(target);
  } catch (e) {
    console.warn(`[ChoyManyToManyField] Failed to create store for model '${target}'`, e);
    return undefined;
  }
});
const { getItems, insertItem, removeItemAt } = binding.asMutableArray();

const itemCount = computed(() => (getItems()?.length ?? 0));
/** Explicit length dep so the embedded table refreshes when the collection is replaced. */
const tableRows = computed(() => {
  const items = getItems() || [];
  void items.length;
  return items;
});

// List/Kanban cells: count summary instead of embedding a full nested table.
const showCellSummary = computed(() => binding.env?.isForm === false);
const cellSummaryText = computed(() => formatRelationCountSummary(itemCount.value, _t));

// Row height for edit and display modes.
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

// Read the row-key seed.
function readRowKeySeed(row: unknown): string | number | undefined {
  if (!row || typeof row !== 'object') return undefined;
  const r = row as Record<string, any>;
  return r.__rowKey ?? r.Id;
}

// Define a non-enumerable __rowKey.
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

// Ensure existing and future rows all have a hidden __rowKey.
function hydrateRowKeys() {
  const arr = getItems() || [];
  for (const row of arr) {
    if (!row) continue;
    const seed = readRowKeySeed(row);
    defineHiddenRowKey(row, '__rowKey', seed);
  }
}

// Use the cumulative onchange result injected by FormView.
const lastOnchangeResult = inject<Ref<any | null>>('lastOnchangeResult', ref(null));

const fieldName = computed(() => String(binding.prop));
const onchangeConditions = computed<QueryCondition<any>[]>(() => {
  const raw = lastOnchangeResult.value?.condition || [];
  return raw
    .filter((c: any) => c?.field === fieldName.value)
    .map((c: any) => c?.condition)
    .filter(Boolean);
});

const selectedIds = computed(() =>
  (getItems() || [])
    .map((x: any) => x?.Id)
    .filter(Boolean)
    .map(String)
);

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

function openPicker() {
  if (!props.searchList) {
    ChoyMessage.warning(_t('searchList is not configured; cannot open picker'));
    return;
  }
  if (!relationStore.value) {
    ChoyMessage.warning(_t('relationStore is unresolved; cannot open picker'));
    return;
  }
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
    const ids = selected.map(x => x?.Id).filter(Boolean);
    if (!ids.length) return;
    const existIds = new Set(selectedIds.value);
    const newIds = ids.map(String).filter((id: any) => !existIds.has(id));
    if (!newIds.length) return;
    const records = selected.filter(x => newIds.includes(String(x?.Id)));
    for (const rec of records) insertItem(createRowKey({ ...(rec || {}) }) as any);
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

// Expose store for the template.
const store = props.store;
</script>

