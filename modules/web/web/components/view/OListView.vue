<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OViewContainer :showHeader="showHeader">
    <template #header>
      <div class="o-list__action-bar">
        <div class="o-list__actions">
          <div class="o-list__system-actions" v-if="showActions">
            <slot name="system-actions" :selected-items="selectedItems">
              <el-button v-if="createAction && canCreate" size="small" plain type="primary" @click="handleCreate">
                <el-icon><Plus /></el-icon>
                新建
              </el-button>

              <el-button v-if="refreshAction && canRefresh" size="small" plain @click="handleRefresh">
                <el-icon><Refresh /></el-icon>
                刷新
              </el-button>

              <el-button
                v-if="deleteAction && canDelete"
                size="small"
                plain
                type="danger"
                :disabled="selectedItems.length === 0"
                :loading="deleteLoading"
                @click="handleDelete"
              >
                <el-icon><Delete /></el-icon>
                删除 ({{ selectedItems.length }})
              </el-button>
            </slot>
          </div>

          <div class="o-list__user-actions" v-if="showActions">
            <slot name="user-actions" :selected-items="selectedItems" />
          </div>
        </div>

        <!-- Centered search: render only when searchView is provided -->
        <div class="o-list__search" v-if="searchView">
          <component :is="searchView" :store="store" @query-update="onSearch" />
        </div>

        <div class="o-list__header-right">
          <div class="o-list__default-pagination" v-if="showPaginate">
            <OPagination
              :store="store"
              :total="effectiveTotal"
              :limit="effectivePagination.limit"
              :offset="effectivePagination.offset"
              @paginateState="onPaginateState"
            />
          </div>
          <slot name="header-right" />
        </div>
      </div>
    </template>

    <div class="o-list__table" ref="tableWrapRef" :style="{ height: tablePxHeight }">
      <OVTable
        :data="items"
        :row-key="computedRowKey"
        :row-height="effectiveRowHeight"
        :header-height="headerHeight"
        :table-height="tableHeight"
        :selection-api="selection"
        :base-index="baseIndex"
        :store="store"
        @selection-change="onSelectionChange"
        @row-click="onRowClick"
        @sort-change="onTableSortChange"
      >
        <!-- Automatically inject the leading group column in grouped mode -->
        <OVColumn v-if="isGroupMode" col-key="__group_label" :sortable="false">
          <template #default="{ row }">
            <div v-if="row?.kind === 'group'" class="o-group-cell" :style="{ paddingLeft: `${row.depth * 16}px` }">
              <span class="o-group-cell__caret" :class="{ expanded: isExpanded(row.key) }" @click.stop="onToggleGroup(row.key)" />
              <span class="o-group-cell__label">{{ row.label }}</span>
              <span class="o-group-cell__count">({{ row.count ?? 0 }})</span>
            </div>
            <div v-else-if="row?.kind === 'more'" class="o-more-cell">点击加载更多(剩余 {{ Math.max(0, Number(row.remain ?? 0)) }})</div>
            <span v-else></span>
          </template>
        </OVColumn>
        <slot />
        <template #empty>
          <slot name="empty">
            <div class="ovtable__empty">暂无数据</div>
          </slot>
        </template>
      </OVTable>
    </div>
  </OViewContainer>
</template>

<script setup lang="ts" generic="T extends BaseModel">
import type { ConditionGroup, QueryUpdatePayload } from '@/web/web/query/types';
import type { RowEventHandlerParams } from 'element-plus';
import { computed, onMounted, onBeforeUnmount, provide, ref, nextTick, watch, DefineComponent } from 'vue';
import type { ComputedRef, Ref } from 'vue';
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { ClientModel, BaseModel, QueryCondition, OrderBy } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ElMessage, ElMessageBox } from 'element-plus';
import OVTable from '@/web/web/components/vtable/OVTable.vue';
import OPagination from './OPagination.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import { Plus, Refresh, Delete } from '@element-plus/icons-vue';
import { useVTableSelection } from '@/web/web/composables/useVTable';
// Search view type: must accept store and emit query-update
import type { Component } from 'vue';
import type { SearchViewComponent } from '@/web/web/query/types';
import { provideOnchange } from '@/web/web/composables/useOnchange';
import type { ViewMode, ViewContainer } from '@/web/web/components/view/OViewScope.vue';
// Controller: unified loading with grouping-first handling
import { createListController } from '@/web/web/controllers/listController';
import type { OrderByState, PaginationState } from '@/web/web/query/state';
import { useCancelableEmit } from '@/web/web/composables/useCancelableEmit';
import { useAdaptiveHeight } from '@/web/web/composables/useAdaptiveHeight';
import OViewContainer from '@/web/web/components/view/OViewContainer.vue';
import { useVirtualizationAdapter } from '@/web/web/composables/virtualizationAdapter';
import { awaitFieldSelection } from '@/web/web/query/utils/registry/fieldReady';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { canShowAction, type ActionIdMap } from '@/web/web/components/view/actionVisibility';

