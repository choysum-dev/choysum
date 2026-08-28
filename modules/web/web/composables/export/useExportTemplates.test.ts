// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createApp, defineComponent, h } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { parseExportModelRef } from './useExportTemplates';

const { templateMocks, actorState } = vi.hoisted(() => ({
  templateMocks: {
    Search: vi.fn(async () => [] as any[]),
    Create: vi.fn(async (values: any) => ({
      Id: 'tpl-1',
      Name: values.Name,
      Fields: values.Fields,
      ImportCompatible: values.ImportCompatible,
      UserId: values.UserId,
      CreatedUid: 'me',
    })),
    DeleteById: vi.fn(async () => 1),
  },
  actorState: { id: 'me' as string },
}));

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: (model: string) => {
    if (model === 'web.ExportTemplate') return templateMocks;
    return {};
  },
}));

vi.mock('@/web/web/composables/search/actorUserId', () => ({
  actorUserId: () => actorState.id,
}));

import { useExportTemplates } from './useExportTemplates';

function runHook(model = 'partner.Partner') {
  return runHookGetter(() => model);
}

function runHookGetter(getModel: () => string) {
  let api!: ReturnType<typeof useExportTemplates>;
  const app = createApp(
    defineComponent({
      setup() {
        api = useExportTemplates(getModel);
        return () => h('div');
      },
    })
  );
  app.mount(document.createElement('div'));
  app.unmount();
  return api;
}

