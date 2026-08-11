<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OSearch
    :store="store"
    :placeholder="effectivePlaceholder"
    :current-keyword="keywordForChild"
    :current-applied-filters="appliedFiltersForChild"
    :current-applied-groups="appliedGroupsForChild"
    :default-filters="codeDefaultFilters"
    @query-update="onQueryUpdate"
    @defaults-ready="onDefaultsReady"
  />
</template>

<script setup lang="ts" generic="T extends BaseModel">
import { computed, onMounted, ref, nextTick } from 'vue';
import type { BaseModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { ConditionGroup, GroupBySpec, NamedFilter, QueryUpdatePayload } from '@/web/web/query/types';
import OSearch from '@/web/web/components/view/search/OSearch.vue';
import { computeInitialAppliedFilters, computeAppliedGroups } from '@/web/web/query/utils/search/initialQueryState';
import { buildQueryUpdatePayload } from '@/web/web/query/utils/search/payload';
import { mergeUserFilterDefaults } from '@/web/web/composables/search/userFilterDefaults';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/view/OSearchView' });

const props = withDefaults(
  defineProps<{
    store: WebModelStore<T>;
    placeholder?: string;
    // Controlled overrides take precedence over store.state.queryState.
    keyword?: string;
    appliedFilters?: ConditionGroup[];
    appliedGroups?: Array<GroupBySpec<T>>;
    // One-shot default groups apply only on the first frame, then control is released.
    defaultGroups?: Array<GroupBySpec<T>>;
    defaultFilters?: NamedFilter<T>[]; // Reserved; OSearch still reads store.queryState.defaultFilters.
    keywordFields?: string[]; // Preserve only for aggregation and do not pass to OSearch.
    initialEmit?: boolean; // Control whether the first-frame emit happens here.
  }>(),
  {
    initialEmit: true,
  }
);

const effectivePlaceholder = computed(() => props.placeholder ?? _t('Search...'));

defineOptions({ name: 'OSearchView' });

const emit = defineEmits<{ (e: 'query-update', payload: QueryUpdatePayload<T>): void }>();

// Child-controlled data comes from props first, then store.queryState.
const keywordForChild = computed<string | undefined>(() => {
  if (typeof props.keyword === 'string') return props.keyword as string;
  const qs: any = (props.store.state as any)?.queryState;
  return typeof qs?.keyword === 'string' && qs.keyword.length > 0 ? (qs.keyword as string) : undefined;
});

const codeDefaultFilters = computed<NamedFilter<T>[]>(() => {
  const pf = props.defaultFilters;
  if (pf && Array.isArray(pf)) return pf as NamedFilter<T>[];
  if (pf) return [pf as NamedFilter<T>];
  const qs: any = (props.store.state as any)?.queryState;
  const defs = (qs?.defaultFilters || []) as NamedFilter<T>[];
  return Array.isArray(defs) ? defs : [];
});

/**
 * Authoritative Favorites/IsDefault merge comes from OSearch (single UserFilter Search).
 * Until the first defaults-ready, fall back to code-only defaults for the tag UI.
 */
const favoritesDefaults = ref<NamedFilter<T>[] | null>(null);
const mergedDefaultFilters = computed(
  () =>
    (favoritesDefaults.value ??
      mergeUserFilterDefaults({
        codeDefaults: codeDefaultFilters.value as any,
      })) as NamedFilter<T>[]
);

const mounted = ref(false);
const appliedFiltersForChild = computed<ConditionGroup[]>(() => {
  const qs: any = (props.store.state as any)?.queryState;
  return computeInitialAppliedFilters({
    qs,
    mounted: mounted.value,
    initialEmit: props.initialEmit!,
    explicitFilters: props.appliedFilters as any,
    defaultFilters: mergedDefaultFilters.value,
  });
});

const appliedGroupsForChild = computed<Array<GroupBySpec<T>>>(() => {
  const qs: any = (props.store.state as any)?.queryState;
  return computeAppliedGroups(qs, props.appliedGroups as any, {
    mounted: mounted.value,
    initialEmit: props.initialEmit!,
    defaultGroups: props.defaultGroups as any,
  }) as Array<GroupBySpec<T>>;
});

function onQueryUpdate(payload: QueryUpdatePayload<T>) {
  emit('query-update', payload);
}

async function emitFirstFrameIfNeeded(): Promise<void> {
  if (mounted.value || !props.initialEmit) return;
  await nextTick();
  // Recheck after yield: concurrent defaults-ready handlers may both have passed the guard.
  if (mounted.value || !props.initialEmit) return;
  const filtersAtFirstEmit = appliedFiltersForChild.value || [];
  const groupsAtFirstEmit = appliedGroupsForChild.value || [];
  const payload = buildQueryUpdatePayload<T>(keywordForChild.value, filtersAtFirstEmit, groupsAtFirstEmit, {
    explicitGroups: false,
  });
  emit('query-update', payload);
  mounted.value = true;
}

async function onDefaultsReady(defaults: NamedFilter[]): Promise<void> {
  // OSearch already merged private/shared IsDefault with code defaults after one Search.
  favoritesDefaults.value = (Array.isArray(defaults) ? defaults : []) as NamedFilter<T>[];
  await emitFirstFrameIfNeeded();
}

// First-frame emit waits for OSearch defaults-ready (Favorites load) so IsDefault can win.
onMounted(() => {
  if (!props.initialEmit) {
    mounted.value = true;
  }
});
</script>
