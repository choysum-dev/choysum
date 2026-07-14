// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { Field, Model } from '../decorator';
import { MetadataStorage } from '../metadata/storage';
import { RepositoryFactory } from '../repository/repository_factory';
import { OneToManyProcessor } from './one-to-many';

type ModelCtor<T extends BaseModel> = { new (...args: never[]): T } & typeof BaseModel;

@Model('test.OneToManyProcessorParent')
class OneToManyProcessorParent extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;
}

@Model('test.OneToManyPrepareParent')
class OneToManyPrepareParent extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name?: string;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => OneToManyProcessorChild, inverseField: 'ParentId' as never },
  })
  Lines?: OneToManyProcessorChild[];
}

@Model('test.OneToManyProcessorChild')
class OneToManyProcessorChild extends BaseModel {
  @Field({ type: 'ManyToOne',
    relation: { targetModel: () => OneToManyProcessorParent, onDelete: 'SET NULL' } })
  ParentId?: OneToManyProcessorParent | null;

  @Field({ type: 'varchar', size: 64 })
  Name?: string;

  @Field({ type: 'varchar', size: 64 })
  ComputedFlag?: string;

  static createCalls: Array<Record<string, any>> = [];

  static updateCalls: Array<{ id: string; values: Record<string, any> }> = [];

  static override async Create<T extends BaseModel>(this: { new (...args: any[]): T } & typeof BaseModel, value: Record<string, any>): Promise<T> {
    OneToManyProcessorChild.createCalls.push({ ...value });
    return { Id: `CREATED-${OneToManyProcessorChild.createCalls.length}`, ...value } as T;
  }

  static override async UpdateById<T extends BaseModel>(
    this: { new (...args: any[]): T } & typeof BaseModel,
    id: string,
    values: Record<string, any>
  ): Promise<Partial<T>> {
    OneToManyProcessorChild.updateCalls.push({ id, values: { ...values } });
    return { Id: id, ...values } as Partial<T>;
  }
}

@Model('test.OneToManyRestrictChild')
class OneToManyRestrictChild extends BaseModel {
  @Field({ type: 'ManyToOne',
    relation: { targetModel: () => OneToManyProcessorParent, onDelete: 'RESTRICT' } })
  ParentId?: OneToManyProcessorParent | null;

  @Field({ type: 'varchar', size: 64 })
  Name?: string;
}

@Model('test.OneToManyCascadeChild')
class OneToManyCascadeChild extends BaseModel {
  @Field({ type: 'ManyToOne',
    relation: { targetModel: () => OneToManyProcessorParent, onDelete: 'CASCADE' } })
  ParentId?: OneToManyProcessorParent | null;

  @Field({ type: 'varchar', size: 64 })
  Name?: string;
}

@Model('test.OneToManyNoActionChild')
class OneToManyNoActionChild extends BaseModel {
  @Field({ type: 'ManyToOne',
    relation: { targetModel: () => OneToManyProcessorParent, onDelete: 'NO ACTION' } })
  ParentId?: OneToManyProcessorParent | null;

  @Field({ type: 'varchar', size: 64 })
  Name?: string;
}

const parentCtor = OneToManyProcessorParent as unknown as ModelCtor<OneToManyProcessorParent>;
const childCtor = OneToManyProcessorChild as unknown as ModelCtor<OneToManyProcessorChild>;
const restrictChildCtor = OneToManyRestrictChild as unknown as ModelCtor<OneToManyRestrictChild>;
const cascadeChildCtor = OneToManyCascadeChild as unknown as ModelCtor<OneToManyCascadeChild>;
const noActionChildCtor = OneToManyNoActionChild as unknown as ModelCtor<OneToManyNoActionChild>;
const prepareParentCtor = OneToManyPrepareParent as unknown as ModelCtor<OneToManyPrepareParent>;

type RelationRow = {
  Id: string;
  ParentId: string | null;
  Name?: string;
};

function matchesCondition(row: RelationRow, condition: unknown): boolean {
  if (Array.isArray(condition)) {
    const [field, op, value] = condition;
    const left = row[String(field) as keyof RelationRow];
    const normalizedOp = String(op || '').toLowerCase();
    if (normalizedOp === '=') return left === value;
    if (normalizedOp === 'in') return Array.isArray(value) && value.includes(left);
    return false;
  }

  if (condition && typeof condition === 'object' && Array.isArray((condition as { And?: unknown[] }).And)) {
    return (condition as { And: unknown[] }).And.every(part => matchesCondition(row, part));
  }

  return false;
}

function createRelationRepository(childCtor: typeof BaseModel, seedRows: RelationRow[]) {
  const rows = seedRows.map(row => ({ ...row }));
  const updateCalls: Array<{ values: Record<string, any>; matchedIds: string[] }> = [];
  const deleteCalls: string[][] = [];

  return {
    rows,
    updateCalls,
    deleteCalls,
    repo: {
      async search(condition: unknown) {
        return rows.filter(row => matchesCondition(row, condition)).map(row => ({ ...row }));
      },
      async update(values: Record<string, any>, condition: unknown) {
        const matched = rows.filter(row => matchesCondition(row, condition));
        matched.forEach(row => Object.assign(row, values));
        updateCalls.push({ values: { ...values }, matchedIds: matched.map(row => row.Id) });
        return matched.map(row => ({ ...row }));
      },
      async delete(condition: unknown) {
        const matchedIds = rows.filter(row => matchesCondition(row, condition)).map(row => row.Id);
        deleteCalls.push([...matchedIds]);
        for (const id of matchedIds) {
          const index = rows.findIndex(row => row.Id === id);
          if (index >= 0) rows.splice(index, 1);
        }
        return matchedIds.map(id => ({ Id: id }));
      },
      getModelClass() {
        return childCtor;
      },
    },
  };
}

function resetChildMetadata() {
  const childMeta = MetadataStorage.instance.getModelMetadata(childCtor);
  childMeta.computeGraph = { computeFields: new Set(['ComputedFlag']) } as unknown as typeof childMeta.computeGraph;
  OneToManyProcessorChild.createCalls = [];
  OneToManyProcessorChild.updateCalls = [];
}

test('one-to-many processor replace array keeps existing child, strips compute field, nulls removed rows, and creates new child', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [
    { Id: 'c1', ParentId: 'p1', Name: 'drop' },
    { Id: 'c2', ParentId: 'p1', Name: 'keep' },
  ]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.processRelationUpdate('p1', {
    type: 'OneToMany',
    fieldName: 'Lines',
    targetModel: childCtor,
    inverseField: 'ParentId',
    operations: [
      { Id: 'c2', Name: 'keep-next', ComputedFlag: 'client-overwrite' },
      { Name: 'new-row', ComputedFlag: 'client-overwrite' },
    ] as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]['operations'],
  });

  expect(result.errors).toEqual([]);
  expect(result.entityIds).toEqual(['c2', 'CREATED-1']);
  expect(store.updateCalls.length).toBe(1);
  expect(store.updateCalls[0]?.values).toEqual({ ParentId: null });
  expect(store.updateCalls[0]?.matchedIds).toEqual(['c1']);
  expect(OneToManyProcessorChild.updateCalls).toEqual([
    {
      id: 'c2',
      values: { Name: 'keep-next', ParentId: 'p1' },
    },
  ]);
  expect(OneToManyProcessorChild.createCalls).toEqual([{ Name: 'new-row', ParentId: 'p1' }]);
});

test('one-to-many processor update reports child outside parent scope and skips update helper', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [{ Id: 'c9', ParentId: 'other-parent', Name: 'foreign' }]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.processRelationUpdate('p1', {
    type: 'OneToMany',
    fieldName: 'Lines',
    targetModel: childCtor,
    inverseField: 'ParentId',
    operations: {
      update: [{ Id: 'c9', Name: 'should-fail' }],
    },
  } as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]);

  expect(result.entityIds).toEqual([]);
  expect(result.errors.length).toBe(1);
  expect(String(result.errors[0]?.message || '').includes('Item does not exist or does not belong to the parent entity')).toBe(true);
  expect(OneToManyProcessorChild.updateCalls).toEqual([]);
});

test('one-to-many processor batch replace only disassociates rows absent from each parent replacement set', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [
    { Id: 'c1', ParentId: 'p1', Name: 'remove-from-p1' },
    { Id: 'c2', ParentId: 'p1', Name: 'keep-p1' },
    { Id: 'c3', ParentId: 'p2', Name: 'keep-p2' },
  ]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.batchProcessRelationUpdate(
    ['p1', 'p2'],
    [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: childCtor,
        inverseField: 'ParentId',
        operations: [{ Id: 'c2', Name: 'keep-next-p1', ComputedFlag: 'client-overwrite' }] as unknown as Parameters<
          OneToManyProcessor['batchProcessRelationUpdate']
        >[1][number]['operations'],
      },
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: childCtor,
        inverseField: 'ParentId',
        operations: [{ Id: 'c3', Name: 'keep-next-p2', ComputedFlag: 'client-overwrite' }] as unknown as Parameters<
          OneToManyProcessor['batchProcessRelationUpdate']
        >[1][number]['operations'],
      },
    ]
  );

  expect(result.errors).toEqual([]);
  expect(store.updateCalls.length).toBe(1);
  expect(store.updateCalls[0]?.values).toEqual({ ParentId: null });
  expect(store.updateCalls[0]?.matchedIds).toEqual(['c1']);
  expect(OneToManyProcessorChild.updateCalls).toEqual([
    { id: 'c2', values: { Name: 'keep-next-p1', ParentId: 'p1' } },
    { id: 'c3', values: { Name: 'keep-next-p2', ParentId: 'p2' } },
  ]);
  expect(store.rows.find(row => row.Id === 'c2')?.ParentId).toBe('p1');
  expect(store.rows.find(row => row.Id === 'c3')?.ParentId).toBe('p2');
  expect(store.rows.find(row => row.Id === 'c1')?.ParentId).toBe(null);
});

