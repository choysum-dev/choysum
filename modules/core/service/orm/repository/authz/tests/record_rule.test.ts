// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  applyRepositoryRecordRuleToCondition,
  assertRepositoryRecordRuleAllCreatedAllowed,
  assertRepositoryRecordRuleAllTargetsAllowed,
  assertRepositoryRecordRuleCreateAllowed,
  getRepositoryRecordRuleEnvelope,
  replaceRepositoryRecordRuleTokens,
} from '..';

type CoordinatorOverrides = Partial<Record<string, unknown>>;

function createCoordinatorDeps(overrides: CoordinatorOverrides = {}) {
  const summaries: Array<Record<string, unknown>> = [];
  const denied: Array<{ code: string; message: string; metadata?: Record<string, string> }> = [];
  const calls: Record<string, unknown> = {};

  const deps = {
    meta: { fullModelName: 'demo.Model', modelName: 'Model', name: 'Model' },
    userId: 'user_1',
    recordRuleEnabled: () => true,
    getRecordRuleEnvelope: async () => ({ kind: 'true', reason: 'allow' }),
    replaceRecordRuleTokens: (condition: unknown) => condition,
    getReqMethodMeta: () => ({
      fullMethod: '/demo.Model/Search',
      method: 'Search',
      companyMode: 'strict',
      recordRuleMode: 'default',
      fieldRuleMode: 'default',
    }),
    getCompanyScopeFacts: () => ({ activeCompanyId: 'company_a', enabledCompanyIds: ['company_a'] }),
    emitAuthzDecisionSummary: (summary: Record<string, unknown>) => summaries.push(summary),
    permissionDenied: (code: string, message: string, metadata?: Record<string, string>) => {
      denied.push({ code, message, metadata });
      return new Error(`${code}:${message}`);
    },
    countConditionMatches: async (condition: unknown) => {
      calls.countCondition = condition;
      return 0;
    },
    ...overrides,
  } as any;

  return { deps, summaries, denied, calls };
}

test('record rule coordinator returns input condition unchanged when layer disabled', async () => {
  const { deps } = createCoordinatorDeps({ recordRuleEnabled: () => false });
  const input = ['Name', '=', 'demo'] as any;
  const output = await applyRepositoryRecordRuleToCondition(deps, input, 'read');
  expect(output).toEqual(input);
});

test('record rule coordinator returns input condition when envelope is true', async () => {
  const { deps } = createCoordinatorDeps({
    getRecordRuleEnvelope: async () => ({ kind: 'true', reason: 'allow' }),
  });
  const input = ['Name', '=', 'demo'] as any;
  const output = await applyRepositoryRecordRuleToCondition(deps, input, 'read');
  expect(output).toEqual(input);
});

test('record rule coordinator read-deny converts to never condition and emits deny summary', async () => {
  const { deps, summaries } = createCoordinatorDeps({
    getRecordRuleEnvelope: async () => ({ kind: 'false', reason: 'policy_deny' }),
  });
  const output = await applyRepositoryRecordRuleToCondition(deps, ['Name', '=', 'demo'] as any, 'read');

  expect(output).toEqual({
    And: [
      ['Name', '=', 'demo'],
      ['Id', '=', '__choysum_never__'],
    ],
  });
  expect(summaries.length).toBe(1);
  expect(summaries[0]?.decision).toBe('deny');
  expect(summaries[0]?.basis).toBe('record_rule_denied_read_empty_set');
});

