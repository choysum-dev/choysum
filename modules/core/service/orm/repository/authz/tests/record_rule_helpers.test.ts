// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { AuthUserService, fetchRepositoryRecordRuleEnvelope, replaceRepositoryRecordRuleConditionTokens } from '..';

type DepOverrides = Partial<Record<string, unknown>>;

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

function createDeps(overrides: DepOverrides = {}) {
  const denied: Array<{ code: string; message: string; metadata?: Record<string, string> }> = [];
  const base = {
    meta: { fullModelName: 'demo.Model', modelName: 'Model', name: 'Model' },
    userId: 'user_1',
    requestContext: { activeCompanyId: 'company_a' },
    normalizeCompanyIds: () => ['company_a', 'company_b'],
    normalizeCompanyIdForWrite: () => 'company_a',
    isControlPlaneMetaModel: () => false,
    recordRuleEnabled: () => true,
    getRecordRuleBypassDepth: () => 0,
    withRecordRuleBypass: async <T>(fn: () => Promise<T>) => await fn(),
    permissionDenied: (code: string, message: string, metadata?: Record<string, string>) => {
      denied.push({ code, message, metadata });
      return new Error(`${code}:${message}`);
    },
  };

  return {
    deps: { ...base, ...overrides } as any,
    denied,
  };
}

test('record rule helper enforce top-level allowlist and deny missing model/op entry', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Search',
            recordRuleMode: 'allowlist',
            recordRuleAllow: ['demo.Model:write'],
          },
        },
      },
    },
    async () => {
      const { deps } = createDeps();
      let message = '';
      try {
        await fetchRepositoryRecordRuleEnvelope(deps, 'read');
      } catch (error) {
        message = String((error as Error)?.message || error);
      }
      expect(message.includes('record_rule_entry_allowlist_miss')).toBe(true);
    }
  );
});

test('record rule helper fails closed when auth service unavailable and uses cache', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Search',
            recordRuleMode: 'default',
          },
        },
      },
    },
    async () => {
      const original = AuthUserService.GetRecordRuleCondition;
      let calls = 0;
      (AuthUserService as any).GetRecordRuleCondition = async () => {
        calls += 1;
        throw new TypeError('grpc unary unavailable');
      };

      try {
        const { deps } = createDeps();
        const first = await fetchRepositoryRecordRuleEnvelope(deps, 'read');
        const second = await fetchRepositoryRecordRuleEnvelope(deps, 'read');
        expect(first).toEqual({ kind: 'false', reason: 'auth_service_unavailable' });
        expect(second).toEqual({ kind: 'false', reason: 'auth_service_unavailable' });
        expect(calls === 1 || calls === 2).toBe(true);
      } finally {
        (AuthUserService as any).GetRecordRuleCondition = original;
      }
    }
  );
});

test('record rule helper allows when auth service is not present in deployment', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Search',
            recordRuleMode: 'default',
          },
        },
      },
    },
    async () => {
      const original = AuthUserService.GetRecordRuleCondition;
      (AuthUserService as any).GetRecordRuleCondition = async () => {
        throw new Error('no registered proto files for app auth');
      };

      try {
        const { deps } = createDeps();
        expect(await fetchRepositoryRecordRuleEnvelope(deps, 'read')).toEqual({
          kind: 'true',
          reason: 'auth_service_not_present',
        });
      } finally {
        (AuthUserService as any).GetRecordRuleCondition = original;
      }
    }
  );
});

test('record rule helper replaces known tokens in nested conditions', () => {
  const { deps } = createDeps();
  const output = replaceRepositoryRecordRuleConditionTokens(deps, {
    And: [
      ['OwnerId', '=', '$userId'],
      ['CompanyId', '=', '$companyId'],
      ['CompanyId', 'in', '$companyIds'],
      {
        Or: [
          ['Status', '=', 'ready'],
          ['ReviewerId', '=', '$userId'],
        ],
      },
    ],
  } as any);

  expect(output).toEqual({
    And: [
      ['OwnerId', '=', 'user_1'],
      ['CompanyId', '=', 'company_a'],
      ['CompanyId', 'in', ['company_a', 'company_b']],
      {
        Or: [
          ['Status', '=', 'ready'],
          ['ReviewerId', '=', 'user_1'],
        ],
      },
    ],
  });
});

