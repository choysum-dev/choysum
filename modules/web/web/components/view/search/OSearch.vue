<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="o-search">
    <div class="o-search__main" ref="searchMainRef" @click="focusInput">
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
        @focus="onInputFocus"
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
        @save="onSaveDraft"
      />
    </el-dialog>
  </div>
</template>

<script setup lang="ts" generic="T extends BaseModel">
import { ref, computed, watch, nextTick, onMounted } from 'vue';
import { Search as SearchIcon, ArrowDown, Check } from '@element-plus/icons-vue';
import type { BaseModel } from '@/core/rpc';
import { getFieldMetadataView, type WebModelStore } from '@/web/web/stores/modelStore';
import { useSearch } from '@/web/web/composables/search';
import { normalizeFilters } from '@/web/web/query/utils/filter/structures';
import { filtersSignature, shouldApplyControlledFilters } from '@/web/web/query/utils/search/controlledFilters';
import OSearchFilter from './OSearchFilter.vue';
import { normalizeGroupby } from '@/web/web/query/utils/grouping/normalize';
import { ElButton, ElTag, ElTooltip, ElDialog, ElDivider, ElIcon, ElPopover, ElTreeSelect, ElMessage } from 'element-plus';
import { useDebouncedFnCancelable } from '@/web/web/composables/useDebouncedFnCancelable';
import type { TemporalGranularity, GroupBySpec } from '@/core/service/api/query';
import type { ConditionGroup, QueryUpdatePayload, NamedFilter } from '@/web/web/query/types';
import { parseGbString, formatGroupItemForDisplay } from '@/web/web/query/utils/grouping/format';
import { buildQueryUpdatePayload } from '@/web/web/query/utils/search/payload';
import { useGroupingOptions } from '@/web/web/composables/search/useGroupingOptions';
import { useFilterPresets } from '@/web/web/composables/search/useFilterPresets';
import { resolveFieldLabel } from '@/web/web/composables/resolveFieldLabel';
import { createTranslate, getGlobalComposer } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/view/search/OSearch' });

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
}>();
const store = props.store;
const groupingSummary = computed(() => {
  const arr = props.currentAppliedGroups ? (Array.isArray(props.currentAppliedGroups) ? props.currentAppliedGroups : [props.currentAppliedGroups]) : [];
  if (!arr.length) return '';
  return arr.map(x => formatGroupItemForDisplay(x, granLabelMap.value)).join(' > ');
});
const groupingTooltip = computed(() => groupingSummary.value);

/* Split useSearch into state, editor, actions, and helper layers. */
const { state, editor, actions, helpers } = useSearch({});
const keyword = state.keyword;
const filters = state.filters;
const hasActive = state.hasActive;
// Editor layer.
const isEditorOpen = editor.isEditorOpen;
const activeFilterId = editor.activeFilterId;
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
const clearAll = actions.clearAll;
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
const searchMainRef = ref<HTMLElement | null>(null);
const isFocused = ref(false);
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

