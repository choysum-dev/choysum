// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { Field, Model } from '../decorator';
import { MetadataStorage } from '../metadata';
import { Repository } from '../repository/repository';
import { RepositoryFactory } from '../repository/repository_factory';
import { ManyToManyProcessor } from './many-to-many';

type ModelCtor<T extends BaseModel> = { new (...args: never[]): T } & typeof BaseModel;

type JoinRow = {
  Id: string;
  OwnerId: string;
  TagId: string;
};

function matchesCondition(row: Record<string, unknown>, condition: unknown): boolean {
  if (Array.isArray(condition)) {
    const [field, op, value] = condition;
    const left = row[String(field)];
    const normalizedOp = String(op || '').toLowerCase();
    if (normalizedOp === '=') return left === value;
    if (normalizedOp === 'in') return Array.isArray(value) && value.includes(left);
    return false;
  }

  if (condition && typeof condition === 'object' && Array.isArray((condition as { And?: unknown[] }).And)) {
    return (condition as { And: unknown[] }).And.every(part => matchesCondition(row, part));
  }

  if (condition && typeof condition === 'object' && Array.isArray((condition as { Or?: unknown[] }).Or)) {
    return (condition as { Or: unknown[] }).Or.some(part => matchesCondition(row, part));
  }

  return false;
}

function createJoinRepoMock(seedRows: JoinRow[]) {
  const rows = seedRows.map(row => ({ ...row }));
  const createCalls: Array<Record<string, any>[]> = [];
  const deleteCalls: unknown[] = [];

  const originalSearch = Repository.prototype.search;
  const originalDelete = Repository.prototype.delete;
  const originalCreate = Repository.prototype.create;

  Repository.prototype.search = async function (this: Repository, condition: unknown) {
    return rows.filter(row => matchesCondition(row, condition)).map(row => ({ ...row }));
  } as unknown as Repository['search'];

  Repository.prototype.delete = async function (this: Repository, condition: unknown) {
    deleteCalls.push(condition);
    const matchedIds = rows.filter(row => matchesCondition(row, condition)).map(row => row.Id);
    for (const id of matchedIds) {
      const idx = rows.findIndex(row => row.Id === id);
      if (idx >= 0) rows.splice(idx, 1);
    }
    return matchedIds.map(id => ({ Id: id }));
  } as unknown as Repository['delete'];

  Repository.prototype.create = async function (values: Record<string, any>[]) {
    createCalls.push(values.map(v => ({ ...v })));
    for (const value of values) {
      rows.push({
        Id: `j-new-${rows.length + 1}`,
        OwnerId: String(value.OwnerId ?? ''),
        TagId: String(value.TagId ?? ''),
      });
    }
    return values.map((_, idx) => `j-created-${idx + 1}`);
  } as unknown as Repository['create'];

  return {
    rows,
    createCalls,
    deleteCalls,
    restore() {
      Repository.prototype.search = originalSearch;
      Repository.prototype.delete = originalDelete;
      Repository.prototype.create = originalCreate;
    },
  };
}

@Model('test.ManyToManyParent')
class ManyToManyParent extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;
}

@Model('test.ManyToManyTarget')
class ManyToManyTarget extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  static createCalls: Array<Record<string, any>> = [];
  static updateCalls: Array<{ id: string; values: Record<string, any> }> = [];

  static override async Create<T extends BaseModel>(this: { new (...args: any[]): T } & typeof BaseModel, value: Record<string, any>): Promise<T> {
    ManyToManyTarget.createCalls.push({ ...value });
    return { Id: `NEW-TARGET-${ManyToManyTarget.createCalls.length}`, ...value } as T;
  }

  static override async UpdateById<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    values: Record<string, any>
  ): Promise<Partial<T>> {
    ManyToManyTarget.updateCalls.push({ id, values: { ...values } });
    return { Id: id, ...values } as Partial<T>;
  }
}

@Model('test.ManyToManyJoin')
class ManyToManyJoin extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  OwnerId?: string;

  @Field({ type: 'varchar', column: { size: 64 } })
  TagId?: string;
}

@Model('test.ManyToManyPrepareParent')
class ManyToManyPrepareParent extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  @Field({
    type: 'ManyToMany',
    relation: {
      joinModel: () => ManyToManyJoin,
      targetModel: () => ManyToManyTarget,
      joinField: 'OwnerId' as never,
      inverseJoinField: 'TagId' as never,
    },
  })
  Tags?: ManyToManyTarget[];
}

const parentCtor = ManyToManyParent as unknown as ModelCtor<ManyToManyParent>;
const targetCtor = ManyToManyTarget as unknown as ModelCtor<ManyToManyTarget>;
const joinCtor = ManyToManyJoin as unknown as ModelCtor<ManyToManyJoin>;
const prepareParentCtor = ManyToManyPrepareParent as unknown as ModelCtor<ManyToManyPrepareParent>;

function resetTargetCalls() {
  ManyToManyTarget.createCalls = [];
  ManyToManyTarget.updateCalls = [];
}

test('many-to-many processor array replace removes obsolete join rows and creates new relations', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([
    { Id: 'j1', OwnerId: 'p1', TagId: 't1' },
    { Id: 'j2', OwnerId: 'p1', TagId: 't2' },
  ]);

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'ManyToMany',
      fieldName: 'Tags',
      joinModel: joinCtor,
      targetModel: targetCtor,
      joinField: 'OwnerId',
      inverseJoinField: 'TagId',
      operations: [{ Id: 't2' }, { Name: 'fresh-tag' }] as unknown as Parameters<ManyToManyProcessor['processRelationUpdate']>[1]['operations'],
    });

    expect(result.errors).toEqual([]);
    expect(result.entityIds.includes('t2')).toBe(true);
    expect(result.entityIds.includes('NEW-TARGET-1')).toBe(true);
    expect(ManyToManyTarget.createCalls).toEqual([{ Name: 'fresh-tag' }]);
    expect(joinStore.deleteCalls.length).toBe(1);
    expect(joinStore.createCalls.length).toBe(1);
    expect(joinStore.createCalls[0]).toEqual([{ OwnerId: 'p1', TagId: 'NEW-TARGET-1' }]);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor object operations return per-item errors for invalid delete and non-related update', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([
    { Id: 'j1', OwnerId: 'p1', TagId: 't1' },
    { Id: 'j2', OwnerId: 'p1', TagId: 't2' },
  ]);

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'ManyToMany',
      fieldName: 'Tags',
      joinModel: joinCtor,
      targetModel: targetCtor,
      joinField: 'OwnerId',
      inverseJoinField: 'TagId',
      operations: {
        delete: [{} as unknown as { Id: string }],
        create: [{ Name: 'brand-new' }] as unknown as Array<{ Id: string } | Record<string, unknown>>,
        update: [{ Id: 't2', Name: 'updated-name' }, { Id: 't999', Name: 'not-related' }, { Name: 'missing-id' } as unknown as { Id: string; Name: string }],
      },
    } as unknown as Parameters<ManyToManyProcessor['processRelationUpdate']>[1]);

    expect(result.entityIds.includes('NEW-TARGET-1')).toBe(true);
    expect(result.entityIds.includes('t2')).toBe(true);
    expect(result.errors.length).toBe(3);
    expect(result.errors.some(error => String(error.message || '').includes('Could not extract Id from delete item'))).toBe(true);
    expect(result.errors.some(error => String(error.message || '').includes('Attempted to update a non-related entity'))).toBe(true);
    expect(result.errors.some(error => String(error.message || '').includes('Could not extract Id from update item'))).toBe(true);

    expect(ManyToManyTarget.updateCalls).toEqual([{ id: 't2', values: { Name: 'updated-name' } }]);
    expect(ManyToManyTarget.createCalls).toEqual([{ Name: 'brand-new' }]);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor prepareForUpdate removes null collection from write payload without relation operations', async () => {
  const processor = new ManyToManyProcessor(prepareParentCtor);
  const result = await processor.prepareForUpdate({ Name: 'parent-updated', Tags: null });

  expect(result.processedValue).toEqual({ Name: 'parent-updated' });
  expect(result.relations.manyToManyRelations).toEqual([]);
  expect(Array.from(result.relations.touchedCollections || [])).toEqual([]);
});

