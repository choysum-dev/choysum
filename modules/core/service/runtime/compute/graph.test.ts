// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { MetadataStorage } from '../../orm/metadata/storage';
import { buildComputeGraph } from './graph';

@Model('test.GraphParentModel')
class GraphParentModel extends BaseModel {
  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => GraphChildModel, inverseField: 'ParentId' },
  })
  Lines?: GraphChildModel[];

  @Field({
    type: 'int',
    column: {
      compute: {
        expr: () => 0,
        deps: ['Lines.Name' as any, 'Lines' as any],
      },
    },
  })
  Score?: number;
}

@Model('test.GraphChildModel')
class GraphChildModel extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => GraphParentModel },
    column: {},
  })
  ParentId?: GraphParentModel;

  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;
}

@Model('test.GraphDecimalModel')
class GraphDecimalModel extends BaseModel {
  @Field({ type: 'int', column: {} })
  Qty?: number;

  @Field({ type: 'int', column: {} })
  UnitPriceScale?: number;

  @Field({
    type: 'decimal',
    column: {
      scaleField: 'UnitPriceScale',
      compute: {
        expr: () => '0',
        deps: ['Qty' as any],
      },
    } as any,
  })
  UnitPrice?: any;
}

test('buildComputeGraph builds reverse compute index with deduplicated lifecycle and membership triggers', () => {
  const parentMeta = MetadataStorage.instance.getModelMetadata(GraphParentModel as any);
  parentMeta.computeGraph = buildComputeGraph(parentMeta);

  const childMeta = MetadataStorage.instance.getModelMetadata(GraphChildModel as any);
  const childGraph = buildComputeGraph(childMeta);

  const index = childGraph?.reverseComputeIndex;
  expect(index instanceof Map).toBe(true);
  expect(index?.get('Name')?.length).toBe(1);
  expect(index?.get('Name')?.[0]?.triggerMode).toBe('field-change');
  expect(index?.get('Name')?.[0]?.collectionField).toBe('Lines');

  expect(index?.get('__lifecycle')?.length).toBe(1);
  expect(index?.get('__lifecycle')?.[0]?.triggerMode).toBe('lifecycle');

  expect(index?.get('ParentId')?.length).toBe(1);
  expect(index?.get('ParentId')?.[0]?.triggerMode).toBe('membership-change');
});

test('buildComputeGraph appends decimal scaleField as implicit scalar dependency', () => {
  const meta = MetadataStorage.instance.getModelMetadata(GraphDecimalModel as any);
  const graph = buildComputeGraph(meta);

  const deps = graph?.parsedDeps.get('UnitPrice') || [];
  expect(deps.some(dep => dep.kind === 'scalar' && (dep as any).field === 'Qty')).toBe(true);
  expect(deps.some(dep => dep.kind === 'scalar' && (dep as any).field === 'UnitPriceScale')).toBe(true);
});

test('buildComputeGraph partitions persisted/virtual compute fields and builds persist reverse deps', () => {
  class GraphStoreSplitModel extends BaseModel {}
  const meta = {
    fullModelName: 'test.GraphStoreSplitModel',
    modelName: 'GraphStoreSplitModel',
    className: 'GraphStoreSplitModel',
    type: GraphStoreSplitModel,
    fields: new Map([
      ['Name', { type: 'varchar', column: {} }],
      [
        'PersistedTotal',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Name' as any],
            },
          },
        },
      ],
      [
        'VirtualTotal',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Name' as any],
              store: false,
            },
          },
        },
      ],
    ]),
  } as any;

  const graph = buildComputeGraph(meta);
  const reverseFromName = graph?.fastReverseDeps.get('Name') || [];

  expect(graph?.persistedComputeFields?.has('PersistedTotal')).toBe(true);
  expect(graph?.virtualComputeFields?.has('VirtualTotal')).toBe(true);
  expect(reverseFromName.includes('PersistedTotal')).toBe(true);
  expect(reverseFromName.includes('VirtualTotal')).toBe(true);
  expect(graph?.fastPersistReverseDeps?.get('Name')).toEqual(['PersistedTotal']);
});

