<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="choy-search w-full" data-anchor="choy.search">
    <div
      class="choy-search__main flex min-h-[var(--choy-control-height)] cursor-text flex-wrap items-center gap-0.5 rounded-md border border-border bg-background px-2 py-0.5 hover:border-primary-muted focus-within:border-primary-muted"
      @click="focusInput"
    >
      <span>
        <ChoyButton
          size="sm"
          variant="ghost"
          class="choy-search__leading-btn me-0.5 px-1.5 py-0"
          :aria-label="_t('Run search')"
          @mousedown.prevent
          @click.stop="onSearchIconClick"
        >
          <Search class="size-4" />
        </ChoyButton>
      </span>

      <div class="choy-search__tags flex max-w-full flex-wrap items-center gap-0.5">
        <span
          v-if="hasGrouping"
          class="choy-search__tag choy-search__grouptag inline-flex h-control cursor-pointer select-none items-center gap-1 rounded border border-border px-1.5 text-sm transition-colors hover:border-primary hover:bg-muted hover:text-primary active:border-primary active:text-primary [&_button]:text-success hover:[&_button]:text-success"
          @click.stop="onEditGroupClick()"
          :title="groupingTooltip"
        >
          {{ _t('Group: %s', groupingSummary) }}
          <button
            type="button"
            class="choy-search__tag-close m-0 size-3.5 shrink-0 cursor-pointer appearance-none rounded-full border-0 bg-transparent p-0 text-center text-xs leading-[14px] text-primary hover:bg-muted"
            :aria-label="_t('Clear grouping')"
            @click.stop="onGroupingClear"
          >
            ×
          </button>
        </span>

        <span
          v-for="f in filters"
          :key="f.id"
          class="choy-search__tag inline-flex h-control cursor-pointer select-none items-center gap-1 rounded border border-border px-1.5 text-sm transition-colors hover:border-primary hover:bg-muted hover:text-primary active:border-primary active:text-primary"
          :class="{ 'choy-search__tag--pending-delete border-danger bg-danger/10 text-danger hover:border-danger hover:text-danger [&_button]:text-danger': f.id === pendingDeleteFilterId }"
          @click.stop="onTagClick(f.id!)"
          :title="f.name || filterTooltip(f)"
        >
          {{ f.name || summarizeFilterFields(f, 2) }}
          <button
            type="button"
            class="choy-search__tag-close m-0 size-3.5 shrink-0 cursor-pointer appearance-none rounded-full border-0 bg-transparent p-0 text-center text-xs leading-[14px] text-primary hover:bg-muted"
            :aria-label="_t('Remove filter')"
            @click.stop="onTagClose(f.id!)"
          >
            ×
          </button>
        </span>
      </div>

      <input
        ref="inputRef"
        v-model="keyword"
        class="choy-search__input min-w-[140px] flex-1 border-0 bg-transparent p-1 text-sm text-foreground outline-none"
        :placeholder="placeholder"
        :name="inputName"
        :id="inputId"
        @keydown.enter.stop.prevent="onEnter"
        @keydown="onInputKeydown"
        @blur="onInputBlur"
      />

      <div class="choy-search__suffix ml-1 flex items-center gap-0.5 border-l border-border pl-1.5">
        <ChoyButton
          v-if="hasClearableSearch"
          size="sm"
          variant="ghost"
          class="choy-search__clear-btn px-1.5 py-0"
          data-testid="choy-search-clear-all"
          :aria-label="_t('Clear all')"
          @mousedown.prevent
          @click.stop="onClearAll"
        >
          {{ _t('Clear') }}
        </ChoyButton>
        <Popover v-model:open="menuVisible">
          <PopoverTrigger as-child>
            <ChoyButton size="sm" variant="ghost" class="choy-search__trailing-btn px-1.5 py-0" :aria-label="_t('Open search menu')" @click.stop>
              <ChevronDown class="size-4" />
            </ChoyButton>
          </PopoverTrigger>
          <PopoverContent class="choy-search-popover w-auto min-w-fit p-0" align="end">

          <div class="grid max-w-[70vw] grid-cols-2 gap-4 px-3.5 py-3">
            <section class="min-w-[260px]">
              <div class="mb-2 font-semibold text-foreground">{{ _t('Filters') }}</div>
              <div class="flex flex-col">
                <ChoyButton v-for="it in defaultFilterItems" :key="'df:' + it.name" class="choy-search__menu-item m-0 justify-start rounded px-1 py-1.5 hover:bg-primary-subtle" variant="ghost" @click="onToggleDefaultFilter(it)">
                  <span v-if="it.name && appliedFilterNameSet.has(it.name)" class="choy-search__menu-icon choy-search__menu-icon--applied mr-1.5 inline-flex align-[-1px] text-base text-success">
                    <Check />
                  </span>
                  <span class="whitespace-nowrap">
                    {{ it.name || summarizeFilter(it.filter, 2) }}
                  </span>
                </ChoyButton>
              </div>
              <hr class="my-2.5" />
              <div class="my-1.5 font-semibold text-foreground">{{ _t('Favorites') }}</div>
              <div class="flex flex-col">
                <div v-for="it in favoriteMenuItems" :key="'fav:' + it.id" class="flex items-center gap-0.5 [&_.choy-search__menu-item]:min-w-0 [&_.choy-search__menu-item]:flex-1">
                  <ChoyButton class="choy-search__menu-item m-0 justify-start rounded px-1 py-1.5 hover:bg-primary-subtle" variant="ghost" @click="onApplyFavorite(it)">
                    <span v-if="it.name && appliedFilterNameSet.has(it.name)" class="choy-search__menu-icon choy-search__menu-icon--applied mr-1.5 inline-flex align-[-1px] text-base text-success">
                      <Check />
                    </span>
                    <span class="whitespace-nowrap">
                      {{ it.name }}{{ it.shared ? ` (${_t('Shared')})` : '' }}
                    </span>
                  </ChoyButton>
                  <ChoyButton
                    v-if="it.canDelete"
                    class="choy-search__menu-item-edit shrink-0 px-1 py-0 opacity-55 hover:text-primary hover:opacity-100"
                    variant="ghost"
                    size="sm"
                    :aria-label="_t('Edit favorite %s', it.name)"
                    @click.stop="onEditFavorite(it)"
                  >
                    <span class="inline-flex"><Pencil class="size-3.5" /></span>
                  </ChoyButton>
                  <ChoyButton
                    v-if="it.canDelete"
                    class="choy-search__menu-item-delete shrink-0 px-1 py-0 opacity-55 hover:text-danger hover:opacity-100"
                    variant="ghost"
                    size="sm"
                    :aria-label="_t('Delete favorite %s', it.name)"
                    @click.stop="onRemoveFavorite(it)"
                  >
                    ×
                  </ChoyButton>
                </div>
                <div v-if="favoritesLoadError" class="text-muted-foreground">
                  {{ _t('Failed to load favorites') }}
                  <ChoyButton class="justify-start px-1 py-1.5" variant="ghost" @click="onRetryFavorites">{{ _t('Retry') }}</ChoyButton>
                </div>
                <div v-else-if="!favoriteMenuItems.length && !favoritesLoading" class="text-muted-foreground">{{ _t('No favorites yet') }}</div>
              </div>
              <ChoyButton class="justify-start px-1 py-1.5" variant="ghost" @click="onOpenSaveFavorite">{{ _t('Save current filters…') }}</ChoyButton>
              <hr class="my-2.5" />
              <ChoyButton class="justify-start px-1 py-1.5" variant="ghost" @click="onAddFilterClickAndClose">{{ _t('Custom filter…') }}</ChoyButton>
            </section>

            <section class="min-w-[260px] border-l border-border pl-4">
              <div class="mb-2 font-semibold text-foreground">{{ _t('Group by') }}</div>

              <div v-if="currentAppliedGroups.length > 0" class="flex flex-col">
                <ChoyButton
                  v-for="it in appliedGroupItems"
                  :key="it.key"
                  class="choy-search__menu-item m-0 justify-start rounded px-1 py-1.5 hover:bg-primary-subtle"
                  variant="ghost"
                  @click="it.type === 'plain' ? togglePlainGroupby(it.field) : toggleTemporalGroupby(it.field, it.granularity!)"
                >
                  <span class="choy-search__menu-icon choy-search__menu-icon--applied mr-1.5 inline-flex align-[-1px] text-base text-success">
                    <Check />
                  </span>
                  <span class="whitespace-nowrap">{{ it.label }}</span>
                </ChoyButton>
              </div>
              <div v-else class="text-muted-foreground">{{ _t('Not set') }}</div>

              <hr class="my-2.5" />

              <div class="my-1.5 font-semibold text-foreground">{{ _t('Custom group by') }}</div>
              <div class="flex flex-col">
                <ChoyButton
                  v-for="n in flatGroupOptions"
                  :key="n.id"
                  class="m-0 justify-start rounded px-1 py-1.5 hover:bg-primary-subtle"
                  variant="ghost"
                  @click="onTreeSelectChange(n.id)"
                >
                  {{ n.label }}
                </ChoyButton>
              </div>
            </section>
          </div>
          </PopoverContent>
        </Popover>
      </div>
    </div>

    <Dialog v-model:open="isEditorOpen">
      <DialogContent>
        <DialogTitle>{{ filterEditorTitle }}</DialogTitle>
      <SearchFilter
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
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="saveFavoriteOpen">
      <DialogContent class="max-w-md">
        <DialogTitle>{{ saveFavoriteDialogTitle }}</DialogTitle>
      <form @submit.prevent>
        <div>
          <label>{{ _t('Name') }}</label>
          <input class="fav-name" v-model="saveFavoriteName" :placeholder="_t('Favorite name')" @keydown.enter.prevent="onConfirmSaveFavorite" />
        </div>
        <div>
          <label class="fav-check"><input type="checkbox" v-model="saveFavoriteIsDefault" /> {{ _t('Use by default') }}</label>
        </div>
        <div>
          <label class="fav-check"><input type="checkbox" v-model="saveFavoriteShared" /> {{ _t('Share with all users') }}</label>
        </div>
      </form>
      <div class="mt-4 flex justify-end gap-2">
        <ChoyButton size="sm" variant="outline" @click="saveFavoriteOpen = false">{{ _t('Cancel') }}</ChoyButton>
        <ChoyButton size="sm" variant="default" :disabled="saveFavoriteSaving" @click="onConfirmSaveFavorite">{{ _t('Save') }}</ChoyButton>
      </div>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup lang="ts" generic="T extends BaseModel">
