// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, inject, ref, type InjectionKey, type Ref } from 'vue';
import { createStoreByModel as defaultCreateStoreByModel } from '@/web/web/stores/registry';
import type { GroupBySpec, NamedFilter } from '@/web/web/query/types';
import { filtersToQuery as defaultFiltersToQuery } from '@/web/web/query/utils/condition/builder';
import { normalizeGroupby } from '@/web/web/query/utils/grouping/normalize';
import { actorUserId as defaultActorUserId } from './actorUserId';
import {
  mergeUserFilterDefaults,
  pickLatestIsDefault,
  resolveUserFilterUserId,
  userFilterToNamedFilter,
  type UserFilterRow,
} from './userFilterDefaults';
import { normalizeScopeKey } from './scopeKey';

export type UserFavoriteItem = UserFilterRow & {
  Id: string;
  Name: string;
  shared: boolean;
  createUid: string;
  canDelete: boolean;
};

export type UserFavoriteMenuItem = {
  id: string;
  name: string;
  shared: boolean;
  isDefault: boolean;
  canDelete: boolean;
  filter: unknown;
  groups: GroupBySpec[];
};

function normalizeFavoriteGroups(raw: unknown): GroupBySpec[] {
  if (raw == null || raw === '') return [];
  // jsonobject columns may decode empty as {} (not []); only arrays / field specs are grouping.
  if (typeof raw === 'object' && !Array.isArray(raw) && typeof (raw as { field?: unknown }).field !== 'string') {
    return [];
  }
  return normalizeGroupby(raw as GroupBySpec[]) as GroupBySpec[];
}

/** True when a UserFilter RPC failed because GroupBy / group_by is not on the schema yet. */
function isMissingGroupByColumn(e: unknown): boolean {
  const msg = e instanceof Error ? e.message : String(e);
  return /group_?by/i.test(msg) && /(does not exist|unknown|no such|undefined column)/i.test(msg);
}

export type UserFiltersDeps = {
  createStoreByModel?: typeof defaultCreateStoreByModel;
  actorUserId?: typeof defaultActorUserId;
  filtersToQuery?: typeof defaultFiltersToQuery;
};

/** Optional override for `useUserFilters` (unit harness / specialized shells). */
export type UseUserFiltersFn = typeof useUserFilters;
export const UseUserFiltersKey: InjectionKey<UseUserFiltersFn> = Symbol('UseUserFilters');

/** Resolve injected `useUserFilters` factory, else the product default. */
export function useInjectedUserFilters(
  params: Parameters<typeof useUserFilters>[0]
): ReturnType<typeof useUserFilters> {
  const override = inject(UseUserFiltersKey, null);
  return (override ?? useUserFilters)(params);
}

/**
 * Load / apply / save / remove web.UserFilter favorites for the given view store.
 */
