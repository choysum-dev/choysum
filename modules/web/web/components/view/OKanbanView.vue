<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OViewContainer :showHeader="showHeader">
    <template #header>
      <div class="o-kanban__action-bar">
        <div class="o-kanban__actions">
          <div class="o-kanban__system-actions" v-if="showActions">
            <slot name="system-actions" :selected-items="emptySelection">
              <el-button v-if="createAction" size="small" plain type="primary" @click="handleCreate">
                <el-icon><Plus /></el-icon>
                {{ _t('New') }}
              </el-button>

              <el-button v-if="refreshAction" size="small" plain @click="handleRefresh">
                <el-icon><Refresh /></el-icon>
                {{ _t('Refresh') }}
              </el-button>
            </slot>
          </div>

          <div class="o-kanban__user-actions" v-if="showActions">
            <slot name="user-actions" :selected-items="emptySelection" />
          </div>
        </div>

        <!-- Centered search: render only when searchView is provided -->
        <div class="o-kanban__search" v-if="searchView">
          <component :is="searchView" :store="store" @query-update="onSearch" />
        </div>

        <div class="o-kanban__header-right">
          <div class="o-kanban__default-pagination" v-if="showPaginate">
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

    <!-- Field-registration slot takes priority: use #fields when provided, otherwise fall back to the default slot for backward compatibility -->
    <slot name="fields" v-if="hasFieldsSlot" />
    <slot v-else />

    <div class="okanban" ref="boardWrapRef">
      <!-- Grouped mode: render by lane -->
      <div class="okanban__board" v-if="isGroupMode">
        <div v-for="lane in lanes" :key="lane.key" class="okanban__lane">
          <div class="okanban__lane-header">
            <KanbanLaneFieldProvider :lane="lane">
              <slot name="lane-header" :lane="lane">
                <div class="lane-header__default">
                  <span class="title">{{ lane.label }}</span>
                  <span class="count">({{ lane.count ?? 0 }})</span>
                </div>
              </slot>
            </KanbanLaneFieldProvider>
          </div>

          <slot name="lane" :lane="lane" :records="laneRecords[lane.key]">
            <div class="okanban__cards">
              <!-- Fully non-virtualized rendering: use vuedraggable for drag and drop -->
              <draggable
                v-bind="dragOptions"
                :list="laneRecords[lane.key]"
                item-key="key"
                group="okanban-cards"
                class="okanban__cards-draggable"
                :data-lane-key="lane.key"
                :disabled="laneFieldReadonly"
                @start="onDragStart"
                @end="onDragEnd"
                @change="onDragChange"
              >
                <template #item="{ element, index }">
                  <div
                    :class="['okanban__card', laneFieldReadonly ? 'drag-disabled' : '']"
                    @click="emitCardClick(element)"
                    :data-record-id="element.payload?.Id"
                  >
                    <KanbanCardFieldProvider :record="element.payload">
                      <slot name="card" :record="element.payload" :lane="lane">
                        <div class="okanban__card-default">
                          <div class="title">{{ element?.payload?.Title ?? element?.payload?.Id }}</div>
                        </div>
                      </slot>
                    </KanbanCardFieldProvider>
                  </div>
                </template>
                <template #footer>
                  <div v-if="getLaneRemain(lane) > 0" class="okanban__lane-more" @click="onClickLoadMore(lane)">
                    {{ _t('Load more (%s remaining)', getLaneRemain(lane)) }}
                  </div>
                  <div v-else-if="(laneRecords[lane.key] || []).length === 0" class="okanban__lane-empty">
                    <slot name="card-empty" :lane="lane">{{ _t('No data in this column') }}</slot>
                  </div>
                </template>
              </draggable>

              <!-- More/empty hints in non-virtualized mode -->
            </div>
          </slot>
        </div>
      </div>
      <!-- Non-grouped mode: remove lane wrappers and lay cards out across the available width -->
      <div class="okanban__flat" v-else>
        <div v-if="flatRecords.length === 0" class="okanban__flat-empty">
          <slot name="card-empty" :lane="flatLane">{{ _t('No data') }}</slot>
        </div>
        <div v-else class="okanban__flat-cards">
          <div v-for="element in flatRecords" :key="element.key" class="okanban__card" @click="emitCardClick(element)" :data-record-id="element.payload?.Id">
            <KanbanCardFieldProvider :record="element.payload">
              <slot name="card" :record="element.payload" :lane="flatLane">
                <div class="okanban__card-default">
                  <div class="title">{{ element?.payload?.Title ?? element?.payload?.Id }}</div>
                </div>
              </slot>
            </KanbanCardFieldProvider>
          </div>
        </div>
      </div>
    </div>
  </OViewContainer>
