// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getCurrentReq, getOrInitReqServiceState } from '../../runtime/context';
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

function readSudoHits(): any[] {
  const state = getOrInitReqServiceState(getCurrentReq()) as { sudoHits?: any[] } | undefined;
  if (Array.isArray(state?.sudoHits)) return state.sudoHits;
  return ((globalThis as any).__choysumComputeAudit?.sudoHits || []) as any[];
}

test('withModelSudo elevates RR+FR sync and records sudo audit enter', async () => {
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

      const hits = readSudoHits();
      expect(hits.length).toBe(1);
      expect(hits[0]?.source).toBe('sudo');
      expect(hits[0]?.version).toBe(1);
      expect(typeof hits[0]?.at).toBe('string');
    }
  );
});

test('withModelSudo nests and supports async fn', async () => {
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

      const hits = readSudoHits();
      expect(hits.length).toBe(2);
    }
  );
});

test('withModelSudo audit hits stay request-scoped and do not leak across requests', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: { id: 'req-a' },
        },
      },
    },
    () => {
      withModelSudo(() => 'a');
      expect(readSudoHits().length).toBe(1);
    }
  );

  await withPatchedChoysum(
    {
      request: {
        context: {
          req: { id: 'req-b' },
        },
      },
    },
    () => {
      expect(readSudoHits().length).toBe(0);
      withModelSudo(() => 'b');
      expect(readSudoHits().length).toBe(1);
    }
  );
});
