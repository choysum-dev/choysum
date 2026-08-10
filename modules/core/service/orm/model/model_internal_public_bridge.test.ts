// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from './model';
import { CreateOperations } from './model_create';
import { UpdateOperations } from './model_update';
import { Field, Model } from '../decorator';
import { RelationFactory } from '../relation';
import { RepositoryFactory } from '../repository/repository_factory';
import { DefaultOperations } from './model_default';
import { ReadOperations } from './model_read';
import { ComputeCascadeEngine } from '../../runtime/compute/cascade';
import { ComputeEngine } from '../../runtime/compute/engine';
import { MetadataStorage } from '../metadata/storage';

@Model('test.ModelInternalPublicBridge')
class ModelInternalPublicBridge extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;
}

function unexpectedPublicCall(name: string) {
  return (async () => {
    throw new Error(`public ${name} should not be called`);
  }) as any;
}

function emptyToManyRelations() {
  return {
    oneToManyRelations: [],
    manyToManyRelations: [],
    touchedCollections: new Set<string>(),
  };
}

test('CreateOperations.Create uses ModelCtor.DefaultGet and internal Browse helper', async () => {
  const originalDefaultGetMethod = ModelInternalPublicBridge.DefaultGet;
  const originalBrowseMethod = ModelInternalPublicBridge.Browse;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalBrowse = ReadOperations.Browse;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;

  const defaultCalls: any[] = [];
  const browseCalls: any[] = [];
  const createdRows: any[] = [];

  try {
    ModelInternalPublicBridge.DefaultGet = (async function (this: any, value: any) {
      defaultCalls.push({ ModelCtor: this, value });
      return { ...value, Name: value?.Name ?? 'defaulted' };
    }) as any;
    ModelInternalPublicBridge.Browse = unexpectedPublicCall('Browse');

    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value },
      relations: emptyToManyRelations(),
    })) as any;

    RepositoryFactory.getRepository = (() => ({
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
      create: async (rows: any[]) => {
        createdRows.push(...rows);
        return rows.map(row => row.Id);
      },
    })) as any;

    ReadOperations.Browse = (async (ModelCtor: any, id: string, fields?: any) => {
      browseCalls.push({ ModelCtor, id, fields });
      return { Id: id, Name: 'created' };
    }) as any;

    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;

    const result = await CreateOperations.Create(ModelInternalPublicBridge as any, { Id: 'NEW-1' } as any, ['Id', 'Name'] as any);

    expect(result instanceof ModelInternalPublicBridge).toBe(true);
    expect((result as any).Id).toBe('NEW-1');
    expect((result as any).Name).toBe('created');
    expect(defaultCalls.length).toBe(1);
    expect(defaultCalls[0]?.ModelCtor).toBe(ModelInternalPublicBridge);
    expect(browseCalls.length).toBe(1);
    expect(browseCalls[0]?.id).toBe('NEW-1');
    expect(browseCalls[0]?.fields).toEqual(['Id', 'Name']);
    expect(createdRows.length).toBe(1);
    expect(createdRows[0]?.Id).toBe('NEW-1');
    expect(createdRows[0]?.Name).toBe('defaulted');
  } finally {
    ModelInternalPublicBridge.DefaultGet = originalDefaultGetMethod;
    ModelInternalPublicBridge.Browse = originalBrowseMethod;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RepositoryFactory.getRepository = originalGetRepository;
    ReadOperations.Browse = originalBrowse;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
  }
});

test('CreateOperations.CreateMany uses ModelCtor.DefaultGet and internal Search helper', async () => {
  const originalDefaultGetMethod = ModelInternalPublicBridge.DefaultGet;
  const originalSearchMethod = ModelInternalPublicBridge.Search;
  const originalBrowseManyMethod = ModelInternalPublicBridge.BrowseMany;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalSearch = ReadOperations.Search;
  const originalTriggerUpstreamCreateBatch = ComputeCascadeEngine.triggerUpstreamCreateBatch;

  const defaultCalls: any[] = [];
  const searchCalls: any[] = [];

  try {
    ModelInternalPublicBridge.DefaultGet = (async function (this: any, value: any) {
      defaultCalls.push({ ModelCtor: this, value });
      return { ...value };
    }) as any;
    ModelInternalPublicBridge.Search = unexpectedPublicCall('Search');
    ModelInternalPublicBridge.BrowseMany = unexpectedPublicCall('BrowseMany');

    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value },
      relations: emptyToManyRelations(),
    })) as any;

    RepositoryFactory.getRepository = (() => ({
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
      create: async (rows: any[]) => rows.map(row => row.Id),
    })) as any;

    ReadOperations.Search = (async (ModelCtor: any, condition: any, options?: any) => {
      searchCalls.push({ ModelCtor, condition, options });
      const ids = Array.isArray(condition) ? condition[2] || [] : [];
      return (ids as string[]).map((id, index) => ({ Id: id, Name: `name-${index + 1}` }));
    }) as any;

    ComputeCascadeEngine.triggerUpstreamCreateBatch = (async () => {}) as any;

    const withFields = await CreateOperations.CreateMany(
      ModelInternalPublicBridge as any,
      [
        { Id: 'NEW-1', Name: 'a' },
        { Id: 'NEW-2', Name: 'b' },
      ] as any,
      ['Id', 'Name'] as any
    );

    const withoutFields = await CreateOperations.CreateMany(
      ModelInternalPublicBridge as any,
      [
        { Id: 'NEW-3', Name: 'c' },
        { Id: 'NEW-4', Name: 'd' },
      ] as any
    );

    expect(defaultCalls.length).toBe(4);
    expect(searchCalls.length).toBe(2);
    expect(searchCalls[0]?.condition).toEqual(['Id', 'in', ['NEW-1', 'NEW-2']]);
    expect(searchCalls[0]?.options).toEqual({ fields: ['Id', 'Name'] });
    expect(searchCalls[1]?.condition).toEqual(['Id', 'in', ['NEW-3', 'NEW-4']]);
    expect(searchCalls[1]?.options).toEqual({});

    expect(withFields.length).toBe(2);
    expect((withFields[0] as any).Id).toBe('NEW-1');
    expect((withFields[0] as any).fields).toEqual(['Id', 'Name']);

    expect(withoutFields.length).toBe(2);
    expect((withoutFields[1] as any).Id).toBe('NEW-4');
    expect((withoutFields[0] as any).fields).toBe(undefined);
  } finally {
    ModelInternalPublicBridge.DefaultGet = originalDefaultGetMethod;
    ModelInternalPublicBridge.Search = originalSearchMethod;
    ModelInternalPublicBridge.BrowseMany = originalBrowseManyMethod;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RepositoryFactory.getRepository = originalGetRepository;
    ReadOperations.Search = originalSearch;
    ComputeCascadeEngine.triggerUpstreamCreateBatch = originalTriggerUpstreamCreateBatch;
  }
});

test('CreateOperations.Create honors subclass DefaultGet override without column defaults', async () => {
  @Model('test.DefaultGetOverrideCreate')
  class DefaultGetOverrideCreate extends BaseModel {
    @Field({ type: 'varchar', size: 64, default: 'from-column' })
    Name!: string;

    static async DefaultGet(this: any, value: any) {
      // Intentionally skip super.DefaultGet so @Field({ default }) does not run.
      return { ...value, Name: value?.Name ?? 'from-override' };
    }
  }

  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalBrowse = ReadOperations.Browse;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const createdRows: any[] = [];

  try {
    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value },
      relations: emptyToManyRelations(),
    })) as any;

    RepositoryFactory.getRepository = (() => ({
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
      create: async (rows: any[]) => {
        createdRows.push(...rows);
        return rows.map(row => row.Id);
      },
    })) as any;

    ReadOperations.Browse = (async (_ModelCtor: any, id: string) => ({ Id: id, Name: 'created' })) as any;
    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;

    await CreateOperations.Create(DefaultGetOverrideCreate as any, { Id: 'OV-1' } as any);

    expect(createdRows.length).toBe(1);
    expect(createdRows[0]?.Id).toBe('OV-1');
    expect(createdRows[0]?.Name).toBe('from-override');
  } finally {
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RepositoryFactory.getRepository = originalGetRepository;
    ReadOperations.Browse = originalBrowse;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
  }
});

test('UpdateOperations.Update uses internal read facade instead of public Search', async () => {
  const originalSearchMethod = ModelInternalPublicBridge.Search;
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalSearch = ReadOperations.Search;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;

  const repositorySearchCalls: any[] = [];
  const repositoryUpdateCalls: any[] = [];
  const searchCalls: any[] = [];

  try {
    ModelInternalPublicBridge.Search = unexpectedPublicCall('Search');

    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: emptyToManyRelations(),
    })) as any;

    RepositoryFactory.getRepository = (() => ({
      count: async () => 1,
      search: async (condition: any, options?: any) => {
        repositorySearchCalls.push({ condition, options });
        return [{ Id: 'ROW-1', UpdatedAt: new Date('2024-01-01T00:00:00.000Z'), Name: 'before' }];
      },
      update: async (values: any, condition: any) => {
        repositoryUpdateCalls.push({ values, condition });
      },
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    ReadOperations.Search = (async (ModelCtor: any, condition: any, options?: any) => {
      searchCalls.push({ ModelCtor, condition, options });
      return [{ Id: 'ROW-1', Name: 'after' }];
    }) as any;

    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    const result = await UpdateOperations.Update(
      ModelInternalPublicBridge as any,
      ['Id', '=', 'ROW-1'] as any,
      { Name: 'after' } as any,
      ['Id', 'Name'] as any
    );

    expect(result.length).toBe(1);
    expect((result[0] as any).Id).toBe('ROW-1');
    expect((result[0] as any).Name).toBe('after');
    expect((result[0] as any).fields).toEqual(['Id', 'Name']);

    expect(repositorySearchCalls.length).toBe(1);
    expect(repositoryUpdateCalls.length).toBe(1);
    expect(repositoryUpdateCalls[0]?.condition).toEqual(['Id', '=', 'ROW-1']);
    expect(repositoryUpdateCalls[0]?.values?.Name).toBe('after');

    expect(searchCalls.length).toBe(1);
    expect(searchCalls[0]?.condition).toEqual(['Id', 'in', ['ROW-1']]);
    expect(searchCalls[0]?.options).toEqual({ fields: ['Id', 'Name'] });
  } finally {
    ModelInternalPublicBridge.Search = originalSearchMethod;
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    RepositoryFactory.getRepository = originalGetRepository;
    ReadOperations.Search = originalSearch;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
  }
});

test('CreateOperations.Create aggregates relation errors and throws', async () => {
  const originalDefaultGet = DefaultOperations.DefaultGet;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;

  try {
    DefaultOperations.DefaultGet = (async (_ModelCtor: any, value: any) => ({ ...value })) as any;
    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value },
      relations: {
        oneToManyRelations: [{}],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(),
      },
    })) as any;

    RelationFactory.batchProcessToManyRelations = (async () => [
      {
        errors: [new Error('rel-1'), { message: 'rel-2' }],
      },
    ]) as any;

    RepositoryFactory.getRepository = (() => ({
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
      create: async () => ['RID-1'],
    })) as any;

    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;

    let err = '';
    try {
      await CreateOperations.Create(ModelInternalPublicBridge as any, { Id: 'RID-1' } as any);
    } catch (e) {
      err = String((e as Error).message || e);
    }

    expect(err.includes('relation handling failed for 2 item(s)')).toBe(true);
    expect(err.includes('rel-1')).toBe(true);
  } finally {
    DefaultOperations.DefaultGet = originalDefaultGet;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    RepositoryFactory.getRepository = originalGetRepository;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
  }
});

test('CreateOperations.CreateMany returns empty array when input values is empty', async () => {
  const result = await CreateOperations.CreateMany(ModelInternalPublicBridge as any, [] as any);
  expect(result).toEqual([]);
});

