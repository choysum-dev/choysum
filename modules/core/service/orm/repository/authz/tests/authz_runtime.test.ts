// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  emitRepositoryAuthzDecisionSummary,
  getOrInitRepositoryReqServiceState,
  getRepositoryAuthzDecisionLogMode,
  getRepositoryCompanyScopeFacts,
  getRepositoryCurrentReq,
  getRepositoryValidationBypassState,
  getRepositoryFieldRuleBypassDepth,
  getRepositoryRecordRuleBypassDepth,
  getRepositoryReqMethodMeta,
  repositoryAuthzDecisionAuditEnabled,
  getRepositoryValidationBypassDepth,
  isRepositoryTopLevelGrpcCall,
  withRepositoryFieldRuleBypass,
  withRepositoryRecordRuleBypass,
  withRepositoryAuthzRuleBypass,
  withRepositoryValidationBypass,
} from '..';

async function withPatchedChoysum<T>(value: unknown, fn: () => Promise<T>): Promise<T> {
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

test('authz runtime reads current req and method/company facts from request context', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            fullMethod: '/demo.Model/Search',
            method: 'Search',
            companyMode: 'strict',
            recordRuleMode: 'default',
            fieldRuleMode: 'default',
          },
        },
      },
    },
    async () => {
      const req = getRepositoryCurrentReq();
      expect(req?.method).toBe('Search');
      expect(getRepositoryReqMethodMeta()).toEqual({
        fullMethod: '/demo.Model/Search',
        method: 'Search',
        companyMode: 'strict',
        recordRuleMode: 'default',
        fieldRuleMode: 'default',
      });

      expect(getRepositoryCompanyScopeFacts({ activeCompanyId: 'company_a' }, ['company_a'])).toEqual({
        activeCompanyId: 'company_a',
        enabledCompanyIds: ['company_a'],
      });
    }
  );
});

test('authz runtime top-level grpc detection respects depth from wrapper state and req', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: { kind: 'grpc', depth: 0 },
        },
        __choysumServiceState: { depth: 1 },
      },
    },
    async () => {
      expect(isRepositoryTopLevelGrpcCall()).toBe(true);
    }
  );

  await withPatchedChoysum(
    {
      request: {
        context: {
          req: { kind: 'grpc-web', depth: 0 },
        },
        __choysumServiceState: { depth: 2 },
      },
    },
    async () => {
      expect(isRepositoryTopLevelGrpcCall()).toBe(false);
    }
  );
});

test('authz runtime record-rule and field-rule bypass depth are nested and restored', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {},
        },
      },
    },
    async () => {
      const req = getRepositoryCurrentReq();
      const state = getOrInitRepositoryReqServiceState(req);
      expect(state).toBeTruthy();
      expect(getRepositoryRecordRuleBypassDepth()).toBe(0);
      expect(getRepositoryFieldRuleBypassDepth()).toBe(0);

      await withRepositoryRecordRuleBypass(async () => {
        expect(getRepositoryRecordRuleBypassDepth()).toBe(1);
        await withRepositoryRecordRuleBypass(async () => {
          expect(getRepositoryRecordRuleBypassDepth()).toBe(2);
        });
        expect(getRepositoryRecordRuleBypassDepth()).toBe(1);
      });

      await withRepositoryFieldRuleBypass(async () => {
        expect(getRepositoryFieldRuleBypassDepth()).toBe(1);
      });

      expect(getRepositoryRecordRuleBypassDepth()).toBe(0);
      expect(getRepositoryFieldRuleBypassDepth()).toBe(0);
    }
  );
});

test('authz runtime validation bypass falls back to global state when req is absent', async () => {
  const key = '__choysumRepositoryServiceState';
  const previous = (globalThis as Record<string, unknown>)[key];
  delete (globalThis as Record<string, unknown>)[key];

  const choysumPrev = (globalThis as Record<string, unknown>).$choysum;
  delete (globalThis as Record<string, unknown>).$choysum;
  try {
    expect(getRepositoryValidationBypassDepth()).toBe(0);
    await withRepositoryValidationBypass(async () => {
      expect(getRepositoryValidationBypassDepth()).toBe(1);
    });
    expect(getRepositoryValidationBypassDepth()).toBe(0);
  } finally {
    if (previous !== undefined) (globalThis as Record<string, unknown>)[key] = previous;
    else delete (globalThis as Record<string, unknown>)[key];

    if (choysumPrev !== undefined) (globalThis as Record<string, unknown>).$choysum = choysumPrev;
  }
});

