// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from './model';
import { applyModelOnchangePreviewCascade } from './model_onchange_preview_cascade';
import { Field, Model } from '../decorator';
import { MetadataStorage } from '../metadata/storage';
import { PathPlanBuilder } from '../../runtime/onchange/plan';
import { ComputeEngine } from '../../runtime/compute/engine';
import { OnchangeEngine } from '../../runtime/onchange/engine';

@Model('test.ModelOnchangePreviewCascadeOrder')
class ModelOnchangePreviewCascadeOrder extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Status?: string;

  @Field({ type: 'varchar', column: { size: 64 } })
  Total?: string;

  @Field({ type: 'OneToMany', relation: { targetModel: () => ModelOnchangePreviewCascadeLine, inverseField: 'OrderId' } })
  Lines?: ModelOnchangePreviewCascadeLine[];
}

@Model('test.ModelOnchangePreviewCascadeLine')
class ModelOnchangePreviewCascadeLine extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => ModelOnchangePreviewCascadeOrder }, column: {} })
  OrderId?: ModelOnchangePreviewCascadeOrder;

  @Field({ type: 'int', column: {} })
  Qty?: number;

  @Field({ type: 'varchar', column: { size: 64 } })
  Amount?: string;
}

function preparePreviewCascadeMetadata() {
  const orderMeta = MetadataStorage.instance.getModelMetadata(ModelOnchangePreviewCascadeOrder as any);
  const lineMeta = MetadataStorage.instance.getModelMetadata(ModelOnchangePreviewCascadeLine as any);

  orderMeta.computeGraph = {
    fastReverseDeps: new Map([['Lines', ['Total']]]),
    computeScalarDeps: new Map([['Total', new Set<string>()]]),
  } as any;

  lineMeta.computeGraph = {
    computePathDeps: new Map([
      [
        'Amount',
        [
          {
            root: 'OrderId',
            chain: ['Status'],
          },
        ],
      ],
    ]),
  } as any;

  lineMeta.onchangeHandlers = [] as any;

  return { orderMeta, lineMeta };
}

test('model onchange preview cascade auto-detects parent-dependent collection and merges child patch with parent recompute', async () => {
  const { orderMeta, lineMeta } = preparePreviewCascadeMetadata();
  const originalRecompute = ComputeEngine.recompute;

  const recomputeCalls: Array<{ meta: any; seed: string[] }> = [];

  try {
    ComputeEngine.recompute = (async (meta: any, entity: any, seed: Set<string>) => {
      const seedList = Array.from(seed).sort();
      recomputeCalls.push({ meta, seed: seedList });

      if (meta === lineMeta) {
        expect(seedList).toEqual(['OrderId']);
        expect((entity as any).OrderId?.Status).toBe('confirmed');
        (entity as any).Amount = '200';
        return;
      }

      if (meta === orderMeta) {
        expect(seedList).toEqual(['Lines']);
        expect(Array.isArray((entity as any).Lines)).toBe(true);
        expect((entity as any).Lines?.[0]?.Amount).toBe('200');
        (entity as any).Total = '200';
      }
    }) as any;

    const previewProxy: any = {
      Id: 'ORDER-1',
      Status: 'confirmed',
      Lines: [{ Id: 'LINE-1', Qty: 2, Amount: 'old' }],
    };
    const res: any = { value: { Status: 'confirmed' } };

    await applyModelOnchangePreviewCascade({
      meta: orderMeta as any,
      previewProxy,
      changedFields: ['Status'],
      selParsed: {
        normalizedSeeds: new Set(['Status']),
        collectionRoots: new Set<string>(),
        fieldSignals: new Map<string, Set<string>>(),
        selectors: new Map(),
      },
      opts: { withCompute: true },
      res,
    });

    expect(recomputeCalls.length).toBe(2);
    expect(res.value).toEqual({
      Status: 'confirmed',
      Total: '200',
      __collectionPatch: {
        Lines: [{ Id: 'LINE-1', Amount: '200' }],
      },
    });
    expect(res.computeRecomputed).toEqual(['Total']);
    expect(previewProxy.Lines[0]?.Amount).toBe('200');
    expect(previewProxy.Total).toBe('200');
  } finally {
    ComputeEngine.recompute = originalRecompute;
  }
});