export interface SelectionExpose<T = any> {
  selectedItems: Ref<T[]>;
  selectedItem: ComputedRef<T | null>;
}

export type RowEventPayload<T> = {
  row: ClientModel<T>;
  rowIndex: number;
  rowKey: RowEventHandlerParams['rowKey'];
  event: MouseEvent | Event;
};

const props = withDefaults(
  defineProps<{
    store: WebModelStore<T>;
    keywordFields?: string[];
    showHeader?: boolean;
    showActions?: boolean;
    /* Render search only when a searchView component is provided */
    rowHeight?: number;
    headerHeight?: number;
    rowKey?: string;
    showPagination?: boolean;
    viewportGap?: number;
    createAction?: string | RouteLocationRaw;
    refreshAction?: boolean;
    deleteAction?: boolean;
    actionIds?: ActionIdMap;
    hasAction?: (actionId: string | undefined) => boolean;
    selectionMode?: 'multiple' | 'single';
    clickToSelect?: boolean;
    heightMode?: 'auto' | 'viewport' | 'container';
    containerSelector?: string;

    orderBy?: OrderBy<T> | Array<OrderBy<T>>;

    /* Inject a custom search view that must provide query-update */
    searchView?: SearchViewComponent<T> | typeof OSearchView;

    /* Added: control OPagination visibility on the right side of the header */
    showPaginate?: boolean;

    /* Controlled forced condition: always AND with user search conditions, or act as the only condition; dynamic changes trigger re-apply */
    forcedCondition?: QueryCondition<T> | QueryCondition<T>[];
  }>(),
  {
    showHeader: true,
    showActions: true,
    /* Removed: showHeaderRight: true, */
    rowHeight: 40,
    rowKey: 'Id',
    showPagination: true,
    viewportGap: 78,
    refreshAction: true,
    deleteAction: true,
    selectionMode: 'multiple',
    clickToSelect: false,
    heightMode: 'auto',
    containerSelector: '',
    orderBy: undefined,

    /* Added defaults */
    showPaginate: true,

    forcedCondition: undefined,
  }
);

const emit = defineEmits<{
  (e: 'before-load', payload: { query: QueryCondition<T>; page: number; pageSize: number; confirm: () => void; cancel: () => void }): void;
  (e: 'before-refresh', payload: { confirm: () => void; cancel: () => void }): void;
  (e: 'before-delete', payload: { ids: string[]; confirm: () => void; cancel: () => void }): void;

  (e: 'load-success', payload: { items: ClientModel<T>[]; total: number }): void;
  (e: 'refresh-success', payload: { items: ClientModel<T>[]; total: number }): void;
  (e: 'delete-success', payload: { ids: string[] }): void;
  (e: 'selection-change', payload: { rows: ClientModel<T>[]; ids: string[] }): void;
  (e: 'row-click', payload: RowEventPayload<T>): void;
  (e: 'search-change'): void;
  (e: 'paginate', payload: { page: number; pageSize: number }): void;
  (e: 'sort-change', payload: { orderBy: OrderBy<T> | Array<OrderBy<T>> | undefined }): void;

  (e: 'action-error', payload: { action: 'load' | 'delete' | 'refresh' | 'search' | 'sort' | 'paginate'; error: Error }): void;
}>();
const { emitCancelable } = useCancelableEmit(emit as any);

// =============================
// Section 5: View context provisioning
// =============================
// Identify the container type
const viewContainer = ref<ViewContainer>('List');
provide('view-container', viewContainer);

// Future inline editing can use this as a switchable view mode; keep display for now
const listViewMode = ref<ViewMode>('display');
provide('view-mode', listViewMode);

