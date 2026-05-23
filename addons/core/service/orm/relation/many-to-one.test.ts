// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { Field, Model } from '../decorator';
import { RepositoryFactory } from '../repository/repository_factory';
import { ManyToOneProcessor } from './many-to-one';

@Model('test.ManyToOneTarget')
class ManyToOneTarget extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  static createCalls: Array<Record<string, any>> = [];

  static override async Create<T extends BaseModel>(this: { new (...args: any[]): T } & typeof BaseModel, value: Record<string, any>): Promise<T> {
    ManyToOneTarget.createCalls.push({ ...value });
    return { Id: `NEW-M2O-${ManyToOneTarget.createCalls.length}`, ...value } as T;
  }
}

@Model('test.ManyToOneParent')
class ManyToOneParent extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => ManyToOneTarget },
    column: {},
  })
  OwnerId?: ManyToOneTarget | null;
}

@Model('test.ManyToOneBrokenParent')
class ManyToOneBrokenParent extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  @Field({
    type: 'ManyToOne',
    relation: {} as any,
    column: {},
  })
  OwnerId?: any;
}

function resetTargetCalls() {
  ManyToOneTarget.createCalls = [];
}

function createParentRepoMock() {
  const updateCalls: Array<{ values: Record<string, any>; condition: any }> = [];

  return {
    updateCalls,
    repo: {
      async update(values: Record<string, any>, condition: any) {
        updateCalls.push({ values: { ...values }, condition });
        return [{ Id: 'ok' }];
      },
    },
  };
}

test('many-to-one processor prepareForCreate normalizes relation object into target Id', async () => {
  resetTargetCalls();

  const processor = new ManyToOneProcessor(ManyToOneParent as any);
  const result = await processor.prepareForCreate({ Name: 'parent-1', OwnerId: { Name: 'target-new' } as any });

  expect(result.processedValue).toEqual({ Name: 'parent-1', OwnerId: 'NEW-M2O-1' });
  expect(result.relations.oneToManyRelations).toEqual([]);
  expect(result.relations.manyToManyRelations).toEqual([]);
  expect(ManyToOneTarget.createCalls).toEqual([{ Name: 'target-new' }]);
});

test('many-to-one processor prepareForCreate handles null relation values and metadata fallback loop', async () => {
  resetTargetCalls();

  const processor = new ManyToOneProcessor(ManyToOneParent as any) as any;
  const detached = await processor.prepareForCreate({ Name: 'parent-null', OwnerId: null });
  expect(detached.processedValue).toEqual({ Name: 'parent-null', OwnerId: null });

  processor.metadata = { ...processor.metadata, fields: undefined };
  const noMetadata = await processor.prepareForCreate({ Name: 'parent-raw', OwnerId: { Name: 'raw-target' } as any });
  expect(noMetadata.processedValue).toEqual({ Name: 'parent-raw', OwnerId: { Name: 'raw-target' } });
  expect(ManyToOneTarget.createCalls).toEqual([]);
});

test('many-to-one processor prepareForCreate throws when relation targetModel is missing', async () => {
  const processor = new ManyToOneProcessor(ManyToOneBrokenParent as any);

  let message = '';
  try {
    await processor.prepareForCreate({ OwnerId: { Name: 'x' } as any });
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('ManyToOne field is missing relation.targetModel')).toBe(true);
  expect(message.includes('ManyToOneBrokenParent.OwnerId')).toBe(true);
});

test('many-to-one processor processRelationUpdate sets null to detach relation', async () => {
  const parentStore = createParentRepoMock();
  RepositoryFactory.setRepository(ManyToOneParent as any, parentStore.repo as any);

  const processor = new ManyToOneProcessor(ManyToOneParent as any);
  const result = await processor.processRelationUpdate('p1', {
    type: 'ManyToOne',
    fieldName: 'OwnerId',
    targetModel: ManyToOneTarget as any,
    value: null,
  });

  expect(result.errors).toEqual([]);
  expect(result.entityIds).toEqual(['p1']);
  expect(result.affectedCount).toBe(1);
  expect(parentStore.updateCalls).toEqual([
    {
      values: { OwnerId: null },
      condition: ['Id', '=', 'p1'],
    },
  ]);
});