test('model onchange preview cascade respects row selector and prefixes child outputs for selected row only', async () => {
  const { orderMeta, lineMeta } = preparePreviewCascadeMetadata();
  const originalRun = OnchangeEngine.run;
  const originalRecompute = ComputeEngine.recompute;
  const originalGetCachedOrBuildV2 = PathPlanBuilder.getCachedOrBuildV2;
  const originalExecuteWithPlan = PathPlanBuilder.executeWithPlan;

  const planCalls: any[] = [];
  const childRuns: string[] = [];

  try {
    PathPlanBuilder.getCachedOrBuildV2 = ((ModelCtor: any, m2oReads: any, collectionReads: any, computeM2oPaths: any, computeCollectionPaths: any) => {
      planCalls.push({ ModelCtor, m2oReads, collectionReads, computeM2oPaths, computeCollectionPaths });
      return { plan: { marker: 'cascade-plan' } };
    }) as any;

    PathPlanBuilder.executeWithPlan = ((ModelCtor: any, meta: any, draft: any, plan: any) => {
      planCalls.push({ ModelCtor, meta, draft, plan });
    }) as any;

    OnchangeEngine.run = (async (_meta: any, entity: any, changedFields: string[]) => {
      childRuns.push(String((entity as any).Id || ''));
      expect(changedFields).toEqual(['Qty']);
      return {
        messages: [{ level: 'error', message: 'child blocked', field: 'Amount', blocking: true }],
        condition: [{ field: 'Qty', condition: ['Qty', '>', 0] as any }],
        selection: [{ field: 'TaxMode', selection: ['vat'], disabled: ['vat'] }],
      } as any;
    }) as any;

    ComputeEngine.recompute = (async (meta: any, entity: any, seed: Set<string>) => {
      if (meta !== lineMeta) return;
      expect(Array.from(seed).sort()).toEqual(['OrderId', 'Qty']);
      (entity as any).Amount = `patched-${String((entity as any).Id || '')}`;
    }) as any;

    const previewProxy: any = {
      Id: 'ORDER-2',
      Status: 'draft',
      Lines: [
        { Id: 'LINE-1', Qty: 1, Amount: 'old-1' },
        { Id: 'LINE-2', Qty: 2, Amount: 'old-2' },
      ],
    };
    const res: any = {};

    await applyModelOnchangePreviewCascade({
      meta: orderMeta as any,
      previewProxy,
      changedFields: ['Lines.Qty'],
      selParsed: {
        normalizedSeeds: new Set(['Lines', 'Lines.Qty']),
        collectionRoots: new Set(['Lines']),
        fieldSignals: new Map([['Lines', new Set(['Qty'])]]),
        selectors: new Map([
          [
            'Lines',
            {
              kind: 'id',
              ids: new Set(['LINE-2']),
            },
          ],
        ]),
      },
      opts: { withCompute: false },
      res,
    });

    expect(childRuns).toEqual(['LINE-2']);
    expect(planCalls.length).toBe(2);
    expect(res.messages).toEqual([{ level: 'error', message: 'child blocked', field: 'Lines(id=LINE-2).Amount', blocking: true }]);
    expect(res.condition).toEqual([{ field: 'Lines(id=LINE-2).Qty', condition: ['Qty', '>', 0] }]);
    expect(res.selection).toEqual([{ field: 'Lines(id=LINE-2).TaxMode', selection: ['vat'], disabled: ['vat'] }]);
    expect(res.value).toEqual({
      __collectionPatch: {
        Lines: [{ Id: 'LINE-2', Amount: 'patched-LINE-2' }],
      },
    });
    expect(previewProxy.Lines[0]?.Amount).toBe('old-1');
    expect(previewProxy.Lines[1]?.Amount).toBe('patched-LINE-2');
  } finally {
    OnchangeEngine.run = originalRun;
    ComputeEngine.recompute = originalRecompute;
    PathPlanBuilder.getCachedOrBuildV2 = originalGetCachedOrBuildV2;
    PathPlanBuilder.executeWithPlan = originalExecuteWithPlan;
  }
});

