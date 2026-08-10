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
    :default-filters="mergedDefaultFilters"
    @query-update="onQueryUpdate"
    @defaults-ready="onDefaultsReady"
  />
</template>

<script setup lang="ts" generic="T extends BaseModel">
import { computed, onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import type { BaseModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { ConditionGroup, GroupBySpec, NamedFilter, QueryUpdatePayload } from '@/web/web/query/types';
import OSearch from '@/web/web/components/view/search/OSearch.vue';
import { computeInitialAppliedFilters, computeAppliedGroups } from '@/web/web/query/utils/search/initialQueryState';
import { buildQueryUpdatePayload } from '@/web/web/query/utils/search/payload';
import { createStoreByModel } from '@/web/web/stores/registry';
import { actorUserId } from '@/web/web/composables/search/actorUserId';
import { mergeUserFilterDefaults, pickLatestIsDefault, type UserFilterRow } from '@/web/web/composables/search/userFilterDefaults';
import { normalizeScopeKey } from '@/web/web/composables/search/scopeKey';
import { trySetupHook } from '@/web/web/composables/search/trySetupHook';
import { createTranslate } from '@/web/web/i18n';

const currentRoute = trySetupHook(() => useRoute());

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

/** Server IsDefault rows; merge order shared with useUserFilters via mergeUserFilterDefaults. */
const serverPrivateDefault = ref<UserFilterRow | null>(null);
const serverSharedDefault = ref<UserFilterRow | null>(null);
let serverDefaultsLoadGen = 0;
const mergedDefaultFilters = computed(
  () =>
    mergeUserFilterDefaults({
      privateDefault: serverPrivateDefault.value,
      sharedDefault: serverSharedDefault.value,
      codeDefaults: codeDefaultFilters.value as any,
    }) as NamedFilter<T>[]
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

async function onDefaultsReady(_defaults: NamedFilter[]) {
  // Refresh after favorite save/delete (and ignore code-only selected presets).
  await loadServerDefaults();
}

async function loadServerDefaults(): Promise<void> {
  const gen = ++serverDefaultsLoadGen;
  const app = String((props.store as any)?.application || '').trim();
  const model = String((props.store as any)?.modelName || '').trim();
  if (!app || !model) {
    // Yield so a newer loadServerDefaults can bump the gen before we clear.
    await Promise.resolve();
    if (gen === serverDefaultsLoadGen) {
      serverPrivateDefault.value = null;
      serverSharedDefault.value = null;
    }
    return;
  }

  try {
    const me = actorUserId();
    const scope = normalizeScopeKey(currentRoute?.path ?? '');
    const uf = createStoreByModel('web.UserFilter') as any;
    const rows = (await uf.Search(
      {
        And: [
          ['Application', '=', app],
          ['ModelName', '=', model],
          ['ScopeKey', '=', scope],
          ['IsDefault', '=', true],
          {
            Or: me
              ? [
                  ['UserId', '=', me],
                  ['UserId', '=', null],
                ]
              : [['UserId', '=', null]],
          },
        ],
      },
      { fields: ['Id', 'Name', 'ScopeKey', 'Condition', 'IsDefault', 'UserId', 'UpdatedAt', 'CreatedAt'] }
    )) as UserFilterRow[];

    if (gen !== serverDefaultsLoadGen) return;
    serverPrivateDefault.value = pickLatestIsDefault(rows, 'private');
    serverSharedDefault.value = pickLatestIsDefault(rows, 'shared');
  } catch {
    // Store may be unavailable before module codegen; fall back to code defaults.
    if (gen === serverDefaultsLoadGen) {
      serverPrivateDefault.value = null;
      serverSharedDefault.value = null;
    }
  }
}

// First-frame emit waits for UserFilter defaults so private/shared IsDefault can win.
onMounted(async () => {
  await loadServerDefaults();
  if (!props.initialEmit) {
    mounted.value = true;
    return;
  }
  const filtersAtFirstEmit = appliedFiltersForChild.value || [];
  const groupsAtFirstEmit = appliedGroupsForChild.value || [];
  const payload = buildQueryUpdatePayload<T>(keywordForChild.value, filtersAtFirstEmit, groupsAtFirstEmit, { explicitGroups: false });
  emit('query-update', payload);
  mounted.value = true;
});
</script>