describe('parseExportModelRef', () => {
  it('splits application and model name', () => {
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
    actorState.id = 'me';
    templateMocks.Search.mockReset();
    templateMocks.Create.mockReset();
    templateMocks.DeleteById.mockReset();
    templateMocks.Search.mockResolvedValue([]);
  });

  it('load queries own and shared templates for target model', async () => {
    templateMocks.Search.mockResolvedValue([
      { Id: '1', Name: 'Basic', Fields: ['Name'], UserId: 'me', CreatedUid: 'me' },
      { Id: '2', Name: 'Shared', Fields: ['Code'], UserId: null, CreatedUid: 'other' },
      { Id: '3', Name: '', Fields: ['Name'] },
      { Id: '4', Fields: ['Name'] },
      null,
    ]);
    const api = runHook();
    await api.load();
    expect(templateMocks.Search).toHaveBeenCalled();
    expect(api.templates.value).toHaveLength(2);
    expect(api.templates.value[0].canDelete).toBe(true);
    expect(api.templates.value[1].shared).toBe(true);
    expect(api.templates.value[1].canDelete).toBe(false);
  });

  it('load maps missing CreatedUid and treats null Search rows as empty', async () => {
    templateMocks.Search.mockResolvedValue(null);
    const api = runHook();
    await api.load();
    expect(api.templates.value).toEqual([]);

    templateMocks.Search.mockResolvedValue([
      { Id: 's1', Name: 'NoCreatedUid', Fields: ['Name'], UserId: null },
    ]);
    await api.load();
    expect(api.templates.value[0]).toMatchObject({ createUid: '', canDelete: false });
  });

  it('load clears loading when a newer load invalidates the model ref', async () => {
    let resolveSlow!: (rows: any[]) => void;
    const slow = new Promise<any[]>(resolve => {
      resolveSlow = resolve;
    });
    templateMocks.Search.mockImplementationOnce(() => slow);
    let model = 'partner.Partner';
    const api = runHookGetter(() => model);
    const first = api.load();
    expect(api.loading.value).toBe(true);
    model = 'invalid';
    const second = api.load();
    resolveSlow([{ Id: 'stale', Name: 'Stale', Fields: ['Name'], UserId: 'me', CreatedUid: 'me' }]);
    await Promise.all([first, second]);
    expect(api.templates.value).toEqual([]);
    expect(api.loading.value).toBe(false);
    expect(api.loadError.value).toBeNull();
  });

  it('load maps shared/private canDelete and normalizes Fields', async () => {
    templateMocks.Search.mockResolvedValue([
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

  it('load clears templates for invalid model refs', async () => {
    const api = runHook('invalid');
    await api.load();
    expect(templateMocks.Search).not.toHaveBeenCalled();
    expect(api.templates.value).toEqual([]);
    expect(api.loading.value).toBe(false);
    expect(api.loadError.value).toBeNull();
  });

  it('load clears templates for empty model refs', async () => {
    const api = runHook('');
    await api.load();
    expect(templateMocks.Search).not.toHaveBeenCalled();
    expect(api.templates.value).toEqual([]);
  });

  it('load treats null model ref as invalid', async () => {
    const api = runHookGetter(() => null as any);
    await api.load();
    expect(templateMocks.Search).not.toHaveBeenCalled();
    expect(api.templates.value).toEqual([]);
  });

  it('load without actor only queries shared templates', async () => {
    actorState.id = '';
    const api = runHook();
    await api.load();
    const query = templateMocks.Search.mock.calls[0]?.[0];
    expect(query.And[2]).toEqual({ Or: [['UserId', '=', null]] });
  });

  it('load surfaces search failures', async () => {
    templateMocks.Search.mockRejectedValue(new Error('search failed'));
    const api = runHook();
    await api.load();
    expect(api.loadError.value).toBe('search failed');
    expect(api.templates.value).toEqual([]);
    expect(api.loading.value).toBe(false);
  });

  it('load stringifies non-Error throws', async () => {
    templateMocks.Search.mockRejectedValue('plain-string-fail');
    const api = runHook();
    await api.load();
    expect(api.loadError.value).toBe('plain-string-fail');
    expect(api.templates.value).toEqual([]);
  });

  it('load without actor sets canDelete false', async () => {
    actorState.id = '';
    templateMocks.Search.mockResolvedValue([
      { Id: 's1', Name: 'Shared', Fields: ['Name'], UserId: null, CreatedUid: 'me' },
      { Id: 'p1', Name: 'Private', Fields: ['Code'], UserId: 'someone', CreatedUid: 'someone' },
    ]);
    const api = runHook();
    await api.load();
    expect(api.templates.value.every(row => row.canDelete === false)).toBe(true);
  });

  it('load ignores stale Search rejection after a newer load', async () => {
    let rejectSlow!: (err: Error) => void;
    const slow = new Promise<any[]>((_resolve, reject) => {
      rejectSlow = reject;
    });
    templateMocks.Search
      .mockImplementationOnce(() => slow)
      .mockResolvedValueOnce([{ Id: '2', Name: 'Fresh', Fields: ['Code'], UserId: 'me', CreatedUid: 'me' }]);
    const api = runHook();
    const first = api.load();
    const second = api.load();
    rejectSlow(new Error('stale network'));
    await Promise.all([first, second]);
    expect(api.loadError.value).toBeNull();
    expect(api.templates.value.map(row => row.Id)).toEqual(['2']);
    expect(api.loading.value).toBe(false);
  });

  it('load ignores stale empty-context clear when a newer load starts', async () => {
    const api = runHook('invalid');
    api.templates.value = [{ Id: 'keep', Name: 'Keep', shared: false, createUid: 'me', canDelete: true } as any];
    const first = api.load();
    const second = api.load();
    await Promise.all([first, second]);
    expect(api.templates.value).toEqual([]);
    expect(templateMocks.Search).not.toHaveBeenCalled();
  });

  it('load ignores stale responses after a newer load starts', async () => {
    let resolveFirst!: (rows: any[]) => void;
    templateMocks.Search
      .mockImplementationOnce(
        () =>
          new Promise<any[]>(resolve => {
            resolveFirst = resolve;
          })
      )
      .mockResolvedValueOnce([{ Id: '2', Name: 'Fresh', Fields: ['Code'], UserId: 'me', CreatedUid: 'me' }]);
    const api = runHook();
    const first = api.load();
    const second = api.load();
    resolveFirst([{ Id: '1', Name: 'Stale', Fields: ['Name'], UserId: 'me', CreatedUid: 'me' }]);
    await first;
    await second;
    expect(api.templates.value.map(row => row.Id)).toEqual(['2']);
  });

  it('saveCurrent persists selected fields', async () => {
    const api = runHook();
    const saved = await api.saveCurrent({ name: 'Cols', fields: ['Name', 'Code'] });
    expect(saved?.Name).toBe('Cols');
    expect(templateMocks.Create).toHaveBeenCalledWith(
      expect.objectContaining({
        Application: 'partner',
        ModelName: 'Partner',
        Fields: ['Name', 'Code'],
        UserId: 'me',
      }),
      expect.any(Array)
    );
  });

  it('saveCurrent omits UserId when actor is empty (private)', async () => {
    actorState.id = '';
    const api = runHook();
    await api.saveCurrent({ name: 'NoActor', fields: ['Name'] });
    expect(templateMocks.Create).toHaveBeenCalledWith(
      expect.not.objectContaining({ UserId: expect.anything() }),
      expect.any(Array)
    );
  });

  it('saveCurrent uses CreatedUid and Name fallbacks from actor', async () => {
    actorState.id = 'me';
    templateMocks.Create.mockResolvedValueOnce({
      Id: 'tpl-2',
      Fields: ['Name'],
      UserId: null,
    });
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

  it('saveCurrent treats omitted fields as empty and returns null', async () => {
    const api = runHook();
    expect(await api.saveCurrent({ name: 'FallbackName', fields: undefined as any })).toBeNull();
    expect(templateMocks.Create).not.toHaveBeenCalled();
  });

  it('saveCurrent falls back createUid to empty when actor and CreatedUid are missing', async () => {
    actorState.id = '';
    templateMocks.Create.mockResolvedValueOnce({
      Id: 'tpl-3',
      Name: '',
      Fields: ['Name'],
      UserId: null,
    });
    const api = runHook();
    const saved = await api.saveCurrent({ name: 'FallbackName', shared: true, fields: ['Name'] });
    expect(saved).toMatchObject({
      Id: 'tpl-3',
      Name: 'FallbackName',
      createUid: '',
      shared: true,
    });
  });

  it('saveCurrent supports shared and import-compatible templates', async () => {
    const api = runHook();
    const saved = await api.saveCurrent({
      name: 'Shared cols',
      shared: true,
      fields: ['Name'],
      importCompatible: true,
    });
    expect(saved?.shared).toBe(true);
    expect(templateMocks.Create).toHaveBeenCalledWith(
      expect.objectContaining({
        UserId: null,
        ImportCompatible: true,
      }),
      expect.any(Array)
    );
  });

  it('saveCurrent returns null for invalid payloads', async () => {
    const api = runHook('invalid');
    expect(await api.saveCurrent({ name: 'x', fields: ['Name'] })).toBeNull();
    expect(await api.saveCurrent({ name: '', fields: ['Name'] })).toBeNull();
    expect(await api.saveCurrent({ name: 'x', fields: [] })).toBeNull();
  });

  it('apply returns ordered field paths', () => {
    const api = runHook();
    expect(api.apply({ Fields: ['Name', 'CompanyId/Code', '  ', null as any] })).toEqual(['Name', 'CompanyId/Code']);
    expect(api.apply({})).toEqual([]);
  });

  it('remove deletes by id and reloads', async () => {
    const api = runHook();
    await api.remove('tpl-1');
    expect(templateMocks.DeleteById).toHaveBeenCalledWith('tpl-1');
    expect(templateMocks.Search).toHaveBeenCalled();
  });

  it('remove ignores blank ids', async () => {
    const api = runHook();
    await api.remove('   ');
    await api.remove(null as any);
    expect(templateMocks.DeleteById).not.toHaveBeenCalled();
  });
});
