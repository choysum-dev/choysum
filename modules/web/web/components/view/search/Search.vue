<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div ref="searchRootRef" class="choy-search w-full" data-anchor="choy.search">
    <Popover v-model:open="menuVisible">
      <!-- Anchor the panel to the whole search shell (not the chevron alone). -->
      <PopoverAnchor as-child>
        <InputGroup
          class="choy-search__main gap-0.5 overflow-hidden bg-card shadow-none"
          @click="focusInput"
        >
          <InputGroupAddon align="inline-start" class="choy-search__leading ps-0 pe-0">
            <ChoyButton
              size="icon"
              variant="ghost"
              class="choy-search__leading-btn shrink-0 rounded-none"
              :aria-label="_t('Run search')"
              @mousedown.prevent
              @click.stop="onSearchIconClick"
            >
              <Search class="size-4" />
            </ChoyButton>
          </InputGroupAddon>

          <div v-if="hasFacetTags" class="choy-search__tags flex max-w-full flex-nowrap items-center gap-0.5">
            <ChoyBadge
              v-if="activeFavorite"
              variant="secondary"
              class="choy-search__tag choy-search__favtag cursor-pointer gap-1 rounded-md px-1.5 py-0 text-sm font-normal"
              data-testid="choy-search-favorite-tag"
              :title="activeFavorite.name"
            >
              <Star class="size-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
              <span class="truncate">{{ activeFavorite.name }}</span>
              <button
                type="button"
                class="choy-search__tag-close m-0 inline-flex size-3.5 shrink-0 cursor-pointer items-center justify-center appearance-none rounded-full border-0 bg-transparent p-0 text-muted-foreground hover:bg-accent hover:text-foreground"
                :aria-label="_t('Clear favorite')"
                data-testid="choy-search-favorite-clear"
                @click.stop="onClearActiveFavorite"
              >
                <X class="size-3" />
              </button>
            </ChoyBadge>

            <ChoyBadge
              v-if="hasVisibleGrouping"
              variant="secondary"
              class="choy-search__tag choy-search__grouptag cursor-pointer gap-1 rounded-md px-1.5 py-0 text-sm font-normal"
              :title="groupingTooltip"
              @click.stop="onEditGroupClick()"
            >
              <Layers class="size-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
              <span class="truncate">{{ groupingSummary }}</span>
              <button
                v-if="canClearVisibleGrouping"
                type="button"
                class="choy-search__tag-close m-0 inline-flex size-3.5 shrink-0 cursor-pointer items-center justify-center appearance-none rounded-full border-0 bg-transparent p-0 text-muted-foreground hover:bg-accent hover:text-foreground"
                :aria-label="_t('Clear grouping')"
                data-testid="choy-search-groupby-clear"
                @click.stop="onGroupingClear"
              >
                <X class="size-3" />
              </button>
            </ChoyBadge>

            <ChoyBadge
              v-for="f in visibleFilters"
              :key="f.id"
              :variant="f.id === pendingDeleteFilterId ? 'destructive' : 'secondary'"
              class="choy-search__tag cursor-pointer gap-1 rounded-md px-1.5 py-0 text-sm font-normal"
              :class="{ 'choy-search__tag--pending-delete': f.id === pendingDeleteFilterId }"
              :title="f.name || filterTooltip(f)"
              @click.stop="onTagClick(f.id!)"
            >
              <Funnel class="size-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
              <span class="truncate">{{ f.name || summarizeFilterFields(f, 2) }}</span>
              <button
                type="button"
                class="choy-search__tag-close m-0 inline-flex size-3.5 shrink-0 cursor-pointer items-center justify-center appearance-none rounded-full border-0 bg-transparent p-0 text-muted-foreground hover:bg-accent hover:text-foreground"
                :aria-label="_t('Remove filter')"
                @click.stop="onTagClose(f.id!)"
              >
                <X class="size-3" />
              </button>
            </ChoyBadge>
          </div>

          <InputGroupInput
            ref="inputRef"
            v-model="keyword"
            class="choy-search__input min-w-[140px] px-1"
            :placeholder="placeholder"
            :name="inputName"
            :id="inputId"
            @focus="onInputFocus"
            @keydown.enter.stop.prevent="onEnter"
            @keydown="onInputKeydown"
            @blur="onInputBlur"
          />

          <InputGroupAddon align="inline-end" class="choy-search__suffix gap-0 border-l border-border ps-0">
            <ChoyButton
              v-if="hasClearableSearch"
              size="icon"
              variant="ghost"
              class="choy-search__clear-btn shrink-0 rounded-none"
              data-testid="choy-search-clear-all"
              :aria-label="_t('Clear all')"
              @mousedown.prevent
              @click.stop="onClearAll"
            >
              <X class="size-4" />
            </ChoyButton>
            <ChoyButton
              size="icon"
              variant="ghost"
              class="choy-search__trailing-btn shrink-0 rounded-none"
              data-testid="choy-search-menu-toggle"
              :aria-label="_t('Toggle search menu')"
              :aria-expanded="menuVisible"
              @mousedown.prevent
              @click.stop="onToggleSearchMenu"
            >
              <ChevronDown class="size-4" />
            </ChoyButton>
          </InputGroupAddon>
        </InputGroup>
      </PopoverAnchor>

      <PopoverContent
        class="choy-search-popover p-0"
        align="start"
        side="bottom"
        :side-offset="0"
        @open-auto-focus.prevent
        @close-auto-focus.prevent
        @pointer-down-outside="onMenuDismissOutside"
        @focus-outside="onMenuDismissOutside"
      >
        <div class="choy-search-menu" data-slot="search-menu">
          <section class="choy-search-menu__col" data-slot="search-filters">
            <div class="choy-search-menu__heading">
              <Funnel class="size-3.5 shrink-0" aria-hidden="true" />
              {{ _t('Filters') }}
            </div>
            <div class="choy-search-menu__col-body">
              <ChoyButton
                v-for="it in defaultFilterItems"
                :key="'df:' + it.name"
                class="choy-search__menu-item"
                variant="ghost"
                @click="onToggleDefaultFilter(it)"
              >
                <span
                  v-if="it.name && appliedFilterNameSet.has(it.name)"
                  class="choy-search__menu-icon choy-search__menu-icon--applied"
                >
                  <Check class="size-3.5" />
                </span>
                <span class="truncate">
                  {{ it.name || summarizeFilter(it.filter, 2) }}
                </span>
              </ChoyButton>
              <div v-if="!defaultFilterItems.length" class="choy-search-menu__empty">
                {{ _t('No preset filters') }}
              </div>
            </div>
            <div class="choy-search-menu__footer">
              <ChoyButton class="choy-search__menu-item" variant="ghost" @click="onAddFilterClickAndClose">
                {{ _t('Custom filter…') }}
              </ChoyButton>
            </div>
          </section>

          <section class="choy-search-menu__col choy-search-menu__col--groupby" data-slot="search-groupby">
            <div class="choy-search-menu__heading">
              <Layers class="size-3.5 shrink-0" aria-hidden="true" />
              {{ _t('Group by') }}
            </div>

            <div v-if="currentAppliedGroups.length > 0" class="choy-search-menu__applied">
              <ChoyButton
                v-for="it in appliedGroupItems"
                :key="it.key"
                class="choy-search__menu-item"
                variant="ghost"
                @click="
                  it.type === 'plain'
                    ? togglePlainGroupby(it.field)
                    : toggleTemporalGroupby(it.field, it.granularity!)
                "
              >
                <span class="choy-search__menu-icon choy-search__menu-icon--applied">
                  <Check class="size-3.5" />
                </span>
                <span class="truncate">{{ it.label }}</span>
              </ChoyButton>
            </div>
            <div v-else class="choy-search-menu__empty">{{ _t('Not set') }}</div>

            <div class="choy-search-menu__subheading">{{ _t('Custom group by') }}</div>
            <Combobox
              v-model="groupFieldSelect"
              v-model:open="groupFieldOpen"
              ignore-filter
              open-on-focus
              open-on-click
              class="choy-search__group-combobox w-full"
              @update:model-value="onGroupFieldSelect"
            >
              <ComboboxAnchor class="w-full">
                <ComboboxInput
                  v-model="groupFieldQuery"
                  class="choy-search__group-select"
                  data-testid="choy-search-groupby-select"
                  :placeholder="_t('Select a field…')"
                  :aria-label="_t('Select a field…')"
                  @focus="groupFieldOpen = true"
                />
              </ComboboxAnchor>
              <ComboboxContent class="choy-search__group-select-content p-0">
                <div class="choy-search__group-select-list max-h-56 overflow-y-auto p-1">
                  <div
                    v-if="!filteredGroupOptions.length"
                    class="px-2 py-1.5 text-sm text-muted-foreground"
                  >
                    {{ _t('No fields') }}
                  </div>
                  <ComboboxItem
                    v-for="n in filteredGroupOptions"
                    :key="n.id"
                    :value="n.id"
                    class="choy-search__group-select-item"
                  >
                    {{ n.label }}
                  </ComboboxItem>
                </div>
              </ComboboxContent>
            </Combobox>
          </section>

          <section class="choy-search-menu__col choy-search-menu__col--favorites" data-slot="search-favorites">
            <div class="choy-search-menu__heading">
              <Star class="size-3.5 shrink-0" aria-hidden="true" />
              {{ _t('Favorites') }}
            </div>
            <div class="choy-search-menu__col-body">
              <div
                v-for="it in favoriteMenuItems"
                :key="'fav:' + it.id"
                class="choy-search__favorite-row"
              >
                <ChoyButton
                  class="choy-search__menu-item choy-search__menu-item--grow"
                  variant="ghost"
                  @click="onApplyFavorite(it)"
                >
                  <span
                    v-if="activeFavorite?.id === it.id"
                    class="choy-search__menu-icon choy-search__menu-icon--applied"
                  >
                    <Check class="size-3.5" />
                  </span>
                  <span class="truncate">
                    {{ it.name }}{{ it.shared ? ` (${_t('Shared')})` : '' }}
                  </span>
                </ChoyButton>
                <ChoyButton
                  v-if="it.canDelete"
                  class="choy-search__menu-item-edit"
                  variant="ghost"
                  size="sm"
                  :aria-label="_t('Edit favorite %s', it.name)"
                  @click.stop="onEditFavorite(it)"
                >
                  <Pencil class="size-3.5" />
                </ChoyButton>
                <ChoyButton
                  v-if="it.canDelete"
                  class="choy-search__menu-item-delete"
                  variant="ghost"
                  size="sm"
                  :aria-label="_t('Delete favorite %s', it.name)"
                  @click.stop="onRemoveFavorite(it)"
                >
                  <X class="size-3.5" />
                </ChoyButton>
              </div>
              <div v-if="favoritesLoadError" class="choy-search-menu__status choy-search-menu__status--error">
                <span>{{ _t('Failed to load favorites') }}</span>
                <button type="button" class="choy-search-menu__link" @click="onRetryFavorites">
                  {{ _t('Retry') }}
                </button>
              </div>
              <div
                v-else-if="!favoriteMenuItems.length && !favoritesLoading"
                class="choy-search-menu__empty"
              >
                {{ _t('No favorites yet') }}
              </div>
            </div>
            <div class="choy-search-menu__footer">
              <ChoyButton class="choy-search__menu-item" variant="ghost" @click="onOpenSaveFavorite">
                {{ _t('Save current filters…') }}
              </ChoyButton>
            </div>
          </section>
        </div>
      </PopoverContent>
    </Popover>

    <ChoyDialog v-model:open="isEditorOpen">
      <ChoyDialogContent class="max-w-5xl gap-5">
        <ChoyDialogTitle>{{ filterEditorTitle }}</ChoyDialogTitle>
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
      </ChoyDialogContent>
    </ChoyDialog>

    <ChoyDialog v-model:open="saveFavoriteOpen">
      <ChoyDialogContent class="max-w-md">
        <ChoyDialogTitle>{{ saveFavoriteDialogTitle }}</ChoyDialogTitle>
      <form class="flex flex-col gap-3" @submit.prevent>
        <div class="flex flex-col gap-1.5">
          <label class="text-sm font-medium text-foreground" for="choy-search-fav-name">{{ _t('Name') }}</label>
          <Input
            id="choy-search-fav-name"
            class="fav-name"
            v-model="saveFavoriteName"
            :placeholder="_t('Favorite name')"
            @keydown.enter.prevent="onConfirmSaveFavorite"
          />
        </div>
        <label class="fav-check flex items-center gap-2 text-sm text-foreground">
          <Checkbox v-model="saveFavoriteIsDefault" />
          {{ _t('Use by default') }}
        </label>
        <label class="fav-check flex items-center gap-2 text-sm text-foreground">
          <Checkbox v-model="saveFavoriteShared" />
          {{ _t('Share with all users') }}
        </label>
      </form>
      <div class="mt-4 flex justify-end gap-2">
        <ChoyButton size="sm" variant="outline" @click="saveFavoriteOpen = false">{{ _t('Cancel') }}</ChoyButton>
        <ChoyButton size="sm" variant="default" :disabled="saveFavoriteSaving" @click="onConfirmSaveFavorite">{{ _t('Save') }}</ChoyButton>
      </div>
      </ChoyDialogContent>
    </ChoyDialog>
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
import {
  useInjectedUserFilters,
  type UserFavoriteMenuItem,
} from '@/web/web/composables/search/useUserFilters';
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
import { mergeFavoriteGroups, removeFavoriteGroups } from '@/web/web/composables/search/mergeFavoriteGroups';
import { createTranslate } from '@/web/web/i18n';
import { useBreadcrumbStore } from '@/web/web/stores/breadcrumbStore';
import { useMenuStore } from '@/web/web/stores/menuStore';
import { useRoute } from 'vue-router';
import { Search, ChevronDown, Check, Pencil, X, Funnel, Layers, Star } from 'lucide-vue-next';
import ChoyBadge from '@/web/web/components/layout/ChoyBadge.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from '@/web/web/components/layout/choyDialog';
import Checkbox from '@/web/web/components/vendor/ui/checkbox/Checkbox.vue';
import Input from '@/web/web/components/vendor/ui/input/Input.vue';
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from '@/web/web/components/vendor/ui/input-group';
import Popover from '@/web/web/components/vendor/ui/popover/Popover.vue';
import PopoverAnchor from '@/web/web/components/vendor/ui/popover/PopoverAnchor.vue';
import PopoverContent from '@/web/web/components/vendor/ui/popover/PopoverContent.vue';
import Combobox from '@/web/web/components/vendor/ui/combobox/Combobox.vue';
import ComboboxAnchor from '@/web/web/components/vendor/ui/combobox/ComboboxAnchor.vue';
import ComboboxContent from '@/web/web/components/vendor/ui/combobox/ComboboxContent.vue';
import ComboboxInput from '@/web/web/components/vendor/ui/combobox/ComboboxInput.vue';
import ComboboxItem from '@/web/web/components/vendor/ui/combobox/ComboboxItem.vue';

