// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { OnchangeHandlerMeta, ModelMetadata } from '../metadata/model';
import type { ModelCtor } from '../metadata/field';
import type { OnchangeDraft, PrefetchExecStats } from '../../runtime/onchange/types';
import { parseOnchangeReadsEx } from '../../runtime/onchange/reads';
import { PathPlanBuilder } from '../../runtime/onchange/plan';
import { extractComputePathDeps, extractComputeCollectionPathDeps } from '../../runtime/onchange/plan';
import { createPreviewProxy } from '../../runtime/proxy';
import { buildNeededFields } from '../../runtime/onchange/needed';
import { parseChangedSelectors, type ParsedChangedSelectors } from '../../runtime/onchange/selectors';
import type BaseModel from './model';
import { getModelRepository } from './model_internal_facade';
import { getModelRuntimeMetadata } from './model_runtime_service_facade';
import type { ObjectRecord } from '../../../utils/types';

type ModelOnchangePrepareParams = {
  ModelCtor: ModelCtor<BaseModel> & typeof BaseModel;
  draft: OnchangeDraft;
  changed: Array<string | BaseModel>;
};

export type ModelOnchangePreparedPreviewState = {
  meta: ModelMetadata;
  selParsed: ParsedChangedSelectors;
  changedFields: string[];
  mergedDraft: OnchangeDraft;
  previewProxy: ObjectRecord;
};

export type ModelOnchangePreparedDiagnostics = {
  missingCount: number;
  usedCache: boolean;
  pathDepthMax: number;
  cachedSignature: string;
  execStats?: PrefetchExecStats;
  readsRoot: Set<string>;
};

export type ModelOnchangePreparation = {
  preview: ModelOnchangePreparedPreviewState;
  diagnostics: ModelOnchangePreparedDiagnostics;
};

function collectModelOnchangeComputeSubset(meta: ModelMetadata, changedFields: string[], activeHandlers: OnchangeHandlerMeta[]): Set<string> {
  const computeSubset = new Set<string>();
  if (!meta.computeGraph) return computeSubset;

  const seed = new Set<string>([...changedFields]);
  activeHandlers.forEach(h => h.triggers.forEach((t: string) => seed.add(t)));
  const queue: string[] = [...seed];
  const seen = new Set<string>();
  while (queue.length) {
    const src = queue.shift()!;
    const affected = meta.computeGraph.fastReverseDeps.get(src);
    if (!affected) continue;
    for (const cf of affected) {
      if (!computeSubset.has(cf)) {
        computeSubset.add(cf);
        if (!seen.has(cf)) {
          seen.add(cf);
          const next = meta.computeGraph.fastReverseDeps.get(cf);
          if (next) queue.push(...next);
        }
      }
    }
  }
  return computeSubset;
}

export function __collectModelOnchangeComputeSubsetForTest(meta: ModelMetadata, changedFields: string[], activeHandlers: OnchangeHandlerMeta[]): Set<string> {
  return collectModelOnchangeComputeSubset(meta, changedFields, activeHandlers);
}

function collectModelOnchangeReadsRoot(meta: ModelMetadata, activeHandlers: OnchangeHandlerMeta[]): Set<string> {
  const readsRoot = new Set<string>();
  try {
    const readsParsed = parseOnchangeReadsEx(meta, activeHandlers);
    for (const k of readsParsed.m2o.keys()) readsRoot.add(k);
    for (const k of readsParsed.collections.keys()) readsRoot.add(k);
  } catch {
    // ignore
  }

  for (const h of activeHandlers) {
    for (const r of h.reads || []) {
      const root = r.split('.').filter(Boolean)[0];
      if (root) readsRoot.add(root);
    }
  }

  return readsRoot;
}

export function __collectModelOnchangeReadsRootForTest(meta: ModelMetadata, activeHandlers: OnchangeHandlerMeta[]): Set<string> {
  return collectModelOnchangeReadsRoot(meta, activeHandlers);
}

export function __normalizeModelOnchangeChangedFieldsRawForTest(changed: Array<string | BaseModel> | undefined): string[] {
  return (changed || []).filter(Boolean).map(String);
}

export function __collectModelOnchangeParentAllFieldNamesForTest(meta: ModelMetadata): Set<string> {
  return new Set<string>(Array.from(meta.fields?.keys?.() || []));
}

export async function prepareModelOnchangePreview(params: ModelOnchangePrepareParams): Promise<ModelOnchangePreparation> {
  const { ModelCtor, draft, changed } = params;

  const meta = getModelRuntimeMetadata(ModelCtor);
  const changedFieldsRaw = __normalizeModelOnchangeChangedFieldsRawForTest(changed);
  const selParsed = parseChangedSelectors(changedFieldsRaw);
  const changedFields = Array.from(selParsed.normalizedSeeds);

  const { needed, activeHandlers } = buildNeededFields(meta, draft, changedFields);

  const missing = [...needed].filter(f => !(f in draft));
  const missingCount = missing.length;
  let mergedDraft: OnchangeDraft = { ...draft };
  const isEdit = !!draft.Id;
  if (isEdit && missing.length) {
    try {
      const repo = getModelRepository(ModelCtor);
      const rows = await repo.search(['Id', '=', draft.Id], {
        fields: ['Id', ...missing],
      });
      if (rows.length) {
        mergedDraft = { ...rows[0], ...mergedDraft };
      }
    } catch {
      // ignore
    }
  }

  const computeSubset = collectModelOnchangeComputeSubset(meta, changedFields, activeHandlers);

  let usedCache = false;
  let pathDepthMax = 1;
  let cachedSignature = '';
  let execStats: PrefetchExecStats | undefined;
  const reads = parseOnchangeReadsEx(meta, activeHandlers);
  const computeM2oPaths = extractComputePathDeps(meta, computeSubset);
  const computeCollectionPaths = extractComputeCollectionPathDeps(meta, computeSubset);
  const cached = PathPlanBuilder.getCachedOrBuildV2(ModelCtor, reads.m2o, reads.collections, computeM2oPaths, computeCollectionPaths);
  usedCache = cached.fromCache;
  pathDepthMax = cached.pathDepthMax;
  cachedSignature = cached.signature;
  execStats = await PathPlanBuilder.executeWithPlan(ModelCtor, meta, mergedDraft, cached.plan);

  const previewInstance = Object.assign(Object.create(ModelCtor.prototype) as BaseModel, mergedDraft);
  Object.defineProperty(previewInstance, '__preview', {
    value: true,
    enumerable: false,
    configurable: false,
  });

  const readsRoot = collectModelOnchangeReadsRoot(meta, activeHandlers);
  const parentAllFieldNames = __collectModelOnchangeParentAllFieldNamesForTest(meta);
  const parentReads = new Set<string>([...readsRoot, ...parentAllFieldNames]);

  const previewProxy = createPreviewProxy(previewInstance, {
    meta,
    triggers: new Set(changedFields),
    reads: parentReads,
    loaded: new Set(Object.keys(mergedDraft)),
  });

  return {
    preview: {
      meta,
      selParsed,
      changedFields,
      mergedDraft,
      previewProxy: previewProxy as unknown as ObjectRecord,
    },
    diagnostics: {
      missingCount,
      usedCache,
      pathDepthMax,
      cachedSignature,
      execStats,
      readsRoot,
    },
  };
}
