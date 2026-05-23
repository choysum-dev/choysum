// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Phase 1 keeps only reads parsing here, while prefetch logic lives in PathPlanBuilder.execute from plan.ts.
// Extensions:
//  - parseOnchangeReadsEx adds support for collection chains such as Lines.Product.Name with strict validation.
//  - parseOnchangeReads remains backward compatible and returns only the ManyToOne portion.

import type { ModelMetadata, OnchangeHandlerMeta } from '../../orm/metadata/model';
import { MetadataStorage } from '../../orm/metadata/storage';
import { MAX_PATH_DEPTH, ENABLE_MULTI_HOP_PREVIEW, ENABLE_COLLECTION_PATH_READS, MAX_MULTI_HOP_DEPTH } from './constants';

/**
 * Reads map keyed by root field, where each value stores the remaining path chains.
 */
export type PathReadsMap = Map<string, string[][]>; // root -> chains, where each chain stores the remaining segments.

/**
 * Parsed reads split into ManyToOne and collection-root paths.
 */
export interface ParsedReads {
  m2o: Map<string, string[][]>;
  collections: Map<string, string[][]>;
}

/**
 * Strict chain validation for ManyToOne paths, with optional multi-hop support.
 * Rules:
 *  - The root must be ManyToOne whenever a chain exists.
 *  - Every segment along the chain must exist in the corresponding model metadata.
 *  - Intermediate segments must stay ManyToOne to continue deeper.
 *  - Collection segments are never allowed inside the chain.
 */
function validateChainStrict(meta: ModelMetadata, root: string, chain: string[]): void {
  // parseOnchangeReadsEx already guarantees that the root exists and is ManyToOne when a chain is present.
  const rootMeta = meta.fields.get(root)!;
  const rel = rootMeta.relation;
  if (!rel?.targetModel) {
    throw new Error(`invalid reads path "${root}.${chain.join('.')}"; root "${root}" is missing a targetModel definition`);
  }
  let currentCtor = rel.targetModel();
  let currentMeta = MetadataStorage.instance.getModelMetadata(currentCtor);

  for (let i = 0; i < chain.length; i++) {
    const seg = chain[i];
    const f = currentMeta.fields.get(seg);
    const modelName = currentMeta.fullModelName || currentMeta.modelName || currentMeta.className || 'Unknown';
    if (!f) {
      throw new Error(`invalid reads path "${root}.${chain.join('.')}"; segment "${seg}" does not exist on model ${modelName}`);
    }
    if (f.type === 'OneToMany' || f.type === 'ManyToMany') {
      // Even when collection-path reads are enabled, nested collection segments are not allowed here.
      throw new Error(`invalid reads path "${root}.${chain.join('.')}"; intermediate segment "${seg}" cannot be a collection field`);
    }
    const notLast = i < chain.length - 1;
    if (notLast) {
      if (f.type !== 'ManyToOne') {
        throw new Error(`invalid reads path "${root}.${chain.join('.')}"; intermediate segment "${seg}" must be ManyToOne`);
      }
      const nextCtor = f.relation?.targetModel?.();
      if (!nextCtor) {
        throw new Error(`invalid reads path "${root}.${chain.join('.')}"; segment "${seg}" is missing a targetModel definition`);
      }
      currentCtor = nextCtor;
      currentMeta = MetadataStorage.instance.getModelMetadata(currentCtor);
    }
  }
}

/**
 * Strict validation for collection chains rooted at OneToMany or ManyToMany fields.
 * Rules:
 *  - The root must be a collection field.
 *  - Validation starts from the child model and checks each segment in order.
 *  - Intermediate segments must be ManyToOne and may not re-enter collections.
 *  - The last segment may be scalar or ManyToOne.
 */