const { _t } = createTranslate('web', { scope: 'web/components/view/search/Search' });

type ActiveFavorite = {
  id: string;
  name: string;
  filter: unknown;
  groups: Array<{ field: string; granularity?: string }>;
  /** Chip id contributed by apply; omitted when the favorite added no named chip. */
  filterChipId?: string;
};

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
  /**
   * Group-by field names that cannot be cleared (e.g. kanban board lane field).
   * Still shown as a facet; clear / clear-all leave them applied.
   */
  lockedGroups?: string[];
}>();

const emit = defineEmits<{
  (e: 'query-update', payload: QueryUpdatePayload): void;
  (e: 'defaults-ready', defaults: NamedFilter[]): void;
}>();
const store = props.store;

/* Split useSearch into state, editor, actions, and helper layers. */
const { state, editor, actions, helpers } = useSearch({ attachStore: store as any });
const keyword = state.keyword;
const filters = state.filters;
// Editor layer.
const isEditorOpen = editor.isEditorOpen;
const draftFilter = editor.draftFilter;
const filterEditorTitle = computed(() =>
  draftFilter.value?.baseId ? _t('Edit filter') : _t('Custom filter')
);
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

/** Active Favorites facet (Odoo-style star tag); at most one at a time. */
const activeFavorite = ref<ActiveFavorite | null>(null);

