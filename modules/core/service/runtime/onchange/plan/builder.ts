// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { PathPrefetchPlan } from '../types';
import { getOnchangeRuntimeFlags } from '../constants';
import { computePlanDepth } from './cache';

/**
 * Enforces runtime depth limits for a path chain.
 */
export function ensureDepthAllowed(chain: string[], label: string, _isCollection = false) {
  const depth = chain.length;
  const runtimeFlags = getOnchangeRuntimeFlags();
  if (!depth) return;
  if (!runtimeFlags.ENABLE_MULTI_HOP_PREVIEW && depth > runtimeFlags.MAX_PATH_DEPTH) {
    throw new Error(`PathPlanBuilder: ${label} depth (${depth}) exceeds limit (${runtimeFlags.MAX_PATH_DEPTH}) while multi-hop is disabled`);
  }
  if (runtimeFlags.ENABLE_MULTI_HOP_PREVIEW && depth > runtimeFlags.MAX_MULTI_HOP_DEPTH) {
    throw new Error(`PathPlanBuilder: ${label} depth (${depth}) exceeds limit (${runtimeFlags.MAX_MULTI_HOP_DEPTH})`);
  }
}

/**
 * Merges parsed onchange reads that already separate ManyToOne and collection roots.
 */
export function mergeOnchangeReadsExIntoPlan(plan: PathPrefetchPlan, parsed: { m2o: Map<string, string[][]>; collections: Map<string, string[][]> }) {
  parsed.m2o.forEach((chains, root) => {
    if (!plan.m2oChains.has(root)) plan.m2oChains.set(root, []);
    const arr = plan.m2oChains.get(root)!;
    for (const ch of chains) {
      ensureDepthAllowed(ch, `reads m2o "${root}.${ch.join('.')}"`);
      arr.push(ch);
      if (!plan.rootManyToOne.has(root)) plan.rootManyToOne.set(root, new Set());
      if (ch.length >= 1) plan.rootManyToOne.get(root)!.add(ch[0]);
    }
  });

  parsed.collections.forEach((chains, coll) => {
    if (!plan.collections.has(coll)) plan.collections.set(coll, { chains: [] });
    const v = plan.collections.get(coll)!;
    for (const ch of chains) {
      ensureDepthAllowed(ch, `reads collection "${coll}.${ch.join('.')}"`, true);
      v.chains.push(ch);
    }
  });
}

/**
 * Merges ManyToOne-only onchange reads into a path plan.
 */
export function mergeOnchangeReadsIntoPlan(plan: PathPrefetchPlan, readsMap: Map<string, string[][]>) {
  readsMap.forEach((chains, root) => {
    if (!plan.m2oChains.has(root)) plan.m2oChains.set(root, []);
    const arr = plan.m2oChains.get(root)!;
    for (const ch of chains) {
      ensureDepthAllowed(ch, `reads m2o "${root}.${ch.join('.')}"`);
      arr.push(ch);
      if (!plan.rootManyToOne.has(root)) plan.rootManyToOne.set(root, new Set());
      if (ch.length >= 1) plan.rootManyToOne.get(root)!.add(ch[0]);
    }
  });
}

/**
 * Merges compute-driven path requirements into a path plan.
 */
export function mergeComputePathsIntoPlan(plan: PathPrefetchPlan, m2oPaths: Map<string, string[][]>, collectionPaths?: Map<string, string[][]>) {
  m2oPaths.forEach((chains, root) => {
    if (!plan.m2oChains.has(root)) plan.m2oChains.set(root, []);
    const arr = plan.m2oChains.get(root)!;
    for (const ch of chains) {
      if (!ch.length) continue;
      ensureDepthAllowed(ch, `compute path "${root}.${ch.join('.')}"`);
      arr.push(ch);
      if (!plan.rootManyToOne.has(root)) plan.rootManyToOne.set(root, new Set());
      plan.rootManyToOne.get(root)!.add(ch[0]);
    }
  });

  if (!collectionPaths) return;

  collectionPaths.forEach((chains, coll) => {
    if (!plan.collections.has(coll)) plan.collections.set(coll, { chains: [] });
    const v = plan.collections.get(coll)!;
    for (const ch of chains) {
      ensureDepthAllowed(ch, `compute collection "${coll}.${ch.join('.')}"`, true);
      v.chains.push(ch);
    }
  });
}

/**
 * Merges ManyToOne chains into a path plan.
 */
export function mergeM2OChainsIntoPlan(plan: PathPrefetchPlan, m2o: Map<string, string[][]>) {
  m2o.forEach((chains, root) => {
    if (!plan.m2oChains.has(root)) plan.m2oChains.set(root, []);
    const arr = plan.m2oChains.get(root)!;
    for (const ch of chains) {
      if (!ch.length) continue;
      ensureDepthAllowed(ch, `m2o "${root}.${ch.join('.')}"`);
      arr.push(ch);
      if (!plan.rootManyToOne.has(root)) plan.rootManyToOne.set(root, new Set());
      plan.rootManyToOne.get(root)!.add(ch[0]);
    }
  });
}

/**
 * Merges collection chains into a path plan.
 */
export function mergeCollectionsIntoPlan(plan: PathPrefetchPlan, colls: Map<string, string[][]>) {
  colls.forEach((chains, coll) => {
    if (!plan.collections.has(coll)) plan.collections.set(coll, { chains: [] });
    const v = plan.collections.get(coll)!;
    for (const ch of chains) {
      if (!ch.length) continue;
      ensureDepthAllowed(ch, `collection "${coll}.${ch.join('.')}"`, true);
      v.chains.push(ch);
    }
  });
}

/**
 * Deduplicates plan chains and returns the final computed depth.
 */
export function finalizePlan(plan: PathPrefetchPlan): number {
  plan.m2oChains.forEach((chains, root) => {
    const dedup = new Set<string>();
    const filtered = chains.filter(c => {
      const sig = c.join('.');
      if (dedup.has(sig)) return false;
      dedup.add(sig);
      return true;
    });
    plan.m2oChains.set(root, filtered);
  });

  plan.collections.forEach(v => {
    const dedup = new Set<string>();
    const filtered = v.chains.filter(c => {
      const sig = c.join('.');
      if (dedup.has(sig)) return false;
      dedup.add(sig);
      return true;
    });
    v.chains = filtered;
  });

  plan.m2oChains.forEach((chains, root) => {
    if (!plan.rootManyToOne.has(root)) plan.rootManyToOne.set(root, new Set());
    const set = plan.rootManyToOne.get(root)!;
    for (const ch of chains) if (ch.length >= 1) set.add(ch[0]);
  });

  return computePlanDepth(plan);
}
