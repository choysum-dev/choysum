<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyViewContainer class="gap-5">
    <template v-if="showHeader" #header>
      <div data-anchor="choy.list.view-chrome">
        <ChoyViewActionBar
          layout="thirds"
          class="gap-2 border-b-0 [border-bottom-width:0px]"
          :system-actions-label="_t('System actions')"
          :user-actions-label="_t('User actions')"
        >
          <template v-if="showActions || pageTitle || showTitleOverflow" #system-actions>
            <!-- Keep Save/Discard outside the overridable slot so custom toolbars cannot hide them. -->
            <ChoyButton
              v-if="editable && isEditing"
              type="button"
              size="default"
              variant="default"
              :disabled="inlineSaving"
              @click="handleInlineSave"
            >
              <Check class="size-4" aria-hidden="true" />
              {{ _t('Save') }}
            </ChoyButton>

            <ChoyButton
              v-if="editable && isEditing"
              type="button"
              size="default"
              variant="outline"
              @click="handleInlineDiscard"
            >
              <X class="size-4" aria-hidden="true" />
              {{ _t('Discard') }}
            </ChoyButton>

            <slot
              v-if="showActions"
              name="system-actions"
              :selected-items="selectedItems"
              :is-editing="isEditing"
              :inline-saving="inlineSaving"
              :save="handleInlineSave"
              :discard="handleInlineDiscard"
            >
              <ChoyButton
                v-if="resolvedCreateAction && canCreate"
                type="button"
                size="default"
                variant="default"
                @click="handleCreate"
              >
                <Plus class="size-4" aria-hidden="true" />
                {{ _t('New') }}
              </ChoyButton>
            </slot>

            <!-- Page title + ⋮ (IO / refresh / domain overflow) live in view chrome, not a separate Page band. -->
            <h1
              v-if="pageTitle"
              :id="pageTitleId || undefined"
              data-slot="page-title"
              class="choy-list-view__page-title min-w-0 max-w-[10rem] truncate text-base font-semibold text-foreground sm:max-w-[14rem]"
            >
              {{ pageTitle }}
            </h1>
            <ChoyPageTitleActions
              v-if="showTitleOverflow"
              :has-io-menu="!!pageTitleActionsIo"
              :action-import="pageTitleActionsIo?.actionImport ?? false"
              :action-export="pageTitleActionsIo?.actionExport ?? false"
              :action-import-upload-hint="pageTitleActionsIo?.actionImportUploadHint"
              :action-import-column-mapping="pageTitleActionsIo?.actionImportColumnMapping"
              :action-list-ref="(pageTitleActionsIo?.actionListRef as any)"
              :action-company-id="pageTitleActionsIo?.actionCompanyId"
              :store="store"
            >
              <template v-if="showRefreshInOverflow || $slots['overflow-actions']" #overflow>
                <DropdownMenuItem
                  v-if="showRefreshInOverflow"
                  data-testid="list-overflow-refresh"
                  @select="handleRefresh"
                >
                  <RefreshCw class="size-4" aria-hidden="true" />
                  {{ _t('Refresh') }}
                </DropdownMenuItem>
                <slot name="overflow-actions" :selected-items="selectedItems" />
              </template>
            </ChoyPageTitleActions>
          </template>

          <!-- Only register when filled — empty tray would paint a dangling ActionBar separator. -->
          <template v-if="showActions && !!slots['user-actions']" #user-actions>
            <slot name="user-actions" :selected-items="selectedItems" />
          </template>

          <!-- Selection replaces Search in the center column. Search stays mounted. -->
          <template #center>
            <div
              v-show="hasSelection"
              data-anchor="choy.list.selection-bar"
              class="flex min-w-0 w-full flex-nowrap items-center gap-2 overflow-x-auto"
              role="toolbar"
              :aria-label="_t('Selection actions')"
            >
              <!-- Count as muted text (shadcn data-table); clear is a ghost icon button. -->
              <div class="flex min-w-0 items-center gap-1">
                <span class="truncate text-sm text-muted-foreground tabular-nums">
                  {{ _t('%s selected', selectedItems.length) }}
                </span>
                <ChoyButton
                  type="button"
                  variant="ghost"
                  size="icon"
                  class="size-control shrink-0 text-muted-foreground"
                  :aria-label="_t('Clear selection')"
                  @click="clearSelection"
                >
                  <X class="size-4" aria-hidden="true" />
                </ChoyButton>
              </div>

              <ChoyButton
                v-if="deleteAction && canDelete"
                size="default"
                variant="outline-destructive"
                :disabled="deleteLoading"
                @click="handleDelete"
              >
                <Trash2 class="size-4" aria-hidden="true" />
                {{ _t('Delete') }}
              </ChoyButton>

              <slot name="selection-actions" :selected-items="selectedItems" :clear="clearSelection" />
            </div>
            <ChoySearchFit
              v-if="resolvedSearchView"
              v-show="!hasSelection"
              :field-visible="!hasSelection"
            >
              <component :is="resolvedSearchView" class="min-w-0 w-full" :store="store" @query-update="onSearch" />
            </ChoySearchFit>
          </template>

          <template #header-right>
            <!-- View-mode chrome, then a hairline, then pager at the trailing edge. -->
            <div v-if="$slots['header-right']" class="flex items-center gap-1">
              <slot name="header-right" />
            </div>
            <Separator
              v-if="$slots['header-right'] && showPaginate && canMountPagination"
              orientation="vertical"
              class="mx-0 !h-4 w-px shrink-0 max-md:hidden"
            />
            <div v-if="showPaginate && canMountPagination">
              <ChoyPagination
                class="gap-2"
                :total="listPaginationEffective.total"
                v-model:page="listPageModel"
                v-model:page-size="listPageSizeModel"
              />
            </div>
          </template>
        </ChoyViewActionBar>
      </div>
    </template>

    <ChoyViewBusyHost :busy="listBusy" class="flex min-h-0 min-w-0 flex-1 flex-col">
    <div
      ref="tableWrapRef"
      data-anchor="choy.list.table"
      class="choy-list-view__table flex min-h-0 min-w-0 w-full flex-1 flex-col overflow-hidden [&_.choy-field-base]:mb-0 [&_.choy-data-table]:h-full [&_.choy-data-table]:rounded-none [&_.choy-data-table]:border-0"
      :style="listTableWrapStyle"
    >
      <!-- form-root + edit view-mode only under the table so header search cannot touch the row draft -->
      <ListInlineEditScope
        class="flex min-h-0 flex-1 flex-col"
        :form-root="inlineFormRoot"
        :view-mode="inlineTableViewMode"
      >
        <ChoyTableHost
          class="min-h-0 flex-1"
          :data="tableItems"
          :row-key="computedRowKey"
          :row-height="effectiveRowHeight"
          :header-height="headerHeight"
          :table-height="pixelTableHeight"
          :selection-api="selection"
          :base-index="baseIndex"
          :store="store"
          @selection-change="onSelectionChange"
          @row-click="onRowClick"
          @sort-change="onTableSortChange"
        >
          <ChoyTableColumn v-if="showHandleColumn" type="handle" col-key="__handle__" :vColumnProps="{ width: 36, align: 'center' }" />
          <!-- Automatically inject the leading group column in grouped mode -->
          <ChoyTableColumn v-if="isGroupMode" col-key="__group_label" :sortable="false">
            <template #default="{ row }">
              <div v-if="row?.kind === 'group'" class="inline-flex min-w-0 items-center gap-1.5" :style="{ paddingLeft: `${row.depth * 16}px` }">
                <button
                  type="button"
                  class="inline-flex size-4 shrink-0 items-center justify-center rounded-sm text-muted-foreground hover:bg-accent hover:text-foreground"
                  :aria-expanded="isExpanded(row.key)"
                  :aria-label="_t('Toggle group %s', String(row.label ?? ''))"
                  @click.stop="onToggleGroup(row.key)"
                >
                  <ChevronRight
                    class="size-3.5 transition-transform duration-100"
                    :class="{ 'rotate-90': isExpanded(row.key) }"
                    aria-hidden="true"
                  />
                </button>
                <span class="max-w-full truncate font-medium text-foreground">{{ row.label }}</span>
                <span class="text-muted-foreground">({{ row.count ?? 0 }})</span>
              </div>
              <div v-else-if="row?.kind === 'more'" class="w-full cursor-pointer py-1.5 text-center text-foreground">{{ _t('Click to load more (%s remaining)', Math.max(0, Number(row.remain ?? 0))) }}</div>
              <span v-else></span>
            </template>
          </ChoyTableColumn>
          <slot />
          <template #empty>
            <slot name="empty">
              <!-- Create stays in the list chrome toolbar only; empty state is copy, not a second New. -->
              <ChoyEmpty class="w-full !gap-3 !p-4 md:!p-6" :description="_t('No data')" />
            </slot>
          </template>
        </ChoyTableHost>
      </ListInlineEditScope>
    </div>
    </ChoyViewBusyHost>
  </ChoyViewContainer>