</template>

<script setup lang="ts" generic="T extends BaseModel">
import { computed, onMounted, ref, watch, nextTick, useSlots, DefineComponent } from 'vue';
import type { RouteLocationRaw } from 'vue-router';
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { BaseModel, ClientModel, QueryCondition } from '@/core/rpc';
import type { GroupBySpec } from '@/core/service/api/query';
import type { OrderByState } from '@/web/web/query/state';
import { createKanbanController } from '@/web/web/controllers/kanbanController';
import type { ConditionGroup, QueryUpdatePayload } from '@/web/web/query/types';
import type { PaginationState } from '@/web/web/query/state';
import { awaitFieldSelection } from '@/web/web/query/utils/registry/fieldReady';
import { useCancelableEmit } from '@/web/web/composables/useCancelableEmit';
import OViewContainer from '@/web/web/components/view/OViewContainer.vue';
// Search view type: must accept store and emit query-update
import type { Component } from 'vue';
import type { SearchViewComponent } from '@/web/web/query/types';
import OPagination from '@/web/web/components/view/OPagination.vue';
import { Plus, Refresh } from '@element-plus/icons-vue';
import { ElMessage } from 'element-plus';
import draggable from 'vuedraggable';
import { provide, defineComponent, reactive } from 'vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { shouldDeferViewFirstFrame } from '@/web/web/components/view/kanbanFirstFrame';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/view/OKanbanView' });

// Props aligned with the simplified OListView surface

const props = withDefaults(
  defineProps<{
    store: WebModelStore<T>;
    keywordFields?: string[];
    showHeader?: boolean;
    showActions?: boolean;
    /* Render search only when a searchView component is provided */
    createAction?: string | RouteLocationRaw;
    refreshAction?: boolean;
    orderBy?: OrderByState[];
    showPaginate?: boolean;
    forcedCondition?: QueryCondition<T> | QueryCondition<T>[];
    // Inject a custom search view that must expose a store prop and a query-update event
    searchView?: SearchViewComponent<T> | typeof OSearchView;
    /**
     * Upper bound for lanes to preload automatically on first load or after changes.
     * - >0: preload only the first N lanes
     * - <=0/null/undefined: no limit, preload every visible lane
     */
    preloadLaneLimit?: number | null;
    // laneLoadLimit has been removed; the controller drives in-lane loading via queryState.pagination
  }>(),
  {
    showHeader: true,
    showActions: true,
    refreshAction: true,
    orderBy: undefined,
    showPaginate: true,
    forcedCondition: undefined,
    preloadLaneLimit: undefined,
  }
);

// Emits, simplified and renamed from row-click to card-click, with card-move added
const emit = defineEmits<{
  (e: 'before-load', payload: { query: any; page: number; pageSize: number; confirm: () => void; cancel: () => void }): void;
  (e: 'before-refresh', payload: { confirm: () => void; cancel: () => void }): void;
  (e: 'load-success', payload: { items: ClientModel<T>[]; total: number }): void;
  (e: 'refresh-success', payload: { items: ClientModel<T>[]; total: number }): void;
  (e: 'search-change'): void;
  (e: 'paginate', payload: { page: number; pageSize: number }): void;
  (e: 'card-click', payload: { row: ClientModel<T> }): void;
  (e: 'card-move', payload: { cardId: string; fromLane: string; toLane: string; index: number }): void;
  (e: 'action-error', payload: { action: 'load' | 'refresh' | 'search' | 'paginate' | 'move'; error: Error }): void;
}>();
const { emitCancelable } = useCancelableEmit(emit as any);