export function useUserFilters(params: {
  store: any;
  filtersRef: Ref<any[]>;
  keywordRef?: Ref<string>;
  applyNamedFilter: (nf: NamedFilter) => void;
  /** Current list grouping to persist with Favorites (GroupBySpec[]). */
  getAppliedGroups?: () => Array<GroupBySpec<any>> | undefined;
  codeDefaults?: () => NamedFilter[] | NamedFilter | undefined;
  /** Current route path (normalized to ScopeKey). */
  scopeKey?: () => string;
  createStoreByModel?: UserFiltersDeps['createStoreByModel'];
  actorUserId?: UserFiltersDeps['actorUserId'];
  filtersToQuery?: UserFiltersDeps['filtersToQuery'];
}) {
  const { store, filtersRef, keywordRef, applyNamedFilter, codeDefaults } = params;
  const createStoreByModel = params.createStoreByModel ?? defaultCreateStoreByModel;
  const resolveActorUserId = params.actorUserId ?? defaultActorUserId;
  const runFiltersToQuery = params.filtersToQuery ?? defaultFiltersToQuery;
  const favorites = ref<UserFavoriteItem[]>([]);
  const loading = ref(false);
  const loadError = ref<string | null>(null);
  let loadGeneration = 0;

  const application = computed(() => String((store as any)?.application || '').trim());
  const modelName = computed(() => String((store as any)?.modelName || '').trim());

  function currentScopeKey(): string {
    return normalizeScopeKey(params.scopeKey?.() ?? '');
  }

  function userFilterStore() {
    return createStoreByModel('web.UserFilter');
  }

  async function load(): Promise<void> {
    const gen = ++loadGeneration;
    const app = application.value;
    const model = modelName.value;
    if (!app || !model) {
      // Yield so a newer load() can bump gen before we clear (stale empty-context guard).
      await Promise.resolve();
      if (gen === loadGeneration) {
        favorites.value = [];
        loading.value = false;
        loadError.value = null;
      }
      return;
    }
    loading.value = true;
    loadError.value = null;
    try {
      const me = resolveActorUserId();
      const scope = currentScopeKey();
      const uf = userFilterStore() as any;
      const condition = {
        And: [
          ['Application', '=', app],
          ['ModelName', '=', model],
          ['ScopeKey', '=', scope],
          {
            Or: me
              ? [
                  ['UserId', '=', me],
                  ['UserId', '=', null],
                ]
              : [['UserId', '=', null]],
          },
        ],
      };
      const fieldsWithoutGroupBy = [
        'Id',
        'Name',
        'ScopeKey',
        'Condition',
        'IsDefault',
        'UserId',
        'CreatedUid',
        'UpdatedAt',
        'CreatedAt',
      ];
      const fieldsWithGroupBy = [
        'Id',
        'Name',
        'ScopeKey',
        'Condition',
        'GroupBy',
        'IsDefault',
        'UserId',
        'CreatedUid',
        'UpdatedAt',
        'CreatedAt',
      ];
      const orderBy = { field: 'Name', order: 'asc' as const };
      // Prefer GroupBy when the column exists; fall back after failed upgrades /
      // mixed FE–BE where web_user_filter.group_by is not migrated yet.
      let rows: UserFilterRow[];
      try {
        rows = (await uf.Search(condition, {
          fields: fieldsWithGroupBy,
          orderBy,
        })) as UserFilterRow[];
      } catch (e) {
        if (!isMissingGroupByColumn(e)) throw e;
        rows = (await uf.Search(condition, {
          fields: fieldsWithoutGroupBy,
          orderBy,
        })) as UserFilterRow[];
      }

      if (gen !== loadGeneration) return;

      favorites.value = (rows || [])
        .filter(r => r && r.Id && r.Name)
        .map((r): UserFavoriteItem => {
          const createUid = String(r.CreatedUid || '').trim();
          const ownerId = resolveUserFilterUserId(r.UserId);
          const shared = !ownerId;
          return {
            ...r,
            Id: String(r.Id),
            Name: String(r.Name),
            GroupBy: normalizeFavoriteGroups(r.GroupBy),
            shared,
            createUid,
            // Private: owner; shared: creator only.
            canDelete: shared ? !!me && createUid === me : !!me && ownerId === me,
          };
        });
    } catch (e: any) {
      if (gen !== loadGeneration) return;
      loadError.value = e instanceof Error ? e.message : String(e);
      favorites.value = [];
    } finally {
      if (gen === loadGeneration) {
        loading.value = false;
      }
    }
  }

  const privateDefault = computed(() => pickLatestIsDefault(favorites.value, 'private'));
  const sharedDefault = computed(() => pickLatestIsDefault(favorites.value, 'shared'));

  const defaultsForOpen = computed<NamedFilter[]>(() =>
    mergeUserFilterDefaults({
      privateDefault: privateDefault.value,
      sharedDefault: sharedDefault.value,
      codeDefaults: codeDefaults ? codeDefaults() : undefined,
    })
  );

  const favoriteMenuItems = computed<UserFavoriteMenuItem[]>(() =>
    favorites.value.map(f => ({
      id: f.Id,
      name: f.Name,
      shared: f.shared,
      isDefault: !!f.IsDefault,
      canDelete: !!f.canDelete,
      filter: f.Condition ?? {},
      groups: normalizeFavoriteGroups(f.GroupBy),
    }))
  );

  function apply(fav: { name: string; filter: any }) {
    applyNamedFilter({ name: fav.name, query: fav.filter } as NamedFilter);
  }

  async function saveCurrent(opts: { name: string; isDefault?: boolean; shared?: boolean }): Promise<UserFavoriteItem | null> {
    const app = application.value;
    const model = modelName.value;
    const name = String(opts.name || '').trim();
    if (!app || !model || !name) return null;

    const conditionGroups = Array.isArray(filtersRef.value) ? filtersRef.value : [];
    const keyword = keywordRef?.value?.trim() || undefined;
    const fieldsMeta = (store as any)?.fieldsMetadata as Record<string, any> | undefined;
    const keywordFields = ((store as any)?.state?.queryState?.keywordFields || undefined) as string[] | undefined;
    // filtersToQuery returns [] when there are no conditions; persist {} so jsonobject
    // stays object-shaped and apply can treat empty favorites as groupby-only.
    const rawCondition = runFiltersToQuery(conditionGroups as any, keyword, keywordFields, fieldsMeta);
    const condition =
      rawCondition == null || (Array.isArray(rawCondition) && rawCondition.length === 0) ? {} : rawCondition;
    const groups = normalizeFavoriteGroups(params.getAppliedGroups?.() ?? []);

    const me = resolveActorUserId();
    const uf = userFilterStore() as any;
    // Private: omit UserId so the service defaults to the actor (avoids empty→shared).
    // Shared: explicit null.
    const values: Record<string, any> = {
      Name: name,
      ScopeKey: currentScopeKey(),
      Application: app,
      ModelName: model,
      Condition: condition,
      GroupBy: groups,
      IsDefault: !!opts.isDefault,
    };
    if (opts.shared) {
      values.UserId = null;
    } else if (me) {
      values.UserId = me;
    }
    const createFieldsWithGroupBy = [
      'Id',
      'Name',
      'ScopeKey',
      'Condition',
      'GroupBy',
      'IsDefault',
      'UserId',
      'CreatedUid',
    ];
    const createFieldsWithoutGroupBy = [
      'Id',
      'Name',
      'ScopeKey',
      'Condition',
      'IsDefault',
      'UserId',
      'CreatedUid',
    ];
    let created: any;
    try {
      created = await uf.Create(values, createFieldsWithGroupBy);
    } catch (e) {
      if (!isMissingGroupByColumn(e)) throw e;
      // Schema not migrated yet: persist Condition only so Favorites still save.
      delete values.GroupBy;
      created = await uf.Create(values, createFieldsWithoutGroupBy);
    }
    await load();
    const createUid = String((created as UserFilterRow)?.CreatedUid || me || '').trim();
    const shared = !resolveUserFilterUserId((created as any).UserId);
    return {
      ...(created as any),
      Id: String((created as any).Id),
      Name: String((created as any).Name || name),
      GroupBy: normalizeFavoriteGroups((created as any).GroupBy ?? groups),
      shared,
      createUid,
      canDelete: true,
    };
  }

  async function remove(id: string): Promise<void> {
    const uf = userFilterStore() as any;
    await uf.DeleteById(String(id));
    await load();
  }

  /** Update Name / IsDefault / shared only; leave Condition unchanged. */
  async function updateMeta(
    id: string,
    opts: { name: string; isDefault?: boolean; shared?: boolean }
  ): Promise<void> {
    const name = String(opts.name || '').trim();
    const favId = String(id || '').trim();
    if (!favId || !name) return;

    const me = resolveActorUserId();
    const uf = userFilterStore() as any;
    const values: Record<string, any> = {
      Name: name,
      IsDefault: !!opts.isDefault,
    };
    if (opts.shared) {
      values.UserId = null;
    } else if (me) {
      values.UserId = me;
    }
    await uf.UpdateById(favId, values);
    await load();
  }

  return {
    favorites,
    favoriteMenuItems,
    loading,
    loadError,
    load,
    apply,
    saveCurrent,
    updateMeta,
    remove,
    defaultsForOpen,
    privateDefault,
    sharedDefault,
    toNamedFilter: userFilterToNamedFilter,
  } as const;
}