</template>

<script setup lang="ts" generic="T extends BaseModel">
import type { ConditionGroup, QueryUpdatePayload } from '@/web/web/query/types';
import { ChoyMessage } from '../../composables/useChoyMessage';
import {
  computed,
  onMounted,
  onBeforeUnmount,
  provide,
  ref,
  nextTick,
  watch,
  markRaw,
  toRaw,
  useSlots,
} from 'vue';
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { ClientModel, BaseModel, QueryCondition, OrderBy } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import ChoyTableHost from '@/web/web/components/internal/ChoyTableHost.vue';
import ChoyPagination from './ChoyPagination.vue';
import ChoyEmpty from '../layout/ChoyEmpty.vue';
import {
  canMountListPagination,
  listPageSizeToPaginateState,
  listPageToPaginateState,
  resolveListPaginationEffective,
  sameListPaginateState,
} from './listPaginationAdapter';
import ChoyTableColumn from '@/web/web/components/table/ChoyTableColumn.vue';
import { LIST_SHOW_SELECTION_COLUMN_KEY, useTableSelection } from '@/web/web/composables/useTable';
import { CHOY_LIST_HEADER_HEIGHT, CHOY_LIST_ROW_HEIGHT } from '@/web/web/composables/listDensity';
// Search view type: must accept store and emit query-update
import type { Component } from 'vue';
import type { SearchViewComponent } from '@/web/web/query/types';
import type { Ref } from 'vue';
import type { ViewMode } from '@/web/web/components/view/viewScopeKeys';
import { useProvideViewScope } from '@/web/web/components/view/viewScopeKeys';
import { useListInlineEdit } from '@/web/web/composables/useListInlineEdit';
import { hasHandleField, isListRecordRow, listRecordId, unwrapListRecord } from '@/web/web/composables/listRowEdit';
import { LIST_HANDLE_API_KEY, useListHandleReorder } from '@/web/web/composables/useListHandleReorder';
import {
  buildHandleReorderWrites,
  persistHandleReorder,
  shouldDiscardInvisibleEdit,
  syncFlatRowsFromVisibleItems
} from '@/web/web/composables/listViewHandlePersist';
import ListInlineEditScope from '@/web/web/components/view/ListInlineEditScope.vue';
// Controller: unified loading with grouping-first handling
import { createListController } from '@/web/web/controllers/listController';
import type { OrderByState, PaginationState } from '@/web/web/query/state';
import { useCancelableEmit } from '@/web/web/composables/useCancelableEmit';
import { useAdaptiveHeight } from '@/web/web/composables/useAdaptiveHeight';
import ChoyViewContainer from '@/web/web/components/view/ChoyViewContainer.vue';
import ChoyViewActionBar from '@/web/web/components/view/ChoyViewActionBar.vue';
import ChoySearchFit from '@/web/web/components/view/ChoySearchFit.vue';
import ChoyViewBusyHost from '@/web/web/components/view/ChoyViewBusyHost.vue';
import { useListViewActions } from '@/web/web/composables/useListViewActions';
import { useVirtualizationAdapter } from '@/web/web/composables/virtualizationAdapter';
import { awaitFieldSelection } from '@/web/web/query/utils/registry/fieldReady';
import ChoySearchView from '@/web/web/components/view/ChoySearchView.vue';
import { shouldDeferViewFirstFrame } from '@/web/web/components/view/kanbanFirstFrame';
import { canShowAction, type ActionIdMap } from '@/web/web/components/view/actionVisibility';
import { createTranslate } from '@/web/web/i18n';
import type { SelectionExpose, RowEventPayload, RowEventHandlerParams } from '@/web/web/components/view/listViewTypes';
import {
  resolvePageStore,
  useHostPageTitleInView,
  useRegisterPageActionTarget,
} from '@/web/web/composables/usePageContext';
import { useResolvedCreateAction } from '@/web/web/composables/resolveCreateRoute';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyPageTitleActions from '@/web/web/components/layout/ChoyPageTitleActions.vue';
import { Separator } from '@/web/web/components/vendor/ui/separator';
import DropdownMenuItem from '@/web/web/components/vendor/ui/dropdown-menu/DropdownMenuItem.vue';
import { Check, ChevronRight, Plus, RefreshCw, Trash2, X } from 'lucide-vue-next';