const router = useRouter();
const store = props.store; // WebModelStore<T>
const canCreate = computed(() => canShowAction(props.actionIds?.create, props.hasAction));
const canRefresh = computed(() => canShowAction(props.actionIds?.refresh, props.hasAction));
const canDelete = computed(() => canShowAction(props.actionIds?.delete, props.hasAction));
// List controller
const controller = createListController(store as any);

// Provide a list-level onchange controller for future inline editing
provideOnchange(store, 'ListView');

// Default filters and groups are injected by the outer layer when present; this view only forwards searchView

// =============================
// Section 6: Row key strategy & mode detection
// =============================
// Note: row keys always use the wrapped key field; props.rowKey no longer participates directly

// Whether grouped mode is active, based on the controller result
const isGroupMode = computed(() => controller.vm.result?.kind === 'group');

// Always use the unified key field from DataSetSnapshot RecordRow/GroupRow, aligned with controller output
// If a legacy recordList fallback exists in non-grouped mode, supplement it with a key field
const computedRowKey = computed(() => 'key');

// =============================
// Section 7: Data source selection (visible nodes)
// =============================
// The view no longer handles legacy wrappers and relies on controller visible nodes
const items = computed<any[]>(() => controller.vm.visibleNodes || []);

// =============================
// Section 8: Selection management
// =============================
// Prefer the new wrapped key first while remaining compatible with legacy __rowKey/Id/id
const selection = useVTableSelection(
  (row: any) =>
    row?.key ?? row?.__rowKey ?? row?.Id ?? row?.id ?? (typeof row === 'object' ? ((row as any)?.payload?.Id ?? (row as any)?.payload?.id) : undefined),
  props.selectionMode
);

// Selected items shown in the header bar
const selectedItems = ref<any[]>([]);
function onSelectionChange(rows: any[]) {
  // In group-tree mode, keep only the actual record Ids of detail rows
  const ids = (rows || [])
    .map((r: any) => {
      if (r?.type === 'record') return r?.record?.Id ?? r?.record?.id;
      if (r?.kind === 'record') return r?.payload?.Id ?? r?.payload?.id;
      const rec = r?.payload ?? r?.record ?? r;
      return rec?.Id ?? rec?.id;
    })
    .filter((x: any) => x != null) as string[];
  selectedItems.value = rows as ClientModel<T[]>;
  emit('selection-change', { rows: rows as any, ids });
}

// Convenience accessor for single selection; unwrap record rows to their record
const selectedItem = computed<ClientModel<T> | null>(() => {
  const arr = selectedItems.value as unknown as any[];
  if (!arr || arr.length === 0) return null;
  const r = arr[0];
  return (r?.type === 'record' ? r?.record : r?.kind === 'record' ? r?.payload : r) || null;
});

// Unified first-frame flag: run the initial apply only once, from either the search component or the view itself
const firstApplied = ref(false);

// =============================
// Section 9: Effective pagination & total
// =============================
// Effective pagination based on whether the result is search or group shaped
const effectivePagination = computed<PaginationState>(() => {
  const qs: any = (store.state as any)?.queryState;
  return qs && qs.pagination ? (qs.pagination as PaginationState) : { limit: 20, offset: 0 };
});

const effectiveTotal = computed<number>(() => {
  const t = Number(((store.state as any).result?.total ?? controller.vm.result?.total ?? 0) as any);
  return Number.isFinite(t) ? t : 0;
});

// OSearchView owns controlled search display and event forwarding; the view no longer passes controlled filters or groups

// =============================
// Section 11: Base index (1-based row numbering)
// =============================
// Base row index derived from offset
const baseIndex = computed(() => (effectivePagination.value.offset ?? 0) + 1);

// =============================
// Section 12: Side-effects on data change (reset selection)
// =============================
// Clear selection after page changes or data refreshes
watch(items, () => {
  selection.clear();
  selectedItems.value = [];
});

// =============================
// Section 13: Adaptive height composable integration
// =============================
const tableWrapRef = ref<HTMLElement | null>(null);
const {
  height: tableHeight,
  pxHeight: tablePxHeight,
  recompute: recomputeTableHeight,
} = useAdaptiveHeight(tableWrapRef as Ref<Element | null, any>, {
  mode: props.heightMode,
  containerSelector: props.containerSelector,
  viewportGap: props.viewportGap,
  minContainerHeight: 160,
  minViewportHeight: 240,
  containerPadding: 8,
});

