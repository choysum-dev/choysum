// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { appendModelOnchangeValidationIssues, attachModelOnchangeDiagnostics, finalizeModelOnchangeTransport } from './model_onchange_postprocess';

test('model onchange postprocess appends validation issues as onchange messages', () => {
  const res: any = {
    messages: [{ level: 'info', message: 'keep' }],
  };

  appendModelOnchangeValidationIssues(res, [
    {
      scope: 'platform',
      code: 'platform_issue',
      message: 'bad input',
      severity: 'error',
      field: 'Name',
      method: 'checkName',
    },
    {
      scope: 'constraint',
      code: 'warn_issue',
      message: 'watch out',
      severity: 'warning',
    },
  ] as any);

  expect(res.messages).toEqual([
    { level: 'info', message: 'keep' },
    { level: 'error', message: 'bad input', field: 'Name', blocking: true, title: 'checkName' },
    { level: 'warn', message: 'watch out', field: undefined, blocking: false, title: undefined },
  ]);
});

test('model onchange postprocess appends validation issues when messages is missing', () => {
  const res: any = {};

  appendModelOnchangeValidationIssues(res, [{ message: 'new warning', severity: 'warning' }] as any);

  expect(res.messages).toEqual([
    {
      level: 'warn',
      message: 'new warning',
      field: undefined,
      blocking: false,
      title: undefined,
    },
  ]);
});

test('model onchange postprocess attaches diagnostics with exec stats fallback defaults', () => {
  const res: any = {
    messages: [],
    computeRecomputed: ['Name'],
    iterations: 2,
  };

  attachModelOnchangeDiagnostics({
    res,
    missingCount: 3,
    prefetchTimeMs: 12,
    pathDepthMax: 2,
    readsRoot: new Set(['PartnerId']),
    changedFields: ['Name'],
    usedCache: true,
    cachedSignature: 'sig-1',
    execStats: {} as any,
  });

  expect(res.diagnostics).toMatchObject({
    missingCount: 3,
    prefetchTimeMs: 12,
    pathDepthMax: 2,
    cachedPlanUsed: true,
    iterations: 2,
  });
});

test('model onchange postprocess clears value and hides collection patch when final result has error', () => {
  const res: any = {
    value: { __collectionPatch: { Lines: [{ Id: '1' }] }, Name: 'demo' },
    messages: [{ level: 'error', message: 'blocked' }],
    condition: [{ field: 'Name', condition: ['Name', '=', 'x'] }],
  };

  const transport = finalizeModelOnchangeTransport(res);

  expect(transport.value).toBe(undefined);
  expect(transport.messages).toEqual([{ level: 'error', message: 'blocked' }]);
  expect(transport.condition).toEqual([{ field: 'Name', condition: ['Name', '=', 'x'] }]);
});

test('model onchange postprocess executes guarded collection patch cleanup branch', () => {
  let internalValue: any = { __collectionPatch: { Lines: [{ Id: '1' }] }, Name: 'demo' };
  const res: any = {
    messages: [{ level: 'error', message: 'blocked' }],
  };

  Object.defineProperty(res, 'value', {
    configurable: true,
    enumerable: true,
    get() {
      return internalValue;
    },
    set(next: any) {
      // Keep a synthetic collection patch after overwrite so cleanup guard is exercised.
      internalValue = { ...(next || {}), __collectionPatch: { Lines: [{ Id: '1' }] } };
    },
  });

  const transport = finalizeModelOnchangeTransport(res);

  expect('__collectionPatch' in internalValue).toBe(false);
  expect(internalValue).toEqual({});
  expect(transport.value).toBeUndefined();
});

test('model onchange postprocess keeps only non-empty transport fields', () => {
  const transport = finalizeModelOnchangeTransport({
    value: { Name: 'demo' },
    messages: [],
    selection: [{ field: 'State', selection: ['draft', 'done'] }],
  } as any);

  expect(transport).toEqual({
    value: { Name: 'demo' },
    selection: [{ field: 'State', selection: ['draft', 'done'] }],
  });
});
