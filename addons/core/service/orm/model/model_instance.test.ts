// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field } from '../decorator';
import { RepositoryFactory } from '../repository/repository_factory';
import { RelationFactory } from '../relation';
import { MODEL_SYMBOLS } from '../../runtime/proxy';
import { ComputeEngine } from '../../runtime/compute/engine';
import { ComputeCascadeEngine } from '../../runtime/compute/cascade';
import BaseModel from './model';
import { deleteModelInstance, loadModelInstance, reloadModelInstance, toTransportObject, updateModelInstance } from './model_instance';

class ModelInstanceHarness extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name!: string;
}

function newHarness(entity: Record<string, any>) {
  const token = (BaseModel as any).FACTORY_TOKEN as symbol;
  return new ModelInstanceHarness(token, entity as any, undefined as any) as any;
}

test('update/delete/load/reload throw when instance id is missing', async () => {
  const instance = newHarness({});

  let updateErr = '';
  try {
    await updateModelInstance(instance as any);
  } catch (error) {
    updateErr = String((error as Error).message || error);
  }
  expect(updateErr).toContain('Cannot update an instance without Id');

  let deleteErr = '';
  try {
    await deleteModelInstance(instance as any);
  } catch (error) {
    deleteErr = String((error as Error).message || error);
  }
  expect(deleteErr).toContain('Cannot delete an instance without Id');

  let loadErr = '';
  try {
    await loadModelInstance(instance as any);
  } catch (error) {
    loadErr = String((error as Error).message || error);
  }
  expect(loadErr).toContain('Cannot load an instance without Id');

  let reloadErr = '';
  try {
    await reloadModelInstance(instance as any);
  } catch (error) {
    reloadErr = String((error as Error).message || error);
  }
  expect(reloadErr).toContain('Cannot reload an instance without Id');
});

test('updateModelInstance returns instance when nothing changed', async () => {
  const instance = newHarness({ Id: 'MI-1', Name: 'a' });
  instance[MODEL_SYMBOLS.getChangedFields] = () => [];
  instance[MODEL_SYMBOLS.collectRelationChanges] = () => ({});

  const out = await updateModelInstance(instance as any);
  expect(out).toBe(instance);
});

