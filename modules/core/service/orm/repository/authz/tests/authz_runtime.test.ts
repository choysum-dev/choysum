// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  emitRepositoryAuthzDecisionSummary,
  getOrInitRepositoryReqServiceState,
  getRepositoryAuthzDecisionLogMode,
  getRepositoryCompanyScopeFacts,
  getRepositoryCurrentReq,
  getValidationBypassState,
  getFieldRuleBypassDepth,
  getRecordRuleBypassDepth,
  getRepositoryReqMethodMeta,
  repositoryAuthzDecisionAuditEnabled,
  getValidationBypassDepth,
  isRepositoryTopLevelGrpcCall,
  withFieldRuleBypass,
  withRecordRuleBypass,
  withRecordRuleAndFieldRuleBypass,
  withValidationBypass,
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
      expect(getRecordRuleBypassDepth()).toBe(0);
      expect(getFieldRuleBypassDepth()).toBe(0);

      await withRecordRuleBypass(async () => {
        expect(getRecordRuleBypassDepth()).toBe(1);
        await withRecordRuleBypass(async () => {
          expect(getRecordRuleBypassDepth()).toBe(2);
        });
        expect(getRecordRuleBypassDepth()).toBe(1);
      });

      await withFieldRuleBypass(async () => {
        expect(getFieldRuleBypassDepth()).toBe(1);
      });

      expect(getRecordRuleBypassDepth()).toBe(0);
      expect(getFieldRuleBypassDepth()).toBe(0);
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
    expect(getValidationBypassDepth()).toBe(0);
    await withValidationBypass(async () => {
      expect(getValidationBypassDepth()).toBe(1);
    });
    expect(getValidationBypassDepth()).toBe(0);
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
    expect(await withRecordRuleBypass(async () => 'rr-ok')).toBe('rr-ok');
    expect(await withFieldRuleBypass(async () => 'fr-ok')).toBe('fr-ok');
    expect(withRecordRuleAndFieldRuleBypass(() => 'rr-fr-ok')).toBe('rr-fr-ok');
    expect(withRecordRuleBypass(() => 'rr-sync')).toBe('rr-sync');
    expect(withFieldRuleBypass(() => 'fr-sync')).toBe('fr-sync');
  } finally {
    if (previous !== undefined) (globalThis as Record<string, unknown>).$choysum = previous;
  }
});

test('authz runtime bypass restore runs on sync throw', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {},
        },
      },
    },
    async () => {
      expect(() =>
        withRecordRuleAndFieldRuleBypass(() => {
          expect(getRecordRuleBypassDepth()).toBe(1);
          throw new Error('bypass-boom');
        })
      ).toThrow('bypass-boom');
      expect(getRecordRuleBypassDepth()).toBe(0);
      expect(getFieldRuleBypassDepth()).toBe(0);

      expect(() =>
        withRecordRuleBypass(() => {
          throw new Error('rr-boom');
        })
      ).toThrow('rr-boom');
      expect(getRecordRuleBypassDepth()).toBe(0);
    }
  );
});