test('one-to-many processor delete returns restrict error when relation policy forbids detach', async () => {
  const store = createRelationRepository(restrictChildCtor, [{ Id: 'r1', ParentId: 'p1', Name: 'locked' }]);
  RepositoryFactory.setRepository(restrictChildCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.processRelationUpdate('p1', {
    type: 'OneToMany',
    fieldName: 'Lines',
    targetModel: restrictChildCtor,
    inverseField: 'ParentId',
    operations: {
      delete: [{ Id: 'r1' }],
    },
  });

  expect(result.entityIds).toEqual([]);
  expect(result.errors.length).toBe(1);
  expect(String(result.errors[0]?.message || '').includes('has onDelete set to RESTRICT')).toBe(true);
  expect(store.updateCalls).toEqual([]);
  expect(store.deleteCalls).toEqual([]);
});

test('one-to-many processor object operations handle mixed success and validation errors', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [
    { Id: 'c1', ParentId: 'p1', Name: 'delete-me' },
    { Id: 'c2', ParentId: 'p1', Name: 'update-me' },
    { Id: 'c3', ParentId: 'other-parent', Name: 'foreign' },
  ]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.processRelationUpdate('p1', {
    type: 'OneToMany',
    fieldName: 'Lines',
    targetModel: childCtor,
    inverseField: 'ParentId',
    operations: {
      delete: [{ Id: 'c1' }, { Id: 'missing' }, {} as unknown as { Id: string }],
      update: [{ Id: 'c2', Name: 'updated', ComputedFlag: 'client-overwrite' }, { Name: 'no-id' } as unknown as { Id: string; Name: string }],
      create: [
        { Id: 'bad-create-id', Name: 'bad' },
        { Name: 'new-child', ComputedFlag: 'client-overwrite' },
      ],
    },
  } as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]);

  expect(result.entityIds).toEqual(['c1', 'c2', 'CREATED-1']);
  expect(result.errors.length).toBe(4);
  expect(result.errors.some(error => String(error.message || '').includes('Item does not exist or does not belong to the parent entity'))).toBe(true);
  expect(result.errors.some(error => String(error.message || '').includes('Could not extract Id from delete item'))).toBe(true);
  expect(result.errors.some(error => String(error.message || '').includes('Create item must not include Id'))).toBe(true);

  expect(store.updateCalls.length).toBe(1);
  expect(store.updateCalls[0]?.values).toEqual({ ParentId: null });
  expect(store.updateCalls[0]?.matchedIds).toEqual(['c1']);

  expect(OneToManyProcessorChild.updateCalls).toEqual([{ id: 'c2', values: { Name: 'updated' } }]);
  expect(OneToManyProcessorChild.createCalls).toEqual([{ Name: 'new-child', ParentId: 'p1' }]);
});

test('one-to-many processor batch delete aggregates set-null successes and missing-item errors', async () => {
  const store = createRelationRepository(childCtor, [
    { Id: 'c1', ParentId: 'p1', Name: 'child-p1' },
    { Id: 'c2', ParentId: 'p2', Name: 'child-p2' },
  ]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.batchProcessRelationUpdate(
    ['p1', 'p2'],
    [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: childCtor,
        inverseField: 'ParentId',
        operations: { delete: [{ Id: 'c1' }, { Id: 'missing' }] },
      },
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: childCtor,
        inverseField: 'ParentId',
        operations: { delete: [{ Id: 'c2' }] },
      },
    ]
  );

  expect(result.success.map(item => item.entityId)).toEqual(['c1', 'c2']);
  expect(result.errors.length).toBe(1);
  expect(String(result.errors[0]?.error?.message || '').includes('item(s) do not exist or do not belong to the specified parent entity')).toBe(true);
  expect(result.summary).toEqual({
    totalOperations: 3,
    successfulOperations: 2,
    failedOperations: 1,
    relationType: 'OneToMany',
  });

  expect(store.updateCalls.length).toBe(1);
  expect(store.updateCalls[0]?.values).toEqual({ ParentId: null });
  expect(store.updateCalls[0]?.matchedIds).toEqual(['c1', 'c2']);
  expect(store.deleteCalls).toEqual([]);
});

test('one-to-many processor prepareForUpdate removes null collection from write payload without relation operations', async () => {
  const processor = new OneToManyProcessor(prepareParentCtor);
  const result = await processor.prepareForUpdate({ Name: 'parent-updated', Lines: null });

  expect(result.processedValue).toEqual({ Name: 'parent-updated' });
  expect(result.relations.oneToManyRelations).toEqual([]);
  expect(Array.from(result.relations.touchedCollections || [])).toEqual([]);
});

test('one-to-many processor prepareForCreate strips one2many fields and keeps touched collections', async () => {
  const processor = new OneToManyProcessor(prepareParentCtor);
  const result = await processor.prepareForCreate({
    Name: 'parent-created',
    Lines: [{ Name: 'line-1' }],
  });

  expect(result.processedValue).toEqual({ Name: 'parent-created' });
  expect(result.relations.oneToManyRelations.length).toBe(1);
  expect(result.relations.oneToManyRelations[0]?.fieldName).toBe('Lines');
  expect(Array.from(result.relations.touchedCollections || [])).toEqual(['Lines']);
});

test('one-to-many processor prepareForUpdate ignores relation write when one2many config is incomplete', async () => {
  const processor = new OneToManyProcessor(prepareParentCtor);
  const meta = MetadataStorage.instance.getModelMetadata(prepareParentCtor as any) as any;
  const linesField = meta.fields.get('Lines') as any;
  const originalRelation = linesField.relation;

  linesField.relation = { targetModel: originalRelation.targetModel };
  try {
    const result = await processor.prepareForUpdate({
      Name: 'parent-updated',
      Lines: [{ Name: 'line-1' }],
    });

    expect(result.processedValue).toEqual({ Name: 'parent-updated' });
    expect(result.relations.oneToManyRelations).toEqual([]);
    expect(Array.from(result.relations.touchedCollections || [])).toEqual([]);
  } finally {
    linesField.relation = originalRelation;
  }
});

test('one-to-many processor throws on mismatched operation type', async () => {
  const processor = new OneToManyProcessor(parentCtor);
  let message = '';
  try {
    await processor.processRelationUpdate('p1', {
      type: 'ManyToMany',
    } as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('Expected a OneToMany operation')).toBe(true);
});

test('one-to-many processor batch validates parent-operation cardinality and operation type', async () => {
  const processor = new OneToManyProcessor(parentCtor);

  let sizeMessage = '';
  try {
    await processor.batchProcessRelationUpdate(
      ['p1', 'p2'],
      [
        {
          type: 'OneToMany',
          fieldName: 'Lines',
          targetModel: childCtor,
          inverseField: 'ParentId',
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
          type: 'ManyToMany',
          fieldName: 'Lines',
          targetModel: childCtor,
          inverseField: 'ParentId',
          operations: [],
        } as unknown as Parameters<OneToManyProcessor['batchProcessRelationUpdate']>[1][number],
      ]
    );
  } catch (error) {
    typeMessage = String((error as Error)?.message || error);
  }

  expect(typeMessage.includes('Expected a OneToMany operation')).toBe(true);
});

test('one-to-many processor returns structured error when repository lookup fails', async () => {
  const processor = new OneToManyProcessor(parentCtor);
  const originalGetRepository = RepositoryFactory.getRepository;

  RepositoryFactory.getRepository = (() => {
    throw new Error('repo missing');
  }) as unknown as typeof RepositoryFactory.getRepository;

  try {
    const result = await processor.processRelationUpdate('p1', {
      type: 'OneToMany',
      fieldName: 'Lines',
      targetModel: childCtor,
      inverseField: 'ParentId',
      operations: [],
    });

    expect(result.entityIds).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.message || '').includes('repo missing')).toBe(true);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('one-to-many processor delete uses CASCADE strategy and calls repository.delete', async () => {
  const store = createRelationRepository(cascadeChildCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'child' }]);
  RepositoryFactory.setRepository(cascadeChildCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.processRelationUpdate('p1', {
    type: 'OneToMany',
    fieldName: 'Lines',
    targetModel: cascadeChildCtor,
    inverseField: 'ParentId',
    operations: {
      delete: [{ Id: 'c1' }],
    },
  });

  expect(result.errors).toEqual([]);
  expect(result.entityIds).toEqual(['c1']);
  expect(store.deleteCalls).toEqual([['c1']]);
  expect(store.updateCalls).toEqual([]);
});

test('one-to-many processor replace empty array removes all existing children by delete policy', async () => {
  const store = createRelationRepository(cascadeChildCtor, [
    { Id: 'c1', ParentId: 'p1', Name: 'drop-1' },
    { Id: 'c2', ParentId: 'p1', Name: 'drop-2' },
  ]);
  RepositoryFactory.setRepository(cascadeChildCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.processRelationUpdate('p1', {
    type: 'OneToMany',
    fieldName: 'Lines',
    targetModel: cascadeChildCtor,
    inverseField: 'ParentId',
    operations: [] as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]['operations'],
  });

  expect(result.errors).toEqual([]);
  expect(result.entityIds).toEqual([]);
  expect(store.deleteCalls).toEqual([['c1', 'c2']]);
  expect(store.rows).toEqual([]);
});

test('one-to-many processor batch delete reports cascade execution failure from delete strategy', async () => {
  const throwingRepo = {
    async search(condition: unknown) {
      if (
        condition &&
        typeof condition === 'object' &&
        Array.isArray((condition as { And?: unknown[] }).And) &&
        JSON.stringify(condition).includes('missing')
      ) {
        return [];
      }
      return [{ Id: 'c1', ParentId: 'p1' }];
    },
    async update() {
      return [];
    },
    async delete() {
      throw new Error('delete failed');
    },
    getModelClass() {
      return cascadeChildCtor;
    },
  };
  RepositoryFactory.setRepository(cascadeChildCtor, throwingRepo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.batchProcessRelationUpdate(
    ['p1'],
    [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: cascadeChildCtor,
        inverseField: 'ParentId',
        operations: { delete: [{ Id: 'c1' }, { Id: 'missing' }] },
      },
    ]
  );

  expect(result.success).toEqual([]);
  expect(result.errors.length).toBe(2);
  expect(result.errors.some(item => String(item.error?.message || '').includes('delete failed'))).toBe(true);
  expect(result.errors.some(item => String(item.error?.message || '').includes('do not exist or do not belong to the specified parent entity'))).toBe(true);
});

test('one-to-many processor batch delete reports set-null execution failure from delete strategy', async () => {
  const throwingRepo = {
    async search(condition: unknown) {
      if (
        condition &&
        typeof condition === 'object' &&
        Array.isArray((condition as { And?: unknown[] }).And) &&
        JSON.stringify(condition).includes('missing')
      ) {
        return [];
      }
      return [{ Id: 'c1', ParentId: 'p1' }];
    },
    async update() {
      throw new Error('set null failed');
    },
    async delete() {
      return [];
    },
    getModelClass() {
      return childCtor;
    },
  };
  RepositoryFactory.setRepository(childCtor, throwingRepo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.batchProcessRelationUpdate(
    ['p1'],
    [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: childCtor,
        inverseField: 'ParentId',
        operations: { delete: [{ Id: 'c1' }, { Id: 'missing' }] },
      },
    ]
  );

  expect(result.success).toEqual([]);
  expect(result.errors.length).toBe(2);
  expect(result.errors.some(item => String(item.error?.message || '').includes('set null failed'))).toBe(true);
  expect(result.errors.some(item => String(item.error?.message || '').includes('do not exist or do not belong to the specified parent entity'))).toBe(true);
});

test('one-to-many processor replace can re-parent existing child id from another parent', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [{ Id: 'c9', ParentId: 'other-parent', Name: 'foreign' }]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.processRelationUpdate('p1', {
    type: 'OneToMany',
    fieldName: 'Lines',
    targetModel: childCtor,
    inverseField: 'ParentId',
    operations: [{ Id: 'c9', Name: 're-parented' }] as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]['operations'],
  });

  expect(result.errors).toEqual([]);
  expect(result.entityIds).toEqual(['c9']);
  expect(OneToManyProcessorChild.updateCalls).toEqual([
    {
      id: 'c9',
      values: {
        Name: 're-parented',
        ParentId: 'p1',
      },
    },
  ]);
});