import { ref, computed, watch, nextTick, onMounted, inject } from 'vue';
import { ChoyMessage } from '../../../composables/useChoyMessage';
import { confirmChoyAction, confirmChoyChoice } from '../../../composables/confirmChoyAction';
import type { BaseModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useSearch } from '@/web/web/composables/search';
import { normalizeFilters } from '@/web/web/query/utils/filter/structures';
import { filtersSignature, shouldApplyControlledFilters } from '@/web/web/query/utils/search/controlledFilters';
import SearchFilter from './SearchFilter.vue';
import { useDebouncedFnCancelable } from '@/web/web/composables/useDebouncedFnCancelable';
import type { GroupBySpec } from '@/core/service/api/query';
import type { ConditionGroup, QueryUpdatePayload, NamedFilter } from '@/web/web/query/types';
import { formatGroupItemForDisplay } from '@/web/web/query/utils/grouping/format';
import { normalizeGroupby } from '@/web/web/query/utils/grouping/normalize';
import { buildQueryUpdatePayload } from '@/web/web/query/utils/search/payload';
import { useFilterPresets } from '@/web/web/composables/search/useFilterPresets';
import { useInjectedUserFilters } from '@/web/web/composables/search/useUserFilters';
import {
  modelIdentityFromStore,
  pickDefaultFavoriteName,
  routeTitleFromLocation,
  stableTitleSource
} from '@/web/web/composables/search/defaultFavoriteName';
import { trySetupHook } from '@/web/web/composables/search/trySetupHook';
import { SearchNavContextKey } from '@/web/web/composables/search/searchNavContext';
import { useFilterableSearchFields } from '@/web/web/composables/search/useSearchFieldOptions';
import { useSearchGrouping, type SearchGroupByItem } from '@/web/web/composables/search/useSearchGrouping';
import { createTranslate } from '@/web/web/i18n';
import { useBreadcrumbStore } from '@/web/web/stores/breadcrumbStore';
import { useMenuStore } from '@/web/web/stores/menuStore';
import { useRoute } from 'vue-router';
import { Search, ChevronDown, Check, Pencil } from 'lucide-vue-next';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import Dialog from '@/web/web/components/vendor/ui/dialog/Dialog.vue';
import DialogContent from '@/web/web/components/vendor/ui/dialog/DialogContent.vue';
import DialogTitle from '@/web/web/components/vendor/ui/dialog/DialogTitle.vue';
import Popover from '@/web/web/components/vendor/ui/popover/Popover.vue';
import PopoverContent from '@/web/web/components/vendor/ui/popover/PopoverContent.vue';
import PopoverTrigger from '@/web/web/components/vendor/ui/popover/PopoverTrigger.vue';

