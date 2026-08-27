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
  let api!: ReturnType<typeof useExportTemplates>;
  const app = createApp(
    defineComponent({
      setup() {
        api = useExportTemplates(() => model);
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
    ]);
    const api = runHook();
    await api.load();
    expect(templateMocks.Search).toHaveBeenCalled();
    expect(api.templates.value).toHaveLength(2);
    expect(api.templates.value[1].shared).toBe(true);
    expect(api.templates.value[1].canDelete).toBe(false);
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
      }),
      expect.any(Array)
    );
  });

  it('apply returns ordered field paths', () => {
    const api = runHook();
    expect(api.apply({ Fields: ['Name', 'CompanyId/Code'] })).toEqual(['Name', 'CompanyId/Code']);
  });

  it('remove deletes by id and reloads', async () => {
    const api = runHook();
    await api.remove('tpl-1');
    expect(templateMocks.DeleteById).toHaveBeenCalledWith('tpl-1');
    expect(templateMocks.Search).toHaveBeenCalled();
  });
});