/**
 * Groups not contributed by the active favorite (user-stacked).
 * Favorite groupbys stay applied but are folded into the star tag, like Odoo.
 */
const userFacingGroups = computed(() => {
  const fav = activeFavorite.value;
  if (!fav?.groups?.length) {
    const gb = props.currentAppliedGroups;
    if (!gb) return [] as Array<{ field: string; granularity?: string }>;
    return (normalizeGroupby(gb as any) as Array<{ field: string; granularity?: string }>).slice();
  }
  return removeFavoriteGroups(props.currentAppliedGroups as any, fav.groups);
});

const hasVisibleGrouping = computed(() => userFacingGroups.value.length > 0);

const lockedGroupFieldSet = computed(
  () => new Set((props.lockedGroups || []).map((f) => String(f || '').trim()).filter(Boolean)),
);

function isLockedGroupField(field: string): boolean {
  return lockedGroupFieldSet.value.has(String(field || '').trim());
}

/** True when the grouping facet has at least one removable (non-locked) group. */
const canClearVisibleGrouping = computed(() =>
  userFacingGroups.value.some((g) => !isLockedGroupField(String(g.field || ''))),
);

/** Re-inject locked group fields so board lanes cannot collapse to a flat list. */
function ensureLockedGroups(
  groups: Array<GroupBySpec<any> | SearchGroupByItem> | undefined | null,
): GroupBySpec<any>[] {
  const normalized = (normalizeGroupby((groups || []) as any) || []) as GroupBySpec<any>[];
  const fields = new Set(
    normalized.map((g) => String((typeof g === 'string' ? g : g.field) || '').trim()),
  );
  const next = normalized.slice();
  for (const field of lockedGroupFieldSet.value) {
    if (!fields.has(field)) next.unshift({ field } as GroupBySpec<any>);
  }
  return next;
}