// =============================
// Section 14: Virtualization adapter (row height tuning)
// =============================
// Future rowHeight and overscan tuning can be driven here by column config or density
const virtualization = useVirtualizationAdapter({ rowHeight: props.rowHeight });
const effectiveRowHeight = computed(() => virtualization.config.value.rowHeight || props.rowHeight || 40);
watch(
  () => props.rowHeight,
  v => {
    if (v && v !== virtualization.config.value.rowHeight) {
      virtualization.config.value.rowHeight = v;
    }
  }
);

// Utility: recompute height on next tick after layout changes
function afterLayoutRecompute() {
  return nextTick().then(recomputeTableHeight);
}

// =============================
// Section 15: Lifecycle - onMounted (initial load)
// =============================
onMounted(async () => {
  await nextTick();

  // Sync external sorting from controlled props
  if (props.orderBy !== undefined) {
    (store.state as any).orderBy = props.orderBy as any;
  }

  // Initial height sync
  recomputeTableHeight();

  // Wait for field registration; if the search component already triggered the initial apply, do not repeat it
  await awaitFieldSelection(store, { requireNonEmpty: true });
  if (!firstApplied.value) {
    firstApplied.value = true;
    await controller.apply({
      // Single entry point: pass only forcedCondition and let the controller merge the rest
      forcedCondition: props.forcedCondition as any,
    });
  }
  await afterLayoutRecompute();
});

// =============================
// Section 16: Lifecycle - onBeforeUnmount cleanup
// =============================
onBeforeUnmount(() => {
  /* useAdaptiveHeight already handles cleanup */
  window.removeEventListener('visibilitychange', visibilityHandler);
  window.removeEventListener('pageshow', pageShowHandler);
});

// =============================
// Section 17: External imperative load API
// =============================
// Load data through the controller
async function loadData() {
  selection.clear();
  selectedItems.value = [];
  try {
    const totalBefore = Number(((store.state as any).result?.total ?? 0) as any);
    await controller.apply();
    const totalAfter = Number(((store.state as any).result?.total ?? 0) as any);
    emit('load-success', { items: (items.value as any) || [], total: totalAfter || totalBefore || 0 });
    afterLayoutRecompute();
  } catch (e: any) {
    const err = e instanceof Error ? e : new Error(String(e));
    emit('action-error', { action: 'load', error: err });
    ElMessage.error('列表加载失败');
  }
}

// Sync props.orderBy to state.orderBy
watch(
  () => props.orderBy,
  v => {
    (store.state as any).orderBy = v as any;
  }
);

// Unified note: resetToFirstPageOrLoad was removed because it was unused; all refresh logic now goes through controller.apply()

// =============================
// Section 18: Search handler
// =============================
// Accept a structured payload and hand it off to controller.apply
function onSearch(payload: QueryUpdatePayload<T>) {
  emit('search-change');
  // Store the latest search context for forcedCondition-driven refreshes
  lastSearchPayload.value = payload;
  if (payload) {
    // debug log removed
    // Ensure registered fields participate in the first query
    // When OSearch fires onMounted on the first frame, table columns and fields may not have finished registering yet
    // Proactively wait for one registration cycle, up to a few nextTick turns
    if (!firstApplied.value) firstApplied.value = true;
    awaitFieldSelection(store, { requireNonEmpty: true }).then(() => {
      controller
        .apply({
          // The view layer passes only the external forced condition; the UI tag tree can use payload.appliedFilters directly
          forcedCondition: props.forcedCondition as any,
          appliedFilters: payload.appliedFilters as any,
          keyword: payload.keyword,
          keywordFields: props.keywordFields || undefined,
          appliedGroups: payload.appliedGroups as any,
        })
        .then(afterLayoutRecompute);
    });
  }
}

// =============================
// Section 19: Pagination handler
// =============================
// OPagination only updates state and emits events
function onPaginateState({ limit, offset }: { limit: number; offset: number }) {
  // Still emit paginate externally for legacy listeners
  const page = Math.floor(offset / Math.max(1, limit)) + 1;
  const pageSize = limit;
  emit('paginate', { page, pageSize });
  const p: PaginationState = { limit, offset };
  controller.paginate(p).then(afterLayoutRecompute);
}