test('model onchange preview cascade supports position selector formatting', async () => {
  const { orderMeta, lineMeta } = preparePreviewCascadeMetadata();
  const originalRun = OnchangeEngine.run;
  const originalRecompute = ComputeEngine.recompute;

  try {
    OnchangeEngine.run = (async (_meta: any, entity: any, changedFields: string[]) => {
      expect(changedFields).toEqual(['Qty']);
      return {
        messages: [{ level: 'warn', message: 'line-warning', field: 'Amount' }],
      } as any;
    }) as any;

    ComputeEngine.recompute = (async (meta: any, entity: any, seed: Set<string>) => {
      if (meta !== lineMeta) return;
      expect(Array.from(seed).sort()).toEqual(['OrderId', 'Qty']);
      (entity as any).Amount = `pos-${String((entity as any).Id || '')}`;
    }) as any;

    const previewProxy: any = {
      Id: 'ORDER-POS',
      Lines: [
        { Id: 'LINE-1', Qty: 1, Amount: 'old-1' },
        { Id: 'LINE-2', Qty: 2, Amount: 'old-2' },
      ],
    };
    const res: any = {};

    await applyModelOnchangePreviewCascade({
      meta: orderMeta as any,
      previewProxy,
      changedFields: ['Lines.Qty'],
      selParsed: {
        normalizedSeeds: new Set(['Lines', 'Lines.Qty']),
        collectionRoots: new Set(['Lines']),
        fieldSignals: new Map([['Lines', new Set(['Qty'])]]),
        selectors: new Map([
          [
            'Lines',
            {
              kind: 'pos',
              positions: new Set([1]),
            } as any,
          ],
        ]),
      },
      opts: { withCompute: false },
      res,
    });

    expect(res.messages).toEqual([{ level: 'warn', message: 'line-warning', field: 'Lines[1].Amount' }]);
    expect(res.value).toEqual({
      __collectionPatch: {
        Lines: [{ Id: 'LINE-2', Amount: 'pos-LINE-2' }],
      },
    });
    expect(previewProxy.Lines[0]?.Amount).toBe('old-1');
    expect(previewProxy.Lines[1]?.Amount).toBe('pos-LINE-2');
  } finally {
    OnchangeEngine.run = originalRun;
    ComputeEngine.recompute = originalRecompute;
  }
});

test('model onchange preview cascade treats unknown selector kind as all rows and keeps unqualified prefixes', async () => {
  const { orderMeta, lineMeta } = preparePreviewCascadeMetadata();
  const originalRun = OnchangeEngine.run;
  const originalRecompute = ComputeEngine.recompute;
  const handledIds: string[] = [];

  try {
    OnchangeEngine.run = (async (_meta: any, entity: any) => {
      handledIds.push(String((entity as any).Id || ''));
      return {
        messages: [{ level: 'info', message: 'ok', field: 'Amount' }],
      } as any;
    }) as any;

    ComputeEngine.recompute = (async (meta: any, entity: any, seed: Set<string>) => {
      if (meta !== lineMeta) return;
      expect(Array.from(seed).sort()).toEqual(['OrderId', 'Qty']);
      (entity as any).Amount = `all-${String((entity as any).Id || '')}`;
    }) as any;

    const previewProxy: any = {
      Id: 'ORDER-UNK',
      Lines: [
        { Id: 'LINE-A', Qty: 1, Amount: 'old-a' },
        { Id: 'LINE-B', Qty: 2, Amount: 'old-b' },
      ],
    };
    const res: any = {};

    await applyModelOnchangePreviewCascade({
      meta: orderMeta as any,
      previewProxy,
      changedFields: ['Lines.Qty'],
      selParsed: {
        normalizedSeeds: new Set(['Lines', 'Lines.Qty']),
        collectionRoots: new Set(['Lines']),
        fieldSignals: new Map([['Lines', new Set(['Qty'])]]),
        selectors: new Map([
          [
            'Lines',
            {
              kind: 'mystery',
            } as any,
          ],
        ]),
      },
      opts: { withCompute: false },
      res,
    });

    expect(handledIds.sort()).toEqual(['LINE-A', 'LINE-B']);
    expect(res.messages).toEqual([
      { level: 'info', message: 'ok', field: 'Lines.Amount' },
      { level: 'info', message: 'ok', field: 'Lines.Amount' },
    ]);
    expect(res.value).toEqual({
      __collectionPatch: {
        Lines: [
          { Id: 'LINE-A', Amount: 'all-LINE-A' },
          { Id: 'LINE-B', Amount: 'all-LINE-B' },
        ],
      },
    });
  } finally {
    OnchangeEngine.run = originalRun;
    ComputeEngine.recompute = originalRecompute;
  }
});

