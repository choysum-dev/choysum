// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createApp, defineComponent, h, reactive, ref } from 'vue';
import { useUserFilters } from './useUserFilters';

type CallRecorder = { calls: unknown[][] };

function fnRecorder<T = undefined, A extends unknown[] = unknown[]>(
  impl?: (...args: A) => T | Promise<T>
): CallRecorder & ((...args: A) => T | Promise<T>) {
  const rec: CallRecorder & ((...args: A) => T | Promise<T>) = Object.assign(
    (...args: A) => {
      rec.calls.push(args);
      return impl ? impl(...args) : (undefined as T);
    },
    { calls: [] as unknown[][] }
  );
  return rec;
}

function mutableFn<T = any>(initial?: (...a: any[]) => T) {
  let impl = initial ?? (async () => undefined as T);
  const fn = Object.assign(
    (...args: any[]) => {
      fn.calls.push(args);
      return impl(...args);
    },
    {
      calls: [] as unknown[][],
      setImpl(next: any) {
        impl = next;
      },
      reset(next?: any) {
        fn.calls.length = 0;
        impl = next ?? (async () => undefined as T);
      },
    }
  );
  return fn;
}

let actorId = 'me';
const Search = mutableFn(async () => [] as any[]);
const Create = mutableFn(async (values: any) => ({
  Id: 'created-1',
  Name: values.Name,
  Condition: values.Condition,
  IsDefault: values.IsDefault,
  UserId: values.UserId,
  CreatedUid: 'me',
}));
const UpdateById = mutableFn(async () => ({}));
const DeleteById = mutableFn(async () => 1);
const filtersToQuery = mutableFn(() => ({ And: [['Name', '=', 'x']] }) as any);