defineOptions({ name: 'ChoyListView', inheritAttrs: false });

/**
 * Store-only list engine. Requires :store or a page-provided store
 * (resolvePageStore); there is no chrome-only dual-mode path.
 */
const { _t } = createTranslate('web', { scope: 'web/components/view/ListView' });

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
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
    /**
     * When false, do not host the page title / IO menu in the list chrome
     * (Search more dialogs still show search + pager).
     */
    hostPageTitle?: boolean;
    /** When false, skip registering type="selection" columns (click-to-select pickers). */
    showSelectionColumn?: boolean;
    heightMode?: 'auto' | 'viewport' | 'container';
    containerSelector?: string;

    orderBy?: OrderBy<T> | Array<OrderBy<T>>;

    /* Inject a custom search view that must provide query-update */
    searchView?: SearchViewComponent<T> | typeof ChoySearchView;

    /* Added: control Pagination visibility on the right side of the header */
    showPaginate?: boolean;

    /* Controlled forced condition: always AND with user search conditions, or act as the only condition; dynamic changes trigger re-apply */
    forcedCondition?: QueryCondition<T> | QueryCondition<T>[];

    /**
     * When set, List Search/Count And the source field's meta `condition`
     * (same option as NameSearch `relationConditionSource`).
     */
    relationConditionSource?: { model: string; field: string };

    /** Opt-in S2 row inline edit (click record row → draft → Save/Discard). */
    editable?: boolean;
    /** Sequence / order field for handle drag reorder; default Sequence. */
    handleField?: string;
    /** Show handle column when editable and metadata has handleField. */
    showHandle?: boolean;
    /** Opt out of page IO action-target registration (embedded lists). */
    registerActionTarget?: boolean;
  }>(),
  {
    showHeader: true,
    showActions: true,
    /* Removed: showHeaderRight: true, */
    rowHeight: CHOY_LIST_ROW_HEIGHT,
    headerHeight: CHOY_LIST_HEADER_HEIGHT,
    rowKey: 'Id',
    showPagination: true,
    // Shell path uses footer.top; this only applies outside AppShell.
    viewportGap: 12,
    refreshAction: true,
    deleteAction: true,
    selectionMode: 'multiple',
    clickToSelect: false,
    hostPageTitle: true,
    showSelectionColumn: true,
    heightMode: 'auto',
    containerSelector: '',
    orderBy: undefined,

    /* Added defaults */
    showPaginate: true,

    forcedCondition: undefined,
    relationConditionSource: undefined,
    editable: false,
    handleField: 'Sequence',
    showHandle: true,
    // Keep omitted as undefined (not false) so matching page-store views auto-register.
    registerActionTarget: undefined,
  }
);

