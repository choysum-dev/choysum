// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type BaseModel from './model';
import type { ModelMetadata } from '../metadata';
import type { OnchangeTrigger } from '../metadata/field';
import type { Insertable } from '../repository/types';
import type { OnchangeDraft, OnchangeResult } from '../../runtime/onchange/types';
import type { ComputeMode } from '../../runtime/compute/engine';
import type { UpstreamChangeEvent } from '../../runtime/compute/cascade';
import { OnchangeEngine } from '../../runtime/onchange/engine';
import { MAX_ITERATIONS, DEFAULT_LOOP_THRESHOLD } from '../../runtime/onchange/constants';
import { DefaultOperations } from './model_default';
import { OnchangeOperations } from './model_onchange';
import { ComputeEngine } from '../../runtime/compute/engine';
import { ComputeCascadeEngine } from '../../runtime/compute/cascade';
import { getCachedModelMetadata } from './model_runtime';
import type { RuntimeModelCtor } from './types';
import { asObjectRecord } from '../../../utils/object';
import type { UnknownRecord } from '../../../utils/types';

type ModelRuntimeServiceFacadeCtor<T extends BaseModel> = RuntimeModelCtor<T>;

type ModelOnchangeOptions = {
  withCompute?: boolean;
  maxIterations?: number;
  loopThreshold?: number;
};

type RuntimeEntityInput = UnknownRecord | BaseModel;

/**
 * Apply `@Field({ default })` column defaults via {@link DefaultOperations.DefaultGet}
 * (alias of {@link applyFieldColumnDefaults}). Used by `BaseModel.DefaultGet`.
 *
 * Create/CreateMany must call `ModelCtor.DefaultGet` (polymorphic hook), not this helper.
 * Do not call `ModelCtor.DefaultGet` from here — that would recurse through the base hook.
 */
export async function defaultModelValues<T extends BaseModel>(
  ModelCtor: ModelRuntimeServiceFacadeCtor<T>,
  value: Partial<Insertable<T & BaseModel>>
): Promise<Partial<Insertable<T & BaseModel>>> {
  return await DefaultOperations.DefaultGet(ModelCtor, value);
}

export async function runModelOnchange<T extends BaseModel>(
  ModelCtor: ModelRuntimeServiceFacadeCtor<T>,
  draft: OnchangeDraft,
  changed: OnchangeTrigger<T>[],
  opts?: ModelOnchangeOptions
): Promise<OnchangeResult> {
  return await OnchangeOperations.Onchange<T>(ModelCtor, draft, changed, opts);
}

export function getModelRuntimeMetadata<T extends BaseModel>(ModelCtor: ModelRuntimeServiceFacadeCtor<T>): ModelMetadata {
  return getCachedModelMetadata(ModelCtor);
}

export async function runModelOnchangePreviewEngine(
  meta: ModelMetadata,
  entity: RuntimeEntityInput,
  changedFields: string[],
  opts?: ModelOnchangeOptions
): Promise<OnchangeResult> {
  const entityRecord = asObjectRecord(entity);
  if (!entityRecord) {
    throw new Error('Invalid preview entity: expected object-like input');
  }

  return await OnchangeEngine.run(meta, entityRecord, changedFields, {
    withCompute: opts?.withCompute !== false,
    maxIterations: opts?.maxIterations ?? MAX_ITERATIONS,
    loopThreshold: opts?.loopThreshold ?? DEFAULT_LOOP_THRESHOLD,
    computePreview: async (nextEntity, seed) => {
      await recomputeModelMetadata(meta, nextEntity, seed, 'preview');
    },
  });
}

export async function recomputeModelMetadata(meta: ModelMetadata, entity: RuntimeEntityInput, baseChanged: Set<string>, mode: ComputeMode): Promise<void> {
  if (!meta.computeGraph) return;

  const entityRecord = asObjectRecord(entity);
  if (!entityRecord) return;

  await ComputeEngine.recompute(meta, entityRecord, baseChanged, mode);
}

export function collectModelUpstreamInverseFields<T extends BaseModel>(ModelCtor: ModelRuntimeServiceFacadeCtor<T>): string[] {
  return ComputeCascadeEngine.collectUpstreamInverseFields(ModelCtor);
}

export async function triggerModelUpstream(event: UpstreamChangeEvent): Promise<void> {
  await ComputeCascadeEngine.triggerUpstream(event);
}

export async function triggerModelUpstreamCreateBatch<T extends BaseModel>(ModelCtor: ModelRuntimeServiceFacadeCtor<T>, rows: UnknownRecord[]): Promise<void> {
  await ComputeCascadeEngine.triggerUpstreamCreateBatch(ModelCtor, rows);
}

export async function triggerModelDownstream<T extends BaseModel>(
  ModelCtor: ModelRuntimeServiceFacadeCtor<T>,
  changedFields: string[],
  recordId: string
): Promise<void> {
  await ComputeCascadeEngine.triggerDownstream(ModelCtor, changedFields, recordId);
}
