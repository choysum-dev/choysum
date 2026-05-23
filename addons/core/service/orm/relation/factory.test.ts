// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../model/model';
import { Field, Model } from '../decorator';
import { MetadataStorage } from '../metadata';
import { RelationFactory } from './factory';
import { ExtractedRelations, RelationArrayMethod, type RelationChangesCollection, type RelationFieldType } from './types';

type ModelCtor<T extends BaseModel> = { new (...args: never[]): T } & typeof BaseModel;

@Model('test.RelationFactoryTarget')
class RelationFactoryTarget extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;
}

@Model('test.RelationFactoryJoin')
class RelationFactoryJoin extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  OwnerId?: string;

  @Field({ type: 'varchar', column: { size: 64 } })
  TagId?: string;
}

@Model('test.RelationFactoryParent')
class RelationFactoryParent extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => RelationFactoryTarget }, column: {} })
  OwnerId?: RelationFactoryTarget | null;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => RelationFactoryTarget, inverseField: 'OwnerId' as never },
  })
  Lines?: RelationFactoryTarget[];

  @Field({
    type: 'ManyToMany',
    relation: {
      joinModel: () => RelationFactoryJoin,
      targetModel: () => RelationFactoryTarget,
      joinField: 'OwnerId' as never,
      inverseJoinField: 'TagId' as never,
    },
  })
  Tags?: RelationFactoryTarget[];
}

const parentCtor = RelationFactoryParent as unknown as ModelCtor<RelationFactoryParent>;

test('relation factory createProcessor caches processor instance by model and relation type', () => {
  RelationFactory.clearCache();

  const first = RelationFactory.createProcessor(parentCtor, 'OneToMany');
  const second = RelationFactory.createProcessor(parentCtor, 'OneToMany');

  expect(first).toBe(second);
  expect(RelationFactory.hasProcessorCached(parentCtor, 'OneToMany')).toBe(true);
});

test('relation factory getProcessorsForModel returns one processor per supported relation type', () => {
  RelationFactory.clearCache();

  const processors = RelationFactory.getProcessorsForModel(parentCtor);

  expect(processors.has('ManyToOne')).toBe(true);
  expect(processors.has('OneToMany')).toBe(true);
  expect(processors.has('ManyToMany')).toBe(true);
  expect(processors.size).toBe(3);
});

test('relation factory prepareForCreate merges processor outputs and touched collections', async () => {
  const originalGetProcessors = RelationFactory.getProcessorsForModel;

  try {
    const manyToOneProcessor = {
      prepareForCreate: async (value: Record<string, any>) => ({
        processedValue: { ...value, OwnerId: 't-1' },
        relations: {
          oneToManyRelations: [],
          manyToManyRelations: [],
          touchedCollections: new Set<string>(),
        },
      }),
    };

    const oneToManyProcessor = {
      prepareForCreate: async (value: Record<string, any>) => ({
        processedValue: { ...value, Lines: undefined },
        relations: {
          oneToManyRelations: [
            {
              type: 'OneToMany',
              fieldName: 'Lines',
              targetModel: RelationFactoryTarget,
              inverseField: 'OwnerId',
              operations: [{ Id: 'l1' }],
            },
          ],
          manyToManyRelations: [],
          touchedCollections: new Set<string>(['Lines']),
        },
      }),
    };

    const manyToManyProcessor = {
      prepareForCreate: async (value: Record<string, any>) => ({
        processedValue: { ...value, Tags: undefined },
        relations: {
          oneToManyRelations: [],
          manyToManyRelations: [
            {
              type: 'ManyToMany',
              fieldName: 'Tags',
              joinModel: RelationFactoryJoin,
              targetModel: RelationFactoryTarget,
              joinField: 'OwnerId',
              inverseJoinField: 'TagId',
              operations: [{ Id: 't2' }],
            },
          ],
          touchedCollections: new Set<string>(['Tags']),
        },
      }),
    };

    RelationFactory.getProcessorsForModel = (() =>
      new Map<RelationFieldType, unknown>([
        ['ManyToOne', manyToOneProcessor],
        ['OneToMany', oneToManyProcessor],
        ['ManyToMany', manyToManyProcessor],
      ])) as unknown as typeof RelationFactory.getProcessorsForModel;

    const result = await RelationFactory.prepareForCreate(parentCtor, {
      Name: 'parent',
      OwnerId: { Name: 'owner' },
      Lines: [{ Name: 'line-1' }],
      Tags: [{ Name: 'tag-1' }],
    });

    expect(result.processedValue).toEqual({
      Name: 'parent',
      OwnerId: 't-1',
      Lines: undefined,
      Tags: undefined,
    });
    expect(result.relations.oneToManyRelations.length).toBe(1);
    expect(result.relations.manyToManyRelations.length).toBe(1);
    expect(Array.from(result.relations.touchedCollections || []).sort()).toEqual(['Lines', 'Tags']);
  } finally {
    RelationFactory.getProcessorsForModel = originalGetProcessors;
  }
});