test('many-to-many processor prepareForCreate strips m2m fields and records touched collection', async () => {
  const processor = new ManyToManyProcessor(prepareParentCtor);
  const result = await processor.prepareForCreate({
    Name: 'parent-created',
    Tags: [{ Id: 't1' }],
  });

  expect(result.processedValue).toEqual({ Name: 'parent-created' });
  expect(result.relations.manyToManyRelations.length).toBe(1);
  expect(result.relations.manyToManyRelations[0]?.fieldName).toBe('Tags');
  expect(Array.from(result.relations.touchedCollections || [])).toEqual(['Tags']);
});

test('many-to-many processor prepareForUpdate collects m2m relations when config is complete', async () => {
  const processor = new ManyToManyProcessor(prepareParentCtor);
  const result = await processor.prepareForUpdate({
    Name: 'parent-updated',
    Tags: [{ Id: 't1' }],
  });

  expect(result.processedValue).toEqual({ Name: 'parent-updated' });
  expect(result.relations.manyToManyRelations.length).toBe(1);
  expect(result.relations.manyToManyRelations[0]?.fieldName).toBe('Tags');
  expect(Array.from(result.relations.touchedCollections || [])).toEqual(['Tags']);
});

test('many-to-many processor prepareForUpdate ignores relation write when m2m config is incomplete', async () => {
  const processor = new ManyToManyProcessor(prepareParentCtor);
  const meta = MetadataStorage.instance.getModelMetadata(prepareParentCtor as any) as any;
  const tagsField = meta.fields.get('Tags') as any;
  const originalRelation = tagsField.relation;

  tagsField.relation = {
    joinModel: originalRelation.joinModel,
    targetModel: originalRelation.targetModel,
  };

  try {
    const result = await processor.prepareForUpdate({
      Name: 'parent-updated',
      Tags: [{ Id: 't1' }],
    });

    expect(result.processedValue).toEqual({ Name: 'parent-updated' });
    expect(result.relations.manyToManyRelations).toEqual([]);
    expect(Array.from(result.relations.touchedCollections || [])).toEqual([]);
  } finally {
    tagsField.relation = originalRelation;
  }
});

test('many-to-many processor prepareForUpdate ignores relation write when relation metadata is missing', async () => {
  const processor = new ManyToManyProcessor(prepareParentCtor);
  const meta = MetadataStorage.instance.getModelMetadata(prepareParentCtor as any) as any;
  const tagsField = meta.fields.get('Tags') as any;
  const originalRelation = tagsField.relation;

  tagsField.relation = undefined;

  try {
    const result = await processor.prepareForUpdate({
      Name: 'parent-updated',
      Tags: [{ Id: 't1' }],
    });

    expect(result.processedValue).toEqual({ Name: 'parent-updated' });
    expect(result.relations.manyToManyRelations).toEqual([]);
    expect(Array.from(result.relations.touchedCollections || [])).toEqual([]);
  } finally {
    tagsField.relation = originalRelation;
  }
});

test('many-to-many processor batch replace uses OR delete strategy for small diff set', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([
    { Id: 'j1', OwnerId: 'p1', TagId: 't1' },
    { Id: 'j2', OwnerId: 'p1', TagId: 't2' },
    { Id: 'j3', OwnerId: 'p2', TagId: 't3' },
    { Id: 'j4', OwnerId: 'p2', TagId: 't4' },
  ]);

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1', 'p2'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: [{ Id: 't2' }],
        },
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: [{ Id: 't4' }],
        },
      ]
    );

    expect(result.errors).toEqual([]);
    expect(result.success.map(item => item.entityId).sort()).toEqual(['t1', 't2', 't3', 't4']);
    expect(joinStore.deleteCalls.length).toBe(1);
    const firstDelete = joinStore.deleteCalls[0] as { Or?: unknown[] } | undefined;
    expect(Array.isArray(firstDelete?.Or)).toBe(true);
    expect(joinStore.rows.map(r => r.Id).sort()).toEqual(['j2', 'j4']);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor batch replace uses grouped in-delete strategy for large diff set', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const seedRows: JoinRow[] = [];
  for (let i = 1; i <= 101; i++) {
    seedRows.push({ Id: `j${i}`, OwnerId: 'p1', TagId: `t${i}` });
  }
  const joinStore = createJoinRepoMock(seedRows);

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: [],
        },
      ]
    );

    expect(result.errors).toEqual([]);
    expect(joinStore.deleteCalls.length).toBe(1);
    expect(joinStore.deleteCalls[0]).toEqual({
      And: [
        ['OwnerId', '=', 'p1'],
        ['TagId', 'in', seedRows.map(row => row.TagId)],
      ],
    });
    expect(joinStore.rows.length).toBe(0);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor batch delete reports id extraction errors while deleting valid ids', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([{ Id: 'j1', OwnerId: 'p1', TagId: 't1' }]);

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: {
            delete: [{ Id: 't1' }, {} as any],
          },
        },
      ]
    );

    expect(result.success.some(item => item.entityId === 't1')).toBe(true);
    expect(result.errors.some(error => String(error.error?.message || '').includes('Could not extract Id from delete item'))).toBe(true);
    expect(joinStore.deleteCalls.length).toBe(1);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor throws on mismatched operation type', async () => {
  const processor = new ManyToManyProcessor(parentCtor);
  let message = '';
  try {
    await processor.processRelationUpdate('p1', {
      type: 'OneToMany',
    } as unknown as Parameters<ManyToManyProcessor['processRelationUpdate']>[1]);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('Expected a ManyToMany operation')).toBe(true);
});

test('many-to-many processor batch validates parent-operation cardinality and operation type', async () => {
  const processor = new ManyToManyProcessor(parentCtor);

  let sizeMessage = '';
  try {
    await processor.batchProcessRelationUpdate(
      ['p1', 'p2'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: [],
        },
      ]
    );
  } catch (error) {
    sizeMessage = String((error as Error)?.message || error);
  }

  expect(sizeMessage.includes('Parent entity Id array length must match relation operation array length')).toBe(true);

  let typeMessage = '';
  try {
    await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'OneToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: [],
        } as unknown as Parameters<ManyToManyProcessor['batchProcessRelationUpdate']>[1][number],
      ]
    );
  } catch (error) {
    typeMessage = String((error as Error)?.message || error);
  }

  expect(typeMessage.includes('Expected a ManyToMany operation')).toBe(true);
});

test('many-to-many processor batch create skips already-related ids before creating new links', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([{ Id: 'j1', OwnerId: 'p1', TagId: 't1' }]);
  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: {
            create: [{ Id: 't1' }, { Name: 'fresh-tag' }],
          },
        } as unknown as Parameters<ManyToManyProcessor['batchProcessRelationUpdate']>[1][number],
      ]
    );

    expect(result.errors).toEqual([]);
    expect(result.success.some(item => item.entityId === 't1')).toBe(false);
    expect(ManyToManyTarget.createCalls).toEqual([{ Name: 'fresh-tag' }]);
    expect(joinStore.createCalls.length).toBe(1);
    expect(joinStore.createCalls[0]).toEqual([{ OwnerId: 'p1', TagId: 'NEW-TARGET-1' }]);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor wraps top-level process errors into relation result', async () => {
  resetTargetCalls();
  const originalGetModelMetadata = MetadataStorage.instance.getModelMetadata;

  MetadataStorage.instance.getModelMetadata = (() => {
    throw new Error('join metadata missing');
  }) as unknown as typeof MetadataStorage.instance.getModelMetadata;

  try {
    let message = '';
    try {
      const processor = new ManyToManyProcessor(parentCtor);
      await processor.processRelationUpdate('p1', {
        type: 'ManyToMany',
        fieldName: 'Tags',
        joinModel: joinCtor,
        targetModel: targetCtor,
        joinField: 'OwnerId',
        inverseJoinField: 'TagId',
        operations: [],
      });
    } catch (error) {
      message = String((error as Error)?.message || error);
    }

    expect(message.includes('join metadata missing')).toBe(true);
  } finally {
    MetadataStorage.instance.getModelMetadata = originalGetModelMetadata;
  }
});

