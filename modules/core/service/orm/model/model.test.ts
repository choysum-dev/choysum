// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field } from '../decorator';
import { MetadataStorage } from '../metadata/storage';
import { RepositoryFactory } from '../repository/repository_factory';
import BaseModel from './model';
import { CreateOperations } from './model_create';
import { DefaultOperations } from './model_default';
import { DeleteOperations } from './model_delete';
import { OnchangeOperations } from './model_onchange';
import { ReadOperations } from './model_read';
import { UpdateOperations } from './model_update';

class ModelSurfaceHarness extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name!: string;

  @Field({ type: 'varchar', column: { size: 64 } })
  Username!: string;
}

function makeInstance(entity: Record<string, any>, fields?: any) {
  const token = (BaseModel as any).FACTORY_TOKEN as symbol;
  return new ModelSurfaceHarness(token, entity as any, fields) as any;
}

test('model constructor rejects direct instantiation without factory token', () => {
  let message = '';
  try {
    new (ModelSurfaceHarness as any)(Symbol('wrong-token'), {} as any);
  } catch (error) {
    message = String((error as Error).message || error);
  }
  expect(message).toContain('Models cannot be directly instantiated');
});

test('model instance CRUD wrappers delegate and keep missing-id guard behavior', async () => {
  const instance = makeInstance({ Name: 'x' });

  let updateErr = '';
  try {
    await instance.update();
  } catch (error) {
    updateErr = String((error as Error).message || error);
  }
  expect(updateErr).toContain('Cannot update an instance without Id');

  let deleteErr = '';
  try {
    await instance.delete();
  } catch (error) {
    deleteErr = String((error as Error).message || error);
  }
  expect(deleteErr).toContain('Cannot delete an instance without Id');

  let loadErr = '';
  try {
    await instance.load();
  } catch (error) {
    loadErr = String((error as Error).message || error);
  }
  expect(loadErr).toContain('Cannot load an instance without Id');

  let reloadErr = '';
  try {
    await instance.reload();
  } catch (error) {
    reloadErr = String((error as Error).message || error);
  }
  expect(reloadErr).toContain('Cannot reload an instance without Id');
});

test('model static CreateMany handles empty payload fast path', async () => {
  const out = await ModelSurfaceHarness.CreateMany([] as any);
  expect(out).toEqual([]);
});

test('model static getRepository and withSavepoint delegate to repository layer', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const calls: Array<{ name: string; arg?: any }> = [];
  const repository = {
    withSavepoint: async (fn: () => Promise<string>, name?: string) => {
      calls.push({ name: 'withSavepoint', arg: name });
      return await fn();
    },
  };

  try {
    RepositoryFactory.getRepository = (() => repository as any) as any;

    const repo = ModelSurfaceHarness.getRepository();
    expect(repo).toBe(repository as any);

    const value = await ModelSurfaceHarness.withSavepoint(async () => 'ok', 'sp1');
    expect(value).toBe('ok');
    expect(calls).toEqual([{ name: 'withSavepoint', arg: 'sp1' }]);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model serialization helpers and Hydrate return expected values', () => {
  const instance = makeInstance({ Id: 'M-1', Name: 'name-1' }, ['Id', 'Name']);

  const transport = instance.toTransportObject();
  const plain = instance.toPlainObject();
  const entity = instance.toEntity();

  expect(transport.Id).toBe('M-1');
  expect(plain.Id).toBe('M-1');
  expect(entity).toEqual({ Id: 'M-1', Name: 'name-1' });

  const hydrated = ModelSurfaceHarness.Hydrate({ Id: 'M-2', Name: 'name-2' } as any, ['Id', 'Name'] as any) as any;
  expect(hydrated.Id).toBe('M-2');
  expect(hydrated.Name).toBe('name-2');
});

test('model context accessors are available on static and instance surfaces', () => {
  const instance = makeInstance({ Id: 'CTX-1', Name: 'ctx' });

  expect(ModelSurfaceHarness.ctx).toBeDefined();
  expect(instance.ctx).toBeDefined();

  expect(instance.companyId).toBe(ModelSurfaceHarness.companyId);
  expect(instance.companyIds).toEqual(ModelSurfaceHarness.companyIds);
  expect(instance.lang).toBe(ModelSurfaceHarness.lang);
  expect(instance.tz).toBe(ModelSurfaceHarness.tz);
  expect(instance.userId).toBe(ModelSurfaceHarness.userId);

  const staticWithContext = ModelSurfaceHarness.withContext({ lang: 'zh-CN' } as any, () => 'ok', { merge: true });
  const instanceWithContext = instance.withContext({ lang: 'zh-CN' } as any, () => 'ok', { merge: true });
  expect(staticWithContext).toBe('ok');
  expect(instanceWithContext).toBe('ok');
});