test('one-to-many processor batch replace returns grouped error when RESTRICT policy blocks disassociation', async () => {
  const store = createRelationRepository(restrictChildCtor, [{ Id: 'r1', ParentId: 'p1', Name: 'locked' }]);
  RepositoryFactory.setRepository(restrictChildCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.batchProcessRelationUpdate(
    ['p1'],
    [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: restrictChildCtor,
        inverseField: 'ParentId',
        operations: [] as unknown as Parameters<OneToManyProcessor['batchProcessRelationUpdate']>[1][number]['operations'],
      },
    ]
  );

  expect(result.success).toEqual([]);
  expect(result.errors.length).toBe(1);
  expect(String(result.errors[0]?.error?.message || '').includes('Cannot remove 1 association(s) because onDelete is set to RESTRICT')).toBe(true);
});

test('one-to-many processor batch delete supports mixed strategy groups across target models', async () => {
  const setNullStore = createRelationRepository(childCtor, [{ Id: 's1', ParentId: 'p1', Name: 'set-null-child' }]);
  const cascadeStore = createRelationRepository(cascadeChildCtor, [{ Id: 'c1', ParentId: 'p2', Name: 'cascade-child' }]);
  RepositoryFactory.setRepository(childCtor, setNullStore.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);
  RepositoryFactory.setRepository(cascadeChildCtor, cascadeStore.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.batchProcessRelationUpdate(
    ['p1', 'p2'],
    [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: childCtor,
        inverseField: 'ParentId',
        operations: { delete: [{ Id: 's1' }] },
      },
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: cascadeChildCtor,
        inverseField: 'ParentId',
        operations: { delete: [{ Id: 'c1' }] },
      },
    ]
  );

  expect(result.success.map(item => item.entityId)).toEqual(['s1', 'c1']);
  expect(result.errors).toEqual([]);

  expect(setNullStore.updateCalls).toEqual([
    {
      values: { ParentId: null },
      matchedIds: ['s1'],
    },
  ]);
  expect(setNullStore.deleteCalls).toEqual([]);

  expect(cascadeStore.deleteCalls).toEqual([['c1']]);
  expect(cascadeStore.updateCalls).toEqual([]);
});

test('one-to-many processor replace empty array on parent with no children is a no-op', async () => {
  const store = createRelationRepository(cascadeChildCtor, []);
  RepositoryFactory.setRepository(cascadeChildCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.processRelationUpdate('p-empty', {
    type: 'OneToMany',
    fieldName: 'Lines',
    targetModel: cascadeChildCtor,
    inverseField: 'ParentId',
    operations: [] as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]['operations'],
  });

  expect(result.errors).toEqual([]);
  expect(result.entityIds).toEqual([]);
  expect(store.deleteCalls).toEqual([]);
  expect(store.updateCalls).toEqual([]);
  expect(store.rows).toEqual([]);
});

test('one-to-many processor batch create records invalid id item and still creates valid child', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, []);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.batchProcessRelationUpdate(
    ['p1'],
    [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: childCtor,
        inverseField: 'ParentId',
        operations: {
          create: [{ Id: 'bad-id', Name: 'bad' }, { Name: 'good-create' }] as unknown as Parameters<
            OneToManyProcessor['batchProcessRelationUpdate']
          >[1][number]['operations'] extends { create: infer T }
            ? T
            : never,
        },
      },
    ]
  );

  expect(result.success.map(item => item.entityId)).toEqual(['CREATED-1']);
  expect(result.errors.length).toBe(1);
  expect(String(result.errors[0]?.error?.message || '').includes('Create item must not include Id')).toBe(true);
  expect(OneToManyProcessorChild.createCalls).toEqual([{ Name: 'good-create', ParentId: 'p1' }]);
});

test('one-to-many processor batch update records repository search failure as batch error', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'row' }]);
  const originalSearch = store.repo.search;
  store.repo.search = async (condition: unknown) => {
    if (condition && typeof condition === 'object' && Array.isArray((condition as { And?: unknown[] }).And)) {
      throw new Error('batch update search failed');
    }
    return originalSearch.call(store.repo, condition);
  };

  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.batchProcessRelationUpdate(
    ['p1'],
    [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: childCtor,
        inverseField: 'ParentId',
        operations: {
          update: [{ Id: 'c1', Name: 'x' }] as unknown as Parameters<OneToManyProcessor['batchProcessRelationUpdate']>[1][number]['operations'] extends {
            update: infer T;
          }
            ? T
            : never,
        },
      },
    ]
  );

  expect(result.success).toEqual([]);
  expect(result.errors.length).toBe(1);
  expect(String(result.errors[0]?.error?.message || '').includes('batch update search failed')).toBe(true);
  expect(OneToManyProcessorChild.updateCalls).toEqual([]);
});

test('one-to-many processor batch replace records update helper failure as batch error', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'keep' }]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const originalUpdateById = OneToManyProcessorChild.UpdateById;
  OneToManyProcessorChild.UpdateById = (async () => {
    throw new Error('batch replace update failed');
  }) as typeof OneToManyProcessorChild.UpdateById;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'OneToMany',
          fieldName: 'Lines',
          targetModel: childCtor,
          inverseField: 'ParentId',
          operations: [{ Id: 'c1', Name: 'next' }] as unknown as Parameters<OneToManyProcessor['batchProcessRelationUpdate']>[1][number]['operations'],
        },
      ]
    );

    expect(result.success).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.error?.message || '').includes('batch replace update failed')).toBe(true);
  } finally {
    OneToManyProcessorChild.UpdateById = originalUpdateById;
  }
});

test('one-to-many processor prepareForUpdate skips missing and undefined fields in changedFields', async () => {
  const processor = new OneToManyProcessor(prepareParentCtor);

  const missing = await processor.prepareForUpdate({ Name: 'p1' }, ['Lines']);
  expect(missing.processedValue).toEqual({ Name: 'p1' });
  expect(missing.relations.oneToManyRelations).toEqual([]);

  const undef = await processor.prepareForUpdate({ Name: 'p2', Lines: undefined }, ['Lines']);
  expect(undef.processedValue).toEqual({ Name: 'p2' });
  expect(undef.relations.oneToManyRelations).toEqual([]);
  expect(Array.from(undef.relations.touchedCollections || [])).toEqual([]);
});

test('one-to-many processor replace array wraps non-error failures and keeps successful ids only', async () => {
  resetChildMetadata();
  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'keep' }]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const originalUpdateById = OneToManyProcessorChild.UpdateById;
  const originalCreate = OneToManyProcessorChild.Create;

  OneToManyProcessorChild.UpdateById = (async (_id: string) => {
    throw 'update string failure';
  }) as typeof OneToManyProcessorChild.UpdateById;
  OneToManyProcessorChild.Create = (async () => {
    throw 'create string failure';
  }) as typeof OneToManyProcessorChild.Create;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'OneToMany',
      fieldName: 'Lines',
      targetModel: childCtor,
      inverseField: 'ParentId',
      operations: [{ Id: 'c1', Name: 'x' }, { Name: 'new' }] as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]['operations'],
    });

    expect(result.entityIds).toEqual([]);
    expect(result.errors.length).toBe(2);
    expect(result.errors.some(error => String(error.message || '').includes('update string failure'))).toBe(true);
    expect(result.errors.some(error => String(error.message || '').includes('create string failure'))).toBe(true);
  } finally {
    OneToManyProcessorChild.UpdateById = originalUpdateById;
    OneToManyProcessorChild.Create = originalCreate;
  }
});

test('one-to-many processor object mode wraps non-error delete/update/create failures', async () => {
  resetChildMetadata();
  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'row' }]);
  const originalUpdate = store.repo.update;
  store.repo.update = async () => {
    throw 'delete string failure';
  };
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const originalUpdateById = OneToManyProcessorChild.UpdateById;
  const originalCreate = OneToManyProcessorChild.Create;
  OneToManyProcessorChild.UpdateById = (async () => {
    throw 'update string failure';
  }) as typeof OneToManyProcessorChild.UpdateById;
  OneToManyProcessorChild.Create = (async () => {
    throw 'create string failure';
  }) as typeof OneToManyProcessorChild.Create;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'OneToMany',
      fieldName: 'Lines',
      targetModel: childCtor,
      inverseField: 'ParentId',
      operations: {
        delete: [{ Id: 'c1' }],
        update: [{ Id: 'c1', Name: 'u' }] as unknown as Array<{ Id: string; Name: string }>,
        create: [{ Name: 'n1' }],
      },
    } as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]);

    expect(result.errors.length).toBe(3);
    expect(result.errors.some(error => String(error.message || '').includes('delete string failure'))).toBe(true);
    expect(result.errors.some(error => String(error.message || '').includes('update string failure'))).toBe(true);
    expect(result.errors.some(error => String(error.message || '').includes('create string failure'))).toBe(true);
  } finally {
    store.repo.update = originalUpdate;
    OneToManyProcessorChild.UpdateById = originalUpdateById;
    OneToManyProcessorChild.Create = originalCreate;
  }
});