const groupingSummary = computed(() => {
  const arr = userFacingGroups.value;
  if (!arr.length) return '';
  return arr.map(x => formatGroupItemForDisplay(x, granLabelMap.value)).join(' > ');
});
const groupingTooltip = computed(() => groupingSummary.value);

/** Funnel chips excluding the active favorite's own named condition chip. */
const visibleFilters = computed(() => {
  const list = Array.isArray(filters.value) ? filters.value : [];
  const fav = activeFavorite.value;
  if (!fav) return list;
  if (fav.filterChipId) {
    return list.filter(f => f.id !== fav.filterChipId);
  }
  // Groupby-only / empty favorites contribute no chip; leave other same-name filters.
  return list;
});

const hasFacetTags = computed(
  () => !!activeFavorite.value || hasVisibleGrouping.value || visibleFilters.value.length > 0
);

/* Input and helper state. */
const placeholder = computed(() => props.placeholder || _t('Search...'));
const inputName = computed(() => `${(store as any)?.storeId || 'search'}-keyword`);
const inputId = computed(() => `${inputName.value}-input`);
const inputRef = ref<{ $el?: HTMLInputElement } | HTMLInputElement | null>(null);
const pendingDeleteFilterId = ref<string | null>(null);
const menuVisible = ref(false);
/** Transient Combobox value for custom group-by; cleared after apply / menu close. */
const groupFieldSelect = ref<string>('');
const groupFieldQuery = ref('');
const groupFieldOpen = ref(false);


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
  getAppliedGroups: () => (props.currentAppliedGroups as any) ?? [],
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