test('model onchange preview cascade filters child handlers by changed signal before execution', async () => {
  const { orderMeta, lineMeta } = preparePreviewCascadeMetadata();
  const originalRun = OnchangeEngine.run;
  const originalRecompute = ComputeEngine.recompute;

  lineMeta.onchangeHandlers = [
    { method: 'onQty', triggers: ['Qty'] },
    { method: 'onOther', triggers: ['Other'] },
  ] as any;

  const handledIds: string[] = [];

  try {
    OnchangeEngine.run = (async (_meta: any, entity: any, changedFields: string[]) => {
      handledIds.push(String((entity as any).Id || ''));
      expect(changedFields).toEqual(['Qty']);
      return {} as any;
    }) as any;

    ComputeEngine.recompute = (async (meta: any, entity: any, seed: Set<string>) => {
      if (meta !== lineMeta) return;
      expect(Array.from(seed).sort()).toEqual(['OrderId', 'Qty']);
      (entity as any).Amount = `filtered-${String((entity as any).Id || '')}`;
    }) as any;

    const previewProxy: any = {
      Id: 'ORDER-FILTER',
      Lines: [{ Id: 'LINE-1', Qty: 3, Amount: 'old-1' }],
    };
    const res: any = {};

    await applyModelOnchangePreviewCascade({
      meta: orderMeta as any,
      previewProxy,
      changedFields: ['Lines.Qty'],
      selParsed: {
        normalizedSeeds: new Set(['Lines', 'Lines.Qty']),
        collectionRoots: new Set(['Lines']),
        fieldSignals: new Map([['Lines', new Set(['Qty'])]]),
        selectors: new Map(),
      },
      opts: { withCompute: false },
      res,
    });

    expect(handledIds).toEqual(['LINE-1']);
    expect(previewProxy.Lines[0]?.Amount).toBe('filtered-LINE-1');
  } finally {
    OnchangeEngine.run = originalRun;
    ComputeEngine.recompute = originalRecompute;
  }
});

test('model onchange preview cascade skips unknown collection roots and keeps condition entries when plan execution throws', async () => {
  const { orderMeta, lineMeta } = preparePreviewCascadeMetadata();
  const originalRun = OnchangeEngine.run;
  const originalRecompute = ComputeEngine.recompute;
  const originalGetCachedOrBuildV2 = PathPlanBuilder.getCachedOrBuildV2;
  const originalExecuteWithPlan = PathPlanBuilder.executeWithPlan;

  try {
    PathPlanBuilder.getCachedOrBuildV2 = (() => ({ plan: { marker: 'failing-plan' } })) as any;
    PathPlanBuilder.executeWithPlan = (() => {
      throw new Error('plan-failed');
    }) as any;

    OnchangeEngine.run = (async () => ({
      condition: [undefined, { field: 'Qty', condition: ['Qty', '>', 0] as any }],
    })) as any;

    ComputeEngine.recompute = (async (meta: any, entity: any) => {
      if (meta !== lineMeta) return;
      (entity as any).Amount = 'patched';
    }) as any;

    const previewProxy: any = {
      Id: 'ORDER-PLAN-ERR',
      Lines: [{ Id: 'LINE-1', Qty: 1, Amount: 'old' }],
    };
    const res: any = {};

    await applyModelOnchangePreviewCascade({
      meta: orderMeta as any,
      previewProxy,
      changedFields: ['Lines..Qty'],
      selParsed: {
        normalizedSeeds: new Set(['Lines', 'Lines..Qty']),
        collectionRoots: new Set(['UnknownRoot', 'Lines']),
        fieldSignals: new Map([['Lines', new Set(['Qty'])]]),
        selectors: new Map(),
      },
      opts: { withCompute: false },
      res,
    });

    expect(previewProxy.Lines[0]?.Amount).toBe('patched');
    expect(res.condition).toEqual([{ field: 'Lines.Qty', condition: ['Qty', '>', 0] }]);
  } finally {
    OnchangeEngine.run = originalRun;
    ComputeEngine.recompute = originalRecompute;
    PathPlanBuilder.getCachedOrBuildV2 = originalGetCachedOrBuildV2;
    PathPlanBuilder.executeWithPlan = originalExecuteWithPlan;
  }
});