function validateCollectionChainStrict(meta: ModelMetadata, rootCollection: string, chain: string[]): void {
  // parseOnchangeReadsEx already guarantees that the root exists and is a collection field.
  const rootMeta = meta.fields.get(rootCollection)!;
  const rel = rootMeta.relation;
  const targetCtor = rel?.targetModel?.();
  if (!targetCtor) {
    throw new Error(`invalid collection reads path "${rootCollection}.${chain.join('.')}"; root "${rootCollection}" is missing a targetModel definition`);
  }
  let currentMeta = MetadataStorage.instance.getModelMetadata(targetCtor);

  for (let i = 0; i < chain.length; i++) {
    const seg = chain[i];
    const f = currentMeta.fields.get(seg);
    const modelName = currentMeta.fullModelName || currentMeta.modelName || currentMeta.className || 'Unknown';
    if (!f) {
      throw new Error(`invalid collection reads path "${rootCollection}.${chain.join('.')}"; segment "${seg}" does not exist on model ${modelName}`);
    }
    if (f.type === 'OneToMany' || f.type === 'ManyToMany') {
      throw new Error(`invalid collection reads path "${rootCollection}.${chain.join('.')}"; nested collection segment "${seg}" is not allowed`);
    }
    const notLast = i < chain.length - 1;
    if (notLast) {
      if (f.type !== 'ManyToOne') {
        throw new Error(`invalid collection reads path "${rootCollection}.${chain.join('.')}"; intermediate segment "${seg}" must be ManyToOne`);
      }
      const nextCtor = f.relation?.targetModel?.();
      if (!nextCtor) {
        throw new Error(`invalid collection reads path "${rootCollection}.${chain.join('.')}"; segment "${seg}" is missing a targetModel definition`);
      }
      currentMeta = MetadataStorage.instance.getModelMetadata(nextCtor);
    }
  }
}

/**
 * Extended reads parser:
 * - m2o: path sets rooted at scalar or ManyToOne fields (root -> chains)
 * - collections: path sets rooted at collection fields (root -> chains). When only the root is declared,
 *   chains may stay empty to mean that root access alone is allowed.
 */
export function parseOnchangeReadsEx(meta: ModelMetadata, activeHandlers: OnchangeHandlerMeta[]): ParsedReads {
  const m2o: Map<string, string[][]> = new Map();
  const collections: Map<string, string[][]> = new Map();

  for (const h of activeHandlers) {
    for (const r of h.reads || []) {
      if (!r) continue;
      const segs = r.split('.').filter(Boolean);
      if (!segs.length) continue;

      const root = segs[0];
      const chain = segs.slice(1);

      const rootMeta = meta.fields.get(root);
      if (!rootMeta) {
        throw new Error(`root field "${root}" from reads path "${r}" does not exist on the model`);
      }

      // Collection root.
      if (rootMeta.type === 'OneToMany' || rootMeta.type === 'ManyToMany') {
        if (!collections.has(root)) collections.set(root, []);

        // Depth checks for collection chains are controlled by ENABLE_MULTI_HOP_PREVIEW and MAX_MULTI_HOP_DEPTH.
        if (ENABLE_MULTI_HOP_PREVIEW && chain.length > MAX_MULTI_HOP_DEPTH) {
          throw new Error(`collection reads path "${r}" exceeds the allowed depth (${MAX_MULTI_HOP_DEPTH})`);
        }

        // Strict validation.
        validateCollectionChainStrict(meta, root, chain);

        // Root-only declarations allow reading the collection root without adding an empty chain.
        if (chain.length) {
          collections.get(root)!.push(chain);
        }
        continue;
      }

      // Non-collection roots such as scalar, ManyToOne, or compute fields.

      // Whenever a chain exists, the root must be ManyToOne and the cross-model chain must validate strictly.
      if (chain.length >= 1) {
        if (rootMeta.type !== 'ManyToOne') {
          throw new Error(`root "${root}" from reads path "${r}" must be a ManyToOne field before continuing to "${chain[0]}"`);
        }
        validateChainStrict(meta, root, chain);
      }

      if (!m2o.has(root)) m2o.set(root, []);
      if (chain.length) {
        m2o.get(root)!.push(chain);
      }
      // Scalar roots do not push empty chains.
    }
  }

  // Deduplicate chains.
  const dedupMap = (map: Map<string, string[][]>) => {
    map.forEach((chains, root) => {
      if (!chains.length) return;
      const dedup = new Set<string>();
      const filtered = chains.filter(c => {
        const sig = c.join('.');
        if (dedup.has(sig)) return false;
        dedup.add(sig);
        return true;
      });
      map.set(root, filtered);
    });
  };
  dedupMap(m2o);
  dedupMap(collections);

  return { m2o, collections };
}

/**
 * Backward-compatible wrapper that returns only the ManyToOne portion for existing plan builders.
 */
export function parseOnchangeReads(meta: ModelMetadata, activeHandlers: OnchangeHandlerMeta[]): PathReadsMap {
  const parsed = parseOnchangeReadsEx(meta, activeHandlers);
  return parsed.m2o;
}