/** Remove only the chip this favorite contributed (by id); no-op when none was added. */
function removeActiveFavoriteChip(fav: ActiveFavorite) {
  const chipId = fav.filterChipId;
  if (!chipId) return;
  filters.value = (filters.value || []).filter(f => f.id !== chipId);
}

function onApplyFavorite(it: UserFavoriteMenuItem) {
  const prev = activeFavorite.value;
  const filtersSnapshot = (filters.value || []).slice();
  let baseGroups = (props.currentAppliedGroups as any) ?? [];
  if (prev) {
    // Replace previous favorite contribution; keep user-added filters/groups.
    removeActiveFavoriteChip(prev);
    baseGroups = removeFavoriteGroups(baseGroups, prev.groups);
  }
  const favGroups = normalizeGroupby((it.groups ?? []) as any) as Array<{
    field: string;
    granularity?: string;
  }>;
  const beforeIds = new Set((filters.value || []).map(f => f.id));
  try {
    // Apply condition into query (named chip may exist); UI folds it into the star tag.
    applyFavorite({ name: it.name, filter: it.filter });
  } catch (e) {
    // Roll back chip removal so local filters still match the applied query.
    filters.value = filtersSnapshot;
    ChoyMessage.error(e instanceof Error ? e.message : String(e));
    return;
  }
  const added = (filters.value || []).find(f => f.id && !beforeIds.has(f.id));
  activeFavorite.value = {
    id: String(it.id),
    name: String(it.name || '').trim(),
    filter: it.filter,
    groups: favGroups,
    ...(added?.id ? { filterChipId: String(added.id) } : {}),
  };
  const merged = mergeFavoriteGroups(baseGroups, favGroups);
  emitQueryUpdate(buildPayload(merged as any));
  menuVisible.value = false;
}