// =============================
// Section 20: Sort handler
// =============================
function onTableSortChange(payload: { field: string; direction?: 'asc' | 'desc' }) {
  const orderBy: OrderByState[] = payload.direction ? [{ field: payload.field, direction: payload.direction }] : [];
  emit('sort-change', { orderBy: orderBy as any });
  controller.sort(orderBy).then(afterLayoutRecompute);
}

// =============================
// Section 21: Top action bar (refresh/create/delete)
// =============================
async function handleRefresh() {
  const ok = await emitCancelable('before-refresh');
  if (!ok) return;

  controller
    .apply()
    .then(() => {
      ElMessage.success('列表数据已刷新');
      const total = Number(((store.state as any).result?.total ?? 0) as any) || 0;
      emit('refresh-success', { items: (items.value as any) || [], total });
      return afterLayoutRecompute();
    })
    .catch(e => {
      const err = e instanceof Error ? e : new Error(String(e));
      emit('action-error', { action: 'refresh', error: err });
      ElMessage.error('列表刷新数据失败');
    });
}

async function handleCreate() {
  if (!props.createAction) return;
  try {
    await router.push(props.createAction);
  } catch (e) {
    const err = e instanceof Error ? e : new Error(String(e));
    emit('action-error', { action: 'paginate', error: err });
  }
}

const deleteLoading = ref(false);
async function handleDelete() {
  if (selectedItems.value.length === 0) return;
  const count = selectedItems.value.length;
  try {
    await ElMessageBox.confirm(`确定要删除选中的 ${count} 条记录吗？此操作不可恢复。`, '确认删除', {
      confirmButtonText: '确定删除',
      cancelButtonText: '取消',
      type: 'warning',
      confirmButtonClass: 'el-button--danger',
    });
    // Prefer reading Id from record rows
    const ids = (selectedItems.value as any[])
      .map((r: any) => {
        if (r?.type === 'record') return r?.record?.Id ?? r?.record?.id;
        if (r?.kind === 'record') return r?.payload?.Id ?? r?.payload?.id;
        const rec = r?.payload ?? r?.record ?? r;
        return rec?.Id ?? rec?.id;
      })
      .filter((id: any) => id != null) as string[];

    const ok = await emitCancelable('before-delete', { ids });
    if (!ok) return;

    deleteLoading.value = true;
    if (ids.length === 0) {
      ElMessage.warning('所选记录缺少有效 ID');
      return;
    }

    await store.Delete(['Id', 'in', ids] as QueryCondition<T>);
    ElMessage.success(`成功删除 ${ids.length} 条记录`);
    selection.clear();
    selectedItems.value = [];
    await loadData();
    emit('delete-success', { ids });
  } catch (e) {
    if (e !== 'cancel') {
      const err = e instanceof Error ? e : new Error(String(e));
      emit('action-error', { action: 'delete', error: err });
      ElMessage.error('删除失败');
    }
  } finally {
    deleteLoading.value = false;
  }
}

// =============================
// Section 22: Row click interactions (groups & records)
// =============================
// In tree mode, handle expand, load-more, and detail-row clicks; keep default behavior otherwise
async function onRowClick(p: RowEventHandlerParams) {
  const row = p.rowData as any;

  if (props.clickToSelect) {
    const nextChecked = !selection.isSelected?.(row);
    selection.toggleRow?.(row, nextChecked);
    return;
  }

  if (isGroupMode.value) {
    if (row?.kind === 'group') {
      const key = row?.key as string;
      const expanded = isExpanded(key);
      controller.expandGroup(key, !expanded);
      return;
    }
    if (row?.kind === 'more') {
      const gk = (row as any)?.groupKey || String((row as any)?.key || '').replace(/^more(-g)?:/, '');
      const target = (row as any)?.target || 'records';
      if (gk) {
        if (target === 'groups') {
          await (controller as any).loadMoreGroupChildren(gk);
        } else {
          await (controller as any).loadMoreGroupRecords(gk);
        }
      }
      return;
    }
    // Detail-row click inside a group: emit payload as the record
    if (row?.kind === 'record') {
      emit('row-click', {
        row: row.payload as ClientModel<T>,
        rowIndex: p.rowIndex,
        rowKey: p.rowKey as RowEventHandlerParams['rowKey'],
        event: p.event as MouseEvent,
      });
      return;
    }
    return;
  }

  // In non-grouped mode, unwrap wrapped RecordRow objects to the actual record
  const record = row?.type === 'record' ? row?.record : row?.kind === 'record' ? row?.payload : row;
  emit('row-click', {
    row: record as ClientModel<T>,
    rowIndex: p.rowIndex,
    rowKey: p.rowKey as RowEventHandlerParams['rowKey'],
    event: p.event as MouseEvent,
  });
}

