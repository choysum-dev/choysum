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
    <template #edit="{ inputId, labelId, describedBy, invalid }">
      <span
        v-if="showCellSummary"
        class="truncate text-sm text-foreground"
        data-testid="choy-o2m-summary"
      >{{ cellSummaryText }}</span>
      <ChoyViewScope v-else :view-mode="binding.env.viewMode" :kind="'List'" :field-prefix="String(prop)">
        <div
          class="choy-o2m-list w-full min-w-0 overflow-hidden rounded-md border border-border"
          :id="inputId || undefined"
          tabindex="-1"
          :aria-labelledby="labelId || undefined"
          :aria-describedby="describedBy || undefined"
          :aria-invalid="invalid ? 'true' : undefined"
        >
          <!-- Always mount Host (incl. 0 rows) so nested columns register before Browse. -->
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
            <ChoyTableColumn
              v-if="showHandleColumn"
              type="handle"
              col-key="__handle__"
              :vColumnProps="{ width: 36, align: 'center' }"
            />
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
                  data-testid="choy-o2m-row-remove"
                  @click="onRemove($index)"
                >
                  <Trash2 class="size-3.5" aria-hidden="true" />
                </ChoyButton>
              </template>
            </ChoyTableColumn>
            <template #footer>
              <div class="flex h-full items-center px-3">
                <ChoyButton
                  type="button"
                  size="sm"
                  variant="link"
                  class="h-auto px-0"
                  data-testid="choy-o2m-add-row"
                  @click="handleAddItem"
                >
                  {{ _t('Add a line…') }}
                </ChoyButton>
              </div>
            </template>
          </ChoyTableHost>
        </div>
      </ChoyViewScope>
    </template>

    <template #display>
      <span
        v-if="showCellSummary"
        class="truncate text-sm text-foreground"
        data-testid="choy-o2m-summary"
      >{{ cellSummaryText }}</span>
      <ChoyViewScope v-else view-mode="display" :kind="'List'" :field-prefix="String(prop)">
        <div class="choy-o2m-list w-full min-w-0 overflow-hidden rounded-md border border-border">
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
            <slot />
            <!-- Header-only empty: suppress host default "No data". -->
            <template #empty />
          </ChoyTableHost>
        </div>
      </ChoyViewScope>
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, ClientModel<BaseModel>[]>, V = FieldPathType<T, P>">
import { computed, nextTick, ref, watch, onMounted, provide } from 'vue';
import { Trash2 } from 'lucide-vue-next';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType, ClientModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import ChoyTableHost from '@/web/web/components/internal/ChoyTableHost.vue';
import ChoyTableColumn from '@/web/web/components/table/ChoyTableColumn.vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import ChoyViewScope from '@/web/web/components/view/ChoyViewScope.vue';
import { createTranslate } from '@/web/web/i18n';
import { confirmChoyAction } from '@/web/web/composables/confirmChoyAction';
import { hasHandleField } from '@/web/web/composables/listRowEdit';
import { LIST_HANDLE_API_KEY, useListHandleReorder } from '@/web/web/composables/useListHandleReorder';
import { formatRelationCountSummary } from './relationCountSummary';
import { CHOY_LIST_HEADER_HEIGHT, CHOY_LIST_ROW_HEIGHT } from '@/web/web/composables/listDensity';

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyOneToManyField' });

defineOptions({ name: 'ChoyOneToManyField', inheritAttrs: false });

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
    defaultRecord?: Record<string, any> | (() => Record<string, any>);

    rowHeightEdit?: number;
    rowHeightDisplay?: number;
    headerHeight?: number;
    minTableHeight?: number;
    maxTableHeight?: number;

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;

    strategy?: 'live' | 'idle' | 'blur';
    idleDelay?: number;
    commitOnBlur?: boolean;
    /** Sequence field for drag handle; default Sequence. */
    handleField?: string;
    /** Show handle when relation metadata includes handleField. */
    showHandle?: boolean;
    /** Confirm before removing a row (aligned with O2M Kanban). */
    confirmOnRemove?: boolean;
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

    defaultRecord: () => ({}),
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    strategy: 'live',
    idleDelay: 200,
    commitOnBlur: true,
    handleField: 'Sequence',
    showHandle: true,
    confirmOnRemove: true,
    renderMode: 'auto',
    showInlineError: false,
  }
);

/** Footer strip for Add a line inside the table frame (edit mode). */
const addLineFooterHeight = 36;

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P })) as UseField<T, V>;
binding.registerFields(`${binding.prop}.DisplayName`);

const { getItems, insertItem, removeItemAt } = binding.asMutableArray<any>();
const store = props.store;

const itemCount = computed(() => getItems()?.length ?? 0);
/** Explicit length dep so the embedded table refreshes when the collection is replaced. */
const tableRows = computed(() => {
  const items = getItems() || [];
  void items.length;
  return items;
});

// List/Kanban cells: count summary instead of embedding a full nested table.
const showCellSummary = computed(() => binding.env?.isForm === false);
const cellSummaryText = computed(() => formatRelationCountSummary(itemCount.value, _t));

const showHandleColumn = computed(
  () => props.showHandle !== false && hasHandleField(binding.relationStore, props.handleField)
);

const o2mHandleEnabled = computed(() => showHandleColumn.value && binding.env.isEditMode);

const o2mHandleReorder = useListHandleReorder({
  rows: () => getItems(),
  enabled: o2mHandleEnabled,
  handleField: props.handleField,
  onReorder: rows => {
    (binding.fieldRef() as { value: any }).value = rows.slice();
  },
});

provide('list-editing-row-id', ref(null));
provide(LIST_HANDLE_API_KEY, o2mHandleReorder);

const rowHeightEditRes = computed(() => props.rowHeightEdit!);
const rowHeightDisplayRes = computed(() => props.rowHeightDisplay!);

/**
 * Cap height only when content would exceed maxTableHeight.
 * Returning undefined lets the host size to content (no one-row scrollbar).
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

function hydrateRowKeys() {
  const arr = getItems() || [];
  for (const row of arr) {
    if (!row) continue;
    const seed = readRowKeySeed(row);
    defineHiddenRowKey(row, '__rowKey', seed);
  }
}

function ensureArrayInitialized() {
  const refVal = binding.fieldRef() as any;
  if (!Array.isArray(refVal.value)) {
    refVal.value = [];
  }
}

async function onRemove(i: number) {
  if (!props.confirmOnRemove) {
    removeItemAt(i);
    return;
  }
  try {
    await confirmChoyAction(
      _t('Are you sure you want to delete this record? This action cannot be undone.'),
      _t('Confirm delete'),
      {
        confirmText: _t('Delete'),
        cancelText: _t('Cancel'),
        destructive: true,
      }
    );
    removeItemAt(i);
  } catch {
    /* cancelled */
  }
}

async function handleAddItem() {
  ensureArrayInitialized();
  const targetIndex = getItems().length;
  insertItem(makeDefaultItem());
  await nextTick();
  ovTableRef.value?.scrollToRow?.(targetIndex, 'end');
}

function makeDefaultItem(): Record<string, any> {
  const rec = typeof props.defaultRecord === 'function' ? (props.defaultRecord as any)() : props.defaultRecord;
  const row = { ...(rec || {}) };
  const seed = readRowKeySeed(row) ?? Math.random().toString(36).slice(2);
  defineHiddenRowKey(row, '__rowKey', seed);
  return row;
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
</script>