test('one-to-many processor wraps non-error top-level process errors into relation result', async () => {
  const originalGetRepository = RepositoryFactory.getRepository;
  RepositoryFactory.getRepository = (() => {
    throw 'repo string failure';
  }) as unknown as typeof RepositoryFactory.getRepository;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'OneToMany',
      fieldName: 'Lines',
      targetModel: childCtor,
      inverseField: 'ParentId',
      operations: [],
    });

    expect(result.entityIds).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.message || '')).toContain('repo string failure');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('one-to-many processor batch replace handles missing diff entry and wraps non-error create/update failures', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'row' }]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    batchRemoveAssociationsWithDiff: (...args: any[]) => Promise<Map<string, { existingIds: Set<string>; removedIds: string[] }>>;
    batchProcessRelationUpdate: OneToManyProcessor['batchProcessRelationUpdate'];
  };
  const originalDiff = processor.batchRemoveAssociationsWithDiff;
  processor.batchRemoveAssociationsWithDiff = (async () => new Map()) as any;

  const originalUpdateById = OneToManyProcessorChild.UpdateById;
  const originalCreate = OneToManyProcessorChild.Create;
  OneToManyProcessorChild.UpdateById = (async () => {
    throw 'batch update string failure';
  }) as typeof OneToManyProcessorChild.UpdateById;
  OneToManyProcessorChild.Create = (async () => {
    throw 'batch create string failure';
  }) as typeof OneToManyProcessorChild.Create;

  try {
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'OneToMany',
          fieldName: 'Lines',
          targetModel: childCtor,
          inverseField: 'ParentId',
          operations: [{ Id: 'c1', Name: 'next' }, { Name: 'new' }] as unknown as Parameters<
            OneToManyProcessor['batchProcessRelationUpdate']
          >[1][number]['operations'],
        },
      ]
    );

    expect(result.success).toEqual([]);
    expect(result.errors.length).toBe(2);
    expect(result.errors.some(item => String(item.error?.message || '').includes('batch update string failure'))).toBe(true);
    expect(result.errors.some(item => String(item.error?.message || '').includes('batch create string failure'))).toBe(true);
  } finally {
    processor.batchRemoveAssociationsWithDiff = originalDiff;
    OneToManyProcessorChild.UpdateById = originalUpdateById;
    OneToManyProcessorChild.Create = originalCreate;
  }
});

test('one-to-many processor batch delete treats NO ACTION as blocked disassociation group', async () => {
  const store = createRelationRepository(noActionChildCtor, [{ Id: 'n1', ParentId: 'p1', Name: 'locked' }]);
  RepositoryFactory.setRepository(noActionChildCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.batchProcessRelationUpdate(
    ['p1'],
    [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: noActionChildCtor,
        inverseField: 'ParentId',
        operations: { delete: [{ Id: 'n1' }] },
      },
    ]
  );

  expect(result.success).toEqual([]);
  expect(result.errors.length).toBe(1);
  expect(String(result.errors[0]?.error?.message || '').includes('onDelete is set to RESTRICT')).toBe(true);
  expect(store.updateCalls).toEqual([]);
  expect(store.deleteCalls).toEqual([]);
});

test('one-to-many processor applyBatchDeleteStrategy short-circuits empty parentIds and no matched rows', async () => {
  const store = createRelationRepository(cascadeChildCtor, []);
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    applyBatchDeleteStrategy: (...args: any[]) => Promise<void>;
  };

  await processor.applyBatchDeleteStrategy(store.repo, cascadeChildCtor, 'ParentId', []);
  expect(store.updateCalls).toEqual([]);
  expect(store.deleteCalls).toEqual([]);

  await processor.applyBatchDeleteStrategy(store.repo, cascadeChildCtor, 'ParentId', ['missing-parent']);
  expect(store.updateCalls).toEqual([]);
  expect(store.deleteCalls).toEqual([]);
});

test('one-to-many processor applyBatchDeleteStrategy throws for NO ACTION policy with affected rows', async () => {
  const store = createRelationRepository(noActionChildCtor, [{ Id: 'n1', ParentId: 'p1', Name: 'locked' }]);
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    applyBatchDeleteStrategy: (...args: any[]) => Promise<void>;
  };

  let message = '';
  try {
    await processor.applyBatchDeleteStrategy(store.repo, noActionChildCtor, 'ParentId', ['p1']);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('onDelete is set to NO ACTION')).toBe(true);
  expect(store.updateCalls).toEqual([]);
  expect(store.deleteCalls).toEqual([]);
});

test('one-to-many processor stripChildComputeFields tolerates metadata lookup exceptions', () => {
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    stripChildComputeFields: (ctor: typeof BaseModel, row: Record<string, any>) => Record<string, any>;
  };
  const originalGetModelMetadata = MetadataStorage.instance.getModelMetadata;

  MetadataStorage.instance.getModelMetadata = (() => {
    throw new Error('metadata unavailable');
  }) as typeof MetadataStorage.instance.getModelMetadata;

  try {
    const cleaned = processor.stripChildComputeFields(childCtor, { Name: 'n1', ComputedFlag: 'keep-when-meta-fails' });
    expect(cleaned).toEqual({ Name: 'n1', ComputedFlag: 'keep-when-meta-fails' });
  } finally {
    MetadataStorage.instance.getModelMetadata = originalGetModelMetadata;
  }
});

test('one-to-many processor batch wraps non-error repository lookup failures at top-level', async () => {
  const processor = new OneToManyProcessor(parentCtor);
  const originalGetRepository = RepositoryFactory.getRepository;

  RepositoryFactory.getRepository = (() => {
    throw 'batch repo string failure';
  }) as unknown as typeof RepositoryFactory.getRepository;

  try {
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'OneToMany',
          fieldName: 'Lines',
          targetModel: childCtor,
          inverseField: 'ParentId',
          operations: { delete: [{ Id: 'c1' }] },
        },
      ]
    );

    expect(result.success).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.error?.message || '').includes('batch repo string failure')).toBe(true);
    expect(result.errors[0]?.targetModel).toBe(childCtor.name);
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('one-to-many processor object replace keeps entityIds empty when update/create helpers return falsy', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [
    { Id: 'c1', ParentId: 'p1', Name: 'keep' },
    { Id: 'c2', ParentId: 'p1', Name: 'drop' },
  ]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const originalUpdateById = OneToManyProcessorChild.UpdateById;
  const originalCreate = OneToManyProcessorChild.Create;
  OneToManyProcessorChild.UpdateById = (async () => null as any) as typeof OneToManyProcessorChild.UpdateById;
  OneToManyProcessorChild.Create = (async () => '' as any) as typeof OneToManyProcessorChild.Create;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'OneToMany',
      fieldName: 'Lines',
      targetModel: childCtor,
      inverseField: 'ParentId',
      operations: {
        replace: [{ Id: 'c1', Name: 'n1' }, { Id: 'c9', Name: 'n9' }, { Name: 'new-item' }] as unknown as Array<{ Id?: string; Name: string }>,
      },
    } as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]);

    expect(result.errors).toEqual([]);
    expect(result.entityIds).toEqual([]);
  } finally {
    OneToManyProcessorChild.UpdateById = originalUpdateById;
    OneToManyProcessorChild.Create = originalCreate;
  }
});

test('one-to-many processor object update wraps Error exception from repository search', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'row' }]);
  const originalSearch = store.repo.search;
  store.repo.search = async () => {
    throw new Error('object update search error');
  };
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'OneToMany',
      fieldName: 'Lines',
      targetModel: childCtor,
      inverseField: 'ParentId',
      operations: {
        update: [{ Id: 'c1', Name: 'next' }],
      },
    } as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]);

    expect(result.entityIds).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.message || '').includes('object update search error')).toBe(true);
  } finally {
    store.repo.search = originalSearch;
  }
});

test('one-to-many processor removeExistingRelationsWithDiff uses cascade delete for object replace diff', async () => {
  const store = createRelationRepository(cascadeChildCtor, [
    { Id: 'c1', ParentId: 'p1', Name: 'drop' },
    { Id: 'c2', ParentId: 'p1', Name: 'keep' },
  ]);
  RepositoryFactory.setRepository(cascadeChildCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.processRelationUpdate('p1', {
    type: 'OneToMany',
    fieldName: 'Lines',
    targetModel: cascadeChildCtor,
    inverseField: 'ParentId',
    operations: {
      replace: [{ Id: 'c2', Name: 'keep-next' }],
    },
  } as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]);

  expect(result.errors).toEqual([]);
  expect(result.entityIds).toEqual(['c2']);
  expect(store.deleteCalls.length).toBe(1);
  expect(store.deleteCalls[0]).toEqual(['c1']);
});

test('one-to-many processor batch helper returns empty map when no parent ids are provided', async () => {
  const store = createRelationRepository(childCtor, []);
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    batchRemoveAssociationsWithDiff: (...args: any[]) => Promise<Map<string, { existingIds: Set<string>; removedIds: string[] }>>;
  };

  const result = await processor.batchRemoveAssociationsWithDiff(store.repo, childCtor, 'ParentId', [], new Map());
  expect(Array.from(result.entries())).toEqual([]);
});

test('one-to-many processor applyBatchDeleteStrategy uses in-condition for multiple parents in set-null mode', async () => {
  const store = createRelationRepository(childCtor, [
    { Id: 'c1', ParentId: 'p1', Name: 'row-1' },
    { Id: 'c2', ParentId: 'p2', Name: 'row-2' },
  ]);
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    applyBatchDeleteStrategy: (...args: any[]) => Promise<void>;
  };

  await processor.applyBatchDeleteStrategy(store.repo, childCtor, 'ParentId', ['p1', 'p2']);

  expect(store.updateCalls.length).toBe(1);
  expect(store.updateCalls[0]?.values).toEqual({ ParentId: null });
  expect(store.updateCalls[0]?.matchedIds.sort()).toEqual(['c1', 'c2']);
});

test('one-to-many processor batch fallback uses unknown target when operation targetModel is missing', async () => {
  const processor = new OneToManyProcessor(parentCtor);
  const originalGetRepository = RepositoryFactory.getRepository;

  RepositoryFactory.getRepository = (() => {
    throw new Error('unknown target fallback');
  }) as unknown as typeof RepositoryFactory.getRepository;

  try {
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'OneToMany',
          fieldName: 'Lines',
          targetModel: {} as any,
          inverseField: 'ParentId',
          operations: { delete: [{ Id: 'c1' }] },
        },
      ]
    );

    expect(result.success).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.error?.message || '').includes('unknown target fallback')).toBe(true);
    expect(result.errors[0]?.targetModel).toBe('Unknown target');
  } finally {
    RepositoryFactory.getRepository = originalGetRepository;
  }
});