const store = resolvePageStore(props.store, 'ListView');
// Relation pickers set registerActionTarget=false; do not inherit the host page's Create route.
const resolvedCreateAction = useResolvedCreateAction(() => props.createAction, {
  enabled: () => props.registerActionTarget !== false,
});
const { pageTitle, pageTitleId, pageTitleActionsIo } = useHostPageTitleInView(
  () => Boolean(props.showHeader && props.hostPageTitle),
);
provide(
  LIST_SHOW_SELECTION_COLUMN_KEY,
  computed(() => props.showSelectionColumn !== false),
);
const slots = useSlots();

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

  (e: 'action-error', payload: { action: 'load' | 'delete' | 'refresh' | 'search' | 'sort' | 'paginate' | 'create'; error: Error }): void;
}>();
const { emitCancelable } = useCancelableEmit(emit as any);

// =============================
// Section 5: View context provisioning
// =============================
// List root stays display so header/search fields never inherit S2 edit mode.
const listViewMode = ref<ViewMode>('display');
useProvideViewScope({ kind: 'List', mode: listViewMode });

const router = useRouter();
// Avoid Vue "component made reactive" warn when a Component is passed as searchView prop.
const resolvedSearchView = computed(() => {
  const view = props.searchView;
  return view ? markRaw(toRaw(view as object)) : null;
});
const canCreate = computed(() => canShowAction(props.actionIds?.create, props.hasAction));
const canRefresh = computed(() => canShowAction(props.actionIds?.refresh, props.hasAction));
const canDelete = computed(() => canShowAction(props.actionIds?.delete, props.hasAction));
const showRefreshInOverflow = computed(
  () => props.showActions && props.refreshAction && canRefresh.value,
);
const showTitleOverflow = computed(
  () => !!pageTitleActionsIo.value || showRefreshInOverflow.value || !!slots['overflow-actions'],
);
// List controller
const controller = createListController(store as any);
const listBusy = computed(() => !!controller.vm.loading);