test('record rule coordinator write-deny throws permission denied', async () => {
  const { deps } = createCoordinatorDeps({
    getRecordRuleEnvelope: async () => ({ kind: 'false', reason: 'policy_deny' }),
  });

  let message = '';
  try {
    await applyRepositoryRecordRuleToCondition(deps, ['Name', '=', 'demo'] as any, 'write');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_denied')).toBe(true);
});

test('record rule coordinator expr applies token replacement and emits allow summary', async () => {
  const { deps, summaries } = createCoordinatorDeps({
    getRecordRuleEnvelope: async () => ({ kind: 'expr', expr: ['OwnerId', '=', '$userId'], reason: 'rr_expr' }),
    replaceRecordRuleTokens: (condition: unknown) => {
      if (Array.isArray(condition)) {
        return ['OwnerId', '=', 'user_1'];
      }
      return condition;
    },
  });

  const output = await applyRepositoryRecordRuleToCondition(deps, ['Name', '=', 'demo'] as any, 'read');
  expect(output).toEqual({
    And: [
      ['Name', '=', 'demo'],
      ['OwnerId', '=', 'user_1'],
    ],
  });
  expect(summaries.length).toBe(1);
  expect(summaries[0]?.decision).toBe('allow');
  expect(summaries[0]?.basis).toBe('record_rule_expr_applied');
});

test('record rule target-guard validates full match count and throws on partial mismatch', async () => {
  const { deps, calls } = createCoordinatorDeps({
    getRecordRuleEnvelope: async () => ({ kind: 'expr', expr: ['OwnerId', '=', '$userId'], reason: 'rr_expr' }),
    replaceRecordRuleTokens: () => ['OwnerId', '=', 'user_1'],
    countConditionMatches: async (condition: unknown) => {
      calls.checkCondition = condition;
      return 1;
    },
  });

  let message = '';
  try {
    await assertRepositoryRecordRuleAllTargetsAllowed(deps, 'write', ['id_1', 'id_2']);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_violation')).toBe(true);
  expect(calls.checkCondition).toEqual({
    And: [
      ['Id', 'in', ['id_1', 'id_2']],
      ['OwnerId', '=', 'user_1'],
    ],
  });
});

test('record rule create guard denies create when envelope false', async () => {
  const { deps } = createCoordinatorDeps({
    getRecordRuleEnvelope: async () => ({ kind: 'false', reason: 'deny_create' }),
  });

  let message = '';
  try {
    await assertRepositoryRecordRuleCreateAllowed(deps);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_denied')).toBe(true);
});

test('record rule created-set guard verifies expr envelope and skips non-expr or disabled cases', async () => {
  const createdIds = ['id_1', 'id_2'];

  const pass = createCoordinatorDeps({
    replaceRecordRuleTokens: () => ['OwnerId', '=', 'user_1'],
    countConditionMatches: async () => 2,
  });
  await assertRepositoryRecordRuleAllCreatedAllowed(pass.deps, createdIds, {
    kind: 'expr',
    expr: ['OwnerId', '=', '$userId'],
    reason: 'rr_expr',
  } as any);

  const mismatch = createCoordinatorDeps({
    replaceRecordRuleTokens: () => ['OwnerId', '=', 'user_1'],
    countConditionMatches: async () => 1,
  });
  let mismatchMessage = '';
  try {
    await assertRepositoryRecordRuleAllCreatedAllowed(mismatch.deps, createdIds, {
      kind: 'expr',
      expr: ['OwnerId', '=', '$userId'],
      reason: 'rr_expr',
    } as any);
  } catch (error) {
    mismatchMessage = String((error as Error)?.message || error);
  }
  expect(mismatchMessage.includes('record_rule_violation')).toBe(true);

  const skipByKind = createCoordinatorDeps({
    countConditionMatches: async () => {
      throw new Error('should not be called for non-expr env');
    },
  });
  await assertRepositoryRecordRuleAllCreatedAllowed(skipByKind.deps, createdIds, { kind: 'true', reason: 'allow' } as any);

  const skipByDisabled = createCoordinatorDeps({
    recordRuleEnabled: () => false,
    countConditionMatches: async () => {
      throw new Error('should not be called when disabled');
    },
  });
  await assertRepositoryRecordRuleAllCreatedAllowed(skipByDisabled.deps, createdIds, {
    kind: 'expr',
    expr: ['OwnerId', '=', '$userId'],
  } as any);
});

test('record rule wrapper functions delegate to helper deps', async () => {
  const deps = {
    getCurrentReq: () => ({
      context: {
        req: {
          kind: 'grpc',
        },
      },
    }),
    meta: { fullModelName: 'demo.Model', modelName: 'Model', name: 'Model' },
    userId: 'user_1',
    normalizeCompanyIds: () => ['company_a'],
    normalizeCompanyIdForWrite: () => 'company_a',
    isControlPlaneMetaModel: () => false,
    recordRuleEnabled: () => true,
    getRecordRuleBypassDepth: () => 0,
    withRecordRuleBypass: async (fn: () => Promise<unknown>) => await fn(),
    permissionDenied: (code: string, message: string) => new Error(`${code}:${message}`),
  } as any;

  const envelope = await getRepositoryRecordRuleEnvelope(deps, 'read');
  expect(envelope.kind).toBe('true');

  const replaced = replaceRepositoryRecordRuleTokens(deps, ['OwnerId', '=', '$userId'] as any);
  expect(replaced).toEqual(['OwnerId', '=', 'user_1']);
});

test('record rule coordinator uses modelName fallback and denied reason default in read deny summary', async () => {
  const { deps, summaries } = createCoordinatorDeps({
    meta: { fullModelName: '', modelName: 'ModelOnly', name: '' },
    getRecordRuleEnvelope: async () => ({ kind: 'false', reason: '' }),
  });

  const out = await applyRepositoryRecordRuleToCondition(deps, ['Name', '=', 'demo'] as any, 'read');
  expect(out).toEqual({
    And: [
      ['Name', '=', 'demo'],
      ['Id', '=', '__choysum_never__'],
    ],
  });
  expect(summaries[0]?.model).toBe('ModelOnly');
  expect(summaries[0]?.reason).toBe('denied');
});

test('record rule coordinator write deny uses name fallback and denied reason default', async () => {
  const { deps, denied } = createCoordinatorDeps({
    meta: { fullModelName: '', modelName: '', name: 'NameOnly' },
    getRecordRuleEnvelope: async () => ({ kind: 'false', reason: '' }),
  });

  let message = '';
  try {
    await applyRepositoryRecordRuleToCondition(deps, ['Name', '=', 'demo'] as any, 'write');
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_denied')).toBe(true);
  expect(denied[0]?.metadata?.model).toBe('NameOnly');
  expect(denied[0]?.metadata?.reason).toBe('denied');
});

test('record rule coordinator expr summary uses model fallback and empty reason default', async () => {
  const { deps, summaries } = createCoordinatorDeps({
    meta: { fullModelName: '', modelName: '', name: 'NameOnly' },
    getRecordRuleEnvelope: async () => ({ kind: 'expr', expr: ['OwnerId', '=', '$userId'] }),
    replaceRecordRuleTokens: () => ['OwnerId', '=', 'user_1'],
  });

  const out = await applyRepositoryRecordRuleToCondition(deps, ['Name', '=', 'demo'] as any, 'read');
  expect(out).toEqual({
    And: [
      ['Name', '=', 'demo'],
      ['OwnerId', '=', 'user_1'],
    ],
  });
  expect(summaries[0]?.model).toBe('NameOnly');
  expect(summaries[0]?.reason).toBe('');
});

test('record rule target guard covers early returns and denial metadata fallbacks', async () => {
  const disabled = createCoordinatorDeps({
    recordRuleEnabled: () => false,
    countConditionMatches: async () => {
      throw new Error('should not run when disabled');
    },
  });
  await assertRepositoryRecordRuleAllTargetsAllowed(disabled.deps, 'write', ['id_1']);

  const emptyTargets = createCoordinatorDeps({
    countConditionMatches: async () => {
      throw new Error('should not run when no targets');
    },
  });
  await assertRepositoryRecordRuleAllTargetsAllowed(emptyTargets.deps, 'write', []);

  const allowEnv = createCoordinatorDeps({
    getRecordRuleEnvelope: async () => ({ kind: 'true' }),
    countConditionMatches: async () => {
      throw new Error('should not run when env is true');
    },
  });
  await assertRepositoryRecordRuleAllTargetsAllowed(allowEnv.deps, 'delete', ['id_1']);

  const deniedByEnv = createCoordinatorDeps({
    meta: { fullModelName: '', modelName: 'ModelOnly', name: '' },
    getRecordRuleEnvelope: async () => ({ kind: 'false', reason: '' }),
  });
  let message = '';
  try {
    await assertRepositoryRecordRuleAllTargetsAllowed(deniedByEnv.deps, 'delete', ['id_1']);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_denied')).toBe(true);
  expect(deniedByEnv.denied[0]?.metadata?.model).toBe('ModelOnly');
  expect(deniedByEnv.denied[0]?.metadata?.reason).toBe('denied');
});

test('record rule target/create guards use fallback metadata on violation paths', async () => {
  const mismatch = createCoordinatorDeps({
    meta: { fullModelName: '', modelName: '', name: 'NameOnly' },
    getRecordRuleEnvelope: async () => ({ kind: 'expr', expr: ['OwnerId', '=', '$userId'], reason: '' }),
    replaceRecordRuleTokens: () => ['OwnerId', '=', 'user_1'],
    countConditionMatches: async () => 0,
  });

  let message = '';
  try {
    await assertRepositoryRecordRuleAllTargetsAllowed(mismatch.deps, 'write', ['id_1']);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_violation')).toBe(true);
  expect(mismatch.denied[0]?.metadata?.model).toBe('NameOnly');
  expect(mismatch.denied[0]?.metadata?.reason).toBe('denied');

  const createDisabled = createCoordinatorDeps({
    recordRuleEnabled: () => false,
    getRecordRuleEnvelope: async () => {
      throw new Error('should not fetch when create guard disabled');
    },
  });
  await assertRepositoryRecordRuleCreateAllowed(createDisabled.deps);

  const createDenied = createCoordinatorDeps({
    meta: { fullModelName: '', modelName: 'ModelOnly', name: '' },
    getRecordRuleEnvelope: async () => ({ kind: 'false', reason: '' }),
  });
  message = '';
  try {
    await assertRepositoryRecordRuleCreateAllowed(createDenied.deps);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_denied')).toBe(true);
  expect(createDenied.denied[0]?.metadata?.model).toBe('ModelOnly');
  expect(createDenied.denied[0]?.metadata?.reason).toBe('denied');
});

test('record rule created-set guard covers early returns and fallback violation metadata', async () => {
  const disabled = createCoordinatorDeps({
    recordRuleEnabled: () => false,
    countConditionMatches: async () => {
      throw new Error('should not run when disabled');
    },
  });
  await assertRepositoryRecordRuleAllCreatedAllowed(disabled.deps, ['id_1'], { kind: 'expr', expr: ['Id', '=', 'id_1'] } as any);

  const emptyIds = createCoordinatorDeps({
    countConditionMatches: async () => {
      throw new Error('should not run when ids are empty');
    },
  });
  await assertRepositoryRecordRuleAllCreatedAllowed(emptyIds.deps, [], { kind: 'expr', expr: ['Id', '=', 'id_1'] } as any);

  const nonExpr = createCoordinatorDeps({
    countConditionMatches: async () => {
      throw new Error('should not run when env is non-expr');
    },
  });
  await assertRepositoryRecordRuleAllCreatedAllowed(nonExpr.deps, ['id_1'], { kind: 'true' } as any);

  const mismatch = createCoordinatorDeps({
    meta: { fullModelName: '', modelName: '', name: 'NameOnly' },
    replaceRecordRuleTokens: () => ['OwnerId', '=', 'user_1'],
    countConditionMatches: async () => 0,
  });
  let message = '';
  try {
    await assertRepositoryRecordRuleAllCreatedAllowed(mismatch.deps, ['id_1'], { kind: 'expr', expr: ['OwnerId', '=', '$userId'], reason: '' } as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_violation')).toBe(true);
  expect(mismatch.denied[0]?.metadata?.model).toBe('NameOnly');
  expect(mismatch.denied[0]?.metadata?.reason).toBe('denied');
});

test('record rule deny paths support empty model fallback chain', async () => {
  const readDeny = createCoordinatorDeps({
    meta: { fullModelName: '', modelName: '', name: '' },
    getRecordRuleEnvelope: async () => ({ kind: 'false', reason: '' }),
  });
  await applyRepositoryRecordRuleToCondition(readDeny.deps, ['Name', '=', 'demo'] as any, 'read');
  expect(readDeny.summaries[0]?.model).toBe('');

  const targetDeny = createCoordinatorDeps({
    meta: { fullModelName: '', modelName: '', name: '' },
    getRecordRuleEnvelope: async () => ({ kind: 'false', reason: '' }),
  });
  let message = '';
  try {
    await assertRepositoryRecordRuleAllTargetsAllowed(targetDeny.deps, 'write', ['id_1']);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_denied')).toBe(true);
  expect(targetDeny.denied[0]?.metadata?.model).toBe('');

  const createDeny = createCoordinatorDeps({
    meta: { fullModelName: '', modelName: '', name: '' },
    getRecordRuleEnvelope: async () => ({ kind: 'false', reason: '' }),
  });
  message = '';
  try {
    await assertRepositoryRecordRuleCreateAllowed(createDeny.deps);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('record_rule_denied')).toBe(true);
  expect(createDeny.denied[0]?.metadata?.model).toBe('');
});