test('many-to-many processor batch create records search failure as batch error', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([{ Id: 'j1', OwnerId: 'p1', TagId: 't1' }]);
  const originalSearch = Repository.prototype.search;
  Repository.prototype.search = async function (this: Repository, condition: unknown) {
    if (condition && typeof condition === 'object' && Array.isArray((condition as { And?: unknown[] }).And)) {
      const andItems = (condition as { And: unknown[] }).And;
      const first = andItems[0];
      if (Array.isArray(first) && first[0] === 'OwnerId') {
        throw new Error('join search failed');
      }
    }
    return originalSearch.call(this, condition as never);
  } as unknown as Repository['search'];

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: {
            create: [{ Id: 't1' }],
          },
        },
      ]
    );

    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.error?.message || '').includes('join search failed')).toBe(true);
  } finally {
    Repository.prototype.search = originalSearch;
    joinStore.restore();
  }
});

test('many-to-many processor create records join-table create failure as relation error', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([]);
  const originalCreate = Repository.prototype.create;
  Repository.prototype.create = async function () {
    throw new Error('join create failed');
  } as unknown as Repository['create'];

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'ManyToMany',
      fieldName: 'Tags',
      joinModel: joinCtor,
      targetModel: targetCtor,
      joinField: 'OwnerId',
      inverseJoinField: 'TagId',
      operations: {
        create: [{ Id: 't1' }],
      },
    } as unknown as Parameters<ManyToManyProcessor['processRelationUpdate']>[1]);

    expect(result.entityIds).toEqual(['t1']);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.message || '').includes('join create failed')).toBe(true);
  } finally {
    Repository.prototype.create = originalCreate;
    joinStore.restore();
  }
});

test('many-to-many processor create records invalid target create id as relation error', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([]);
  const originalCreate = ManyToManyTarget.Create;
  ManyToManyTarget.Create = (async () => ({ Name: 'no-id' }) as unknown as ManyToManyTarget) as typeof ManyToManyTarget.Create;

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'ManyToMany',
      fieldName: 'Tags',
      joinModel: joinCtor,
      targetModel: targetCtor,
      joinField: 'OwnerId',
      inverseJoinField: 'TagId',
      operations: {
        create: [{ Name: 'bad-created-id' }],
      },
    } as unknown as Parameters<ManyToManyProcessor['processRelationUpdate']>[1]);

    expect(result.entityIds).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.message || '').includes('did not return a valid Id')).toBe(true);
  } finally {
    ManyToManyTarget.Create = originalCreate;
    joinStore.restore();
  }
});

test('many-to-many processor update records target update exception as per-item error', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([{ Id: 'j1', OwnerId: 'p1', TagId: 't1' }]);
  const originalUpdateById = ManyToManyTarget.UpdateById;
  ManyToManyTarget.UpdateById = (async () => {
    throw new Error('target update failed');
  }) as typeof ManyToManyTarget.UpdateById;

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'ManyToMany',
      fieldName: 'Tags',
      joinModel: joinCtor,
      targetModel: targetCtor,
      joinField: 'OwnerId',
      inverseJoinField: 'TagId',
      operations: {
        update: [{ Id: 't1', Name: 'x' }],
      },
    } as unknown as Parameters<ManyToManyProcessor['processRelationUpdate']>[1]);

    expect(result.entityIds).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.message || '').includes('target update failed')).toBe(true);
  } finally {
    ManyToManyTarget.UpdateById = originalUpdateById;
    joinStore.restore();
  }
});

test('many-to-many processor array replace with empty payload removes all join rows', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([
    { Id: 'j1', OwnerId: 'p1', TagId: 't1' },
    { Id: 'j2', OwnerId: 'p1', TagId: 't2' },
  ]);

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'ManyToMany',
      fieldName: 'Tags',
      joinModel: joinCtor,
      targetModel: targetCtor,
      joinField: 'OwnerId',
      inverseJoinField: 'TagId',
      operations: [] as unknown as Parameters<ManyToManyProcessor['processRelationUpdate']>[1]['operations'],
    });

    expect(result.errors).toEqual([]);
    expect(result.entityIds).toEqual(['t1', 't2']);
    expect(joinStore.rows).toEqual([]);
    expect(joinStore.deleteCalls.length).toBe(1);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor replace tolerates duplicate relation ids and keeps deterministic order', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([
    { Id: 'j1', OwnerId: 'p1', TagId: 't1' },
    { Id: 'j2', OwnerId: 'p1', TagId: 't2' },
  ]);

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'ManyToMany',
      fieldName: 'Tags',
      joinModel: joinCtor,
      targetModel: targetCtor,
      joinField: 'OwnerId',
      inverseJoinField: 'TagId',
      operations: [{ Id: 't2' }, { Id: 't2' }, { Name: 'fresh' }] as unknown as Parameters<ManyToManyProcessor['processRelationUpdate']>[1]['operations'],
    });

    expect(result.errors).toEqual([]);
    expect(result.entityIds).toEqual(['NEW-TARGET-1', 't1', 't2']);
    expect(joinStore.createCalls.length).toBe(1);
    expect(joinStore.createCalls[0]).toEqual([{ OwnerId: 'p1', TagId: 'NEW-TARGET-1' }]);
    expect(joinStore.rows.map(row => row.TagId).sort()).toEqual(['NEW-TARGET-1', 't2']);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor delete keeps success ids when some join rows are missing', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([{ Id: 'j1', OwnerId: 'p1', TagId: 't1' }]);

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'ManyToMany',
      fieldName: 'Tags',
      joinModel: joinCtor,
      targetModel: targetCtor,
      joinField: 'OwnerId',
      inverseJoinField: 'TagId',
      operations: {
        delete: [{ Id: 't1' }, { Id: 't-missing' }],
      },
    } as unknown as Parameters<ManyToManyProcessor['processRelationUpdate']>[1]);

    expect(result.errors).toEqual([]);
    expect(result.entityIds).toEqual(['t1', 't-missing']);
    expect(joinStore.rows).toEqual([]);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor create accepts string ids and links them directly without target create', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([]);
  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'ManyToMany',
      fieldName: 'Tags',
      joinModel: joinCtor,
      targetModel: targetCtor,
      joinField: 'OwnerId',
      inverseJoinField: 'TagId',
      operations: {
        create: ['t1', 't2'] as unknown as Array<{ Id: string }>,
      },
    } as unknown as Parameters<ManyToManyProcessor['processRelationUpdate']>[1]);

    expect(result.errors).toEqual([]);
    expect(result.entityIds).toEqual(['t1', 't2']);
    expect(ManyToManyTarget.createCalls).toEqual([]);
    expect(joinStore.createCalls).toEqual([
      [
        { OwnerId: 'p1', TagId: 't1' },
        { OwnerId: 'p1', TagId: 't2' },
      ],
    ]);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor replace uses id-in delete when only a small subset is removed', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([
    { Id: 'j1', OwnerId: 'p1', TagId: 't1' },
    { Id: 'j2', OwnerId: 'p1', TagId: 't2' },
    { Id: 'j3', OwnerId: 'p1', TagId: 't3' },
  ]);

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'ManyToMany',
      fieldName: 'Tags',
      joinModel: joinCtor,
      targetModel: targetCtor,
      joinField: 'OwnerId',
      inverseJoinField: 'TagId',
      operations: [{ Id: 't1' }, { Id: 't2' }] as unknown as Parameters<ManyToManyProcessor['processRelationUpdate']>[1]['operations'],
    });

    expect(result.errors).toEqual([]);
    expect(result.entityIds.sort()).toEqual(['t1', 't2', 't3']);
    expect(joinStore.deleteCalls.length).toBe(1);
    expect(joinStore.deleteCalls[0]).toEqual(['Id', 'in', ['j3']]);
    expect(joinStore.rows.map(row => row.TagId).sort()).toEqual(['t1', 't2']);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor batch replace with one removed pair uses single-and delete shape', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([
    { Id: 'j1', OwnerId: 'p1', TagId: 't1' },
    { Id: 'j2', OwnerId: 'p1', TagId: 't2' },
  ]);

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: [{ Id: 't1' }],
        },
      ]
    );

    expect(result.errors).toEqual([]);
    expect(result.success.map(item => item.entityId).sort()).toEqual(['t1', 't2']);
    expect(joinStore.deleteCalls.length).toBe(1);
    expect(joinStore.deleteCalls[0]).toEqual({
      And: [
        ['OwnerId', '=', 'p1'],
        ['TagId', '=', 't2'],
      ],
    });
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor processRelationUpdate wraps top-level repository factory failures into relation result', async () => {
  resetTargetCalls();

  const originalGetRepository = RepositoryFactory.getRepository;
  RepositoryFactory.getRepository = (() => {
    throw new Error('target repository unavailable');
  }) as unknown as typeof RepositoryFactory.getRepository;

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'ManyToMany',
      fieldName: 'Tags',
      joinModel: joinCtor,
      targetModel: targetCtor,
      joinField: 'OwnerId',
      inverseJoinField: 'TagId',
      operations: [],
    });

    expect(result.affectedCount).toBe(0);
    expect(result.entityIds).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.message || '').includes('target repository unavailable')).toBe(true);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('many-to-many processor batch delete uses single-id path and records delete failure as batch error', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([{ Id: 'j1', OwnerId: 'p1', TagId: 't1' }]);
  const originalDelete = Repository.prototype.delete;
  Repository.prototype.delete = async function (this: Repository, condition: unknown) {
    if (condition && typeof condition === 'object' && Array.isArray((condition as { And?: unknown[] }).And)) {
      const andItems = (condition as { And: unknown[] }).And;
      const second = andItems[1];
      if (Array.isArray(second) && second[0] === 'TagId' && second[1] === '=' && second[2] === 't1') {
        throw new Error('single delete failed');
      }
    }
    return originalDelete.call(this, condition as never);
  } as unknown as Repository['delete'];

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: {
            delete: [{ Id: 't1' }],
          },
        },
      ]
    );

    expect(result.success).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.error?.message || '').includes('single delete failed')).toBe(true);
  } finally {
    Repository.prototype.delete = originalDelete;
    joinStore.restore();
  }
});

