// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createApp, defineComponent, h } from 'vue';
import { parseExportModelRef, useExportTemplates } from './useExportTemplates';

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
  Id: 'tpl-1',
  Name: values.Name,
  Fields: values.Fields,
  ImportCompatible: values.ImportCompatible,
  UserId: values.UserId,
  CreatedUid: 'me',
}));
const DeleteById = mutableFn(async () => 1);

function deps() {
  return {
    createStoreByModel: ((model: string) =>
      model === 'web.ExportTemplate' ? { Search, Create, DeleteById } : {}) as any,
    actorUserId: (() => actorId) as any,
  };
}

function runHook(model = 'partner.Partner') {
  return runHookGetter(() => model);
}

function runHookGetter(getModel: () => string) {
  let api!: ReturnType<typeof useExportTemplates>;
  const app = createApp(
    defineComponent({
      setup() {
        api = useExportTemplates(getModel, deps());
        return () => h('div');
      },
    })
  );
  app.mount(document.createElement('div'));
  app.unmount();
  return api;
}

describe('parseExportModelRef', () => {
  test('splits application and model name', () => {
    expect(parseExportModelRef('partner.Partner')).toEqual({ application: 'partner', modelName: 'Partner' });
    expect(parseExportModelRef('')).toEqual({ application: '', modelName: '' });
    expect(parseExportModelRef('invalid')).toEqual({ application: '', modelName: '' });
    expect(parseExportModelRef('.Partner')).toEqual({ application: '', modelName: '' });
    expect(parseExportModelRef('partner.')).toEqual({ application: '', modelName: '' });
    expect(parseExportModelRef(null as any)).toEqual({ application: '', modelName: '' });
    expect(parseExportModelRef(undefined as any)).toEqual({ application: '', modelName: '' });
  });
});

