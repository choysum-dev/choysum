// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ENABLE_PREVIEW_KERNEL_VALIDATION, PREVIEW_KERNEL_RULES } from '../../runtime/onchange/constants';
import type { OnchangeDraft, OnchangeResult } from '../../runtime/onchange/types';
import { validateRuntimeIssues } from '../../runtime/runtime_validation_facade';
import type { ModelCtor } from '../metadata/field';
import type { ModelMetadata } from '../metadata/model';
import type { KernelValidationRule } from '../repository/validation';
import type BaseModel from './model';
import { getModelRepository } from './model_internal_facade';
import { appendModelOnchangeValidationIssues } from './model_onchange_postprocess';
import type { ObjectRecord } from '../../../utils/types';

export function buildPreviewValidationOptions(includeKernel: boolean): {
  includeKernel: boolean;
  kernelRules?: KernelValidationRule[];
  includePlatform: true;
  includeConstraints: true;
} {
  return {
    includeKernel,
    kernelRules: includeKernel ? ([...PREVIEW_KERNEL_RULES] as KernelValidationRule[]) : undefined,
    includePlatform: true,
    includeConstraints: true,
  };
}

export async function applyModelOnchangePreviewValidation(params: {
  ModelCtor: ModelCtor<BaseModel> & typeof BaseModel;
  draft: OnchangeDraft;
  meta: ModelMetadata;
  previewProxy: ObjectRecord;
  mergedDraft: OnchangeDraft;
  changedFields: string[];
  res: OnchangeResult;
}): Promise<void> {
  const { ModelCtor, draft, meta, previewProxy, mergedDraft, changedFields, res } = params;

  try {
    const repository = getModelRepository(ModelCtor);
    const issues = await validateRuntimeIssues(
      {
        mode: 'preview',
        model: ModelCtor,
        metadata: meta,
        self: previewProxy as unknown as BaseModel,
        current: draft as Partial<BaseModel> & ObjectRecord,
        values: mergedDraft as Partial<BaseModel> & ObjectRecord,
        changedFields: new Set(changedFields),
        repository,
        requestContext: (ModelCtor as typeof BaseModel).ctx,
      },
      buildPreviewValidationOptions(ENABLE_PREVIEW_KERNEL_VALIDATION)
    );

    appendModelOnchangeValidationIssues(res, issues);
  } catch {
    // ignore
  }
}