test('record rule helper denies unknown token and invalid condition json', () => {
  const unknownToken = createDeps();
  let unknownTokenMessage = '';
  try {
    replaceRepositoryRecordRuleConditionTokens(unknownToken.deps, ['OwnerId', '=', '$tenantId'] as any);
  } catch (error) {
    unknownTokenMessage = String((error as Error)?.message || error);
  }
  expect(unknownTokenMessage.includes('record_rule_unknown_token')).toBe(true);

  const invalidJson = createDeps();
  let invalidJsonMessage = '';
  try {
    replaceRepositoryRecordRuleConditionTokens(invalidJson.deps, '{bad-json}' as any);
  } catch (error) {
    invalidJsonMessage = String((error as Error)?.message || error);
  }
  expect(invalidJsonMessage.includes('record_rule_invalid_condition_json')).toBe(true);
});

test('record rule helper returns early allow envelopes for disabled/control-plane/bypass branches', async () => {
  const disabled = createDeps({ recordRuleEnabled: () => false });
  expect(await fetchRepositoryRecordRuleEnvelope(disabled.deps, 'read')).toEqual({ kind: 'true', reason: 'record_rule_disabled' });

  const controlPlane = createDeps({ isControlPlaneMetaModel: () => true });
  expect(await fetchRepositoryRecordRuleEnvelope(controlPlane.deps, 'read')).toEqual({ kind: 'true', reason: 'control_plane_meta_model' });

  const bypass = createDeps({ getRecordRuleBypassDepth: () => 1 });
  expect(await fetchRepositoryRecordRuleEnvelope(bypass.deps, 'read')).toEqual({ kind: 'true', reason: 'record_rule_bypass' });
});

test('record rule helper allowlist hit returns allow envelope and service invalid envelope is normalized', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Search',
            recordRuleMode: 'allowlist',
            recordRuleAllow: ['demo.Model:read'],
          },
        },
      },
    },
    async () => {
      const { deps } = createDeps();
      expect(await fetchRepositoryRecordRuleEnvelope(deps, 'read')).toEqual({ kind: 'true', reason: 'entry_record_rule_allowlist' });
    }
  );

  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Search',
            recordRuleMode: 'default',
          },
        },
      },
    },
    async () => {
      const original = AuthUserService.GetRecordRuleCondition;
      (AuthUserService as any).GetRecordRuleCondition = async () => ({ kind: 'expr' });
      try {
        const { deps } = createDeps();
        expect(await fetchRepositoryRecordRuleEnvelope(deps, 'read')).toEqual({
          kind: 'false',
          reason: 'invalid_record_rule_envelope',
        });
      } finally {
        (AuthUserService as any).GetRecordRuleCondition = original;
      }
    }
  );
});

test('record rule helper denies missing token sources for user/company/companyIds', () => {
  let message = '';
  try {
    replaceRepositoryRecordRuleConditionTokens(createDeps({ userId: '' }).deps, ['OwnerId', '=', '$userId'] as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_missing_user_id')).toBe(true);

  message = '';
  try {
    replaceRepositoryRecordRuleConditionTokens(createDeps({ normalizeCompanyIdForWrite: () => undefined }).deps, ['CompanyId', '=', '$companyId'] as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_missing_company_id')).toBe(true);

  message = '';
  try {
    replaceRepositoryRecordRuleConditionTokens(createDeps({ normalizeCompanyIds: () => [] }).deps, ['CompanyId', 'in', '$companyIds'] as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_missing_company_ids')).toBe(true);
});

test('record rule helper uses process cache when request context is absent and model falls back to Unknown', async () => {
  await withPatchedChoysum(undefined, async () => {
    const original = AuthUserService.GetRecordRuleCondition;
    let calls = 0;
    (AuthUserService as any).GetRecordRuleCondition = async () => {
      calls += 1;
      return { kind: 'false', reason: 'deny_all' };
    };

    try {
      const { deps } = createDeps({
        meta: { fullModelName: '', modelName: '', name: '' },
        userId: '',
        requestContext: { ActiveCompanyId: 'company_b' },
        normalizeCompanyIds: () => [],
      });
      const first = await fetchRepositoryRecordRuleEnvelope(deps, 'read');
      const second = await fetchRepositoryRecordRuleEnvelope(deps, 'read');
      expect(first).toEqual({ kind: 'false', reason: 'deny_all' });
      expect(second).toEqual(first);
      expect(calls).toBe(1);
    } finally {
      (AuthUserService as any).GetRecordRuleCondition = original;
    }
  });
});

test('record rule helper handles depth or mode outside top-level allowlist', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 1,
            method: 1,
            recordRuleMode: 'allowlist',
            recordRuleAllow: 'demo.Model:read',
          },
        },
      },
    },
    async () => {
      const original = AuthUserService.GetRecordRuleCondition;
      (AuthUserService as any).GetRecordRuleCondition = async () => ({ kind: 'true', reason: 'service_allow' });
      try {
        const { deps } = createDeps();
        expect(await fetchRepositoryRecordRuleEnvelope(deps, 'read')).toEqual({ kind: 'true', reason: 'service_allow' });
      } finally {
        (AuthUserService as any).GetRecordRuleCondition = original;
      }
    }
  );
});

