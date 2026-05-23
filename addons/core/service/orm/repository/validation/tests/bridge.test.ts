// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ValidationPipelineError } from '../../../metadata';
import { validateRepositoryWrite } from '..';
import { ValidationEngine } from '../../../../runtime/validation';
import BaseModel from '../../../model/model';

class ValidationBridgeModel extends BaseModel {}

test('validation bridge composes create whitelist from context and internal compute fields', async () => {
  const original = ValidationEngine.validateOrThrow;

  const captured: any = {
    ctx: undefined,
    options: undefined,
  };

  try {
    ValidationEngine.validateOrThrow = (async (ctx: any, options: any) => {
      captured.ctx = ctx;
      captured.options = options;
      options?.onPlatformCreateWhitelistHit?.([' Name ', 'Name']);
    }) as any;

    const requestContext: any = {
      requestId: 'req_bridge_1',
      validation: {
        platformCreateWriteWhitelistByModel: {
          'test.ValidationBridgeModel': [' Name ', 'Code', 'Name'],
        },
      },
      platformRejectUnknownFields: true,
    };

    await validateRepositoryWrite({
      meta: {
        fullModelName: 'test.ValidationBridgeModel',
        modelName: 'ValidationBridgeModel',
        name: 'ValidationBridgeModel',
        type: ValidationBridgeModel,
        fields: new Map(),
        computeGraph: {
          computeFields: new Set(['ComputedA', 'ComputedA']),
        },
      } as any,
      repository: {} as any,
      requestContext,
      getValidationBypassDepth: () => 1,
      input: {
        Name: 'n1',
      } as any,
      mode: 'create',
      current: {
        Id: 'row_1',
      },
    });

    expect(captured.ctx.mode).toBe('create');
    expect(captured.ctx.changedFields instanceof Set).toBe(true);
    expect(Array.from(captured.ctx.changedFields).sort()).toEqual(['Name']);

    expect(captured.options.platformRejectUnknownFields).toBe(true);
    expect(captured.options.platformCreateWriteWhitelist).toEqual(['Name', 'Code', 'ComputedA']);
    expect(requestContext.__validationAudit?.platformCreateWhitelistHits?.length).toBe(1);
  } finally {
    ValidationEngine.validateOrThrow = original;
  }
});

test('validation bridge keeps whitelist empty for non-create mode without bypass', async () => {
  const original = ValidationEngine.validateOrThrow;
  let capturedOptions: any;

  try {
    ValidationEngine.validateOrThrow = (async (_ctx: any, options: any) => {
      capturedOptions = options;
    }) as any;

    await validateRepositoryWrite({
      meta: {
        fullModelName: 'test.ValidationBridgeModel',
        modelName: 'ValidationBridgeModel',
        name: 'ValidationBridgeModel',
        type: ValidationBridgeModel,
        fields: new Map(),
        computeGraph: {
          computeFields: new Set(['ComputedA']),
        },
      } as any,
      repository: {} as any,
      requestContext: {
        validation: {
          platformCreateWriteWhitelist: ['Code'],
        },
      },
      getValidationBypassDepth: () => 0,
      input: {
        Name: 'n1',
      } as any,
      mode: 'update',
    });

    expect(capturedOptions.platformCreateWriteWhitelist).toEqual([]);
  } finally {
    ValidationEngine.validateOrThrow = original;
  }
});

test('validation bridge wraps ValidationPipelineError and rethrows non-validation errors', async () => {
  const original = ValidationEngine.validateOrThrow;

  try {
    ValidationEngine.validateOrThrow = (async () => {
      throw new ValidationPipelineError('validation failed', []);
    }) as any;

    let wrapped: any;
    try {
      await validateRepositoryWrite({
        meta: {
          fullModelName: 'test.ValidationBridgeModel',
          modelName: 'ValidationBridgeModel',
          name: 'ValidationBridgeModel',
          type: ValidationBridgeModel,
          fields: new Map(),
          computeGraph: {
            computeFields: new Set(),
          },
        } as any,
        repository: {} as any,
        requestContext: {},
        getValidationBypassDepth: () => 0,
        input: {} as any,
        mode: 'create',
      });
    } catch (error) {
      wrapped = error;
    }

    expect(Boolean(wrapped)).toBe(true);
    expect(wrapped?.code).toBe('validation_failed');

    ValidationEngine.validateOrThrow = (async () => {
      throw new Error('boom_non_validation');
    }) as any;

    let raw: any;
    try {
      await validateRepositoryWrite({
        meta: {
          fullModelName: 'test.ValidationBridgeModel',
          modelName: 'ValidationBridgeModel',
          name: 'ValidationBridgeModel',
          type: ValidationBridgeModel,
          fields: new Map(),
          computeGraph: {
            computeFields: new Set(),
          },
        } as any,
        repository: {} as any,
        requestContext: {},
        getValidationBypassDepth: () => 0,
        input: {} as any,
        mode: 'update',
      });
    } catch (error) {
      raw = error;
    }

    expect(String(raw?.message || raw)).toContain('boom_non_validation');
  } finally {
    ValidationEngine.validateOrThrow = original;
  }
});

test('validation bridge builds empty changedFields when input is undefined', async () => {
  const original = ValidationEngine.validateOrThrow;
  let capturedCtx: any;

  try {
    ValidationEngine.validateOrThrow = (async (ctx: any) => {
      capturedCtx = ctx;
    }) as any;

    await validateRepositoryWrite({
      meta: {
        fullModelName: 'test.ValidationBridgeModel',
        modelName: 'ValidationBridgeModel',
        name: 'ValidationBridgeModel',
        type: ValidationBridgeModel,
        fields: new Map(),
        computeGraph: {
          computeFields: new Set(),
        },
      } as any,
      repository: {} as any,
      requestContext: {},
      getValidationBypassDepth: () => 0,
      input: undefined as any,
      mode: 'update',
    });

    expect(Array.from(capturedCtx.changedFields || [])).toEqual([]);
  } finally {
    ValidationEngine.validateOrThrow = original;
  }
});