function onClearActiveFavorite() {
  const prev = activeFavorite.value;
  if (!prev) return;
  removeActiveFavoriteChip(prev);
  const nextGroups = removeFavoriteGroups(props.currentAppliedGroups as any, prev.groups);
  activeFavorite.value = null;
  emitQueryUpdate(buildPayload(nextGroups as any));
}

async function onRemoveFavorite(it: { id: string; name: string }) {
  // Close the search menu before the confirm dialog (same as Edit/Save) so the
  // popover is not dismissed as a focus side-effect of AlertDialog.
  const reopenMenu = menuVisible.value;
  menuVisible.value = false;
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
    if (reopenMenu) menuVisible.value = true;
    return;
  }
  try {
    await removeFavorite(it.id);
    if (activeFavorite.value?.id === it.id) onClearActiveFavorite();
    emit('defaults-ready', defaultsForOpen.value as NamedFilter[]);
    ChoyMessage.success(_t('Favorite deleted'));
  } catch (e: any) {
    ChoyMessage.error(e instanceof Error ? e.message : String(e));
  } finally {
    if (reopenMenu) menuVisible.value = true;
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
  lockedGroupFields: () => props.lockedGroups,
  onGroupsChange: (next: SearchGroupByItem[]) => {
    emitQueryUpdate(buildPayload(ensureLockedGroups(next)));
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

const filteredGroupOptions = computed(() => {
  const q = groupFieldQuery.value.trim().toLowerCase();
  if (!q) return flatGroupOptions.value;
  return flatGroupOptions.value.filter(
    (n) => n.label.toLowerCase().includes(q) || n.id.toLowerCase().includes(q),
  );
});

watch(menuVisible, (open) => {
  if (!open) {
    groupFieldSelect.value = '';
    groupFieldQuery.value = '';
    groupFieldOpen.value = false;
  }
});

function onGroupFieldSelect(id: string | null | undefined): void {
  if (id == null || id === '') return;
  onTreeSelectChange(id);
  groupFieldSelect.value = '';
  groupFieldQuery.value = '';
  groupFieldOpen.value = false;
  menuVisible.value = false;
}

const searchRootRef = ref<HTMLElement | null>(null);
/** Ignore DismissableLayer outside events right after opening (same click/focus that opened). */
let suppressMenuDismissUntil = 0;

function markMenuDismissSuppressed(ms = 300) {
  suppressMenuDismissUntil = Date.now() + ms;
}

function isEventFromSearchShell(event: Event): boolean {
  const detail = (event as CustomEvent<{ originalEvent?: Event }>).detail;
  const original = (detail?.originalEvent ?? event) as Event & {
    composedPath?: () => EventTarget[];
    target?: EventTarget | null;
  };
  const root = searchRootRef.value;
  if (!root) return false;
  if (typeof original.composedPath === 'function') {
    try {
      if (original.composedPath().includes(root)) return true;
    } catch {
      /* ignore */
    }
  }
  const target = original.target as { closest?: (s: string) => Element | null } | null;
  if (target && typeof target.closest === 'function') {
    try {
      if (target.closest('[data-anchor="choy.search"]')) return true;
    } catch {
      /* ignore */
    }
  }
  let n: { parentNode?: unknown; parentElement?: unknown } | null = target as any;
  while (n) {
    if (n === root) return true;
    n = ((n as any).parentNode ?? (n as any).parentElement ?? null) as any;
  }
  return false;
}

/** Prevent popover dismiss when interacting with the search shell (input / tags / chevron). */
function onMenuDismissOutside(event: Event) {
  if (Date.now() < suppressMenuDismissUntil || isEventFromSearchShell(event)) {
    event.preventDefault();
  }
}

function onInputFocus() {
  markMenuDismissSuppressed();
  menuVisible.value = true;
}

function onToggleSearchMenu() {
  if (!menuVisible.value) markMenuDismissSuppressed();
  menuVisible.value = !menuVisible.value;
}

function onInputBlur() {
  pendingDeleteFilterId.value = null;
}

function onInputKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && menuVisible.value) {
    menuVisible.value = false;
    e.preventDefault();
    return;
  }
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
  const r = inputRef.value as { $el?: HTMLInputElement; focus?: () => void } | HTMLInputElement | null;
  if (!r) return;
  markMenuDismissSuppressed();
  if (typeof (r as HTMLInputElement).focus === 'function' && !(r as { $el?: unknown }).$el) {
    (r as HTMLInputElement).focus();
    // Focus may be a no-op when already focused; still open the menu (Odoo-style).
    menuVisible.value = true;
    return;
  }
  const el = (r as { $el?: HTMLInputElement }).$el;
  if (el && typeof el.focus === 'function') el.focus();
  menuVisible.value = true;
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
  if (!canClearVisibleGrouping.value) return;
  // Clear removable user-facing groups; keep favorite contribution and locked board groups.
  const favGroups = (activeFavorite.value?.groups ?? []) as SearchGroupByItem[];
  emitQueryUpdate(buildPayload(ensureLockedGroups(favGroups)));
}

