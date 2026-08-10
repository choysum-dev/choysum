// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref, type Ref } from 'vue';
import { createStoreByModel } from '@/web/web/stores/registry';
import type { NamedFilter } from '@/web/web/query/types';
import { filtersToQuery } from '@/web/web/query/utils/condition/builder';
import { actorUserId } from './actorUserId';
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

/**
 * Load / apply / save / remove web.UserFilter favorites for the given view store.
 */
export function useUserFilters(params: {
  store: any;
  filtersRef: Ref<any[]>;
  keywordRef?: Ref<string>;
  applyNamedFilter: (nf: NamedFilter) => void;
  codeDefaults?: () => NamedFilter[] | NamedFilter | undefined;
  /** Current route path (normalized to ScopeKey). */
  scopeKey?: () => string;
}) {
  const { store, filtersRef, keywordRef, applyNamedFilter, codeDefaults } = params;
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
      if (gen === loadGeneration) {
        favorites.value = [];
      }
      return;
    }
    loading.value = true;
    loadError.value = null;
    try {
      const me = actorUserId();
      const scope = currentScopeKey();
      const uf = userFilterStore() as any;
      const rows = (await uf.Search(
        {
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
        },
        {
          fields: ['Id', 'Name', 'ScopeKey', 'Condition', 'IsDefault', 'UserId', 'CreatedUid', 'UpdatedAt', 'CreatedAt'],
          orderBy: { field: 'Name', order: 'asc' },
        }
      )) as UserFilterRow[];

      if (gen !== loadGeneration) return;

      favorites.value = (rows || [])
        .filter(r => r && r.Id && r.Name)
        .map(r => {
          const createUid = String(r.CreatedUid || '').trim();
          const ownerId = resolveUserFilterUserId(r.UserId);
          const shared = !ownerId;
          return {
            ...r,
            Id: String(r.Id),
            Name: String(r.Name),
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

  const favoriteMenuItems = computed(() =>
    favorites.value.map(f => ({
      id: f.Id,
      name: f.Name,
      shared: f.shared,
      isDefault: !!f.IsDefault,
      canDelete: !!f.canDelete,
      filter: f.Condition ?? {},
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
    const condition = filtersToQuery(conditionGroups as any, keyword, keywordFields, fieldsMeta) ?? {};

    const me = actorUserId();
    const uf = userFilterStore() as any;
    // Private: omit UserId so the service defaults to the actor (avoids empty→shared).
    // Shared: explicit null.
    const values: Record<string, any> = {
      Name: name,
      ScopeKey: currentScopeKey(),
      Application: app,
      ModelName: model,
      Condition: condition,
      IsDefault: !!opts.isDefault,
    };
    if (opts.shared) {
      values.UserId = null;
    } else if (me) {
      values.UserId = me;
    }
    const created = await uf.Create(values, ['Id', 'Name', 'ScopeKey', 'Condition', 'IsDefault', 'UserId', 'CreatedUid']);
    await load();
    const createUid = String((created as UserFilterRow)?.CreatedUid || me || '').trim();
    const shared = !resolveUserFilterUserId((created as any).UserId);
    return {
      ...(created as any),
      Id: String((created as any).Id),
      Name: String((created as any).Name || name),
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

  return {
    favorites,
    favoriteMenuItems,
    loading,
    loadError,
    load,
    apply,
    saveCurrent,
    remove,
    defaultsForOpen,
    privateDefault,
    sharedDefault,
    toNamedFilter: userFilterToNamedFilter,
  } as const;
}
