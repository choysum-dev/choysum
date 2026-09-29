<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    v-if="useStoreEngine"
    class="choy-search-view"
    data-test="choy-search"
    :class="{ 'pointer-events-none select-none': disabled }"
    :inert="disabled || undefined"
    :aria-disabled="disabled ? 'true' : undefined"
    v-bind="splitAttrs.bind"
    v-on="splitAttrs.listeners"
    @keydown.capture="onDisabledKeydown"
  >
    <Search
      :store="boundStore"
      :placeholder="effectivePlaceholder"
      :current-keyword="keywordForChild"
      :current-applied-filters="appliedFiltersForChild"
      :current-applied-groups="appliedGroupsForChild"
      :default-filters="codeDefaultFilters"
      @query-update="onQueryUpdate"
      @defaults-ready="onDefaultsReady"
    />
  </div>

  <div
    v-else
    v-bind="($attrs as any)"
    data-anchor="choy.search-view"
    :class="['choy-search-view flex flex-wrap items-center gap-2', props.class]"
  >
    <Input
      v-model="keyword"
      class="min-w-[12rem] flex-1"
      :placeholder="placeholder"
      :disabled="disabled"
      :aria-label="placeholder"
      @keydown="onKeydown"
    />
    <ChoyButton
      type="button"
      size="sm"
      :disabled="disabled"
      @click="submit"
    >
      Search
    </ChoyButton>
  </div>
</template>

<script setup lang="ts" generic="T extends BaseModel">
import { computed, getCurrentInstance, onMounted, ref, nextTick, useAttrs } from 'vue';
import type { BaseModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { ConditionGroup, GroupBySpec, NamedFilter, QueryUpdatePayload } from '@/web/web/query/types';
import Search from '@/web/web/components/view/search/Search.vue';
import { computeInitialAppliedFilters, computeAppliedGroups } from '@/web/web/query/utils/search/initialQueryState';
import { buildQueryUpdatePayload } from '@/web/web/query/utils/search/payload';
import { mergeUserFilterDefaults } from '@/web/web/composables/search/userFilterDefaults';
import { createTranslate } from '@/web/web/i18n';
import { hasChoyStoreEngine, splitChoyAttrsListeners } from '@/web/web/composables/choyStoreMode';
import { resolvePageStore, useOptionalPageStore } from '@/web/web/composables/usePageContext';
import Input from '../vendor/ui/input/Input.vue';
import ChoyButton from '../layout/ChoyButton.vue';
import type { ClassValue } from '../../lib/utils';
import {
  buildChoySearchQuery,
  choySearchQueryFromPayload,
  type ChoySearchQuery
} from './searchViewHelpers';

defineOptions({ name: 'ChoySearchView', inheritAttrs: false });

const { _t } = createTranslate('web', { scope: 'web/components/view/SearchView' });

const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    placeholder?: string;
    disabled?: boolean;
    store?: WebModelStore<T>;
    keyword?: string;
    appliedFilters?: ConditionGroup[];
    appliedGroups?: Array<GroupBySpec<T>>;
    defaultGroups?: Array<GroupBySpec<T>>;
    defaultFilters?: NamedFilter<T> | NamedFilter<T>[];
    keywordFields?: string[];
    initialEmit?: boolean;
  }>(),
  {
    disabled: false,
    initialEmit: true,
  },
);

const attrs = useAttrs();
const pageStore = useOptionalPageStore<WebModelStore<T>>();
const useStoreEngine = computed(() => hasChoyStoreEngine(props.store, pageStore.value));
const splitAttrs = computed(() => splitChoyAttrsListeners(attrs as Record<string, unknown>));

const rawPropKeys = new Set(
  Object.keys(((getCurrentInstance()?.vnode.props as Record<string, unknown> | null) || {}) as Record<string, unknown>),
);
const hasExplicitKeywordProp = () => rawPropKeys.has('keyword');

const boundStore = computed((): WebModelStore<T> => {
  if (!useStoreEngine.value) {
    return null as unknown as WebModelStore<T>;
  }
  return resolvePageStore(props.store ?? pageStore.value, 'ChoySearchView');
});

const effectivePlaceholder = computed(() => props.placeholder ?? _t('Search...'));