test('buildComputeGraph rejects persisted compute depending on virtual compute', () => {
  class GraphPersistDependsVirtualModel extends BaseModel {}
  const meta = {
    fullModelName: 'test.GraphPersistDependsVirtualModel',
    modelName: 'GraphPersistDependsVirtualModel',
    className: 'GraphPersistDependsVirtualModel',
    type: GraphPersistDependsVirtualModel,
    fields: new Map([
      ['Name', { type: 'varchar', column: {} }],
      [
        'VirtualTotal',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Name' as any],
              store: false,
            },
          },
        },
      ],
      [
        'PersistedTotal',
        {
          type: 'int',
          column: {
            compute: {
              expr: (self: any) => Number(self.VirtualTotal || 0),
              deps: ['VirtualTotal' as any],
            },
          },
        },
      ],
    ]),
  } as any;

  let message = '';
  try {
    buildComputeGraph(meta);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('persisted field PersistedTotal cannot depend on virtual field VirtualTotal')).toBe(true);
});

test('buildComputeGraph fails fast when cross-model dependency path has unknown segment', () => {
  class GraphStrictPathOrder extends BaseModel {}
  const meta = {
    fullModelName: 'test.GraphStrictPathOrder',
    modelName: 'GraphStrictPathOrder',
    className: 'GraphStrictPathOrder',
    type: GraphStrictPathOrder,
    fields: new Map([
      [
        'CustomerId',
        {
          type: 'ManyToOne',
          relation: { targetModel: () => GraphChildModel },
          column: {},
        },
      ],
      [
        'Score',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 0,
              deps: ['CustomerId.MissingField' as any],
            },
          },
        },
      ],
    ]),
  } as any;

  let message = '';
  try {
    buildComputeGraph(meta);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('CustomerId.MissingField')).toBe(true);
  expect(message.includes('has no field')).toBe(true);
});

test('buildComputeGraph keeps graph result when reverse index build throws', () => {
  const storage = MetadataStorage.instance as any;
  const meta = MetadataStorage.instance.getModelMetadata(GraphDecimalModel as any);
  const originalGetModelMetadata = storage.getModelMetadata;
  const originalWarn = console.warn;
  const warns: string[] = [];

  try {
    storage.getModelMetadata = (() => {
      throw new Error('reverse-index-boom');
    }) as any;

    console.warn = ((...args: any[]) => {
      warns.push(args.map(item => String(item)).join(' '));
    }) as any;

    const graph = buildComputeGraph(meta);
    expect(graph).toBeTruthy();
    expect(graph?.reverseComputeIndex).toBeUndefined();
    expect(warns.some(msg => msg.includes('failed to build reverse index'))).toBe(true);
  } finally {
    storage.getModelMetadata = originalGetModelMetadata;
    console.warn = originalWarn;
  }
});

test('buildComputeGraph throws on cyclic compute dependency and reports remaining fields', () => {
  class GraphCycleModel extends BaseModel {}
  const meta = {
    fullModelName: 'test.GraphCycleModel',
    modelName: 'GraphCycleModel',
    className: 'GraphCycleModel',
    type: GraphCycleModel,
    fields: new Map([
      [
        'A',
        {
          type: 'int',
          column: {
            compute: {
              expr: (self: any) => Number(self.B || 0),
              deps: ['B' as any],
            },
          },
        },
      ],
      [
        'B',
        {
          type: 'int',
          column: {
            compute: {
              expr: (self: any) => Number(self.A || 0),
              deps: ['A' as any],
            },
          },
        },
      ],
    ]),
  } as any;

  let message = '';
  try {
    buildComputeGraph(meta);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('dependency cycle')).toBe(true);
  expect(message.includes('A') || message.includes('B')).toBe(true);
});

