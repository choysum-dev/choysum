// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from './model';
import {
  __collectModelOnchangeComputeSubsetForTest,
  __collectModelOnchangeParentAllFieldNamesForTest,
  __collectModelOnchangeReadsRootForTest,
  __normalizeModelOnchangeChangedFieldsRawForTest,
  prepareModelOnchangePreview,
} from './model_onchange_prepare';
import { Field, Model, Onchange } from '../decorator';
import { MetadataStorage } from '../metadata/storage';
import { RepositoryFactory } from '../repository/repository_factory';
import { buildComputeGraph } from '../../runtime/compute/graph';
import { PathPlanBuilder } from '../../runtime/onchange/plan';

@Model('test.ModelOnchangePreparePartner')
class ModelOnchangePreparePartner extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;
}

@Model('test.ModelOnchangePrepareLine')
class ModelOnchangePrepareLine extends BaseModel {
  @Field({ type: 'decimal', column: { precision: 10, scale: 2 } })
  Qty?: any;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => ModelOnchangePrepareModel }, column: {} })
  OrderId?: ModelOnchangePrepareModel;
}

@Model('test.ModelOnchangePrepareModel')
class ModelOnchangePrepareModel extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  @Field({ type: 'varchar', column: { size: 64 } })
  Status?: string;

  @Field({ type: 'varchar', column: { size: 64 } })
  Code?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => ModelOnchangePreparePartner }, column: {} })
  PartnerId?: ModelOnchangePreparePartner;

  @Field({ type: 'OneToMany', relation: { targetModel: () => ModelOnchangePrepareLine, inverseField: 'OrderId' } })
  Lines?: ModelOnchangePrepareLine[];

  @Field({
    type: 'varchar',
    column: {
      size: 128,
      compute: {
        expr: (self: ModelOnchangePrepareModel) => `${self.Name || ''}:${self.Code || ''}`,
        deps: ['Name' as any, 'Code' as any],
      },
    },
  })
  Summary?: string;

  @Onchange<ModelOnchangePrepareModel>('Name', { reads: ['Status', 'PartnerId.Name'] })
  handleNamePreview() {}
}

function prepareModelMetadata() {
  delete (ModelOnchangePrepareModel as any).metadata;
  const meta = MetadataStorage.instance.getModelMetadata(ModelOnchangePrepareModel as any);
  meta.computeGraph = buildComputeGraph(meta);
  return meta;
}

