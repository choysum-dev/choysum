// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { Field, Model } from '../decorator';
import { RelationProcessor } from './processor';
import type { RelationFieldType } from './types';

@Model('test.RelationProcessorJoin')
class RelationProcessorJoin extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  OwnerId?: string;

  @Field({ type: 'varchar', column: { size: 64 } })
  TagId?: string;
}

@Model('test.RelationProcessorTarget')
class RelationProcessorTarget extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;
}

@Model('test.RelationProcessorParent')
class RelationProcessorParent extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => RelationProcessorTarget }, column: {} })
  OwnerId?: RelationProcessorTarget | null;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => RelationProcessorChild, inverseField: 'ParentId' as never },
  })
  Lines?: RelationProcessorChild[];

  @Field({
    type: 'ManyToMany',
    relation: {
      joinModel: () => RelationProcessorJoin,
      targetModel: () => RelationProcessorTarget,
      joinField: 'OwnerId' as never,
      inverseJoinField: 'TagId' as never,
    },
  })
  Tags?: RelationProcessorTarget[];

  @Field({
    type: 'ManyToMany',
    relation: {
      joinModel: () => RelationProcessorJoin,
      targetModel: () => RelationProcessorTarget,
    } as any,
  })
  BrokenTags?: RelationProcessorTarget[];
}

@Model('test.RelationProcessorChild')
class RelationProcessorChild extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => RelationProcessorParent, onDelete: 'CASCADE' },
    column: {},
  })
  ParentId?: RelationProcessorParent | null;
}

@Model('test.RelationProcessorRestrictChild')
class RelationProcessorRestrictChild extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => RelationProcessorParent, onDelete: 'RESTRICT' },
    column: {},
  })
  ParentId?: RelationProcessorParent | null;
}

@Model('test.RelationProcessorNoPolicyChild')
class RelationProcessorNoPolicyChild extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => RelationProcessorParent },
    column: {},
  })
  ParentId?: RelationProcessorParent | null;
}

@Model('test.RelationProcessorPlainChild')
class RelationProcessorPlainChild extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  ParentId?: string;
}

@Model('test.RelationProcessorInvalidParent')
class RelationProcessorInvalidParent extends BaseModel {
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => RelationProcessorChild } as any,
  })
  BrokenLines?: RelationProcessorChild[];
}

@Model('test.RelationProcessorCreateTarget')
class RelationProcessorCreateTarget extends BaseModel {
  static calls = 0;

  static override async Create<T extends BaseModel>(this: { new (...args: any[]): T } & typeof BaseModel, value: Record<string, any>): Promise<T> {
    RelationProcessorCreateTarget.calls += 1;
    return {
      Id: `crt-${RelationProcessorCreateTarget.calls}`,
      ...value,
    } as T;
  }
}

@Model('test.RelationProcessorCreateFailTarget')
class RelationProcessorCreateFailTarget extends BaseModel {
  static override async Create<T extends BaseModel>(): Promise<T> {
    throw new Error('create failed');
  }
}

class DummyProcessor extends RelationProcessor<any> {
  constructor(modelClass: any = RelationProcessorParent) {
    super(modelClass);
  }

  protected override extractId(value: unknown): string | null {
    if (value && typeof value === 'object' && (value as any).ThrowExtract) {
      throw new Error('extract failed');
    }
    return super.extractId(value);
  }

  protected override ensureObject(value: unknown): Record<string, any> {
    if (value && typeof value === 'object' && (value as any).ThrowEnsure) {
      throw new Error('ensure failed');
    }
    return super.ensureObject(value);
  }

  async prepareForCreate(value: Record<string, any>): Promise<any> {
    return {
      processedValue: { ...value },
      relations: {
        oneToManyRelations: [],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(),
      },
    };
  }

  async prepareForUpdate(value: Record<string, any>): Promise<any> {
    return {
      processedValue: { ...value },
      relations: {
        oneToManyRelations: [],
        manyToManyRelations: [],
        touchedCollections: new Set<string>(),
      },
    };
  }