test('buildComputeGraph reverse-index path handles missing relation target and missing inverseField warnings', () => {
  const storage = MetadataStorage.instance as any;
  const originalModels = storage.models;
  const originalGetModelMetadata = storage.getModelMetadata;
  const originalWarn = console.warn;
  const warns: string[] = [];

  class ChildModel extends BaseModel {}
  class ParentNoTarget extends BaseModel {}
  class ParentNoInverse extends BaseModel {}

  const childMeta = {
    fullModelName: 'test.ChildModel',
    modelName: 'ChildModel',
    className: 'ChildModel',
    type: ChildModel,
    fields: new Map([
      [
        'Name',
        {
          type: 'varchar',
          column: {
            compute: {
              expr: () => 'x',
              deps: undefined,
            },
          },
        },
      ],
    ]),
  } as any;

  const parentNoTargetMeta = {
    fullModelName: '',
    modelName: '',
    className: '',
    type: ParentNoTarget,
    fields: new Map([
      ['Lines', { type: 'OneToMany', relation: {} }],
      [
        'Score',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Lines.Name' as any],
            },
          },
        },
      ],
    ]),
    computeGraph: {
      parsedDeps: new Map([
        [
          'Score',
          [
            { kind: 'collectionPath', collection: 'Lines', chain: ['Name'] },
            { kind: 'collection', collection: 'Lines', chain: [] },
          ],
        ],
      ]),
    },
  } as any;

  const parentNoInverseMeta = {
    fullModelName: '',
    modelName: '',
    className: '',
    type: ParentNoInverse,
    fields: new Map([
      ['Lines', { type: 'OneToMany', relation: { targetModel: () => ChildModel } }],
      [
        'Score',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Lines.Name' as any],
            },
          },
        },
      ],
    ]),
    computeGraph: {
      parsedDeps: new Map([
        [
          'Score',
          [
            { kind: 'collectionPath', collection: 'Lines', chain: ['Name'] },
            { kind: 'collection', collection: 'Lines', chain: [] },
          ],
        ],
      ]),
    },
  } as any;

  try {
    storage.models = new Map<any, any>([
      [ParentNoTarget, parentNoTargetMeta],
      [ParentNoInverse, parentNoInverseMeta],
    ]);

    storage.getModelMetadata = ((ctor: any) => {
      if (ctor === ChildModel) return childMeta;
      if (ctor === ParentNoTarget) return parentNoTargetMeta;
      if (ctor === ParentNoInverse) return parentNoInverseMeta;
      return originalGetModelMetadata.call(storage, ctor);
    }) as any;

    console.warn = ((...args: any[]) => {
      warns.push(args.map(item => String(item)).join(' '));
    }) as any;

    const graph = buildComputeGraph(childMeta);
    expect(graph).toBeTruthy();
    expect(graph?.parsedDeps.get('Name')?.length).toBe(0);
    expect(warns.some(msg => msg.includes('is missing inverseField'))).toBe(true);
    expect(graph?.reverseComputeIndex instanceof Map).toBe(true);
  } finally {
    storage.models = originalModels;
    storage.getModelMetadata = originalGetModelMetadata;
    console.warn = originalWarn;
  }
});

test('buildComputeGraph cycle error and reverse-index warn fallback to Unknown model name', () => {
  class UnknownCycle extends BaseModel {}

  const cycleMeta = {
    fullModelName: '',
    modelName: '',
    className: '',
    type: UnknownCycle,
    fields: new Map([
      [
        'A',
        {
          type: 'int',
          column: { compute: { expr: (self: any) => Number(self.B || 0), deps: ['B' as any] } },
        },
      ],
      [
        'B',
        {
          type: 'int',
          column: { compute: { expr: (self: any) => Number(self.A || 0), deps: ['A' as any] } },
        },
      ],
    ]),
  } as any;

  let cycleMessage = '';
  try {
    buildComputeGraph(cycleMeta);
  } catch (error) {
    cycleMessage = String((error as Error)?.message || error);
  }
  expect(cycleMessage.includes('Unknown')).toBe(true);

  const storage = MetadataStorage.instance as any;
  const originalGetModelMetadata = storage.getModelMetadata;
  const originalWarn = console.warn;
  const warns: string[] = [];
  try {
    storage.getModelMetadata = (() => {
      throw new Error('reverse-index-fail');
    }) as any;
    console.warn = ((...args: any[]) => {
      warns.push(args.map(item => String(item)).join(' '));
    }) as any;

    const graph = buildComputeGraph({
      ...cycleMeta,
      fields: new Map([
        [
          'X',
          {
            type: 'int',
            column: { compute: { expr: () => 1, deps: [] } },
          },
        ],
      ]),
    } as any);

    expect(graph).toBeTruthy();
    expect(warns.some(msg => msg.includes('Unknown'))).toBe(true);
  } finally {
    storage.getModelMetadata = originalGetModelMetadata;
    console.warn = originalWarn;
  }
});