test('relation factory batchProcessToManyRelations captures processor batch failure as failed summary', async () => {
  const originalCreateProcessor = RelationFactory.createProcessor;

  try {
    const fakeOneToManyProcessor = {
      batchProcessRelationUpdate: async () => {
        throw new Error('batch o2m failed');
      },
    };
    const fakeManyToManyProcessor = {
      batchProcessRelationUpdate: async () => ({
        success: [{ entityId: 'tag-1', targetModel: 'test.RelationFactoryTarget' }],
        errors: [],
        summary: {
          totalOperations: 1,
          successfulOperations: 1,
          failedOperations: 0,
          relationType: 'ManyToMany',
        },
      }),
    };

    RelationFactory.createProcessor = ((_: ModelCtor<BaseModel>, relationType: RelationFieldType) => {
      if (relationType === 'OneToMany') return fakeOneToManyProcessor as unknown as ReturnType<typeof RelationFactory.createProcessor>;
      if (relationType === 'ManyToMany') return fakeManyToManyProcessor as unknown as ReturnType<typeof RelationFactory.createProcessor>;
      throw new Error('unexpected processor type');
    }) as unknown as typeof RelationFactory.createProcessor;

    const relationsList: ExtractedRelations[] = [
      {
        oneToManyRelations: [
          {
            type: 'OneToMany',
            fieldName: 'Lines',
            targetModel: RelationFactoryTarget,
            inverseField: 'OwnerId',
            operations: [{ Id: 'l1' }],
          },
        ],
        manyToManyRelations: [
          {
            type: 'ManyToMany',
            fieldName: 'Tags',
            joinModel: RelationFactoryJoin,
            targetModel: RelationFactoryTarget,
            joinField: 'OwnerId',
            inverseJoinField: 'TagId',
            operations: [{ Id: 'tag-1' }],
          },
        ],
      },
    ];

    const results = await RelationFactory.batchProcessToManyRelations(parentCtor, ['p1'], relationsList);

    expect(results.length).toBe(2);
    expect(results[0]?.summary).toEqual({
      totalOperations: 1,
      successfulOperations: 0,
      failedOperations: 1,
      relationType: 'OneToMany',
    });
    expect(String(results[0]?.errors?.[0]?.error?.message || '').includes('batch o2m failed')).toBe(true);
    expect(results[1]?.summary).toEqual({
      totalOperations: 1,
      successfulOperations: 1,
      failedOperations: 0,
      relationType: 'ManyToMany',
    });
  } finally {
    RelationFactory.createProcessor = originalCreateProcessor;
  }
});

test('relation factory prepareRelationChanges updates existing one-to-many and adds many-to-many operations', () => {
  const modelInstance = {
    Id: 'p1',
    Lines: [{ Id: 'l1' }, { Id: 'l2' }],
    Tags: [{ Id: 't1' }],
  } as unknown as BaseModel;

  const relationChanges: RelationChangesCollection = {
    Lines: [{ method: RelationArrayMethod.PUSH, args: [], timestamp: 1 }],
    Tags: [{ method: RelationArrayMethod.PUSH, args: [], timestamp: 2 }],
  };

  const relations: ExtractedRelations = {
    oneToManyRelations: [
      {
        type: 'OneToMany',
        fieldName: 'Lines',
        targetModel: RelationFactoryTarget,
        inverseField: 'OwnerId',
        operations: [{ Id: 'old' }],
      },
    ],
    manyToManyRelations: [],
    touchedCollections: new Set<string>(),
  };

  RelationFactory.prepareRelationChanges(parentCtor, modelInstance, relationChanges, relations);

  expect(relations.oneToManyRelations.length).toBe(1);
  expect(relations.oneToManyRelations[0]?.operations).toEqual([{ Id: 'l1' }, { Id: 'l2' }]);

  expect(relations.manyToManyRelations.length).toBe(1);
  expect(relations.manyToManyRelations[0]?.fieldName).toBe('Tags');
  expect(relations.manyToManyRelations[0]?.operations).toEqual([{ Id: 't1' }]);

  expect(Array.from(relations.touchedCollections || []).sort()).toEqual(['Lines', 'Tags']);
});