test('UpdateOperations.Update rejects multi-row collection updates', async () => {
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalGetRepository = RepositoryFactory.getRepository;

  try {
    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: {
        oneToManyRelations: [{}],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(['Lines']),
      },
    })) as any;

    RepositoryFactory.getRepository = (() => ({
      count: async () => 2,
      search: async () => [],
      update: async () => {},
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    let err = '';
    try {
      await UpdateOperations.Update(ModelInternalPublicBridge as any, ['Id', '!=', ''] as any, { Name: 'x' } as any);
    } catch (e) {
      err = String((e as Error).message || e);
    }

    expect(err.includes('collection field updates (OneToMany / ManyToMany) are not supported for multi-row updates')).toBe(true);
  } finally {
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('UpdateOperations.UpdateById throws when target is not found', async () => {
  const originalUpdate = UpdateOperations.Update;
  try {
    UpdateOperations.Update = (async () => []) as any;

    let err = '';
    try {
      await UpdateOperations.UpdateById(ModelInternalPublicBridge as any, 'MISS-ID', { Name: 'x' } as any);
    } catch (e) {
      err = String((e as Error).message || e);
    }

    expect(err.includes('Record MISS-ID not found')).toBe(true);
  } finally {
    UpdateOperations.Update = originalUpdate;
  }
});

test('UpdateOperations.Update forwards withDeleted option when returning fields', async () => {
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalSearch = ReadOperations.Search;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;

  const searchCalls: any[] = [];

  try {
    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: emptyToManyRelations(),
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      count: async () => 1,
      search: async () => [{ Id: 'ROW-2', UpdatedAt: new Date('2024-01-01T00:00:00.000Z') }],
      update: async () => {},
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    ReadOperations.Search = (async (_ModelCtor: any, _condition: any, options?: any) => {
      searchCalls.push(options);
      return [{ Id: 'ROW-2', Name: 'next' }];
    }) as any;

    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    await UpdateOperations.Update(
      ModelInternalPublicBridge as any,
      ['Id', '=', 'ROW-2'] as any,
      { Name: 'next' } as any,
      ['Id', 'Name'] as any,
      { withDeleted: true } as any
    );

    expect(searchCalls.length).toBe(1);
    expect(searchCalls[0]).toEqual({ fields: ['Id', 'Name'], withDeleted: true });
  } finally {
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    ReadOperations.Search = originalSearch;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
  }
});

test('UpdateOperations.Update forwards onlyDeleted option when returning fields', async () => {
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalSearch = ReadOperations.Search;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;

  const searchCalls: any[] = [];

  try {
    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: emptyToManyRelations(),
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      count: async () => 1,
      search: async () => [{ Id: 'ROW-3', UpdatedAt: new Date('2024-01-01T00:00:00.000Z') }],
      update: async () => {},
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    ReadOperations.Search = (async (_ModelCtor: any, _condition: any, options?: any) => {
      searchCalls.push(options);
      return [{ Id: 'ROW-3', Name: 'next' }];
    }) as any;

    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    await UpdateOperations.Update(
      ModelInternalPublicBridge as any,
      ['Id', '=', 'ROW-3'] as any,
      { Name: 'next' } as any,
      ['Id', 'Name'] as any,
      { onlyDeleted: true } as any
    );

    expect(searchCalls.length).toBe(1);
    expect(searchCalls[0]).toEqual({ fields: ['Id', 'Name'], onlyDeleted: true });
  } finally {
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    ReadOperations.Search = originalSearch;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
  }
});

test('CreateOperations.CreateMany aggregates relation errors and throws', async () => {
  const originalDefaultGet = DefaultOperations.DefaultGet;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalTriggerUpstreamCreateBatch = ComputeCascadeEngine.triggerUpstreamCreateBatch;

  try {
    DefaultOperations.DefaultGet = (async (_ModelCtor: any, value: any) => ({ ...value })) as any;
    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value },
      relations: {
        oneToManyRelations: [{}],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(),
      },
    })) as any;

    RelationFactory.batchProcessToManyRelations = (async () => [
      {
        errors: [{ message: 'rel-many-1' }, { error: new Error('rel-many-2') }],
      },
    ]) as any;

    RepositoryFactory.getRepository = (() => ({
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
      create: async (rows: any[]) => rows.map((row: any) => row.Id),
    })) as any;

    ComputeCascadeEngine.triggerUpstreamCreateBatch = (async () => {}) as any;

    let err = '';
    try {
      await CreateOperations.CreateMany(
        ModelInternalPublicBridge as any,
        [
          { Id: 'A-1', Name: 'a' },
          { Id: 'A-2', Name: 'b' },
        ] as any
      );
    } catch (e) {
      err = String((e as Error).message || e);
    }

    expect(err.includes('[CreateMany] relation handling failed for 2 item(s)')).toBe(true);
    expect(err.includes('rel-many-1')).toBe(true);
  } finally {
    DefaultOperations.DefaultGet = originalDefaultGet;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    RepositoryFactory.getRepository = originalGetRepository;
    ComputeCascadeEngine.triggerUpstreamCreateBatch = originalTriggerUpstreamCreateBatch;
  }
});

test('CreateOperations.Create swallows upstream error and still returns created model', async () => {
  const originalDefaultGetMethod = ModelInternalPublicBridge.DefaultGet;
  const originalBrowseMethod = ModelInternalPublicBridge.Browse;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalBrowse = ReadOperations.Browse;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalWarn = console.warn;
  const warnings: string[] = [];

  try {
    ModelInternalPublicBridge.DefaultGet = (async (_value: any) => ({ ..._value })) as any;
    ModelInternalPublicBridge.Browse = unexpectedPublicCall('Browse');

    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value },
      relations: emptyToManyRelations(),
    })) as any;
    RepositoryFactory.getRepository = (() => ({
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
      create: async () => ['UP-1'],
    })) as any;
    ReadOperations.Browse = (async () => ({ Id: 'UP-1', Name: 'ok' })) as any;
    ComputeCascadeEngine.triggerUpstream = (async () => {
      throw new Error('upstream-failed');
    }) as any;

    console.warn = (...args: any[]) => {
      warnings.push(args.map(x => String(x)).join(' '));
    };

    const result = await CreateOperations.Create(ModelInternalPublicBridge as any, { Id: 'UP-1' } as any, ['Id', 'Name'] as any);

    expect((result as any).Id).toBe('UP-1');
    expect(warnings.some(msg => msg.includes('upstream recompute failed and was ignored'))).toBe(true);
  } finally {
    ModelInternalPublicBridge.DefaultGet = originalDefaultGetMethod;
    ModelInternalPublicBridge.Browse = originalBrowseMethod;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RepositoryFactory.getRepository = originalGetRepository;
    ReadOperations.Browse = originalBrowse;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    console.warn = originalWarn;
  }
});

test('UpdateOperations.Update returns empty list when locked rows are empty', async () => {
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;

  try {
    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: emptyToManyRelations(),
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      search: async () => [],
      update: async () => {
        throw new Error('should not update when locked is empty');
      },
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    const result = await UpdateOperations.Update(ModelInternalPublicBridge as any, ['Id', '=', 'MISS'] as any, { Name: 'x' } as any);

    expect(result).toEqual([]);
  } finally {
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
  }
});

test('UpdateOperations.Update swallows downstream error and returns id list when no returnFields', async () => {
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;
  const originalWarn = console.warn;

  const warnings: string[] = [];
  const updates: any[] = [];

  try {
    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: emptyToManyRelations(),
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      search: async () => [{ Id: 'ROW-D', UpdatedAt: new Date('2024-01-01T00:00:00.000Z') }],
      update: async (values: any, condition: any) => {
        updates.push({ values, condition });
      },
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {
      throw new Error('downstream-failed');
    }) as any;

    console.warn = (...args: any[]) => {
      warnings.push(args.map(x => String(x)).join(' '));
    };

    const result = await UpdateOperations.Update(ModelInternalPublicBridge as any, ['Id', '=', 'ROW-D'] as any, { Name: 'after' } as any);

    expect(result).toEqual([{ Id: 'ROW-D' }]);
    expect(updates.length).toBe(1);
    expect(updates[0]?.condition).toEqual(['Id', '=', 'ROW-D']);
    expect(warnings.some(msg => msg.includes('downstream cascade recompute failed and was ignored'))).toBe(true);
  } finally {
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
    console.warn = originalWarn;
  }
});

test('UpdateOperations.Update throws when relation batch processing returns errors', async () => {
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;

  try {
    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: {
        oneToManyRelations: [{}],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(['Lines']),
      },
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      count: async () => 1,
      search: async () => [{ Id: 'ROW-R', UpdatedAt: new Date('2024-01-01T00:00:00.000Z') }],
      update: async () => {},
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    RelationFactory.batchProcessToManyRelations = (async () => [
      {
        errors: [{ error: new Error('rel-update-failed') }],
      },
    ]) as any;

    let err = '';
    try {
      await UpdateOperations.Update(ModelInternalPublicBridge as any, ['Id', '=', 'ROW-R'] as any, { Name: 'after' } as any);
    } catch (e) {
      err = String((e as Error).message || e);
    }

    expect(err.includes('[Update] relation handling failed')).toBe(true);
    expect(err.includes('rel-update-failed')).toBe(true);
  } finally {
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
  }
});

test('UpdateOperations.Update covers compute graph diffusion, collection prefetch forks, and follow-up decimal scale updates', async () => {
  class ChildModel {}

  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalRecompute = ComputeEngine.recompute;
  const originalCollectUpstreamInverseFields = ComputeCascadeEngine.collectUpstreamInverseFields;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;

  const meta = MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any) as any;
  const originalComputeGraph = meta.computeGraph;
  const snapshotFields = new Map(meta.fields);

  const childSearchCalls: any[] = [];
  const mainUpdateCalls: any[] = [];
  const bypassCalls: number[] = [];

  try {
    meta.fields.set('Amount', { type: 'decimal', column: { scaleField: 'AmountScale' } });
    meta.fields.set('AmountScale', { type: 'int' });
    meta.fields.set('Total', { type: 'decimal', column: { scaleField: 'TotalScale' } });
    meta.fields.set('TotalScale', { type: 'int' });
    meta.fields.set('Lines', { type: 'OneToMany', relation: { targetModel: () => ChildModel, inverseField: 'ParentId' } });
    meta.fields.set('MissingTarget', { type: 'OneToMany', relation: { inverseField: 'ParentId' } });
    meta.fields.set('MissingInverse', { type: 'OneToMany', relation: { targetModel: () => ChildModel } });

    meta.computeGraph = {
      computeFields: new Set<string>(['Total']),
      fastReverseDeps: new Map<string, string[]>([
        ['Amount', ['Total']],
        ['Lines', ['Total']],
        ['MissingTarget', ['Total']],
        ['MissingInverse', ['Total']],
        ['Total', ['GrandTotal']],
      ]),
      computeScalarDeps: new Map<string, string[][]>([['Total', ['Amount'] as any]]),
      computeCollectionPathDeps: new Map<string, Array<{ collection: string; chain: string[] }>>([
        [
          'Total',
          [
            { collection: 'Lines', chain: ['Product', 'Name'] },
            { collection: 'MissingTarget', chain: ['Name'] },
            { collection: 'MissingInverse', chain: [] },
          ],
        ],
      ]),
    } as any;

    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: {
        oneToManyRelations: [],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(['Lines', 'MissingTarget', 'MissingInverse']),
      },
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      count: async () => 1,
      search: async () => [
        {
          Id: 'ROW-G',
          UpdatedAt: new Date('2024-01-01T00:00:00.000Z'),
          Amount: '1.00',
          AmountScale: 2,
          Total: '1.00',
          TotalScale: 2,
        },
      ],
      update: async (values: any, condition: any) => {
        mainUpdateCalls.push({ values, condition });
      },
      withValidationBypass: async (fn: () => Promise<any>) => {
        bypassCalls.push(1);
        return await fn();
      },
    })) as any;

    RepositoryFactory.getRepository = ((ModelCtor: any) => {
      if (ModelCtor === ChildModel) {
        return {
          search: async (condition: any, options: any) => {
            childSearchCalls.push({ condition, options });
            return [{ Id: 'C-1', Product: { Name: 'P-1' } }];
          },
        } as any;
      }
      throw new Error('unexpected repository ctor');
    }) as any;

    ComputeEngine.recompute = (async (_meta: any, entity: any) => {
      entity.Total = '9.99';
      entity.TotalScale = 4;
    }) as any;

    ComputeCascadeEngine.collectUpstreamInverseFields = (() => ['ParentId']) as any;
    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    const result = await UpdateOperations.Update(
      ModelInternalPublicBridge as any,
      ['Id', '=', 'ROW-G'] as any,
      {
        Amount: '2.45',
        Total: '999.00',
      } as any
    );

    expect(result).toEqual([{ Id: 'ROW-G' }]);
    expect(childSearchCalls.length).toBe(1);
    expect(childSearchCalls[0]?.condition).toEqual(['ParentId', '=', 'ROW-G']);
    expect(childSearchCalls[0]?.options?.fields).toEqual(['Id', 'Product.Name']);

    expect(mainUpdateCalls.length).toBe(2);
    expect(mainUpdateCalls[0]?.values?.Amount).toBe('2.45');
    expect(mainUpdateCalls[0]?.values?.AmountScale).toBe(2);
    expect(mainUpdateCalls[1]?.values?.Total).toBe('9.99');
    expect(mainUpdateCalls[1]?.values?.TotalScale).toBe(4);
    expect(bypassCalls.length).toBe(1);
  } finally {
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    RepositoryFactory.getRepository = originalGetRepository;
    ComputeEngine.recompute = originalRecompute;
    ComputeCascadeEngine.collectUpstreamInverseFields = originalCollectUpstreamInverseFields;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;

    meta.computeGraph = originalComputeGraph;
    meta.fields = new Map(snapshotFields);
  }
});

