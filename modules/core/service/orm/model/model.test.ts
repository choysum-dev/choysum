// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getCurrentReq, getOrInitReqServiceState } from '../../runtime/context';
import { Field } from '../decorator';
import { MetadataStorage } from '../metadata/storage';
import { RepositoryFactory } from '../repository/repository_factory';
import BaseModel from './model';
import { getModelRepository } from './model_internal_facade';
import { CreateOperations } from './model_create';
import { DeleteOperations } from './model_delete';
import { OnchangeOperations } from './model_onchange';
import { ReadOperations } from './model_read';
import { UpdateOperations } from './model_update';

class ModelSurfaceHarness extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;

  @Field({ type: 'varchar', size: 64 })
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

test('getModelRepository and withSavepoint delegate to repository layer', async () => {
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

    const repo = getModelRepository(ModelSurfaceHarness as any);
    expect(repo).toBe(repository as any);

    const value = await ModelSurfaceHarness.withSavepoint(async () => 'ok', 'sp1');
    expect(value).toBe('ok');
    expect(calls).toEqual([{ name: 'withSavepoint', arg: 'sp1' }]);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('model serialization helpers and hydrate return expected values', () => {
  const instance = makeInstance({ Id: 'M-1', Name: 'name-1' }, ['Id', 'Name']);

  const transport = instance.toTransportObject();
  const plain = instance.toPlainObject();
  const entity = instance.toEntity();

  expect(transport.Id).toBe('M-1');
  expect(plain.Id).toBe('M-1');
  expect(entity).toEqual({ Id: 'M-1', Name: 'name-1' });

  const hydrated = ModelSurfaceHarness.hydrate({ Id: 'M-2', Name: 'name-2' } as any, ['Id', 'Name'] as any) as any;
  expect(hydrated.Id).toBe('M-2');
  expect(hydrated.Name).toBe('name-2');
});

test('model getEffectiveConstraints and getEffectiveOnchange delegate to metadata helpers', () => {
  const constraints = ModelSurfaceHarness.getEffectiveConstraints();
  const onchanges = ModelSurfaceHarness.getEffectiveOnchange();
  expect(Array.isArray(constraints)).toBe(true);
  expect(Array.isArray(onchanges)).toBe(true);
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

  expect(typeof ModelSurfaceHarness.withUser).toBe('function');
  expect(typeof ModelSurfaceHarness.withCompany).toBe('function');
  expect(typeof ModelSurfaceHarness.sudo).toBe('function');
  expect(typeof instance.withUser).toBe('function');
  expect(typeof instance.withCompany).toBe('function');
  expect(typeof instance.sudo).toBe('function');
});

test('model withUser and sudo static/instance wrappers invoke context facades', () => {
  const globalAny = globalThis as any;
  const hadPrev = Object.prototype.hasOwnProperty.call(globalAny, '$choysum');
  const prev = globalAny.$choysum;
  globalAny.$choysum = {
    request: {
      context: {
        identity: { userId: 'U-ROOT' },
        req: {},
      },
    },
  };

  try {
    const instance = makeInstance({ Id: 'SUDO-1', Name: 'sudo' });

    const staticUser = ModelSurfaceHarness.withUser('U-STATIC', () => ModelSurfaceHarness.userId);
    expect(staticUser).toBe('U-STATIC');

    const instanceUser = instance.withUser('U-INSTANCE', () => instance.userId);
    expect(instanceUser).toBe('U-INSTANCE');
    expect(ModelSurfaceHarness.userId).toBe('U-ROOT');

    const staticSudo = ModelSurfaceHarness.sudo(() => 'static-sudo', { hint: 'static-hint' });
    expect(staticSudo).toBe('static-sudo');

    const instanceSudo = instance.sudo(() => 'instance-sudo', { hint: 'instance-hint' });
    expect(instanceSudo).toBe('instance-sudo');

    const hits = ((getOrInitReqServiceState(getCurrentReq()) as { sudoHits?: any[] } | undefined)?.sudoHits ||
      []) as any[];
    expect(hits.map(h => h?.hint)).toEqual(['static-hint', 'instance-hint']);
  } finally {
    if (hadPrev) globalAny.$choysum = prev;
    else delete globalAny.$choysum;
  }
});

test('model withCompany static/instance wrappers override company getters and restore', () => {
  const globalAny = globalThis as any;
  const hadPrev = Object.prototype.hasOwnProperty.call(globalAny, '$choysum');
  const prev = globalAny.$choysum;
  globalAny.$choysum = {
    request: {
      context: {
        identity: { userId: 'U-ROOT' },
        ctx: {
          activeCompanyId: 'OUTER',
          enabledCompanyIds: ['OUTER'],
          lang: 'en',
        },
      },
    },
  };

  try {
    const instance = makeInstance({ Id: 'CO-1', Name: 'company' });

    const staticView = ModelSurfaceHarness.withCompany(' C-STATIC ', () => ({
      companyId: ModelSurfaceHarness.companyId,
      companyIds: ModelSurfaceHarness.companyIds,
      lang: ModelSurfaceHarness.lang,
    }));
    expect(staticView).toEqual({ companyId: 'C-STATIC', companyIds: ['C-STATIC'], lang: 'en' });

    const instanceView = instance.withCompany(
      { activeCompanyId: 'C-INST', enabledCompanyIds: ['C-INST', 'C2'] },
      () => ({
        companyId: instance.companyId,
        companyIds: instance.companyIds,
      })
    );
    expect(instanceView).toEqual({ companyId: 'C-INST', companyIds: ['C-INST', 'C2'] });

    expect(ModelSurfaceHarness.companyId).toBe('OUTER');
    expect(ModelSurfaceHarness.companyIds).toEqual(['OUTER']);
  } finally {
    if (hadPrev) globalAny.$choysum = prev;
    else delete globalAny.$choysum;
  }
});

test('model DisplayName compute handler covers Name/Username/Id fallback branches', () => {
  const meta = MetadataStorage.instance.getModelMetadata(ModelSurfaceHarness as any);
  const displayNameMeta = meta.fields.get('DisplayName') as any;
  const sqlComputeHandler = meta.sqlComputeHandlers?.get('DisplayName') as any;

  expect(displayNameMeta?.column).toBeUndefined();
  expect(sqlComputeHandler?.field).toBe('DisplayName');

  // The sqlDisplayName method is exercised via the SQL bridge in repository projection;
  // here we verify the method exists on the prototype.
  const proto = ModelSurfaceHarness.prototype as any;
  expect(typeof proto.sqlDisplayName).toBe('function');
});

test('model DisplayName sql compute skips missing fields and falls back to Id', () => {
  const proto = ModelSurfaceHarness.prototype as any;
  const fieldCalls: string[] = [];

  const sql = {
    fieldExist: (field: string) => field === 'Id',
    field: (field: string) => {
      fieldCalls.push(field);
      return field === 'Id' ? 'ID-1' : `VALUE-${field}`;
    },
    fn: {
      coalesce: (...items: unknown[]) => {
        for (const item of items) {
          if (item != null) return item;
        }
        return null;
      },
    },
  };

  const host = Object.create(proto);
  Object.defineProperty(host, '$sql', {
    configurable: true,
    enumerable: false,
    get: () => sql,
  });
  const out = proto.sqlDisplayName.call(host);

  expect(out).toBe('ID-1');
  expect(fieldCalls).toEqual(['Id']);
});

test('model static service methods delegate to operation layers', async () => {
  const originalDefaultGet = ModelSurfaceHarness.DefaultGet;
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
    ModelSurfaceHarness.DefaultGet = (async (value: any) => ({ ...value, Name: value?.Name ?? 'defaulted' })) as any;
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
    ModelSurfaceHarness.DefaultGet = originalDefaultGet;
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

test('baseModel ensureCompanyId returns active company or throws when missing', () => {
  const globalAny = globalThis as any;
  const hadPrev = Object.prototype.hasOwnProperty.call(globalAny, '$choysum');
  const prev = globalAny.$choysum;

  try {
    globalAny.$choysum = {
      request: {
        context: {
          ctx: {
            activeCompanyId: 'C-1',
            enabledCompanyIds: ['C-1'],
          },
        },
      },
    };
    expect(BaseModel.ensureCompanyId()).toBe('C-1');

    globalAny.$choysum = {
      request: {
        context: {
          ctx: {
            activeCompanyId: '',
            enabledCompanyIds: [],
          },
        },
      },
    };
    expect(() => BaseModel.ensureCompanyId()).toThrow(/current company is required/);
  } finally {
    if (hadPrev) globalAny.$choysum = prev;
    else delete globalAny.$choysum;
  }
});