test('authz runtime bypass wrappers run without req state and return callback results', async () => {
  const previous = (globalThis as Record<string, unknown>).$choysum;
  delete (globalThis as Record<string, unknown>).$choysum;
  try {
    expect(await withRepositoryRecordRuleBypass(async () => 'rr-ok')).toBe('rr-ok');
    expect(await withRepositoryFieldRuleBypass(async () => 'fr-ok')).toBe('fr-ok');
  } finally {
    if (previous !== undefined) (globalThis as Record<string, unknown>).$choysum = previous;
  }
});

test('authz runtime combined RR+FR bypass is sync-friendly and nested', () => {
  return withPatchedChoysum(
    {
      request: {
        context: {
          req: {},
        },
      },
    },
    async () => {
      const syncValue = withRepositoryAuthzRuleBypass(() => {
        expect(getRepositoryRecordRuleBypassDepth()).toBe(1);
        expect(getRepositoryFieldRuleBypassDepth()).toBe(1);
        return 'sync-ok';
      });
      expect(syncValue).toBe('sync-ok');
      expect(getRepositoryRecordRuleBypassDepth()).toBe(0);

      await withRepositoryAuthzRuleBypass(async () => {
        expect(getRepositoryRecordRuleBypassDepth()).toBe(1);
        expect(getRepositoryFieldRuleBypassDepth()).toBe(1);
      });
      expect(getRepositoryFieldRuleBypassDepth()).toBe(0);
    }
  );
});

test('authz runtime field bypass nested depth restores previous value', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {},
        },
      },
    },
    async () => {
      await withRepositoryFieldRuleBypass(async () => {
        expect(getRepositoryFieldRuleBypassDepth()).toBe(1);
        await withRepositoryFieldRuleBypass(async () => {
          expect(getRepositoryFieldRuleBypassDepth()).toBe(2);
        });
        expect(getRepositoryFieldRuleBypassDepth()).toBe(1);
      });
      expect(getRepositoryFieldRuleBypassDepth()).toBe(0);
    }
  );
});

test('authz runtime concurrent sibling bypasses survive out-of-order completion', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {},
        },
      },
    },
    async () => {
      let releaseSlow: (() => void) | undefined;
      const slowGate = new Promise<void>(resolve => {
        releaseSlow = resolve;
      });

      const slow = withRepositoryRecordRuleBypass(async () => {
        await slowGate;
        expect(getRepositoryRecordRuleBypassDepth()).toBeGreaterThanOrEqual(1);
      });

      const fast = withRepositoryRecordRuleBypass(async () => {
        expect(getRepositoryRecordRuleBypassDepth()).toBe(2);
      });

      await fast;
      // First-started bypass must still be elevated after the second finishes.
      expect(getRepositoryRecordRuleBypassDepth()).toBe(1);
      releaseSlow?.();
      await slow;
      expect(getRepositoryRecordRuleBypassDepth()).toBe(0);
    }
  );
});

