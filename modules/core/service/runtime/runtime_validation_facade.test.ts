// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ValidationEngine } from './validation';
import { validateRuntimeIssues, validateRuntimeOrThrow } from './runtime_validation_facade';

test('runtime validation facade delegates validate unchanged', async () => {
  const originalValidate = ValidationEngine.validate;
  const calls: any[] = [];
  const ctx = { mode: 'preview' } as any;
  const options = { includeKernel: false, includePlatform: true } as any;

  try {
    ValidationEngine.validate = (async (nextCtx: any, nextOptions: any) => {
      calls.push({ nextCtx, nextOptions });
      return [{ code: 'demo_issue' }];
    }) as any;

    const result = await validateRuntimeIssues(ctx, options);

    expect(result).toEqual([{ code: 'demo_issue' }]);
    expect(calls.length).toBe(1);
    expect(calls[0]?.nextCtx).toBe(ctx);
    expect(calls[0]?.nextOptions).toBe(options);
  } finally {
    ValidationEngine.validate = originalValidate;
  }
});

test('runtime validation facade delegates validateOrThrow unchanged', async () => {
  const originalValidateOrThrow = ValidationEngine.validateOrThrow;
  const calls: any[] = [];
  const ctx = { mode: 'create' } as any;
  const options = { includeConstraints: false } as any;

  try {
    ValidationEngine.validateOrThrow = (async (nextCtx: any, nextOptions: any) => {
      calls.push({ nextCtx, nextOptions });
    }) as any;

    await validateRuntimeOrThrow(ctx, options);

    expect(calls.length).toBe(1);
    expect(calls[0]?.nextCtx).toBe(ctx);
    expect(calls[0]?.nextOptions).toBe(options);
  } finally {
    ValidationEngine.validateOrThrow = originalValidateOrThrow;
  }
});
