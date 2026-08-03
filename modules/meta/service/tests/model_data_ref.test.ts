// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { createServiceByModel, registerServiceFactory } from '@/core/service/rpc';
import MetaModelData, { parseMetaModelDataKey } from '../models/model_data';

async function expectRejects(promise: Promise<unknown>, code: string) {
  try {
    await promise;
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof ChoysumError).toBe(true);
    expect((err as ChoysumError).code).toBe(code);
  }
}

test('parseMetaModelDataKey accepts module.name and rejects invalid keys', () => {
  expect(parseMetaModelDataKey(' base.company_main ')).toEqual({ module: 'base', name: 'company_main' });
  // Align with host splitRef (SplitN('.', 2)): dots after the first belong to name.
  expect(parseMetaModelDataKey('foo.bar.baz')).toEqual({ module: 'foo', name: 'bar.baz' });

  for (const bad of ['', '   ', 'a', '.', 'a.', '.b', ' . ', 'mod. ', null, undefined] as any[]) {
    try {
      parseMetaModelDataKey(bad);
      expect(false).toBe(true);
    } catch (err) {
      expect(err instanceof ChoysumError).toBe(true);
      expect((err as ChoysumError).code).toBe('EXTERNAL_ID_INVALID_KEY');
    }
  }
});

test('MetaModelData.Ref returns ResId; RefOrNull returns null; missing Ref raises', async () => {
  const store: Array<{ Module: string; Name: string; Model: string; ResId?: string | null }> = [
    { Module: 'base', Name: 'company_main', Model: 'base.Company', ResId: 'res-company-1' },
    { Module: 'foo', Name: 'bar.baz', Model: 'foo.Thing', ResId: 'res-foo-1' },
    { Module: 'empty', Name: 'res', Model: 'empty.Thing', ResId: '   ' },
    { Module: 'blank', Name: 'res', Model: 'blank.Thing' },
  ];
  const originalSearch = MetaModelData.Search;
  MetaModelData.Search = (async (condition: any) => {
    const and = condition?.And || [];
    const module = and.find((x: any) => x[0] === 'Module')?.[2];
    const name = and.find((x: any) => x[0] === 'Name')?.[2];
    return store.filter(row => row.Module === module && row.Name === name);
  }) as any;

  try {
    expect(await MetaModelData.Ref('base.company_main')).toBe('res-company-1');
    expect(await MetaModelData.Ref('foo.bar.baz')).toBe('res-foo-1');
    expect(await MetaModelData.RefOrNull('base.company_main')).toBe('res-company-1');
    expect(await MetaModelData.RefOrNull('base.missing')).toBeNull();
    // Row present but blank/missing ResId still counts as missing.
    expect(await MetaModelData.RefOrNull('empty.res')).toBeNull();
    expect(await MetaModelData.RefOrNull('blank.res')).toBeNull();
    await expectRejects(MetaModelData.Ref('empty.res'), 'EXTERNAL_ID_NOT_FOUND');
    await expectRejects(MetaModelData.Ref('blank.res'), 'EXTERNAL_ID_NOT_FOUND');
    await expectRejects(MetaModelData.Ref('base.missing'), 'EXTERNAL_ID_NOT_FOUND');
    await expectRejects(MetaModelData.Ref(''), 'EXTERNAL_ID_INVALID_KEY');
    await expectRejects(MetaModelData.Ref('solo'), 'EXTERNAL_ID_INVALID_KEY');
    await expectRejects(MetaModelData.RefOrNull('solo'), 'EXTERNAL_ID_INVALID_KEY');
    expect(await MetaModelData.RefOrNull('base.missing.extra')).toBeNull();
  } finally {
    MetaModelData.Search = originalSearch;
  }
});

test('createServiceByModel(meta.MetaModelData) dials Ref after factory registration', async () => {
  // Production registers meta.MetaModelData via generated service clients
  // (internal/module/artifact/generate/serviceclient.ts.tpl → home/generated/service/meta/service.ts).
  // Unit tests stub a thin factory so Ref can run against a mocked Search without gRPC.
  registerServiceFactory('meta.MetaModelData', () => ({
    Ref: (xmlId: string) => MetaModelData.Ref(xmlId),
    RefOrNull: (xmlId: string) => MetaModelData.RefOrNull(xmlId),
  }));
  const dialed = createServiceByModel('meta.MetaModelData') as {
    Ref: (xmlId: string) => Promise<string>;
    RefOrNull: (xmlId: string) => Promise<string | null>;
  };
  expect(typeof dialed.Ref).toBe('function');
  expect(typeof dialed.RefOrNull).toBe('function');

  const originalSearch = MetaModelData.Search;
  MetaModelData.Search = (async (condition: any) => {
    const and = condition?.And || [];
    const module = and.find((x: any) => x[0] === 'Module')?.[2];
    const name = and.find((x: any) => x[0] === 'Name')?.[2];
    if (module === 'auth' && name === 'user_admin') {
      return [{ ResId: 'user-admin-id' }];
    }
    return [];
  }) as any;

  try {
    expect(await dialed.Ref('auth.user_admin')).toBe('user-admin-id');
    expect(await dialed.RefOrNull('auth.missing')).toBeNull();
  } finally {
    MetaModelData.Search = originalSearch;
  }
});
