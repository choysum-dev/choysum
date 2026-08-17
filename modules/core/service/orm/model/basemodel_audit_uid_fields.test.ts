// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { setGlobalRequestContextProvider, clearGlobalRequestContextProvider } from '../../../rpc/context';
import { Field } from '../decorator/field';
import { Model } from '../decorator/model';
import { MetadataStorage } from '../metadata/storage';
import { RepositoryFactory } from '../repository/repository_factory';
import BaseModel from './model';

type TestI18nBridge = { t: (...args: string[]) => string };
type TestChoysumRoot = { i18n?: TestI18nBridge; request?: unknown };

const initialI18nBridge = (globalThis as { $choysum?: TestChoysumRoot }).$choysum?.i18n;

function setTestI18nBridge(i18n: TestI18nBridge): void {
  const root = globalThis as { $choysum?: TestChoysumRoot };
  if (!root.$choysum) {
    root.$choysum = {};
  }
  root.$choysum.i18n = i18n;
}

function resetTestState(): void {
  clearGlobalRequestContextProvider();
  const root = globalThis as { $choysum?: TestChoysumRoot };
  if (root.$choysum) {
    if (initialI18nBridge) {
      root.$choysum.i18n = initialI18nBridge;
    } else {
      delete root.$choysum.i18n;
    }
  }
}

@Model('AuditUidFieldsGetProbe', { application: 'core', softDelete: false })
class AuditUidFieldsGetProbe extends BaseModel {
  @Field({ type: 'varchar', size: 64, string: 'Name' })
  Name!: string;
}

test('BaseModel audit uid fields are ManyToOneRef to auth.User with copy:false', () => {
  const meta = MetadataStorage.instance.getModelMetadata(AuditUidFieldsGetProbe as any);
  for (const name of ['CreatedUid', 'UpdatedUid', 'DeletedUid'] as const) {
    const field = meta.fields.get(name);
    expect(field?.type).toBe('ManyToOneRef');
    expect((field as any)?.relation?.targetModel).toBe('auth.User');
    expect(field?.copy).toBe(false);
    // ManyToOneRef forbids onDelete — orphan ids are kept.
    expect((field as any)?.relation?.onDelete).toBeUndefined();
  }
});

test('FieldsGet exposes audit uid fields without requiring auth.User registration', async () => {
  resetTestState();
  setGlobalRequestContextProvider({ lang: 'en_US' });
  setTestI18nBridge({ t: (_m, _l, _s, src) => src });
  RepositoryFactory.setRepository(AuditUidFieldsGetProbe as any, {
    getDenyReadFields: async () => ({ denyReadFields: [] }),
    getDenyWriteFields: async () => ({ denyWriteFields: [] }),
  } as any);

  try {
    const out = await (AuditUidFieldsGetProbe as any).FieldsGet(
      ['CreatedUid', 'UpdatedUid', 'DeletedUid'],
      ['type', 'copy', 'string']
    );
    expect(out.CreatedUid?.type).toBe('ManyToOneRef');
    expect(out.UpdatedUid?.type).toBe('ManyToOneRef');
    expect(out.DeletedUid?.type).toBe('ManyToOneRef');
    expect(out.CreatedUid?.copy).toBe(false);
    expect(out.UpdatedUid?.copy).toBe(false);
    expect(out.DeletedUid?.copy).toBe(false);
  } finally {
    resetTestState();
  }
});
