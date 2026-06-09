// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as computeDepsApi from './compute_deps';
import * as planApi from './index';
import { PathPlanExecutor } from './executor';
import { PathPlanBuilder } from './index';

test('path plan entrypoint export surface stays limited to facade and compute dependency re-exports', () => {
  expect(Object.keys(planApi).sort()).toEqual(['PathPlanBuilder', 'extractComputeCollectionPathDeps', 'extractComputePathDeps']);
  expect(planApi.extractComputePathDeps).toBe(computeDepsApi.extractComputePathDeps);
  expect(planApi.extractComputeCollectionPathDeps).toBe(computeDepsApi.extractComputeCollectionPathDeps);
});

test('path plan builder facade keeps fluent merge and finalize behavior stable', () => {
  const builder = PathPlanBuilder.builder()
    .mergeOnchangeReadsEx({
      m2o: new Map([['PartnerId', [['Name']]]]),
      collections: new Map([['Lines', [['Qty']]]]),
    })
    .mergeComputePaths(new Map([['PartnerId', [['CompanyId', 'Name']]]]), new Map([['Lines', [['ProductId']]]]))
    .finalize();

  expect(builder.getPathDepthMax()).toBe(2);
  expect(Array.from(builder.getPlan().rootManyToOne.get('PartnerId') || []).sort()).toEqual(['CompanyId', 'Name']);
  expect(builder.getPlan().m2oChains.get('PartnerId')).toEqual([['Name'], ['CompanyId', 'Name']]);
  expect(builder.getPlan().collections.get('Lines')?.chains).toEqual([['Qty'], ['ProductId']]);
});

test('path plan builder facade delegates execute paths to PathPlanExecutor and normalizes missing maps', async () => {
  const originalExecute = PathPlanExecutor.prototype.execute;
  const calls: Array<{ plan: any; meta: any; draft: any }> = [];
  const sentinelStats = { batches: [], totalBatches: 9, totalRows: 13 };

  PathPlanExecutor.prototype.execute = async function (this: any, meta: any, draft: any) {
    calls.push({ plan: (this as any).plan, meta, draft });
    return sentinelStats as any;
  } as any;

  try {
    const builder = PathPlanBuilder.builder()
      .mergeM2OChains(new Map([['PartnerId', [['Name']]]]))
      .finalize();
    const meta = { marker: 'meta' };
    const draft = { Id: 'ROW-1', PartnerId: 'P1' };

    const builderStats = await builder.execute(undefined as any, meta as any, draft);
    const staticStats = await PathPlanBuilder.executeWithPlan(undefined as any, meta as any, draft, {
      rootManyToOne: new Map([['PartnerId', new Set(['Name'])]]),
      m2oChains: undefined as any,
      collections: undefined as any,
    } as any);

    expect(builderStats).toBe(sentinelStats as any);
    expect(staticStats).toBe(sentinelStats as any);
    expect(calls.length).toBe(2);
    expect(calls[0]?.plan).toBe(builder.getPlan());
    expect(calls[0]?.meta).toBe(meta as any);
    expect(calls[0]?.draft).toBe(draft);
    expect(calls[1]?.plan.rootManyToOne.get('PartnerId') instanceof Set).toBe(true);
    expect(calls[1]?.plan.m2oChains instanceof Map).toBe(true);
    expect(calls[1]?.plan.collections instanceof Map).toBe(true);
    expect(calls[1]?.plan.m2oChains.size).toBe(0);
    expect(calls[1]?.plan.collections.size).toBe(0);
  } finally {
    PathPlanExecutor.prototype.execute = originalExecute;
  }
});

test('path plan builder static helpers createEmptyPlan and computeDepth stay stable', () => {
  const empty = PathPlanBuilder.createEmptyPlan();
  expect(empty.rootManyToOne instanceof Map).toBe(true);
  expect(empty.m2oChains instanceof Map).toBe(true);
  expect(empty.collections instanceof Map).toBe(true);
  expect(empty.rootManyToOne.size).toBe(0);

  const builder = PathPlanBuilder.builder()
    .mergeOnchangeReads(new Map([['PartnerId', [['Name']]]]))
    .finalize();

  const depth = PathPlanBuilder.computeDepth(builder.getPlan());
  expect(depth).toBe(builder.getPathDepthMax());
});