test('UpdateOperations.Update includes monetary currencyField companions on lock and scalar write', async () => {
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;
  const originalCollectUpstreamInverseFields = ComputeCascadeEngine.collectUpstreamInverseFields;

  const meta = MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any) as any;
  const originalComputeGraph = meta.computeGraph;
  const snapshotFields = new Map(meta.fields);
  const searchCalls: any[] = [];
  const updateCalls: any[] = [];

  try {
    meta.fields.set('CurrencyId', { type: 'ManyToOneRef', column: { name: 'CurrencyId' } });
    meta.fields.set('Amount', { type: 'monetary', column: { currencyField: 'CurrencyId' } });
    meta.computeGraph = undefined;

    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: {
        oneToManyRelations: [],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(),
      },
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      count: async () => 1,
      search: async (_condition: any, options: any) => {
        searchCalls.push(options);
        return [
          {
            Id: 'ROW-M',
            UpdatedAt: new Date('2024-01-01T00:00:00.000Z'),
            Amount: '1.00',
            CurrencyId: { Id: 'C1', DecimalDigits: 2 },
          },
        ];
      },
      update: async (values: any) => {
        updateCalls.push(values);
      },
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    ComputeCascadeEngine.collectUpstreamInverseFields = (() => []) as any;
    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    const result = await UpdateOperations.Update(
      ModelInternalPublicBridge as any,
      ['Id', '=', 'ROW-M'] as any,
      { Amount: '2.50' } as any
    );

    expect(result).toEqual([{ Id: 'ROW-M' }]);
    expect(searchCalls[0]?.fields).toContain('Amount');
    expect(searchCalls[0]?.fields).toContain('CurrencyId');
    expect(updateCalls[0]?.Amount).toBe('2.50');
    expect(updateCalls[0]?.CurrencyId).toEqual({ Id: 'C1', DecimalDigits: 2 });
  } finally {
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
    ComputeCascadeEngine.collectUpstreamInverseFields = originalCollectUpstreamInverseFields;
    meta.computeGraph = originalComputeGraph;
    meta.fields = new Map(snapshotFields);
  }
});

test('UpdateOperations.Update skips monetary companions when currencyField is absent', async () => {
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;
  const originalCollectUpstreamInverseFields = ComputeCascadeEngine.collectUpstreamInverseFields;

  const meta = MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any) as any;
  const originalComputeGraph = meta.computeGraph;
  const snapshotFields = new Map(meta.fields);
  const searchCalls: any[] = [];

  try {
    meta.fields.set('Amount', { type: 'monetary', column: {} });
    meta.computeGraph = undefined;

    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: {
        oneToManyRelations: [],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(),
      },
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      count: async () => 1,
      search: async (_condition: any, options: any) => {
        searchCalls.push(options);
        return [
          {
            Id: 'ROW-M2',
            UpdatedAt: new Date('2024-01-01T00:00:00.000Z'),
            Amount: '1.00',
          },
        ];
      },
      update: async () => {},
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    ComputeCascadeEngine.collectUpstreamInverseFields = (() => []) as any;
    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    await UpdateOperations.Update(ModelInternalPublicBridge as any, ['Id', '=', 'ROW-M2'] as any, { Amount: '2.50' } as any);

    expect(searchCalls[0]?.fields).toContain('Amount');
    expect(searchCalls[0]?.fields || []).not.toContain('CurrencyId');
  } finally {
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
    ComputeCascadeEngine.collectUpstreamInverseFields = originalCollectUpstreamInverseFields;
    meta.computeGraph = originalComputeGraph;
    meta.fields = new Map(snapshotFields);
  }
});

test('UpdateOperations.Update handles monetary field with undefined column metadata', async () => {
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;
  const originalCollectUpstreamInverseFields = ComputeCascadeEngine.collectUpstreamInverseFields;

  const meta = MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any) as any;
  const originalComputeGraph = meta.computeGraph;
  const snapshotFields = new Map(meta.fields);
  const searchCalls: any[] = [];

  try {
    meta.fields.set('Amount', { type: 'monetary', column: undefined });
    meta.computeGraph = undefined;

    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: {
        oneToManyRelations: [],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(),
      },
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      count: async () => 1,
      search: async (_condition: any, options: any) => {
        searchCalls.push(options);
        return [
          {
            Id: 'ROW-M3',
            UpdatedAt: new Date('2024-01-01T00:00:00.000Z'),
            Amount: '1.00',
          },
        ];
      },
      update: async () => {},
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    ComputeCascadeEngine.collectUpstreamInverseFields = (() => []) as any;
    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    await UpdateOperations.Update(ModelInternalPublicBridge as any, ['Id', '=', 'ROW-M3'] as any, { Amount: '2.50' } as any);
    expect(searchCalls[0]?.fields).toContain('Amount');
  } finally {
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
    ComputeCascadeEngine.collectUpstreamInverseFields = originalCollectUpstreamInverseFields;
    meta.computeGraph = originalComputeGraph;
    meta.fields = new Map(snapshotFields);
  }
});

test('UpdateOperations.Update keeps monetary currency already present in the update payload', async () => {
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;
  const originalCollectUpstreamInverseFields = ComputeCascadeEngine.collectUpstreamInverseFields;

  const meta = MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any) as any;
  const originalComputeGraph = meta.computeGraph;
  const snapshotFields = new Map(meta.fields);
  const updateCalls: any[] = [];

  try {
    meta.fields.set('CurrencyId', { type: 'ManyToOneRef', column: { name: 'CurrencyId' } });
    meta.fields.set('Amount', { type: 'monetary', column: { currencyField: 'CurrencyId' } });
    meta.computeGraph = undefined;

    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: {
        oneToManyRelations: [],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(),
      },
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      count: async () => 1,
      search: async () => [
        {
          Id: 'ROW-M2',
          UpdatedAt: new Date('2024-01-01T00:00:00.000Z'),
          Amount: '1.00',
          CurrencyId: { Id: 'C-OLD', DecimalDigits: 2 },
        },
      ],
      update: async (values: any) => {
        updateCalls.push(values);
      },
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    ComputeCascadeEngine.collectUpstreamInverseFields = (() => []) as any;
    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    await UpdateOperations.Update(
      ModelInternalPublicBridge as any,
      ['Id', '=', 'ROW-M2'] as any,
      { Amount: '2.50', CurrencyId: { Id: 'C-NEW', DecimalDigits: 0 } } as any
    );

    expect(updateCalls[0]?.CurrencyId).toEqual({ Id: 'C-NEW', DecimalDigits: 0 });
  } finally {
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
    ComputeCascadeEngine.collectUpstreamInverseFields = originalCollectUpstreamInverseFields;
    meta.computeGraph = originalComputeGraph;
    meta.fields = new Map(snapshotFields);
  }
});

test('UpdateOperations.Update writes monetary compute follow-up with currency companions', async () => {
  class ChildModel {}

  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;
  const originalCollectUpstreamInverseFields = ComputeCascadeEngine.collectUpstreamInverseFields;
  const originalRecompute = ComputeEngine.recompute;

  const meta = MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any) as any;
  const originalComputeGraph = meta.computeGraph;
  const snapshotFields = new Map(meta.fields);
  const searchCalls: any[] = [];
  const updateCalls: any[] = [];

  try {
    meta.fields.set('CurrencyId', { type: 'ManyToOneRef', column: { name: 'CurrencyId' } });
    meta.fields.set('Amount', { type: 'monetary', column: { currencyField: 'CurrencyId' } });
    meta.fields.set('Total', { type: 'monetary', column: { currencyField: 'CurrencyId' } });
    meta.fields.set('Lines', { type: 'OneToMany', relation: { targetModel: () => ChildModel, inverseField: 'ParentId' } });
    meta.computeGraph = {
      computeFields: new Set<string>(['Total']),
      fastReverseDeps: new Map<string, string[]>([['Lines', ['Total']]]),
      computeScalarDeps: new Map<string, string[][]>([['Total', ['Amount'] as any]]),
      computeCollectionPathDeps: new Map<string, Array<{ collection: string; chain: string[] }>>([
        ['Total', [{ collection: 'Lines', chain: ['Name'] }]],
      ]),
    } as any;

    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: {
        oneToManyRelations: [],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(['Lines']),
      },
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      count: async () => 1,
      search: async (_condition: any, options: any) => {
        searchCalls.push(options);
        return [
          {
            Id: 'ROW-MC',
            UpdatedAt: new Date('2024-01-01T00:00:00.000Z'),
            Amount: '1.00',
            Total: '1.00',
            CurrencyId: { Id: 'C1', DecimalDigits: 2 },
          },
        ];
      },
      update: async (values: any) => {
        updateCalls.push(values);
      },
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    RepositoryFactory.getRepository = ((ModelCtor: any) => {
      if (ModelCtor === ChildModel) {
        return {
          search: async () => [{ Id: 'C-1', Name: 'line' }],
        } as any;
      }
      throw new Error('unexpected repository ctor');
    }) as any;

    ComputeEngine.recompute = (async (_meta: any, entity: any) => {
      entity.Total = '3.00';
    }) as any;
    ComputeCascadeEngine.collectUpstreamInverseFields = (() => []) as any;
    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    await UpdateOperations.Update(ModelInternalPublicBridge as any, ['Id', '=', 'ROW-MC'] as any, { Name: 'next' } as any);

    expect(searchCalls[0]?.fields).toContain('Amount');
    expect(searchCalls[0]?.fields).toContain('CurrencyId');
    expect(searchCalls[0]?.fields).toContain('Total');
    const followUp = updateCalls.find(v => v?.Total != null);
    expect(followUp?.Total).toBe('3.00');
    expect(followUp?.CurrencyId).toEqual({ Id: 'C1', DecimalDigits: 2 });
  } finally {
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    RepositoryFactory.getRepository = originalGetRepository;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
    ComputeCascadeEngine.collectUpstreamInverseFields = originalCollectUpstreamInverseFields;
    ComputeEngine.recompute = originalRecompute;
    meta.computeGraph = originalComputeGraph;
    meta.fields = new Map(snapshotFields);
  }
});