const router = useRouter();
const store = props.store;
const controller = createKanbanController(store as any);

// Prefer the named #fields slot when present; otherwise fall back to the default slot
const slots = useSlots();
const hasFieldsSlot = computed(() => !!slots.fields);

// Keep header slot params compatible; Kanban does not support selection, so this stays an empty array
const emptySelection = ref<any[]>([]);

// Default filters and groups are handled centrally by OSearchView

// Controlled current groups, exposed for OSearch display
const currentAppliedGroupsForSearch = computed<GroupBySpec<T>[]>(() => {
  const qs: any = (store.state as any)?.queryState;
  const gb = qs?.appliedGroups;
  return Array.isArray(gb) ? gb : gb ? [gb] : [];
});

// Controlled filters passed to OSearch for the tag tree
const currentAppliedFiltersForSearch = computed<ConditionGroup[]>(() => {
  const qs: any = (store.state as any)?.queryState;
  return Array.isArray(qs?.appliedFilters) ? (qs.appliedFilters as ConditionGroup[]) : [];
});

// Effective pagination and total, aligned with OListView
const effectivePagination = computed<PaginationState>(() => {
  const qs: any = (store.state as any)?.queryState;
  return qs && qs.pagination ? (qs.pagination as PaginationState) : { limit: 20, offset: 0 };
});
const effectiveTotal = computed<number>(() => {
  const t = Number(((store.state as any).result?.total ?? controller.vm.result?.total ?? 0) as any);
  return Number.isFinite(t) ? t : 0;
});

// Current keyword, used to preserve search state on refreshes and condition changes
const currentKeyword = computed<string | undefined>(() => {
  const kw = (store.state as any)?.queryState?.keyword;
  return typeof kw === 'string' && kw.length > 0 ? kw : undefined;
});

// Lane definition for the first grouping level
type Lane = { key: string; label: string; count?: number; condition?: any };
const lanes = controller.lanes;

// Lane-to-record-array cache
type RecordRow = { kind: 'record'; key: string; payload: any; groupIndex?: number };
const laneRecords = controller.laneRecords as any;

// Non-grouped mode: flattened records and a synthetic lane definition
const isGroupMode = computed(() => controller.vm.result?.kind === 'group');
const flatRecords = computed<RecordRow[]>(() => (controller.vm.result?.kind === 'search' ? (controller.vm.result?.rows as any[]) || [] : []));
const flatLane = computed(() => ({ key: 'all', label: _t('All'), count: flatRecords.value.length }));

// Drag options, preserving the baseline behavior
const dragOptions = { animation: 150, forceFallback: false } as any;

// Disable dragging when the grouping field is readonly in field metadata
const laneFieldReadonly = computed(() => {
  if (!isGroupMode.value) return false;
  const field = controller.getLaneField();
  if (!field) return false;
  const meta = (store.fieldsMetadata || {})[field];
  return meta?.isReadonly === true;
});

// Drag helper state; no extra switching logic is needed after removing virtualization

// Internal provider: field components inside Kanban cards use inline render mode by default, unless renderMode is explicitly set
const KanbanCardFieldProvider = defineComponent({
  name: 'KanbanCardFieldProvider',
  props: { record: { type: Object, default: null } },
  setup(p, { slots }) {
    // Force field components to use inline render mode
    provide('o-field-render-override', 'inline');
    // Inject a pseudo form-root so useField can access the current card record through fieldRef in inline mode
    // Only provide draft, not getField/setField, to keep it readonly in the default Kanban card display mode
    if (p.record && typeof p.record === 'object') {
      provide('form-root', { draft: p.record });
    }
    return () => slots.default?.();
  },
});