test('model onchange prepare reloads missing edit fields and forwards normalized prefetch context', async () => {
  const originalGetCachedOrBuildV2 = PathPlanBuilder.getCachedOrBuildV2;
  const originalExecuteWithPlan = PathPlanBuilder.executeWithPlan;

  const searchCalls: any[] = [];
  const prefetchCalls: any[] = [];
  const execStats = { queries: 1, rows: 2 } as any;
  const meta = prepareModelMetadata();

  RepositoryFactory.setRepository(
    ModelOnchangePrepareModel as any,
    {
      search: async (condition: any, options: any) => {
        searchCalls.push({ condition, options });
        return [
          {
            Id: 'ROW-1',
            Status: 'draft',
            Code: 'C001',
            PartnerId: { Id: 'P1', Name: 'Partner A' },
          },
        ];
      },
    } as any
  );

  try {
    PathPlanBuilder.getCachedOrBuildV2 = ((ModelCtor: any, m2oReads: any, collectionReads: any, computeM2oPaths: any, computeCollectionPaths: any) => {
      prefetchCalls.push({ ModelCtor, m2oReads, collectionReads, computeM2oPaths, computeCollectionPaths });
      return {
        plan: { marker: 'plan' },
        fromCache: true,
        signature: 'prepare-sig',
        pathDepthMax: 3,
      };
    }) as any;

    PathPlanBuilder.executeWithPlan = (async (_ModelCtor: any, nextMeta: any, mergedDraft: any, plan: any) => {
      prefetchCalls.push({ nextMeta, mergedDraft, plan });
      return execStats;
    }) as any;

    const result = await prepareModelOnchangePreview({
      ModelCtor: ModelOnchangePrepareModel as any,
      draft: { Id: 'ROW-1', Name: 'front-name' },
      changed: ['Name'],
    });

    expect(result.preview.meta).toBe(meta as any);
    expect(result.preview.changedFields).toEqual(['Name']);
    expect(result.diagnostics.missingCount).toBe(3);
    expect(result.preview.mergedDraft).toEqual({
      Id: 'ROW-1',
      Status: 'draft',
      Code: 'C001',
      PartnerId: { Id: 'P1', Name: 'Partner A' },
      Name: 'front-name',
    });
    expect(result.diagnostics.usedCache).toBe(true);
    expect(result.diagnostics.pathDepthMax).toBe(3);
    expect(result.diagnostics.cachedSignature).toBe('prepare-sig');
    expect(result.diagnostics.execStats).toBe(execStats);
    expect(Array.from(result.diagnostics.readsRoot).sort()).toEqual(['PartnerId', 'Status']);
    expect(result.preview.previewProxy.Name).toBe('front-name');
    expect((result.preview.previewProxy as any).PartnerId?.Name).toBe('Partner A');

    expect(searchCalls.length).toBe(1);
    expect(searchCalls[0]?.condition).toEqual(['Id', '=', 'ROW-1']);
    expect([...(searchCalls[0]?.options?.fields || [])].sort()).toEqual(['Code', 'Id', 'PartnerId', 'Status']);

    expect(prefetchCalls.length).toBe(2);
    expect(prefetchCalls[0]?.ModelCtor).toBe(ModelOnchangePrepareModel as any);
    expect(Array.from(prefetchCalls[0]?.m2oReads?.entries?.() || [])).toEqual([
      ['Status', []],
      ['PartnerId', [['Name']]],
    ]);
    expect(Array.from(prefetchCalls[0]?.collectionReads?.entries?.() || [])).toEqual([]);
    expect(Array.from(prefetchCalls[0]?.computeM2oPaths?.entries?.() || [])).toEqual([]);
    expect(Array.from(prefetchCalls[0]?.computeCollectionPaths?.entries?.() || [])).toEqual([]);
    expect(prefetchCalls[1]?.nextMeta).toBe(meta as any);
    expect(prefetchCalls[1]?.mergedDraft).toEqual({
      Id: 'ROW-1',
      Status: 'draft',
      Code: 'C001',
      PartnerId: { Id: 'P1', Name: 'Partner A' },
      Name: 'front-name',
    });
    expect(prefetchCalls[1]?.plan).toEqual({ marker: 'plan' });
  } finally {
    PathPlanBuilder.getCachedOrBuildV2 = originalGetCachedOrBuildV2;
    PathPlanBuilder.executeWithPlan = originalExecuteWithPlan;
  }
});