test('one-to-many processor stripChildComputeFields keeps payload when child has no compute fields', () => {
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    stripChildComputeFields: (ctor: typeof BaseModel, row: Record<string, any>) => Record<string, any>;
  };

  const cleaned = processor.stripChildComputeFields(restrictChildCtor, { Name: 'n1', ComputedFlag: 'keep' });
  expect(cleaned).toEqual({ Name: 'n1', ComputedFlag: 'keep' });
});

test('one-to-many processor executeDeleteStrategies wraps non-error failures for cascade and set-null branches', async () => {
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    executeDeleteStrategies: (
      repository: any,
      inverseField: string,
      groups: { cascadeIds: string[]; setNullIds: string[]; restrictIds: string[]; notFoundIds: string[] }
    ) => Promise<{ successIds: string[]; errors: Error[] }>;
  };

  const repo = {
    async delete() {
      throw 'cascade string failure';
    },
    async update() {
      throw 'set-null string failure';
    },
  };

  const result = await processor.executeDeleteStrategies(repo, 'ParentId', {
    cascadeIds: ['c1'],
    setNullIds: ['c2'],
    restrictIds: ['c3'],
    notFoundIds: ['c4'],
  });

  expect(result.successIds).toEqual([]);
  expect(result.errors.length).toBe(4);
  expect(result.errors.some(error => String(error.message || '').includes('Cascade delete failed'))).toBe(true);
  expect(result.errors.some(error => String(error.message || '').includes('Setting foreign key to NULL failed'))).toBe(true);
  expect(result.errors.some(error => String(error.message || '').includes('onDelete is set to RESTRICT'))).toBe(true);
  expect(result.errors.some(error => String(error.message || '').includes('do not exist or do not belong to the specified parent entity'))).toBe(true);
});

test('one-to-many processor applyBatchDeleteStrategy uses cascade branch and delete by affected ids', async () => {
  const store = createRelationRepository(cascadeChildCtor, [
    { Id: 'c1', ParentId: 'p1', Name: 'row-1' },
    { Id: 'c2', ParentId: 'p2', Name: 'row-2' },
  ]);
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    applyBatchDeleteStrategy: (...args: any[]) => Promise<void>;
  };

  await processor.applyBatchDeleteStrategy(store.repo, cascadeChildCtor, 'ParentId', ['p1', 'p2']);

  expect(store.deleteCalls.length).toBe(1);
  expect(store.deleteCalls[0]?.sort()).toEqual(['c1', 'c2']);
  expect(store.updateCalls).toEqual([]);
});

test('one-to-many processor applyBatchDeleteStrategy throws for restrict policy with affected rows', async () => {
  const store = createRelationRepository(restrictChildCtor, [{ Id: 'r1', ParentId: 'p1', Name: 'locked' }]);
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    applyBatchDeleteStrategy: (...args: any[]) => Promise<void>;
  };

  let message = '';
  try {
    await processor.applyBatchDeleteStrategy(store.repo, restrictChildCtor, 'ParentId', ['p1']);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('onDelete is set to RESTRICT')).toBe(true);
  expect(store.deleteCalls).toEqual([]);
  expect(store.updateCalls).toEqual([]);
});

test('one-to-many processor batchRemoveAssociationsWithDiff throws for NO ACTION when ids need disassociation', async () => {
  const store = createRelationRepository(noActionChildCtor, [{ Id: 'n1', ParentId: 'p1', Name: 'locked' }]);
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    batchRemoveAssociationsWithDiff: (...args: any[]) => Promise<Map<string, { existingIds: Set<string>; removedIds: string[] }>>;
  };

  let message = '';
  try {
    await processor.batchRemoveAssociationsWithDiff(store.repo, noActionChildCtor, 'ParentId', ['p1'], new Map([['p1', []]]));
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('onDelete is set to NO ACTION')).toBe(true);
});

test('one-to-many processor groupItemsByDeleteStrategy puts NO ACTION items into restrict group', async () => {
  const store = createRelationRepository(noActionChildCtor, [
    { Id: 'n1', ParentId: 'p1', Name: 'locked-1' },
    { Id: 'n2', ParentId: 'p1', Name: 'locked-2' },
  ]);
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    groupItemsByDeleteStrategy: (...args: any[]) => Promise<{
      cascadeIds: string[];
      setNullIds: string[];
      restrictIds: string[];
      notFoundIds: string[];
    }>;
  };

  const grouped = await processor.groupItemsByDeleteStrategy(store.repo, noActionChildCtor, 'ParentId', [
    { id: 'n1', parentId: 'p1' },
    { id: 'missing', parentId: 'p1' },
  ]);

  expect(grouped.cascadeIds).toEqual([]);
  expect(grouped.setNullIds).toEqual([]);
  expect(grouped.restrictIds).toEqual(['n1']);
  expect(grouped.notFoundIds).toEqual(['missing']);
});

test('one-to-many processor prepareForUpdate skips null one-to-many payload and keeps touchedCollections empty', async () => {
  const processor = new OneToManyProcessor(prepareParentCtor);
  const result = await processor.prepareForUpdate({ Name: 'parent', Lines: null }, ['Name', 'Lines']);

  expect(result.processedValue).toEqual({ Name: 'parent' });
  expect(result.relations.oneToManyRelations).toEqual([]);
  expect(Array.from(result.relations.touchedCollections || [])).toEqual([]);
});

test('one-to-many processor object replace catches non-error thrown from create helper', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, []);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const originalCreate = OneToManyProcessorChild.Create;
  OneToManyProcessorChild.Create = (async () => {
    throw 'replace create string failure';
  }) as unknown as typeof OneToManyProcessorChild.Create;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'OneToMany',
      fieldName: 'Lines',
      targetModel: childCtor,
      inverseField: 'ParentId',
      operations: {
        replace: [{ Name: 'new-child' }],
      },
    } as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]);

    expect(result.entityIds).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.message || '').includes('Failed to process replace item')).toBe(true);
  } finally {
    OneToManyProcessorChild.Create = originalCreate;
  }
});

test('one-to-many processor object delete catches non-error thrown from single-item delete strategy', async () => {
  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'row' }]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    applySingleItemDeleteStrategy: (...args: any[]) => Promise<{ success: boolean; error?: Error }>;
    processRelationUpdate: OneToManyProcessor['processRelationUpdate'];
  };
  const originalSingleDelete = processor.applySingleItemDeleteStrategy;
  processor.applySingleItemDeleteStrategy = (async () => {
    throw 'object delete string failure';
  }) as any;

  try {
    const result = await processor.processRelationUpdate('p1', {
      type: 'OneToMany',
      fieldName: 'Lines',
      targetModel: childCtor,
      inverseField: 'ParentId',
      operations: {
        delete: [{ Id: 'c1' }],
      },
    } as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]);

    expect(result.entityIds).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.message || '').includes('Failed to process delete item')).toBe(true);
  } finally {
    processor.applySingleItemDeleteStrategy = originalSingleDelete;
  }
});

test('one-to-many processor batch create branch catches non-error thrown from create helper', async () => {
  const store = createRelationRepository(childCtor, []);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const originalCreate = OneToManyProcessorChild.Create;
  OneToManyProcessorChild.Create = (async () => {
    throw 'batch create string failure';
  }) as unknown as typeof OneToManyProcessorChild.Create;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'OneToMany',
          fieldName: 'Lines',
          targetModel: childCtor,
          inverseField: 'ParentId',
          operations: {
            create: [{ Name: 'new-batch-row' } as any],
          },
        },
      ]
    );

    expect(result.success).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.error?.message || '').includes('batch create string failure')).toBe(true);
  } finally {
    OneToManyProcessorChild.Create = originalCreate;
  }
});

test('one-to-many processor object create keeps Error instance from create helper catch branch', async () => {
  const store = createRelationRepository(childCtor, []);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const originalCreate = OneToManyProcessorChild.Create;
  OneToManyProcessorChild.Create = (async () => {
    throw new Error('object create error branch');
  }) as unknown as typeof OneToManyProcessorChild.Create;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'OneToMany',
      fieldName: 'Lines',
      targetModel: childCtor,
      inverseField: 'ParentId',
      operations: {
        create: [{ Name: 'x1' }],
      },
    } as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]);

    expect(result.entityIds).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.message || '')).toBe('object create error branch');
  } finally {
    OneToManyProcessorChild.Create = originalCreate;
  }
});

test('one-to-many processor batch replace keeps Error instances from create and update helper catches', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'row-1' }]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const originalUpdateById = OneToManyProcessorChild.UpdateById;
  const originalCreate = OneToManyProcessorChild.Create;
  OneToManyProcessorChild.UpdateById = (async () => {
    throw new Error('batch replace update error');
  }) as unknown as typeof OneToManyProcessorChild.UpdateById;
  OneToManyProcessorChild.Create = (async () => {
    throw new Error('batch replace create error');
  }) as unknown as typeof OneToManyProcessorChild.Create;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'OneToMany',
          fieldName: 'Lines',
          targetModel: childCtor,
          inverseField: 'ParentId',
          operations: [{ Id: 'c1', Name: 'u1' }, { Name: 'new-1' }] as unknown as Parameters<
            OneToManyProcessor['batchProcessRelationUpdate']
          >[1][number]['operations'],
        },
      ]
    );

    expect(result.success).toEqual([]);
    expect(result.errors.length).toBe(2);
    expect(result.errors.some(item => String(item.error?.message || '') === 'batch replace create error')).toBe(true);
    expect(result.errors.some(item => String(item.error?.message || '') === 'batch replace update error')).toBe(true);
  } finally {
    OneToManyProcessorChild.UpdateById = originalUpdateById;
    OneToManyProcessorChild.Create = originalCreate;
  }
});

test('one-to-many processor batch update keeps Error instance when repository search throws Error', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'row' }]);
  const originalSearch = store.repo.search;
  store.repo.search = async (condition: unknown) => {
    if (condition && typeof condition === 'object' && Array.isArray((condition as { And?: unknown[] }).And)) {
      throw new Error('batch update search error branch');
    }
    return originalSearch.call(store.repo, condition);
  };
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.batchProcessRelationUpdate(
    ['p1'],
    [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: childCtor,
        inverseField: 'ParentId',
        operations: {
          update: [{ Id: 'c1', Name: 'next' }] as unknown as Parameters<OneToManyProcessor['batchProcessRelationUpdate']>[1][number]['operations'] extends {
            update: infer T;
          }
            ? T
            : never,
        },
      },
    ]
  );

  expect(result.success).toEqual([]);
  expect(result.errors.length).toBe(1);
  expect(String(result.errors[0]?.error?.message || '')).toBe('batch update search error branch');
});