test('model DisplayName compute handler covers Name/Username/Id fallback branches', () => {
  const meta = MetadataStorage.instance.getModelMetadata(ModelSurfaceHarness as any);
  const displayNameMeta = meta.fields.get('DisplayName') as any;
  const computeHandler = meta.computeHandlers?.get('DisplayName') as any;

  expect(displayNameMeta?.column).toBeUndefined();
  expect(computeHandler?.store).toBe(false);

  const byName = makeInstance({ Id: 'I-1', Name: 'N-1' }).computeDisplayName();
  const byUsername = makeInstance({ Id: 'I-2', Username: 'U-2' } as any).computeDisplayName();
  const byId = makeInstance({ Id: 'I-3' }).computeDisplayName();

  expect(byName).toBe('N-1');
  expect(byUsername).toBe('U-2');
  expect(byId).toBe('I-3');
});

test('model static service methods delegate to operation layers', async () => {
  const originalDefaultGet = DefaultOperations.DefaultGet;
  const originalCreate = CreateOperations.Create;
  const originalCreateMany = CreateOperations.CreateMany;
  const originalBrowse = ReadOperations.Browse;
  const originalSearch = ReadOperations.Search;
  const originalCount = ReadOperations.Count;
  const originalReadGroup = ReadOperations.ReadGroup;
  const originalReadGroupCount = ReadOperations.ReadGroupCount;
  const originalUpdate = UpdateOperations.Update;
  const originalUpdateById = UpdateOperations.UpdateById;
  const originalDelete = DeleteOperations.Delete;
  const originalDeleteById = DeleteOperations.DeleteById;
  const originalOnchange = OnchangeOperations.Onchange;

  try {
    DefaultOperations.DefaultGet = (async (_ModelCtor: any, value: any) => ({ ...value, Name: value?.Name ?? 'defaulted' })) as any;
    CreateOperations.Create = (async (_ModelCtor: any, value: any) => ({ Id: value?.Id || 'C-1', Name: value?.Name || 'created' })) as any;
    CreateOperations.CreateMany = (async (_ModelCtor: any, values: any[]) =>
      values.map((v, i) => ({ Id: v?.Id || `CM-${i + 1}`, Name: v?.Name || 'created' }))) as any;

    ReadOperations.Browse = (async (_ModelCtor: any, id: string) => ({ Id: id, Name: 'browsed' })) as any;
    ReadOperations.Search = (async (_ModelCtor: any, condition: any) => {
      if (Array.isArray(condition) && condition[1] === 'in' && Array.isArray(condition[2])) {
        return condition[2].map((id: string) => ({ Id: id, Name: `s-${id}` }));
      }
      return [{ Id: 'S-1', Name: 'searched' }];
    }) as any;
    ReadOperations.Count = (async () => 7) as any;
    ReadOperations.ReadGroup = (async () => [{ depth: 0, keys: {}, labels: {}, metrics: {}, count: 1 }]) as any;
    ReadOperations.ReadGroupCount = (async () => 9) as any;

    UpdateOperations.Update = (async () => [{ Id: 'U-1' }]) as any;
    UpdateOperations.UpdateById = (async () => ({ Id: 'U-2' })) as any;
    DeleteOperations.Delete = (async () => 3) as any;
    DeleteOperations.DeleteById = (async () => 1) as any;

    OnchangeOperations.Onchange = (async () => ({
      values: { Name: 'new-name' },
      changedFields: ['Name'],
      diagnostics: {},
    })) as any;

    const defaulted = await ModelSurfaceHarness.DefaultGet({} as any);
    expect((defaulted as any).Name).toBe('defaulted');

    const created = await ModelSurfaceHarness.Create({ Id: 'C-1', Name: 'n1' } as any, ['Id', 'Name'] as any);
    expect((created as any).Id).toBe('C-1');

    const createdMany = await ModelSurfaceHarness.CreateMany([{ Id: 'CM-1' }, { Id: 'CM-2' }] as any, ['Id'] as any);
    expect(createdMany.length).toBe(2);

    const browsed = await ModelSurfaceHarness.Browse('B-1', ['Id', 'Name'] as any);
    expect((browsed as any).Id).toBe('B-1');

    const browsedMany = await ModelSurfaceHarness.BrowseMany(['BM-1', 'BM-2'] as any, ['Id'] as any);
    expect(browsedMany.length).toBe(2);

    const searched = await ModelSurfaceHarness.Search([] as any, { fields: ['Id'] } as any);
    expect(searched.length).toBe(1);

    const counted = await ModelSurfaceHarness.Count([] as any);
    expect(counted).toBe(7);

    const grouped = await ModelSurfaceHarness.ReadGroup(['Id'] as any, [] as any, {} as any);
    expect(Array.isArray(grouped)).toBe(true);

    const groupedCount = await ModelSurfaceHarness.ReadGroupCount(['Id'] as any, [] as any, {} as any);
    expect(groupedCount).toBe(9);

    const updated = await ModelSurfaceHarness.Update(['Id', '=', 'U-1'] as any, { Name: 'u' } as any, ['Id'] as any);
    expect(updated.length).toBe(1);

    const updatedById = await ModelSurfaceHarness.UpdateById('U-2', { Name: 'u2' } as any, ['Id'] as any);
    expect((updatedById as any).Id).toBe('U-2');

    const deleted = await ModelSurfaceHarness.Delete(['Id', '=', 'D-1'] as any);
    expect(deleted).toBe(3);

    const deletedById = await ModelSurfaceHarness.DeleteById('D-2');
    expect(deletedById).toBe(1);

    const onchange = await ModelSurfaceHarness.Onchange({ Name: 'a' }, ['Name'] as any, { withCompute: true });
    expect((onchange as any).values?.Name).toBe('new-name');
  } finally {
    DefaultOperations.DefaultGet = originalDefaultGet;
    CreateOperations.Create = originalCreate;
    CreateOperations.CreateMany = originalCreateMany;
    ReadOperations.Browse = originalBrowse;
    ReadOperations.Search = originalSearch;
    ReadOperations.Count = originalCount;
    ReadOperations.ReadGroup = originalReadGroup;
    ReadOperations.ReadGroupCount = originalReadGroupCount;
    UpdateOperations.Update = originalUpdate;
    UpdateOperations.UpdateById = originalUpdateById;
    DeleteOperations.Delete = originalDelete;
    DeleteOperations.DeleteById = originalDeleteById;
    OnchangeOperations.Onchange = originalOnchange;
  }
});