  async processRelationUpdate(): Promise<any> {
    return {
      affectedCount: 0,
      entityIds: [],
      errors: [],
      targetModel: RelationProcessorTarget,
      relationType: 'OneToMany',
    };
  }

  async batchProcessRelationUpdate(): Promise<any> {
    return {
      success: [],
      errors: [],
      summary: {
        totalOperations: 0,
        successfulOperations: 0,
        failedOperations: 0,
        relationType: 'OneToMany',
      },
    };
  }

  exposeExtractRelations(value: Record<string, any>) {
    return this.extractRelations(value);
  }

  exposeGroup(parentIds: string[], operations: any[]) {
    return this.groupOperationsByTarget(parentIds, operations as any);
  }

  exposeClassify(parentMap: Map<string, any>) {
    return this.classifyOperations(parentMap);
  }

  exposePrepareBatchCreateEntities(parentMap: Map<string, any[]>, foreignKeyField: string) {
    return this.prepareBatchCreateEntities(parentMap, foreignKeyField);
  }

  async exposeBatchDeleteRecords(repository: any, idsToDelete: string[]) {
    await this.batchDeleteRecords(repository as any, idsToDelete);
  }

  exposeCollectIds(items: any[]) {
    return this.collectIds(items);
  }

  exposePrepareBatchUpdateOperations(items: any[]) {
    return this.prepareBatchUpdateOperations(items);
  }

  exposeCreateBatchResult(successIds: string[], errors: Error[], relationType: RelationFieldType, targetModelName: string, joinModelName?: string) {
    return this.createBatchResult(successIds, errors, relationType, targetModelName, joinModelName);
  }

  exposeEnsureObject(value: unknown) {
    return super.ensureObject(value);
  }

  exposeExtractId(value: unknown) {
    return super.extractId(value);
  }

  exposeIsModelLike(value: unknown) {
    return this.isModelLike(value);
  }

  exposeIsIdRelationItem(value: unknown) {
    return this.isIdRelationItem(value);
  }

  exposeIsModelRelationItem(value: unknown) {
    return this.isModelRelationItem(value);
  }

  async exposeGetOrCreateId(value: unknown, targetClass: any) {
    return await this.getOrCreateId(value, targetClass);
  }

  exposeCalculateRelationDiff(existingIds: string[], newItems: any[]) {
    return this.calculateRelationDiff(existingIds, newItems);
  }

  exposeGetOnDeletePolicy(targetClass: any, foreignKeyField: string) {
    return this.getOnDeletePolicy(targetClass, foreignKeyField);
  }

  async exposeBatchRemoveAssociations(repository: any, foreignKeyField: string, parentIds: string[], targetClass?: any, newItemsMap?: Map<string, any[]>) {
    return await this.batchRemoveAssociations(repository as any, foreignKeyField, parentIds, targetClass, newItemsMap);
  }

  exposeMarkCollectionTouched(relations: any, fieldName: string) {
    this.markCollectionTouched(relations, fieldName);
  }
}

function matchCondition(row: Record<string, any>, condition: any): boolean {
  if (!Array.isArray(condition)) return false;
  const [field, op, value] = condition;
  const left = row[String(field)];
  if (op === '=') return left === value;
  if (String(op).toLowerCase() === 'in') return Array.isArray(value) && value.includes(left);
  return false;
}

function createAssociationRepo(seedRows: Array<{ Id: string; ParentId: string }>) {
  const rows = seedRows.map(row => ({ ...row }));
  const calls = {
    search: [] as any[],
    update: [] as any[],
    delete: [] as any[],
  };

  return {
    calls,
    repo: {
      async search(condition: any) {
        calls.search.push(condition);
        return rows.filter(row => matchCondition(row, condition)).map(row => ({ ...row }));
      },
      async update(values: Record<string, any>, condition: any) {
        calls.update.push({ values: { ...values }, condition });
      },
      async delete(condition: any) {
        calls.delete.push(condition);
      },
    },
  };
}