// Lane-header field provider so field components can read aggregate and count fields in headers
const KanbanLaneFieldProvider = defineComponent({
  name: 'KanbanLaneFieldProvider',
  props: { lane: { type: Object, default: null } },
  setup(p, { slots }) {
    // Always use inline render mode to match the card presentation
    provide('o-field-render-override', 'inline');
    // Provide form-root while keeping the draft reference stable; sync properties only so child fields keep their context
    const draft = reactive<any>({});
    if (p.lane && typeof p.lane === 'object') Object.assign(draft, p.lane);
    provide('form-root', { draft });
    watch(
      () => p.lane,
      nv => {
        // Copy the new lane fields onto the same draft reference so children stay connected
        // Clear old fields first; only repopulate when nv exists
        const keys = Object.keys(draft);
        for (const k of keys) delete (draft as any)[k];
        if (nv && typeof nv === 'object') Object.assign(draft, nv as any);
      }
    );
    return () => slots.default?.();
  },
});

// Resolve the lane field from groupby[0]
// kanbanController derives the lane field and lane value

// Derive the lane value from __condition or key
// kanbanController owns extractLaneValue

// Compute remaining items for a lane
function getLaneRemain(lane: Lane): number {
  return controller.getLaneRemain(lane as any);
}

// Load the first batch of lane records
async function ensureLaneRecords(key: string): Promise<void> {
  await controller.preloadLane(key);
  // If preloaded records exceed the threshold, virtualization would be enabled dynamically by shouldVirtualize
}

// Load more records
async function onClickLoadMore(lane: Lane) {
  await controller.loadMoreLane(lane.key);
}

// Drag hooks: change for local updates and end for persistence
function onDragChange() {
  // vuedraggable has already updated the local array
}

function onDragStart(evt: any) {
  try {
    const fromEl: HTMLElement | undefined = evt?.from;
    const fromLaneKey: string | undefined = fromEl?.getAttribute('data-lane-key') || (fromEl as any)?.dataset?.laneKey;
    // No need to cache scroll position in non-virtualized mode
  } catch {}
}

async function onDragEnd(evt: any) {
  try {
    const toEl: HTMLElement | undefined = evt?.to;
    const fromEl: HTMLElement | undefined = evt?.from;
    const newIndex: number = Number(evt?.newIndex ?? -1);
    const itemEl: HTMLElement | undefined = evt?.item;
    const toLaneKey: string | undefined = toEl?.getAttribute('data-lane-key') || (toEl as any)?.dataset?.laneKey;
    const fromLaneKey: string | undefined = fromEl?.getAttribute('data-lane-key') || (fromEl as any)?.dataset?.laneKey;
    if (!toLaneKey || newIndex < 0) return;
    // Ensure the target lane already has an array
    await ensureLaneRecords(toLaneKey);
    let moved = (laneRecords.value[toLaneKey] || [])[newIndex];
    // Fallback: if the target lane was previously empty and the model has not caught up yet, read the record id from the DOM
    if (!moved || !moved.payload) {
      const domId = itemEl?.getAttribute('data-record-id');
      if (!domId) return;
      // Even without the full payload, move can still run before both sides are refreshed locally
      moved = { payload: { Id: domId } } as any;
    }
    const id = String(moved.payload?.Id ?? moved.payload?.id);
    const toLane = lanes.value.find(l => l.key === toLaneKey);
    const fromLane = lanes.value.find(l => l.key === fromLaneKey);

    // Cross-lane moves are persisted by kanbanController
    if (fromLaneKey && toLaneKey && fromLaneKey !== toLaneKey) {
      try {
        await controller.moveCard(id, fromLaneKey, toLaneKey, newIndex);
        emit('card-move', { cardId: id, fromLane: fromLaneKey, toLane: toLaneKey, index: newIndex });
      } catch (e: any) {
        ElMessage.error(_t('Move failed; refreshed to recover'));
        emit('action-error', { action: 'move', error: e instanceof Error ? e : new Error(String(e)) });
        // Refresh on failure so the UI rolls back
        await controller.apply();
        // Refresh each lane cache
        for (const ln of lanes.value) {
          const arr = (controller.vm.result?.groupRecords?.get?.(ln.key) as any) || [];
          laneRecords.value[ln.key] = Array.isArray(arr) ? arr.slice() : [];
        }
      }
    }
    // No need to restore scroll position in non-virtualized mode
  } catch (e) {
    // Ignore exceptions here; action-error already reports them
  }
}

