// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { PathPrefetchPlan } from '../types';
import { getOnchangeRuntimeFlags } from '../constants';
import type { ModelCtor } from './shared';

/**
 * Serializable V2 snapshot of a path prefetch plan.
 */
export type PlanSkeletonV2 = {
  rootManyToOne: Record<string, string[]>;
  m2oChains: Record<string, string[][]>;
  collections: Record<string, string[][]>;
};

const cacheV2 = new WeakMap<ModelCtor, Map<string, PlanSkeletonV2>>();

/**
 * Appends source chains into a destination chain map.
 */
export function mergeChainMaps(src: Map<string, string[][]>, dst: Map<string, string[][]>) {
  src.forEach((chains, root) => {
    if (!dst.has(root)) dst.set(root, []);
    const arr = dst.get(root)!;
    arr.push(...chains.map(c => [...c]));
  });
}

/**
 * Truncates collection paths so only the first hop after the collection root remains.
 */
export function truncateCollectionPathsToFirst(src: Map<string, string[][]>): Map<string, string[][]> {
  const out = new Map<string, string[][]>();
  src.forEach((chains, root) => {
    const firstOnly = chains.filter(c => c.length >= 1).map(c => [c[0]]);
    if (firstOnly.length) out.set(root, firstOnly);
  });
  return out;
}

/**
 * Normalizes and deduplicates plan chains against runtime depth limits.
 */
export function normalizePlanChains(m: Map<string, string[][]>) {
  const runtimeFlags = getOnchangeRuntimeFlags();
  m.forEach((chains, root) => {
    const dedup = new Set<string>();
    const filtered = chains.filter(c => {
      const depth = c.length;
      if (depth > runtimeFlags.MAX_PATH_DEPTH && !runtimeFlags.ENABLE_MULTI_HOP_PREVIEW) return false;
      if (runtimeFlags.ENABLE_MULTI_HOP_PREVIEW && depth > runtimeFlags.MAX_MULTI_HOP_DEPTH) return false;
      const sig = c.join('.');
      if (dedup.has(sig)) return false;
      dedup.add(sig);
      return true;
    });
    m.set(root, filtered);
  });
}

/**
 * Builds the cache signature for a V2 path plan.
 */
export function makePlanSignatureV2(m2oChains: Map<string, string[][]>, collections: Map<string, string[][]>): string {
  const runtimeFlags = getOnchangeRuntimeFlags();
  const parts: string[] = [`V=${runtimeFlags.PLAN_SIGNATURE_VERSION}`, `D=${runtimeFlags.MAX_MULTI_HOP_DEPTH}`];

  const rootFirst = new Map<string, Set<string>>();
  m2oChains.forEach((chains, root) => {
    const set = rootFirst.get(root) || new Set<string>();
    for (const c of chains) if (c.length >= 1) set.add(c[0]);
    if (set.size) rootFirst.set(root, set);
  });

  const collFirst = new Map<string, Set<string>>();
  collections.forEach((chains, coll) => {
    const set = collFirst.get(coll) || new Set<string>();
    for (const c of chains) if (c.length >= 1) set.add(c[0]);
    if (set.size) collFirst.set(coll, set);
  });

  const rootKeys = [...rootFirst.keys()].sort();
  for (const r of rootKeys) {
    const fields = [...rootFirst.get(r)!].sort().join(',');
    parts.push(`R:${r}=${fields}`);
  }

  const collKeys = [...collFirst.keys()].sort();
  for (const c of collKeys) {
    const fields = [...collFirst.get(c)!].sort().join(',');
    parts.push(`C:${c}=${fields}`);
  }

  return parts.join('|');
}

/**
 * Converts a runtime plan into its serializable V2 skeleton.
 */