test('relation processor extractRelations handles valid/incomplete configs and touched collections', () => {
  const processor = new DummyProcessor(RelationProcessorParent as any);
  const extracted = processor.exposeExtractRelations({
    OwnerId: 'owner-1',
    Lines: [{ Id: 'c1' }],
    Tags: [{ Id: 't1' }],
    BrokenTags: [{ Id: 't2' }],
  });

  expect(extracted.oneToManyRelations.length).toBe(1);
  expect(extracted.manyToManyRelations.length).toBe(1);
  expect(Array.from(extracted.touchedCollections || []).sort()).toEqual(['Lines', 'Tags']);

  const invalidProcessor = new DummyProcessor(RelationProcessorInvalidParent as any);
  let message = '';
  try {
    invalidProcessor.exposeExtractRelations({ BrokenLines: [{ Id: 'x1' }] });
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('Missing OneToMany configuration')).toBe(true);
});

test('relation processor grouping and classification cover mismatch, unsupported and mixed operation forms', () => {
  const processor = new DummyProcessor();

  let mismatchMessage = '';
  try {
    processor.exposeGroup(['p1'], []);
  } catch (error) {
    mismatchMessage = String((error as Error)?.message || error);
  }
  expect(mismatchMessage.includes('Parent entity Id array length must match relation operation array length')).toBe(true);

  const grouped = processor.exposeGroup(['p1', 'p2', 'p3'], [
    {
      type: 'OneToMany',
      fieldName: 'Lines',
      targetModel: RelationProcessorChild,
      inverseField: 'ParentId',
      operations: [{ Id: 'c1' }],
    },
    {
      type: 'Unsupported',
      fieldName: 'Broken',
      operations: [],
    },
    {
      type: 'ManyToMany',
      fieldName: 'Tags',
      joinModel: RelationProcessorJoin,
      targetModel: RelationProcessorTarget,
      joinField: 'OwnerId',
      inverseJoinField: 'TagId',
      operations: [{ Id: 't1' }],
    },
  ] as any[]);

  expect(grouped.size).toBe(2);
  expect(grouped.get('Lines_RelationProcessorChild')?.parentMap.get('p1')).toEqual([{ Id: 'c1' }]);
  expect(grouped.get('Tags_RelationProcessorJoin_RelationProcessorTarget')?.parentMap.get('p3')).toEqual([{ Id: 't1' }]);

  const classified = processor.exposeClassify(
    new Map<string, any>([
      ['p1', [{ Id: 'r1' }]],
      [
        'p2',
        {
          replace: [{ Id: 'r2' }],
          create: [{ Name: 'new' }],
          update: [{ Id: 'u1', Name: 'update' }],
          delete: [{ Id: 'd1' }],
        },
      ],
      ['p3', null],
    ])
  );

  expect(classified.replace.get('p1')).toEqual([{ Id: 'r1' }]);
  expect(classified.replace.get('p2')).toEqual([{ Id: 'r2' }]);
  expect(classified.create.get('p2')).toEqual([{ Name: 'new' }]);
  expect(classified.update.get('p2')).toEqual([{ Id: 'u1', Name: 'update' }]);
  expect(classified.delete.get('p2')).toEqual([{ Id: 'd1' }]);
});