test('many-to-many processor batch update covers missing id, missing relation and successful update branches', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([{ Id: 'j1', OwnerId: 'p1', TagId: 't1' }]);

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: {
            update: [{ Name: 'missing-id' }, { Id: 't-missing', Name: 'x' }, { Id: 't1', Name: 'ok' }] as unknown as Array<{
              Id: string;
              Name: string;
            }>,
          },
        },
      ]
    );

    expect(result.errors.length).toBe(2);
    expect(result.errors.some(item => String(item.error?.message || '').includes('Could not extract Id from update item'))).toBe(true);
    expect(result.errors.some(item => String(item.error?.message || '').includes('Attempted to update a non-related entity'))).toBe(true);
    expect(result.success.some(item => item.entityId === 't1')).toBe(true);
    expect(ManyToManyTarget.updateCalls).toEqual([{ id: 't1', values: { Name: 'ok' } }]);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor prepareForUpdate skips fields not present in value even when listed in changedFields', async () => {
  const processor = new ManyToManyProcessor(prepareParentCtor);
  const result = await processor.prepareForUpdate({ Name: 'p1' }, ['Tags']);

  expect(result.processedValue).toEqual({ Name: 'p1' });
  expect(result.relations.manyToManyRelations).toEqual([]);
  expect(Array.from(result.relations.touchedCollections || [])).toEqual([]);
});

test('many-to-many processor processRelationUpdate object replace path keeps only existing ids when no new items', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([
    { Id: 'j1', OwnerId: 'p1', TagId: 't1' },
    { Id: 'j2', OwnerId: 'p1', TagId: 't2' },
  ]);

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'ManyToMany',
      fieldName: 'Tags',
      joinModel: joinCtor,
      targetModel: targetCtor,
      joinField: 'OwnerId',
      inverseJoinField: 'TagId',
      operations: {
        replace: [{ Id: 't1' }, { Id: 't2' }],
      },
    } as unknown as Parameters<ManyToManyProcessor['processRelationUpdate']>[1]);

    expect(result.errors).toEqual([]);
    expect(result.entityIds.sort()).toEqual(['t1', 't2']);
    expect(joinStore.createCalls).toEqual([]);
    expect(joinStore.rows.map(row => row.TagId).sort()).toEqual(['t1', 't2']);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor processRelationUpdate wraps non-error throws from metadata lookup', async () => {
  resetTargetCalls();
  const processor = new ManyToManyProcessor(parentCtor);
  const originalGetModelMetadata = MetadataStorage.instance.getModelMetadata;

  MetadataStorage.instance.getModelMetadata = (() => {
    throw 'metadata exploded';
  }) as unknown as typeof MetadataStorage.instance.getModelMetadata;

  try {
    const result = await processor.processRelationUpdate('p1', {
      type: 'ManyToMany',
      fieldName: 'Tags',
      joinModel: joinCtor,
      targetModel: targetCtor,
      joinField: 'OwnerId',
      inverseJoinField: 'TagId',
      operations: [],
    });

    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.message || '')).toContain('metadata exploded');
  } finally {
    MetadataStorage.instance.getModelMetadata = originalGetModelMetadata;
  }
});

test('many-to-many processor batch delete single id success path and non-error failure path are both handled', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([
    { Id: 'j1', OwnerId: 'p1', TagId: 't1' },
    { Id: 'j2', OwnerId: 'p2', TagId: 't2' },
  ]);
  const originalDelete = Repository.prototype.delete;
  Repository.prototype.delete = async function (this: Repository, condition: unknown) {
    if (condition && typeof condition === 'object' && Array.isArray((condition as { And?: unknown[] }).And)) {
      const andItems = (condition as { And: unknown[] }).And;
      const first = andItems[0];
      if (Array.isArray(first) && first[2] === 'p2') {
        throw 'delete string error';
      }
    }
    return originalDelete.call(this, condition as never);
  } as unknown as Repository['delete'];

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1', 'p2'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: { delete: [{ Id: 't1' }] },
        },
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: { delete: [{ Id: 't2' }] },
        },
      ]
    );

    expect(result.success.some(item => item.entityId === 't1')).toBe(true);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.error?.message || '')).toContain('delete string error');
  } finally {
    Repository.prototype.delete = originalDelete;
    joinStore.restore();
  }
});