/** Clears keyword, filter chips, favorite facet, and removable grouping in one action. */
function onClearAll() {
  if (!hasClearableSearch.value) return;
  debouncedTrigger.cancel();
  syncingKeyword.value = true;
  keyword.value = '' as any;
  nextTick(() => {
    syncingKeyword.value = false;
  });
  filters.value = [];
  activeFavorite.value = null;
  pendingDeleteFilterId.value = null;
  emitQueryUpdate(buildPayload(ensureLockedGroups([])));
}

const hasClearableSearch = computed(() => {
  const hasKw = Boolean(String(keyword.value ?? '').trim());
  const hasFilters = Array.isArray(filters.value) && filters.value.length > 0;
  return hasKw || hasFilters || canClearVisibleGrouping.value || !!activeFavorite.value;
});

function onEditGroupClick() {
  markMenuDismissSuppressed();
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

<!--
  Teleported PopoverContent is outside this SFC's scope tree, so menu layout
  cannot rely on scoped selectors or on md:* utilities missing from FullCSS.
-->
<style>
.choy-search-popover[data-slot='popover-content'] {
  /* Compact panel — do not stretch to the full search-shell width. */
  width: min(90vw, 36rem) !important;
  min-width: min(90vw, 22rem);
  max-width: 90vw;
  padding: 0;
  overflow: hidden;
}

.choy-search-popover .choy-search-menu {
  display: grid;
  grid-template-columns: 1fr;
  gap: 0;
  max-height: min(70vh, 24rem);
  overflow: auto;
}

.choy-search-popover .choy-search-menu__col {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  padding: 0.75rem 0.875rem;
}

.choy-search-popover .choy-search-menu__col--groupby,
.choy-search-popover .choy-search-menu__col--favorites {
  border-top: 1px solid var(--border);
}

.choy-search-popover .choy-search-menu__heading {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  min-height: 1.25rem;
  margin: 0;
  font-size: 0.8125rem;
  font-weight: 600;
  line-height: 1.25;
  color: var(--foreground);
}

.choy-search-popover .choy-search-menu__heading svg {
  color: var(--muted-foreground);
}

.choy-search-popover .choy-search-menu__subheading {
  margin: 0.5rem 0 0.25rem;
  font-size: 0.6875rem;
  font-weight: 600;
  letter-spacing: 0.02em;
  text-transform: uppercase;
  color: var(--muted-foreground);
}

.choy-search-popover .choy-search-menu__empty {
  padding: 0.25rem 0.125rem;
  font-size: 0.8125rem;
  color: var(--muted-foreground);
}

.choy-search-popover .choy-search-menu__status {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0.375rem;
  padding: 0.125rem 0;
  font-size: 0.8125rem;
}

.choy-search-popover .choy-search-menu__status--error {
  background: transparent;
  color: var(--muted-foreground);
}

.choy-search-popover .choy-search-menu__link {
  appearance: none;
  margin: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--primary);
  font: inherit;
  font-size: inherit;
  text-decoration: underline;
  text-underline-offset: 2px;
  cursor: pointer;
}

.choy-search-popover .choy-search-menu__col-body {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  max-height: 12rem;
  overflow-x: hidden;
  overflow-y: auto;
}

.choy-search-popover .choy-search-menu__footer {
  margin-top: 0.375rem;
  padding-top: 0.375rem;
  border-top: 1px solid color-mix(in oklab, var(--border) 80%, transparent);
}

.choy-search-popover .choy-search__menu-item {
  justify-content: flex-start;
  margin: 0;
  height: auto;
  min-height: 1.75rem;
  padding: 0.25rem 0.375rem;
  border-radius: calc(var(--radius-md) - 2px);
  font-size: 0.8125rem;
  font-weight: 400;
}