test('relation factory prepareRelationChanges updates existing many-to-many operations in place', () => {
  const modelInstance = {
    Id: 'p1',
    Tags: [{ Id: 't2' }, { Id: 't3' }],
  } as unknown as BaseModel;

  const relationChanges: RelationChangesCollection = {
    Tags: [{ method: RelationArrayMethod.PUSH, args: [], timestamp: 3 }],
  };

  const relations: ExtractedRelations = {
    oneToManyRelations: [],
    manyToManyRelations: [
      {
        type: 'ManyToMany',
        fieldName: 'Tags',
        joinModel: RelationFactoryJoin,
        targetModel: RelationFactoryTarget,
        joinField: 'OwnerId',
        inverseJoinField: 'TagId',
        operations: [{ Id: 'old' }],
      },
    ],
    touchedCollections: new Set<string>(),
  };

  RelationFactory.prepareRelationChanges(parentCtor, modelInstance, relationChanges, relations);

  expect(relations.manyToManyRelations.length).toBe(1);
  expect(relations.manyToManyRelations[0]?.operations).toEqual([{ Id: 't2' }, { Id: 't3' }]);
  expect(Array.from(relations.touchedCollections || [])).toEqual(['Tags']);
});

test('relation factory getProcessorsForModel returns empty map when metadata has no fields', () => {
  class RelationFactoryNoFieldModel extends BaseModel {}

  const ctor = RelationFactoryNoFieldModel as unknown as ModelCtor<BaseModel>;
  const original = MetadataStorage.instance.getModelMetadata;

  try {
    MetadataStorage.instance.getModelMetadata = (() => ({ fields: undefined })) as any;
    const processors = RelationFactory.getProcessorsForModel(ctor as any);
    expect(processors.size).toBe(0);
  } finally {
    MetadataStorage.instance.getModelMetadata = original;
  }
});

test('relation factory getProcessorsForModel tolerates createProcessor failure and continues', () => {
  const originalCreateProcessor = RelationFactory.createProcessor;

  try {
    RelationFactory.createProcessor = ((_: ModelCtor<BaseModel>, relationType: RelationFieldType) => {
      if (relationType === 'OneToMany') {
        throw new Error('inject one2many fail');
      }
      return {
        prepareForCreate: async (v: any) => ({ processedValue: v, relations: { oneToManyRelations: [], manyToManyRelations: [] } }),
        prepareForUpdate: async (v: any) => ({ processedValue: v, relations: { oneToManyRelations: [], manyToManyRelations: [] } }),
        processRelationUpdate: async () => ({}),
        batchProcessRelationUpdate: async () => ({}),
      } as any;
    }) as unknown as typeof RelationFactory.createProcessor;

    const processors = RelationFactory.getProcessorsForModel(parentCtor);
    expect(processors.has('ManyToOne')).toBe(true);
    expect(processors.has('OneToMany')).toBe(false);
    expect(processors.has('ManyToMany')).toBe(true);
    expect(processors.size).toBe(2);
  } finally {
    RelationFactory.createProcessor = originalCreateProcessor;
  }
});

test('relation factory prepareForUpdate merges relations and touched collections from processors', async () => {
  const originalGetProcessors = RelationFactory.getProcessorsForModel;

  try {
    RelationFactory.getProcessorsForModel = (() =>
      new Map<RelationFieldType, unknown>([
        [
          'OneToMany',
          {
            prepareForUpdate: async (value: Record<string, any>) => ({
              processedValue: { ...value, Lines: undefined },
              relations: {
                oneToManyRelations: [
                  {
                    type: 'OneToMany',
                    fieldName: 'Lines',
                    targetModel: RelationFactoryTarget,
                    inverseField: 'OwnerId',
                    operations: [{ Id: 'l-u1' }],
                  },
                ],
                manyToManyRelations: [],
                touchedCollections: new Set<string>(['Lines']),
              },
            }),
          },
        ],
        [
          'ManyToMany',
          {
            prepareForUpdate: async (value: Record<string, any>) => ({
              processedValue: { ...value, Tags: undefined },
              relations: {
                oneToManyRelations: [],
                manyToManyRelations: [
                  {
                    type: 'ManyToMany',
                    fieldName: 'Tags',
                    joinModel: RelationFactoryJoin,
                    targetModel: RelationFactoryTarget,
                    joinField: 'OwnerId',
                    inverseJoinField: 'TagId',
                    operations: [{ Id: 't-u1' }],
                  },
                ],
                touchedCollections: new Set<string>(['Tags']),
              },
            }),
          },
        ],
      ])) as unknown as typeof RelationFactory.getProcessorsForModel;

    const result = await RelationFactory.prepareForUpdate(
      parentCtor,
      {
        Name: 'parent-u',
        Lines: [{ Id: 'l-u1' }],
        Tags: [{ Id: 't-u1' }],
      },
      ['Lines', 'Tags']
    );

    expect(result.processedValue).toEqual({
      Name: 'parent-u',
      Lines: undefined,
      Tags: undefined,
    });
    expect(result.relations.oneToManyRelations.length).toBe(1);
    expect(result.relations.manyToManyRelations.length).toBe(1);
    expect(Array.from(result.relations.touchedCollections || []).sort()).toEqual(['Lines', 'Tags']);
  } finally {
    RelationFactory.getProcessorsForModel = originalGetProcessors;
  }
});