test('authz runtime bypass restore treats non-finite depth as zero', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {},
        },
      },
    },
    async () => {
      withRecordRuleBypass(() => {
        const req = getRepositoryCurrentReq();
        const state = getOrInitRepositoryReqServiceState(req)!;
        state.recordRuleBypassDepth = Number.NaN;
      });
      expect(getRecordRuleBypassDepth()).toBe(0);

      withFieldRuleBypass(() => {
        const req = getRepositoryCurrentReq();
        const state = getOrInitRepositoryReqServiceState(req)!;
        state.fieldRuleBypassDepth = Number.POSITIVE_INFINITY;
      });
      expect(getFieldRuleBypassDepth()).toBe(0);

      withRecordRuleAndFieldRuleBypass(() => {
        const req = getRepositoryCurrentReq();
        const state = getOrInitRepositoryReqServiceState(req)!;
        state.recordRuleBypassDepth = 'x' as any;
        state.fieldRuleBypassDepth = undefined;
      });
      expect(getRecordRuleBypassDepth()).toBe(0);
      expect(getFieldRuleBypassDepth()).toBe(0);
    }
  );
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
      const syncValue = withRecordRuleAndFieldRuleBypass(() => {
        expect(getRecordRuleBypassDepth()).toBe(1);
        expect(getFieldRuleBypassDepth()).toBe(1);
        return 'sync-ok';
      });
      expect(syncValue).toBe('sync-ok');
      expect(getRecordRuleBypassDepth()).toBe(0);

      await withRecordRuleAndFieldRuleBypass(async () => {
        expect(getRecordRuleBypassDepth()).toBe(1);
        expect(getFieldRuleBypassDepth()).toBe(1);
        await withRecordRuleAndFieldRuleBypass(async () => {
          expect(getRecordRuleBypassDepth()).toBe(2);
          expect(getFieldRuleBypassDepth()).toBe(2);
        });
        expect(getRecordRuleBypassDepth()).toBe(1);
        expect(getFieldRuleBypassDepth()).toBe(1);
      });
      expect(getRecordRuleBypassDepth()).toBe(0);
      expect(getFieldRuleBypassDepth()).toBe(0);
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
      await withFieldRuleBypass(async () => {
        expect(getFieldRuleBypassDepth()).toBe(1);
        await withFieldRuleBypass(async () => {
          expect(getFieldRuleBypassDepth()).toBe(2);
        });
        expect(getFieldRuleBypassDepth()).toBe(1);
      });
      expect(getFieldRuleBypassDepth()).toBe(0);
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

      const slow = withRecordRuleBypass(async () => {
        await slowGate;
        expect(getRecordRuleBypassDepth()).toBeGreaterThanOrEqual(1);
      });

      const fast = withRecordRuleBypass(async () => {
        expect(getRecordRuleBypassDepth()).toBe(2);
      });

      await fast;
      // First-started bypass must still be elevated after the second finishes.
      expect(getRecordRuleBypassDepth()).toBe(1);
      releaseSlow?.();
      await slow;
      expect(getRecordRuleBypassDepth()).toBe(0);
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
      const state = getValidationBypassState();
      expect(state?.validationBypassDepth).toBe(7);
      expect(getValidationBypassDepth()).toBe(7);
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

    logs.length = 0;
    emitRepositoryAuthzDecisionSummary({
      decision: 'deny',
      layer: 'record_rule',
      basis: 'record_rule_denied',
      metadata: { reason: 'no_grant_read_deny', hitRuleIds: 'rule_b,rule_a,rule_a' },
    });
    expect(logs.length).toBe(2);
    expect(logs[0].includes('"reason":"no_grant_read_deny"')).toBe(true);
    expect(logs[0].includes('"hitRuleIds":["rule_a","rule_b"]')).toBe(true);

    logs.length = 0;
    emitRepositoryAuthzDecisionSummary({
      decision: 'deny',
      layer: 'record_rule',
      reason: '  top_level  ',
      hitRuleIds: [' hit_2 ', '', null as any, 'hit_1', 'hit_1'],
      metadata: { reason: 'ignored_when_top_level_present', hitRuleIds: 'should_not_win' },
    });
    expect(logs.length).toBe(2);
    expect(logs[0].includes('"reason":"top_level"')).toBe(true);
    expect(logs[0].includes('"hitRuleIds":["hit_1","hit_2"]')).toBe(true);

    logs.length = 0;
    emitRepositoryAuthzDecisionSummary({
      decision: 'deny',
      layer: 'field_rule',
      basis: 'field_rule_readonly_violation',
      hitRuleIds: ['', '  '],
      metadata: {},
    });
    expect(logs.length).toBe(2);
    expect(logs[0].includes('hitRuleIds')).toBe(false);

    logs.length = 0;
    emitRepositoryAuthzDecisionSummary({
      decision: 'deny',
      layer: 'record_rule',
      reason: '   ',
      metadata: { reason: 'from_metadata' },
    });
    expect(logs.length).toBe(2);
    expect(logs[0].includes('"reason":"from_metadata"')).toBe(true);

    logs.length = 0;
    emitRepositoryAuthzDecisionSummary({
      decision: 'deny',
      layer: 'record_rule',
      hitRuleIds: 42 as any,
      metadata: 'not-an-object' as any,
    });
    expect(logs.length).toBe(2);
    expect(logs[0].includes('hitRuleIds')).toBe(false);
  } finally {
    (console as any).error = originalError;
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = originalEnv;
  }
});