test('model onchange prepare normalizes selector changes and creates collection-safe preview proxy', async () => {
  const originalGetCachedOrBuildV2 = PathPlanBuilder.getCachedOrBuildV2;
  const originalExecuteWithPlan = PathPlanBuilder.executeWithPlan;

  const prefetchCalls: any[] = [];
  prepareModelMetadata();

  try {
    PathPlanBuilder.getCachedOrBuildV2 = ((ModelCtor: any, m2oReads: any, collectionReads: any, computeM2oPaths: any, computeCollectionPaths: any) => {
      prefetchCalls.push({ ModelCtor, m2oReads, collectionReads, computeM2oPaths, computeCollectionPaths });
      return {
        plan: { marker: 'selector-plan' },
        fromCache: false,
        signature: 'selector-sig',
        pathDepthMax: 1,
      };
    }) as any;

    PathPlanBuilder.executeWithPlan = (async (_ModelCtor: any, _nextMeta: any, mergedDraft: any, plan: any) => {
      prefetchCalls.push({ mergedDraft, plan });
      return undefined;
    }) as any;

    const result = await prepareModelOnchangePreview({
      ModelCtor: ModelOnchangePrepareModel as any,
      draft: { Name: 'draft-only' },
      changed: ['Lines[2].Qty'],
    });

    expect(result.preview.changedFields.slice().sort()).toEqual(['Lines', 'Lines.Qty']);
    expect(Array.from(result.preview.selParsed.collectionRoots).sort()).toEqual(['Lines']);
    expect(Array.from(result.preview.selParsed.fieldSignals.get('Lines') || [])).toEqual(['Qty']);
    expect(result.preview.selParsed.selectors.get('Lines')).toEqual({ kind: 'pos', positions: new Set([2]) });
    expect(result.diagnostics.missingCount).toBe(1);
    expect(result.diagnostics.usedCache).toBe(false);
    expect(result.diagnostics.cachedSignature).toBe('selector-sig');
    expect(Array.isArray((result.preview.previewProxy as any).Lines)).toBe(true);
    expect((result.preview.previewProxy as any).Lines.length).toBe(0);

    expect(prefetchCalls.length).toBe(2);
    expect(Array.from(prefetchCalls[0]?.m2oReads?.entries?.() || [])).toEqual([]);
    expect(Array.from(prefetchCalls[0]?.collectionReads?.entries?.() || [])).toEqual([]);
    expect(Array.from(prefetchCalls[0]?.computeM2oPaths?.entries?.() || [])).toEqual([]);
    expect(Array.from(prefetchCalls[0]?.computeCollectionPaths?.entries?.() || [])).toEqual([]);
    expect(prefetchCalls[1]?.mergedDraft).toEqual({ Name: 'draft-only' });
    expect(prefetchCalls[1]?.plan).toEqual({ marker: 'selector-plan' });
  } finally {
    PathPlanBuilder.getCachedOrBuildV2 = originalGetCachedOrBuildV2;
    PathPlanBuilder.executeWithPlan = originalExecuteWithPlan;
  }
});

test('model onchange prepare compute subset helper supports missing graph and transitive reverse deps', () => {
  const noGraphMeta = {} as any;
  const noGraphSubset = __collectModelOnchangeComputeSubsetForTest(noGraphMeta, ['Name'], [] as any);
  expect(Array.from(noGraphSubset)).toEqual([]);

  const transitiveMeta = {
    computeGraph: {
      fastReverseDeps: new Map<string, Set<string>>([
        ['Name', new Set(['Summary'])],
        ['Summary', new Set(['DisplayName'])],
      ]),
    },
  } as any;

  const subset = __collectModelOnchangeComputeSubsetForTest(transitiveMeta, ['Name'], [] as any);
  expect(Array.from(subset).sort()).toEqual(['Summary']);
});

test('model onchange prepare reads-root helper handles missing reads and empty segments', () => {
  const roots = __collectModelOnchangeReadsRootForTest({} as any, [{ reads: undefined }, { reads: ['', 'PartnerId.Name'] }] as any);

  expect(Array.from(roots).sort()).toEqual(['PartnerId']);
});

test('model onchange prepare helper normalizes changed fallback and parent-field fallback', () => {
  expect(__normalizeModelOnchangeChangedFieldsRawForTest(undefined as any)).toEqual([]);
  expect(__normalizeModelOnchangeChangedFieldsRawForTest(['Name', '' as any, 0 as any, 7 as any] as any)).toEqual(['Name', '7']);

  const withKeys = __collectModelOnchangeParentAllFieldNamesForTest({
    fields: new Map<string, any>([
      ['Id', {}],
      ['Name', {}],
    ]),
  } as any);
  expect(Array.from(withKeys).sort()).toEqual(['Id', 'Name']);

  const withoutKeys = __collectModelOnchangeParentAllFieldNamesForTest({ fields: {} } as any);
  expect(Array.from(withoutKeys)).toEqual([]);
});
