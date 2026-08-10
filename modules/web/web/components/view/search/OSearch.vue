<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="o-search">
    <div class="o-search__main" @click="focusInput">
      <el-tooltip :content="_t('Search')" placement="top">
        <el-button
          size="small"
          text
          :icon="SearchIcon"
          class="o-search__leading-btn"
          :aria-label="_t('Run search')"
          @mousedown.prevent
          @click.stop="onSearchIconClick"
        />
      </el-tooltip>

      <div class="o-search__tags">
        <el-tag
          v-if="hasGrouping"
          class="o-search__tag o-search__grouptag"
          type="success"
          effect="plain"
          closable
          round
          @click.stop="onEditGroupClick()"
          @close.stop="onGroupingClear"
          :title="groupingTooltip"
        >
          {{ _t('Group: %s', groupingSummary) }}
        </el-tag>

        <el-tag
          v-for="f in filters"
          :key="f.id"
          class="o-search__tag"
          :class="{ 'o-search__tag--pending-delete': f.id === pendingDeleteFilterId }"
          type="primary"
          effect="plain"
          closable
          round
          @close.stop="onTagClose(f.id!)"
          @click.stop="onTagClick(f.id!)"
          :title="f.name || filterTooltip(f)"
        >
          {{ f.name || summarizeFilterFields(f, 2) }}
        </el-tag>
      </div>

      <input
        ref="inputRef"
        v-model="keyword"
        class="o-search__input"
        :placeholder="placeholder"
        :name="inputName"
        :id="inputId"
        @keydown.enter.stop.prevent="onEnter"
        @keydown="onInputKeydown"
        @blur="onInputBlur"
      />

      <div class="o-search__suffix">
        <el-popover
          v-model:visible="menuVisible"
          placement="bottom-end"
          trigger="click"
          :teleported="true"
          popper-class="o-search-popover"
          :popper-style="{ width: 'auto', minWidth: 'fit-content' }"
        >
          <template #reference>
            <el-button size="small" text :icon="ArrowDown" class="o-search__trailing-btn" :aria-label="_t('Open search menu')" @click.stop />
          </template>

          <div class="o-search__menu-grid">
            <section class="o-search__menu-col">
              <div class="o-search__menu-title">{{ _t('Filters') }}</div>
              <div class="o-search__menu-list">
                <el-button v-for="it in defaultFilterItems" :key="'df:' + it.name" class="o-search__menu-item" text @click="onToggleDefaultFilter(it)">
                  <el-icon v-if="it.name && appliedFilterNameSet.has(it.name)" class="o-search__menu-icon o-search__menu-icon--applied">
                    <Check />
                  </el-icon>
                  <span class="o-search__menu-item-label">
                    {{ it.name || summarizeFilter(it.filter, 2) }}
                  </span>
                </el-button>
              </div>
              <el-divider class="o-search__menu-divider" />
              <div class="o-search__menu-subtitle">{{ _t('Favorites') }}</div>
              <div class="o-search__menu-list">
                <div v-for="it in favoriteMenuItems" :key="'fav:' + it.id" class="o-search__menu-row">
                  <el-button class="o-search__menu-item" text @click="onApplyFavorite(it)">
                    <el-icon v-if="it.name && appliedFilterNameSet.has(it.name)" class="o-search__menu-icon o-search__menu-icon--applied">
                      <Check />
                    </el-icon>
                    <span class="o-search__menu-item-label">
                      {{ it.name }}{{ it.shared ? ` (${_t('Shared')})` : '' }}
                    </span>
                  </el-button>
                  <el-button
                    v-if="it.canDelete"
                    class="o-search__menu-item-delete"
                    text
                    size="small"
                    :aria-label="_t('Delete favorite %s', it.name)"
                    @click.stop="onRemoveFavorite(it)"
                  >
                    ×
                  </el-button>
                </div>
                <div v-if="favoritesLoadError" class="o-search__empty">
                  {{ _t('Failed to load favorites') }}
                  <el-button class="o-search__menu-action" text @click="onRetryFavorites">{{ _t('Retry') }}</el-button>
                </div>
                <div v-else-if="!favoriteMenuItems.length && !favoritesLoading" class="o-search__empty">{{ _t('No favorites yet') }}</div>
              </div>
              <el-button class="o-search__menu-action" text @click="onOpenSaveFavorite">{{ _t('Save current filters…') }}</el-button>
              <el-divider class="o-search__menu-divider" />
              <el-button class="o-search__menu-action" text @click="onAddFilterClickAndClose">{{ _t('Custom filter…') }}</el-button>
            </section>

            <section class="o-search__menu-col o-search__menu-col--right">
              <div class="o-search__menu-title">{{ _t('Group by') }}</div>

              <div v-if="currentAppliedGroups.length > 0" class="o-search__menu-list">
                <el-button
                  v-for="it in appliedGroupItems"
                  :key="it.key"
                  class="o-search__menu-item"
                  text
                  @click="it.type === 'plain' ? togglePlainGroupby(it.field) : toggleTemporalGroupby(it.field, it.granularity!)"
                >
                  <el-icon class="o-search__menu-icon o-search__menu-icon--applied">
                    <Check />
                  </el-icon>
                  <span class="o-search__menu-item-label">{{ it.label }}</span>
                </el-button>
              </div>
              <div v-else class="o-search__empty">{{ _t('Not set') }}</div>

              <el-divider class="o-search__menu-divider" />

              <div class="o-search__menu-subtitle">{{ _t('Custom group by') }}</div>
              <div class="o-search__menu-list">
                <el-tree-select
                  class="o-search__tree"
                  v-model="treeSelectValue"
                  :data="groupTreeData"
                  :props="treeProps"
                  node-key="id"
                  filterable
                  clearable
                  :teleported="false"
                  :render-after-expand="false"
                  :default-expand-all="false"
                  :expand-on-click-node="true"
                  :placeholder="_t('Select field or date granularity')"
                  @change="onTreeSelectChange"
                />
              </div>
            </section>
          </div>
        </el-popover>
      </div>
    </div>

    <el-dialog v-model="isEditorOpen" :title="filterEditorTitle" append-to-body destroy-on-close @close="closeEditor(true)">
      <OSearchFilter
        v-if="draftFilter"
        :store="store"
        :draft="draftFilter"
        :fields="availableFields"
        @logic-change="(logic, gid) => setDraftLogic(logic, gid)"
        @add-group="gid => addDraftGroup(gid)"
        @remove-group="gid => removeDraftGroup(gid)"
        @add-condition="gid => addDraftCondition(gid)"
        @update-condition="updateDraftCondition"
        @remove-condition="removeDraftCondition"
        @cancel="onEditorCancel"
        @confirm="onConfirmDraft"
      />
    </el-dialog>

    <el-dialog
      v-model="saveFavoriteOpen"
      :title="saveFavoriteDialogTitle"
      append-to-body
      destroy-on-close
      width="420px"
    >
      <el-form label-position="top" @submit.prevent>
        <el-form-item :label="_t('Name')">
          <el-input v-model="saveFavoriteName" :placeholder="_t('Favorite name')" @keydown.enter.prevent="onConfirmSaveFavorite" />
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="saveFavoriteIsDefault">{{ _t('Use by default') }}</el-checkbox>
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="saveFavoriteShared">{{ _t('Share with all users') }}</el-checkbox>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="saveFavoriteOpen = false">{{ _t('Cancel') }}</el-button>
        <el-button type="primary" :loading="saveFavoriteSaving" @click="onConfirmSaveFavorite">{{ _t('Save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts" generic="T extends BaseModel">
import { ref, computed, watch, nextTick, onMounted } from 'vue';
import { Search as SearchIcon, ArrowDown, Check } from '@element-plus/icons-vue';
import type { BaseModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useSearch } from '@/web/web/composables/search';
import { normalizeFilters } from '@/web/web/query/utils/filter/structures';
import { filtersSignature, shouldApplyControlledFilters } from '@/web/web/query/utils/search/controlledFilters';
import OSearchFilter from './OSearchFilter.vue';
import {
  ElButton,
  ElTag,
  ElTooltip,
  ElDialog,
  ElDivider,
  ElIcon,
  ElPopover,
  ElTreeSelect,
  ElMessage,
  ElMessageBox,
  ElForm,
  ElFormItem,
  ElInput,
  ElCheckbox,
} from 'element-plus';
import { useDebouncedFnCancelable } from '@/web/web/composables/useDebouncedFnCancelable';
import type { GroupBySpec } from '@/core/service/api/query';
import type { ConditionGroup, QueryUpdatePayload, NamedFilter } from '@/web/web/query/types';
import { formatGroupItemForDisplay } from '@/web/web/query/utils/grouping/format';
import { normalizeGroupby } from '@/web/web/query/utils/grouping/normalize';
import { buildQueryUpdatePayload } from '@/web/web/query/utils/search/payload';
import { useFilterPresets } from '@/web/web/composables/search/useFilterPresets';
import { useSavedFilters } from '@/web/web/composables/search/useSavedFilters';
import {
  modelIdentityFromStore,
  pickDefaultFavoriteName,
  routeTitleFromLocation,
  stableTitleSource,
} from '@/web/web/composables/search/defaultFavoriteName';
import { useFilterableSearchFields } from '@/web/web/composables/search/useSearchFieldOptions';
import { useSearchGrouping, type SearchGroupByItem } from '@/web/web/composables/search/useSearchGrouping';
import { createTranslate } from '@/web/web/i18n';
import { useBreadcrumbStore } from '@/web/web/stores/breadcrumbStore';
import { useMenuStore } from '@/web/web/stores/menuStore';
import { useRoute } from 'vue-router';

const { _t } = createTranslate('web', { scope: 'web/components/view/search/OSearch' });

function trySetupHook<T>(fn: () => T): T | null {
  try {
    return fn();
  } catch {
    return null;
  }
}

/** Captured in setup so click handlers never call inject()-based APIs. */
const breadcrumbStore = trySetupHook(() => useBreadcrumbStore());
const menuStore = trySetupHook(() => useMenuStore());
const currentRoute = trySetupHook(() => useRoute());

function resolveDefaultFavoriteName(viewStore: { application?: unknown; modelName?: unknown }): string {
  const stack = breadcrumbStore?.breadcrumbStack as Array<{ title?: string; titleText?: any }> | undefined;
  const tip = Array.isArray(stack) && stack.length ? stack[stack.length - 1] : undefined;
  const breadcrumbTip = tip ? stableTitleSource(tip.title, tip.titleText) : '';
  const routeTitle = currentRoute ? routeTitleFromLocation(currentRoute) : '';
  const menu = menuStore?.activeMenu as { title?: string; titleText?: any } | null | undefined;
  const menuTitle = menu ? stableTitleSource(menu.title, menu.titleText) : '';
  return pickDefaultFavoriteName({
    breadcrumbTip,
    routeTitle,
    menuTitle,
    modelIdentity: modelIdentityFromStore(viewStore),
  });
}

/* Grouping summary labels. */
const granLabelMap = computed(() => ({
  year: _t('Year'),
  quarter: _t('Quarter'),
  month: _t('Month'),
  week: _t('Week'),
  day: _t('Day'),
}));

const props = defineProps<{
  store: WebModelStore<T>;
  placeholder?: string;
  /** Controlled grouping supplied entirely by the parent as GroupBySpec[]. */
  currentAppliedGroups?: GroupBySpec<T>[];
  /** Controlled filters supplied by the parent as the applied filter tag tree. */
  currentAppliedFilters?: ConditionGroup[];
  /** Controlled keyword supplied by the parent instead of reading local state. */
  currentKeyword?: string;
  /** Default named filters used for menu display only. */
  defaultFilters?: NamedFilter<T> | NamedFilter<T>[];
}>();

const emit = defineEmits<{
  (e: 'query-update', payload: QueryUpdatePayload): void;
  (e: 'defaults-ready', defaults: NamedFilter[]): void;
}>();
const store = props.store;
const groupingSummary = computed(() => {
  const arr = props.currentAppliedGroups ? (Array.isArray(props.currentAppliedGroups) ? props.currentAppliedGroups : [props.currentAppliedGroups]) : [];
  if (!arr.length) return '';
  return arr.map(x => formatGroupItemForDisplay(x, granLabelMap.value)).join(' > ');
});
const groupingTooltip = computed(() => groupingSummary.value);

/* Split useSearch into state, editor, actions, and helper layers. */
const { state, editor, actions, helpers } = useSearch({ attachStore: store as any });
const keyword = state.keyword;
const filters = state.filters;
// Editor layer.
const isEditorOpen = editor.isEditorOpen;
const draftFilter = editor.draftFilter;
const filterEditorTitle = computed(() => (draftFilter.value?.baseId ? _t('Edit filter') : _t('New filter')));
const openNewFilter = editor.openNew;
const openEditFilter = editor.openEdit;
const closeEditor = editor.close;
const setDraftLogic = editor.setLogic;
const addDraftGroup = editor.addGroup;
const removeDraftGroup = editor.removeGroup;
const addDraftCondition = editor.addCondition;
const updateDraftCondition = editor.updateCondition;
const removeDraftCondition = editor.removeCondition;
const saveDraft = editor.saveDraft;
const deleteFilter = editor.deleteFilter;
// Action layer.
const applyNamedFilter = actions.applyNamedFilter;
const popLastFilter = actions.popLastFilter;
// Helper layer.
const { summarizeFilter, summarizeFilterFields, filterTooltip } = helpers;

/* Group editing stays locally controlled and emits changes upward. */

const hasGrouping = computed(() => {
  const gb = props.currentAppliedGroups;
  if (!gb) return false;
  return Array.isArray(gb) ? gb.length > 0 : !!gb;
});

/* Input and helper state. */
const placeholder = computed(() => props.placeholder || _t('Search...'));
const inputName = computed(() => `${(store as any)?.storeId || 'search'}-keyword`);
const inputId = computed(() => `${inputName.value}-input`);
const inputRef = ref<HTMLInputElement | null>(null);
const pendingDeleteFilterId = ref<string | null>(null);
const menuVisible = ref(false);

/* Menu support: named presets come from a reusable composable. */
type FilterMenuItem = { name: string; filter: any };
const { defaultFilterItems, appliedFilterNameSet, toggleDefaultFilter } = useFilterPresets({
  store,
  filtersRef: filters as any,
  applyNamedFilter,
  defaultFiltersOverride: () => {
    const df = props.defaultFilters as any;
    if (!df) return undefined;
    return Array.isArray(df) ? df : [df];
  },
});

const {
  favoriteMenuItems,
  loading: favoritesLoading,
  loadError: favoritesLoadError,
  load: loadFavorites,
  apply: applyFavorite,
  saveCurrent: saveFavoriteCurrent,
  remove: removeFavorite,
  defaultsForOpen,
} = useSavedFilters({
  store,
  filtersRef: filters as any,
  keywordRef: keyword as any,
  applyNamedFilter,
  codeDefaults: () => {
    const df = props.defaultFilters as any;
    if (!df) return undefined;
    return Array.isArray(df) ? df : [df];
  },
  scopeKey: () => String(currentRoute?.path ?? ''),
});

const saveFavoriteOpen = ref(false);
const saveFavoriteName = ref('');
const saveFavoriteIsDefault = ref(false);
const saveFavoriteShared = ref(false);
const saveFavoriteSaving = ref(false);
const saveFavoriteDialogTitle = computed(() => _t('Save current filters'));

function onApplyFavorite(it: { name: string; filter: any }) {
  const before = filters.value.length;
  applyFavorite(it);
  if (filters.value.length !== before) {
    emitQueryUpdate();
  }
  menuVisible.value = false;
}

async function onRemoveFavorite(it: { id: string; name: string }) {
  try {
    await ElMessageBox.confirm(
      _t('Delete favorite "%s"? This cannot be undone.', it.name),
      _t('Confirm delete'),
      {
        type: 'warning',
        confirmButtonText: _t('Delete'),
        cancelButtonText: _t('Cancel'),
        confirmButtonClass: 'el-button--danger',
      }
    );
  } catch {
    return;
  }
  try {
    await removeFavorite(it.id);
    emit('defaults-ready', defaultsForOpen.value as NamedFilter[]);
    ElMessage.success(_t('Favorite deleted'));
  } catch (e: any) {
    ElMessage.error(e instanceof Error ? e.message : String(e));
  }
}

function onRetryFavorites() {
  void loadFavorites();
}

function onOpenSaveFavorite() {
  menuVisible.value = false;
  // Align with Odoo CustomFavoriteItem seeding, but keep Name language-stable (term src / model id).
  saveFavoriteName.value = resolveDefaultFavoriteName(store);
  saveFavoriteIsDefault.value = false;
  saveFavoriteShared.value = false;
  saveFavoriteOpen.value = true;
}

async function onConfirmSaveFavorite() {
  if (saveFavoriteSaving.value) return;
  const name = saveFavoriteName.value.trim();
  if (!name) {
    ElMessage.warning(_t('Enter a favorite name'));
    return;
  }
  saveFavoriteSaving.value = true;
  try {
    await saveFavoriteCurrent({
      name,
      isDefault: saveFavoriteIsDefault.value,
      shared: saveFavoriteShared.value,
    });
    saveFavoriteOpen.value = false;
    emit('defaults-ready', defaultsForOpen.value as NamedFilter[]);
    ElMessage.success(_t('Favorite saved'));
  } catch (e: any) {
    ElMessage.error(e instanceof Error ? e.message : String(e));
  } finally {
    saveFavoriteSaving.value = false;
  }
}

onMounted(async () => {
  await loadFavorites();
  emit('defaults-ready', defaultsForOpen.value as NamedFilter[]);
});

/* Debounced query emission. */
const lastEmittedFiltersSig = ref('');
const awaitingFiltersEcho = ref(false);
function emitQueryUpdate(payload?: QueryUpdatePayload<any>) {
  const p = payload ?? buildPayload();
  const nextSig = filtersSignature(
    normalizeFilters((Array.isArray(p.appliedFilters) ? p.appliedFilters : []) as any)
  );
  const parentSig = filtersSignature(
    normalizeFilters((Array.isArray(props.currentAppliedFilters) ? props.currentAppliedFilters : []) as any)
  );
  lastEmittedFiltersSig.value = nextSig;
  // Only await an echo when filter content diverges from the parent's current snapshot
  // (keyword/groupby-only emits must not block later external filter updates).
  if (nextSig !== parentSig) awaitingFiltersEcho.value = true;
  emit('query-update', p);
}

const debouncedTrigger = useDebouncedFnCancelable(() => {
  emitQueryUpdate();
}, 400);

// Prevent emits triggered by syncing keyword from props into local state.
const syncingKeyword = ref(false);

watch(keyword, () => {
  if (syncingKeyword.value) return;
  debouncedTrigger();
  if (pendingDeleteFilterId.value) pendingDeleteFilterId.value = null;
});

/* Toggle named filter presets. */
function onToggleDefaultFilter(it: FilterMenuItem) {
  toggleDefaultFilter(it as any, changed => {
    if (changed) emitQueryUpdate();
  });
}

/* Available filter fields (shared helper; D6 / T4.1). */
const availableFields = useFilterableSearchFields(store as any);

/* Grouping menu/tree controls. */
const {
  currentAppliedGroups,
  groupTreeData,
  appliedGroupItems,
  treeSelectValue,
  treeProps,
  togglePlainGroupby,
  toggleTemporalGroupby,
  onTreeSelectChange,
} = useSearchGrouping({
  store: store as any,
  currentAppliedGroups: () => props.currentAppliedGroups as any,
  onGroupsChange: (next: SearchGroupByItem[]) => {
    emitQueryUpdate(buildPayload(next as any));
  },
});

function onInputBlur() {
  pendingDeleteFilterId.value = null;
}

function onInputKeydown(e: KeyboardEvent) {
  if (e.key === 'Backspace') {
    const el = e.target as HTMLInputElement;
    if (el.selectionStart === 0 && el.selectionEnd === 0) {
      if (filters.value.length === 0) return;
      const lastId = filters.value[filters.value.length - 1].id;
      if (!lastId) return;
      if (pendingDeleteFilterId.value !== lastId) {
        pendingDeleteFilterId.value = lastId;
        e.preventDefault();
      } else {
        const removed = popLastFilter(true);
        if (removed) emitQueryUpdate();
        pendingDeleteFilterId.value = null;
        e.preventDefault();
      }
    } else if (pendingDeleteFilterId.value) {
      pendingDeleteFilterId.value = null;
    }
  } else if (pendingDeleteFilterId.value && e.key.length === 1) {
    pendingDeleteFilterId.value = null;
  }
}

function onEnter() {
  debouncedTrigger.cancel();
  emitQueryUpdate();
  pendingDeleteFilterId.value = null;
}

function onSearchIconClick() {
  debouncedTrigger.cancel();
  emitQueryUpdate();
  pendingDeleteFilterId.value = null;
}

function focusInput() {
  inputRef.value?.focus();
}

function onTagClick(id: string) {
  openEditFilter(id);
  pendingDeleteFilterId.value = null;
}

function onTagClose(id: string) {
  deleteFilter(id);
  emitQueryUpdate();
  if (pendingDeleteFilterId.value === id) pendingDeleteFilterId.value = null;
}

function onAddFilterClick() {
  openNewFilter();
  pendingDeleteFilterId.value = null;
}

function onEditorCancel() {
  closeEditor(true);
}

async function onConfirmDraft() {
  const draft = draftFilter.value;
  if (!draft) return;
  const editingId = draft.baseId;
  const ok = saveDraft();
  if (!ok) {
    // Edited tag disappeared (e.g. cleared while dialog open) — close without the incomplete warning.
    if (editingId && !(filters.value || []).some(f => f.id === editingId)) {
      closeEditor(true);
      return;
    }
    ElMessage.warning(_t('Add at least one complete condition before confirming'));
    return;
  }
  emitQueryUpdate();
  closeEditor(true);
  pendingDeleteFilterId.value = null;
  await nextTick();
}

function onAddFilterClickAndClose() {
  menuVisible.value = false;
  onAddFilterClick();
}

function onGroupingClear() {
  // Pass an explicit empty array so buildPayload preserves [].
  emitQueryUpdate(buildPayload([]));
}

function onEditGroupClick() {
  menuVisible.value = true;
}

// Build the normalized payload consumed by parent views such as List and Kanban.
function buildPayload(overrideAppliedGroups?: Array<GroupBySpec<any>> | SearchGroupByItem[]): QueryUpdatePayload<any> {
  const kw = keyword.value?.trim() || undefined;
  const conditionGroups: ConditionGroup[] = Array.isArray(filters.value) ? (filters.value as ConditionGroup[]) : [];
  const gbArrSrc = overrideAppliedGroups !== undefined ? overrideAppliedGroups : currentAppliedGroups.value;
  const normalized = normalizeGroupby(gbArrSrc as any) as Array<{ field: string; granularity?: any }>;
  const specs: Array<GroupBySpec<any>> = normalized.map(g => (g.granularity ? { field: g.field, granularity: g.granularity } : { field: g.field })) as any;
  const explicit = overrideAppliedGroups !== undefined;
  return buildQueryUpdatePayload<any>(kw, conditionGroups, specs, { explicitGroups: explicit });
}

// Sync the controlled keyword into local input state without triggering a query.
watch(
  () => props.currentKeyword,
  v => {
    syncingKeyword.value = true;
    keyword.value = (v ?? '') as any;
    nextTick(() => {
      syncingKeyword.value = false;
    });
  },
  { immediate: true }
);

// Sync controlled filters into local state by content signature (not length alone).
// immediate: true hydrates route-restored tags on first frame via the same echo guard.
watch(
  () => props.currentAppliedFilters,
  next => {
    if (next == null) return;
    const decision = shouldApplyControlledFilters({
      local: filters.value || [],
      incoming: next as any,
      lastEmittedSig: lastEmittedFiltersSig.value,
      awaitingEcho: awaitingFiltersEcho.value,
    });
    if (decision.acknowledged) awaitingFiltersEcho.value = false;
    if (decision.apply) {
      filters.value = decision.normalized;
      awaitingFiltersEcho.value = false;
    }
  },
  { deep: true, immediate: true }
);
</script>

<style scoped lang="scss">
.o-search {
  width: 100%;
}
.o-search__main {
  display: flex;
  align-items: center;
  gap: 2px;
  border: 1px solid var(--el-border-color);
  padding: 2px 8px;
  border-radius: 4px;
  cursor: text;
  flex-wrap: wrap;
  &:focus-within,
  &:hover {
    border-color: var(--el-color-primary-light-7);
  }
  :deep(.el-button) {
    padding: 0 6px;
    margin: 0;
  }
}
.o-search__leading-btn {
  margin-right: 2px;
}
:deep(.o-search-popover) {
  width: auto !important;
  min-width: fit-content;
  padding: 0;
}
.o-search__menu-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  padding: 12px 14px;
  max-width: 70vw;
}
.o-search__menu-col {
  min-width: 260px;
}
.o-search__menu-col--right {
  border-left: 1px solid var(--el-border-color-lighter);
  padding-left: 16px;
}
.o-search__menu-title {
  font-weight: 600;
  margin-bottom: 8px;
  color: var(--el-text-color-primary);
}
.o-search__menu-subtitle {
  font-weight: 600;
  margin: 6px 0;
  color: var(--el-text-color-regular);
}
.o-search__menu-list {
  display: flex;
  flex-direction: column;
}
.o-search__menu-row {
  display: flex;
  align-items: center;
  gap: 2px;
}
.o-search__menu-row .o-search__menu-item {
  flex: 1;
  min-width: 0;
}
.o-search__menu-item-delete {
  flex: 0 0 auto;
  opacity: 0.55;
  padding: 0 4px !important;
}
.o-search__menu-item-delete:hover {
  opacity: 1;
  color: var(--el-color-danger);
}
.o-search__menu-item {
  justify-content: flex-start;
  padding: 6px 4px;
  border-radius: 4px;
  margin: 0;
}
.o-search__menu-item:hover {
  background: var(--el-color-primary-light-9);
}
.o-search__menu-item-label {
  white-space: nowrap;
}
.o-search__menu-divider {
  margin: 10px 0;
}
.o-search__menu-action {
  justify-content: flex-start;
  padding: 6px 4px;
}
.o-search__tags {
  display: flex;
  gap: 2px;
  flex-wrap: wrap;
  align-items: center;
  max-width: 100%;
}
.o-search__tag {
  cursor: pointer;
  user-select: none;
  transition:
    background-color 0.12s ease,
    border-color 0.12s ease,
    color 0.12s ease;
  :deep(.el-tag__close) {
    color: var(--el-color-primary);
    transition:
      background-color 0.12s ease,
      color 0.12s ease;
  }
  :deep(.el-tag__close:hover) {
    background-color: var(--el-color-primary-light-8);
    color: var(--el-color-primary);
  }
}
.o-search__grouptag {
  :deep(.el-tag__close) {
    color: var(--el-color-success);
  }
  :deep(.el-tag__close:hover) {
    background-color: var(--el-color-success-light-8);
    color: var(--el-color-success);
  }
}
.o-search__tag:hover {
  background-color: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
  :deep(.el-tag__close) {
    color: var(--el-color-primary);
  }
}
.o-search__tag:active {
  background-color: var(--el-color-primary-light-8);
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}
.o-search__tag--pending-delete {
  border-color: var(--el-color-danger) !important;
  background-color: var(--el-color-danger-light-9) !important;
  color: var(--el-color-danger) !important;
  :deep(.el-tag__close) {
    color: var(--el-color-danger) !important;
  }
  :deep(.el-tag__close:hover) {
    background-color: var(--el-color-danger-light-7) !important;
    color: var(--el-color-danger) !important;
  }
}
.o-search__tag--pending-delete:hover {
  background-color: var(--el-color-danger-light-8) !important;
  border-color: var(--el-color-danger) !important;
  color: var(--el-color-danger) !important;
}
.o-search__input {
  flex: 1;
  border: none;
  outline: none;
  min-width: 140px;
  padding: 4px;
  font-size: 13px;
  background: transparent;
}
.o-search__suffix {
  display: flex;
  align-items: center;
  gap: 2px;
  margin-left: 4px;
  padding-left: 6px;
  border-left: 1px solid var(--el-border-color);
}
.o-search__menu-icon {
  margin-right: 6px;
  color: var(--el-color-success);
  font-size: 16px;
  vertical-align: -1px;
}
.o-search__empty {
  color: var(--el-text-color-secondary);
}
.o-search__tree {
  width: 260px;
}
</style>