test('UpdateOperations.Update tolerates undefined relation batch result and supports String(e) fallback in relation errors', async () => {
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;

  try {
    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: {
        oneToManyRelations: [{}],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(['Lines']),
      },
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      count: async () => 1,
      search: async () => [{ Id: 'ROW-U', UpdatedAt: new Date('2024-01-01T00:00:00.000Z') }],
      update: async () => {},
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    RelationFactory.batchProcessToManyRelations = (async () => undefined as any) as any;
    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    const ok = await UpdateOperations.Update(ModelInternalPublicBridge as any, ['Id', '=', 'ROW-U'] as any, { Name: 'next' } as any);
    expect(ok).toEqual([{ Id: 'ROW-U' }]);

    RelationFactory.batchProcessToManyRelations = (async () => [{ errors: [0] }]) as any;

    let err = '';
    try {
      await UpdateOperations.Update(ModelInternalPublicBridge as any, ['Id', '=', 'ROW-U'] as any, { Name: 'next' } as any);
    } catch (e) {
      err = String((e as Error).message || e);
    }
    expect(err.includes('[Update] relation handling failed')).toBe(true);
    expect(err.includes('0')).toBe(true);
  } finally {
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
  }
});

test('CreateOperations.Create and CreateMany use String(e) fallback and tolerate undefined child rows in collection prefetch', async () => {
  class ChildModel {}

  const originalDefaultGet = DefaultOperations.DefaultGet;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalRecompute = ComputeEngine.recompute;
  const originalGetModelMetadata = MetadataStorage.instance.getModelMetadata;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerUpstreamCreateBatch = ComputeCascadeEngine.triggerUpstreamCreateBatch;

  try {
    const meta = MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any) as any;
    const fieldSnapshot = new Map(meta.fields);
    const originalGraph = meta.computeGraph;

    meta.fields.set('Lines', { type: 'OneToMany', relation: { targetModel: () => ChildModel, inverseField: 'ParentId' } });
    meta.fields.set('Total', { type: 'decimal', column: { scaleField: 'TotalScale' } });
    meta.fields.set('TotalScale', { type: 'int' });
    meta.computeGraph = {
      computeFields: new Set(['Total']),
      fastReverseDeps: new Map([['Lines', ['Total']]]),
      computeScalarDeps: new Map<string, string[][]>([['Total', ['Name'] as any]]),
      computeCollectionPathDeps: new Map([['Total', [{ collection: 'Lines', chain: ['Name'] }]]]),
    } as any;

    MetadataStorage.instance.getModelMetadata = function (ctor: any) {
      if (ctor === ChildModel) {
        return {
          type: ChildModel,
          fields: new Map([['Name', { type: 'varchar', column: { name: 'Name' } }]]),
        } as any;
      }
      return originalGetModelMetadata.call(MetadataStorage.instance, ctor);
    } as any;

    DefaultOperations.DefaultGet = (async (_ModelCtor: any, value: any) => ({ ...value })) as any;
    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value, Id: value.Id || 'C-ROW' },
      relations: {
        oneToManyRelations: [{}],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(['Lines']),
      },
    })) as any;

    const parentUpdates: any[] = [];
    RepositoryFactory.getRepository = ((ModelCtor: any) => {
      if (ModelCtor === ChildModel) {
        return {
          search: async () => undefined as any,
        } as any;
      }
      return {
        withValidationBypass: async (fn: () => Promise<any>) => await fn(),
        create: async (rows: any[]) => rows.map((r: any) => r.Id || 'C-ROW'),
        search: async () => [{ Id: 'C-ROW', Name: 'before', Total: '0', TotalScale: 2 }],
        update: async (vals: any) => {
          parentUpdates.push(vals);
        },
      } as any;
    }) as any;

    ComputeEngine.recompute = (async (_meta: any, entity: any) => {
      entity.Total = '3.14';
      entity.TotalScale = 2;
    }) as any;
    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerUpstreamCreateBatch = (async () => {}) as any;

    RelationFactory.batchProcessToManyRelations = (async () => [{ errors: [0] }]) as any;

    let createErr = '';
    try {
      await CreateOperations.Create(ModelInternalPublicBridge as any, { Id: 'C-ROW', Name: 'x' } as any);
    } catch (e) {
      createErr = String((e as Error).message || e);
    }
    expect(createErr.includes('[Create] relation handling failed')).toBe(true);
    expect(createErr.includes('0')).toBe(true);

    RelationFactory.batchProcessToManyRelations = (async () => [undefined]) as any;
    const created = await CreateOperations.Create(ModelInternalPublicBridge as any, { Id: 'C-ROW', Name: 'x' } as any);
    expect(created instanceof ModelInternalPublicBridge).toBe(true);
    expect(parentUpdates.length >= 1).toBe(true);

    RelationFactory.batchProcessToManyRelations = (async () => [{ errors: [0] }]) as any;
    let manyErr = '';
    try {
      await CreateOperations.CreateMany(ModelInternalPublicBridge as any, [{ Id: 'M-1', Name: 'x' }] as any);
    } catch (e) {
      manyErr = String((e as Error).message || e);
    }
    expect(manyErr.includes('[CreateMany] relation handling failed')).toBe(true);
    expect(manyErr.includes('0')).toBe(true);

    meta.computeGraph = originalGraph;
    meta.fields = fieldSnapshot;
  } finally {
    DefaultOperations.DefaultGet = originalDefaultGet;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    RepositoryFactory.getRepository = originalGetRepository;
    ComputeEngine.recompute = originalRecompute;
    MetadataStorage.instance.getModelMetadata = originalGetModelMetadata;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerUpstreamCreateBatch = originalTriggerUpstreamCreateBatch;
  }
});

test('CreateOperations.Create handles post-relations compute prefetch forks and updates parent through validation bypass', async () => {
  class ChildCreateModel {}

  const originalDefaultGet = DefaultOperations.DefaultGet;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalRecompute = ComputeEngine.recompute;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalBrowse = ReadOperations.Browse;
  const originalWarn = console.warn;

  const meta = MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any) as any;
  const originalComputeGraph = meta.computeGraph;
  const snapshotFields = new Map(meta.fields);

  const parentUpdates: any[] = [];
  const childSearchCalls: any[] = [];
  const warnings: string[] = [];
  let bypassCalls = 0;

  try {
    meta.fields.set('Amount', { type: 'decimal', column: { scaleField: 'AmountScale' } });
    meta.fields.set('AmountScale', { type: 'int' });
    meta.fields.set('Total', { type: 'decimal', column: { scaleField: 'TotalScale' } });
    meta.fields.set('TotalScale', { type: 'int' });
    meta.fields.set('Lines', { type: 'OneToMany', relation: { targetModel: () => ChildCreateModel, inverseField: 'ParentId' } });
    meta.fields.set('MissingTarget', { type: 'OneToMany', relation: { inverseField: 'ParentId' } });
    meta.fields.set('MissingInverse', { type: 'OneToMany', relation: { targetModel: () => ChildCreateModel } });

    meta.computeGraph = {
      computeFields: new Set<string>(['Total']),
      fastReverseDeps: new Map<string, string[]>([
        ['Lines', ['Total']],
        ['MissingTarget', ['Total']],
        ['MissingInverse', ['Total']],
      ]),
      computeScalarDeps: new Map<string, string[]>([['Total', ['Amount']]]),
      computeCollectionPathDeps: new Map<string, Array<{ collection: string; chain: string[] }>>([
        [
          'Total',
          [
            { collection: 'Lines', chain: ['Product', 'Name'] },
            { collection: 'MissingTarget', chain: ['Name'] },
            { collection: 'MissingInverse', chain: [] },
          ],
        ],
      ]),
    } as any;

    DefaultOperations.DefaultGet = (async (_ModelCtor: any, value: any) => ({
      ...value,
      Total: '999.00',
    })) as any;

    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value, Amount: '2.50' },
      relations: {
        oneToManyRelations: [{}],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(['Lines', 'MissingTarget', 'MissingInverse']),
      },
    })) as any;

    RelationFactory.batchProcessToManyRelations = (async () => [{ errors: [] }]) as any;

    RepositoryFactory.getRepository = ((ModelCtor: any) => {
      if (ModelCtor === ModelInternalPublicBridge) {
        return {
          withValidationBypass: async (fn: () => Promise<any>) => {
            bypassCalls += 1;
            return await fn();
          },
          create: async () => ['CRT-1'],
          search: async () => [{ Id: 'CRT-1', Amount: '2.50', AmountScale: 2, Total: '1.00', TotalScale: 2 }],
          update: async (values: any, condition: any) => {
            parentUpdates.push({ values, condition });
          },
        } as any;
      }
      if (ModelCtor === ChildCreateModel) {
        return {
          search: async (condition: any, options: any) => {
            childSearchCalls.push({ condition, options });
            return [{ Id: 'CH-1', Product: { Name: 'P' } }];
          },
        } as any;
      }
      throw new Error('unexpected repository ctor in create test');
    }) as any;

    ComputeEngine.recompute = (async (_meta: any, entity: any) => {
      entity.Total = '8.88';
      entity.TotalScale = 3;
    }) as any;

    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ReadOperations.Browse = (async () => ({ Id: 'CRT-1', Name: 'created' })) as any;

    console.warn = (...args: any[]) => {
      warnings.push(args.map(x => String(x)).join(' '));
    };

    const result = await CreateOperations.Create(ModelInternalPublicBridge as any, { Name: 'n1' } as any, ['Id', 'Name'] as any);

    expect((result as any).Id).toBe('CRT-1');
    expect(childSearchCalls.length).toBe(1);
    expect(childSearchCalls[0]?.condition).toEqual(['ParentId', '=', 'CRT-1']);
    expect(childSearchCalls[0]?.options?.fields).toEqual(['Id', 'Product.Name']);

    expect(parentUpdates.length).toBe(1);
    expect(parentUpdates[0]?.condition).toEqual(['Id', '=', 'CRT-1']);
    expect(parentUpdates[0]?.values?.Total).toBe('8.88');
    expect(parentUpdates[0]?.values?.TotalScale).toBe(undefined);
    // UpdatedAt / audit uids are stamped in repository update prepare, not at the model layer.
    expect(parentUpdates[0]?.values?.UpdatedAt).toBeUndefined();
    expect(bypassCalls).toBe(2);
    expect(warnings.some(msg => msg.includes('MissingTarget'))).toBe(true);
    expect(warnings.some(msg => msg.includes('MissingInverse'))).toBe(true);
  } finally {
    DefaultOperations.DefaultGet = originalDefaultGet;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    RepositoryFactory.getRepository = originalGetRepository;
    ComputeEngine.recompute = originalRecompute;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ReadOperations.Browse = originalBrowse;
    console.warn = originalWarn;

    meta.computeGraph = originalComputeGraph;
    meta.fields = new Map(snapshotFields);
  }
});

test('CreateOperations.CreateMany swallows upstream batch errors and still returns searched models for returnFields', async () => {
  const originalDefaultGet = DefaultOperations.DefaultGet;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalTriggerUpstreamCreateBatch = ComputeCascadeEngine.triggerUpstreamCreateBatch;
  const originalSearch = ReadOperations.Search;
  const originalWarn = console.warn;

  const searchCalls: any[] = [];
  const warnings: string[] = [];

  try {
    DefaultOperations.DefaultGet = (async (_ModelCtor: any, value: any) => ({ ...value })) as any;
    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value },
      relations: emptyToManyRelations(),
    })) as any;
    RelationFactory.batchProcessToManyRelations = (async () => [{ errors: [] }]) as any;

    RepositoryFactory.getRepository = (() => ({
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
      create: async (_rows: any[]) => ['CM-1', 'CM-2'],
    })) as any;

    ComputeCascadeEngine.triggerUpstreamCreateBatch = (async () => {
      throw new Error('batch-upstream-failed');
    }) as any;

    ReadOperations.Search = (async (_ModelCtor: any, condition: any, options?: any) => {
      searchCalls.push({ condition, options });
      return [
        { Id: 'CM-1', Name: 'a' },
        { Id: 'CM-2', Name: 'b' },
      ];
    }) as any;

    console.warn = (...args: any[]) => {
      warnings.push(args.map(x => String(x)).join(' '));
    };

    const result = await CreateOperations.CreateMany(ModelInternalPublicBridge as any, [{ Name: 'a' }, { Name: 'b' }] as any, ['Id', 'Name'] as any);

    expect(result.length).toBe(2);
    expect((result[0] as any).Id).toBe('CM-1');
    expect(searchCalls.length).toBe(1);
    expect(searchCalls[0]?.condition).toEqual(['Id', 'in', ['CM-1', 'CM-2']]);
    expect(searchCalls[0]?.options).toEqual({ fields: ['Id', 'Name'] });
    expect(warnings.some(msg => msg.includes('CreateMany') && msg.includes('upstream recompute failed'))).toBe(true);
  } finally {
    DefaultOperations.DefaultGet = originalDefaultGet;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    RepositoryFactory.getRepository = originalGetRepository;
    ComputeCascadeEngine.triggerUpstreamCreateBatch = originalTriggerUpstreamCreateBatch;
    ReadOperations.Search = originalSearch;
    console.warn = originalWarn;
  }
});