test('many-to-one processor batchProcessRelationUpdate groups same target id into one update', async () => {
  const parentStore = createParentRepoMock();
  RepositoryFactory.setRepository(ManyToOneParent as any, parentStore.repo as any);

  const processor = new ManyToOneProcessor(ManyToOneParent as any);
  const result = await processor.batchProcessRelationUpdate(
    ['p1', 'p2', 'p3'],
    [
      { type: 'ManyToOne', fieldName: 'OwnerId', targetModel: ManyToOneTarget as any, value: 't1' },
      { type: 'ManyToOne', fieldName: 'OwnerId', targetModel: ManyToOneTarget as any, value: { Id: 't1' } as any },
      { type: 'ManyToOne', fieldName: 'OwnerId', targetModel: ManyToOneTarget as any, value: null },
    ]
  );

  expect(result.errors).toEqual([]);
  expect(result.summary).toEqual({
    totalOperations: 3,
    successfulOperations: 3,
    failedOperations: 0,
    relationType: 'ManyToOne',
  });

  expect(parentStore.updateCalls.length).toBe(2);
  expect(parentStore.updateCalls[0]).toEqual({
    values: { OwnerId: 't1' },
    condition: ['Id', 'in', ['p1', 'p2']],
  });
  expect(parentStore.updateCalls[1]).toEqual({
    values: { OwnerId: null },
    condition: ['Id', '=', 'p3'],
  });
});

test('many-to-one processor prepareForUpdate respects changedFields and normalizes included relation field', async () => {
  resetTargetCalls();

  const processor = new ManyToOneProcessor(ManyToOneParent as any);
  const skipped = await processor.prepareForUpdate({ OwnerId: { Name: 'skip' } as any }, ['Name']);
  expect(skipped.processedValue).toEqual({ OwnerId: { Name: 'skip' } });

  const normalized = await processor.prepareForUpdate({ OwnerId: { Name: 'target-updated' } as any }, ['OwnerId']);
  expect(normalized.processedValue).toEqual({ OwnerId: 'NEW-M2O-1' });
  expect(ManyToOneTarget.createCalls).toEqual([{ Name: 'target-updated' }]);
});

test('many-to-one processor prepareForUpdate throws when relation targetModel is missing', async () => {
  const processor = new ManyToOneProcessor(ManyToOneBrokenParent as any);

  let message = '';
  try {
    await processor.prepareForUpdate({ OwnerId: { Name: 'x' } as any }, ['OwnerId']);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('ManyToOne field is missing relation.targetModel')).toBe(true);
  expect(message.includes('ManyToOneBrokenParent.OwnerId')).toBe(true);
});

test('many-to-one processor prepareForUpdate handles missing relation key, null values and metadata fallback loop', async () => {
  const processor = new ManyToOneProcessor(ManyToOneParent as any) as any;

  const missingKey = await processor.prepareForUpdate({ Name: 'untouched' }, ['OwnerId']);
  expect(missingKey.processedValue).toEqual({ Name: 'untouched' });

  const detached = await processor.prepareForUpdate({ OwnerId: null }, ['OwnerId']);
  expect(detached.processedValue).toEqual({ OwnerId: null });

  processor.metadata = { ...processor.metadata, fields: undefined };
  const noMetadata = await processor.prepareForUpdate({ OwnerId: { Name: 'raw-target' } as any }, ['OwnerId']);
  expect(noMetadata.processedValue).toEqual({ OwnerId: { Name: 'raw-target' } });
});

test('many-to-one processor processRelationUpdate rejects non-many-to-one operation', async () => {
  const processor = new ManyToOneProcessor(ManyToOneParent as any);

  let message = '';
  try {
    await processor.processRelationUpdate('p1', {
      type: 'ManyToMany' as any,
      fieldName: 'OwnerId',
      targetModel: ManyToOneTarget as any,
      value: null,
    });
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe('Expected a ManyToOne operation, but received ManyToMany');
});

test('many-to-one processor batchProcessRelationUpdate validates input shape and aggregates conversion errors', async () => {
  const parentStore = createParentRepoMock();
  RepositoryFactory.setRepository(ManyToOneParent as any, parentStore.repo as any);
  const processor = new ManyToOneProcessor(ManyToOneParent as any) as any;

  let mismatchMessage = '';
  try {
    await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        { type: 'ManyToOne', fieldName: 'OwnerId', targetModel: ManyToOneTarget as any, value: 't1' },
        { type: 'ManyToOne', fieldName: 'OwnerId', targetModel: ManyToOneTarget as any, value: 't2' },
      ]
    );
  } catch (error) {
    mismatchMessage = String((error as Error)?.message || error);
  }
  expect(mismatchMessage).toBe('Parent entity Id array length must match relation operation array length');

  let typeMessage = '';
  try {
    await processor.batchProcessRelationUpdate(['p1'], [{ type: 'OneToMany' as any, fieldName: 'OwnerId', targetModel: ManyToOneTarget as any, value: null }]);
  } catch (error) {
    typeMessage = String((error as Error)?.message || error);
  }
  expect(typeMessage).toBe('Expected a ManyToOne operation, but received OneToMany');

  processor.getOrCreateId = async (value: any) => {
    if (value === 'bad-target') throw new Error('bad-target');
    return String(value);
  };

  const result = await processor.batchProcessRelationUpdate(
    ['p1', 'p2'],
    [
      { type: 'ManyToOne', fieldName: 'OwnerId', targetModel: ManyToOneTarget as any, value: 'bad-target' },
      { type: 'ManyToOne', fieldName: 'OwnerId', targetModel: ManyToOneTarget as any, value: 'ok-target' },
    ]
  );

  expect(result.summary).toEqual({
    totalOperations: 2,
    successfulOperations: 1,
    failedOperations: 1,
    relationType: 'ManyToOne',
  });
  expect(result.success.length).toBe(1);
  expect(result.success[0]?.entityId).toBe('p2');
  expect(result.errors.length).toBe(1);
  expect(String(result.errors[0]?.error?.message || '')).toBe('bad-target');
  expect(parentStore.updateCalls).toEqual([
    {
      values: { OwnerId: 'ok-target' },
      condition: ['Id', '=', 'p2'],
    },
  ]);
});