const emit = defineEmits<{
  'query-update': [query: ChoySearchQuery];
}>();

/** Without inert, pointer-events-none alone leaves keyboard focusable; block keys. */
function onDisabledKeydown(event: KeyboardEvent): void {
  if (!props.disabled) return;
  event.preventDefault();
  event.stopPropagation();
}

const keyword = defineModel<string>('keyword', { default: '' });

function submit(): void {
  if (props.disabled) {
    return;
  }
  const query = buildChoySearchQuery(keyword.value);
  keyword.value = query.keyword;
  emit('query-update', query);
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key !== 'Enter' || event.isComposing || event.keyCode === 229) {
    return;
  }
  event.preventDefault();
  submit();
}

// --- Store-bound search engine ---

const keywordForChild = computed<string | undefined>(() => {
  if (!useStoreEngine.value) return undefined;
  if (hasExplicitKeywordProp() && typeof props.keyword === 'string') {
    return props.keyword as string;
  }
  const qs: any = (boundStore.value.state as any)?.queryState;
  return typeof qs?.keyword === 'string' && qs.keyword.length > 0 ? (qs.keyword as string) : undefined;
});

const codeDefaultFilters = computed<NamedFilter<T>[]>(() => {
  if (!useStoreEngine.value) return [];
  const pf = props.defaultFilters;
  if (pf && Array.isArray(pf)) return pf as NamedFilter<T>[];
  if (pf) return [pf as NamedFilter<T>];
  const qs: any = (boundStore.value.state as any)?.queryState;
  const defs = (qs?.defaultFilters || []) as NamedFilter<T>[];
  return Array.isArray(defs) ? defs : [];
});

const favoritesDefaults = ref<NamedFilter<T>[] | null>(null);
const mergedDefaultFilters = computed(
  () =>
    (favoritesDefaults.value ??
      mergeUserFilterDefaults({
        codeDefaults: codeDefaultFilters.value as any,
      })) as NamedFilter<T>[],
);

const mounted = ref(false);
const appliedFiltersForChild = computed<ConditionGroup[]>(() => {
  if (!useStoreEngine.value) return [];
  const qs: any = (boundStore.value.state as any)?.queryState;
  return computeInitialAppliedFilters({
    qs,
    mounted: mounted.value,
    initialEmit: props.initialEmit!,
    explicitFilters: props.appliedFilters as any,
    defaultFilters: mergedDefaultFilters.value,
  });
});

const appliedGroupsForChild = computed<Array<GroupBySpec<T>>>(() => {
  if (!useStoreEngine.value) return [];
  const qs: any = (boundStore.value.state as any)?.queryState;
  return computeAppliedGroups(qs, props.appliedGroups as any, {
    mounted: mounted.value,
    initialEmit: props.initialEmit!,
    defaultGroups: props.defaultGroups as any,
  }) as Array<GroupBySpec<T>>;
});

function onQueryUpdate(payload: QueryUpdatePayload<T>) {
  const query = choySearchQueryFromPayload(payload ?? {});
  keyword.value = query.keyword;
  emit('query-update', query);
}

async function emitFirstFrameIfNeeded(): Promise<void> {
  if (!useStoreEngine.value || mounted.value || !props.initialEmit) return;
  await nextTick();
  if (mounted.value || !props.initialEmit) return;
  const filtersAtFirstEmit = appliedFiltersForChild.value;
  const groupsAtFirstEmit = appliedGroupsForChild.value;
  const payload = buildQueryUpdatePayload<T>(keywordForChild.value, filtersAtFirstEmit, groupsAtFirstEmit, {
    explicitGroups: false,
  });
  const query = choySearchQueryFromPayload(payload ?? {});
  keyword.value = query.keyword;
  emit('query-update', query);
  mounted.value = true;
}

async function onDefaultsReady(defaults: NamedFilter[]): Promise<void> {
  if (!useStoreEngine.value) return;
  favoritesDefaults.value = (Array.isArray(defaults) ? defaults : []) as NamedFilter<T>[];
  await emitFirstFrameIfNeeded();
}

onMounted(() => {
  if (!useStoreEngine.value) return;
  if (!props.initialEmit) {
    mounted.value = true;
  }
});
</script>