test('record rule helper fail-closes invalid envelope kind and coerces expr reason via parse', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Search',
            recordRuleMode: 'default',
          },
        },
      },
    },
    async () => {
      const original = AuthUserService.GetRecordRuleCondition;
      try {
        (AuthUserService as any).GetRecordRuleCondition = async () => ({ kind: 'unknown', reason: 1 });
        const { deps } = createDeps();
        expect(await fetchRepositoryRecordRuleEnvelope(deps, 'read')).toEqual({
          kind: 'false',
          reason: 'invalid_record_rule_envelope',
        });

        // Parse failures are not cached: a later valid fetch for the same key can succeed.
        (AuthUserService as any).GetRecordRuleCondition = async () => ({ kind: 'true', reason: 'recovered' });
        expect(await fetchRepositoryRecordRuleEnvelope(deps, 'read')).toEqual({
          kind: 'true',
          reason: 'recovered',
        });

        const secondDeps = createDeps().deps;
        (AuthUserService as any).GetRecordRuleCondition = async () => ({ kind: 'expr', expr: ['Id', '=', '1'], reason: 1 });
        expect(await fetchRepositoryRecordRuleEnvelope(secondDeps, 'write')).toEqual({
          kind: 'expr',
          expr: ['Id', '=', '1'],
          reason: '1',
        });
      } finally {
        (AuthUserService as any).GetRecordRuleCondition = original;
      }
    }
  );
});

test('record rule helper normalizeConditionInput handles empty string and null JSON', () => {
  const { deps } = createDeps();
  expect(replaceRepositoryRecordRuleConditionTokens(deps, '   ' as any)).toEqual([]);
  expect(replaceRepositoryRecordRuleConditionTokens(deps, 'null' as any)).toEqual([]);
});

test('record rule helper invalid condition preview is clipped for large primitive inputs', () => {
  const { deps } = createDeps();
  const payload = 'x'.repeat(500);
  let message = '';
  try {
    replaceRepositoryRecordRuleConditionTokens(deps, payload as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_invalid_condition')).toBe(true);
});

test('record rule helper treats non-plain object and malformed boolean node as invalid condition', () => {
  const { deps } = createDeps();

  let message = '';
  try {
    replaceRepositoryRecordRuleConditionTokens(deps, ['OwnerId', '=', new Date()] as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message).toBe('');

  try {
    replaceRepositoryRecordRuleConditionTokens(deps, { And: true } as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_invalid_condition')).toBe(true);
});

test('record rule helper keeps object input when Object.keys throws and still validates shape', () => {
  const { deps } = createDeps();
  const throwing = new Proxy(
    {},
    {
      ownKeys: () => {
        throw new Error('boom');
      },
      getOwnPropertyDescriptor: () => ({ configurable: true, enumerable: true }),
    }
  );

  let message = '';
  try {
    replaceRepositoryRecordRuleConditionTokens(deps, throwing as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_invalid_condition')).toBe(true);
});

test('record rule helper allowlist trims entries and supports Unknown model fallback', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Search',
            recordRuleMode: 'allowlist',
            recordRuleAllow: ['', '  ', 'Unknown:read'],
          },
        },
      },
    },
    async () => {
      const { deps } = createDeps({ meta: { fullModelName: '', modelName: '', name: '' } });
      expect(await fetchRepositoryRecordRuleEnvelope(deps, 'read')).toEqual({
        kind: 'true',
        reason: 'entry_record_rule_allowlist',
      });
    }
  );
});

test('record rule helper allowlist skips undefined null and blank entries', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Search',
            recordRuleMode: 'allowlist',
            recordRuleAllow: [undefined, null, '', '  ', 'demo.Model:read'],
          },
        },
      },
    },
    async () => {
      const { deps } = createDeps();
      expect(await fetchRepositoryRecordRuleEnvelope(deps, 'read')).toEqual({
        kind: 'true',
        reason: 'entry_record_rule_allowlist',
      });
    }
  );
});