const { _t } = createTranslate('web', { scope: 'web/components/view/search/Search' });

/** Captured in setup so click handlers never call inject()-based APIs. */
const navCtx = inject(SearchNavContextKey, null);
const breadcrumbStore = navCtx
  ? (navCtx.breadcrumbStore ?? null)
  : trySetupHook(() => useBreadcrumbStore());
const menuStore = navCtx ? (navCtx.menuStore ?? null) : trySetupHook(() => useMenuStore());
const currentRoute = navCtx ? (navCtx.route ?? null) : trySetupHook(() => useRoute());

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
  updateMeta: updateFavoriteMeta,
  remove: removeFavorite,
  defaultsForOpen,
} = useInjectedUserFilters({
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
const editingFavoriteId = ref<string | null>(null);
const saveFavoriteName = ref('');
const saveFavoriteIsDefault = ref(false);
const saveFavoriteShared = ref(false);
const saveFavoriteSaving = ref(false);
const saveFavoriteDialogTitle = computed(() =>
  editingFavoriteId.value ? _t('Edit favorite') : _t('Save current filters')
);

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
    await confirmChoyAction(
      _t('Delete favorite "%s"? This cannot be undone.', it.name),
      _t('Confirm delete'),
      {
        confirmText: _t('Delete'),
        cancelText: _t('Cancel'),
        destructive: true,
      }
    );
  } catch {
    return;
  }
  try {
    await removeFavorite(it.id);
    emit('defaults-ready', defaultsForOpen.value as NamedFilter[]);
    ChoyMessage.success(_t('Favorite deleted'));
  } catch (e: any) {
    ChoyMessage.error(e instanceof Error ? e.message : String(e));
  }
}