test('authz runtime log mode and audit switch normalize mixed env values', () => {
  const originalEnv = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
  try {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_AUTHZ_DECISION_LOG: 'deny', CHOYSUM_AUTHZ_DECISION_AUDIT_ENABLED: false };
    expect(getRepositoryAuthzDecisionLogMode()).toBe('deny');
    expect(repositoryAuthzDecisionAuditEnabled()).toBe(false);

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_AUTHZ_DECISION_LOG: null, CHOYSUM_AUTHZ_DECISION_AUDIT_ENABLED: true };
    expect(getRepositoryAuthzDecisionLogMode()).toBe('off');
    expect(repositoryAuthzDecisionAuditEnabled()).toBe(true);

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_AUTHZ_DECISION_LOG: ' ALL ', CHOYSUM_AUTHZ_DECISION_AUDIT_ENABLED: 'TRUE' };
    expect(getRepositoryAuthzDecisionLogMode()).toBe('all');
    expect(repositoryAuthzDecisionAuditEnabled()).toBe(true);

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_AUTHZ_DECISION_LOG: 'deny', CHOYSUM_AUTHZ_DECISION_AUDIT_ENABLED: 'x' };
    expect(getRepositoryAuthzDecisionLogMode()).toBe('deny');
    expect(repositoryAuthzDecisionAuditEnabled()).toBe(false);

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_AUTHZ_DECISION_LOG: 'deny', CHOYSUM_AUTHZ_DECISION_AUDIT_ENABLED: 1 };
    expect(getRepositoryAuthzDecisionLogMode()).toBe('deny');
    expect(repositoryAuthzDecisionAuditEnabled()).toBe(false);
  } finally {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = originalEnv;
  }
});

test('authz runtime decision summary handles missing decision field', () => {
  const originalEnv = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
  const originalError = console.error;
  const logs: string[] = [];
  (console as any).error = (...args: any[]) => {
    logs.push(args.map(item => String(item)).join(' '));
  };

  try {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = {
      CHOYSUM_AUTHZ_DECISION_LOG: 'deny',
      CHOYSUM_AUTHZ_DECISION_AUDIT_ENABLED: false,
    };
    emitRepositoryAuthzDecisionSummary({ layer: 'record_rule' });
    expect(logs.length).toBe(0);

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = {
      CHOYSUM_AUTHZ_DECISION_LOG: 'all',
      CHOYSUM_AUTHZ_DECISION_AUDIT_ENABLED: false,
    };
    emitRepositoryAuthzDecisionSummary({ layer: 'record_rule' });
    expect(logs.length).toBe(1);
    expect(logs[0].includes('[AUTHZ]')).toBe(true);
  } finally {
    (console as any).error = originalError;
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = originalEnv;
  }
});

test('authz runtime validation bypass state prefers req state when present', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            __choysumServiceState: {
              validationBypassDepth: 7,
            },
          },
        },
      },
    },
    async () => {
      const state = getRepositoryValidationBypassState();
      expect(state?.validationBypassDepth).toBe(7);
      expect(getRepositoryValidationBypassDepth()).toBe(7);
    }
  );
});

test('authz runtime decision summary log mode off/deny/all with audit toggle', async () => {
  const originalEnv = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
  const originalError = console.error;
  const logs: string[] = [];
  (console as any).error = (...args: any[]) => {
    logs.push(args.map(item => String(item)).join(' '));
  };

  try {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = {
      CHOYSUM_AUTHZ_DECISION_LOG: 'off',
      CHOYSUM_AUTHZ_DECISION_AUDIT_ENABLED: false,
    };
    emitRepositoryAuthzDecisionSummary({ decision: 'deny', layer: 'record_rule' });
    expect(logs.length).toBe(0);

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = {
      CHOYSUM_AUTHZ_DECISION_LOG: 'deny',
      CHOYSUM_AUTHZ_DECISION_AUDIT_ENABLED: false,
    };
    emitRepositoryAuthzDecisionSummary({ decision: 'allow', layer: 'company_filter' });
    expect(logs.length).toBe(0);

    emitRepositoryAuthzDecisionSummary({ decision: 'deny', layer: 'record_rule', basis: 'denied' });
    expect(logs.length).toBe(1);
    expect(logs[0].includes('[AUTHZ]')).toBe(true);
    expect(logs[0].includes('"decision":"deny"')).toBe(true);

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = {
      CHOYSUM_AUTHZ_DECISION_LOG: 'all',
      CHOYSUM_AUTHZ_DECISION_AUDIT_ENABLED: 'true',
    };
    emitRepositoryAuthzDecisionSummary({ decision: 'allow', layer: 'field_rule', basis: 'applied' });
    expect(logs.length).toBe(3);
    expect(logs[1].includes('[AUTHZ]')).toBe(true);
    expect(logs[2].includes('[AUDIT]')).toBe(true);
  } finally {
    (console as any).error = originalError;
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = originalEnv;
  }
});