test('CreateOperations.CreateMany tolerates relation results with non-array errors and still browses models', async () => {
  const originalDefaultGet = DefaultOperations.DefaultGet;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalBrowseMany = ReadOperations.Search;

  try {
    DefaultOperations.DefaultGet = (async (_ModelCtor: any, value: any) => ({ ...value })) as any;
    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value },
      relations: {
        oneToManyRelations: [{}],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(),
      },
    })) as any;

    RelationFactory.batchProcessToManyRelations = (async () => [{ errors: null }]) as any;

    RepositoryFactory.getRepository = (() => ({
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
      create: async () => ['CMX-1'],
    })) as any;

    ReadOperations.Search = (async () => [{ Id: 'CMX-1', Name: 'ok' }]) as any;

    const result = await CreateOperations.CreateMany(ModelInternalPublicBridge as any, [{ Name: 'ok' }] as any);
    expect(result.length).toBe(1);
    expect((result[0] as any).Id).toBe('CMX-1');
  } finally {
    DefaultOperations.DefaultGet = originalDefaultGet;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    RepositoryFactory.getRepository = originalGetRepository;
    ReadOperations.Search = originalBrowseMany;
  }
});

test('UpdateOperations.Update swallows upstream error and keeps returning ids', async () => {
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;
  const originalWarn = console.warn;

  const warnings: string[] = [];

  try {
    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: {
        oneToManyRelations: [],
        manyToManyRelations: [],
      } as any,
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      search: async () => [{ Id: 'UPS-1', UpdatedAt: new Date('2024-01-01T00:00:00.000Z') }],
      update: async () => {},
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    ComputeCascadeEngine.triggerUpstream = (async () => {
      throw new Error('upstream-failed-update');
    }) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    console.warn = (...args: any[]) => {
      warnings.push(args.map(x => String(x)).join(' '));
    };

    const result = await UpdateOperations.Update(ModelInternalPublicBridge as any, ['Id', '=', 'UPS-1'] as any, { Name: 'N' } as any);
    expect(result).toEqual([{ Id: 'UPS-1' }]);
    expect(warnings.some(msg => msg.includes('upstream recompute failed and was ignored'))).toBe(true);
  } finally {
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
    console.warn = originalWarn;
  }
});

test('UpdateOperations.Update parses sparse relation results and extracts nested error message', async () => {
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;

  try {
    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: {
        oneToManyRelations: [{}],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(['Lines']),
      },
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      count: async () => 1,
      search: async () => [{ Id: 'ROW-S', UpdatedAt: new Date('2024-01-01T00:00:00.000Z') }],
      update: async () => {},
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    RelationFactory.batchProcessToManyRelations = (async () => [
      undefined,
      { errors: 'bad-shape' },
      { errors: [{ error: { message: 'nested-rel-error' } }] },
    ]) as any;

    let err = '';
    try {
      await UpdateOperations.Update(ModelInternalPublicBridge as any, ['Id', '=', 'ROW-S'] as any, { Name: 'x' } as any);
    } catch (e) {
      err = String((e as Error).message || e);
    }

    expect(err.includes('nested-rel-error')).toBe(true);
  } finally {
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
  }
});

test('CreateOperations.CreateMany builds upstream rows even when repository returns extra ids', async () => {
  const originalDefaultGet = DefaultOperations.DefaultGet;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalTriggerUpstreamCreateBatch = ComputeCascadeEngine.triggerUpstreamCreateBatch;
  const originalSearch = ReadOperations.Search;

  const upstreamRows: any[] = [];

  try {
    DefaultOperations.DefaultGet = (async (_ModelCtor: any, value: any) => ({ ...value })) as any;
    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value },
      relations: emptyToManyRelations(),
    })) as any;
    RelationFactory.batchProcessToManyRelations = (async () => [{ errors: [] }]) as any;

    RepositoryFactory.getRepository = (() => ({
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
      create: async () => ['EX-1', 'EX-2', 'EX-3'],
    })) as any;

    ComputeCascadeEngine.triggerUpstreamCreateBatch = (async (_ModelCtor: any, rows: any[]) => {
      upstreamRows.push(...rows);
    }) as any;

    ReadOperations.Search = (async (_ModelCtor: any, condition: any) => {
      const ids = Array.isArray(condition) ? condition[2] || [] : [];
      return (ids as string[]).map(id => ({ Id: id }));
    }) as any;

    const result = await CreateOperations.CreateMany(ModelInternalPublicBridge as any, [{ Name: 'a' }, { Name: 'b' }] as any, ['Id'] as any);

    expect(result.length).toBe(3);
    expect(upstreamRows.length).toBe(3);
    expect(upstreamRows[2]).toEqual({ Id: 'EX-3' });
  } finally {
    DefaultOperations.DefaultGet = originalDefaultGet;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    RepositoryFactory.getRepository = originalGetRepository;
    ComputeCascadeEngine.triggerUpstreamCreateBatch = originalTriggerUpstreamCreateBatch;
    ReadOperations.Search = originalSearch;
  }
});

test('CreateOperations.Create handles sparse relation results and extracts nested error message', async () => {
  const originalDefaultGet = DefaultOperations.DefaultGet;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;

  try {
    DefaultOperations.DefaultGet = (async (_ModelCtor: any, value: any) => ({ ...value })) as any;
    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value },
      relations: {
        oneToManyRelations: [{}],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(),
      },
    })) as any;

    RelationFactory.batchProcessToManyRelations = (async () => [
      undefined,
      { errors: 'bad-shape' },
      { errors: [{ error: { message: 'create-rel-error' } }] },
    ]) as any;

    RepositoryFactory.getRepository = (() => ({
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
      create: async () => ['RID-X'],
    })) as any;

    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;

    let err = '';
    try {
      await CreateOperations.Create(ModelInternalPublicBridge as any, { Id: 'RID-X' } as any);
    } catch (e) {
      err = String((e as Error).message || e);
    }

    expect(err.includes('create-rel-error')).toBe(true);
  } finally {
    DefaultOperations.DefaultGet = originalDefaultGet;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    RepositoryFactory.getRepository = originalGetRepository;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
  }
});

test('UpdateOperations.Update skips compute block when computeGraph is absent and ignores undefined inverse fields', async () => {
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalCollectUpstreamInverseFields = ComputeCascadeEngine.collectUpstreamInverseFields;

  const meta = MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any) as any;
  const originalComputeGraph = meta.computeGraph;

  const searchCalls: any[] = [];
  const updateCalls: any[] = [];

  try {
    meta.computeGraph = undefined;

    ComputeCascadeEngine.collectUpstreamInverseFields = (() => [undefined]) as any;

    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: emptyToManyRelations(),
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      search: async (_condition: any, options?: any) => {
        searchCalls.push(options);
        return [{ Id: 'ROW-NC', UpdatedAt: new Date('2024-01-01T00:00:00.000Z') }];
      },
      update: async (values: any, condition: any) => {
        updateCalls.push({ values, condition });
      },
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    const result = await UpdateOperations.Update(ModelInternalPublicBridge as any, ['Id', '=', 'ROW-NC'] as any, { Name: 'n' } as any);

    expect(result).toEqual([{ Id: 'ROW-NC' }]);
    expect(searchCalls.length).toBe(1);
    expect(searchCalls[0]?.fields).toEqual(['Id', 'UpdatedAt', 'Name', undefined]);
    expect(updateCalls.length).toBe(1);
    expect(updateCalls[0]?.condition).toEqual(['Id', '=', 'ROW-NC']);
  } finally {
    meta.computeGraph = originalComputeGraph;
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    ComputeCascadeEngine.collectUpstreamInverseFields = originalCollectUpstreamInverseFields;
  }
});

test('UpdateOperations.Update skips relation batch when touchedCollections exists but relation arrays are empty', async () => {
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;

  const updates: any[] = [];
  let batchCalled = 0;

  try {
    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: {
        oneToManyRelations: [],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(['Lines']),
      },
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      count: async () => 1,
      search: async () => [{ Id: 'ROW-SKIP', UpdatedAt: new Date('2024-01-01T00:00:00.000Z') }],
      update: async (values: any, condition: any) => {
        updates.push({ values, condition });
      },
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    RelationFactory.batchProcessToManyRelations = (async () => {
      batchCalled += 1;
      return [];
    }) as any;

    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    const result = await UpdateOperations.Update(ModelInternalPublicBridge as any, ['Id', '=', 'ROW-SKIP'] as any, { Name: 'after' } as any);

    expect(result.length).toBe(1);
    expect((result[0] as any).Id).toBe('ROW-SKIP');
    expect(batchCalled).toBe(0);
    expect(updates.length).toBe(1);
  } finally {
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
  }
});

test('CreateOperations.Create tolerates undefined relation batch result array', async () => {
  const originalDefaultGet = DefaultOperations.DefaultGet;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalBrowse = ReadOperations.Browse;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;

  try {
    DefaultOperations.DefaultGet = (async (_ModelCtor: any, value: any) => ({ ...value })) as any;
    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value, Id: 'CRT-U' },
      relations: {
        oneToManyRelations: [{}],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(),
      },
    })) as any;

    RelationFactory.batchProcessToManyRelations = (async () => undefined) as any;

    RepositoryFactory.getRepository = (() => ({
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
      create: async () => ['CRT-U'],
    })) as any;

    ReadOperations.Browse = (async () => ({ Id: 'CRT-U', Name: 'ok' })) as any;
    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;

    const result = await CreateOperations.Create(ModelInternalPublicBridge as any, { Name: 'ok' } as any, ['Id', 'Name'] as any);
    expect((result as any).Id).toBe('CRT-U');
  } finally {
    DefaultOperations.DefaultGet = originalDefaultGet;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    RepositoryFactory.getRepository = originalGetRepository;
    ReadOperations.Browse = originalBrowse;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
  }
});

test('CreateOperations.CreateMany tolerates undefined relation batch result array', async () => {
  const originalDefaultGet = DefaultOperations.DefaultGet;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalSearch = ReadOperations.Search;
  const originalTriggerUpstreamCreateBatch = ComputeCascadeEngine.triggerUpstreamCreateBatch;

  try {
    DefaultOperations.DefaultGet = (async (_ModelCtor: any, value: any) => ({ ...value })) as any;
    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value },
      relations: {
        oneToManyRelations: [{}],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(),
      },
    })) as any;

    RelationFactory.batchProcessToManyRelations = (async () => undefined) as any;

    RepositoryFactory.getRepository = (() => ({
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
      create: async () => ['CM-U1', 'CM-U2'],
    })) as any;

    ReadOperations.Search = (async (_ModelCtor: any, condition: any, options?: any) => {
      const ids = Array.isArray(condition) ? condition[2] || [] : [];
      return (ids as string[]).map(id => ({ Id: id, Name: 'ok', fields: options?.fields }));
    }) as any;

    ComputeCascadeEngine.triggerUpstreamCreateBatch = (async () => {}) as any;

    const result = await CreateOperations.CreateMany(ModelInternalPublicBridge as any, [{ Name: 'a' }, { Name: 'b' }] as any, ['Id', 'Name'] as any);

    expect(result.length).toBe(2);
    expect((result[0] as any).Id).toBe('CM-U1');
    expect((result[1] as any).Id).toBe('CM-U2');
  } finally {
    DefaultOperations.DefaultGet = originalDefaultGet;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    RepositoryFactory.getRepository = originalGetRepository;
    ReadOperations.Search = originalSearch;
    ComputeCascadeEngine.triggerUpstreamCreateBatch = originalTriggerUpstreamCreateBatch;
  }
});