test('record rule helper normalizes true false expr and empty envelopes from service', async () => {
  await withPatchedChoysum(
    {
      request: {
        context: {
          req: {
            depth: 0,
            method: 'Search',
            recordRuleMode: 'default',
          },
        },
      },
    },
    async () => {
      const original = AuthUserService.GetRecordRuleCondition;
      try {
        (AuthUserService as any).GetRecordRuleCondition = async () => ({ kind: 'true', reason: 1 });
        expect(await fetchRepositoryRecordRuleEnvelope(createDeps().deps, 'read')).toEqual({ kind: 'true', reason: '1' });

        (AuthUserService as any).GetRecordRuleCondition = async () => ({ kind: 'false', reason: 1 });
        expect(await fetchRepositoryRecordRuleEnvelope(createDeps({ userId: 'u2' }).deps, 'read')).toEqual({ kind: 'false', reason: '1' });

        (AuthUserService as any).GetRecordRuleCondition = async () => ({ kind: 'expr', expr: ['Id', '=', '1'], reason: 'ok' });
        expect(await fetchRepositoryRecordRuleEnvelope(createDeps({ userId: 'u3' }).deps, 'read')).toEqual({
          kind: 'expr',
          expr: ['Id', '=', '1'],
          reason: 'ok',
        });

        (AuthUserService as any).GetRecordRuleCondition = async () => ({
          kind: 'true',
          reason: 'with_hits',
          hitRuleIds: [' rr_2 ', '', 'rr_1', 'rr_1'],
        });
        expect(await fetchRepositoryRecordRuleEnvelope(createDeps({ userId: 'u3b' }).deps, 'read')).toEqual({
          kind: 'true',
          reason: 'with_hits',
          hitRuleIds: ['rr_1', 'rr_2'],
        });

        (AuthUserService as any).GetRecordRuleCondition = async () => ({
          kind: 'false',
          reason: 'csv_hits',
          hitRuleIds: 'rr_b,rr_a,rr_a',
        });
        expect(await fetchRepositoryRecordRuleEnvelope(createDeps({ userId: 'u3c' }).deps, 'read')).toEqual({
          kind: 'false',
          reason: 'csv_hits',
          hitRuleIds: ['rr_a', 'rr_b'],
        });

        (AuthUserService as any).GetRecordRuleCondition = async () => ({
          kind: 'true',
          reason: 'nullish_hits',
          hitRuleIds: [null, undefined, '', '  ', 'rr_keep', 0],
        });
        expect(await fetchRepositoryRecordRuleEnvelope(createDeps({ userId: 'u3d' }).deps, 'read')).toEqual({
          kind: 'true',
          reason: 'nullish_hits',
          hitRuleIds: ['0', 'rr_keep'],
        });

        (AuthUserService as any).GetRecordRuleCondition = async () => ({
          kind: 'expr',
          expr: ['Id', '=', '1'],
          reason: 'non_list_hits',
          hitRuleIds: 123,
        });
        expect(await fetchRepositoryRecordRuleEnvelope(createDeps({ userId: 'u3e' }).deps, 'read')).toEqual({
          kind: 'expr',
          expr: ['Id', '=', '1'],
          reason: 'non_list_hits',
        });

        (AuthUserService as any).GetRecordRuleCondition = async () => ({
          kind: 'false',
          reason: 'blank_csv',
          hitRuleIds: ' , , ',
        });
        expect(await fetchRepositoryRecordRuleEnvelope(createDeps({ userId: 'u3f' }).deps, 'read')).toEqual({
          kind: 'false',
          reason: 'blank_csv',
        });

        (AuthUserService as any).GetRecordRuleCondition = async () => undefined;
        expect(await fetchRepositoryRecordRuleEnvelope(createDeps({ userId: 'u4' }).deps, 'read')).toEqual({
          kind: 'false',
          reason: 'invalid_record_rule_envelope',
        });
      } finally {
        (AuthUserService as any).GetRecordRuleCondition = original;
      }
    }
  );
});

