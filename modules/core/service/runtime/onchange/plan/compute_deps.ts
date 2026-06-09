// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../../orm/metadata/model';

/**
 * Extract regular path dependencies from computeGraph.parsedDeps as root -> chains[].
 * toRecompute is the set of compute fields that need recomputation.
 */
export function extractComputePathDeps(meta: ModelMetadata, toRecompute: Set<string>): Map<string, string[][]> {
  const result = new Map<string, string[][]>();
  const g = meta.computeGraph;
  if (!g) return result;

  for (const f of toRecompute) {
    const deps = g.parsedDeps.get(f) || [];
    for (const d of deps) {
      if (d.kind === 'path') {
        if (!result.has(d.root)) result.set(d.root, []);
        result.get(d.root)!.push(d.chain);
      }
    }
  }

  result.forEach((chains, root) => {
    const dedup = new Set<string>();
    const filtered = chains.filter(c => {
      const sig = c.join('.');
      if (dedup.has(sig)) return false;
      dedup.add(sig);
      return true;
    });
    result.set(root, filtered);
  });

  return result;
}

/**
 * Extract collection-path dependencies from computeGraph as collection -> chains[].
 */
export function extractComputeCollectionPathDeps(meta: ModelMetadata, toRecompute: Set<string>): Map<string, string[][]> {
  const result = new Map<string, string[][]>();
  const g = meta.computeGraph;
  if (!g) return result;

  for (const f of toRecompute) {
    const deps = g.computeCollectionPathDeps?.get(f) || [];
    for (const d of deps) {
      const root = d.collection;
      if (!result.has(root)) result.set(root, []);
      result.get(root)!.push(d.chain);
    }
  }

  result.forEach((chains, root) => {
    const dedup = new Set<string>();
    const filtered = chains.filter(c => {
      const sig = c.join('.');
      if (dedup.has(sig)) return false;
      dedup.add(sig);
      return true;
    });
    result.set(root, filtered);
  });

  return result;
}