test('CreateOperations.Create auto-generates Id when processed value has no Id', async () => {
  const originalDefaultGet = DefaultOperations.DefaultGet;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalBrowse = ReadOperations.Browse;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;

  const oldXid = ($choysum as any).xid;
  const createdRows: any[] = [];

  try {
    ($choysum as any).xid = {
      New: () => 'AUTO-ID-1',
    };

    DefaultOperations.DefaultGet = (async (_ModelCtor: any, value: any) => ({ ...value })) as any;
    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value },
      relations: emptyToManyRelations(),
    })) as any;

    RepositoryFactory.getRepository = (() => ({
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
      create: async (rows: any[]) => {
        createdRows.push(...rows);
        return rows.map(row => row.Id);
      },
    })) as any;

    ReadOperations.Browse = (async () => ({ Id: 'AUTO-ID-1', Name: 'auto' })) as any;
    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;

    const result = await CreateOperations.Create(ModelInternalPublicBridge as any, { Name: 'n' } as any, ['Id', 'Name'] as any);

    expect((result as any).Id).toBe('AUTO-ID-1');
    expect(createdRows.length).toBe(1);
    expect(createdRows[0]?.Id).toBe('AUTO-ID-1');
  } finally {
    ($choysum as any).xid = oldXid;
    DefaultOperations.DefaultGet = originalDefaultGet;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RepositoryFactory.getRepository = originalGetRepository;
    ReadOperations.Browse = originalBrowse;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
  }
});

test('CreateOperations.CreateMany skips relation processing when repository returns empty ids', async () => {
  const originalDefaultGet = DefaultOperations.DefaultGet;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalTriggerUpstreamCreateBatch = ComputeCascadeEngine.triggerUpstreamCreateBatch;
  const originalSearch = ReadOperations.Search;

  let batchCalled = 0;
  let searchCalled = 0;

  try {
    DefaultOperations.DefaultGet = (async (_ModelCtor: any, value: any) => ({ ...value })) as any;
    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value },
      relations: {
        oneToManyRelations: [{}],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(),
      },
    })) as any;

    RelationFactory.batchProcessToManyRelations = (async () => {
      batchCalled += 1;
      return [];
    }) as any;

    RepositoryFactory.getRepository = (() => ({
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
      create: async () => [],
    })) as any;

    ComputeCascadeEngine.triggerUpstreamCreateBatch = (async () => {}) as any;
    ReadOperations.Search = (async () => {
      searchCalled += 1;
      return [];
    }) as any;

    const result = await CreateOperations.CreateMany(ModelInternalPublicBridge as any, [{ Name: 'x' }] as any, ['Id'] as any);

    expect(result).toEqual([]);
    expect(batchCalled).toBe(0);
    expect(searchCalled).toBe(1);
  } finally {
    DefaultOperations.DefaultGet = originalDefaultGet;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    RepositoryFactory.getRepository = originalGetRepository;
    ComputeCascadeEngine.triggerUpstreamCreateBatch = originalTriggerUpstreamCreateBatch;
    ReadOperations.Search = originalSearch;
  }
});

test('UpdateOperations.Update skips direct update when processed scalar payload is empty', async () => {
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;

  const updateCalls: any[] = [];

  try {
    RelationFactory.prepareForUpdate = (async () => ({
      processedValue: {},
      relations: emptyToManyRelations(),
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      search: async () => [{ Id: 'ROW-EMPTY', UpdatedAt: new Date('2024-01-01T00:00:00.000Z') }],
      update: async (vals: any, cond: any) => {
        updateCalls.push({ vals, cond });
      },
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    const result = await UpdateOperations.Update(ModelInternalPublicBridge as any, ['Id', '=', 'ROW-EMPTY'] as any, {} as any);

    expect(result).toEqual([{ Id: 'ROW-EMPTY' }]);
    expect(updateCalls.length).toBe(0);
  } finally {
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
  }
});

test('UpdateOperations.Update includes decimal scale field from column spec', async () => {
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;

  const meta = MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any) as any;
  const snapshotFields = new Map(meta.fields);
  const searchCalls: any[] = [];

  try {
    meta.fields.set('SelectDecimal', { type: 'decimal', column: { scaleField: 'SelectDecimalScale' } });
    meta.fields.set('SelectDecimalScale', { type: 'int' });

    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: emptyToManyRelations(),
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      search: async (_condition: any, options?: any) => {
        searchCalls.push(options);
        return [
          {
            Id: 'ROW-SD',
            UpdatedAt: new Date('2024-01-01T00:00:00.000Z'),
            SelectDecimalScale: 3,
          },
        ];
      },
      update: async () => {},
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    const result = await UpdateOperations.Update(ModelInternalPublicBridge as any, ['Id', '=', 'ROW-SD'] as any, { SelectDecimal: '1.23' } as any);

    expect(result).toEqual([{ Id: 'ROW-SD' }]);
    expect(searchCalls.length).toBe(1);
    expect(searchCalls[0]?.fields).toContain('SelectDecimal');
    expect(searchCalls[0]?.fields).toContain('SelectDecimalScale');
  } finally {
    meta.fields = new Map(snapshotFields);
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
  }
});

test('UpdateOperations.Update handles empty changed key and decimal without scale spec in helper fallbacks', async () => {
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;

  const meta = MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any) as any;
  const snapshotFields = new Map(meta.fields);
  const searchCalls: any[] = [];
  const updateCalls: any[] = [];

  try {
    meta.fields.set('LooseDecimal', { type: 'decimal' });

    RelationFactory.prepareForUpdate = (async () => ({
      processedValue: {
        '': 'ignored-empty-key',
        LooseDecimal: '1.25',
      },
      relations: emptyToManyRelations(),
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      search: async (_condition: any, options?: any) => {
        searchCalls.push(options);
        return [{ Id: 'ROW-LD', UpdatedAt: new Date('2024-01-01T00:00:00.000Z') }];
      },
      update: async (vals: any, cond: any) => {
        updateCalls.push({ vals, cond });
      },
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    const result = await UpdateOperations.Update(
      ModelInternalPublicBridge as any,
      ['Id', '=', 'ROW-LD'] as any,
      {
        LooseDecimal: '1.25',
        '': 'ignored-empty-key',
      } as any
    );

    expect(result).toEqual([{ Id: 'ROW-LD' }]);
    expect(searchCalls.length).toBe(1);
    expect(searchCalls[0]?.fields).toContain('LooseDecimal');
    expect(updateCalls.length).toBe(1);
    expect(updateCalls[0]?.vals?.LooseDecimal).toBe('1.25');
  } finally {
    meta.fields = new Map(snapshotFields);
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
  }
});

test('UpdateOperations.Update assigns empty collection rows when child prefetch returns undefined', async () => {
  class ChildPrefetchModel {}

  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalRecompute = ComputeEngine.recompute;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;

  const ctorMeta = (ModelInternalPublicBridge as any).metadata as any;
  const meta = (ctorMeta || MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any)) as any;
  if (!ctorMeta) {
    (ModelInternalPublicBridge as any).metadata = meta;
  }
  const snapshotFields = new Map(meta.fields);
  const originalComputeGraph = meta.computeGraph;
  const parentUpdates: any[] = [];
  const childSearchCalls: any[] = [];

  try {
    meta.fields.set('Lines', { type: 'OneToMany', relation: { targetModel: () => ChildPrefetchModel, inverseField: 'ParentId' } });
    meta.fields.set('Total', { type: 'decimal', column: { scaleField: 'TotalScale' } });
    meta.fields.set('TotalScale', { type: 'int' });

    meta.computeGraph = {
      computeFields: new Set<string>(['Total']),
      fastReverseDeps: new Map<string, string[]>([
        ['Lines', ['Total']],
        ['Name', ['Total']],
      ]),
      computeScalarDeps: new Map<string, string[][]>(),
      computeCollectionPathDeps: new Map<string, Array<{ collection: string; chain: string[] }>>([['Total', [{ collection: 'Lines', chain: ['Name'] }]]]),
    } as any;

    RelationFactory.prepareForUpdate = (async () => ({
      processedValue: { Name: 'n1' },
      relations: {
        oneToManyRelations: [],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(['Lines']),
      },
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      count: async () => 1,
      search: async () => [
        {
          Id: 'ROW-PF',
          UpdatedAt: new Date('2024-01-01T00:00:00.000Z'),
          Total: '0',
        },
      ],
      update: async (vals: any, cond: any) => {
        parentUpdates.push({ vals, cond });
      },
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    RepositoryFactory.getRepository = ((ctor: any) => {
      if (ctor === ChildPrefetchModel) {
        return {
          search: async (condition: any, options?: any) => {
            childSearchCalls.push({ condition, options });
            return undefined as any;
          },
        } as any;
      }
      throw new Error('unexpected child repo ctor');
    }) as any;

    ComputeEngine.recompute = (async (_m: any, entity: any) => {
      entity.Total = '5.00';
    }) as any;

    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    const result = await UpdateOperations.Update(ModelInternalPublicBridge as any, ['Id', '=', 'ROW-PF'] as any, { Name: 'n1' } as any);
    expect(result).toEqual([{ Id: 'ROW-PF' }]);
    if (childSearchCalls.length > 0) {
      expect(childSearchCalls[0]?.condition).toEqual(['ParentId', '=', 'ROW-PF']);
      expect(childSearchCalls[0]?.options?.fields).toEqual(['Id', 'Name']);
    }
    expect(parentUpdates.length).toBeGreaterThanOrEqual(1);
  } finally {
    meta.fields = new Map(snapshotFields);
    meta.computeGraph = originalComputeGraph;
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    RepositoryFactory.getRepository = originalGetRepository;
    ComputeEngine.recompute = originalRecompute;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
  }
});

test('UpdateOperations.Update keeps prefetched collection rows when child prefetch returns records', async () => {
  class ChildPrefetchRowsModel {}

  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalRecompute = ComputeEngine.recompute;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;

  const meta = MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any) as any;
  const snapshotFields = new Map(meta.fields);
  const originalComputeGraph = meta.computeGraph;
  const prefetchedRows = [{ Id: 'L-1', Name: 'line-1' }];

  try {
    meta.fields.set('Lines', { type: 'OneToMany', relation: { targetModel: () => ChildPrefetchRowsModel, inverseField: 'ParentId' } });
    meta.fields.set('Total', { type: 'decimal', column: { scaleField: 'TotalScale' } });
    meta.fields.set('TotalScale', { type: 'int' });

    meta.computeGraph = {
      computeFields: new Set<string>(['Total']),
      fastReverseDeps: new Map<string, string[]>([
        ['Lines', ['Total']],
        ['Name', ['Total']],
      ]),
      computeScalarDeps: new Map<string, string[][]>(),
      computeCollectionPathDeps: new Map<string, Array<{ collection: string; chain: string[] }>>([['Total', [{ collection: 'Lines', chain: ['Name'] }]]]),
    } as any;

    RelationFactory.prepareForUpdate = (async () => ({
      processedValue: { Name: 'after' },
      relations: {
        oneToManyRelations: [],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(['Lines']),
      },
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      count: async () => 1,
      search: async () => [
        {
          Id: 'ROW-PF-ROWS',
          UpdatedAt: new Date('2024-01-01T00:00:00.000Z'),
          Name: 'before',
          Total: '0',
        },
      ],
      update: async () => {},
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    RepositoryFactory.getRepository = (() => {
      return {
        search: async () => prefetchedRows,
      } as any;
    }) as any;

    ComputeEngine.recompute = (async (_m: any, entity: any) => {
      entity.Total = '7.00';
    }) as any;

    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    const result = await UpdateOperations.Update(ModelInternalPublicBridge as any, ['Id', '=', 'ROW-PF-ROWS'] as any, {} as any);
    expect(result).toEqual([{ Id: 'ROW-PF-ROWS' }]);
  } finally {
    meta.fields = new Map(snapshotFields);
    meta.computeGraph = originalComputeGraph;
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    RepositoryFactory.getRepository = originalGetRepository;
    ComputeEngine.recompute = originalRecompute;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
  }
});

test('CreateOperations.Create handles duplicate reverse deps and empty collection chain fallback', async () => {
  class CreateChildModel {}

  const originalDefaultGet = DefaultOperations.DefaultGet;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalRecompute = ComputeEngine.recompute;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalBrowse = ReadOperations.Browse;

  const meta = MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any) as any;
  const snapshotFields = new Map(meta.fields);
  const originalComputeGraph = meta.computeGraph;
  const parentUpdateCalls: any[] = [];
  const childSearchCalls: any[] = [];

  try {
    meta.fields.set('Lines', { type: 'OneToMany', relation: { targetModel: () => CreateChildModel, inverseField: 'ParentId' } });
    meta.fields.set('Total', { type: 'decimal', column: { scaleField: 'TotalScale' } });
    meta.fields.set('Bonus', { type: 'decimal', column: { scaleField: 'BonusScale' } });
    meta.fields.set('Growth', { type: 'int' });
    meta.fields.set('TotalScale', { type: 'int' });
    meta.fields.set('BonusScale', { type: 'int' });

    meta.computeGraph = {
      computeFields: new Set<string>(['Total', 'Bonus', 'Growth']),
      fastReverseDeps: new Map<string, string[]>([
        ['Lines', ['Total', 'Bonus']],
        ['Total', ['Bonus', 'Growth']],
      ]),
      computeScalarDeps: new Map<string, Set<string>>([['Total', new Set<string>(['Name'])]]),
      computeCollectionPathDeps: new Map<string, Array<{ collection: string; chain: string[] }>>([['Total', [{ collection: 'Lines', chain: [] }]]]),
    } as any;

    DefaultOperations.DefaultGet = (async (_ModelCtor: any, value: any) => ({ ...value })) as any;

    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value, Id: 'C-DUP-1' },
      relations: {
        oneToManyRelations: [{}],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(['Lines']),
      },
    })) as any;

    RelationFactory.batchProcessToManyRelations = (async () => [{ errors: [] }]) as any;

    RepositoryFactory.getRepository = ((ctor: any) => {
      if (ctor === ModelInternalPublicBridge) {
        return {
          withValidationBypass: async (fn: () => Promise<any>) => await fn(),
          create: async () => ['C-DUP-1'],
          search: async () => [
            {
              Id: 'C-DUP-1',
              Name: 'row',
              Total: '1.00',
              TotalScale: 2,
              Bonus: '2.00',
              BonusScale: 2,
            },
          ],
          update: async (vals: any, cond: any) => {
            parentUpdateCalls.push({ vals, cond });
          },
        } as any;
      }
      if (ctor === CreateChildModel) {
        return {
          search: async (cond: any, options?: any) => {
            childSearchCalls.push({ cond, options });
            return [{ Id: 'CL-1' }];
          },
        } as any;
      }
      throw new Error('unexpected repo ctor');
    }) as any;

    ComputeEngine.recompute = (async (_m: any, entity: any) => {
      entity.Total = '9.00';
      entity.Bonus = '3.00';
    }) as any;

    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ReadOperations.Browse = (async (_ModelCtor: any, id: string) => ({ Id: id, Name: 'ok' })) as any;

    const result = await CreateOperations.Create(ModelInternalPublicBridge as any, { Name: 'ok' } as any);

    expect((result as any).Id).toBe('C-DUP-1');
    expect(childSearchCalls.length).toBe(1);
    expect(childSearchCalls[0]?.options?.fields).toEqual(['Id']);
    expect(parentUpdateCalls.length).toBe(1);
    expect(parentUpdateCalls[0]?.vals?.Total).toBe('9.00');
    expect(parentUpdateCalls[0]?.vals?.Bonus).toBe('3.00');
  } finally {
    meta.fields = new Map(snapshotFields);
    meta.computeGraph = originalComputeGraph;
    DefaultOperations.DefaultGet = originalDefaultGet;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    RepositoryFactory.getRepository = originalGetRepository;
    ComputeEngine.recompute = originalRecompute;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ReadOperations.Browse = originalBrowse;
  }
});

test('CreateOperations.CreateMany forwards explicit returnFields and keeps condition id ordering', async () => {
  const originalDefaultGet = DefaultOperations.DefaultGet;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalSearch = ReadOperations.Search;
  const originalTriggerUpstreamCreateBatch = ComputeCascadeEngine.triggerUpstreamCreateBatch;

  const searchCalls: any[] = [];
  const createRows: any[] = [];

  try {
    DefaultOperations.DefaultGet = (async (_ModelCtor: any, value: any) => ({ ...value })) as any;
    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value },
      relations: emptyToManyRelations(),
    })) as any;

    RepositoryFactory.getRepository = (() => ({
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
      create: async (rows: any[]) => {
        createRows.push(...rows);
        return ['ID-A', 'ID-B'];
      },
    })) as any;

    ReadOperations.Search = (async (_ModelCtor: any, condition: any, options?: any) => {
      searchCalls.push({ condition, options });
      return [
        { Id: 'ID-A', Name: 'a' },
        { Id: 'ID-B', Name: 'b' },
      ];
    }) as any;

    ComputeCascadeEngine.triggerUpstreamCreateBatch = (async () => {}) as any;

    const result = await CreateOperations.CreateMany(ModelInternalPublicBridge as any, [{ Name: 'a' }, { Name: 'b' }] as any, ['Id', 'Name'] as any);

    expect(result.length).toBe(2);
    expect(searchCalls).toEqual([
      {
        condition: ['Id', 'in', ['ID-A', 'ID-B']],
        options: { fields: ['Id', 'Name'] },
      },
    ]);
    expect(createRows.length).toBe(2);
  } finally {
    DefaultOperations.DefaultGet = originalDefaultGet;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RepositoryFactory.getRepository = originalGetRepository;
    ReadOperations.Search = originalSearch;
    ComputeCascadeEngine.triggerUpstreamCreateBatch = originalTriggerUpstreamCreateBatch;
  }
});

