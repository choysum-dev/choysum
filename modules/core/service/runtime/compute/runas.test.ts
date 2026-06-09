// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getReadonlyCtx } from '../context';
import { recordComputeRunAsAudit, withComputeRunAsExecution } from './runas';

function makeMeta() {
  return {
    fullModelName: 'test.RunAsModel',
    modelName: 'RunAsModel',
    className: 'RunAsModel',
  } as any;
}

test('compute runAs helper keeps user path as no-op and does not record audit', () => {
  delete (globalThis as any).__choysumComputeAudit;

  const value = withComputeRunAsExecution(makeMeta(), 'SecureValue', 'user', 'expr', () => {
    const ctx = getReadonlyCtx() as any;
    return String(ctx?.__computeRunAs || 'none');
  });

  expect(value).toBe('none');
  const hits = ((globalThis as any).__choysumComputeAudit?.runAsHits || []) as any[];
  expect(hits.length).toBe(0);
});

test('compute runAs helper initializes and repairs global audit bucket while reusing existing bucket', () => {
  (globalThis as any).__choysumComputeAudit = 123;
  recordComputeRunAsAudit(makeMeta(), 'SecureValue', 'sudo', 'search');

  let bucket = (globalThis as any).__choysumComputeAudit;
  expect(bucket && typeof bucket === 'object').toBe(true);
  expect(Array.isArray(bucket.runAsHits)).toBe(true);
  expect(bucket.runAsHits.length).toBe(1);

  const reused = { runAsHits: [{ version: 1, model: 'x', field: 'y', runAs: 'sudo', phase: 'expr', at: 't' }], token: 'keep' };
  (globalThis as any).__choysumComputeAudit = reused;
  recordComputeRunAsAudit(makeMeta(), 'SecureValue', 'sudo', 'expr', 'persist');
  bucket = (globalThis as any).__choysumComputeAudit;

  expect(bucket).toBe(reused);
  expect(bucket.token).toBe('keep');
  expect(bucket.runAsHits.length).toBe(2);

  (globalThis as any).__choysumComputeAudit = { runAsHits: 'broken' };
  recordComputeRunAsAudit(makeMeta(), 'SecureValue', 'sudo', 'expr');
  bucket = (globalThis as any).__choysumComputeAudit;
  expect(Array.isArray(bucket.runAsHits)).toBe(true);
  expect(bucket.runAsHits.length).toBe(1);
});

test('compute runAs helper propagates thrown errors and applies sudo context marker in callback', () => {
  delete (globalThis as any).__choysumComputeAudit;

  let message = '';
  try {
    withComputeRunAsExecution(makeMeta(), 'SecureValue', 'sudo', 'expr', () => {
      const ctx = getReadonlyCtx() as any;
      throw new Error(`boom:${String(ctx?.__computeRunAs || 'none')}`);
    });
  } catch (error) {
    message = String((error as Error)?.message || error);
  }

  expect(message).toBe('boom:sudo');
  const hits = ((globalThis as any).__choysumComputeAudit?.runAsHits || []) as any[];
  expect(hits.length).toBe(1);
  expect(hits[0]?.runAs).toBe('sudo');
  expect(hits[0]?.phase).toBe('expr');
});

test('compute runAs helper normalizes unknown model/field and blank mode in audit entry', () => {
  delete (globalThis as any).__choysumComputeAudit;

  recordComputeRunAsAudit(
    {
      fullModelName: '   ',
      modelName: '',
      className: '',
    } as any,
    '   ',
    'sudo',
    'search',
    '   '
  );

  const hits = ((globalThis as any).__choysumComputeAudit?.runAsHits || []) as any[];
  expect(hits.length).toBe(1);
  expect(hits[0]?.model).toBe('Unknown');
  expect(hits[0]?.field).toBe('unknown');
  expect(hits[0]?.mode).toBe('');
  expect(hits[0]?.phase).toBe('search');
});
