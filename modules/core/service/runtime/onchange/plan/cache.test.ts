// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel } from '@/core/service';
import type { PathPrefetchPlan } from '../types';
import { computePlanDepth, getCachedOrBuildPlanV2 } from './cache';

class PlanCacheModel extends BaseModel {}

function makePlan(): PathPrefetchPlan {
  return {
    rootManyToOne: new Map(),
    m2oChains: new Map(),
    collections: new Map(),
  };
}

test('getCachedOrBuildPlanV2 normalizes merged chains and returns cloned cached plans', () => {
  const buildCalls: Array<{
    m2oChains: Array<[string, string[][]]>;
    collections: Array<[string, string[][]]>;
  }> = [];

  const buildPlan = (m2oChains: Map<string, string[][]>, collections: Map<string, string[][]>) => {
    buildCalls.push({
      m2oChains: Array.from(m2oChains.entries()).map(([root, chains]) => [root, chains.map(chain => [...chain])]),
      collections: Array.from(collections.entries()).map(([root, chains]) => [root, chains.map(chain => [...chain])]),
    });

    return {
      plan: {
        rootManyToOne: new Map([['PartnerId', new Set(['Email', 'Name'])]]),
        m2oChains: new Map([['PartnerId', [['Name'], ['Email'], ['Email', 'Code']]]]),
        collections: new Map([['Lines', { chains: [['Product'], ['Qty']] }]]),
      } satisfies PathPrefetchPlan,
      pathDepthMax: 2,
    };
  };

  const first = getCachedOrBuildPlanV2(
    PlanCacheModel as any,
    new Map([['PartnerId', [['Name'], ['Name'], ['Email', 'Code'], ['Ignored', 'Too', 'Deep']]]]),
    new Map([['Lines', [['Product'], ['Product']]]]),
    new Map([['PartnerId', [['Email'], ['Email']]]]),
    new Map([['Lines', [['Product', 'Name'], ['Qty']]]]),
    buildPlan
  );

  expect(first.fromCache).toBe(false);
  expect(first.signature).toBe('V=1|D=2|R:PartnerId=Email,Name|C:Lines=Product,Qty');
  expect(first.pathDepthMax).toBe(2);
  expect(buildCalls).toEqual([
    {
      m2oChains: [['PartnerId', [['Name'], ['Email', 'Code'], ['Email']]]],
      collections: [['Lines', [['Product'], ['Qty']]]],
    },
  ]);

  first.plan.rootManyToOne.get('PartnerId')?.add('Mutated');
  first.plan.m2oChains.get('PartnerId')?.[0].push('Mutated');
  first.plan.collections.get('Lines')?.chains[0].push('Mutated');

  const second = getCachedOrBuildPlanV2(
    PlanCacheModel as any,
    new Map([['PartnerId', [['Name'], ['Name'], ['Email', 'Code'], ['Ignored', 'Too', 'Deep']]]]),
    new Map([['Lines', [['Product'], ['Product']]]]),
    new Map([['PartnerId', [['Email'], ['Email']]]]),
    new Map([['Lines', [['Product', 'Name'], ['Qty']]]]),
    buildPlan
  );

  expect(second.fromCache).toBe(true);
  expect(buildCalls.length).toBe(1);
  expect(Array.from(second.plan.rootManyToOne.get('PartnerId') || []).sort()).toEqual(['Email', 'Name']);
  expect(second.plan.m2oChains.get('PartnerId')).toEqual([['Name'], ['Email'], ['Email', 'Code']]);
  expect(second.plan.collections.get('Lines')?.chains).toEqual([['Product'], ['Qty']]);
});

test('computePlanDepth returns the capped maximum chain depth across plan sections', () => {
  const plan = makePlan();
  plan.m2oChains.set('PartnerId', [['Name'], ['CompanyId', 'Name'], ['Too', 'Deep', 'Ignored']]);
  plan.collections.set('Lines', { chains: [['Product'], ['Product', 'Name']] });

  expect(computePlanDepth(plan)).toBe(2);
});

test('plan cache runtime overrides cover disabled preview filtering and cache bypass branch', () => {
  const g = globalThis as any;
  const previous = g.__CHOYSUM_TEST_ONCHANGE_FLAGS__;

  g.__CHOYSUM_TEST_ONCHANGE_FLAGS__ = {
    ...(previous || {}),
    ENABLE_MULTI_HOP_PREVIEW: false,
    MAX_PATH_DEPTH: 1,
    MAX_MULTI_HOP_DEPTH: 2,
    PLAN_CACHE_ENABLED: false,
  };

  const calls: Array<{ m2oChains: Array<[string, string[][]]>; collections: Array<[string, string[][]]> }> = [];

  try {
    const result = getCachedOrBuildPlanV2(
      PlanCacheModel as any,
      new Map(),
      new Map(),
      new Map([['PartnerId', [['Name'], ['Too', 'Deep']]]]),
      new Map(),
      (m2oChains, collections) => {
        calls.push({
          m2oChains: Array.from(m2oChains.entries()).map(([root, chains]) => [root, chains.map(chain => [...chain])]),
          collections: Array.from(collections.entries()).map(([root, chains]) => [root, chains.map(chain => [...chain])]),
        });
        return { plan: makePlan(), pathDepthMax: 1 };
      }
    );

    expect(result.fromCache).toBe(false);
    expect(calls).toEqual([
      {
        m2oChains: [['PartnerId', [['Name']]]],
        collections: [],
      },
    ]);
  } finally {
    if (previous === undefined) {
      delete g.__CHOYSUM_TEST_ONCHANGE_FLAGS__;
    } else {
      g.__CHOYSUM_TEST_ONCHANGE_FLAGS__ = previous;
    }
  }
});