// Whether grouped mode is active, based on the controller result
const isGroupMode = computed(() => controller.vm.result?.kind === 'group');

const flatRows = ref<any[]>([]);

// items must be declared before syncFlatRowsFromItems (hoisted via function body at call time).
function syncFlatRowsFromItems() {
  flatRows.value = syncFlatRowsFromVisibleItems(items.value, isGroupMode.value);
}

const editableEnabled = computed(() => props.editable === true);
const inlineEdit = useListInlineEdit<T>({
  store,
  enabled: editableEnabled,
  translateScope: 'web/components/view/ListView',
  onSaved: async () => {
    await controller.apply();
    syncFlatRowsFromItems();
  },
});
const isEditing = inlineEdit.isEditing;
const inlineSaving = inlineEdit.saving;
// Top-level bindings so the template auto-unwraps refs (inlineEdit itself is a plain object).
const inlineFormRoot = inlineEdit.formRoot;
const inlineTableViewMode = inlineEdit.tableViewMode;

// Always use the unified key field from DataSetSnapshot RecordRow/GroupRow, aligned with controller output
const computedRowKey = computed(() => 'key');

// The view no longer handles legacy wrappers and relies on controller visible nodes
const items = computed<any[]>(() => controller.vm.visibleNodes || []);

watch(
  items,
  async () => {
    if (inlineEdit.isEditing.value) {
      const { discard, warn } = shouldDiscardInvisibleEdit(
        inlineEdit.isEditing.value,
        inlineEdit.editingRowId.value,
        items.value,
        inlineEdit.isDirty()
      );
      if (discard) {
        if (warn) {
          ChoyMessage.warning(_t('Editing was discarded because the row is no longer visible.'));
        }
        await inlineEdit.discard();
      }
    }
    // Always resync flatRows so pagination/sort/refresh do not leave a stale table.
    // mapItemsWithDraft still overlays the active draft onto the matching row id.
    syncFlatRowsFromItems();
  },
  { immediate: true }
);

