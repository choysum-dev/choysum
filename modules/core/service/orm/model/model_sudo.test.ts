// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getRepositoryFieldRuleBypassDepth, getRepositoryRecordRuleBypassDepth } from '../repository/authz';
import { withModelSudo } from './model_sudo';

async function withPatchedChoysum<T>(value: unknown, fn: () => Promise<T> | T): Promise<T> {
  const key = '$choysum';
  const hadOwn = Object.prototype.hasOwnProperty.call(globalThis as object, key);
  const previous = (globalThis as Record<string, unknown>)[key];
  (globalThis as Record<string, unknown>)[key] = value as unknown;
  try {
    return await fn();
  } finally {
    if (hadOwn) (globalThis as Record<string, unknown>)[key] = previous;
    else delete (globalThis as Record<string, unknown>)[key];
  }
}

test('withModelSudo elevates RR+FR sync and records sudo audit enter', async () => {
  delete (globalThis as any).__choysumComputeAudit;

  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {},
        },
      },
    },
    () => {
      expect(getRepositoryRecordRuleBypassDepth()).toBe(0);
      expect(getRepositoryFieldRuleBypassDepth()).toBe(0);

      const value = withModelSudo(() => {
        expect(getRepositoryRecordRuleBypassDepth()).toBe(1);
        expect(getRepositoryFieldRuleBypassDepth()).toBe(1);
        return 'elevated';
      });

      expect(value).toBe('elevated');
      expect(getRepositoryRecordRuleBypassDepth()).toBe(0);
      expect(getRepositoryFieldRuleBypassDepth()).toBe(0);

      const hits = ((globalThis as any).__choysumComputeAudit?.sudoHits || []) as any[];
      expect(hits.length).toBe(1);
      expect(hits[0]?.source).toBe('sudo');
      expect(hits[0]?.version).toBe(1);
      expect(typeof hits[0]?.at).toBe('string');
    }
  );
});

test('withModelSudo nests and supports async fn', async () => {
  delete (globalThis as any).__choysumComputeAudit;

  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {},
        },
      },
    },
    async () => {
      await withModelSudo(async () => {
        expect(getRepositoryRecordRuleBypassDepth()).toBe(1);
        expect(getRepositoryFieldRuleBypassDepth()).toBe(1);
        await withModelSudo(async () => {
          expect(getRepositoryRecordRuleBypassDepth()).toBe(2);
          expect(getRepositoryFieldRuleBypassDepth()).toBe(2);
          return undefined;
        });
        expect(getRepositoryRecordRuleBypassDepth()).toBe(1);
        expect(getRepositoryFieldRuleBypassDepth()).toBe(1);
        return undefined;
      });
      expect(getRepositoryRecordRuleBypassDepth()).toBe(0);
      expect(getRepositoryFieldRuleBypassDepth()).toBe(0);

      const hits = ((globalThis as any).__choysumComputeAudit?.sudoHits || []) as any[];
      expect(hits.length).toBe(2);
    }
  );
});
