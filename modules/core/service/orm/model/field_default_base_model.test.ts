// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field, Model } from '../decorator';
import BaseModel from './model';
import FieldDefaultBaseModel from './field_default_base_model';
import { ChoysumError } from '@/core/service/error';

@Model('Widget', { application: 'fd2test' })
class Fd2Widget extends BaseModel {
  @Field({ type: 'varchar', size: 64, default: 'from-column' })
  Name!: string;

  @Field({ type: 'varchar', size: 64 })
  Code!: string;

  @Field({ type: 'OneToMany', relation: { targetModel: () => Fd2Widget, related: 'ParentId' } as any })
  Lines!: any;

  @Field({ type: 'binary' })
  Blob!: any;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => Fd2Widget } as any })
  ParentId?: Fd2Widget;
}

@Model('FieldDefault', { application: 'fd2test' })
class Fd2FieldDefault extends FieldDefaultBaseModel {}

@Model('Other', { application: 'otherapp' })
class OtherAppModel extends BaseModel {
  @Field({ type: 'varchar', size: 32 })
  Name!: string;
}

function expectCode(err: unknown, code: string) {
  expect(err instanceof ChoysumError).toBe(true);
  expect((err as ChoysumError).code).toBe(code);
}

test('FieldDefault Set upserts and Get reads exact scope; userId true uses context', async () => {
  const store: any[] = [];
  const originalSearch = Fd2FieldDefault.Search;
  const originalCreate = Fd2FieldDefault.Create;
  const originalUpdateById = Fd2FieldDefault.UpdateById;
  const originalWithSavepoint = Fd2FieldDefault.withSavepoint;
  const originalEnsure = (Fd2FieldDefault as any).ensureScopeUniqueIndex;

  (Fd2FieldDefault as any).ensureScopeUniqueIndex = async () => {};
  Fd2FieldDefault.withSavepoint = (async (fn: any) => await fn()) as any;
  Fd2FieldDefault.Search = (async (condition: any) => {
    const and = condition?.And || [];
    const model = and.find((x: any) => x[0] === 'Model')?.[2];
    const field = and.find((x: any) => x[0] === 'Field')?.[2];
    const userClause = and.find((x: any) => x[0] === 'UserId');
    const companyClause = and.find((x: any) => x[0] === 'CompanyId');
    const userId = userClause?.[1] === 'is' ? null : userClause?.[2];
    const companyId = companyClause?.[1] === 'is' ? null : companyClause?.[2];
    return store.filter(
      row => row.Model === model && row.Field === field && row.UserId === userId && row.CompanyId === companyId
    );
  }) as any;
  Fd2FieldDefault.Create = (async (value: any) => {
    const row = { Id: `FD-${store.length + 1}`, ...value };
    store.push(row);
    return row;
  }) as any;
  Fd2FieldDefault.UpdateById = (async (id: string, value: any) => {
    const row = store.find(r => r.Id === id);
    Object.assign(row, value);
    return row;
  }) as any;

  try {
    await Fd2FieldDefault.Set('Widget', 'Name', 'global-name');
    expect(await Fd2FieldDefault.Get('Widget', 'Name')).toBe('global-name');

    await Fd2FieldDefault.withUser('U-9', async () => {
      await Fd2FieldDefault.Set('Widget', 'Name', 'user-name', { userId: true });
      expect(await Fd2FieldDefault.Get('Widget', 'Name', { userId: true })).toBe('user-name');
    });

    await Fd2FieldDefault.Set('Widget', 'Name', 'global-name-2');
    expect(store.filter(r => r.UserId == null).length).toBe(1);
    expect(await Fd2FieldDefault.Get('Widget', 'Name')).toBe('global-name-2');
  } finally {
    Fd2FieldDefault.Search = originalSearch;
    Fd2FieldDefault.Create = originalCreate;
    Fd2FieldDefault.UpdateById = originalUpdateById;
    Fd2FieldDefault.withSavepoint = originalWithSavepoint;
    (Fd2FieldDefault as any).ensureScopeUniqueIndex = originalEnsure;
  }
});

test('FieldDefault Get returns undefined when no row', async () => {
  const originalSearch = Fd2FieldDefault.Search;
  Fd2FieldDefault.Search = (async () => []) as any;
  try {
    expect(await Fd2FieldDefault.Get('Widget', 'Code')).toBeUndefined();
  } finally {
    Fd2FieldDefault.Search = originalSearch;
  }
});

