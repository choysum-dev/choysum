// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createApp, defineComponent, h, ref } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const { sfMocks, actorState } = vi.hoisted(() => ({
  sfMocks: {
    Search: vi.fn(async () => [] as any[]),
    Create: vi.fn(async (values: any) => ({
      Id: 'created-1',
      Name: values.Name,
      Condition: values.Condition,
      IsDefault: values.IsDefault,
      UserId: values.UserId,
      CreatedUid: 'me',
    })),
    DeleteById: vi.fn(async () => 1),
  },
  actorState: { id: 'me' as string },
}));

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: (model: string) => {
    if (model === 'web.UserFilter') return sfMocks;
    return {};
  },
}));

vi.mock('./actorUserId', () => ({
  actorUserId: () => actorState.id,
}));

vi.mock('@/web/web/query/utils/condition/builder', () => ({
  filtersToQuery: vi.fn(() => ({ And: [['Name', '=', 'x']] })),
}));

import { filtersToQuery } from '@/web/web/query/utils/condition/builder';
import { useUserFilters } from './useUserFilters';

function runInSetup<T>(fn: () => T): T {
  let result!: T;
  const app = createApp(
    defineComponent({
      setup() {
        result = fn();
        return () => h('div');
      },
    })
  );
  app.mount(document.createElement('div'));
  app.unmount();
  return result;
}