function deps() {
  return {
    createStoreByModel: ((model: string) =>
      model === 'web.UserFilter' ? { Search, Create, UpdateById, DeleteById } : {}) as any,
    actorUserId: (() => actorId) as any,
    filtersToQuery: filtersToQuery as any,
  };
}

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
    actorId = 'me';
    Search.reset(async () => []);
    Create.reset(async (values: any) => ({
      Id: 'created-1',
      Name: values.Name,
      Condition: values.Condition,
      IsDefault: values.IsDefault,
      UserId: values.UserId,
      CreatedUid: 'me',
    }));
    UpdateById.reset(async () => ({}));
    DeleteById.reset(async () => 1);
    filtersToQuery.reset(() => ({ And: [['Name', '=', 'x']] }));
  });

  test('load clears favorites when app or model is missing', async () => {
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: '', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    api.favorites.value = [{ Id: 'x', Name: 'Y' } as any];
    await api.load();
    expect(api.favorites.value).toEqual([]);
    expect(Search.calls.length).toBe(0);

    const apiModel = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: '' },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    apiModel.favorites.value = [{ Id: 'x', Name: 'Y' } as any];
    await apiModel.load();
    expect(apiModel.favorites.value).toEqual([]);
    expect(Search.calls.length).toBe(0);
  });

  test('load maps shared/private canDelete and exposes defaults', async () => {
    Search.setImpl(async () => [
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
    const applyNamedFilter = fnRecorder();
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter,
        codeDefaults: () => [{ name: 'Code', query: ['X', '=', 1], selected: true }],
        ...deps(),
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

  test('load records error and clears favorites', async () => {
    Search.setImpl(async () => {
      throw new Error('network');
    });
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    await api.load();
    expect(api.loadError.value).toBe('network');
    expect(api.favorites.value).toEqual([]);
    expect(api.loading.value).toBe(false);
  });

  test('load ignores stale responses when a newer load started', async () => {
    let resolveFirst!: (rows: any[]) => void;
    const first = new Promise<any[]>(resolve => {
      resolveFirst = resolve;
    });
    Search.setImpl(() => {
      Search.setImpl(async () => [
        {
          Id: 'new',
          Name: 'Newer',
          Condition: {},
          IsDefault: false,
          UserId: 'me',
          CreatedUid: 'me',
        },
      ]);
      return first;
    });
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
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

  test('load stringifies non-Error throws', async () => {
    Search.setImpl(async () => {
      throw 'plain-string-fail';
    });
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    await api.load();
    expect(api.loadError.value).toBe('plain-string-fail');
    expect(api.favorites.value).toEqual([]);
  });

  test('load without me only requests shared favorites', async () => {
    actorId = '';
    Search.setImpl(async () => [
      { Id: 's1', Name: 'Shared', UserId: null, CreatedUid: 'x', IsDefault: false, Condition: {} },
    ]);
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    await api.load();
    const cond = Search.calls[0]![0] as any;
    expect(cond.And).toEqual([
      ['Application', '=', 'demo'],
      ['ModelName', '=', 'Widget'],
      ['ScopeKey', '=', ''],
      { Or: [['UserId', '=', null]] },
    ]);
    expect(api.favorites.value[0].canDelete).toBe(false);
  });

  test('load and saveCurrent filter/write ScopeKey from scopeKey()', async () => {
    Search.setImpl(async () => []);
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget', fieldsMetadata: {}, state: { queryState: {} } },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        scopeKey: () => '/web/partners/42/edit?x=1',
        ...deps(),
      })
    );
    await api.load();
    expect(Search.calls[0]![0]).toMatchObject({
      And: [
        ['Application', '=', 'demo'],
        ['ModelName', '=', 'Widget'],
        ['ScopeKey', '=', '/web/partners/:id/edit'],
        { Or: [['UserId', '=', 'me'], ['UserId', '=', null]] },
      ],
    });
    await api.saveCurrent({ name: 'Scoped' });
    expect(Create.calls[0]![0]).toMatchObject({
      Name: 'Scoped',
      ScopeKey: '/web/partners/:id/edit',
    });
  });

  test('saveCurrent returns null when name/app/model missing', async () => {
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    expect(await api.saveCurrent({ name: '  ' })).toBeNull();
    expect(Create.calls.length).toBe(0);

    const noApp = runInSetup(() =>
      useUserFilters({
        store: { application: '', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    expect(await noApp.saveCurrent({ name: 'X' })).toBeNull();
  });

  test('load tolerates null Search rows and missing Condition', async () => {
    Search.setImpl(async () => null as any);
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    await api.load();
    expect(api.favorites.value).toEqual([]);

    Search.setImpl(async () => [
      { Id: 'p1', Name: 'NoCond', IsDefault: false, UserId: 'me', CreatedUid: 'me' },
      { Id: 'p2', Name: 'OtherPriv', IsDefault: false, UserId: 'other', CreatedUid: 'other' },
      { Id: 'p3', Name: 'NullCond', IsDefault: false, UserId: 'me', CreatedUid: 'me', Condition: null },
    ]);
    await api.load();
    expect(api.favoriteMenuItems.value[0]).toMatchObject({ id: 'p1', filter: {}, canDelete: true });
    expect(api.favoriteMenuItems.value[1]).toMatchObject({ id: 'p2', canDelete: false });
    expect(api.favoriteMenuItems.value[2]).toMatchObject({ id: 'p3', filter: {} });
    expect(api.defaultsForOpen.value).toEqual([]);
  });

  test('saveCurrent creates private and shared favorites', async () => {
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
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    const privateCreated = await api.saveCurrent({ name: 'Priv', isDefault: true });
    expect(filtersToQuery.calls.length).toBe(1);
    expect(filtersToQuery.calls[0]).toEqual([groups, 'find-me', ['Name', 'Code'], fieldsMeta]);
    expect(Create.calls[0]![0]).toMatchObject({
      Name: 'Priv',
      ScopeKey: '',
      UserId: 'me',
      IsDefault: true,
      Condition: { And: [['Name', '=', 'x']] },
    });
    expect(privateCreated).toMatchObject({ Id: 'created-1', shared: false, canDelete: true });

    const sharedCreated = await api.saveCurrent({ name: 'Shared', shared: true });
    expect(Create.calls[1]![0]).toMatchObject({
      Name: 'Shared',
      UserId: null,
    });
    expect(sharedCreated?.shared).toBe(true);
  });

  test('saveCurrent omits UserId when actor is empty (private)', async () => {
    actorId = '';
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget', fieldsMetadata: {}, state: { queryState: {} } },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    await api.saveCurrent({ name: 'NoActor' });
    const values = Create.calls[0]![0] as Record<string, unknown>;
    expect(values).toMatchObject({ Name: 'NoActor', Application: 'demo', ModelName: 'Widget' });
    expect(!('UserId' in values)).toBe(true);
  });

  test('saveCurrent uses empty Condition when filtersToQuery returns null', async () => {
    filtersToQuery.setImpl(() => null);
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget', fieldsMetadata: {}, state: { queryState: {} } },
        filtersRef: ref(null as any),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    await api.saveCurrent({ name: 'NullCond' });
    expect(Create.calls[0]![0]).toMatchObject({
      Name: 'NullCond',
      Condition: {},
      UserId: 'me',
    });
  });

  test('saveCurrent treats empty UserId as shared and falls back createUid to actor', async () => {
    Create.setImpl(async () => ({
      Id: 'created-shared',
      Name: 'SharedEmpty',
      Condition: {},
      IsDefault: false,
      UserId: '',
    }));
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget', fieldsMetadata: {}, state: { queryState: {} } },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
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

  test('apply and remove delegate to helpers/store', async () => {
    const applyNamedFilter = fnRecorder();
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter,
        ...deps(),
      })
    );
    api.apply({ name: 'Fav', filter: { And: [['A', '=', 1]] } });
    expect(applyNamedFilter.calls.length).toBe(1);
    expect(applyNamedFilter.calls[0]).toEqual([{ name: 'Fav', query: { And: [['A', '=', 1]] } }]);

    await api.remove('fav-1');
    expect(DeleteById.calls.length).toBe(1);
    expect(DeleteById.calls[0]).toEqual(['fav-1']);
    expect(Search.calls.length).toBeGreaterThan(0);
  });

  test('updateMeta writes Name/IsDefault/UserId only and reloads', async () => {
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([{ id: 'g1' }]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    Search.calls.length = 0;
    await api.updateMeta('fav-9', { name: 'Renamed', isDefault: true, shared: false });
    expect(UpdateById.calls.length).toBe(1);
    expect(UpdateById.calls[0]).toEqual([
      'fav-9',
      {
        Name: 'Renamed',
        IsDefault: true,
        UserId: 'me',
      },
    ]);
    const payload = UpdateById.calls[0]![1] as Record<string, unknown>;
    expect(!('Condition' in payload)).toBe(true);
    expect(Search.calls.length).toBeGreaterThan(0);

    UpdateById.calls.length = 0;
    await api.updateMeta('fav-9', { name: 'SharedNow', shared: true });
    expect(UpdateById.calls[0]).toEqual([
      'fav-9',
      {
        Name: 'SharedNow',
        IsDefault: false,
        UserId: null,
      },
    ]);
  });

  test('load clears loading when a newer load invalidates app/model', async () => {
    let resolveSlow!: (rows: any[]) => void;
    const slow = new Promise<any[]>(resolve => {
      resolveSlow = resolve;
    });
    Search.setImpl(() => slow);
    const store = reactive({ application: 'demo', modelName: 'Widget' });
    const api = runInSetup(() =>
      useUserFilters({
        store,
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    const p1 = api.load();
    expect(api.loading.value).toBe(true);
    store.application = '';
    const p2 = api.load();
    resolveSlow([{ Id: 'stale', Name: 'Stale', UserId: 'me', CreatedUid: 'me' }]);
    await Promise.all([p1, p2]);
    expect(api.favorites.value).toEqual([]);
    expect(api.loading.value).toBe(false);
    expect(api.loadError.value).toBeNull();
  });

  test('load ignores stale empty-context clear when a newer load started', async () => {
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: '', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    api.favorites.value = [{ Id: 'keep', Name: 'Keep' } as any];
    const p1 = api.load();
    const p2 = api.load();
    await Promise.all([p1, p2]);
    // Newer empty load wins the clear; older gen skips after yield.
    expect(api.favorites.value).toEqual([]);
    expect(Search.calls.length).toBe(0);
  });

  test('load ignores stale Search rejection after a newer load', async () => {
    let rejectSlow!: (err: Error) => void;
    const slow = new Promise<any[]>((_resolve, reject) => {
      rejectSlow = reject;
    });
    Search.setImpl(() => {
      Search.setImpl(async () => [{ Id: 'ok', Name: 'Ok', UserId: 'me', CreatedUid: 'me' }]);
      return slow;
    });
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    const p1 = api.load();
    const p2 = api.load();
    rejectSlow(new Error('stale network'));
    await Promise.all([p1, p2]);
    expect(api.loadError.value).toBeNull();
    expect(api.favorites.value[0].Id).toBe('ok');
    expect(api.loading.value).toBe(false);
  });

  test('updateMeta no-ops when id or name missing', async () => {
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    await api.updateMeta('', { name: 'X' });
    await api.updateMeta('fav-1', { name: '  ' });
    await api.updateMeta(null as any, { name: 'X' });
    await api.updateMeta('fav-1', { name: null as any });
    await api.updateMeta('fav-1', { name: undefined as any });
    expect(UpdateById.calls.length).toBe(0);
  });

  test('updateMeta omits UserId when actor empty and private', async () => {
    actorId = '';
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    await api.updateMeta('fav-1', { name: 'NoActor', shared: false });
    const values = UpdateById.calls[0]![1] as Record<string, unknown>;
    expect(values).toMatchObject({ Name: 'NoActor', IsDefault: false });
    expect(!('UserId' in values)).toBe(true);
  });

  test('load maps missing CreatedUid and private canDelete when actor empty', async () => {
    Search.setImpl(async () => [
      { Id: 's1', Name: 'NoCreatedUid', IsDefault: false, UserId: null },
      { Id: 'p1', Name: 'Priv', IsDefault: false, UserId: 'other' },
      // Falsy non-null UserId stays private and hits `UserId || ''` in canDelete.
      { Id: 'p0', Name: 'ZeroUid', IsDefault: false, UserId: 0 as any, CreatedUid: 'me' },
    ]);
    const withMe = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    await withMe.load();
    expect(withMe.favorites.value[0]).toMatchObject({ createUid: '', canDelete: false });
    expect(withMe.favorites.value[1]).toMatchObject({ shared: false, canDelete: false });
    expect(withMe.favorites.value[2]).toMatchObject({ shared: false, canDelete: false });

    actorId = '';
    Search.setImpl(async () => [
      { Id: 'p2', Name: 'PrivNoMe', IsDefault: false, UserId: 'someone', CreatedUid: 'someone' },
    ]);
    const noMe = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget' },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    await noMe.load();
    expect(noMe.favorites.value[0].canDelete).toBe(false);
  });

  test('saveCurrent null name and empty CreatedUid/Name fallbacks', async () => {
    const api = runInSetup(() =>
      useUserFilters({
        store: { application: 'demo', modelName: 'Widget', fieldsMetadata: {}, state: { queryState: {} } },
        filtersRef: ref([]),
        applyNamedFilter: fnRecorder(),
        ...deps(),
      })
    );
    expect(await api.saveCurrent({ name: null as any })).toBeNull();
    expect(await api.saveCurrent({ name: undefined as any })).toBeNull();
    expect(Create.calls.length).toBe(0);

    actorId = '';
    Create.setImpl(async () => ({
      Id: 'created-2',
      Name: '',
      Condition: {},
      IsDefault: false,
      UserId: null,
    }));
    const created = await api.saveCurrent({ name: 'FallbackName', shared: true });
    expect(created).toMatchObject({
      Name: 'FallbackName',
      createUid: '',
      shared: true,
    });
  });
});