test('model read APIs apply default arguments when omitted', async () => {
  const originalSearch = ReadOperations.Search;
  const originalCount = ReadOperations.Count;
  const originalReadGroup = ReadOperations.ReadGroup;
  const originalReadGroupCount = ReadOperations.ReadGroupCount;

  const calls: Array<{ name: string; args: any[] }> = [];

  try {
    ReadOperations.Search = (async (...args: any[]) => {
      calls.push({ name: 'Search', args });
      return [];
    }) as any;
    ReadOperations.Count = (async (...args: any[]) => {
      calls.push({ name: 'Count', args });
      return 0;
    }) as any;
    ReadOperations.ReadGroup = (async (...args: any[]) => {
      calls.push({ name: 'ReadGroup', args });
      return [];
    }) as any;
    ReadOperations.ReadGroupCount = (async (...args: any[]) => {
      calls.push({ name: 'ReadGroupCount', args });
      return 0;
    }) as any;

    await ModelSurfaceHarness.Search();
    await ModelSurfaceHarness.Count();
    await ModelSurfaceHarness.ReadGroup([] as any);
    await ModelSurfaceHarness.ReadGroupCount([] as any);

    expect(calls[0].name).toBe('Search');
    expect(calls[0].args[1]).toEqual([]);
    expect(calls[0].args[2]).toBeUndefined();

    expect(calls[1].name).toBe('Count');
    expect(calls[1].args[1]).toEqual([]);
    expect(calls[1].args[2]).toBeUndefined();

    expect(calls[2].name).toBe('ReadGroup');
    expect(calls[2].args[1]).toEqual([]);
    expect(calls[2].args[2]).toEqual([]);
    expect(calls[2].args[3]).toEqual({});

    expect(calls[3].name).toBe('ReadGroupCount');
    expect(calls[3].args[1]).toEqual([]);
    expect(calls[3].args[2]).toEqual([]);
    expect(calls[3].args[3]).toEqual({});
  } finally {
    ReadOperations.Search = originalSearch;
    ReadOperations.Count = originalCount;
    ReadOperations.ReadGroup = originalReadGroup;
    ReadOperations.ReadGroupCount = originalReadGroupCount;
  }
});

test('baseModel resolveModelConstructor returns undefined for empty or unknown keys', () => {
  expect(BaseModel.resolveModelConstructor('')).toBe(undefined);
  expect(BaseModel.resolveModelConstructor('   ')).toBe(undefined);
  expect(BaseModel.resolveModelConstructor('__completely_unknown_model__')).toBe(undefined);
});

test('baseModel resolveModelConstructor resolves by fullModelName, modelName, name, and className', () => {
  const storage = MetadataStorage.instance as any;
  const savedModels = storage.models;

  class ResolveTestModel extends BaseModel {}
  const testCtor = ResolveTestModel as any;

  try {
    const models = new Map();
    models.set(testCtor, {
      fullModelName: 'test.ResolveModel',
      modelName: 'ResolveModel',
      name: 'TestResolveModelShort',
    });
    storage.models = models;

    expect(BaseModel.resolveModelConstructor('test.ResolveModel')).toBe(testCtor);
    expect(BaseModel.resolveModelConstructor('ResolveModel')).toBe(testCtor);
    expect(BaseModel.resolveModelConstructor('TestResolveModelShort')).toBe(testCtor);
    expect(BaseModel.resolveModelConstructor('ResolveTestModel')).toBe(testCtor); // className
  } finally {
    storage.models = savedModels;
  }
});