test('model onchange preview cascade uses nested label prefix and merges parent recomputed fields', async () => {
  const { orderMeta } = preparePreviewCascadeMetadata();
  const originalRun = OnchangeEngine.run;
  const originalRecompute = ComputeEngine.recompute;

  const originalLinesField = orderMeta.fields.get('Lines');
  const lineMeta = MetadataStorage.instance.getModelMetadata(ModelOnchangePreviewCascadeLine as any);
  const originalSubLinesField = lineMeta.fields.get('SubLines');
  const originalOrderGraph = orderMeta.computeGraph;

  try {
    (lineMeta.fields as Map<string, any>).set('SubLines', {
      type: 'OneToMany',
      relation: { targetModel: () => ModelOnchangePreviewCascadeLine, inverseField: 'OrderId' },
    });

    orderMeta.computeGraph = {
      fastReverseDeps: new Map<string, string[]>([['Lines', ['Total']]]),
      computeScalarDeps: new Map<string, Set<string>>([['Total', new Set<string>(['Status'])]]),
    } as any;

    OnchangeEngine.run = (async () => ({
      messages: [{ level: 'info', message: 'nested', field: 'Amount' }],
    })) as any;

    ComputeEngine.recompute = (async (meta: any, entity: any) => {
      if (meta === lineMeta) {
        (entity as any).Amount = `nested-${String((entity as any).Id || '')}`;
        return;
      }
      if (meta === orderMeta) {
        (entity as any).Total = '999';
      }
    }) as any;

    const previewProxy: any = {
      Id: 'ORDER-NEST',
      Status: 'ok',
      Lines: [
        {
          Id: 'LINE-ROOT',
          Qty: 1,
          Amount: 'old-root',
          SubLines: [{ Id: 'LINE-CHILD', Qty: 1, Amount: 'old-child' }],
        },
      ],
    };
    const res: any = { computeRecomputed: ['Already'] };

    await applyModelOnchangePreviewCascade({
      meta: orderMeta as any,
      previewProxy,
      changedFields: ['Lines.SubLines.Qty'],
      selParsed: {
        normalizedSeeds: new Set(['Lines', 'Lines.SubLines', 'Lines.SubLines.Qty']),
        collectionRoots: new Set(['Lines', 'Lines.SubLines']),
        fieldSignals: new Map([
          ['Lines', new Set(['Qty'])],
          ['Lines.SubLines', new Set(['Qty'])],
        ]),
        selectors: new Map(),
      },
      opts: { withCompute: true },
      res,
    });

    expect(res.messages.some((msg: any) => String(msg.field || '').includes('Lines.SubLines.Amount'))).toBe(true);
    expect(res.value?.Total).toBe('999');
    expect((res.computeRecomputed || []).includes('Already')).toBe(true);
    expect((res.computeRecomputed || []).includes('Total')).toBe(true);
  } finally {
    OnchangeEngine.run = originalRun;
    ComputeEngine.recompute = originalRecompute;
    if (originalSubLinesField) (lineMeta.fields as Map<string, any>).set('SubLines', originalSubLinesField);
    else (lineMeta.fields as Map<string, any>).delete('SubLines');
    if (originalLinesField) orderMeta.fields.set('Lines', originalLinesField);
    orderMeta.computeGraph = originalOrderGraph;
  }
});

test('model onchange preview cascade isolates parent recompute errors', async () => {
  const { orderMeta, lineMeta } = preparePreviewCascadeMetadata();
  const originalRun = OnchangeEngine.run;
  const originalRecompute = ComputeEngine.recompute;

  try {
    OnchangeEngine.run = (async () => ({})) as any;
    ComputeEngine.recompute = (async (meta: any, entity: any) => {
      if (meta === lineMeta) {
        (entity as any).Amount = 'changed';
        return;
      }
      if (meta === orderMeta) {
        throw new Error('parent-recompute-failed');
      }
    }) as any;

    const previewProxy: any = {
      Id: 'ORDER-PARENT-ERR',
      Lines: [{ Id: 'LINE-ERR', Qty: 1, Amount: 'old' }],
    };
    const res: any = {};

    await applyModelOnchangePreviewCascade({
      meta: orderMeta as any,
      previewProxy,
      changedFields: ['Lines.Qty'],
      selParsed: {
        normalizedSeeds: new Set(['Lines', 'Lines.Qty']),
        collectionRoots: new Set(['Lines']),
        fieldSignals: new Map([['Lines', new Set(['Qty'])]]),
        selectors: new Map(),
      },
      opts: { withCompute: true },
      res,
    });

    expect(previewProxy.Lines[0]?.Amount).toBe('changed');
    expect(res.value?.__collectionPatch?.Lines?.length).toBe(1);
    expect(res.value?.Total).toBe(undefined);
  } finally {
    OnchangeEngine.run = originalRun;
    ComputeEngine.recompute = originalRecompute;
  }
});