export function planToSkeletonV2(plan: PathPrefetchPlan): PlanSkeletonV2 {
  const r: Record<string, string[]> = {};
  const m: Record<string, string[][]> = {};
  const c: Record<string, string[][]> = {};

  plan.rootManyToOne.forEach((set, root) => (r[root] = [...set].sort()));
  plan.m2oChains.forEach((chains, root) => (m[root] = chains.map(ch => [...ch])));
  plan.collections.forEach((v, coll) => (c[coll] = v.chains.map(ch => [...ch])));

  return { rootManyToOne: r, m2oChains: m, collections: c };
}

/**
 * Reconstructs a runtime plan from a V2 skeleton.
 */
export function planFromSkeletonV2(s: PlanSkeletonV2): PathPrefetchPlan {
  const p: PathPrefetchPlan = {
    rootManyToOne: new Map(),
    m2oChains: new Map(),
    collections: new Map(),
  };
  Object.keys(s.rootManyToOne).forEach(root => p.rootManyToOne.set(root, new Set(s.rootManyToOne[root])));
  Object.keys(s.m2oChains).forEach(root =>
    p.m2oChains.set(
      root,
      s.m2oChains[root].map(ch => [...ch])
    )
  );
  Object.keys(s.collections).forEach(coll => p.collections.set(coll, { chains: s.collections[coll].map(ch => [...ch]) }));
  return p;
}

/**
 * Computes the effective maximum depth of a path plan.
 */
export function computePlanDepth(plan: PathPrefetchPlan): number {
  const runtimeFlags = getOnchangeRuntimeFlags();
  let max = 1;
  plan.m2oChains.forEach(chains => chains.forEach(c => (max = Math.max(max, Math.min(c.length, runtimeFlags.MAX_MULTI_HOP_DEPTH)))));
  plan.collections.forEach(v => v.chains.forEach(c => (max = Math.max(max, Math.min(c.length, runtimeFlags.MAX_MULTI_HOP_DEPTH)))));
  return max;
}

/**
 * Returns a cached V2 plan when available or builds and stores a new one.
 */
export function getCachedOrBuildPlanV2(
  modelCtor: ModelCtor,
  m2oReads: Map<string, string[][]>,
  collectionReads: Map<string, string[][]>,
  computeM2oPaths: Map<string, string[][]>,
  computeCollectionPaths: Map<string, string[][]>,
  buildPlan: (m2oChains: Map<string, string[][]>, collections: Map<string, string[][]>) => { plan: PathPrefetchPlan; pathDepthMax: number }
): { plan: PathPrefetchPlan; fromCache: boolean; signature: string; pathDepthMax: number } {
  const runtimeFlags = getOnchangeRuntimeFlags();
  const m2oChains = new Map<string, string[][]>();
  const collections = new Map<string, string[][]>();

  mergeChainMaps(m2oReads, m2oChains);
  mergeChainMaps(computeM2oPaths, m2oChains);
  mergeChainMaps(collectionReads, collections);
  mergeChainMaps(truncateCollectionPathsToFirst(computeCollectionPaths), collections);

  normalizePlanChains(m2oChains);
  normalizePlanChains(collections);

  const signature = makePlanSignatureV2(m2oChains, collections);

  if (!runtimeFlags.PLAN_CACHE_ENABLED) {
    const built = buildPlan(m2oChains, collections);
    return { plan: built.plan, fromCache: false, signature, pathDepthMax: built.pathDepthMax };
  }

  let store = cacheV2.get(modelCtor);
  if (!store) {
    store = new Map<string, PlanSkeletonV2>();
    cacheV2.set(modelCtor, store);
  }

  const hit = store.get(signature);
  if (hit) {
    const plan = planFromSkeletonV2(hit);
    return { plan, fromCache: true, signature, pathDepthMax: computePlanDepth(plan) };
  }

  const built = buildPlan(m2oChains, collections);
  store.set(signature, planToSkeletonV2(built.plan));
  return { plan: built.plan, fromCache: false, signature, pathDepthMax: built.pathDepthMax };
}