test('relation processor batch helpers cover create/delete/id collection/update prep and result assembly', async () => {
  const processor = new DummyProcessor();
  const entities = processor.exposePrepareBatchCreateEntities(
    new Map<string, any[]>([
      ['p1', ['id-only', { Name: 'n1' }]],
      ['p2', [{ Name: 'n2' }]],
    ]),
    'ParentId'
  );

  expect(entities).toEqual([
    { Id: 'id-only', ParentId: 'p1' },
    { Name: 'n1', ParentId: 'p1' },
    { Name: 'n2', ParentId: 'p2' },
  ]);

  const repoCalls: any[] = [];
  const repo = {
    async delete(condition: any) {
      repoCalls.push(condition);
    },
  };

  await processor.exposeBatchDeleteRecords(repo, []);
  await processor.exposeBatchDeleteRecords(repo, ['a1']);
  await processor.exposeBatchDeleteRecords(repo, ['a1', 'a2']);
  expect(repoCalls).toEqual([
    ['Id', '=', 'a1'],
    ['Id', 'in', ['a1', 'a2']],
  ]);

  const collected = processor.exposeCollectIds([{ Id: 'c1' }, { ThrowExtract: true }, {}, 'c2']);
  expect(collected.ids).toEqual(['c1', 'c2']);
  expect(collected.errors.length).toBe(2);
  expect(collected.errors.some(error => String(error.message || '').includes('extract failed'))).toBe(true);

  const updates = processor.exposePrepareBatchUpdateOperations([{ Id: 'u1', Name: 'n1' }, { Id: 'u2', ThrowEnsure: true }, { Name: 'missing' }]);
  expect(Array.from(updates.updateOps.keys())).toEqual(['u1']);
  expect(updates.errors.length).toBe(2);
  expect(updates.errors.some(error => String(error.message || '').includes('Update operation is missing Id'))).toBe(true);
  expect(updates.errors.some(error => String(error.message || '').includes('ensure failed'))).toBe(true);

  const result = processor.exposeCreateBatchResult(['ok-1'], [new Error('bad')], 'OneToMany', 'TargetModel', 'JoinModel');
  expect(result.summary).toEqual({
    totalOperations: 2,
    successfulOperations: 1,
    failedOperations: 1,
    relationType: 'OneToMany',
  });
  expect(result.success[0]).toEqual({ entityId: 'ok-1', targetModel: 'TargetModel', joinModel: 'JoinModel' });
  expect(result.errors[0]?.targetModel).toBe('TargetModel');
});

test('relation processor value helpers and getOrCreateId branches are handled', async () => {
  const processor = new DummyProcessor();

  const modelInstance = Object.create(RelationProcessorTarget.prototype) as RelationProcessorTarget;
  (modelInstance as any).Id = 'model-1';
  modelInstance.Name = 'demo';
  (modelInstance as any).toEntity = () => ({ Id: 'model-1', Name: 'demo' });

  expect(processor.exposeEnsureObject('id-1')).toEqual({ Id: 'id-1' });
  expect(processor.exposeEnsureObject(modelInstance).Id).toBe('model-1');
  expect(processor.exposeEnsureObject({ Name: 'obj' })).toEqual({ Name: 'obj' });
  expect(processor.exposeEnsureObject(42)).toEqual({});

  expect(processor.exposeExtractId(null)).toBe(null);
  expect(processor.exposeExtractId('id-1')).toBe('id-1');
  expect(processor.exposeExtractId({ Id: 'id-2' })).toBe('id-2');
  expect(processor.exposeExtractId(modelInstance)).toBe('model-1');
  expect(processor.exposeExtractId({ Id: 1 })).toBe(null);

  expect(processor.exposeIsModelLike(modelInstance)).toBe(true);
  expect(processor.exposeIsModelLike({ Id: 'id-3' })).toBe(true);
  expect(processor.exposeIsModelLike({ Name: 'obj' })).toBe(true);
  expect(processor.exposeIsModelLike([1, 2])).toBe(false);

  expect(processor.exposeIsIdRelationItem('id-4')).toBe(true);
  expect(processor.exposeIsIdRelationItem({ Id: 'id-5' })).toBe(true);
  expect(processor.exposeIsIdRelationItem({ Id: 'id-6', Name: 'x' })).toBe(false);

  expect(processor.exposeIsModelRelationItem(modelInstance)).toBe(true);
  expect(processor.exposeIsModelRelationItem({ Name: 'obj' })).toBe(true);
  expect(processor.exposeIsModelRelationItem({ Id: 'id-only' })).toBe(false);

  RelationProcessorCreateTarget.calls = 0;
  expect(await processor.exposeGetOrCreateId('id-7', RelationProcessorCreateTarget as any)).toBe('id-7');
  expect(await processor.exposeGetOrCreateId({ Name: 'created' }, RelationProcessorCreateTarget as any)).toBe('crt-1');

  let createFailMessage = '';
  try {
    await processor.exposeGetOrCreateId({ Name: 'bad' }, RelationProcessorCreateFailTarget as any);
  } catch (error) {
    createFailMessage = String((error as Error)?.message || error);
  }
  expect(createFailMessage.includes('Failed to create relation entity')).toBe(true);

  let invalidMessage = '';
  try {
    await processor.exposeGetOrCreateId(123, RelationProcessorCreateTarget as any);
  } catch (error) {
    invalidMessage = String((error as Error)?.message || error);
  }
  expect(invalidMessage.includes('Invalid relation item')).toBe(true);
});