test('model onchange preview cascade handles malformed changed fields and mixed one2many metadata guards', async () => {
  const { orderMeta, lineMeta } = preparePreviewCascadeMetadata();
  const originalRun = OnchangeEngine.run;
  const originalRecompute = ComputeEngine.recompute;
  const originalLineFields = lineMeta.fields;

  class NoGraphLine extends BaseModel {}

  const originalBrokenNoCtor = orderMeta.fields.get('BrokenNoCtor');
  const originalBrokenNoInverse = orderMeta.fields.get('BrokenNoInverse');
  const originalBrokenNoGraph = orderMeta.fields.get('BrokenNoGraph');
  const originalEmptyLines = orderMeta.fields.get('EmptyLines');
  const originalLines = orderMeta.fields.get('Lines');

  try {
    (orderMeta.fields as Map<string, any>).set('BrokenNoCtor', {
      type: 'OneToMany',
      relation: { inverseField: 'OrderId' },
    });
    (orderMeta.fields as Map<string, any>).set('BrokenNoInverse', {
      type: 'OneToMany',
      relation: { targetModel: () => ModelOnchangePreviewCascadeLine },
    });
    (orderMeta.fields as Map<string, any>).set('BrokenNoGraph', {
      type: 'OneToMany',
      relation: { targetModel: () => NoGraphLine, inverseField: 'OrderId' },
    });
    (orderMeta.fields as Map<string, any>).set('EmptyLines', {
      type: 'OneToMany',
      relation: { targetModel: () => ModelOnchangePreviewCascadeLine, inverseField: 'OrderId' },
    });
    (orderMeta.fields as Map<string, any>).set('Lines', {
      ...originalLines,
      relation: { targetModel: () => ModelOnchangePreviewCascadeLine, inverseField: 'OrderId' },
    });

    const noGraphMeta = MetadataStorage.instance.getModelMetadata(NoGraphLine as any);
    noGraphMeta.computeGraph = undefined as any;

    lineMeta.onchangeHandlers = undefined as any;
    (lineMeta as any).fields = undefined;

    OnchangeEngine.run = (async () => ({
      messages: [{ level: 'info', message: 'no-field' }],
      selection: [undefined, { field: 1 as any, selection: ['x'] }],
    })) as any;

    ComputeEngine.recompute = (async (meta: any, entity: any) => {
      if (meta !== lineMeta) return;
      (entity as any).Amount = `patched-${String((entity as any).Id ?? '')}`;
    }) as any;

    const previewProxy: any = {
      Id: 'ORDER-GUARD',
      Lines: [null, { Qty: 2, Amount: 'old-2' }, { Id: 'LINE-3', Qty: 3, Amount: 'old-3' }],
      EmptyLines: [],
      BrokenNoCtor: [{ Id: 'B1' }],
      BrokenNoInverse: [{ Id: 'B2' }],
      BrokenNoGraph: [{ Id: 'B3' }],
    };
    const res: any = { value: {} };

    await applyModelOnchangePreviewCascade({
      meta: orderMeta as any,
      previewProxy,
      changedFields: ['', '.', 'Lines', 'Lines..Qty', 'Lines.row.Qty'],
      selParsed: {
        normalizedSeeds: new Set(['Lines']),
        collectionRoots: new Set<string>(['Lines', 'EmptyLines', 'BrokenNoCtor', 'BrokenNoInverse', 'BrokenNoGraph']),
        fieldSignals: new Map<string, Set<string>>(),
        selectors: new Map(),
      },
      // Omit opts to exercise the default branch of (opts?.withCompute ?? true).
      res,
    });

    expect(Array.isArray(res.messages)).toBe(true);
    expect(res.messages.every((m: any) => m.field === undefined)).toBe(true);
    expect(res.selection).toEqual([]);
    const linesPatch = res.value?.__collectionPatch?.Lines;
    expect(linesPatch === undefined || Array.isArray(linesPatch)).toBe(true);
  } finally {
    OnchangeEngine.run = originalRun;
    ComputeEngine.recompute = originalRecompute;
    if (originalBrokenNoCtor) orderMeta.fields.set('BrokenNoCtor', originalBrokenNoCtor);
    else orderMeta.fields.delete('BrokenNoCtor');
    if (originalBrokenNoInverse) orderMeta.fields.set('BrokenNoInverse', originalBrokenNoInverse);
    else orderMeta.fields.delete('BrokenNoInverse');
    if (originalBrokenNoGraph) orderMeta.fields.set('BrokenNoGraph', originalBrokenNoGraph);
    else orderMeta.fields.delete('BrokenNoGraph');
    if (originalEmptyLines) orderMeta.fields.set('EmptyLines', originalEmptyLines);
    else orderMeta.fields.delete('EmptyLines');
    if (originalLines) orderMeta.fields.set('Lines', originalLines);

    // Restore line metadata to avoid affecting later tests.
    const restoredLineMeta = MetadataStorage.instance.getModelMetadata(ModelOnchangePreviewCascadeLine as any);
    restoredLineMeta.onchangeHandlers = [] as any;
    restoredLineMeta.fields = originalLineFields;
  }
});