// =============================
// Section 23: Group cell helpers (expand/toggle)
// =============================
// Read grouped expansion state with a type-safe fallback
const isExpanded = (key: string) => (controller as any)?.vm?.expandedKeys?.has?.(key);
const onToggleGroup = async (key: string) => {
  controller.expandGroup(key, !isExpanded(key));
};

// =============================
// Section 24: Exposed public API (selection & load)
// =============================
defineExpose<SelectionExpose<ClientModel<T>> & { load: () => Promise<void> }>({
  selectedItems: selectedItems as any,
  selectedItem: selectedItem as any,
  load: loadData,
});

// =============================
// Section 25: Visibility & pageshow handlers (layout recovery)
// =============================
function visibilityHandler() {
  if (document.visibilityState === 'visible') {
    // Retry recompute on the next macro or micro task to avoid layout jitter after page restore
    setTimeout(() => afterLayoutRecompute(), 32);
  }
}
function pageShowHandler() {
  // Fallback recompute after bfcache restores or history navigation
  setTimeout(() => afterLayoutRecompute(), 32);
}
window.addEventListener('visibilitychange', visibilityHandler);
window.addEventListener('pageshow', pageShowHandler);

// =============================
// Section 26: Forced condition watcher (single apply entry point)
// =============================
// Latest search payload (keyword / appliedFilters / appliedGroups)
const lastSearchPayload = ref<QueryUpdatePayload<T> | null>(null);

// Watch dynamic forcedCondition changes; do not merge them in the view layer, let the controller handle them
watch(
  () => props.forcedCondition,
  () => {
    const base = lastSearchPayload.value;
    controller
      .apply({
        forcedCondition: props.forcedCondition as any,
        appliedFilters: (base?.appliedFilters || []) as any,
        keyword: base?.keyword,
        keywordFields: props.keywordFields || undefined,
        appliedGroups: base?.appliedGroups as any,
      })
      .then(afterLayoutRecompute)
      .catch(() => {});
  },
  { flush: 'post' }
);

// In views without a search bar, inject forcedCondition during the initial onMounted apply
</script>

<style lang="scss" scoped>
.o-list {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  min-width: 0;

  :deep(.o-field-base__cell-item) {
    margin-bottom: 0 !important;
  }
}

.o-list__table {
  flex: 1 1 auto;
  min-height: 0;
  min-width: 0;

  :deep(.el-form-item--default) {
    margin-bottom: 0 !important;
  }
}

/* Header bar styles, kept as a placeholder to avoid empty rules */
/* .o-list__header { padding-bottom: 0; } */
.o-list__action-bar {
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 12px;
  padding-bottom: 4px;
  border-bottom: 1px solid var(--el-border-color-light);
  min-height: 40px;
}

.o-list__search {
  display: flex;
  justify-content: center;
  align-items: center;
  min-width: 240px;
}

.o-list__actions {
  display: flex;
  align-items: center;
  gap: 16px;
}
.o-list__header-right {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

@media (max-width: 768px) {
  .o-list__action-bar {
    grid-template-columns: 1fr;
    grid-auto-rows: auto;
  }
  .o-list__search {
    order: 2;
    justify-content: center;
  }
}

.ovtable__empty {
  width: 100%;
  padding: 24px 0;
  text-align: center;
  color: var(--el-text-color-secondary);
}

.o-group-cell {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.o-group-cell__caret {
  display: inline-block;
  width: 0;
  height: 0;
  border-top: 4px solid transparent;
  border-bottom: 4px solid transparent;
  border-left: 6px solid var(--el-text-color-regular);
  transition: transform 0.12s ease;
  cursor: pointer;
}
.o-group-cell__caret.expanded {
  transform: rotate(90deg);
}
.o-group-cell__label {
  font-weight: 500;
  color: var(--el-text-color-primary);
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.o-group-cell__count {
  color: var(--el-text-color-secondary);
}

.o-more-cell {
  width: 100%;
  text-align: center;
  color: var(--el-text-color-primary);
  cursor: pointer;
  padding: 6px 0;
}
</style>