test('FieldDefault GetEffective merges by scope priority', async () => {
  const originalSearch = Fd2FieldDefault.Search;
  Fd2FieldDefault.Search = (async () => [
    { Id: '1', Field: 'Name', UserId: null, CompanyId: null, Value: 'global' },
    { Id: '2', Field: 'Name', UserId: 'U1', CompanyId: null, Value: 'user' },
    { Id: '3', Field: 'Code', UserId: null, CompanyId: 'C1', Value: 'co-code' },
  ]) as any;

  try {
    const out = await Fd2FieldDefault.withUser('U1', () =>
      Fd2FieldDefault.withCompany('C1', () => Fd2FieldDefault.GetEffective('Widget'))
    );
    expect(out.Name).toBe('user');
    expect(out.Code).toBe('co-code');
  } finally {
    Fd2FieldDefault.Search = originalSearch;
  }
});

test('FieldDefault Unset deletes exact scope row', async () => {
  const store: any[] = [{ Id: 'FD-1', Model: 'Widget', Field: 'Name', UserId: null, CompanyId: null, Value: 'x' }];
  const originalSearch = Fd2FieldDefault.Search;
  const originalDeleteById = Fd2FieldDefault.DeleteById;
  Fd2FieldDefault.Search = (async () => store.slice()) as any;
  Fd2FieldDefault.DeleteById = (async (id: string) => {
    const idx = store.findIndex(r => r.Id === id);
    if (idx >= 0) store.splice(idx, 1);
    return 1;
  }) as any;

  try {
    await Fd2FieldDefault.Unset('Widget', 'Name');
    expect(store.length).toBe(0);
  } finally {
    Fd2FieldDefault.Search = originalSearch;
    Fd2FieldDefault.DeleteById = originalDeleteById;
  }
});

test('FieldDefault Set rejects unsupported types and cross-app models', async () => {
  try {
    await Fd2FieldDefault.Set('Widget', 'Lines', []);
    expect(false).toBe(true);
  } catch (err) {
    expectCode(err, 'FIELD_DEFAULT_INVALID_VALUE');
  }

  try {
    await Fd2FieldDefault.Set('Widget', 'Blob', 'x');
    expect(false).toBe(true);
  } catch (err) {
    expectCode(err, 'FIELD_DEFAULT_INVALID_VALUE');
  }

  try {
    await Fd2FieldDefault.Set('Widget', 'Missing', 'x');
    expect(false).toBe(true);
  } catch (err) {
    expectCode(err, 'FIELD_DEFAULT_UNKNOWN_FIELD');
  }

  try {
    await Fd2FieldDefault.Set('Other', 'Name', 'x');
    expect(false).toBe(true);
  } catch (err) {
    expectCode(err, 'FIELD_DEFAULT_CROSS_APP_MODEL');
  }

  expect(OtherAppModel).toBeTruthy();
  expect(Fd2Widget).toBeTruthy();
});

test('FieldDefault Set normalizes ManyToOne values to id strings', async () => {
  const created: any[] = [];
  const originalSearch = Fd2FieldDefault.Search;
  const originalCreate = Fd2FieldDefault.Create;
  const originalWithSavepoint = Fd2FieldDefault.withSavepoint;
  const originalEnsure = (Fd2FieldDefault as any).ensureScopeUniqueIndex;

  (Fd2FieldDefault as any).ensureScopeUniqueIndex = async () => {};
  Fd2FieldDefault.withSavepoint = (async (fn: any) => await fn()) as any;
  Fd2FieldDefault.Search = (async () => []) as any;
  Fd2FieldDefault.Create = (async (value: any) => {
    created.push(value);
    return { Id: 'FD-m2o', ...value };
  }) as any;

  try {
    await Fd2FieldDefault.Set('Widget', 'ParentId', { Id: 'W-1' });
    expect(created[0]?.Value).toBe('W-1');
  } finally {
    Fd2FieldDefault.Search = originalSearch;
    Fd2FieldDefault.Create = originalCreate;
    Fd2FieldDefault.withSavepoint = originalWithSavepoint;
    (Fd2FieldDefault as any).ensureScopeUniqueIndex = originalEnsure;
  }
});