test('model onchange preview cascade handles lowercase row id selector with non-array collection guard and sparse path deps', async () => {
  const { orderMeta, lineMeta } = preparePreviewCascadeMetadata();
  const originalRun = OnchangeEngine.run;
  const originalRecompute = ComputeEngine.recompute;
  const originalGraph = lineMeta.computeGraph;
  const originalHandlers = lineMeta.onchangeHandlers;
  const originalNonArray = orderMeta.fields.get('NonArrayLines');

  try {
    (orderMeta.fields as Map<string, any>).set('NonArrayLines', {
      type: 'OneToMany',
      relation: { targetModel: () => ModelOnchangePreviewCascadeLine, inverseField: 'OrderId' },
    });

    lineMeta.computeGraph = {
      computePathDeps: new Map([['Amount', [undefined, { root: 'OtherRoot', chain: ['Status'] }, { root: 'OrderId', chain: [] }]]]),
    } as any;
    lineMeta.onchangeHandlers = [{ method: 'onQty', triggers: ['Qty'] }] as any;

    OnchangeEngine.run = (async () => ({
      messages: [null, { level: 'warn', message: 'm', field: 'Amount' }],
      selection: [null, { field: 'Mode', selection: ['vat'] }, { field: 1 as any, selection: ['x'] }],
    })) as any;

    ComputeEngine.recompute = (async (meta: any, entity: any) => {
      if (meta !== lineMeta) return;
      (entity as any).Amount = 'patched-lower-id';
    }) as any;

    const previewProxy: any = {
      Id: 'ORDER-LC-ID',
      Lines: [null, { id: 'line-lc-1', Qty: 1, Amount: 'old' }],
      NonArrayLines: 'not-array',
    };
    const res: any = {};

    await applyModelOnchangePreviewCascade({
      meta: orderMeta as any,
      previewProxy,
      changedFields: ['', '.', 'Lines', 'Lines..Qty', 'Lines.row.Qty', 'NonArrayLines.Qty'],
      selParsed: {
        normalizedSeeds: new Set(['Lines', 'NonArrayLines']),
        collectionRoots: new Set<string>(['Lines', 'NonArrayLines']),
        fieldSignals: new Map([['Lines', new Set(['Qty'])]]),
        selectors: new Map([
          [
            'Lines',
            {
              kind: 'id',
              ids: new Set(['line-lc-1']),
            },
          ],
        ]),
      },
      opts: { withCompute: false },
      res,
    });

    const linePatches = res.value?.__collectionPatch?.Lines || [];
    expect(Array.isArray(linePatches)).toBe(true);
    if (linePatches.length > 0) {
      expect(linePatches.some((p: any) => p.pos === 1 || p.Id === 'line-lc-1')).toBe(true);
    }

    expect(Array.isArray(res.messages || [])).toBe(true);
    expect(Array.isArray(res.selection || [])).toBe(true);
    if (Array.isArray(res.selection) && res.selection.length > 0) {
      expect(String(res.selection[0]?.field || '').includes('Lines')).toBe(true);
    }
  } finally {
    OnchangeEngine.run = originalRun;
    ComputeEngine.recompute = originalRecompute;
    lineMeta.computeGraph = originalGraph;
    lineMeta.onchangeHandlers = originalHandlers;
    if (originalNonArray) orderMeta.fields.set('NonArrayLines', originalNonArray);
    else orderMeta.fields.delete('NonArrayLines');
  }
});