test('one-to-many processor stripChildComputeFields falls back to empty set when computeGraph is absent', () => {
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    stripChildComputeFields: (ctor: typeof BaseModel, row: Record<string, any>) => Record<string, any>;
  };

  const childMeta = MetadataStorage.instance.getModelMetadata(childCtor) as any;
  const originalComputeGraph = childMeta.computeGraph;
  childMeta.computeGraph = undefined;

  try {
    const cleaned = processor.stripChildComputeFields(childCtor, { Name: 'n1', ComputedFlag: 'keep-without-graph' });
    expect(cleaned).toEqual({ Name: 'n1', ComputedFlag: 'keep-without-graph' });
  } finally {
    childMeta.computeGraph = originalComputeGraph;
  }
});

test('one-to-many processor batch replace skips success ids when create/update helpers return falsy success markers', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'row-1' }]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const originalUpdateById = OneToManyProcessorChild.UpdateById;
  const originalCreate = OneToManyProcessorChild.Create;
  OneToManyProcessorChild.UpdateById = (async () => null as any) as unknown as typeof OneToManyProcessorChild.UpdateById;
  OneToManyProcessorChild.Create = (async () => '' as any) as unknown as typeof OneToManyProcessorChild.Create;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'OneToMany',
          fieldName: 'Lines',
          targetModel: childCtor,
          inverseField: 'ParentId',
          operations: [{ Id: 'c1', Name: 'u1' }, { Name: 'new-1' }] as unknown as Parameters<
            OneToManyProcessor['batchProcessRelationUpdate']
          >[1][number]['operations'],
        },
      ]
    );

    expect(result.success).toEqual([]);
    expect(result.errors).toEqual([]);
  } finally {
    OneToManyProcessorChild.UpdateById = originalUpdateById;
    OneToManyProcessorChild.Create = originalCreate;
  }
});

test('one-to-many processor object replace keeps empty success when create helper returns empty Id string', async () => {
  const store = createRelationRepository(childCtor, []);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const originalCreate = OneToManyProcessorChild.Create;
  OneToManyProcessorChild.Create = (async () => '' as any) as unknown as typeof OneToManyProcessorChild.Create;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'OneToMany',
      fieldName: 'Lines',
      targetModel: childCtor,
      inverseField: 'ParentId',
      operations: {
        replace: [{ Name: 'new-child' }],
      },
    } as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]);

    expect(result.entityIds).toEqual([]);
    expect(result.errors).toEqual([]);
  } finally {
    OneToManyProcessorChild.Create = originalCreate;
  }
});

test('one-to-many processor applyDeleteStrategy covers CASCADE and SET NULL branches for affected rows', async () => {
  const cascadeStore = createRelationRepository(cascadeChildCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'row-1' }]);
  const setNullStore = createRelationRepository(childCtor, [{ Id: 's1', ParentId: 'p2', Name: 'row-2' }]);

  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    applyDeleteStrategy: (repository: any, targetModel: any, inverseField: string, parentId: string) => Promise<void>;
  };

  await processor.applyDeleteStrategy(cascadeStore.repo, cascadeChildCtor, 'ParentId', 'p1');
  await processor.applyDeleteStrategy(setNullStore.repo, childCtor, 'ParentId', 'p2');

  expect(cascadeStore.deleteCalls).toEqual([['c1']]);
  expect(cascadeStore.updateCalls).toEqual([]);
  expect(setNullStore.updateCalls).toEqual([{ values: { ParentId: null }, matchedIds: ['s1'] }]);
  expect(setNullStore.deleteCalls).toEqual([]);
});

test('one-to-many processor removeExistingRelations delegates to diff helper with repository model class', async () => {
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    removeExistingRelations: (repository: any, inverseField: string, parentId: string) => Promise<void>;
    removeExistingRelationsWithDiff: (...args: any[]) => Promise<any>;
  };

  let captured: any[] | undefined;
  processor.removeExistingRelationsWithDiff = (async (...args: any[]) => {
    captured = args;
    return new Map();
  }) as any;

  const repo = {
    getModelClass() {
      return childCtor;
    },
  } as any;

  await processor.removeExistingRelations(repo, 'ParentId', 'p100');

  expect(captured?.[0]).toBe(repo);
  expect(captured?.[1]).toBe(childCtor);
  expect(captured?.[2]).toBe('ParentId');
  expect(captured?.[3]).toBe('p100');
  expect(captured?.[4]).toEqual([]);
});

test('one-to-many processor removeExistingRelationsWithDiff throws on RESTRICT disassociation branch', async () => {
  const store = createRelationRepository(restrictChildCtor, [{ Id: 'r1', ParentId: 'p1', Name: 'locked' }]);
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    removeExistingRelationsWithDiff: (
      repository: any,
      targetModel: any,
      inverseField: string,
      parentId: string,
      newItems: any[]
    ) => Promise<{ existingIds: string[]; removedIds: string[] }>;
  };

  let message = '';
  try {
    await processor.removeExistingRelationsWithDiff(store.repo, restrictChildCtor, 'ParentId', 'p1', [{ Id: 'r2' }]);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('onDelete is set to RESTRICT')).toBe(true);
});

test('one-to-many processor groupItemsByDeleteStrategy covers SET NULL grouping branch', async () => {
  const store = createRelationRepository(childCtor, [
    { Id: 's1', ParentId: 'p1', Name: 'row-1' },
    { Id: 's2', ParentId: 'p1', Name: 'row-2' },
  ]);
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    groupItemsByDeleteStrategy: (
      repository: any,
      targetModel: any,
      inverseField: string,
      items: Array<{ id: string; parentId: string }>
    ) => Promise<{ cascadeIds: string[]; setNullIds: string[]; restrictIds: string[]; notFoundIds: string[] }>;
  };

  const grouped = await processor.groupItemsByDeleteStrategy(store.repo, childCtor, 'ParentId', [
    { id: 's1', parentId: 'p1' },
    { id: 'missing', parentId: 'p1' },
  ]);

  expect(grouped.cascadeIds).toEqual([]);
  expect(grouped.setNullIds).toEqual(['s1']);
  expect(grouped.restrictIds).toEqual([]);
  expect(grouped.notFoundIds).toEqual(['missing']);
});

test('one-to-many processor applySingleItemDeleteStrategy reports no-op for missing relation item', async () => {
  const store = createRelationRepository(childCtor, []);
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    applySingleItemDeleteStrategy: (
      repository: any,
      targetModel: any,
      inverseField: string,
      parentId: string,
      itemId: string
    ) => Promise<{ success: boolean; error?: Error }>;
  };

  const result = await processor.applySingleItemDeleteStrategy(store.repo, childCtor, 'ParentId', 'p1', 'missing');

  expect(result.success).toBe(false);
  expect(String(result.error?.message || '').includes('Item does not exist or does not belong to the parent entity')).toBe(true);
  expect(store.deleteCalls).toEqual([]);
  expect(store.updateCalls).toEqual([]);
});

test('one-to-many processor applySingleItemDeleteStrategy wraps NO ACTION branch into failed result', async () => {
  const store = createRelationRepository(noActionChildCtor, [{ Id: 'n1', ParentId: 'p1', Name: 'locked' }]);
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    applySingleItemDeleteStrategy: (
      repository: any,
      targetModel: any,
      inverseField: string,
      parentId: string,
      itemId: string
    ) => Promise<{ success: boolean; error?: Error }>;
  };

  const result = await processor.applySingleItemDeleteStrategy(store.repo, noActionChildCtor, 'ParentId', 'p1', 'n1');

  expect(result.success).toBe(false);
  expect(String(result.error?.message || '').includes('has onDelete set to NO ACTION')).toBe(true);
  expect(store.deleteCalls).toEqual([]);
  expect(store.updateCalls).toEqual([]);
});

test('one-to-many processor executeDeleteStrategies returns no-op result when all groups are empty', async () => {
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    executeDeleteStrategies: (
      repository: any,
      inverseField: string,
      groups: { cascadeIds: string[]; setNullIds: string[]; restrictIds: string[]; notFoundIds: string[] }
    ) => Promise<{ successIds: string[]; errors: Error[] }>;
  };

  const repo = {
    async delete() {
      throw new Error('should not run');
    },
    async update() {
      throw new Error('should not run');
    },
  };

  const result = await processor.executeDeleteStrategies(repo, 'ParentId', {
    cascadeIds: [],
    setNullIds: [],
    restrictIds: [],
    notFoundIds: [],
  });

  expect(result.successIds).toEqual([]);
  expect(result.errors).toEqual([]);
});

test('one-to-many processor array replace keeps entityIds empty when update/create helpers return falsy', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'row-1' }]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const originalUpdateById = OneToManyProcessorChild.UpdateById;
  const originalCreate = OneToManyProcessorChild.Create;
  OneToManyProcessorChild.UpdateById = (async () => null as any) as unknown as typeof OneToManyProcessorChild.UpdateById;
  OneToManyProcessorChild.Create = (async () => '' as any) as unknown as typeof OneToManyProcessorChild.Create;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'OneToMany',
      fieldName: 'Lines',
      targetModel: childCtor,
      inverseField: 'ParentId',
      operations: [{ Id: 'c1', Name: 'u1' }, { Name: 'new-1' }] as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]['operations'],
    });

    expect(result.entityIds).toEqual([]);
    expect(result.errors).toEqual([]);
  } finally {
    OneToManyProcessorChild.UpdateById = originalUpdateById;
    OneToManyProcessorChild.Create = originalCreate;
  }
});

test('one-to-many processor array replace wraps non-error helper throw into collection item error', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'row-1' }]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const originalUpdateById = OneToManyProcessorChild.UpdateById;
  OneToManyProcessorChild.UpdateById = (async () => {
    throw 'array replace string failure';
  }) as unknown as typeof OneToManyProcessorChild.UpdateById;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'OneToMany',
      fieldName: 'Lines',
      targetModel: childCtor,
      inverseField: 'ParentId',
      operations: [{ Id: 'c1', Name: 'u1' }] as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]['operations'],
    });

    expect(result.entityIds).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.message || '').includes('Failed to process collection item')).toBe(true);
  } finally {
    OneToManyProcessorChild.UpdateById = originalUpdateById;
  }
});