function onRetryFavorites() {
  void loadFavorites();
}

function onOpenSaveFavorite() {
  menuVisible.value = false;
  editingFavoriteId.value = null;
  // Align with Odoo CustomFavoriteItem seeding, but keep Name language-stable (term src / model id).
  saveFavoriteName.value = resolveDefaultFavoriteName(store);
  saveFavoriteIsDefault.value = false;
  saveFavoriteShared.value = false;
  saveFavoriteOpen.value = true;
}

function onEditFavorite(it: { id: string; name: string; isDefault: boolean; shared: boolean }) {
  menuVisible.value = false;
  editingFavoriteId.value = it.id;
  saveFavoriteName.value = it.name;
  saveFavoriteIsDefault.value = !!it.isDefault;
  saveFavoriteShared.value = !!it.shared;
  saveFavoriteOpen.value = true;
}

async function onConfirmSaveFavorite() {
  if (saveFavoriteSaving.value) return;
  const name = saveFavoriteName.value.trim();
  if (!name) {
    ChoyMessage.warning(_t('Enter a favorite name'));
    return;
  }
  saveFavoriteSaving.value = true;
  try {
    const editId = editingFavoriteId.value;
    if (editId) {
      await updateFavoriteMeta(editId, {
        name,
        isDefault: saveFavoriteIsDefault.value,
        shared: saveFavoriteShared.value,
      });
      ChoyMessage.success(_t('Favorite updated'));
    } else {
      await saveFavoriteCurrent({
        name,
        isDefault: saveFavoriteIsDefault.value,
        shared: saveFavoriteShared.value,
      });
      ChoyMessage.success(_t('Favorite saved'));
    }
    saveFavoriteOpen.value = false;
    editingFavoriteId.value = null;
    emit('defaults-ready', defaultsForOpen.value as NamedFilter[]);
  } catch (e: any) {
    ChoyMessage.error(e instanceof Error ? e.message : String(e));
  } finally {
    saveFavoriteSaving.value = false;
  }
}

onMounted(async () => {
  // Single UserFilter Search for the Favorites menu; emit defaults so SearchView can
  // apply IsDefault on the first frame without a second Search.
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

const flatGroupOptions = computed(() => {
  const out: Array<{ id: string; label: string }> = [];
  const walk = (nodes: any[], prefix = '') => {
    for (const n of nodes || []) {
      const label = String(n.label ?? n.id ?? '');
      const id = String(n.id ?? '');
      if (id && !(n.children && n.children.length)) {
        out.push({ id, label: prefix ? `${prefix} / ${label}` : label });
      }
      if (n.children?.length) walk(n.children, prefix ? `${prefix} / ${label}` : label);
    }
  };
  walk(groupTreeData.value as any[]);
  return out;
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
    ChoyMessage.warning(_t('Add at least one complete condition before confirming'));
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

/** Clears keyword, filter chips, and grouping in one action (instant emit). */
function onClearAll() {
  debouncedTrigger.cancel();
  syncingKeyword.value = true;
  keyword.value = '' as any;
  nextTick(() => {
    syncingKeyword.value = false;
  });
  filters.value = [];
  pendingDeleteFilterId.value = null;
  emitQueryUpdate(buildPayload([]));
}

const hasClearableSearch = computed(() => {
  const hasKw = Boolean(String(keyword.value ?? '').trim());
  const hasFilters = Array.isArray(filters.value) && filters.value.length > 0;
  return hasKw || hasFilters || hasGrouping.value;
});

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