test('many-to-many processor batch create single-id search branch and empty batch path are covered', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([]);
  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: {
            create: [{ Id: 't1' }],
          },
        },
      ]
    );

    expect(result.errors).toEqual([]);
    expect(result.success.some(item => item.entityId === 't1')).toBe(true);

    const empty = await processor.batchProcessRelationUpdate([], []);
    expect(empty.summary.totalOperations).toBe(0);
    expect(empty.success).toEqual([]);
    expect(empty.errors).toEqual([]);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor batchProcessRelationUpdate wraps non-error top-level throws and uses unknown target fallback', async () => {
  const processor = new ManyToManyProcessor(parentCtor);
  const originalGetRepository = RepositoryFactory.getRepository;

  RepositoryFactory.getRepository = (() => {
    throw 'batch repo boom';
  }) as unknown as typeof RepositoryFactory.getRepository;

  try {
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: [],
        },
      ]
    );

    expect(result.success).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.error?.message || '')).toContain('batch repo boom');
    expect(result.errors[0]?.targetModel).toBe('ManyToManyTarget');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('many-to-many processor helper methods cover empty and fallback branches', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);
  const processor = new ManyToManyProcessor(parentCtor) as unknown as {
    batchRemoveAssociationsWithDiff: (
      joinRepo: Repository,
      joinField: string,
      inverseJoinField: string,
      parentIds: string[],
      replacementMap: Map<string, any[]>
    ) => Promise<Map<string, { existingIds: Set<string>; removedIds: string[] }>>;
    processReplaceItems: (
      joinRepo: Repository,
      targetRepo: unknown,
      targetModel: typeof targetCtor,
      parentId: string,
      joinField: string,
      inverseJoinField: string,
      items: any[]
    ) => Promise<{ successIds: string[]; errors: Error[] }>;
    removeExistingRelationsWithDiff: (
      joinRepo: Repository,
      joinField: string,
      inverseJoinField: string,
      parentId: string,
      newItems: any[]
    ) => Promise<{ existingRecords: any[]; existingTargetIds: string[]; removedRecords: any[] }>;
  };

  const emptyJoinStore = createJoinRepoMock([]);
  try {
    const emptyMeta = MetadataStorage.instance.getModelMetadata(joinCtor as any);
    const emptyJoinRepo = new Repository(emptyMeta);
    const emptyDiff = await processor.batchRemoveAssociationsWithDiff(emptyJoinRepo, 'OwnerId', 'TagId', [], new Map());
    expect(Array.from(emptyDiff.entries())).toEqual([]);

    const removeNoExisting = await processor.removeExistingRelationsWithDiff(emptyJoinRepo, 'OwnerId', 'TagId', 'p-empty', [{ Id: 't1' }]);
    expect(removeNoExisting.existingTargetIds).toEqual([]);
    expect(removeNoExisting.removedRecords).toEqual([]);
  } finally {
    emptyJoinStore.restore();
  }

  const joinStore = createJoinRepoMock([{ Id: 'j1', OwnerId: 'p1', TagId: 't1' }]);
  const originalCreate = Repository.prototype.create;
  const originalTargetCreate = ManyToManyTarget.Create;
  let createCallCount = 0;

  Repository.prototype.create = async function (this: Repository, values: Record<string, any>[]) {
    createCallCount += 1;
    if (createCallCount === 1) {
      throw 'join create non-error';
    }
    return originalCreate.call(this, values as never);
  } as unknown as Repository['create'];

  ManyToManyTarget.Create = (async () => {
    throw 'target create non-error';
  }) as typeof ManyToManyTarget.Create;

  try {
    const meta = MetadataStorage.instance.getModelMetadata(joinCtor as any);
    const joinRepo = new Repository(meta);

    const first = await processor.processReplaceItems(joinRepo, {}, targetCtor, 'p1', 'OwnerId', 'TagId', ['t1']);
    expect(first.successIds).toEqual(['t1']);
    expect(first.errors.length).toBe(1);
    expect(String(first.errors[0]?.message || '')).toContain('join create non-error');

    const second = await processor.processReplaceItems(joinRepo, {}, targetCtor, 'p1', 'OwnerId', 'TagId', [{ Id: 't1' }, { Name: 'x' }]);
    expect(second.successIds).toEqual(['t1']);
    expect(second.errors.length).toBe(1);
    expect(String(second.errors[0]?.message || '')).toContain('target create non-error');

    const third = await processor.processReplaceItems(joinRepo, {}, targetCtor, 'p1', 'OwnerId', 'TagId', [{ Name: 'will-fail-id' }]);
    expect(third.successIds).toEqual([]);
    expect(third.errors.length).toBe(1);
    expect(String(third.errors[0]?.message || '')).toContain('target create non-error');

    const groupedFallback = await processor.batchRemoveAssociationsWithDiff(joinRepo, 'OwnerId', 'TagId', ['p1', 'p2'], new Map([['p1', []]]));
    expect(groupedFallback.get('p1')?.removedIds).toEqual(['t1']);
    expect(groupedFallback.get('p2')?.removedIds).toEqual([]);
  } finally {
    Repository.prototype.create = originalCreate;
    ManyToManyTarget.Create = originalTargetCreate;
    joinStore.restore();
  }
});

test('many-to-many processor prepareForUpdate handles undefined field value without collecting relation', async () => {
  const processor = new ManyToManyProcessor(prepareParentCtor);
  const result = await processor.prepareForUpdate({ Name: 'p1', Tags: undefined }, ['Tags']);

  expect(result.processedValue).toEqual({ Name: 'p1' });
  expect(result.relations.manyToManyRelations).toEqual([]);
  expect(Array.from(result.relations.touchedCollections || [])).toEqual([]);
});

test('many-to-many processor processRelationUpdate object replace creates missing links when replace has new items', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([{ Id: 'j1', OwnerId: 'p1', TagId: 't1' }]);
  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'ManyToMany',
      fieldName: 'Tags',
      joinModel: joinCtor,
      targetModel: targetCtor,
      joinField: 'OwnerId',
      inverseJoinField: 'TagId',
      operations: {
        replace: [{ Id: 't1' }, { Name: 'fresh-replace' }],
      },
    } as unknown as Parameters<ManyToManyProcessor['processRelationUpdate']>[1]);

    expect(result.errors).toEqual([]);
    expect(result.entityIds.includes('t1')).toBe(true);
    expect(result.entityIds.includes('NEW-TARGET-1')).toBe(true);
    expect(joinStore.createCalls.length).toBe(1);
    expect(joinStore.createCalls[0]).toEqual([{ OwnerId: 'p1', TagId: 'NEW-TARGET-1' }]);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor processRelationUpdate create wraps non-error item and join-create failures', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([]);
  const originalCreate = Repository.prototype.create;
  const originalTargetCreate = ManyToManyTarget.Create;

  ManyToManyTarget.Create = (async () => '' as any) as typeof ManyToManyTarget.Create;
  Repository.prototype.create = async function () {
    throw 'join create string failure';
  } as unknown as Repository['create'];

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'ManyToMany',
      fieldName: 'Tags',
      joinModel: joinCtor,
      targetModel: targetCtor,
      joinField: 'OwnerId',
      inverseJoinField: 'TagId',
      operations: {
        create: [{ Name: 'x' }, 't1'],
      },
    } as unknown as Parameters<ManyToManyProcessor['processRelationUpdate']>[1]);

    expect(result.entityIds).toEqual(['t1']);
    expect(result.errors.length).toBe(2);
    expect(result.errors.some(error => String(error.message || '').includes('Failed to create target entity'))).toBe(true);
    expect(result.errors.some(error => String(error.message || '').includes('join create string failure'))).toBe(true);
  } finally {
    ManyToManyTarget.Create = originalTargetCreate;
    Repository.prototype.create = originalCreate;
    joinStore.restore();
  }
});

test('many-to-many processor processRelationUpdate delete and update wrap non-error exceptions', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([{ Id: 'j1', OwnerId: 'p1', TagId: 't1' }]);
  const originalDelete = Repository.prototype.delete;
  const originalUpdateById = ManyToManyTarget.UpdateById;

  Repository.prototype.delete = async function () {
    throw 'delete string failure';
  } as unknown as Repository['delete'];
  ManyToManyTarget.UpdateById = (async () => {
    throw 'update string failure';
  }) as typeof ManyToManyTarget.UpdateById;

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'ManyToMany',
      fieldName: 'Tags',
      joinModel: joinCtor,
      targetModel: targetCtor,
      joinField: 'OwnerId',
      inverseJoinField: 'TagId',
      operations: {
        delete: [{ Id: 't1' }],
        update: [{ Id: 't1', Name: 'next' }],
      },
    } as unknown as Parameters<ManyToManyProcessor['processRelationUpdate']>[1]);

    expect(result.errors.length).toBe(2);
    expect(result.errors.some(error => String(error.message || '').includes('delete string failure'))).toBe(true);
    expect(result.errors.some(error => String(error.message || '').includes('update string failure'))).toBe(true);
  } finally {
    Repository.prototype.delete = originalDelete;
    ManyToManyTarget.UpdateById = originalUpdateById;
    joinStore.restore();
  }
});