// Header actions: refresh and create
async function handleRefresh() {
  const ok = await emitCancelable('before-refresh');
  if (!ok) return;
  try {
    const hasForced = props.forcedCondition !== undefined && props.forcedCondition !== null;
    const refreshOverrides: any = {
      // Keep the current grouping so the view remains in grouped mode and does not lose context on apply
      appliedGroups: currentAppliedGroupsForSearch.value as any,
      // Preserve the current keyword and keyword fields while refreshing
      keyword: currentKeyword.value,
      keywordFields: props.keywordFields && props.keywordFields.length > 0 ? props.keywordFields : (store as any)?.state?.queryState?.keywordFields,
    };
    // Only pass filters explicitly when forcedCondition exists;
    // otherwise align with OListView.handleRefresh and keep the current queryState.appliedFilters
    if (hasForced) {
      refreshOverrides.forcedCondition = props.forcedCondition as any;
    }
    await controller.apply(refreshOverrides);
    const total = Number(((store.state as any).result?.total ?? 0) as any) || 0;
    emit('refresh-success', { items: [], total });
    // Clear the lane cache after refresh, then reload lazily as needed
    laneRecords.value = {};
    await nextTick();
    await preloadInitialLanes();
  } catch (e) {
    const err = e instanceof Error ? e : new Error(String(e));
    emit('action-error', { action: 'refresh', error: err });
    ElMessage.error(_t('Kanban refresh failed'));
  }
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

// Search and pagination — first-frame load comes from OSearchView query-update when searchView is set.
function onSearch(payload: QueryUpdatePayload<T>) {
  emit('search-change');
  lastSearchPayload.value = payload;
  if (payload) {
    awaitFieldSelection(store, { requireNonEmpty: true }).then(() => {
      controller
        .apply({
          forcedCondition: props.forcedCondition as any,
          appliedFilters: (payload.appliedFilters || []) as any,
          keyword: payload.keyword,
          keywordFields: props.keywordFields || (store as any)?.state?.queryState?.keywordFields || undefined,
          appliedGroups: payload.appliedGroups as any,
        })
        .then(async () => {
          laneRecords.value = {};
          await nextTick();
          await preloadInitialLanes();
        })
        .catch(() => {});
    });
  }
}

function onPaginateState({ limit, offset }: { limit: number; offset: number }) {
  const page = Math.floor(offset / Math.max(1, limit)) + 1;
  const pageSize = limit;
  emit('paginate', { page, pageSize });
  const p: PaginationState = { limit, offset };
  controller.paginate(p).then(async () => {
    laneRecords.value = {};
    await nextTick();
    await preloadInitialLanes();
  });
}

// Default card-click forwarding
function emitCardClick(rr: RecordRow) {
  const rec = rr?.payload as ClientModel<T>;
  if (!rec) return;
  emit('card-click', { row: rec });
}

// Normalize and merge forced conditions
// Merge helpers and debug output were removed; the view layer now passes only the forced condition

// First-frame load: when searchView is present, wait for its query-update (includes UserFilter defaults).
onMounted(async () => {
  await nextTick();
  if (props.orderBy !== undefined) {
    (store.state as any).orderBy = props.orderBy as any;
  }
  // Only OSearchView guarantees a mount-time query-update with UserFilter defaults.
  // Custom SearchViewComponent implementations may never emit; keep the mount apply.
  if (shouldDeferViewFirstFrame(props.searchView, OSearchView)) {
    return;
  }
  // Custom search views that already emitted query-update (onSearch sets lastSearchPayload
  // synchronously) must not run a second fallback apply that can race/supersede it.
  if (lastSearchPayload.value) {
    return;
  }
  // laneLoadLimit injection has been removed; queryState.pagination controls loading consistently
  await awaitFieldSelection(store, { requireNonEmpty: true });
  await controller.apply({
    forcedCondition: props.forcedCondition as any,
    // If a keyword was saved, for example after route return or state restoration, include it on the first frame
    keyword: currentKeyword.value,
    keywordFields: props.keywordFields && props.keywordFields.length > 0 ? props.keywordFields : (store as any)?.state?.queryState?.keywordFields,
    appliedGroups: currentAppliedGroupsForSearch.value as any,
  });
  await nextTick();
  await preloadInitialLanes();
});

// Watch dynamic forcedCondition changes
watch(
  () => props.forcedCondition,
  fc => {
    controller
      .apply({
        forcedCondition: fc as any,
        // Reuse conditions previously provided through search interaction when available; otherwise fall back to the current query state
        keyword: lastSearchPayload.value?.keyword ?? currentKeyword.value,
        appliedFilters: (lastSearchPayload.value?.appliedFilters || []) as any,
        keywordFields: props.keywordFields && props.keywordFields.length > 0 ? props.keywordFields : (store as any)?.state?.queryState?.keywordFields,
        appliedGroups: (lastSearchPayload.value?.appliedGroups as any) ?? (currentAppliedGroupsForSearch.value as any),
      })
      .then(async () => {
        laneRecords.value = {};
        await nextTick();
        await preloadInitialLanes();
      })
      .catch(() => {});
  }
);

const boardWrapRef = ref<HTMLElement | null>(null);

// Latest search payload (keyword / appliedFilters / appliedGroups)
const lastSearchPayload = ref<QueryUpdatePayload<T> | null>(null);

// ===== Initial preload: fetch the first batch of visible lanes on demand =====
async function preloadInitialLanes() {
  if (!isGroupMode.value) return;
  const allKeys = lanes.value.map(l => l.key);
  const limit = Number(props.preloadLaneLimit ?? 0);
  const keys = limit > 0 ? allKeys.slice(0, limit) : allKeys;
  await Promise.all(
    keys.map(k =>
      ensureLaneRecords(k).catch(() => {
        /* Ignore a single-lane failure so the rest can continue */
      })
    )
  );
}
</script>

<style lang="scss" scoped>
/* Map the original list header styles under the unified o-kanban__* naming */
.o-kanban__action-bar {
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 12px;
  padding-bottom: 4px;
  border-bottom: 1px solid var(--el-border-color-light);
  min-height: 40px;
}
.o-kanban__actions {
  display: flex;
  align-items: center;
  gap: 16px;
}
.o-kanban__system-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.o-kanban__user-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.o-kanban__search {
  display: flex;
  justify-content: center;
  align-items: center;
  min-width: 240px;
}
.o-kanban__header-right {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}
.o-kanban__default-pagination {
  display: flex;
  align-items: center;
}

@media (max-width: 768px) {
  .o-kanban__action-bar {
    grid-template-columns: 1fr;
    grid-auto-rows: auto;
  }
  .o-kanban__search {
    order: 2;
    justify-content: center;
  }
}

/* ===== Base Kanban layout, following the visual rhythm of ListView ===== */
.okanban {
  display: block;
  width: 100%;
  height: 100%;
  overflow-x: auto;
  overflow-y: hidden;
  padding: 4px 0 12px 0; /* Match the top spacing rhythm used by the list view */
  box-sizing: border-box;
}

.okanban__board {
  display: flex;
  flex-direction: row;
  align-items: stretch;
  gap: 16px; /* Slightly widen the gap between lanes for stronger separation */
  padding: 0 8px; /* Keep horizontal padding aligned with the ListView table wrapper */
  min-height: 160px;
}

/* ===== Lane styles ===== */
.okanban__lane {
  background: var(--el-fill-color-blank);
  /* Remove borders and rounded corners */
  border: none;
  border-radius: 0;
  min-width: 300px;
  max-width: 340px;
  flex: 0 0 300px;
  display: flex;
  flex-direction: column;
  position: relative;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
}

.okanban__lane-header {
  padding: 10px 12px 6px 12px;
  border-bottom: 1px solid var(--el-border-color-light);
  background: linear-gradient(180deg, var(--el-fill-color-light) 0%, var(--el-fill-color-blank) 100%);
}

.lane-header__default {
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--el-text-color-primary);
}