/* Debounced query emission. */
const lastEmittedFiltersSig = ref('');
function emitQueryUpdate(payload?: QueryUpdatePayload<any>) {
  const p = payload ?? buildPayload();
  lastEmittedFiltersSig.value = filtersSignature(
    normalizeFilters((Array.isArray(p.appliedFilters) ? p.appliedFilters : []) as any)
  );
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

/* Available fields sorted by WebFieldMetadata.id, then by label (D6 / T4.1). */
const availableFields = computed(() => {
  const md = store.fieldsMetadata as Record<string, any>;
  const composer = getGlobalComposer();
  const items = Object.entries(md)
    .filter(([k, m]: any) => {
      if (k === 'DeletedAt') return false;
      if (k === 'Id') return true;
      const view = getFieldMetadataView(m);
      if (!view.isRelation) return true;
      const lowerType = String(m?.type ?? '').toLowerCase();
      return lowerType === 'manytoone' || lowerType === 'manytooneref';
    })
    .map(([k, m]: any) => ({
      prop: k,
      label: resolveFieldLabel({
        prop: k,
        meta: m,
        composer,
        fieldsGetTranslatedString: store.getFieldsGetTranslatedString?.(k),
      }),
      id: m?.id,
    })) as Array<{ prop: string; label: string; id?: string }>;

  items.sort((a, b) => {
    const idA = a.id ?? '';
    const idB = b.id ?? '';
    const cmp = idA.localeCompare(idB, 'en', { sensitivity: 'base' });
    if (cmp !== 0) return cmp;
    return a.label.localeCompare(b.label, 'en', { sensitivity: 'base' });
  });

  return items.map(i => ({ prop: i.prop, label: i.label }));
});

/* Grouping fields and controls built from a reusable composable. */
type GB = string | { field: string; granularity?: TemporalGranularity };
const {
  availableGroupFields,
  groupTreeData,
  DUMMY_ROOT_SUFFIX,
  temporalComboLabel,
  treeSelectValue,
  resetTreeSelect,
} = useGroupingOptions(store as any);

const currentAppliedGroups = computed<GB[]>(() => {
  const gbArr = props.currentAppliedGroups || [];
  const out: GB[] = [];
  for (const g of gbArr) {
    if (typeof g === 'string') {
      // Accept legacy string group definitions when the backend still returns them.
      out.push(g);
    } else if (g && typeof g === 'object') {
      const field = (g as any).field ?? (g as any).name ?? (g as any).prop;
      const granularity = (g as any).granularity ?? (g as any).gran;
      if (field) out.push(granularity ? { field, granularity } : field);
    }
  }
  return out;
});

function setGroupbyLocal(next: GB[]) {
  // Keep grouping local here and let the parent decide how to apply it.
  const normalized = normalizeGroupby(next as any);
  emitQueryUpdate(buildPayload((normalized || []) as unknown as GB[]));
}

function togglePlainGroupby(field: string) {
  const list = [...currentAppliedGroups.value];
  const i = list.findIndex(gb => (typeof gb === 'string' ? gb === field : gb.field === field && !gb.granularity));
  if (i >= 0) list.splice(i, 1);
  else list.push(field);
  setGroupbyLocal(list);
}

function toggleTemporalGroupby(field: string, gran: TemporalGranularity) {
  const list = [...currentAppliedGroups.value];
  const i = list.findIndex(gb => {
    if (typeof gb === 'string') {
      const p = parseGbString(gb);
      return !!p.granularity && p.field === field && p.granularity === gran;
    }
    return gb.field === field && (gb.granularity || '') === gran;
  });
  if (i >= 0) list.splice(i, 1);
  else list.push({ field, granularity: gran });
  setGroupbyLocal(list);
}

type AppliedGroupItem = {
  key: string;
  type: 'plain' | 'temporal';
  field: string;
  granularity?: TemporalGranularity;
  label: string;
};

const appliedGroupItems = computed<AppliedGroupItem[]>(() => {
  const items: AppliedGroupItem[] = [];
  for (const gb of currentAppliedGroups.value) {
    if (typeof gb === 'string') {
      const p = parseGbString(gb);
      if (p.granularity) {
        items.push({
          key: `cur:temp:${p.field}:${p.granularity}`,
          type: 'temporal',
          field: p.field,
          granularity: p.granularity,
          label: temporalComboLabel(p.field, p.granularity),
        });
      } else {
        const meta = availableGroupFields.value.find(f => f.prop === p.field);
        items.push({
          key: `cur:plain:${p.field}`,
          type: 'plain',
          field: p.field,
          label: meta?.label || p.field,
        });
      }
    } else {
      const field = gb.field;
      const gran = (gb.granularity || '') as TemporalGranularity | '';
      if (gran) {
        items.push({
          key: `cur:temp:${field}:${gran}`,
          type: 'temporal',
          field,
          granularity: gran as TemporalGranularity,
          label: temporalComboLabel(field, gran as TemporalGranularity),
        });
      } else {
        const meta = availableGroupFields.value.find(f => f.prop === field);
        items.push({
          key: `cur:plain:${field}`,
          type: 'plain',
          field,
          label: meta?.label || field,
        });
      }
    }
  }
  return items;
});

/* Grouping tree-select control. */
const treeProps = { value: 'value', label: 'label', children: 'children', selectable: 'selectable' } as const;
function onTreeSelectChange(v?: string) {
  if (!v) return;
  if (v.endsWith(`:${DUMMY_ROOT_SUFFIX}`)) {
    resetTreeSelect();
    return;
  }
  if (v.startsWith('f:')) {
    const field = v.slice(2);
    togglePlainGroupby(field);
  } else if (v.startsWith('d:')) {
    const [, field, gran] = v.split(':');
    if (field && gran) toggleTemporalGroupby(field, gran as any);
  }
  resetTreeSelect();
}

/* Input behavior. */
function onInputFocus() {
  isFocused.value = true;
}

function onInputBlur() {
  isFocused.value = false;
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

async function onSaveDraft() {
  const ok = saveDraft();
  if (!ok) {
    ElMessage.warning(_t('Add at least one complete condition before saving'));
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
function buildPayload(overrideAppliedGroups?: Array<GroupBySpec<any>> | GB[]): QueryUpdatePayload<any> {
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
// Ignore stale echoes of our last emit while local filters have already moved ahead.
watch(
  () => props.currentAppliedFilters,
  next => {
    if (next == null) return;
    const decision = shouldApplyControlledFilters({
      local: filters.value || [],
      incoming: next as any,
      lastEmittedSig: lastEmittedFiltersSig.value,
    });
    if (decision.apply) filters.value = decision.normalized;
  },
  { deep: true }
);

// Force one initial sync so route round-trips do not drop filter tags.
onMounted(() => {
  const cf = props.currentAppliedFilters as any;
  if (Array.isArray(cf) && cf.length > 0) {
    filters.value = normalizeFilters(cf);
  }
});
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