test('many-to-one processor processRelationUpdate normalizes thrown Error and non-Error values', async () => {
  const parentStore = {
    updateCalls: [] as any[],
    repo: {
      update: async (...args: any[]) => {
        parentStore.updateCalls.push(args);
        throw new Error('db-error');
      },
    },
  };
  RepositoryFactory.setRepository(ManyToOneParent as any, parentStore.repo as any);

  const processor = new ManyToOneProcessor(ManyToOneParent as any) as any;
  processor.getOrCreateId = async () => 'T-1';

  const byError = await processor.processRelationUpdate('p1', {
    type: 'ManyToOne',
    fieldName: 'OwnerId',
    targetModel: ManyToOneTarget as any,
    value: 'x',
  });
  expect(String(byError.errors[0]?.message || '')).toBe('db-error');

  parentStore.repo.update = async (...args: any[]) => {
    parentStore.updateCalls.push(args);
    throw 'db-string';
  };
  const byString = await processor.processRelationUpdate('p2', {
    type: 'ManyToOne',
    fieldName: 'OwnerId',
    targetModel: ManyToOneTarget as any,
    value: 'y',
  });
  expect(String(byString.errors[0]?.message || '')).toBe('db-string');
});

test('many-to-one processor batchProcessRelationUpdate collects non-Error conversion and update failures', async () => {
  const parentStore = {
    repo: {
      update: async (_values: Record<string, any>) => {
        throw 'update-string';
      },
    },
  };
  RepositoryFactory.setRepository(ManyToOneParent as any, parentStore.repo as any);

  const processor = new ManyToOneProcessor(ManyToOneParent as any) as any;
  processor.getOrCreateId = async (value: any) => {
    if (value === 'bad-convert') throw 'convert-string';
    return String(value);
  };

  const result = await processor.batchProcessRelationUpdate(
    ['p1', 'p2'],
    [
      { type: 'ManyToOne', fieldName: 'OwnerId', targetModel: ManyToOneTarget as any, value: 'bad-convert' },
      { type: 'ManyToOne', fieldName: 'OwnerId', targetModel: ManyToOneTarget as any, value: 'ok-target' },
    ]
  );

  const messages = result.errors.map((item: any) => String(item.error?.message || item.error));
  expect(messages.includes('convert-string')).toBe(true);
  expect(messages.includes('update-string')).toBe(true);
});

test('many-to-one processor batchProcessRelationUpdate keeps Error objects from update failures', async () => {
  const parentStore = {
    repo: {
      update: async () => {
        throw new Error('update-error');
      },
    },
  };
  RepositoryFactory.setRepository(ManyToOneParent as any, parentStore.repo as any);

  const processor = new ManyToOneProcessor(ManyToOneParent as any) as any;
  processor.getOrCreateId = async (value: any) => String(value);

  const result = await processor.batchProcessRelationUpdate(
    ['p1'],
    [{ type: 'ManyToOne', fieldName: 'OwnerId', targetModel: ManyToOneTarget as any, value: 'ok-target' }]
  );

  expect(result.success.length).toBe(0);
  expect(result.errors.length).toBe(1);
  expect(String(result.errors[0]?.error?.message || '')).toBe('update-error');
});

test('many-to-one processor batchProcessRelationUpdate wraps repository acquisition failures', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;

  try {
    RepositoryFactory.getRepository = (() => {
      throw new Error('outer-error');
    }) as any;

    const processor = new ManyToOneProcessor(ManyToOneParent as any);
    const byError = await processor.batchProcessRelationUpdate(
      ['p1'],
      [{ type: 'ManyToOne', fieldName: 'OwnerId', targetModel: ManyToOneTarget as any, value: 'x' }]
    );
    expect(String(byError.errors[0]?.error?.message || '')).toBe('outer-error');

    RepositoryFactory.getRepository = (() => {
      throw 'outer-string';
    }) as any;

    const byString = await processor.batchProcessRelationUpdate(
      ['p2'],
      [{ type: 'ManyToOne', fieldName: 'OwnerId', targetModel: ManyToOneTarget as any, value: 'y' }]
    );
    expect(String(byString.errors[0]?.error?.message || '')).toBe('outer-string');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});