const showHandleColumn = computed(
  () =>
    props.showHandle !== false &&
    props.editable === true &&
    !isGroupMode.value &&
    hasHandleField(store, props.handleField)
);

const isReordering = ref(false);
const handleEnabled = computed(() => {
  if (!showHandleColumn.value || inlineEdit.isEditing.value || isReordering.value) return false;
  // Reorder only when unsorted, or sorted by handleField ascending (matches renumber semantics).
  const raw = (store.state as any)?.queryState?.orderBy ?? (store.state as any)?.orderBy;
  const list = Array.isArray(raw) ? raw : raw ? [raw] : [];
  if (!list.length) return true;
  const primary = list[0] as { field?: string; direction?: string } | undefined;
  const field = primary?.field;
  const dir = String(primary?.direction || '').toLowerCase();
  return field === props.handleField && dir === 'asc';
});

const handleReorder = useListHandleReorder({
  rows: () => flatRows.value,
  enabled: handleEnabled,
  handleField: props.handleField,
  sequenceStart: () => (effectivePagination.value.offset ?? 0) + 1,
  getRecord: row => unwrapListRecord(row),
  onReorder: async (rows, changed) => {
    const writes = buildHandleReorderWrites(changed);
    // Do not apply an optimistic order when there is nothing to persist; otherwise the
    // table stays reordered until the next items sync snaps it back.
    if (!writes.length) return;

    const previousFlat = flatRows.value;
    flatRows.value = rows.map(row => ({
      ...row,
      payload: { ...unwrapListRecord(row) },
    }));

    const field = props.handleField;
    isReordering.value = true;
    try {
      await persistHandleReorder({
        writes,
        handleField: field,
        updateById: async (id, payload) => {
          await store.UpdateById(id, payload as any);
        },
        refresh: async () => {
          await controller.apply();
          syncFlatRowsFromItems();
        },
        rollbackFlat: () => {
          flatRows.value = previousFlat;
        },
        onError: reason => {
          ChoyMessage.error(
            reason === 'refresh' ? _t('Failed to refresh after reorder') : _t('Failed to reorder rows')
          );
        },
      });
    } finally {
      isReordering.value = false;
    }
  },
});

provide(LIST_HANDLE_API_KEY, handleReorder);

const tableItems = computed(() => {
  if (isGroupMode.value) return inlineEdit.mapItemsWithDraft(items.value);
  return inlineEdit.mapItemsWithDraft(flatRows.value);
});

async function handleInlineSave() {
  try {
    await inlineEdit.save();
  } catch {
    /* error surfaced in composable */
  }
}

async function handleInlineDiscard() {
  await inlineEdit.discard();
  syncFlatRowsFromItems();
}

// =============================
// Section 8: Selection management
// =============================
// Prefer the new wrapped key first, then __rowKey / Id.
const selection = useTableSelection(
  (row: any) =>
    row?.key ?? row?.__rowKey ?? row?.Id ?? (typeof row === 'object' ? ((row as any)?.payload?.Id) : undefined),
  props.selectionMode
);

// Selected items; bulk actions render in the center selection bar (Odoo-style).
const selectedItems = ref<any[]>([]);
const hasSelection = computed(() => selectedItems.value.length > 0);

function clearSelection() {
  selection.clear();
  selectedItems.value = [];
  emit('selection-change', { rows: [], ids: [] });
}