test('relation factory processToManyRelations processes one-to-many and many-to-many operations', async () => {
  const originalCreateProcessor = RelationFactory.createProcessor;

  try {
    const calls: Array<{ type: string; parentId: string; fieldName: string }> = [];
    RelationFactory.createProcessor = ((_: ModelCtor<BaseModel>, relationType: RelationFieldType) => {
      return {
        processRelationUpdate: async (parentId: string, operation: any) => {
          calls.push({ type: relationType, parentId, fieldName: operation.fieldName });
          return { ok: true, type: relationType, fieldName: operation.fieldName };
        },
      } as any;
    }) as unknown as typeof RelationFactory.createProcessor;

    const out = await RelationFactory.processToManyRelations(parentCtor, 'p-process-1', {
      oneToManyRelations: [
        {
          type: 'OneToMany',
          fieldName: 'Lines',
          targetModel: RelationFactoryTarget,
          inverseField: 'OwnerId',
          operations: [{ Id: 'l1' }],
        },
      ],
      manyToManyRelations: [
        {
          type: 'ManyToMany',
          fieldName: 'Tags',
          joinModel: RelationFactoryJoin,
          targetModel: RelationFactoryTarget,
          joinField: 'OwnerId',
          inverseJoinField: 'TagId',
          operations: [{ Id: 't1' }],
        },
      ],
    });

    expect(out.length).toBe(2);
    expect(calls).toEqual([
      { type: 'OneToMany', parentId: 'p-process-1', fieldName: 'Lines' },
      { type: 'ManyToMany', parentId: 'p-process-1', fieldName: 'Tags' },
    ]);
  } finally {
    RelationFactory.createProcessor = originalCreateProcessor;
  }
});

test('relation factory batchProcessToManyRelations validates length and handles empty input', async () => {
  let err: any;
  try {
    await RelationFactory.batchProcessToManyRelations(parentCtor, ['p1', 'p2'], [{ oneToManyRelations: [], manyToManyRelations: [] }]);
  } catch (e) {
    err = e;
  }
  expect(String(err?.message || '')).toContain('Parent entity Id array length must match relation data array length');

  const out = await RelationFactory.batchProcessToManyRelations(parentCtor, [], []);
  expect(out).toEqual([]);
});

test('relation factory createProcessor rejects unsupported relation type via runtime cast', () => {
  RelationFactory.clearCache();
  expect(() => RelationFactory.createProcessor(parentCtor as any, 'UnsupportedType' as any)).toThrow('Unsupported relation type');
});

test('relation factory getProcessorsForModel warning fallback stringifies non-Error throwables', () => {
  const originalCreateProcessor = RelationFactory.createProcessor;
  const originalWarn = console.warn;
  const warns: string[] = [];

  try {
    console.warn = ((msg: string) => {
      warns.push(String(msg));
    }) as any;

    RelationFactory.createProcessor = ((_: ModelCtor<BaseModel>) => {
      throw 'plain-string-error';
    }) as unknown as typeof RelationFactory.createProcessor;

    const processors = RelationFactory.getProcessorsForModel(parentCtor);
    expect(processors.size).toBe(0);
    expect(warns.some(message => message.includes('plain-string-error'))).toBe(true);
  } finally {
    RelationFactory.createProcessor = originalCreateProcessor;
    console.warn = originalWarn;
  }
});

