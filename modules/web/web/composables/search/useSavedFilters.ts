// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref, type Ref } from 'vue';
import { useAuthStore } from '@/auth/web/stores/auth';
import { createStoreByModel } from '@/web/web/stores/registry';
import type { NamedFilter } from '@/web/web/query/types';
import { filtersToQuery } from '@/web/web/query/utils/condition/builder';
import { mergeSavedFilterDefaults, savedFilterToNamedFilter, type SavedFilterRow } from './savedFilterDefaults';

export type SavedFavoriteItem = SavedFilterRow & {
  Id: string;
  Name: string;
  shared: boolean;
};

function actorUserId(): string {
  try {
    // Browser SPA: identity lives on the auth store, not $choysum.request.context.
    const auth = useAuthStore();
    const fromStore = String((auth.currentUser as any)?.Id || (auth.identity as any)?.userId || '').trim();
    if (fromStore) return fromStore;
  } catch {
    // Auth store unavailable (unit harness / early boot).
  }
  try {
    const id = (globalThis as any)?.$choysum?.request?.context?.identity?.userId;
    return String(id || '').trim();
  } catch {
    return '';
  }
}

/**
 * Load / apply / save / remove web.SavedFilter favorites for the given view store.
 */
export function useSavedFilters(params: {
  store: any;
  filtersRef: Ref<any[]>;
  keywordRef?: Ref<string>;
  applyNamedFilter: (nf: NamedFilter) => void;
  codeDefaults?: () => NamedFilter[] | NamedFilter | undefined;
}) {
  const { store, filtersRef, keywordRef, applyNamedFilter, codeDefaults } = params;
  const favorites = ref<SavedFavoriteItem[]>([]);
  const loading = ref(false);
  const loadError = ref<string | null>(null);

  const application = computed(() => String((store as any)?.application || '').trim());
  const modelName = computed(() => String((store as any)?.modelName || '').trim());

  function savedFilterStore() {
    return createStoreByModel('web.SavedFilter');
  }

  async function load(): Promise<void> {
    const app = application.value;
    const model = modelName.value;
    if (!app || !model) {
      favorites.value = [];
      return;
    }
    loading.value = true;
    loadError.value = null;
    try {
      const me = actorUserId();
      const sf = savedFilterStore() as any;
      const rows = (await sf.Search(
        {
          And: [
            ['Application', '=', app],
            ['ModelName', '=', model],
            ['Active', '=', true],
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
          fields: ['Id', 'Name', 'Condition', 'IsDefault', 'UserId', 'CreateUid', 'Sort'],
          orderBy: { field: 'Name', order: 'asc' },
        }
      )) as SavedFilterRow[];

      favorites.value = (rows || [])
        .filter(r => r && r.Id && r.Name)
        .map(r => ({
          ...(r as any),
          Id: String(r.Id),
          Name: String(r.Name),
          shared: r.UserId == null || r.UserId === '',
        }));
    } catch (e: any) {
      loadError.value = e instanceof Error ? e.message : String(e);
      favorites.value = [];
    } finally {
      loading.value = false;
    }
  }

  const privateDefault = computed(() => favorites.value.find(f => f.IsDefault && !f.shared) || null);
  const sharedDefault = computed(() => favorites.value.find(f => f.IsDefault && f.shared) || null);

  const defaultsForOpen = computed<NamedFilter[]>(() =>
    mergeSavedFilterDefaults({
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
      filter: f.Condition ?? {},
    }))
  );

  function apply(fav: { name: string; filter: any }) {
    applyNamedFilter({ name: fav.name, query: fav.filter } as NamedFilter);
  }

  async function saveCurrent(opts: { name: string; isDefault?: boolean; shared?: boolean }): Promise<SavedFavoriteItem | null> {
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
    const sf = savedFilterStore() as any;
    // Private: omit UserId so the service defaults to the actor (avoids empty→shared).
    // Shared: explicit null.
    const values: Record<string, any> = {
      Name: name,
      Application: app,
      ModelName: model,
      Condition: condition,
      IsDefault: !!opts.isDefault,
      Active: true,
    };
    if (opts.shared) {
      values.UserId = null;
    } else if (me) {
      values.UserId = me;
    }
    const created = await sf.Create(values, ['Id', 'Name', 'Condition', 'IsDefault', 'UserId', 'CreateUid']);
    await load();
    return {
      ...(created as any),
      Id: String((created as any).Id),
      Name: String((created as any).Name || name),
      shared: (created as any).UserId == null || (created as any).UserId === '',
    };
  }

  async function remove(id: string): Promise<void> {
    const sf = savedFilterStore() as any;
    await sf.DeleteById(String(id));
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
    toNamedFilter: savedFilterToNamedFilter,
  } as const;
}