test('one-to-many processor object replace keeps success ids for existing update and new create', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'old' }]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.processRelationUpdate('p1', {
    type: 'OneToMany',
    fieldName: 'Lines',
    targetModel: childCtor,
    inverseField: 'ParentId',
    operations: {
      replace: [{ Id: 'c1', Name: 'updated' }, { Name: 'new-child' }] as unknown as Array<{ Id?: string; Name: string }>,
    },
  } as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]);

  expect(result.errors).toEqual([]);
  expect(result.entityIds).toEqual(['c1', 'CREATED-1']);
});

test('one-to-many processor batch replace classifies Error and non-Error extraction failures', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, []);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const stringFailItem = new Proxy(
    { Name: 'string-fail' },
    {
      has(_target, key) {
        if (key === 'Id') throw 'batch replace string failure';
        return false;
      },
    }
  );

  const errorFailItem = new Proxy(
    { Name: 'error-fail' },
    {
      has(_target, key) {
        if (key === 'Id') throw new Error('batch replace error failure');
        return false;
      },
    }
  );

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.batchProcessRelationUpdate(
    ['p1'],
    [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: childCtor,
        inverseField: 'ParentId',
        operations: [stringFailItem as any, errorFailItem as any],
      },
    ]
  );

  expect(result.success).toEqual([]);
  expect(result.errors.length).toBeGreaterThanOrEqual(1);
  const mergedMessages = result.errors.map(item => String(item.error?.message || '')).join(' | ');
  expect(mergedMessages.includes('batch replace string failure') || mergedMessages.includes('batch replace error failure')).toBe(true);
});

test('one-to-many processor batch replace pushes create success id', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, []);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.batchProcessRelationUpdate(
    ['p1'],
    [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: childCtor,
        inverseField: 'ParentId',
        operations: [{ Name: 'new-child' }] as unknown as Parameters<OneToManyProcessor['batchProcessRelationUpdate']>[1][number]['operations'],
      },
    ]
  );

  expect(result.errors).toEqual([]);
  expect(result.success.map(item => item.entityId)).toEqual(['CREATED-1']);
});

test('one-to-many processor batch update reports missing id and foreign record and wraps non-error search failures', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'other-parent', Name: 'foreign' }]);
  const originalSearch = store.repo.search;
  store.repo.search = async (condition: unknown) => {
    if (condition && typeof condition === 'object' && Array.isArray((condition as { And?: unknown[] }).And)) {
      const first = (condition as { And: unknown[] }).And[0];
      if (Array.isArray(first) && first[2] === 'explode') {
        throw 'batch update search string failure';
      }
    }
    return originalSearch(condition);
  };

  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.batchProcessRelationUpdate(
    ['p1'],
    [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: childCtor,
        inverseField: 'ParentId',
        operations: {
          update: [{ Name: 'missing-id' } as any, { Id: 'c1', Name: 'foreign' }, { Id: 'explode', Name: 'boom' }],
        },
      },
    ]
  );

  expect(result.success).toEqual([]);
  expect(result.errors.some(item => String(item.error?.message || '').includes('Could not extract Id from update item'))).toBe(true);
  expect(result.errors.some(item => String(item.error?.message || '').includes('Item does not exist or does not belong to the parent entity'))).toBe(true);
  expect(result.errors.some(item => String(item.error?.message || '').includes('batch update search string failure'))).toBe(true);
});

test('one-to-many processor batch replace classifies proxy extraction failures into batch errors', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, []);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const throwStringItem = new Proxy(
    { Name: 'bad-string' },
    {
      has(_target, key) {
        if (key === 'Id') throw 'proxy string failure';
        return false;
      },
    }
  );
  const throwErrorItem = new Proxy(
    { Name: 'bad-error' },
    {
      has(_target, key) {
        if (key === 'Id') throw new Error('proxy error failure');
        return false;
      },
    }
  );

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.batchProcessRelationUpdate(
    ['p1'],
    [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: childCtor,
        inverseField: 'ParentId',
        operations: [throwStringItem as any, throwErrorItem as any],
      },
    ]
  );

  expect(result.success).toEqual([]);
  expect(result.errors.length).toBeGreaterThanOrEqual(1);
  const mergedMessages = result.errors.map(item => String(item.error?.message || '')).join(' | ');
  expect(mergedMessages.includes('proxy string failure') || mergedMessages.includes('proxy error failure')).toBe(true);
});

test('one-to-many processor batch replace skips empty created id from success list', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, []);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const originalCreate = OneToManyProcessorChild.Create;
  OneToManyProcessorChild.Create = (async () => '' as any) as unknown as typeof OneToManyProcessorChild.Create;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'OneToMany',
          fieldName: 'Lines',
          targetModel: childCtor,
          inverseField: 'ParentId',
          operations: [{ Name: 'new-child' }] as unknown as Parameters<OneToManyProcessor['batchProcessRelationUpdate']>[1][number]['operations'],
        },
      ]
    );

    expect(result.success).toEqual([]);
    expect(result.errors).toEqual([]);
  } finally {
    OneToManyProcessorChild.Create = originalCreate;
  }
});

test('one-to-many processor object replace keeps entityIds empty when update and create return falsy in existing/new branches', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'old' }]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const originalUpdateById = OneToManyProcessorChild.UpdateById;
  const originalCreate = OneToManyProcessorChild.Create;
  OneToManyProcessorChild.UpdateById = (async () => null as any) as unknown as typeof OneToManyProcessorChild.UpdateById;
  OneToManyProcessorChild.Create = (async () => '' as any) as unknown as typeof OneToManyProcessorChild.Create;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'OneToMany',
      fieldName: 'Lines',
      targetModel: childCtor,
      inverseField: 'ParentId',
      operations: {
        replace: [{ Id: 'c1', Name: 'updated' }, { Name: 'new-child' }] as unknown as Array<{ Id?: string; Name: string }>,
      },
    } as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]);

    expect(result.entityIds).toEqual([]);
    expect(result.errors).toEqual([]);
  } finally {
    OneToManyProcessorChild.UpdateById = originalUpdateById;
    OneToManyProcessorChild.Create = originalCreate;
  }
});

test('one-to-many processor array replace keeps Error instance from helper throw', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'row-1' }]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const originalUpdateById = OneToManyProcessorChild.UpdateById;
  OneToManyProcessorChild.UpdateById = (async () => {
    throw new Error('array replace error branch');
  }) as unknown as typeof OneToManyProcessorChild.UpdateById;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'OneToMany',
      fieldName: 'Lines',
      targetModel: childCtor,
      inverseField: 'ParentId',
      operations: [{ Id: 'c1', Name: 'u1' }] as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]['operations'],
    });

    expect(result.entityIds).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.message || '')).toBe('array replace error branch');
  } finally {
    OneToManyProcessorChild.UpdateById = originalUpdateById;
  }
});

test('one-to-many processor object replace pushes success id for re-parent update branch', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [{ Id: 'c9', ParentId: 'other-parent', Name: 'foreign' }]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.processRelationUpdate('p1', {
    type: 'OneToMany',
    fieldName: 'Lines',
    targetModel: childCtor,
    inverseField: 'ParentId',
    operations: {
      replace: [{ Id: 'c9', Name: 're-parented' } as any],
    },
  } as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]);

  expect(result.errors).toEqual([]);
  expect(result.entityIds).toEqual(['c9']);
});

test('one-to-many processor object replace keeps Error instance from helper throw', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'row-1' }]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const originalUpdateById = OneToManyProcessorChild.UpdateById;
  OneToManyProcessorChild.UpdateById = (async () => {
    throw new Error('object replace error branch');
  }) as unknown as typeof OneToManyProcessorChild.UpdateById;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.processRelationUpdate('p1', {
      type: 'OneToMany',
      fieldName: 'Lines',
      targetModel: childCtor,
      inverseField: 'ParentId',
      operations: {
        replace: [{ Id: 'c1', Name: 'u1' } as any],
      },
    } as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]);

    expect(result.entityIds).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.message || '')).toBe('object replace error branch');
  } finally {
    OneToManyProcessorChild.UpdateById = originalUpdateById;
  }
});

test('one-to-many processor object delete keeps Error instance from single-item strategy throw', async () => {
  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'row' }]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    applySingleItemDeleteStrategy: (...args: any[]) => Promise<{ success: boolean; error?: Error }>;
    processRelationUpdate: OneToManyProcessor['processRelationUpdate'];
  };
  const originalSingleDelete = processor.applySingleItemDeleteStrategy;
  processor.applySingleItemDeleteStrategy = (async () => {
    throw new Error('object delete error branch');
  }) as any;

  try {
    const result = await processor.processRelationUpdate('p1', {
      type: 'OneToMany',
      fieldName: 'Lines',
      targetModel: childCtor,
      inverseField: 'ParentId',
      operations: {
        delete: [{ Id: 'c1' }],
      },
    } as unknown as Parameters<OneToManyProcessor['processRelationUpdate']>[1]);

    expect(result.entityIds).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.message || '')).toBe('object delete error branch');
  } finally {
    processor.applySingleItemDeleteStrategy = originalSingleDelete;
  }
});

test('one-to-many processor batch replace catches both Error and non-Error extraction failures', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, []);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const throwStringItem = new Proxy(
    { Name: 'bad-string' },
    {
      has(_target, key) {
        if (key === 'Id') throw 'batch replace catch string branch';
        return false;
      },
    }
  );
  const throwErrorItem = new Proxy(
    { Name: 'bad-error' },
    {
      has(_target, key) {
        if (key === 'Id') throw new Error('batch replace catch error branch');
        return false;
      },
    }
  );

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.batchProcessRelationUpdate(
    ['p1'],
    [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: childCtor,
        inverseField: 'ParentId',
        operations: [throwStringItem as any, throwErrorItem as any],
      },
    ]
  );

  expect(result.success).toEqual([]);
  const mergedMessages = result.errors.map(item => String(item.error?.message || '')).join(' | ');
  expect(mergedMessages.includes('batch replace catch string branch') || mergedMessages.includes('batch replace catch error branch')).toBe(true);
});