/* ===== Card list area, with its own scroll container so the board height does not hide the header ===== */
.okanban__cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 10px 12px 12px 12px;
  overflow-y: auto;
  max-height: calc(100vh - 240px); /* Viewport height minus the reserved header and pagination area */
  box-sizing: border-box;
}

/* Leave slight vertical spacing between cards inside the drag container */
.okanban__cards-draggable {
  display: flex;
  flex-direction: column;
  gap: 8px; /* Slight spacing keeps cards from touching vertically */
}

/* Firefox scrollbar tuning */
.okanban__cards {
  scrollbar-width: thin;
  scrollbar-color: var(--el-border-color-light) transparent;
}
/* WebKit scrollbar tuning */
.okanban__cards::-webkit-scrollbar {
  width: 8px;
}
.okanban__cards::-webkit-scrollbar-track {
  background: transparent;
}
.okanban__cards::-webkit-scrollbar-thumb {
  background: var(--el-border-color-light);
  border-radius: 4px;
}

/* ===== Single card ===== */
/*
  okanban__card acts only as the drag wrapper and keeps styling minimal.
  If the caller does not provide a custom card slot, .okanban__card-default supplies the baseline appearance.
*/
.okanban__card {
  position: relative;
  cursor: grab;
}
.okanban__card.drag-disabled {
  cursor: default;
}
.okanban__card.drag-disabled :deep(*) {
  cursor: default !important;
}
.okanban__card.drag-disabled .okanban__card-default:hover {
  box-shadow: none;
}
.okanban__card.drag-disabled .okanban__card-default:active {
  transform: none;
}

