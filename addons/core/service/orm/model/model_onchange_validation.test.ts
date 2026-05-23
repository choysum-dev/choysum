// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from './model';
import { applyModelOnchangePreviewValidation, buildPreviewValidationOptions } from './model_onchange_validation';
import { Field, Model } from '../decorator';
import { MetadataStorage } from '../metadata/storage';
import { RepositoryFactory } from '../repository/repository_factory';
import { ENABLE_PREVIEW_KERNEL_VALIDATION, PREVIEW_KERNEL_RULES } from '../../runtime/onchange/constants';
import { ValidationEngine } from '../../runtime/validation';

@Model('test.ModelOnchangeValidationModel')
class ModelOnchangeValidationModel extends BaseModel {
  @Field({ type: 'varchar', column: { size: 64 } })
  Name?: string;

  static get ctx() {
    return { userId: 'U1', lang: 'zh-CN' } as any;
  }
}

test('model onchange validation delegates preview validation context and appends issues', async () => {
  const originalValidate = ValidationEngine.validate;
  const calls: any[] = [];
  const meta = MetadataStorage.instance.getModelMetadata(ModelOnchangeValidationModel as any);
  const repository = { tag: 'repo' } as any;
  const draft = { Id: 'ROW-1', Name: 'draft' };
  const mergedDraft = { Id: 'ROW-1', Name: 'next' };
  const previewProxy = { Name: 'next' };
  const res: any = { messages: [{ level: 'info', message: 'keep' }] };

  RepositoryFactory.setRepository(ModelOnchangeValidationModel as any, repository);

  try {
    ValidationEngine.validate = (async (nextCtx: any, nextOptions: any) => {
      calls.push({ nextCtx, nextOptions });
      return [
        {
          scope: 'constraint',
          code: 'preview_issue',
          message: 'blocked',
          severity: 'error',
          field: 'Name',
          method: 'checkName',
        },
      ];
    }) as any;

    await applyModelOnchangePreviewValidation({
      ModelCtor: ModelOnchangeValidationModel as any,
      draft,
      meta: meta as any,
      previewProxy,
      mergedDraft,
      changedFields: ['Name'],
      res,
    });

    expect(calls.length).toBe(1);
    expect(calls[0]?.nextCtx).toEqual({
      mode: 'preview',
      model: ModelOnchangeValidationModel,
      metadata: meta,
      self: previewProxy,
      current: draft,
      values: mergedDraft,
      changedFields: new Set(['Name']),
      repository,
      requestContext: (ModelOnchangeValidationModel as any).ctx,
    });
    expect(calls[0]?.nextOptions).toEqual({
      includeKernel: ENABLE_PREVIEW_KERNEL_VALIDATION,
      kernelRules: ENABLE_PREVIEW_KERNEL_VALIDATION ? [...PREVIEW_KERNEL_RULES] : undefined,
      includePlatform: true,
      includeConstraints: true,
    });
    expect(res.messages).toEqual([
      { level: 'info', message: 'keep' },
      { level: 'error', message: 'blocked', field: 'Name', blocking: true, title: 'checkName' },
    ]);
  } finally {
    ValidationEngine.validate = originalValidate;
  }
});

test('model onchange validation swallows preview validation errors', async () => {
  const originalValidate = ValidationEngine.validate;
  const meta = MetadataStorage.instance.getModelMetadata(ModelOnchangeValidationModel as any);
  const res: any = { messages: [{ level: 'info', message: 'keep' }] };

  try {
    ValidationEngine.validate = (async () => {
      throw new Error('boom');
    }) as any;

    await applyModelOnchangePreviewValidation({
      ModelCtor: ModelOnchangeValidationModel as any,
      draft: { Id: 'ROW-1', Name: 'draft' },
      meta: meta as any,
      previewProxy: { Name: 'next' },
      mergedDraft: { Id: 'ROW-1', Name: 'next' },
      changedFields: ['Name'],
      res,
    });

    expect(res.messages).toEqual([{ level: 'info', message: 'keep' }]);
  } finally {
    ValidationEngine.validate = originalValidate;
  }
});

test('model onchange validation options include or omit kernel rules by flag', () => {
  const enabled = buildPreviewValidationOptions(true);
  expect(enabled).toEqual({
    includeKernel: true,
    kernelRules: [...PREVIEW_KERNEL_RULES],
    includePlatform: true,
    includeConstraints: true,
  });

  const disabled = buildPreviewValidationOptions(false);
  expect(disabled).toEqual({
    includeKernel: false,
    kernelRules: undefined,
    includePlatform: true,
    includeConstraints: true,
  });
});