test('loadModelInstance merges fields and hydrates model data', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const searchCalls: any[] = [];

  try {
    RepositoryFactory.getRepository = (() => ({
      search: async (condition: any, options: any) => {
        searchCalls.push({ condition, options });
        return [{ Id: 'MI-2', Name: 'loaded-name' }];
      },
    })) as any;

    const instance = newHarness({ Id: 'MI-2' });
    await loadModelInstance(instance as any, ['Id'] as any);
    await loadModelInstance(instance as any, ['Name'] as any);

    expect(instance.Name).toBe('loaded-name');
    expect(instance.fields).toEqual(['Id', 'Name']);
    expect(searchCalls.length).toBe(2);
    expect(searchCalls[0]).toEqual({ condition: ['Id', '=', 'MI-2'], options: { fields: ['Id'] } });
    expect(searchCalls[1]).toEqual({ condition: ['Id', '=', 'MI-2'], options: { fields: ['Name'] } });
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('reloadModelInstance uses current fields and resets change trackers', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const searchCalls: any[] = [];

  try {
    RepositoryFactory.getRepository = (() => ({
      search: async (condition: any, options: any) => {
        searchCalls.push({ condition, options });
        return [{ Id: 'MI-3', Name: 'reloaded-name' }];
      },
    })) as any;

    const resetCalls = { changed: 0, relation: 0 };
    const instance = newHarness({ Id: 'MI-3', Name: 'before' });
    instance.fields = ['Id', 'Name'];
    instance[MODEL_SYMBOLS.resetChanges] = () => {
      resetCalls.changed += 1;
    };
    instance[MODEL_SYMBOLS.resetRelationChanges] = () => {
      resetCalls.relation += 1;
    };

    const out = await reloadModelInstance(instance as any);

    expect(out).toBe(instance);
    expect(instance.Name).toBe('reloaded-name');
    expect(searchCalls).toEqual([{ condition: ['Id', '=', 'MI-3'], options: { fields: ['Id', 'Name'] } }]);
    expect(resetCalls).toEqual({ changed: 1, relation: 1 });
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('toTransportObject returns plain payload for model instance', () => {
  const instance = newHarness({ Id: 'MI-TO-1', Name: 'transport-name' });

  const out = toTransportObject(instance as any);

  expect((out as any).Id).toBe('MI-TO-1');
  expect((out as any).Name).toBe('transport-name');
});

test('updateModelInstance wraps non-optimistic repository errors', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;

  try {
    RepositoryFactory.getRepository = (() => ({
      search: async () => {
        throw new Error('db boom');
      },
    })) as any;

    const instance = newHarness({ Id: 'MI-4', Name: 'before' });
    instance.Name = 'after';
    instance[MODEL_SYMBOLS.getChangedFields] = () => ['Name'];
    instance[MODEL_SYMBOLS.collectRelationChanges] = () => ({});

    let message = '';
    try {
      await updateModelInstance(instance as any);
    } catch (error) {
      message = String((error as Error).message || error);
    }

    expect(message).toContain('Update failed: db boom');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('updateModelInstance keeps optimistic-lock style errors unwrapped', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;

  try {
    RepositoryFactory.getRepository = (() => ({
      search: async () => {
        throw new Error('has been modified');
      },
    })) as any;

    const instance = newHarness({ Id: 'MI-5', Name: 'before' });
    instance.Name = 'after';
    instance[MODEL_SYMBOLS.getChangedFields] = () => ['Name'];
    instance[MODEL_SYMBOLS.collectRelationChanges] = () => ({});

    let message = '';
    try {
      await updateModelInstance(instance as any);
    } catch (error) {
      message = String((error as Error).message || error);
    }

    expect(message).toContain('has been modified');
    expect(message.includes('Update failed:')).toBe(false);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('deleteModelInstance throws when repository delete returns empty result', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;

  try {
    RepositoryFactory.getRepository = (() => ({
      search: async () => [{ Id: 'MI-6' }],
      delete: async () => [],
    })) as any;

    const instance = newHarness({ Id: 'MI-6' });
    let message = '';
    try {
      await deleteModelInstance(instance as any);
    } catch (error) {
      message = String((error as Error).message || error);
    }

    expect(message).toContain('Delete failed: Record with Id MI-6 not found');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('loadModelInstance throws when record no longer exists and reload defaults to wildcard fields', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const searchCalls: any[] = [];

  try {
    RepositoryFactory.getRepository = (() => ({
      search: async (_condition: any, options: any) => {
        searchCalls.push(options);
        if (searchCalls.length === 1) return [];
        return [{ Id: 'MI-7', Name: 'after-reload' }];
      },
    })) as any;

    const instance = newHarness({ Id: 'MI-7', Name: 'before-reload' });

    let message = '';
    try {
      await loadModelInstance(instance as any, ['Id'] as any);
    } catch (error) {
      message = String((error as Error).message || error);
    }
    expect(message).toContain('Record with Id MI-7 no longer exists');

    const resetCalls = { changed: 0, relation: 0 };
    instance[MODEL_SYMBOLS.resetChanges] = () => {
      resetCalls.changed += 1;
    };
    instance[MODEL_SYMBOLS.resetRelationChanges] = () => {
      resetCalls.relation += 1;
    };

    await reloadModelInstance(instance as any);

    expect(searchCalls[1]).toEqual({ fields: ['*'] });
    expect(instance.Name).toBe('after-reload');
    expect(resetCalls).toEqual({ changed: 1, relation: 1 });
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('updateModelInstance handles relation processing and follow-up compute persistence', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalPrepareRelationChanges = RelationFactory.prepareRelationChanges;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalRecompute = ComputeEngine.recompute;
  const originalCollectUpstreamInverseFields = ComputeCascadeEngine.collectUpstreamInverseFields;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;

  const searchCalls: any[] = [];
  const updateCalls: any[] = [];
  const resetCalls = { changed: 0, relation: 0 };
  let bypassCalls = 0;
  let relationPrepareCalls = 0;
  let batchCalls = 0;

  try {
    RepositoryFactory.getRepository = (() => ({
      search: async (_condition: any, options: any) => {
        searchCalls.push(options);
        if (searchCalls.length === 1) {
          return [{ Id: 'MI-8', UpdatedAt: new Date('2024-01-01T00:00:00.000Z'), ParentId: 'P-1', Total: 10 }];
        }
        return [{ Id: 'MI-8', Name: 'after', UpdatedAt: new Date('2024-01-02T00:00:00.000Z') }];
      },
      update: async (values: any, condition: any) => {
        updateCalls.push({ values, condition });
        return [{ Id: 'MI-8' }];
      },
      withValidationBypass: async (fn: () => Promise<any>) => {
        bypassCalls += 1;
        return await fn();
      },
    })) as any;

    RelationFactory.prepareForUpdate = (async () => ({
      processedValue: { Name: 'after' },
      relations: {
        oneToManyRelations: [{ fieldName: 'Lines' }],
        manyToManyRelations: [],
        touchedCollections: new Set(['Total']),
      },
    })) as any;

    RelationFactory.prepareRelationChanges = (() => {
      relationPrepareCalls += 1;
    }) as any;

    RelationFactory.batchProcessToManyRelations = (async () => {
      batchCalls += 1;
      return [{ errors: [] }];
    }) as any;

    ComputeEngine.recompute = (async (_meta: any, entity: any, changed: Set<string>) => {
      if (changed.has('Total')) entity.Total = 20;
    }) as any;

    ComputeCascadeEngine.collectUpstreamInverseFields = (() => ['ParentId']) as any;
    ComputeCascadeEngine.triggerUpstream = (async () => {}) as any;

    const instance = newHarness({ Id: 'MI-8', Name: 'before', Total: 10 });
    instance.Name = 'after';
    instance[MODEL_SYMBOLS.getChangedFields] = () => ['Name'];
    instance[MODEL_SYMBOLS.collectRelationChanges] = () => ({ Lines: [{ method: 'push', args: [] }] });
    instance[MODEL_SYMBOLS.resetChanges] = () => {
      resetCalls.changed += 1;
    };
    instance[MODEL_SYMBOLS.resetRelationChanges] = () => {
      resetCalls.relation += 1;
    };

    const out = await updateModelInstance(instance as any);

    expect(out).toBe(instance);
    expect(relationPrepareCalls).toBe(1);
    expect(batchCalls).toBe(1);
    expect(searchCalls[0]).toEqual({ fields: ['Id', 'UpdatedAt', 'ParentId'] });
    expect(updateCalls.length).toBe(2);
    expect(bypassCalls).toBe(2);
    expect(updateCalls[0].condition?.And?.[0]).toEqual(['Id', '=', 'MI-8']);
    expect(updateCalls[1].condition).toEqual(['Id', '=', 'MI-8']);
    expect(updateCalls[1].values.Total).toBe(20);
    expect(resetCalls).toEqual({ changed: 2, relation: 2 });
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    RelationFactory.prepareRelationChanges = originalPrepareRelationChanges;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    ComputeEngine.recompute = originalRecompute;
    ComputeCascadeEngine.collectUpstreamInverseFields = originalCollectUpstreamInverseFields;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
  }
});

test('updateModelInstance fails when relation batch processing returns errors', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalPrepareForUpdate = RelationFactory.prepareForUpdate;
  const originalBatchProcess = RelationFactory.batchProcessToManyRelations;
  const originalRecompute = ComputeEngine.recompute;

  try {
    RepositoryFactory.getRepository = (() => ({
      search: async () => [{ Id: 'MI-9', UpdatedAt: new Date('2024-01-01T00:00:00.000Z') }],
      update: async () => [{ Id: 'MI-9' }],
    })) as any;

    RelationFactory.prepareForUpdate = (async () => ({
      processedValue: { Name: 'after' },
      relations: {
        oneToManyRelations: [{ fieldName: 'Lines' }],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(),
      },
    })) as any;

    RelationFactory.batchProcessToManyRelations = (async () => [{ errors: [{ error: new Error('child failed') }] }]) as any;
    ComputeEngine.recompute = (async () => {}) as any;

    const instance = newHarness({ Id: 'MI-9', Name: 'before' });
    instance.Name = 'after';
    instance[MODEL_SYMBOLS.getChangedFields] = () => ['Name'];
    instance[MODEL_SYMBOLS.collectRelationChanges] = () => ({ Lines: [{ method: 'push', args: [] }] });

    let message = '';
    try {
      await updateModelInstance(instance as any);
    } catch (error) {
      message = String((error as Error).message || error);
    }

    expect(message).toContain('Update failed: [update] relation handling failed');
    expect(message).toContain('child failed');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
    RelationFactory.prepareForUpdate = originalPrepareForUpdate;
    RelationFactory.batchProcessToManyRelations = originalBatchProcess;
    ComputeEngine.recompute = originalRecompute;
  }
});

test('deleteModelInstance swallows upstream trigger error and warns', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  const originalTriggerUpstream = ComputeCascadeEngine.triggerUpstream;
  const originalWarn = console.warn;

  const warnings: string[] = [];

  try {
    RepositoryFactory.getRepository = (() => ({
      search: async () => [{ Id: 'MI-10' }],
      delete: async () => [{ Id: 'MI-10' }],
    })) as any;

    ComputeCascadeEngine.triggerUpstream = (async () => {
      throw new Error('cascade failed');
    }) as any;

    console.warn = ((...args: unknown[]) => {
      warnings.push(args.map(v => String(v)).join(' '));
    }) as unknown as typeof console.warn;

    const instance = newHarness({ Id: 'MI-10' });
    let resetCalls = 0;
    instance[MODEL_SYMBOLS.resetChanges] = () => {
      resetCalls += 1;
    };

    await deleteModelInstance(instance as any);

    expect(resetCalls).toBe(1);
    expect(warnings.some(line => line.includes('[delete] parent compute trigger failed:'))).toBe(true);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
    ComputeCascadeEngine.triggerUpstream = originalTriggerUpstream;
    console.warn = originalWarn;
  }
});