test('many-to-many processor batch replace create path and fallback defaults are covered', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([]);
  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: {
            replace: [{ Name: 'fresh-batch-replace' }] as unknown as Parameters<
              ManyToManyProcessor['batchProcessRelationUpdate']
            >[1][number]['operations'] extends {
              replace: infer T;
            }
              ? T
              : never,
          },
        },
      ]
    );

    expect(result.errors).toEqual([]);
    expect(result.success.some(item => item.entityId === 'NEW-TARGET-1')).toBe(true);
  } finally {
    joinStore.restore();
  }

  const processorForFallback = new ManyToManyProcessor(parentCtor);
  const originalGetRepository = RepositoryFactory.getRepository;
  RepositoryFactory.getRepository = (() => {
    throw 'grouping exploded';
  }) as unknown as typeof RepositoryFactory.getRepository;

  try {
    const fallback = await processorForFallback.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: {} as unknown as typeof targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: {},
        } as unknown as Parameters<ManyToManyProcessor['batchProcessRelationUpdate']>[1][number],
      ]
    );

    expect(fallback.errors.length).toBe(1);
    expect(String(fallback.errors[0]?.error?.message || '')).toContain('grouping exploded');
    expect(fallback.errors[0]?.targetModel).toBe('Unknown target');
    expect(fallback.errors[0]?.joinModel).toBe('ManyToManyJoin');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('many-to-many processor batch create and update wrap non-error failures', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const processor = new ManyToManyProcessor(parentCtor);
  const joinStore = createJoinRepoMock([{ Id: 'j1', OwnerId: 'p1', TagId: 't1' }]);
  const originalCreate = Repository.prototype.create;
  const originalUpdateById = ManyToManyTarget.UpdateById;

  Repository.prototype.create = async function () {
    throw 'batch join create string failure';
  } as unknown as Repository['create'];
  ManyToManyTarget.UpdateById = (async () => {
    throw 'batch update string failure';
  }) as typeof ManyToManyTarget.UpdateById;

  try {
    const createResult = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: {
            create: [{ Id: 't2' }, { Id: 't3' }],
          },
        },
      ]
    );

    expect(createResult.errors.length).toBe(1);
    expect(String(createResult.errors[0]?.error?.message || '')).toContain('batch join create string failure');

    const updateResult = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: {
            update: [{ Id: 't1', Name: 'u1' }] as unknown as Parameters<ManyToManyProcessor['batchProcessRelationUpdate']>[1][number]['operations'] extends {
              update: infer T;
            }
              ? T
              : never,
          },
        },
      ]
    );

    expect(updateResult.errors.length).toBe(1);
    expect(String(updateResult.errors[0]?.error?.message || '')).toContain('batch update string failure');
  } finally {
    Repository.prototype.create = originalCreate;
    ManyToManyTarget.UpdateById = originalUpdateById;
    joinStore.restore();
  }
});

test('many-to-many processor private processReplaceItems covers empty-id and error-object join-create branches', async () => {
  resetTargetCalls();
  const processor = new ManyToManyProcessor(parentCtor) as unknown as {
    processReplaceItems: (
      joinRepo: Repository,
      targetRepo: unknown,
      targetModel: typeof targetCtor,
      parentId: string,
      joinField: string,
      inverseJoinField: string,
      items: any[]
    ) => Promise<{ successIds: string[]; errors: Error[] }>;
  };
  const joinStore = createJoinRepoMock([]);
  const originalCreate = Repository.prototype.create;
  const originalTargetCreate = ManyToManyTarget.Create;

  ManyToManyTarget.Create = (async () => '' as any) as typeof ManyToManyTarget.Create;
  Repository.prototype.create = async function () {
    throw new Error('batch join create error object');
  } as unknown as Repository['create'];

  try {
    const meta = MetadataStorage.instance.getModelMetadata(joinCtor as any);
    const joinRepo = new Repository(meta);
    const result = await processor.processReplaceItems(joinRepo, {}, targetCtor, 'p1', 'OwnerId', 'TagId', [{ Name: 'x' }, 't1']);

    expect(result.successIds).toEqual(['t1']);
    expect(result.errors.length).toBe(2);
    expect(result.errors.some(error => String(error.message || '').includes('Failed to create target entity'))).toBe(true);
    expect(result.errors.some(error => String(error.message || '').includes('batch join create error object'))).toBe(true);
  } finally {
    ManyToManyTarget.Create = originalTargetCreate;
    Repository.prototype.create = originalCreate;
    joinStore.restore();
  }
});

test('many-to-many processor prepareForUpdate skips relation when inverseJoinField is missing', async () => {
  const processor = new ManyToManyProcessor(prepareParentCtor);
  const meta = MetadataStorage.instance.getModelMetadata(prepareParentCtor as any) as any;
  const tagsField = meta.fields.get('Tags') as any;
  const originalRelation = tagsField.relation;

  tagsField.relation = {
    joinModel: originalRelation.joinModel,
    targetModel: originalRelation.targetModel,
    joinField: originalRelation.joinField,
  };

  try {
    const result = await processor.prepareForUpdate({ Name: 'parent-updated', Tags: [{ Id: 't1' }] });
    expect(result.processedValue).toEqual({ Name: 'parent-updated' });
    expect(result.relations.manyToManyRelations).toEqual([]);
  } finally {
    tagsField.relation = originalRelation;
  }
});

test('many-to-many processor processRelationUpdate delete keeps Error instance branch', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([{ Id: 'j1', OwnerId: 'p1', TagId: 't1' }]);
  const originalDelete = Repository.prototype.delete;
  Repository.prototype.delete = async function () {
    throw new Error('delete error object');
  } as unknown as Repository['delete'];

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'ManyToMany',
      fieldName: 'Tags',
      joinModel: joinCtor,
      targetModel: targetCtor,
      joinField: 'OwnerId',
      inverseJoinField: 'TagId',
      operations: { delete: [{ Id: 't1' }] },
    } as unknown as Parameters<ManyToManyProcessor['processRelationUpdate']>[1]);

    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.message || '')).toBe('delete error object');
  } finally {
    Repository.prototype.delete = originalDelete;
    joinStore.restore();
  }
});

test('many-to-many processor batch replace uses empty diff fallback set when parent key is missing', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const processor = new ManyToManyProcessor(parentCtor) as unknown as {
    batchRemoveAssociationsWithDiff: (...args: any[]) => Promise<Map<string, { existingIds: Set<string>; removedIds: string[] }>>;
    batchProcessRelationUpdate: ManyToManyProcessor['batchProcessRelationUpdate'];
  };
  const originalBatchDiff = processor.batchRemoveAssociationsWithDiff;
  processor.batchRemoveAssociationsWithDiff = (async () => new Map()) as any;

  try {
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: [],
        },
      ]
    );

    expect(result.errors).toEqual([]);
    expect(result.success).toEqual([]);
  } finally {
    processor.batchRemoveAssociationsWithDiff = originalBatchDiff;
  }
});

test('many-to-many processor batch delete uses in-branch for multiple ids', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([
    { Id: 'j1', OwnerId: 'p1', TagId: 't1' },
    { Id: 'j2', OwnerId: 'p1', TagId: 't2' },
  ]);

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: { delete: [{ Id: 't1' }, { Id: 't2' }] },
        },
      ]
    );

    expect(result.errors).toEqual([]);
    expect(result.success.map(item => item.entityId).sort()).toEqual(['t1', 't2']);
    expect(joinStore.deleteCalls[0]).toEqual({
      And: [
        ['OwnerId', '=', 'p1'],
        ['TagId', 'in', ['t1', 't2']],
      ],
    });
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor batch update keeps Error instance branch when update helper throws Error', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const joinStore = createJoinRepoMock([{ Id: 'j1', OwnerId: 'p1', TagId: 't1' }]);
  const originalUpdateById = ManyToManyTarget.UpdateById;
  ManyToManyTarget.UpdateById = (async () => {
    throw new Error('batch update error object');
  }) as typeof ManyToManyTarget.UpdateById;

  try {
    const processor = new ManyToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: joinCtor,
          targetModel: targetCtor,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: { update: [{ Id: 't1', Name: 'next' } as unknown as { Id: string }] },
        },
      ]
    );

    expect(result.success).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.error?.message || '')).toBe('batch update error object');
  } finally {
    ManyToManyTarget.UpdateById = originalUpdateById;
    joinStore.restore();
  }
});

