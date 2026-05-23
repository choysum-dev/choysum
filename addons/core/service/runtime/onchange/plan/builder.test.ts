// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { PathPrefetchPlan } from '../types';
import {
  ensureDepthAllowed,
  finalizePlan,
  mergeCollectionsIntoPlan,
  mergeComputePathsIntoPlan,
  mergeM2OChainsIntoPlan,
  mergeOnchangeReadsExIntoPlan,
  mergeOnchangeReadsIntoPlan,
} from './builder';

function makePlan(): PathPrefetchPlan {
  return {
    rootManyToOne: new Map(),
    m2oChains: new Map(),
    collections: new Map(),
  };
}

test('plan builder merges onchange reads and compute paths into deduplicated root plan', () => {
  const plan = makePlan();

  mergeOnchangeReadsExIntoPlan(plan, {
    m2o: new Map([['PartnerId', [['Name'], ['Name']]]]),
    collections: new Map([['Lines', [['Product'], ['Product']]]]),
  });

  mergeComputePathsIntoPlan(plan, new Map([['PartnerId', [['Code'], ['Name']]]]), new Map([['Lines', [['Qty'], ['Product']]]]));

  const depth = finalizePlan(plan);

  expect(depth).toBe(1);
  expect(Array.from(plan.rootManyToOne.get('PartnerId') || []).sort()).toEqual(['Code', 'Name']);
  expect(plan.m2oChains.get('PartnerId')).toEqual([['Name'], ['Code']]);
  expect(plan.collections.get('Lines')?.chains).toEqual([['Product'], ['Qty']]);
});

test('plan builder rejects paths deeper than the allowed preview depth', () => {
  let error: Error | undefined;

  try {
    ensureDepthAllowed(['CompanyId', 'ParentId', 'Name'], 'compute path "PartnerId.CompanyId.ParentId.Name"');
  } catch (cause) {
    error = cause as Error;
  }

  expect(error?.message).toBe('PathPlanBuilder: compute path "PartnerId.CompanyId.ParentId.Name" depth (3) exceeds limit (2)');
});

test('plan builder merges legacy reads map and skips empty m2o chains', () => {
  const plan = makePlan();

  mergeOnchangeReadsIntoPlan(plan, new Map([['PartnerId', [['Name'], ['Code']]]]));
  mergeM2OChainsIntoPlan(plan, new Map([['PartnerId', [[], ['CompanyId']]]]));

  const depth = finalizePlan(plan);

  expect(depth).toBe(1);
  expect(plan.m2oChains.get('PartnerId')).toEqual([['Name'], ['Code'], ['CompanyId']]);
  expect(Array.from(plan.rootManyToOne.get('PartnerId') || []).sort()).toEqual(['Code', 'CompanyId', 'Name']);
});

test('plan builder skips empty collection chains and backfills root fields in finalize', () => {
  const plan = makePlan();

  // Intentionally bypass merge helpers to verify finalize backfill branch.
  plan.m2oChains.set('OwnerId', [['DisplayName']]);
  mergeCollectionsIntoPlan(plan, new Map([['Lines', [[], ['Qty']]]]));

  const depth = finalizePlan(plan);

  expect(depth).toBe(1);
  expect(plan.collections.get('Lines')?.chains).toEqual([['Qty']]);
  expect(Array.from(plan.rootManyToOne.get('OwnerId') || [])).toEqual(['DisplayName']);
});

test('ensureDepthAllowed accepts empty chain', () => {
  expect(() => ensureDepthAllowed([], 'empty')).not.toThrow();
});

test('plan builder mergeComputePaths covers existing roots, empty chains, optional collection paths and existing collections', () => {
  const plan = makePlan();

  plan.m2oChains.set('PartnerId', [['Existing']]);
  plan.rootManyToOne.set('PartnerId', new Set(['Existing']));
  plan.collections.set('Lines', { chains: [['ExistingLine']] });

  // Cover the existing-root path, empty-chain skip, and the branch where collectionPaths is omitted.
  mergeComputePathsIntoPlan(plan, new Map([['PartnerId', [[], ['Name']]]]));

  expect(plan.m2oChains.get('PartnerId')).toEqual([['Existing'], ['Name']]);
  expect(Array.from(plan.rootManyToOne.get('PartnerId') || []).sort()).toEqual(['Existing', 'Name']);

  // Cover the branch where collectionPaths exists and the collection bucket already exists.
  mergeComputePathsIntoPlan(plan, new Map([['PartnerId', [['Code']]]]), new Map([['Lines', [['Qty']]]]));

  expect(plan.m2oChains.get('PartnerId')).toEqual([['Existing'], ['Name'], ['Code']]);
  expect(plan.collections.get('Lines')?.chains).toEqual([['ExistingLine'], ['Qty']]);
});

test('plan builder mergeComputePaths initializes missing root and collection buckets', () => {
  const plan = makePlan();

  mergeComputePathsIntoPlan(plan, new Map([['PartnerId', [['Code']]]]), new Map([['Lines', [['Qty']]]]));

  expect(plan.m2oChains.get('PartnerId')).toEqual([['Code']]);
  expect(Array.from(plan.rootManyToOne.get('PartnerId') || [])).toEqual(['Code']);
  expect(plan.collections.get('Lines')?.chains).toEqual([['Qty']]);
});

test('plan builder ensureDepthAllowed uses runtime override when multi-hop preview is disabled', () => {
  const g = globalThis as any;
  const previous = g.__CHOYSUM_TEST_ONCHANGE_FLAGS__;

  g.__CHOYSUM_TEST_ONCHANGE_FLAGS__ = {
    ...(previous || {}),
    ENABLE_MULTI_HOP_PREVIEW: false,
    MAX_PATH_DEPTH: 1,
    MAX_MULTI_HOP_DEPTH: 2,
  };

  try {
    expect(() => ensureDepthAllowed(['PartnerId', 'Name'], 'runtime-disabled multi-hop')).toThrow('exceeds limit (1)');
  } finally {
    if (previous === undefined) {
      delete g.__CHOYSUM_TEST_ONCHANGE_FLAGS__;
    } else {
      g.__CHOYSUM_TEST_ONCHANGE_FLAGS__ = previous;
    }
  }
});