test('relation factory prepareForCreate and prepareForUpdate tolerate missing relation arrays and empty touchedCollections', async () => {
  const originalGetProcessors = RelationFactory.getProcessorsForModel;

  try {
    RelationFactory.getProcessorsForModel = (() =>
      new Map<RelationFieldType, unknown>([
        [
          'ManyToOne',
          {
            prepareForCreate: async (value: Record<string, any>) => ({
              processedValue: { ...value, Name: 'create-ok' },
              relations: {
                oneToManyRelations: undefined,
                manyToManyRelations: undefined,
                touchedCollections: new Set<string>(),
              },
            }),
            prepareForUpdate: async (value: Record<string, any>) => ({
              processedValue: { ...value, Name: 'update-ok' },
              relations: {
                oneToManyRelations: undefined,
                manyToManyRelations: undefined,
                touchedCollections: new Set<string>(),
              },
            }),
          },
        ],
      ])) as unknown as typeof RelationFactory.getProcessorsForModel;

    const created = await RelationFactory.prepareForCreate(parentCtor, { Name: 'raw' });
    expect(created.processedValue.Name).toBe('create-ok');
    expect(created.relations.oneToManyRelations).toEqual([]);
    expect(created.relations.manyToManyRelations).toEqual([]);
    expect(Array.from(created.relations.touchedCollections || [])).toEqual([]);

    const updated = await RelationFactory.prepareForUpdate(parentCtor, { Name: 'raw' }, ['Name']);
    expect(updated.processedValue.Name).toBe('update-ok');
    expect(updated.relations.oneToManyRelations).toEqual([]);
    expect(updated.relations.manyToManyRelations).toEqual([]);
    expect(Array.from(updated.relations.touchedCollections || [])).toEqual([]);
  } finally {
    RelationFactory.getProcessorsForModel = originalGetProcessors;
  }
});

test('relation factory batchProcessToManyRelations captures many-to-many failures for Error and non-Error values', async () => {
  const originalCreateProcessor = RelationFactory.createProcessor;

  try {
    const relationPayload: ExtractedRelations[] = [
      {
        oneToManyRelations: [],
        manyToManyRelations: [
          {
            type: 'ManyToMany',
            fieldName: 'Tags',
            joinModel: RelationFactoryJoin,
            targetModel: RelationFactoryTarget,
            joinField: 'OwnerId',
            inverseJoinField: 'TagId',
            operations: [{ Id: 't1' }],
          },
        ],
      },
    ];

    RelationFactory.createProcessor = ((_: ModelCtor<BaseModel>, relationType: RelationFieldType) => {
      if (relationType !== 'ManyToMany') {
        return {
          batchProcessRelationUpdate: async () => ({ success: [], errors: [], summary: {} }),
        } as any;
      }
      return {
        batchProcessRelationUpdate: async () => {
          throw new Error('m2m-error-object');
        },
      } as any;
    }) as unknown as typeof RelationFactory.createProcessor;

    const errObjectOut = await RelationFactory.batchProcessToManyRelations(parentCtor, ['p1'], relationPayload);
    expect(String(errObjectOut[0]?.errors?.[0]?.error?.message || '')).toContain('m2m-error-object');

    RelationFactory.createProcessor = ((_: ModelCtor<BaseModel>, relationType: RelationFieldType) => {
      if (relationType !== 'ManyToMany') {
        return {
          batchProcessRelationUpdate: async () => ({ success: [], errors: [], summary: {} }),
        } as any;
      }
      return {
        batchProcessRelationUpdate: async () => {
          throw 'm2m-error-string';
        },
      } as any;
    }) as unknown as typeof RelationFactory.createProcessor;

    const errStringOut = await RelationFactory.batchProcessToManyRelations(parentCtor, ['p1'], relationPayload);
    expect(String(errStringOut[0]?.errors?.[0]?.error?.message || '')).toContain('m2m-error-string');
  } finally {
    RelationFactory.createProcessor = originalCreateProcessor;
  }
});