test('UpdateOperations.UpdateById forwards options to Update and returns first row', async () => {
  const originalUpdate = UpdateOperations.Update;
  try {
    UpdateOperations.Update = (async (_ModelCtor: any, condition: any, values: any, returnFields: any, options: any) => {
      expect(condition).toEqual(['Id', '=', 'ROW-OPT']);
      expect(values).toEqual({ Name: 'next' });
      expect(returnFields).toEqual(['Id', 'Name']);
      expect(options).toEqual({ withDeleted: true, onlyDeleted: true });
      return [{ Id: 'ROW-OPT', Name: 'next' }];
    }) as any;

    const row = await UpdateOperations.UpdateById(
      ModelInternalPublicBridge as any,
      'ROW-OPT',
      { Name: 'next' } as any,
      ['Id', 'Name'] as any,
      { withDeleted: true, onlyDeleted: true } as any
    );

    expect((row as any).Id).toBe('ROW-OPT');
    expect((row as any).Name).toBe('next');
  } finally {
    UpdateOperations.Update = originalUpdate;
  }
});

test('UpdateOperations.Update strips computed input and runs collection-prefetch recompute pipeline', async () => {
  class UpdateChildModel {}

  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalCollectUpstreamInverseFields = ComputeCascadeEngine.collectUpstreamInverseFields;
  const originalRecompute = ComputeEngine.recompute;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;

  const meta = MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any) as any;
  const snapshotFields = new Map(meta.fields);
  const originalComputeGraph = meta.computeGraph;

  const prepareInputs: any[] = [];
  const updateCalls: any[] = [];
  const childSearchCalls: any[] = [];

  try {
    meta.fields.set('Amount', { type: 'decimal', column: { scaleField: 'AmountScale' } });
    meta.fields.set('AmountScale', { type: 'int' });
    meta.fields.set('Total', { type: 'decimal', column: { scaleField: 'TotalScale' } });
    meta.fields.set('TotalScale', { type: 'int' });
    meta.fields.set('Lines', { type: 'OneToMany', relation: { targetModel: () => UpdateChildModel, inverseField: 'ParentId' } });

    meta.computeGraph = {
      computeFields: new Set<string>(['Total']),
      fastReverseDeps: new Map<string, string[]>([
        ['Name', ['Total']],
        ['Lines', ['Total']],
      ]),
      computeScalarDeps: new Map<string, Set<string>>([['Total', new Set<string>(['Amount'])]]),
      computeCollectionPathDeps: new Map<string, Array<{ collection: string; chain: string[] }>>([['Total', [{ collection: 'Lines', chain: ['Qty'] }]]]),
    } as any;

    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => {
      prepareInputs.push({ ...values });
      return {
        processedValue: { ...values },
        relations: {
          oneToManyRelations: [{}],
          manyToManyRelations: [],
          touchedCollections: new Set<string>(['Lines']),
        },
      };
    }) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      count: async () => 1,
      search: async () => [
        {
          Id: 'ROW-UX',
          UpdatedAt: new Date('2024-01-01T00:00:00.000Z'),
          Name: 'before',
          Amount: '1.00',
          AmountScale: 2,
          Total: '0',
          TotalScale: 2,
        },
      ],
      update: async (vals: any, cond: any) => {
        updateCalls.push({ vals: { ...vals }, cond });
      },
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    RepositoryFactory.getRepository = ((ctor: any) => {
      if (ctor === UpdateChildModel) {
        return {
          search: async (condition: any, options?: any) => {
            childSearchCalls.push({ condition, options });
            return [{ Id: 'CL-1', Qty: 2 }];
          },
        } as any;
      }
      throw new Error('unexpected child repository');
    }) as any;

    RelationFactory.batchProcessToManyRelations = (async () => [{ errors: [] }]) as any;
    ComputeCascadeEngine.collectUpstreamInverseFields = (() => ['ParentId']) as any;

    ComputeEngine.recompute = (async (_meta: any, entity: any) => {
      entity.Total = '3.50';
      entity.TotalScale = 4;
    }) as any;

    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    const result = await UpdateOperations.Update(
      ModelInternalPublicBridge as any,
      ['Id', '=', 'ROW-UX'] as any,
      {
        Name: 'after',
        Amount: '2.00',
        Total: '999.00',
      } as any
    );

    expect(result).toEqual([{ Id: 'ROW-UX' }]);
    expect(prepareInputs[0]?.Total).toBe(undefined);
    expect(updateCalls.length).toBe(2);
    expect(updateCalls[0]?.vals?.AmountScale).toBe(2);
    expect(updateCalls[1]?.vals?.Total).toBe('3.50');
    expect(updateCalls[1]?.vals?.TotalScale).toBe(4);
    expect(childSearchCalls).toEqual([
      {
        condition: ['ParentId', '=', 'ROW-UX'],
        options: { fields: ['Id', 'Qty'] },
      },
    ]);
  } finally {
    meta.fields = new Map(snapshotFields);
    meta.computeGraph = originalComputeGraph;
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    RepositoryFactory.getRepository = originalGetRepository;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    ComputeCascadeEngine.collectUpstreamInverseFields = originalCollectUpstreamInverseFields;
    ComputeEngine.recompute = originalRecompute;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
  }
});