.choy-search-popover .choy-search__menu-item--grow {
  flex: 1;
  min-width: 0;
}

.choy-search-popover .choy-search__menu-icon {
  display: inline-flex;
  margin-inline-end: 0.375rem;
  color: var(--success, var(--primary));
}

.choy-search-popover .choy-search__favorite-row {
  display: flex;
  align-items: center;
  gap: 0.125rem;
}

.choy-search-popover .choy-search__menu-item-edit,
.choy-search-popover .choy-search__menu-item-delete {
  height: 1.75rem;
  width: 1.75rem;
  padding: 0;
  opacity: 0.55;
}

.choy-search-popover .choy-search__menu-item-edit:hover {
  opacity: 1;
}

.choy-search-popover .choy-search__menu-item-delete:hover {
  opacity: 1;
  color: var(--destructive);
}

.choy-search-popover .choy-search__group-select {
  height: 2rem;
  min-height: 2rem;
  width: 100%;
  padding-inline: 0.5rem;
  font-size: 0.8125rem;
  box-shadow: none;
}

.choy-search-popover .choy-search__group-select:focus-visible {
  border-color: var(--border);
  box-shadow: none;
  outline: none;
}

.choy-search__group-select-content {
  z-index: 60;
}

.choy-search__group-select-item {
  font-size: 0.8125rem;
}

@media (min-width: 48rem) {
  .choy-search-popover .choy-search-menu {
    grid-template-columns: minmax(9.5rem, 1fr) minmax(12rem, 1.2fr) minmax(9.5rem, 1fr);
    align-items: stretch;
    overflow: hidden;
  }

  .choy-search-popover .choy-search-menu__col {
    min-width: 9.5rem;
  }

  .choy-search-popover .choy-search-menu__col--groupby,
  .choy-search-popover .choy-search-menu__col--favorites {
    border-top: 0;
    border-inline-start: 1px solid var(--border);
  }

  .choy-search-popover .choy-search-menu__col-body {
    max-height: 14rem;
  }
}
</style>

<style scoped>
/*
  Stretch every shell column to the control height and center inside it, so
  icons stay put when facet tags appear or clear. Kit InputGroup ships
  h-control + addon py-1.5; cn cannot twMerge those away.
*/
.choy-search__main {
  height: var(--height-control) !important;
  min-height: var(--height-control) !important;
  flex-wrap: nowrap;
  align-items: stretch;
}

.choy-search__main :deep([data-slot='input-group-addon']) {
  display: inline-flex;
  align-items: center;
  align-self: stretch;
  height: auto !important;
  min-height: 0;
  padding-block: 0 !important;
  margin: 0;
}

.choy-search__main :deep(.choy-search__leading) {
  padding-inline: 0 !important;
  margin-inline-start: 0;
}

.choy-search__main :deep(.choy-search__suffix) {
  padding-inline: 0 !important;
  margin-inline-end: 0;
}

.choy-search__tags {
  align-self: stretch;
  display: flex;
  align-items: center;
  margin: 0;
  padding: 0;
  min-width: 0;
  line-height: 1;
}

.choy-search__main :deep([data-slot='input-group-control'].choy-search__input) {
  align-self: stretch;
  height: auto !important;
  min-height: 0 !important;
  padding-block: 0 !important;
  line-height: 1.25;
  border-width: 0;
  box-shadow: none;
}

.choy-search__main :deep(.choy-search__leading-btn),
.choy-search__main :deep(.choy-search__trailing-btn),
.choy-search__main :deep(.choy-search__clear-btn) {
  height: 100%;
  width: var(--height-control);
  min-height: 0;
  padding: 0;
  border-radius: 0;
}

.choy-search__main :deep(.choy-search__leading-btn svg),
.choy-search__main :deep(.choy-search__trailing-btn svg),
.choy-search__main :deep(.choy-search__clear-btn svg) {
  width: 1rem;
  height: 1rem;
}

.choy-search__main :deep(.choy-search__tag) {
  box-sizing: border-box;
  display: inline-flex;
  align-items: center;
  height: calc(var(--height-control) - 8px);
  max-height: calc(var(--height-control) - 8px);
  margin: 0;
  padding-block: 0 !important;
  line-height: 1;
}

.choy-search__main :deep([data-slot='popover-trigger']) {
  display: inline-flex;
  align-items: center;
  align-self: stretch;
  height: auto;
  line-height: 0;
}
</style>