describe('useUserFilters', () => {
  beforeEach(() => {
    actorState.id = 'me';
    sfMocks.Search.mockReset();
    sfMocks.Create.mockReset();
    sfMocks.DeleteById.mockReset();
    sfMocks.Search.mockResolvedValue([]);
    sfMocks.Create.mockImplementation(async (values: any) => ({
      Id: 'created-1',
      Name: values.Name,
      Condition: values.Condition,
      IsDefault: values.IsDefault,
      UserId: values.UserId,
      CreatedUid: 'me',
    }));
    sfMocks.DeleteById.mockResolvedValue(1);
    (filtersToQuery as any).mockClear?.();
  });

  it('load clears favorites when app or model is missing', async () => {
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: '', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: vi.fn(),
      })
    );
    api.favorites.value = [{ Id: 'x', Name: 'Y' } as any];
    await api.load();
    expect(api.favorites.value).toEqual([]);
    expect(sfMocks.Search).not.toHaveBeenCalled();

    const apiModel = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: '' },
        filtersRef: ref([]),
        applyNamedFilter: vi.fn(),
      })
    );
    apiModel.favorites.value = [{ Id: 'x', Name: 'Y' } as any];
    await apiModel.load();
    expect(apiModel.favorites.value).toEqual([]);
    expect(sfMocks.Search).not.toHaveBeenCalled();
  });

  it('load maps shared/private canDelete and exposes defaults', async () => {
    sfMocks.Search.mockResolvedValue([
      {
        Id: 'p1',
        Name: 'Mine',
        Condition: { And: [['A', '=', 1]] },
        IsDefault: true,
        UserId: 'me',
        CreatedUid: 'me',
      },
      {
        Id: 's1',
        Name: 'Team',
        Condition: {},
        IsDefault: true,
        UserId: null,
        CreatedUid: 'me',
      },
      {
        Id: 's2',
        Name: 'OtherShared',
        Condition: {},
        IsDefault: false,
        UserId: '',
        CreatedUid: 'other',
      },
      { Id: '', Name: 'skip' },
    ]);
    const applyNamedFilter = vi.fn();
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter,
        codeDefaults: () => [{ name: 'Code', query: ['X', '=', 1], selected: true }],
      })
    );
    await api.load();
    expect(api.favorites.value).toHaveLength(3);
    expect(api.favorites.value[0]).toMatchObject({ shared: false, canDelete: true });
    expect(api.favorites.value[1]).toMatchObject({ shared: true, canDelete: true });
    expect(api.favorites.value[2]).toMatchObject({ shared: true, canDelete: false });
    expect(api.privateDefault.value?.Id).toBe('p1');
    expect(api.sharedDefault.value?.Id).toBe('s1');
    expect(api.defaultsForOpen.value[0]).toMatchObject({ name: 'Mine', selected: true });
    expect(api.favoriteMenuItems.value[0]).toMatchObject({
      id: 'p1',
      name: 'Mine',
      shared: false,
      isDefault: true,
      canDelete: true,
    });
  });

  it('load records error and clears favorites', async () => {
    sfMocks.Search.mockRejectedValue(new Error('network'));
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: vi.fn(),
      })
    );
    await api.load();
    expect(api.loadError.value).toBe('network');
    expect(api.favorites.value).toEqual([]);
    expect(api.loading.value).toBe(false);
  });

  it('load ignores stale responses when a newer load started', async () => {
    let resolveFirst!: (rows: any[]) => void;
    const first = new Promise<any[]>(resolve => {
      resolveFirst = resolve;
    });
    sfMocks.Search.mockImplementationOnce(() => first).mockResolvedValueOnce([
      {
        Id: 'new',
        Name: 'Newer',
        Condition: {},
        IsDefault: false,
        UserId: 'me',
        CreatedUid: 'me',
      },
    ]);
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: vi.fn(),
      })
    );
    const p1 = api.load();
    const p2 = api.load();
    resolveFirst([
      {
        Id: 'old',
        Name: 'Stale',
        Condition: {},
        IsDefault: true,
        UserId: 'me',
        CreatedUid: 'me',
      },
    ]);
    await Promise.all([p1, p2]);
    expect(api.favorites.value).toHaveLength(1);
    expect(api.favorites.value[0].Id).toBe('new');
    expect(api.loading.value).toBe(false);
  });

  it('load stringifies non-Error throws', async () => {
    sfMocks.Search.mockRejectedValue('plain-string-fail');
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: vi.fn(),
      })
    );
    await api.load();
    expect(api.loadError.value).toBe('plain-string-fail');
    expect(api.favorites.value).toEqual([]);
  });

  it('load without me only requests shared favorites', async () => {
    actorState.id = '';
    sfMocks.Search.mockResolvedValue([
      { Id: 's1', Name: 'Shared', UserId: null, CreatedUid: 'x', IsDefault: false, Condition: {} },
    ]);
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: vi.fn(),
      })
    );
    await api.load();
    const cond = sfMocks.Search.mock.calls[0]![0] as any;
    expect(cond.And).toEqual(
      expect.arrayContaining([
        ['Application', '=', 'demo'],
        ['ModelName', '=', 'Widget'],
        ['ScopeKey', '=', ''],
        { Or: [['UserId', '=', null]] },
      ])
    );
    expect(api.favorites.value[0].canDelete).toBe(false);
  });

  it('load and saveCurrent filter/write ScopeKey from scopeKey()', async () => {
    sfMocks.Search.mockResolvedValue([]);
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget', fieldsMetadata: {}, state: { queryState: {} } },
        filtersRef: ref([]),
        applyNamedFilter: vi.fn(),
        scopeKey: () => '/web/partners/42/edit?x=1',
      })
    );
    await api.load();
    expect(sfMocks.Search.mock.calls[0]![0].And).toEqual(
      expect.arrayContaining([['ScopeKey', '=', '/web/partners/:id/edit']])
    );
    await api.saveCurrent({ name: 'Scoped' });
    expect(sfMocks.Create.mock.calls[0]![0]).toMatchObject({
      Name: 'Scoped',
      ScopeKey: '/web/partners/:id/edit',
    });
  });

  it('saveCurrent returns null when name/app/model missing', async () => {
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: vi.fn(),
      })
    );
    expect(await api.saveCurrent({ name: '  ' })).toBeNull();
    expect(sfMocks.Create).not.toHaveBeenCalled();

    const noApp = runInSetup(() =>
      useUserFilters({
        store: { application: '', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: vi.fn(),
      })
    );
    expect(await noApp.saveCurrent({ name: 'X' })).toBeNull();
  });

  it('load tolerates null Search rows and missing Condition', async () => {
    sfMocks.Search.mockResolvedValue(null as any);
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: vi.fn(),
      })
    );
    await api.load();
    expect(api.favorites.value).toEqual([]);

    sfMocks.Search.mockResolvedValue([
      { Id: 'p1', Name: 'NoCond', IsDefault: false, UserId: 'me', CreatedUid: 'me' },
      { Id: 'p2', Name: 'OtherPriv', IsDefault: false, UserId: 'other', CreatedUid: 'other' },
    ]);
    await api.load();
    expect(api.favoriteMenuItems.value[0]).toMatchObject({ id: 'p1', filter: {}, canDelete: true });
    expect(api.favoriteMenuItems.value[1]).toMatchObject({ id: 'p2', canDelete: false });
    expect(api.defaultsForOpen.value).toEqual([]);
  });

  it('saveCurrent creates private and shared favorites', async () => {
    const groups = [{ id: 'g', logic: 'And', children: [] }];
    const fieldsMeta = { Name: { type: 'varchar' } };
    const api = runInSetup(() =>
      useUserFilters({
        store: {
          application: 'demo',
          modelName: 'Widget',
          fieldsMetadata: fieldsMeta,
          state: { queryState: { keywordFields: ['Name', 'Code'] } },
        },
        filtersRef: ref(groups),
        keywordRef: ref('  find-me  '),
        applyNamedFilter: vi.fn(),
      })
    );
    const privateCreated = await api.saveCurrent({ name: 'Priv', isDefault: true });
    expect(filtersToQuery).toHaveBeenCalledWith(groups, 'find-me', ['Name', 'Code'], fieldsMeta);
    expect(sfMocks.Create.mock.calls[0]![0]).toMatchObject({
      Name: 'Priv',
      ScopeKey: '',
      UserId: 'me',
      IsDefault: true,
      Condition: { And: [['Name', '=', 'x']] },
    });
    expect(privateCreated).toMatchObject({ Id: 'created-1', shared: false, canDelete: true });

    const sharedCreated = await api.saveCurrent({ name: 'Shared', shared: true });
    expect(sfMocks.Create.mock.calls[1]![0]).toMatchObject({
      Name: 'Shared',
      UserId: null,
    });
    expect(sharedCreated?.shared).toBe(true);
  });

  it('saveCurrent omits UserId when actor is empty (private)', async () => {
    actorState.id = '';
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget', fieldsMetadata: {}, state: { queryState: {} } },
        filtersRef: ref([]),
        applyNamedFilter: vi.fn(),
      })
    );
    await api.saveCurrent({ name: 'NoActor' });
    const values = sfMocks.Create.mock.calls[0]![0] as Record<string, unknown>;
    expect(values).toMatchObject({ Name: 'NoActor', Application: 'demo', ModelName: 'Widget' });
    expect(values).not.toHaveProperty('UserId');
  });

  it('saveCurrent uses empty Condition when filtersToQuery returns null', async () => {
    (filtersToQuery as any).mockReturnValueOnce(null);
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget', fieldsMetadata: {}, state: { queryState: {} } },
        filtersRef: ref(null as any),
        applyNamedFilter: vi.fn(),
      })
    );
    await api.saveCurrent({ name: 'NullCond' });
    expect(sfMocks.Create.mock.calls[0]![0]).toMatchObject({
      Name: 'NullCond',
      Condition: {},
      UserId: 'me',
    });
  });

  it('saveCurrent treats empty UserId as shared and falls back createUid to actor', async () => {
    sfMocks.Create.mockResolvedValueOnce({
      Id: 'created-shared',
      Name: 'SharedEmpty',
      Condition: {},
      IsDefault: false,
      UserId: '',
    });
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget', fieldsMetadata: {}, state: { queryState: {} } },
        filtersRef: ref([]),
        applyNamedFilter: vi.fn(),
      })
    );
    const created = await api.saveCurrent({ name: 'SharedEmpty', shared: true });
    expect(created).toMatchObject({
      Id: 'created-shared',
      shared: true,
      createUid: 'me',
      canDelete: true,
    });
  });

  it('apply and remove delegate to helpers/store', async () => {
    const applyNamedFilter = vi.fn();
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter,
      })
    );
    api.apply({ name: 'Fav', filter: { And: [['A', '=', 1]] } });
    expect(applyNamedFilter).toHaveBeenCalledWith({ name: 'Fav', query: { And: [['A', '=', 1]] } });

    await api.remove('fav-1');
    expect(sfMocks.DeleteById).toHaveBeenCalledWith('fav-1');
    expect(sfMocks.Search).toHaveBeenCalled();
  });

  it('load maps missing CreatedUid and private canDelete when actor empty', async () => {
    sfMocks.Search.mockResolvedValue([
      { Id: 's1', Name: 'NoCreatedUid', IsDefault: false, UserId: null },
      { Id: 'p1', Name: 'Priv', IsDefault: false, UserId: 'other' },
      // Falsy non-null UserId stays private and hits `UserId || ''` in canDelete.
      { Id: 'p0', Name: 'ZeroUid', IsDefault: false, UserId: 0 as any, CreatedUid: 'me' },
    ]);
    const withMe = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: vi.fn(),
      })
    );
    await withMe.load();
    expect(withMe.favorites.value[0]).toMatchObject({ createUid: '', canDelete: false });
    expect(withMe.favorites.value[1]).toMatchObject({ shared: false, canDelete: false });
    expect(withMe.favorites.value[2]).toMatchObject({ shared: false, canDelete: false });

    actorState.id = '';
    sfMocks.Search.mockResolvedValue([
      { Id: 'p2', Name: 'PrivNoMe', IsDefault: false, UserId: 'someone', CreatedUid: 'someone' },
    ]);
    const noMe = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: vi.fn(),
      })
    );
    await noMe.load();
    expect(noMe.favorites.value[0].canDelete).toBe(false);
  });

  it('saveCurrent null name and empty CreatedUid/Name fallbacks', async () => {
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget', fieldsMetadata: {}, state: { queryState: {} } },
        filtersRef: ref([]),
        applyNamedFilter: vi.fn(),
      })
    );
    expect(await api.saveCurrent({ name: null as any })).toBeNull();
    expect(await api.saveCurrent({ name: undefined as any })).toBeNull();
    expect(sfMocks.Create).not.toHaveBeenCalled();

    actorState.id = '';
    sfMocks.Create.mockResolvedValueOnce({
      Id: 'created-2',
      Name: '',
      Condition: {},
      IsDefault: false,
      UserId: null,
    });
    const created = await api.saveCurrent({ name: 'FallbackName', shared: true });
    expect(created).toMatchObject({
      Name: 'FallbackName',
      createUid: '',
      shared: true,
    });
  });
});