function onSelectionChange(rows: any[]) {
  // In group-tree mode, keep only the actual record Ids of detail rows
  const ids = (rows || [])
    .map((r: any) => {
      if (r?.type === 'record') return r?.record?.Id;
      if (r?.kind === 'record') return r?.payload?.Id;
      const rec = r?.payload ?? r?.record ?? r;
      return rec?.Id;
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

const listPaginationEffective = computed(() =>
  resolveListPaginationEffective({
    total: effectiveTotal.value,
    limit: effectivePagination.value.limit,
    offset: effectivePagination.value.offset,
  }),
);
const canMountPagination = computed(() => canMountListPagination(listPaginationEffective.value));
// ChoyPagination resets page→1 after a pageSize change; suppress that follow-up so it
// cannot emit a second paginate with the stale limit before the store catches up.
let suppressListPageModel = false;
const listPageModel = computed({
  get: () => listPaginationEffective.value.currentPage,
  set: (page: number) => {
    if (suppressListPageModel) return;
    const current = listPaginationEffective.value;
    const next = listPageToPaginateState(page, current);
    if (sameListPaginateState(next, { limit: current.limit, offset: current.offset })) return;
    onPaginateState(next);
  },
});
const listPageSizeModel = computed({
  get: () => listPaginationEffective.value.pageSize,
  set: (size: number) => {
    const next = listPageSizeToPaginateState(size, listPaginationEffective.value);
    if (!next) return;
    suppressListPageModel = true;
    onPaginateState(next);
    queueMicrotask(() => {
      suppressListPageModel = false;
    });
  },
});

// ChoySearchView owns controlled search display and event forwarding; the view no longer passes controlled filters or groups

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
// Section 13: Height — workspace flex fill vs dialog pixel height
// =============================
// auto (default): CSS flex-1 fill under AppShell workspace (no shell scroll).
// viewport | container: pixel height for relation pickers / dialog embeds.
const usePixelHeight = computed(
  () => props.heightMode === 'viewport' || props.heightMode === 'container',
);
const adaptiveMode = computed(() => {
  if (props.heightMode === 'container') return 'container' as const;
  if (props.heightMode === 'viewport') return 'viewport' as const;
  return 'none' as const;
});

const tableWrapRef = ref<HTMLElement | null>(null);
const {
  height: adaptiveTableHeight,
  recompute: recomputeTableHeight,
} = useAdaptiveHeight(tableWrapRef as Ref<Element | null>, {
  // Snapshot at setup: workspace lists use auto→none; pickers pass viewport/container.
  mode: adaptiveMode.value,
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
const effectiveRowHeight = computed(
  () => virtualization.config.value.rowHeight || props.rowHeight || CHOY_LIST_ROW_HEIGHT,
);

/** Pixel height only for viewport/container embeds; workspace lists leave this undefined. */
const pixelTableHeight = computed(() => {
  if (!usePixelHeight.value) return undefined;
  const h = adaptiveTableHeight.value;
  return h > 0 ? h : undefined;
});
const listTableWrapStyle = computed(() => {
  const h = pixelTableHeight.value;
  if (h == null) return undefined;
  const px = `${h}px`;
  return { height: px, maxHeight: px };
});
watch(
  () => props.rowHeight,
  v => {
    if (v && v !== virtualization.config.value.rowHeight) {
      virtualization.config.value.rowHeight = v;
    }
  }
);

// Utility: recompute pixel height after layout changes (embed modes only).
function afterLayoutRecompute() {
  if (!usePixelHeight.value) return nextTick();
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

  // Only ChoySearchView guarantees a mount-time query-update with UserFilter defaults.
  // Custom SearchViewComponent implementations may never emit; keep the mount apply.
  if (shouldDeferViewFirstFrame(resolvedSearchView.value, ChoySearchView)) {
    return;
  }

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
    ChoyMessage.error(_t('Failed to load list'));
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
    // Ensure registered fields participate in the first query
    // When Search fires onMounted on the first frame, table columns and fields may not have finished registering yet
    // Proactively wait for one registration cycle, up to a few nextTick turns
    if (!firstApplied.value) firstApplied.value = true;
    awaitFieldSelection(store, { requireNonEmpty: true })
      .then(() =>
        controller.apply({
          // The view layer passes only the external forced condition; the UI tag tree can use payload.appliedFilters directly
          forcedCondition: props.forcedCondition as any,
          appliedFilters: payload.appliedFilters as any,
          keyword: payload.keyword,
          keywordFields: props.keywordFields || undefined,
          appliedGroups: payload.appliedGroups as any,
        }),
      )
      .then(afterLayoutRecompute)
      .catch((e: unknown) => {
        emit('action-error', { action: 'search', error: e instanceof Error ? e : new Error(String(e)) });
      });
  }
}

// =============================
// Section 19: Pagination handler
// =============================
// Pagination only updates state and emits events
function onPaginateState({ limit, offset }: { limit: number; offset: number }) {
  // Still emit paginate externally for legacy listeners
  const page = Math.floor(offset / Math.max(1, limit)) + 1;
  const pageSize = limit;
  emit('paginate', { page, pageSize });
  const p: PaginationState = { limit, offset };
  controller
    .paginate(p)
    .then(afterLayoutRecompute)
    .catch((e: unknown) => {
      emit('action-error', { action: 'paginate', error: e instanceof Error ? e : new Error(String(e)) });
    });
}

// =============================
// Section 20: Sort handler
// =============================
function onTableSortChange(payload: { field: string; direction?: 'asc' | 'desc' }) {
  const orderBy: OrderByState[] = payload.direction ? [{ field: payload.field, direction: payload.direction }] : [];
  emit('sort-change', { orderBy: orderBy as any });
  controller
    .sort(orderBy)
    .then(afterLayoutRecompute)
    .catch((e: unknown) => {
      emit('action-error', { action: 'sort', error: e instanceof Error ? e : new Error(String(e)) });
    });
}

const { deleteLoading, handleRefresh, handleCreate, handleDelete } = useListViewActions({
  store,
  controller,
  router,
  resolvedCreateAction,
  selectedItems,
  items,
  selection,
  emitCancelable,
  emit: emit as (event: string, payload?: unknown) => void,
  loadData,
  afterLayoutRecompute,
  _t,
});

// =============================
// Section 22: Row click interactions (groups & records)
// =============================
// In tree mode, handle expand, load-more, and detail-row clicks; keep default behavior otherwise
async function onRowClick(p: RowEventHandlerParams) {
  const row = p.rowData as any;

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
    // Detail-row click inside a group
    if (row?.kind === 'record') {
      if (props.editable) {
        const entered = await inlineEdit.enterEdit(row);
        if (entered) return;
      }
      if (props.clickToSelect) {
        const nextChecked = !selection.isSelected?.(row);
        selection.toggleRow?.(row, nextChecked);
        return;
      }
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

  // S2 inline edit takes precedence over click-to-select for record rows.
  if (props.editable && isListRecordRow(row)) {
    const entered = await inlineEdit.enterEdit(row);
    if (entered) return;
  }

  if (props.clickToSelect) {
    const nextChecked = !selection.isSelected?.(row);
    selection.toggleRow?.(row, nextChecked);
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
defineExpose<
  SelectionExpose<ClientModel<T>> & {
    load: () => Promise<void>;
    inlineEdit: typeof inlineEdit;
    flatRows: typeof flatRows;
    listPageModel: typeof listPageModel;
    listPageSizeModel: typeof listPageSizeModel;
    canMountPagination: typeof canMountPagination;
    clearSelection: typeof clearSelection;
    onSelectionChange: typeof onSelectionChange;
  }
>({
  selectedItems: selectedItems as any,
  selectedItem: selectedItem as any,
  load: loadData,
  inlineEdit,
  flatRows,
  listPageModel,
  listPageSizeModel,
  canMountPagination,
  clearSelection,
  onSelectionChange,
});

useRegisterPageActionTarget({
  store,
  enabled: () => props.registerActionTarget,
  target: {
    get selectedItems() {
      return selectedItems.value;
    },
    refresh: () => loadData(),
  },
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

// Persist relationConditionSource so listController can pass it on Search/Count.
watch(
  () => props.relationConditionSource,
  v => {
    (store.state as any).queryState = {
      ...(store.state as any).queryState,
      relationConditionSource: v,
    };
  },
  { immediate: true }
);

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

// Re-apply when relationConditionSource changes after the first load.
watch(
  () => props.relationConditionSource,
  (next, prev) => {
    if (!firstApplied.value) return;
    if (next === prev) return;
    const same =
      next?.model === prev?.model && next?.field === prev?.field;
    if (same) return;
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