test('many-to-many processor helper removeExistingRelationsWithDiff clears all links when replacement list is empty', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const processor = new ManyToManyProcessor(parentCtor) as unknown as {
    removeExistingRelationsWithDiff: (
      joinRepo: Repository,
      joinField: string,
      inverseJoinField: string,
      parentId: string,
      newItems: any[]
    ) => Promise<{ existingRecords: any[]; existingTargetIds: string[]; removedRecords: any[] }>;
  };

  const joinStore = createJoinRepoMock([
    { Id: 'j1', OwnerId: 'p1', TagId: 't1' },
    { Id: 'j2', OwnerId: 'p1', TagId: 't2' },
  ]);

  try {
    const meta = MetadataStorage.instance.getModelMetadata(joinCtor as any);
    const joinRepo = new Repository(meta);
    const result = await processor.removeExistingRelationsWithDiff(joinRepo, 'OwnerId', 'TagId', 'p1', []);

    expect(result.existingTargetIds.sort()).toEqual(['t1', 't2']);
    expect(result.removedRecords.map(item => item.TagId).sort()).toEqual(['t1', 't2']);
    expect(joinStore.deleteCalls).toEqual([['OwnerId', '=', 'p1']]);
    expect(joinStore.rows).toEqual([]);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor helper batchRemoveAssociationsWithDiff uses Or delete shape for small multi-pair removals', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const processor = new ManyToManyProcessor(parentCtor) as unknown as {
    batchRemoveAssociationsWithDiff: (
      joinRepo: Repository,
      joinField: string,
      inverseJoinField: string,
      parentIds: string[],
      replacementMap: Map<string, any[]>
    ) => Promise<Map<string, { existingIds: Set<string>; removedIds: string[] }>>;
  };

  const joinStore = createJoinRepoMock([
    { Id: 'j1', OwnerId: 'p1', TagId: 't1' },
    { Id: 'j2', OwnerId: 'p1', TagId: 't2' },
    { Id: 'j3', OwnerId: 'p1', TagId: 't3' },
  ]);

  try {
    const meta = MetadataStorage.instance.getModelMetadata(joinCtor as any);
    const joinRepo = new Repository(meta);
    const result = await processor.batchRemoveAssociationsWithDiff(joinRepo, 'OwnerId', 'TagId', ['p1'], new Map([['p1', [{ Id: 't1' }]]]));

    expect(result.get('p1')?.removedIds.sort()).toEqual(['t2', 't3']);
    expect(joinStore.deleteCalls.length).toBe(1);
    expect(joinStore.deleteCalls[0]).toEqual({
      Or: [
        {
          And: [
            ['OwnerId', '=', 'p1'],
            ['TagId', '=', 't2'],
          ],
        },
        {
          And: [
            ['OwnerId', '=', 'p1'],
            ['TagId', '=', 't3'],
          ],
        },
      ],
    });
    expect(joinStore.rows.map(item => item.TagId)).toEqual(['t1']);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor helper removeExistingRelationsWithDiff uses Id in-delete for sparse removals', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const processor = new ManyToManyProcessor(parentCtor) as unknown as {
    removeExistingRelationsWithDiff: (
      joinRepo: Repository,
      joinField: string,
      inverseJoinField: string,
      parentId: string,
      newItems: any[]
    ) => Promise<{ existingRecords: any[]; existingTargetIds: string[]; removedRecords: any[] }>;
  };

  const joinStore = createJoinRepoMock([
    { Id: 'j1', OwnerId: 'p1', TagId: 't1' },
    { Id: 'j2', OwnerId: 'p1', TagId: 't2' },
    { Id: 'j3', OwnerId: 'p1', TagId: 't3' },
    { Id: 'j4', OwnerId: 'p1', TagId: 't4' },
  ]);

  try {
    const meta = MetadataStorage.instance.getModelMetadata(joinCtor as any);
    const joinRepo = new Repository(meta);
    const result = await processor.removeExistingRelationsWithDiff(joinRepo, 'OwnerId', 'TagId', 'p1', [{ Id: 't1' }, { Id: 't2' }, { Id: 't3' }]);

    expect(result.existingTargetIds.sort()).toEqual(['t1', 't2', 't3', 't4']);
    expect(result.removedRecords.map(item => item.TagId)).toEqual(['t4']);
    expect(joinStore.deleteCalls.length).toBe(1);
    expect(joinStore.deleteCalls[0]).toEqual(['Id', 'in', ['j4']]);
    expect(joinStore.rows.map(item => item.TagId).sort()).toEqual(['t1', 't2', 't3']);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor helper removeExistingRelationsWithDiff uses relation in-delete for dense removals', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const processor = new ManyToManyProcessor(parentCtor) as unknown as {
    removeExistingRelationsWithDiff: (
      joinRepo: Repository,
      joinField: string,
      inverseJoinField: string,
      parentId: string,
      newItems: any[]
    ) => Promise<{ existingRecords: any[]; existingTargetIds: string[]; removedRecords: any[] }>;
  };

  const joinStore = createJoinRepoMock([
    { Id: 'j1', OwnerId: 'p1', TagId: 't1' },
    { Id: 'j2', OwnerId: 'p1', TagId: 't2' },
    { Id: 'j3', OwnerId: 'p1', TagId: 't3' },
    { Id: 'j4', OwnerId: 'p1', TagId: 't4' },
  ]);

  try {
    const meta = MetadataStorage.instance.getModelMetadata(joinCtor as any);
    const joinRepo = new Repository(meta);
    const result = await processor.removeExistingRelationsWithDiff(joinRepo, 'OwnerId', 'TagId', 'p1', [{ Id: 't1' }]);

    expect(result.existingTargetIds.sort()).toEqual(['t1', 't2', 't3', 't4']);
    expect(result.removedRecords.map(item => item.TagId).sort()).toEqual(['t2', 't3', 't4']);
    expect(joinStore.deleteCalls.length).toBe(1);
    expect(joinStore.deleteCalls[0]).toEqual({
      And: [
        ['OwnerId', '=', 'p1'],
        ['TagId', 'in', ['t2', 't3', 't4']],
      ],
    });
    expect(joinStore.rows.map(item => item.TagId)).toEqual(['t1']);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor helper batchRemoveAssociationsWithDiff uses grouped in-delete for large multi-parent removals', async () => {
  resetTargetCalls();
  RepositoryFactory.setRepository(targetCtor, { search: async () => [] } as unknown as Repository);

  const processor = new ManyToManyProcessor(parentCtor) as unknown as {
    batchRemoveAssociationsWithDiff: (
      joinRepo: Repository,
      joinField: string,
      inverseJoinField: string,
      parentIds: string[],
      replacementMap: Map<string, any[]>
    ) => Promise<Map<string, { existingIds: Set<string>; removedIds: string[] }>>;
  };

  const seedRows: JoinRow[] = [];
  for (let i = 1; i <= 60; i++) {
    seedRows.push({ Id: `p1-${i}`, OwnerId: 'p1', TagId: `p1-t${i}` });
    seedRows.push({ Id: `p2-${i}`, OwnerId: 'p2', TagId: `p2-t${i}` });
  }
  const joinStore = createJoinRepoMock(seedRows);

  try {
    const meta = MetadataStorage.instance.getModelMetadata(joinCtor as any);
    const joinRepo = new Repository(meta);
    const result = await processor.batchRemoveAssociationsWithDiff(
      joinRepo,
      'OwnerId',
      'TagId',
      ['p1', 'p2'],
      new Map([
        ['p1', [{ Id: 'p1-t1' }]],
        ['p2', [{ Id: 'p2-t1' }]],
      ])
    );

    expect(result.get('p1')?.removedIds.length).toBe(59);
    expect(result.get('p2')?.removedIds.length).toBe(59);
    expect(joinStore.deleteCalls.length).toBe(2);
    expect(joinStore.deleteCalls[0]).toEqual({
      And: [
        ['OwnerId', '=', 'p1'],
        ['TagId', 'in', result.get('p1')?.removedIds || []],
      ],
    });
    expect(joinStore.deleteCalls[1]).toEqual({
      And: [
        ['OwnerId', '=', 'p2'],
        ['TagId', 'in', result.get('p2')?.removedIds || []],
      ],
    });
    expect(joinStore.rows.map(item => `${item.OwnerId}:${item.TagId}`).sort()).toEqual(['p1:p1-t1', 'p2:p2-t1']);
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor createRelations keeps Error instance branch for target create failure', async () => {
  resetTargetCalls();

  const processor = new ManyToManyProcessor(parentCtor) as unknown as {
    createRelations: (
      joinRepo: Repository,
      targetRepo: unknown,
      targetModel: typeof targetCtor,
      parentId: string,
      joinField: string,
      inverseJoinField: string,
      items: any[],
      affectedIds: string[],
      errors: Error[]
    ) => Promise<void>;
  };

  const joinStore = createJoinRepoMock([]);
  const originalCreate = ManyToManyTarget.Create;
  ManyToManyTarget.Create = (async () => {
    throw new Error('create relation error object');
  }) as typeof ManyToManyTarget.Create;

  try {
    const meta = MetadataStorage.instance.getModelMetadata(joinCtor as any);
    const joinRepo = new Repository(meta);
    const affectedIds: string[] = [];
    const errors: Error[] = [];

    await processor.createRelations(joinRepo, {}, targetCtor, 'p1', 'OwnerId', 'TagId', [{ Name: 'new-tag' }], affectedIds, errors);

    expect(affectedIds).toEqual([]);
    expect(errors.length).toBe(1);
    expect(String(errors[0]?.message || '')).toBe('create relation error object');
  } finally {
    ManyToManyTarget.Create = originalCreate;
    joinStore.restore();
  }
});

test('many-to-many processor prepareForUpdate skips relation when joinModel is missing', async () => {
  const processor = new ManyToManyProcessor(prepareParentCtor);
  const meta = MetadataStorage.instance.getModelMetadata(prepareParentCtor as any) as any;
  const tagsField = meta.fields.get('Tags') as any;
  const originalRelation = tagsField.relation;

  tagsField.relation = {
    targetModel: originalRelation.targetModel,
    joinField: originalRelation.joinField,
    inverseJoinField: originalRelation.inverseJoinField,
  };

  try {
    const result = await processor.prepareForUpdate({ Name: 'parent-updated', Tags: [{ Id: 't1' }] });
    expect(result.processedValue).toEqual({ Name: 'parent-updated' });
    expect(result.relations.manyToManyRelations).toEqual([]);
    expect(Array.from(result.relations.touchedCollections || [])).toEqual([]);
  } finally {
    tagsField.relation = originalRelation;
  }
});

test('many-to-many processor createRelations wraps non-error throw into per-item error', async () => {
  resetTargetCalls();

  const processor = new ManyToManyProcessor(parentCtor) as unknown as {
    createRelations: (
      joinRepo: Repository,
      targetRepo: unknown,
      targetModel: typeof targetCtor,
      parentId: string,
      joinField: string,
      inverseJoinField: string,
      items: any[],
      affectedIds: string[],
      errors: Error[]
    ) => Promise<void>;
  };

  const joinStore = createJoinRepoMock([]);
  const originalCreate = ManyToManyTarget.Create;
  ManyToManyTarget.Create = (async () => {
    throw 'create relation non-error';
  }) as unknown as typeof ManyToManyTarget.Create;

  try {
    const meta = MetadataStorage.instance.getModelMetadata(joinCtor as any);
    const joinRepo = new Repository(meta);
    const affectedIds: string[] = [];
    const errors: Error[] = [];

    await processor.createRelations(joinRepo, {}, targetCtor, 'p1', 'OwnerId', 'TagId', [{ Name: 'new-tag' }], affectedIds, errors);

    expect(affectedIds).toEqual([]);
    expect(errors.length).toBe(1);
    expect(String(errors[0]?.message || '')).toContain('Failed to process relation item: create relation non-error');
  } finally {
    ManyToManyTarget.Create = originalCreate;
    joinStore.restore();
  }
});

test('many-to-many processor helper batchRemoveAssociationsWithDiff uses grouped parent deletes when removal pairs exceed 100', async () => {
  resetTargetCalls();

  const processor = new ManyToManyProcessor(parentCtor) as unknown as {
    batchRemoveAssociationsWithDiff: (
      joinRepo: Repository,
      joinField: string,
      inverseJoinField: string,
      parentIds: string[],
      replacementMap: Map<string, any[]>
    ) => Promise<Map<string, { existingIds: Set<string>; removedIds: string[] }>>;
  };

  const seedRows: JoinRow[] = [];
  for (let i = 1; i <= 60; i++) {
    seedRows.push({ Id: `p1-j${i}`, OwnerId: 'p1', TagId: `p1-t${i}` });
    seedRows.push({ Id: `p2-j${i}`, OwnerId: 'p2', TagId: `p2-t${i}` });
  }

  const joinStore = createJoinRepoMock(seedRows);
  try {
    const meta = MetadataStorage.instance.getModelMetadata(joinCtor as any);
    const joinRepo = new Repository(meta);

    const result = await processor.batchRemoveAssociationsWithDiff(
      joinRepo,
      'OwnerId',
      'TagId',
      ['p1', 'p2'],
      new Map([
        ['p1', []],
        ['p2', []],
      ])
    );

    expect(result.get('p1')?.removedIds.length).toBe(60);
    expect(result.get('p2')?.removedIds.length).toBe(60);
    expect(joinStore.deleteCalls.length).toBe(2);
    expect(joinStore.deleteCalls[0]).toEqual({
      And: [
        ['OwnerId', '=', 'p1'],
        ['TagId', 'in', seedRows.filter(r => r.OwnerId === 'p1').map(r => r.TagId)],
      ],
    });
    expect(joinStore.deleteCalls[1]).toEqual({
      And: [
        ['OwnerId', '=', 'p2'],
        ['TagId', 'in', seedRows.filter(r => r.OwnerId === 'p2').map(r => r.TagId)],
      ],
    });
  } finally {
    joinStore.restore();
  }
});

test('many-to-many processor helper removeExistingRelationsWithDiff keeps links when replacement ids are unchanged', async () => {
  resetTargetCalls();

  const processor = new ManyToManyProcessor(parentCtor) as unknown as {
    removeExistingRelationsWithDiff: (
      joinRepo: Repository,
      joinField: string,
      inverseJoinField: string,
      parentId: string,
      newItems: any[]
    ) => Promise<{ existingRecords: any[]; existingTargetIds: string[]; removedRecords: any[] }>;
  };

  const joinStore = createJoinRepoMock([
    { Id: 'j1', OwnerId: 'p1', TagId: 't1' },
    { Id: 'j2', OwnerId: 'p1', TagId: 't2' },
  ]);
  try {
    const meta = MetadataStorage.instance.getModelMetadata(joinCtor as any);
    const joinRepo = new Repository(meta);

    const diff = await processor.removeExistingRelationsWithDiff(joinRepo, 'OwnerId', 'TagId', 'p1', [{ Id: 't1' }, { Id: 't2' }]);

    expect(diff.existingTargetIds.sort()).toEqual(['t1', 't2']);
    expect(diff.removedRecords).toEqual([]);
    expect(joinStore.deleteCalls).toEqual([]);
  } finally {
    joinStore.restore();
  }
});