test('buildComputeGraph rejects persisted path dependency on virtual compute root', () => {
  class GraphPathRootModel extends BaseModel {}
  class GraphPathTargetModel extends BaseModel {}

  const meta = {
    fullModelName: 'test.GraphPathRootModel',
    modelName: 'GraphPathRootModel',
    className: 'GraphPathRootModel',
    type: GraphPathRootModel,
    fields: new Map([
      ['Name', { type: 'varchar', column: {} }],
      [
        'VirtualOwner',
        {
          type: 'ManyToOne',
          relation: { targetModel: () => GraphPathTargetModel },
          column: {
            compute: {
              expr: () => ({ Id: 'U1' }),
              deps: ['Name' as any],
              store: false,
            },
          },
        },
      ],
      [
        'PersistedScore',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['VirtualOwner.Name' as any],
            },
          },
        },
      ],
    ]),
  } as any;

  let message = '';
  try {
    buildComputeGraph(meta);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).not.toBe('');
  expect(message.includes('PersistedScore')).toBe(true);
  expect(message.includes('VirtualOwner')).toBe(true);
});

test('buildComputeGraph builds fastPersistReverseDeps for empty and mixed trigger sources', () => {
  class GraphPersistReverseModel extends BaseModel {}

  const meta = {
    fullModelName: 'test.GraphPersistReverseModel',
    modelName: 'GraphPersistReverseModel',
    className: 'GraphPersistReverseModel',
    type: GraphPersistReverseModel,
    fields: new Map([
      ['Name', { type: 'varchar', column: {} }],
      ['Code', { type: 'varchar', column: {} }],
      [
        'VirtualFromCode',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Code' as any],
              store: false,
            },
          },
        },
      ],
      [
        'PersistedFromName',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Name' as any],
            },
          },
        },
      ],
      [
        'VirtualFromName',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Name' as any],
              store: false,
            },
          },
        },
      ],
    ]),
  } as any;

  const graph = buildComputeGraph(meta);
  expect(graph?.fastPersistReverseDeps?.has('Code')).toBe(false);
  expect(graph?.fastPersistReverseDeps?.get('Name')).toEqual(['PersistedFromName']);
  expect(graph?.fastReverseDeps?.get('Name')?.includes('VirtualFromName')).toBe(true);
});

test('buildComputeGraph path dependency adds edge for persisted compute root field', () => {
  class GraphPathEdgeModel extends BaseModel {}

  const meta = {
    fullModelName: 'test.GraphPathEdgeModel',
    modelName: 'GraphPathEdgeModel',
    className: 'GraphPathEdgeModel',
    type: GraphPathEdgeModel,
    fields: new Map([
      ['Name', { type: 'varchar', column: {} }],
      [
        'OwnerPersist',
        {
          type: 'ManyToOne',
          relation: { targetModel: () => GraphChildModel },
          column: {
            compute: {
              expr: () => ({ Id: 'C-1' }),
              deps: ['Name' as any],
            },
          },
        },
      ],
      [
        'PersistedScore',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['OwnerPersist.Name' as any],
            },
          },
        },
      ],
    ]),
  } as any;

  const graph = buildComputeGraph(meta);
  expect(graph?.fastReverseDeps?.get('OwnerPersist')?.includes('PersistedScore')).toBe(true);
});