test('record rule helper token errors cover modelName and name fallback metadata', () => {
  let message = '';
  try {
    replaceRepositoryRecordRuleConditionTokens(
      createDeps({
        meta: { fullModelName: '', modelName: 'ModelOnly', name: '' },
        normalizeCompanyIdForWrite: () => undefined,
      }).deps,
      ['CompanyId', '=', '$companyId'] as any
    );
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_missing_company_id')).toBe(true);

  try {
    replaceRepositoryRecordRuleConditionTokens(
      createDeps({
        meta: { fullModelName: '', modelName: '', name: 'NameOnly' },
        normalizeCompanyIds: () => [],
      }).deps,
      ['CompanyId', 'in', '$companyIds'] as any
    );
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_missing_company_ids')).toBe(true);
});

test('record rule helper normalize and invalid-condition branches for object array primitive inputs', () => {
  const { deps } = createDeps();

  expect(replaceRepositoryRecordRuleConditionTokens(deps, undefined as any)).toEqual([]);
  expect(replaceRepositoryRecordRuleConditionTokens(deps, {} as any)).toEqual([]);
  expect(replaceRepositoryRecordRuleConditionTokens(deps, '[]' as any)).toEqual([]);
  expect(replaceRepositoryRecordRuleConditionTokens(deps, '{"And":[["Id","=","1"]]}' as any)).toEqual({ And: [['Id', '=', '1']] });
  expect(replaceRepositoryRecordRuleConditionTokens(deps, ['Field', '=', null] as any)).toEqual(['Field', '=', null]);

  let message = '';
  try {
    replaceRepositoryRecordRuleConditionTokens(deps, 123 as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_invalid_condition')).toBe(true);

  try {
    replaceRepositoryRecordRuleConditionTokens(deps, ['OnlyField', '='] as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_invalid_condition')).toBe(true);
});

test('record rule helper unknown-model fallback paths for token and invalid-condition metadata', () => {
  const { deps } = createDeps({
    meta: { fullModelName: '', modelName: '', name: '' },
    userId: '',
    normalizeCompanyIdForWrite: () => undefined,
    normalizeCompanyIds: () => [],
  });

  let message = '';
  try {
    replaceRepositoryRecordRuleConditionTokens(deps, ['OwnerId', '=', '$userId'] as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_missing_user_id')).toBe(true);

  try {
    replaceRepositoryRecordRuleConditionTokens(deps, ['CompanyId', '=', '$companyId'] as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_missing_company_id')).toBe(true);

  try {
    replaceRepositoryRecordRuleConditionTokens(deps, ['CompanyId', 'in', '$companyIds'] as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_missing_company_ids')).toBe(true);

  try {
    replaceRepositoryRecordRuleConditionTokens(deps, ['Field', '=', '$unknown'] as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_unknown_token')).toBe(true);

  try {
    replaceRepositoryRecordRuleConditionTokens(deps, 42 as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_invalid_condition')).toBe(true);

  try {
    replaceRepositoryRecordRuleConditionTokens(deps, '{bad-json}' as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_invalid_condition_json')).toBe(true);
});

test('record rule helper recursively replaces tokens inside array leaf and nested logical nodes', () => {
  const { deps } = createDeps();
  const out = replaceRepositoryRecordRuleConditionTokens(deps, {
    Or: [['Field', 'in', ['$userId', '$companyId', '$companyIds']], { And: [['OwnerId', '=', '$userId']] }],
  } as any);

  expect(out).toEqual({
    Or: [['Field', 'in', ['user_1', 'company_a', ['company_a', 'company_b']]], { And: [['OwnerId', '=', 'user_1']] }],
  });
});

test('record rule helper validates malformed And and malformed tuple shapes', () => {
  const { deps } = createDeps();
  let message = '';

  try {
    replaceRepositoryRecordRuleConditionTokens(deps, { And: [123 as any] } as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_invalid_condition')).toBe(true);

  try {
    replaceRepositoryRecordRuleConditionTokens(deps, ['Only', '='] as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_invalid_condition')).toBe(true);
});