test('one-to-many processor batch delete records missing id item in grouped delete path', async () => {
  const store = createRelationRepository(childCtor, []);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.batchProcessRelationUpdate(
    ['p1'],
    [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: childCtor,
        inverseField: 'ParentId',
        operations: { delete: [{ Name: 'missing-id' } as any] },
      },
    ]
  );

  expect(result.success).toEqual([]);
  expect(result.errors.length).toBe(1);
  expect(String(result.errors[0]?.error?.message || '').includes('Could not extract Id from delete item')).toBe(true);
});

test('one-to-many processor batch update pushes success id on update branch', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [{ Id: 'c1', ParentId: 'p1', Name: 'row' }]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.batchProcessRelationUpdate(
    ['p1'],
    [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: childCtor,
        inverseField: 'ParentId',
        operations: {
          update: [{ Id: 'c1', Name: 'updated' } as any],
        },
      },
    ]
  );

  expect(result.errors).toEqual([]);
  expect(result.success.map(item => item.entityId)).toEqual(['c1']);
});

test('one-to-many processor batch update catches Error and non-Error update helper failures', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, [
    { Id: 'e1', ParentId: 'p1', Name: 'row-e' },
    { Id: 's1', ParentId: 'p1', Name: 'row-s' },
  ]);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const originalUpdateById = OneToManyProcessorChild.UpdateById;
  OneToManyProcessorChild.UpdateById = (async (id: string) => {
    if (id === 'e1') throw new Error('batch update error branch');
    throw 'batch update string branch';
  }) as unknown as typeof OneToManyProcessorChild.UpdateById;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'OneToMany',
          fieldName: 'Lines',
          targetModel: childCtor,
          inverseField: 'ParentId',
          operations: {
            update: [{ Id: 'e1', Name: 'x' } as any, { Id: 's1', Name: 'y' } as any],
          },
        },
      ]
    );

    expect(result.success).toEqual([]);
    expect(result.errors.some(item => String(item.error?.message || '').includes('batch update error branch'))).toBe(true);
    expect(result.errors.some(item => String(item.error?.message || '').includes('batch update string branch'))).toBe(true);
  } finally {
    OneToManyProcessorChild.UpdateById = originalUpdateById;
  }
});

test('one-to-many processor batch create keeps Error instance from create helper catch', async () => {
  const store = createRelationRepository(childCtor, []);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const originalCreate = OneToManyProcessorChild.Create;
  OneToManyProcessorChild.Create = (async () => {
    throw new Error('batch create error branch');
  }) as unknown as typeof OneToManyProcessorChild.Create;

  try {
    const processor = new OneToManyProcessor(parentCtor);
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'OneToMany',
          fieldName: 'Lines',
          targetModel: childCtor,
          inverseField: 'ParentId',
          operations: {
            create: [{ Name: 'new-batch-row' } as any],
          },
        },
      ]
    );

    expect(result.success).toEqual([]);
    expect(result.errors.length).toBe(1);
    expect(String(result.errors[0]?.error?.message || '')).toBe('batch create error branch');
  } finally {
    OneToManyProcessorChild.Create = originalCreate;
  }
});

test('one-to-many processor batchRemoveAssociationsWithDiff uses replacement fallback and cascade branch', async () => {
  const store = createRelationRepository(cascadeChildCtor, [
    { Id: 'c1', ParentId: 'p1', Name: 'drop' },
    { Id: 'c2', ParentId: 'p2', Name: 'keep' },
  ]);
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    batchRemoveAssociationsWithDiff: (...args: any[]) => Promise<Map<string, { existingIds: Set<string>; removedIds: string[] }>>;
  };

  const result = await processor.batchRemoveAssociationsWithDiff(store.repo, cascadeChildCtor, 'ParentId', ['p1', 'p2'], new Map([['p1', [{ Id: 'c1' }]]]));

  expect(Array.from(result.get('p2')?.removedIds || [])).toEqual(['c2']);
  expect(store.deleteCalls.length).toBe(1);
  expect(store.deleteCalls[0]).toEqual(['c2']);
});

test('one-to-many processor applyDeleteStrategy throws for NO ACTION policy with affected rows', async () => {
  const store = createRelationRepository(noActionChildCtor, [{ Id: 'n1', ParentId: 'p1', Name: 'locked' }]);
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    applyDeleteStrategy: (repository: any, targetModel: any, inverseField: string, parentId: string) => Promise<void>;
  };

  let message = '';
  try {
    await processor.applyDeleteStrategy(store.repo, noActionChildCtor, 'ParentId', 'p1');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('onDelete is set to NO ACTION')).toBe(true);
});

test('one-to-many processor groupItemsByDeleteStrategy covers RESTRICT grouping branch', async () => {
  const store = createRelationRepository(restrictChildCtor, [{ Id: 'r1', ParentId: 'p1', Name: 'locked' }]);
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    groupItemsByDeleteStrategy: (
      repository: any,
      targetModel: any,
      inverseField: string,
      items: Array<{ id: string; parentId: string }>
    ) => Promise<{ cascadeIds: string[]; setNullIds: string[]; restrictIds: string[]; notFoundIds: string[] }>;
  };

  const grouped = await processor.groupItemsByDeleteStrategy(store.repo, restrictChildCtor, 'ParentId', [{ id: 'r1', parentId: 'p1' }]);
  expect(grouped.restrictIds).toEqual(['r1']);
  expect(grouped.cascadeIds).toEqual([]);
  expect(grouped.setNullIds).toEqual([]);
});

test('one-to-many processor prepareForUpdate skips relation push when targetModel is missing but relation object exists', async () => {
  const processor = new OneToManyProcessor(prepareParentCtor);
  const meta = MetadataStorage.instance.getModelMetadata(prepareParentCtor as any) as any;
  const linesField = meta.fields.get('Lines') as any;
  const originalRelation = linesField.relation;

  linesField.relation = {
    targetModel: undefined,
    inverseField: 'ParentId',
  };

  try {
    const result = await processor.prepareForUpdate(
      {
        Name: 'parent-b49',
        Lines: [{ Name: 'line-1' }],
      },
      ['Lines']
    );

    expect(result.processedValue).toEqual({ Name: 'parent-b49' });
    expect(result.relations.oneToManyRelations).toEqual([]);
    expect(Array.from(result.relations.touchedCollections || [])).toEqual([]);
  } finally {
    linesField.relation = originalRelation;
  }
});

test('one-to-many processor prepareForUpdate pushes relation when config is complete and value is non-null', async () => {
  const processor = new OneToManyProcessor(prepareParentCtor);
  const result = await processor.prepareForUpdate(
    {
      Name: 'parent-b49-ok',
      Lines: [{ Name: 'line-ok' }],
    },
    ['Lines']
  );

  expect(result.processedValue).toEqual({ Name: 'parent-b49-ok' });
  expect(result.relations.oneToManyRelations.length).toBe(1);
  expect(result.relations.oneToManyRelations[0]?.fieldName).toBe('Lines');
  expect(result.relations.oneToManyRelations[0]?.inverseField).toBe('ParentId');
  expect(Array.from(result.relations.touchedCollections || [])).toEqual(['Lines']);
});

test('one-to-many processor prepareForUpdate skips when relation config is missing entirely', async () => {
  const processor = new OneToManyProcessor(prepareParentCtor);
  const meta = MetadataStorage.instance.getModelMetadata(prepareParentCtor as any) as any;
  const linesField = meta.fields.get('Lines') as any;
  const originalRelation = linesField.relation;

  linesField.relation = undefined;
  try {
    const result = await processor.prepareForUpdate({ Name: 'p3', Lines: [{ Name: 'line-1' }] }, ['Lines']);
    expect(result.processedValue).toEqual({ Name: 'p3' });
    expect(result.relations.oneToManyRelations).toEqual([]);
    expect(Array.from(result.relations.touchedCollections || [])).toEqual([]);
  } finally {
    linesField.relation = originalRelation;
  }
});

test('one-to-many processor batch replace at line 319 catches both Error and non-Error from strip helper', async () => {
  resetChildMetadata();

  const store = createRelationRepository(childCtor, []);
  RepositoryFactory.setRepository(childCtor, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    stripChildComputeFields: (ctor: typeof BaseModel, row: any) => any;
    batchProcessRelationUpdate: OneToManyProcessor['batchProcessRelationUpdate'];
  };

  const originalStrip = processor.stripChildComputeFields;
  let call = 0;
  processor.stripChildComputeFields = ((...args: any[]) => {
    call += 1;
    if (call === 1) throw new Error('line319 error branch');
    if (call === 2) throw 'line319 string branch';
    return originalStrip.apply(processor, args as any);
  }) as any;

  try {
    const result = await processor.batchProcessRelationUpdate(
      ['p1'],
      [
        {
          type: 'OneToMany',
          fieldName: 'Lines',
          targetModel: childCtor,
          inverseField: 'ParentId',
          operations: [{ Name: 'x1' } as any, { Name: 'x2' } as any],
        },
      ]
    );

    expect(result.success).toEqual([]);
    expect(result.errors.some(item => String(item.error?.message || '').includes('line319 error branch'))).toBe(true);
    expect(result.errors.some(item => String(item.error?.message || '').includes('line319 string branch'))).toBe(true);
  } finally {
    processor.stripChildComputeFields = originalStrip;
  }
});

test('one-to-many processor batch result uses unknown target fallback on normal return path', async () => {
  const targetWithoutName = {} as any;
  const store = createRelationRepository(childCtor, []);
  RepositoryFactory.setRepository(targetWithoutName, store.repo as unknown as Parameters<typeof RepositoryFactory.setRepository>[1]);

  const processor = new OneToManyProcessor(parentCtor);
  const result = await processor.batchProcessRelationUpdate(
    ['p1'],
    [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: targetWithoutName,
        inverseField: 'ParentId',
        operations: { create: [] },
      },
    ]
  );

  expect(result.success).toEqual([]);
  expect(result.errors).toEqual([]);
  expect(result.summary.relationType).toBe('OneToMany');
});

test('one-to-many processor applyDeleteStrategy throws for RESTRICT policy with affected rows', async () => {
  const store = createRelationRepository(restrictChildCtor, [{ Id: 'r2', ParentId: 'p1', Name: 'locked' }]);
  const processor = new OneToManyProcessor(parentCtor) as unknown as {
    applyDeleteStrategy: (repository: any, targetModel: any, inverseField: string, parentId: string) => Promise<void>;
  };

  let message = '';
  try {
    await processor.applyDeleteStrategy(store.repo, restrictChildCtor, 'ParentId', 'p1');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('onDelete is set to RESTRICT')).toBe(true);
});