test('relation factory batchProcessToManyRelations captures one-to-many non-Error failure branch', async () => {
  const originalCreateProcessor = RelationFactory.createProcessor;

  try {
    RelationFactory.createProcessor = ((_: ModelCtor<BaseModel>, relationType: RelationFieldType) => {
      if (relationType === 'OneToMany') {
        return {
          batchProcessRelationUpdate: async () => {
            throw 'o2m-error-string';
          },
        } as any;
      }
      return {
        batchProcessRelationUpdate: async () => ({ success: [], errors: [], summary: {} }),
      } as any;
    }) as unknown as typeof RelationFactory.createProcessor;

    const out = await RelationFactory.batchProcessToManyRelations(
      parentCtor,
      ['p1'],
      [
        {
          oneToManyRelations: [
            {
              type: 'OneToMany',
              fieldName: 'Lines',
              targetModel: RelationFactoryTarget,
              inverseField: 'OwnerId',
              operations: [{ Id: 'o1' }],
            },
          ],
          manyToManyRelations: [],
        },
      ]
    );

    expect(String(out[0]?.errors?.[0]?.error?.message || '')).toContain('o2m-error-string');
  } finally {
    RelationFactory.createProcessor = originalCreateProcessor;
  }
});

test('relation factory prepareRelationChanges skips non-array/default many2one and relation metadata-incomplete branches', () => {
  class RelationFactoryEdgeModel extends BaseModel {}
  const edgeCtor = RelationFactoryEdgeModel as unknown as ModelCtor<BaseModel>;
  const originalGetMeta = MetadataStorage.instance.getModelMetadata;

  try {
    MetadataStorage.instance.getModelMetadata = (() => ({
      fields: new Map([
        ['DefaultManyToOne', { type: 'ManyToOne', relation: { targetModel: () => RelationFactoryTarget } }],
        ['NonArrayOneToMany', { type: 'OneToMany', relation: { targetModel: () => RelationFactoryTarget, inverseField: 'OwnerId' } }],
        ['BrokenOneToMany', { type: 'OneToMany', relation: { targetModel: () => RelationFactoryTarget } }],
        ['BrokenManyToMany', { type: 'ManyToMany', relation: { targetModel: () => RelationFactoryTarget } }],
      ]),
    })) as any;

    const relations: ExtractedRelations = {
      oneToManyRelations: [],
      manyToManyRelations: [],
      touchedCollections: new Set<string>(),
    };

    const modelInstance = {
      DefaultManyToOne: [{ Id: 'm1' }],
      NonArrayOneToMany: 'raw',
      BrokenOneToMany: [{ Id: 'o1' }],
      BrokenManyToMany: [{ Id: 't1' }],
    } as any;

    RelationFactory.prepareRelationChanges(
      edgeCtor as any,
      modelInstance,
      {
        NonArrayOneToMany: [{ method: RelationArrayMethod.PUSH, args: [], timestamp: 1 }],
        DefaultManyToOne: [{ method: RelationArrayMethod.PUSH, args: [], timestamp: 2 }],
        BrokenOneToMany: [{ method: RelationArrayMethod.PUSH, args: [], timestamp: 3 }],
        BrokenManyToMany: [{ method: RelationArrayMethod.PUSH, args: [], timestamp: 4 }],
      },
      relations
    );

    expect(relations.oneToManyRelations).toEqual([]);
    expect(relations.manyToManyRelations).toEqual([]);
    expect(Array.from(relations.touchedCollections || [])).toEqual([]);
  } finally {
    MetadataStorage.instance.getModelMetadata = originalGetMeta;
  }
});

test('relation factory prepareRelationChanges creates one-to-many entry when no existing relation record exists', () => {
  class RelationFactoryNewOneToManyModel extends BaseModel {}
  const edgeCtor = RelationFactoryNewOneToManyModel as unknown as ModelCtor<BaseModel>;
  const originalGetMeta = MetadataStorage.instance.getModelMetadata;

  try {
    MetadataStorage.instance.getModelMetadata = (() => ({
      fields: new Map([
        [
          'Lines',
          {
            type: 'OneToMany',
            relation: { targetModel: () => RelationFactoryTarget, inverseField: 'OwnerId' },
          },
        ],
      ]),
    })) as any;

    const relations: ExtractedRelations = {
      oneToManyRelations: [],
      manyToManyRelations: [],
      touchedCollections: new Set<string>(),
    };

    RelationFactory.prepareRelationChanges(
      edgeCtor as any,
      { Lines: [{ Id: 'l-new' }] } as any,
      { Lines: [{ method: RelationArrayMethod.PUSH, args: [], timestamp: 1 }] },
      relations
    );

    expect(relations.oneToManyRelations.length).toBe(1);
    expect(relations.oneToManyRelations[0]?.operations).toEqual([{ Id: 'l-new' }]);
    expect(Array.from(relations.touchedCollections || [])).toEqual(['Lines']);
  } finally {
    MetadataStorage.instance.getModelMetadata = originalGetMeta;
  }
});