describe('useExportTemplates', () => {
  beforeEach(() => {
    actorId = 'me';
    Search.reset(async () => []);
    Create.reset(async (values: any) => ({
      Id: 'tpl-1',
      Name: values.Name,
      Fields: values.Fields,
      ImportCompatible: values.ImportCompatible,
      UserId: values.UserId,
      CreatedUid: 'me',
    }));
    DeleteById.reset(async () => 1);
  });

  test('load queries own and shared templates for target model', async () => {
    Search.setImpl(async () => [
      { Id: '1', Name: 'Basic', Fields: ['Name'], UserId: 'me', CreatedUid: 'me' },
      { Id: '2', Name: 'Shared', Fields: ['Code'], UserId: null, CreatedUid: 'other' },
      { Id: '3', Name: '', Fields: ['Name'] },
      { Id: '4', Fields: ['Name'] },
      null,
    ]);
    const api = runHook();
    await api.load();
    expect(Search.calls.length).toBeGreaterThan(0);
    expect(api.templates.value).toHaveLength(2);
    expect(api.templates.value[0].canDelete).toBe(true);
    expect(api.templates.value[1].shared).toBe(true);
    expect(api.templates.value[1].canDelete).toBe(false);
  });

  test('load maps missing CreatedUid and treats null Search rows as empty', async () => {
    Search.setImpl(async () => null);
    const api = runHook();
    await api.load();
    expect(api.templates.value).toEqual([]);

    Search.setImpl(async () => [{ Id: 's1', Name: 'NoCreatedUid', Fields: ['Name'], UserId: null }]);
    await api.load();
    expect(api.templates.value[0]).toMatchObject({ createUid: '', canDelete: false });
  });

  test('load clears loading when a newer load invalidates the model ref', async () => {
    let resolveSlow!: (rows: any[]) => void;
    const slow = new Promise<any[]>(resolve => {
      resolveSlow = resolve;
    });
    let searchCalls = 0;
    Search.setImpl(() => {
      searchCalls += 1;
      if (searchCalls === 1) return slow;
      return Promise.resolve([]);
    });
    let model = 'partner.Partner';
    const api = runHookGetter(() => model);
    const first = api.load();
    expect(api.loading.value).toBe(true);
    model = 'invalid';
    const second = api.load();
    // Let the invalidating load bump generation and park on its empty-context yield.
    await Promise.resolve();
    resolveSlow([{ Id: 'stale', Name: 'Stale', Fields: ['Name'], UserId: 'me', CreatedUid: 'me' }]);
    await Promise.all([first, second]);
    expect(api.templates.value).toEqual([]);
    expect(api.loading.value).toBe(false);
    expect(api.loadError.value).toBeNull();
  });

  test('load maps shared/private canDelete and normalizes Fields', async () => {
    Search.setImpl(async () => [
      { Id: 'p1', Name: 'Mine', Fields: ['Name'], UserId: 'me', CreatedUid: 'me' },
      { Id: 's1', Name: 'Team', Fields: 'not-array' as any, UserId: null, CreatedUid: 'me' },
      { Id: 's2', Name: 'OtherShared', Fields: ['Code'], UserId: '', CreatedUid: 'other' },
      { Id: 'p2', Name: 'OtherPrivate', Fields: ['Code'], UserId: 'other', CreatedUid: 'other' },
    ]);
    const api = runHook();
    await api.load();
    expect(api.templates.value).toHaveLength(4);
    expect(api.templates.value[0]).toMatchObject({ shared: false, canDelete: true, Fields: ['Name'] });
    expect(api.templates.value[1]).toMatchObject({ shared: true, canDelete: true, Fields: [] });
    expect(api.templates.value[2]).toMatchObject({ shared: true, canDelete: false });
    expect(api.templates.value[3]).toMatchObject({ shared: false, canDelete: false });
  });

  test('load clears templates for invalid model refs', async () => {
    const api = runHook('invalid');
    await api.load();
    expect(Search.calls.length).toBe(0);
    expect(api.templates.value).toEqual([]);
    expect(api.loading.value).toBe(false);
    expect(api.loadError.value).toBeNull();
  });

  test('load clears templates for empty model refs', async () => {
    const api = runHook('');
    await api.load();
    expect(Search.calls.length).toBe(0);
    expect(api.templates.value).toEqual([]);
  });

  test('load treats null model ref as invalid', async () => {
    const api = runHookGetter(() => null as any);
    await api.load();
    expect(Search.calls.length).toBe(0);
    expect(api.templates.value).toEqual([]);
  });

  test('load without actor only queries shared templates', async () => {
    actorId = '';
    const api = runHook();
    await api.load();
    const query = Search.calls[0]?.[0] as any;
    expect(query.And[2]).toEqual({ Or: [['UserId', '=', null]] });
  });

  test('load surfaces search failures', async () => {
    Search.setImpl(async () => {
      throw new Error('search failed');
    });
    const api = runHook();
    await api.load();
    expect(api.loadError.value).toBe('search failed');
    expect(api.templates.value).toEqual([]);
    expect(api.loading.value).toBe(false);
  });

  test('load stringifies non-Error throws', async () => {
    Search.setImpl(async () => {
      throw 'plain-string-fail';
    });
    const api = runHook();
    await api.load();
    expect(api.loadError.value).toBe('plain-string-fail');
    expect(api.templates.value).toEqual([]);
  });

  test('load without actor sets canDelete false', async () => {
    actorId = '';
    Search.setImpl(async () => [
      { Id: 's1', Name: 'Shared', Fields: ['Name'], UserId: null, CreatedUid: 'me' },
      { Id: 'p1', Name: 'Private', Fields: ['Code'], UserId: 'someone', CreatedUid: 'someone' },
    ]);
    const api = runHook();
    await api.load();
    expect(api.templates.value.every(row => row.canDelete === false)).toBe(true);
  });

  test('load ignores stale Search rejection after a newer load', async () => {
    let rejectSlow!: (err: Error) => void;
    const slow = new Promise<any[]>((_resolve, reject) => {
      rejectSlow = reject;
    });
    Search.setImpl(() => {
      Search.setImpl(async () => [
        { Id: '2', Name: 'Fresh', Fields: ['Code'], UserId: 'me', CreatedUid: 'me' },
      ]);
      return slow;
    });
    const api = runHook();
    const first = api.load();
    const second = api.load();
    rejectSlow(new Error('stale network'));
    await Promise.all([first, second]);
    expect(api.loadError.value).toBeNull();
    expect(api.templates.value.map(row => row.Id)).toEqual(['2']);
    expect(api.loading.value).toBe(false);
  });

  test('load ignores stale empty-context clear when a newer load starts', async () => {
    const api = runHook('invalid');
    api.templates.value = [{ Id: 'keep', Name: 'Keep', shared: false, createUid: 'me', canDelete: true } as any];
    const first = api.load();
    const second = api.load();
    await Promise.all([first, second]);
    expect(api.templates.value).toEqual([]);
    expect(Search.calls.length).toBe(0);
  });

  test('load ignores stale responses after a newer load starts', async () => {
    let resolveFirst!: (rows: any[]) => void;
    Search.setImpl(
      () =>
        new Promise<any[]>(resolve => {
          resolveFirst = resolve;
          Search.setImpl(async () => [
            { Id: '2', Name: 'Fresh', Fields: ['Code'], UserId: 'me', CreatedUid: 'me' },
          ]);
        })
    );
    const api = runHook();
    const first = api.load();
    const second = api.load();
    resolveFirst([{ Id: '1', Name: 'Stale', Fields: ['Name'], UserId: 'me', CreatedUid: 'me' }]);
    await first;
    await second;
    expect(api.templates.value.map(row => row.Id)).toEqual(['2']);
  });

  test('saveCurrent persists selected fields', async () => {
    const api = runHook();
    const saved = await api.saveCurrent({ name: 'Cols', fields: ['Name', 'Code'] });
    expect(saved?.Name).toBe('Cols');
    expect(Create.calls.length).toBe(1);
    expect(Create.calls[0]![0]).toMatchObject({
      Application: 'partner',
      ModelName: 'Partner',
      Fields: ['Name', 'Code'],
      UserId: 'me',
    });
    expect(Array.isArray(Create.calls[0]![1])).toBe(true);
  });

  test('saveCurrent omits UserId when actor is empty (private)', async () => {
    actorId = '';
    const api = runHook();
    await api.saveCurrent({ name: 'NoActor', fields: ['Name'] });
    expect(Create.calls.length).toBe(1);
    const payload = Create.calls[0]![0] as Record<string, unknown>;
    expect(!('UserId' in payload)).toBe(true);
    expect(Array.isArray(Create.calls[0]![1])).toBe(true);
  });

  test('saveCurrent uses CreatedUid and Name fallbacks from actor', async () => {
    actorId = 'me';
    Create.setImpl(async () => ({
      Id: 'tpl-2',
      Fields: ['Name'],
      UserId: null,
    }));
    const api = runHook();
    const saved = await api.saveCurrent({ name: 'Fallback', shared: true, fields: ['Name'] });
    expect(saved).toMatchObject({
      Id: 'tpl-2',
      Name: 'Fallback',
      shared: true,
      createUid: 'me',
      canDelete: true,
    });
  });

  test('saveCurrent treats omitted fields as empty and returns null', async () => {
    const api = runHook();
    expect(await api.saveCurrent({ name: 'FallbackName', fields: undefined as any })).toBeNull();
    expect(Create.calls.length).toBe(0);
  });

  test('saveCurrent falls back createUid to empty when actor and CreatedUid are missing', async () => {
    actorId = '';
    Create.setImpl(async () => ({
      Id: 'tpl-3',
      Name: '',
      Fields: ['Name'],
      UserId: null,
    }));
    const api = runHook();
    const saved = await api.saveCurrent({ name: 'FallbackName', shared: true, fields: ['Name'] });
    expect(saved).toMatchObject({
      Id: 'tpl-3',
      Name: 'FallbackName',
      createUid: '',
      shared: true,
    });
  });

  test('saveCurrent supports shared and import-compatible templates', async () => {
    const api = runHook();
    const saved = await api.saveCurrent({
      name: 'Shared cols',
      shared: true,
      fields: ['Name'],
      importCompatible: true,
    });
    expect(saved?.shared).toBe(true);
    expect(Create.calls[0]![0]).toMatchObject({
      UserId: null,
      ImportCompatible: true,
    });
    expect(Array.isArray(Create.calls[0]![1])).toBe(true);
  });

  test('saveCurrent returns null for invalid payloads', async () => {
    const api = runHook('invalid');
    expect(await api.saveCurrent({ name: 'x', fields: ['Name'] })).toBeNull();
    expect(await api.saveCurrent({ name: '', fields: ['Name'] })).toBeNull();
    expect(await api.saveCurrent({ name: 'x', fields: [] })).toBeNull();
  });

  test('apply returns ordered field paths', () => {
    const api = runHook();
    expect(api.apply({ Fields: ['Name', 'CompanyId/Code', '  ', null as any] })).toEqual(['Name', 'CompanyId/Code']);
    expect(api.apply({})).toEqual([]);
  });

  test('remove deletes by id and reloads', async () => {
    const api = runHook();
    await api.remove('tpl-1');
    expect(DeleteById.calls.length).toBe(1);
    expect(DeleteById.calls[0]).toEqual(['tpl-1']);
    expect(Search.calls.length).toBeGreaterThan(0);
  });

  test('remove ignores blank ids', async () => {
    const api = runHook();
    await api.remove('   ');
    await api.remove(null as any);
    expect(DeleteById.calls.length).toBe(0);
  });
});