test('buildComputeGraph path dependency rejects persisted field depending on virtual compute root', () => {
  class GraphPathVirtualRootModel extends BaseModel {}

  const meta = {
    fullModelName: 'test.GraphPathVirtualRootModel',
    modelName: 'GraphPathVirtualRootModel',
    className: 'GraphPathVirtualRootModel',
    type: GraphPathVirtualRootModel,
    fields: new Map([
      ['Name', { type: 'varchar', column: {} }],
      [
        'OwnerVirtual',
        {
          type: 'ManyToOne',
          relation: { targetModel: () => GraphChildModel },
          column: {
            compute: {
              expr: () => ({ Id: 'C-1' }),
              deps: ['Name' as any],
              store: false,
            },
          },
        },
      ],
      [
        'PersistedScore',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['OwnerVirtual.Name' as any],
            },
          },
        },
      ],
    ]),
  } as any;

  let message = '';
  try {
    buildComputeGraph(meta);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('persisted field PersistedScore cannot depend on virtual field OwnerVirtual')).toBe(true);
});

test('buildComputeGraph appends scaleField when decimal compute declares column.scaleField', () => {
  class GraphSelectScaleModel extends BaseModel {}

  const meta = {
    fullModelName: 'test.GraphSelectScaleModel',
    modelName: 'GraphSelectScaleModel',
    className: 'GraphSelectScaleModel',
    type: GraphSelectScaleModel,
    fields: new Map([
      ['Qty', { type: 'int', column: {} }],
      ['PriceScale', { type: 'int', column: {} }],
      [
        'Price',
        {
          type: 'decimal',
          column: {
            scaleField: 'PriceScale',
            compute: {
              expr: () => '0',
              deps: ['Qty' as any],
            },
          },
        },
      ],
    ]),
  } as any;

  const graph = buildComputeGraph(meta);
  const deps = graph?.parsedDeps.get('Price') || [];
  expect(deps.some(dep => dep.kind === 'scalar' && (dep as any).field === 'PriceScale')).toBe(true);
});

test('buildComputeGraph keeps non-decimal compute deps unchanged (scaleField false branch)', () => {
  class GraphNonDecimalModel extends BaseModel {}

  const meta = {
    fullModelName: 'test.GraphNonDecimalModel',
    modelName: 'GraphNonDecimalModel',
    className: 'GraphNonDecimalModel',
    type: GraphNonDecimalModel,
    fields: new Map([
      ['Name', { type: 'varchar', column: {} }],
      [
        'Score',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Name' as any],
            },
          },
        },
      ],
    ]),
  } as any;

  const graph = buildComputeGraph(meta);
  const deps = graph?.parsedDeps.get('Score') || [];
  expect(deps).toEqual([{ kind: 'scalar', field: 'Name' }]);
});

test('buildComputeGraph rejects persisted scalar dependency on virtual compute field (direct branch)', () => {
  class GraphScalarRejectModel extends BaseModel {}

  const meta = {
    fullModelName: 'test.GraphScalarRejectModel',
    modelName: 'GraphScalarRejectModel',
    className: 'GraphScalarRejectModel',
    type: GraphScalarRejectModel,
    fields: new Map([
      ['Name', { type: 'varchar', column: {} }],
      [
        'VirtualA',
        {
          type: 'int',
          column: {
            compute: {
              expr: () => 1,
              deps: ['Name' as any],
              store: false,
            },
          },
        },
      ],
      [
        'PersistedB',
        {
          type: 'int',
          column: {
            compute: {
              expr: (self: any) => Number(self.VirtualA || 0),
              deps: ['VirtualA' as any],
            },
          },
        },
      ],
    ]),
  } as any;

  let message = '';
  try {
    buildComputeGraph(meta);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message.includes('persisted field PersistedB cannot depend on virtual field VirtualA')).toBe(true);
});