test('relation processor diff and association removal cover delete policy branches', async () => {
  const processor = new DummyProcessor();

  const diff = processor.exposeCalculateRelationDiff(['a', 'b', 'c'], [{ Id: 'b', Name: 'keep' }, { Id: 'x' }, { Name: 'new' }, { Id: 'b' }]);
  expect(Array.from(diff.toKeep)).toEqual(['b']);
  expect(diff.toRemove).toEqual(['a', 'c']);
  expect(diff.toAdd.length).toBe(2);
  expect(Array.from(diff.toUpdate.keys())).toEqual(['b']);

  expect(processor.exposeGetOnDeletePolicy(RelationProcessorChild as any, 'ParentId')).toBe('CASCADE');
  expect(processor.exposeGetOnDeletePolicy(RelationProcessorNoPolicyChild as any, 'ParentId')).toBe('SET NULL');
  expect(processor.exposeGetOnDeletePolicy(RelationProcessorPlainChild as any, 'ParentId')).toBe('SET NULL');

  const empty = await processor.exposeBatchRemoveAssociations(createAssociationRepo([]).repo, 'ParentId', []);
  expect(empty.size).toBe(0);

  const emptySearchStore = createAssociationRepo([{ Id: 'x1', ParentId: 'other' }]);
  const emptySearchResult = await processor.exposeBatchRemoveAssociations(emptySearchStore.repo, 'ParentId', ['p1']);
  expect(emptySearchResult.get('p1')).toEqual({ existingIds: [], removedIds: [] });

  const clearCascadeStore = createAssociationRepo([
    { Id: 'c1', ParentId: 'p1' },
    { Id: 'c2', ParentId: 'p1' },
  ]);
  const clearCascadeResult = await processor.exposeBatchRemoveAssociations(clearCascadeStore.repo, 'ParentId', ['p1'], RelationProcessorChild as any);
  expect(clearCascadeStore.calls.delete).toEqual([['Id', 'in', ['c1', 'c2']]]);
  expect(clearCascadeResult.get('p1')?.removedIds).toEqual(['c1', 'c2']);

  const clearDefaultStore = createAssociationRepo([{ Id: 'c3', ParentId: 'p1' }]);
  await processor.exposeBatchRemoveAssociations(clearDefaultStore.repo, 'ParentId', ['p1']);
  expect(clearDefaultStore.calls.update).toEqual([
    {
      values: { ParentId: null },
      condition: ['ParentId', '=', 'p1'],
    },
  ]);

  const diffSetNullStore = createAssociationRepo([
    { Id: 'c4', ParentId: 'p1' },
    { Id: 'c5', ParentId: 'p1' },
  ]);
  const diffSetNullResult = await processor.exposeBatchRemoveAssociations(
    diffSetNullStore.repo,
    'ParentId',
    ['p1'],
    RelationProcessorNoPolicyChild as any,
    new Map([['p1', [{ Id: 'c5' }]]])
  );
  expect(diffSetNullStore.calls.update).toEqual([
    {
      values: { ParentId: null },
      condition: ['Id', 'in', ['c4']],
    },
  ]);
  expect(diffSetNullResult.get('p1')?.removedIds).toEqual(['c4']);

  const diffRestrictStore = createAssociationRepo([{ Id: 'r1', ParentId: 'p1' }]);
  let restrictMessage = '';
  try {
    await processor.exposeBatchRemoveAssociations(diffRestrictStore.repo, 'ParentId', ['p1'], RelationProcessorRestrictChild as any, new Map([['p1', []]]));
  } catch (error) {
    restrictMessage = String((error as Error)?.message || error);
  }
  expect(restrictMessage.includes('RESTRICT')).toBe(true);

  const relations = {
    oneToManyRelations: [],
    manyToManyRelations: [],
  } as any;
  processor.exposeMarkCollectionTouched(relations, 'Lines');
  expect(Array.from(relations.touchedCollections || [])).toEqual(['Lines']);
});
