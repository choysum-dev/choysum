// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { DEFAULT_LOOP_THRESHOLD } from '../../runtime/onchange/constants';
import type { OnchangeDraft, OnchangeEngineResult, OnchangeResult } from '../../runtime/onchange/types';
import { runModelOnchangePreviewEngine } from './model_runtime_service_facade';
import { applyModelOnchangePreviewCascade } from './model_onchange_preview_cascade';
import { attachModelOnchangeDiagnostics, finalizeModelOnchangeTransport } from './model_onchange_postprocess';
import type { ModelOnchangePreparation } from './model_onchange_prepare';
import { applyModelOnchangePreviewValidation } from './model_onchange_validation';
import type { ModelCtor } from '../metadata/field';
import type BaseModel from './model';

export type ModelOnchangeExecutionOptions = {
  withCompute?: boolean;
  maxIterations?: number;
  loopThreshold?: number;
};

type ModelOnchangeExecutionDeps = {
  runPreviewEngine: typeof runModelOnchangePreviewEngine;
  applyPreviewCascade: typeof applyModelOnchangePreviewCascade;
  attachDiagnostics: typeof attachModelOnchangeDiagnostics;
  validatePreview: typeof applyModelOnchangePreviewValidation;
  finalizeTransport: typeof finalizeModelOnchangeTransport;
};

const defaultModelOnchangeExecutionDeps: ModelOnchangeExecutionDeps = {
  runPreviewEngine: runModelOnchangePreviewEngine,
  applyPreviewCascade: applyModelOnchangePreviewCascade,
  attachDiagnostics: attachModelOnchangeDiagnostics,
  validatePreview: applyModelOnchangePreviewValidation,
  finalizeTransport: finalizeModelOnchangeTransport,
};

export async function executePreparedModelOnchangePreview(
  params: {
    ModelCtor: ModelCtor<BaseModel> & typeof BaseModel;
    draft: OnchangeDraft;
    prepared: ModelOnchangePreparation;
    prefetchTimeMs: number;
    opts?: ModelOnchangeExecutionOptions;
  },
  deps: ModelOnchangeExecutionDeps = defaultModelOnchangeExecutionDeps
): Promise<OnchangeResult> {
  const { ModelCtor, draft, prepared, prefetchTimeMs, opts } = params;
  const { preview, diagnostics } = prepared;
  const { meta, selParsed, changedFields, mergedDraft, previewProxy } = preview;
  const { missingCount, usedCache, pathDepthMax, cachedSignature, execStats, readsRoot } = diagnostics;

  const res = (await deps.runPreviewEngine(meta, previewProxy, changedFields, opts)) as unknown as OnchangeEngineResult;

  await deps.applyPreviewCascade({
    meta,
    previewProxy,
    changedFields,
    selParsed,
    opts,
    res,
  });

  try {
    deps.attachDiagnostics({
      res,
      missingCount,
      prefetchTimeMs,
      pathDepthMax,
      readsRoot,
      changedFields,
      usedCache,
      cachedSignature,
      execStats,
      loopThreshold: opts?.loopThreshold ?? DEFAULT_LOOP_THRESHOLD,
    });
  } catch {
    // ignore
  }

  await deps.validatePreview({
    ModelCtor,
    draft,
    meta,
    previewProxy,
    mergedDraft,
    changedFields,
    res,
  });

  return deps.finalizeTransport(res);
}