/* Default card appearance, used only when no custom slot is provided */
.okanban__card-default {
  background: var(--el-color-white);
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  padding: 10px 12px 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  transition:
    box-shadow 0.18s ease,
    transform 0.18s ease;
}
.okanban__card-default:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}
.okanban__card-default:active {
  transform: scale(0.98);
}

.okanban__card-default .title {
  font-size: 13px;
  font-weight: 500;
  line-height: 1.3;
  color: var(--el-text-color-primary);
  word-break: break-word;
}

/* ===== Lane footer more/empty states ===== */
.okanban__lane-more {
  width: 100%;
  text-align: center;
  color: var(--el-text-color-primary);
  cursor: pointer;
  padding: 8px 0 4px 0;
  font-size: 12px;
  font-weight: 500;
  opacity: 0.9;
  transition: color 0.15s;
}
.okanban__lane-more:hover {
  color: var(--el-color-primary);
}

.okanban__lane-empty {
  opacity: 0.6;
  font-size: 12px;
  text-align: center;
  padding: 12px 0 4px 0;
  color: var(--el-text-color-secondary);
}

/* ===== Responsive behavior: shrink lane width and allow horizontal scrolling on narrow screens ===== */
@media (max-width: 1200px) {
  .okanban__lane {
    min-width: 260px;
    flex: 0 0 260px;
  }
}
@media (max-width: 768px) {
  .okanban__board {
    gap: 12px;
  }
  .okanban__lane {
    min-width: 220px;
    flex: 0 0 220px;
  }
  .okanban__cards {
    max-height: calc(100vh - 300px);
  }
}

/* ===== Non-grouped mode: flattened card container ===== */
.okanban__flat {
  padding: 0 8px;
}
.okanban__flat-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
  align-content: start;
}
.okanban__flat-empty {
  opacity: 0.6;
  font-size: 12px;
  text-align: center;
  padding: 12px 0 4px 0;
  color: var(--el-text-color-secondary);
}
</style>