test('CreateOperations.CreateMany executes relation phase and swallows upstream batch failure', async () => {
  const originalDefaultGet = DefaultOperations.DefaultGet;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalRecompute = ComputeEngine.recompute;
  const originalTriggerUpstreamCreateBatch = ComputeCascadeEngine.triggerUpstreamCreateBatch;
  const originalSearch = ReadOperations.Search;
  const originalWarn = console.warn;

  const meta = MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any) as any;
  const originalComputeGraph = meta.computeGraph;
  const warnings: string[] = [];
  const relBatchCalls: any[] = [];

  try {
    meta.computeGraph = {
      computeFields: new Set<string>(['Name']),
      fastReverseDeps: new Map<string, string[]>([['Name', ['Name']]]),
      computePathDeps: new Map(),
      orderIndex: new Map([['Name', 0]]),
    } as any;

    DefaultOperations.DefaultGet = (async (_ModelCtor: any, value: any) => ({ ...value })) as any;
    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value },
      relations: {
        oneToManyRelations: [{}],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(),
      },
    })) as any;

    ComputeEngine.recompute = (async () => {}) as any;

    RepositoryFactory.getRepository = (() => ({
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
      create: async () => ['CM-R1'],
    })) as any;

    RelationFactory.batchProcessToManyRelations = (async (...args: any[]) => {
      relBatchCalls.push(args);
      return [{ errors: [] }];
    }) as any;

    ComputeCascadeEngine.triggerUpstreamCreateBatch = (async () => {
      throw new Error('batch-upstream-failed');
    }) as any;

    ReadOperations.Search = (async () => [{ Id: 'CM-R1', Name: 'ok' }]) as any;
    console.warn = (...args: any[]) => {
      warnings.push(args.map(item => String(item)).join(' '));
    };

    const result = await CreateOperations.CreateMany(ModelInternalPublicBridge as any, [{ Name: 'ok' }] as any, ['Id', 'Name'] as any);

    expect(result.length).toBe(1);
    expect((result[0] as any).Id).toBe('CM-R1');
    expect((result[0] as any).Name).toBe('ok');
    expect((result[0] as any).fields).toEqual(['Id', 'Name']);
    expect(relBatchCalls.length).toBe(1);
    expect(warnings.some(msg => msg.includes('CreateMany') && msg.includes('upstream recompute failed'))).toBe(true);
  } finally {
    meta.computeGraph = originalComputeGraph;
    DefaultOperations.DefaultGet = originalDefaultGet;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    RepositoryFactory.getRepository = originalGetRepository;
    ComputeEngine.recompute = originalRecompute;
    ComputeCascadeEngine.triggerUpstreamCreateBatch = originalTriggerUpstreamCreateBatch;
    ReadOperations.Search = originalSearch;
    console.warn = originalWarn;
  }
});

test('UpdateOperations.Update forwards soft-delete search options and supports fkField collection prefetch fallback', async () => {
  class UpdateFkChildModel {}

  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalResolveRepository = (UpdateOperations as any).resolveRepository;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalCollectUpstreamInverseFields = ComputeCascadeEngine.collectUpstreamInverseFields;
  const originalRecompute = ComputeEngine.recompute;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalTriggerDownstream = ComputeCascadeEngine.triggerDownstream;
  const originalSearch = ReadOperations.Search;

  const meta = MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any) as any;
  const snapshotFields = new Map(meta.fields);
  const originalComputeGraph = meta.computeGraph;

  const childSearchCalls: any[] = [];
  const updateCalls: any[] = [];
  const returnSearchCalls: any[] = [];

  try {
    meta.fields.set('Total', { type: 'decimal', column: { scaleField: 'TotalScale' } });
    meta.fields.set('TotalScale', { type: 'int' });
    meta.fields.set('Lines', { type: 'OneToMany', relation: { targetModel: () => UpdateFkChildModel, fkField: 'ParentFk' } });
    meta.computeGraph = {
      computeFields: new Set<string>(['Total']),
      fastReverseDeps: new Map<string, string[]>([
        ['Name', ['Total']],
        ['Lines', ['Total']],
      ]),
      computeScalarDeps: new Map<string, Set<string>>(),
      computeCollectionPathDeps: new Map<string, Array<{ collection: string; chain: string[] }>>([['Total', [{ collection: 'Lines', chain: ['Qty'] }]]]),
    } as any;

    RelationFactory.prepareForUpdate = (async (_ModelCtor: any, values: any) => ({
      processedValue: { ...values },
      relations: {
        oneToManyRelations: [{}],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(['Lines']),
      },
    })) as any;

    (UpdateOperations as any).resolveRepository = (() => ({
      count: async () => 1,
      search: async () => [
        {
          Id: 'ROW-FK',
          UpdatedAt: new Date('2024-01-01T00:00:00.000Z'),
          Name: 'before',
          Total: '0.00',
          TotalScale: 2,
        },
      ],
      update: async (vals: any, cond: any) => {
        updateCalls.push({ vals, cond });
      },
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
    })) as any;

    RepositoryFactory.getRepository = ((ctor: any) => {
      if (ctor === UpdateFkChildModel) {
        return {
          search: async (condition: any, options?: any) => {
            childSearchCalls.push({ condition, options });
            return [{ Id: 'L-FK-1', Qty: 1 }];
          },
        } as any;
      }
      throw new Error('unexpected child repo in fkField fallback test');
    }) as any;

    RelationFactory.batchProcessToManyRelations = (async () => [{ errors: [] }]) as any;
    ComputeCascadeEngine.collectUpstreamInverseFields = (() => []) as any;
    ComputeEngine.recompute = (async (_meta: any, entity: any) => {
      entity.Total = '6.50';
      entity.TotalScale = 4;
    }) as any;
    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ComputeCascadeEngine.triggerDownstream = (async () => {}) as any;

    ReadOperations.Search = (async (_ModelCtor: any, condition: any, options?: any) => {
      returnSearchCalls.push({ condition, options });
      return [{ Id: 'ROW-FK', Name: 'after', Total: '6.50' }];
    }) as any;

    const result = await UpdateOperations.Update(
      ModelInternalPublicBridge as any,
      ['Id', '=', 'ROW-FK'] as any,
      { Name: 'after' } as any,
      ['Id', 'Name'] as any,
      { withDeleted: true, onlyDeleted: true } as any
    );

    expect(result.length).toBe(1);
    expect((result[0] as any).Id).toBe('ROW-FK');
    expect(childSearchCalls).toEqual([
      {
        condition: ['ParentFk', '=', 'ROW-FK'],
        options: { fields: ['Id', 'Qty'] },
      },
    ]);
    expect(returnSearchCalls).toEqual([
      {
        condition: ['Id', 'in', ['ROW-FK']],
        options: { fields: ['Id', 'Name'], withDeleted: true, onlyDeleted: true },
      },
    ]);
    expect(updateCalls.length).toBe(2);
  } finally {
    meta.fields = new Map(snapshotFields);
    meta.computeGraph = originalComputeGraph;
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    (UpdateOperations as any).resolveRepository = originalResolveRepository;
    RepositoryFactory.getRepository = originalGetRepository;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    ComputeCascadeEngine.collectUpstreamInverseFields = originalCollectUpstreamInverseFields;
    ComputeEngine.recompute = originalRecompute;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ComputeCascadeEngine.triggerDownstream = originalTriggerDownstream;
    ReadOperations.Search = originalSearch;
  }
});

test('CreateOperations.Create warns and continues when collection prefetch relation misses inverse field', async () => {
  class CreateWarnChildModel {}

  const originalDefaultGet = DefaultOperations.DefaultGet;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalRecompute = ComputeEngine.recompute;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalBrowse = ReadOperations.Browse;
  const originalWarn = console.warn;

  const meta = MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any) as any;
  const snapshotFields = new Map(meta.fields);
  const originalComputeGraph = meta.computeGraph;

  let childSearchCount = 0;
  const parentUpdates: any[] = [];
  const warnings: string[] = [];

  try {
    meta.fields.set('Total', { type: 'decimal', column: { scaleField: 'TotalScale' } });
    meta.fields.set('TotalScale', { type: 'int' });
    meta.fields.set('Lines', { type: 'OneToMany', relation: { targetModel: () => CreateWarnChildModel } });
    meta.computeGraph = {
      computeFields: new Set<string>(['Total']),
      fastReverseDeps: new Map<string, string[]>([['Lines', ['Total']]]),
      computeScalarDeps: new Map<string, Set<string>>(),
      computeCollectionPathDeps: new Map<string, Array<{ collection: string; chain: string[] }>>([['Total', [{ collection: 'Lines', chain: ['Qty'] }]]]),
      orderIndex: new Map([['Total', 0]]),
    } as any;

    DefaultOperations.DefaultGet = (async (_ModelCtor: any, value: any) => ({ ...value })) as any;
    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value },
      relations: {
        oneToManyRelations: [{}],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(['Lines']),
      },
    })) as any;

    RelationFactory.batchProcessToManyRelations = (async () => [{ errors: [] }]) as any;

    RepositoryFactory.getRepository = ((ctor: any) => {
      if (ctor === ModelInternalPublicBridge) {
        return {
          withValidationBypass: async (fn: () => Promise<any>) => await fn(),
          create: async () => ['CW-1'],
          search: async () => [{ Id: 'CW-1', Total: '1.00', TotalScale: 2 }],
          update: async (vals: any, cond: any) => {
            parentUpdates.push({ vals, cond });
          },
        } as any;
      }
      if (ctor === CreateWarnChildModel) {
        return {
          search: async () => {
            childSearchCount += 1;
            return [];
          },
        } as any;
      }
      throw new Error('unexpected repo in create warn test');
    }) as any;

    ComputeEngine.recompute = (async (_m: any, entity: any) => {
      entity.Total = '2.00';
      entity.TotalScale = 3;
    }) as any;
    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;
    ReadOperations.Browse = (async (_ModelCtor: any, id: string) => ({ Id: id, Name: 'ok' })) as any;
    console.warn = (...args: any[]) => {
      warnings.push(args.map(item => String(item)).join(' '));
    };

    const result = await CreateOperations.Create(ModelInternalPublicBridge as any, { Name: 'ok' } as any);

    expect((result as any).Id).toBe('CW-1');
    expect(childSearchCount).toBe(0);
    expect(parentUpdates.length).toBe(1);
    expect(parentUpdates[0]?.vals?.Total).toBe('2.00');
    expect(warnings.some(msg => msg.includes('is missing targetModel or inverseField'))).toBe(true);
  } finally {
    meta.fields = new Map(snapshotFields);
    meta.computeGraph = originalComputeGraph;
    DefaultOperations.DefaultGet = originalDefaultGet;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    RepositoryFactory.getRepository = originalGetRepository;
    ComputeEngine.recompute = originalRecompute;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    ReadOperations.Browse = originalBrowse;
    console.warn = originalWarn;
  }
});

test('CreateOperations.Create delegates create without model-level persist recompute', async () => {
  const originalDefaultGet = DefaultOperations.DefaultGet;
  const originalPrepareForCreate = RelationFactory.prepareForCreate;
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalRecompute = ComputeEngine.recompute;
  const originalBrowse = ReadOperations.Browse;

  const meta = MetadataStorage.instance.getModelMetadata(ModelInternalPublicBridge as any) as any;
  const originalComputeGraph = meta.computeGraph;

  let createCalled = 0;

  try {
    meta.computeGraph = {
      computeFields: new Set<string>(['Name']),
      fastReverseDeps: new Map<string, string[]>(),
      computeScalarDeps: new Map<string, Set<string>>(),
      orderIndex: new Map([['Name', 0]]),
    } as any;

    DefaultOperations.DefaultGet = (async (_ModelCtor: any, value: any) => ({ ...value })) as any;
    RelationFactory.prepareForCreate = (async (_ModelCtor: any, value: any) => ({
      processedValue: { ...value },
      relations: emptyToManyRelations(),
    })) as any;

    RepositoryFactory.getRepository = (() => ({
      withValidationBypass: async (fn: () => Promise<any>) => await fn(),
      create: async () => {
        createCalled += 1;
        return ['ERR-1'];
      },
    })) as any;

    ComputeEngine.recompute = (async () => {
      throw new Error('compute-stage-failed');
    }) as any;

    ReadOperations.Browse = (async (_ModelCtor: any, id: string) => ({ Id: id })) as any;

    const created = await CreateOperations.Create(ModelInternalPublicBridge as any, { Name: 'boom' } as any);

    expect((created as any).Id).toBe('ERR-1');
    expect(createCalled).toBe(1);
  } finally {
    meta.computeGraph = originalComputeGraph;
    DefaultOperations.DefaultGet = originalDefaultGet;
    RelationFactory.prepareForCreate = originalPrepareForCreate;
    RepositoryFactory.getRepository = originalGetRepository;
    ComputeEngine.recompute = originalRecompute;
    ReadOperations.Browse = originalBrowse;
  }
});
