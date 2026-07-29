// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  applyRepositoryCompanyLayer,
  applyRepositoryDefaultCompanyIdOnCreate,
  applyRepositoryDefaultCompanyIdOnUpdate,
  assertRepositoryCompanyWriteAccessForCondition,
  isRepositoryOwnershipFieldNotNull,
  normalizeRepositoryCompanyIdForWrite,
  normalizeRepositoryCompanyIds,
  repositoryCompanyFieldEnabled,
  requireRepositoryOwnershipField,
  validateRepositoryCompanyIdInScope,
  validateRepositoryOwnershipNullability,
} from '..';

type DeniedCall = { code: string; message: string; metadata?: Record<string, string> };

function createCompanyDeps(overrides: Partial<Record<string, unknown>> = {}) {
  const denied: DeniedCall[] = [];
  const summaries: Array<Record<string, unknown>> = [];
  const base = {
    meta: {
      fullModelName: 'demo.Model',
      modelName: 'Model',
      name: 'Model',
      companyField: 'CompanyId',
      fields: new Map<string, unknown>([['CompanyId', { type: 'char' }]]),
    },
    ctx: { activeCompanyId: 'company_a', enabledCompanyIds: ['company_a', 'company_b'] },
    userId: 'user_1',
    companyLayerSkipped: () => false,
    getReqMethodMeta: () => ({
      fullMethod: '/demo.Model/Search',
      method: 'Search',
      companyMode: 'strict',
      recordRuleMode: 'default',
      fieldRuleMode: 'default',
    }),
    getCompanyScopeFacts: () => ({ activeCompanyId: 'company_a', enabledCompanyIds: ['company_a', 'company_b'] }),
    emitAuthzDecisionSummary: (summary: Record<string, unknown>) => summaries.push(summary),
    permissionDenied: (code: string, message: string, metadata?: Record<string, string>) => {
      denied.push({ code, message, metadata });
      return new Error(`${code}:${message}`);
    },
  };

  return {
    deps: { ...base, ...overrides } as any,
    denied,
    summaries,
  };
}

test('company scope normalizers handle active and enabled company values', () => {
  expect(normalizeRepositoryCompanyIds({ enabledCompanyIds: ['company_a', 'company_b', 'company_a'] })).toEqual(['company_a', 'company_b']);
  expect(normalizeRepositoryCompanyIds({ activeCompanyId: 'company_a' })).toEqual(['company_a']);
  expect(normalizeRepositoryCompanyIdForWrite({ activeCompanyId: 'company_a' })).toBe('company_a');
  expect(normalizeRepositoryCompanyIdForWrite({ enabledCompanyIds: ['company_only'] })).toBe('company_only');
});

test('company scope enabled check enforces CompanyId field and context company ids', () => {
  const missingField = createCompanyDeps({
    meta: {
      fullModelName: 'demo.Model',
      companyField: 'CompanyId',
      fields: new Map<string, unknown>(),
    },
  });

  let missingFieldMessage = '';
  try {
    repositoryCompanyFieldEnabled(missingField.deps);
  } catch (error) {
    missingFieldMessage = String((error as Error)?.message || error);
  }
  expect(missingFieldMessage.includes('company_field_missing')).toBe(true);

  const missingCtx = createCompanyDeps({ ctx: {} });
  let missingCtxMessage = '';
  try {
    repositoryCompanyFieldEnabled(missingCtx.deps);
  } catch (error) {
    missingCtxMessage = String((error as Error)?.message || error);
  }
  expect(missingCtxMessage.includes('company_scope_missing_ctx_company')).toBe(true);

  const enabled = createCompanyDeps();
  expect(repositoryCompanyFieldEnabled(enabled.deps)).toBe(true);
});

test('company scope layer appends company condition and emits allow summary', () => {
  const { deps, summaries } = createCompanyDeps();
  const condition = applyRepositoryCompanyLayer(deps, ['Status', '=', 'ready'] as any);

  expect(condition).toEqual({
    And: [
      ['Status', '=', 'ready'],
      {
        Or: [
          ['CompanyId', 'in', ['company_a', 'company_b']],
          ['CompanyId', 'is', null],
        ],
      },
    ],
  });
  expect(summaries.length).toBe(1);
  expect(summaries[0]?.layer).toBe('company_filter');
  expect(summaries[0]?.decision).toBe('allow');
});

test('company scope uses aliased companyField for filter create and write access', async () => {
  const { deps } = createCompanyDeps({
    meta: {
      fullModelName: 'demo.AliasModel',
      modelName: 'AliasModel',
      name: 'AliasModel',
      companyField: 'OwningCompanyId',
      fields: new Map<string, unknown>([['OwningCompanyId', { type: 'char' }]]),
    },
  });

  expect(applyRepositoryCompanyLayer(deps, [] as any)).toEqual({
    Or: [
      ['OwningCompanyId', 'in', ['company_a', 'company_b']],
      ['OwningCompanyId', 'is', null],
    ],
  });
  expect(applyRepositoryDefaultCompanyIdOnCreate(deps, { Name: 'n1' } as any)).toEqual({
    Name: 'n1',
    OwningCompanyId: 'company_a',
  });

  const selected: string[][] = [];
  const query = {
    select(cols: string[]) {
      selected.push(cols);
      return this;
    },
    where() {
      return this;
    },
  };
  const writeDeps = {
    ...deps,
    db: {
      selectFrom() {
        return query;
      },
    },
    table: 'demo_alias',
    applySoftLayer: (condition: unknown) => condition,
    isEmptyCondition: () => true,
    convertCondition: () => ({}),
    execute: async () => [{ Id: 'row_1', OwningCompanyId: 'company_a' }],
  };
  await assertRepositoryCompanyWriteAccessForCondition(writeDeps as any, [] as any);
  expect(selected[0]).toEqual(['Id', 'OwningCompanyId']);
});

test('company scope rejects null ownership on private (notNull) models', () => {
  const { deps } = createCompanyDeps({
    meta: {
      fullModelName: 'demo.PrivateModel',
      modelName: 'PrivateModel',
      name: 'PrivateModel',
      companyField: 'CompanyId',
      fields: new Map<string, unknown>([['CompanyId', { type: 'char', column: { notNull: true } }]]),
    },
  });

  let createNull = '';
  try {
    applyRepositoryDefaultCompanyIdOnCreate(deps, { Name: 'n1', CompanyId: null } as any);
  } catch (error) {
    createNull = String((error as Error)?.message || error);
  }
  expect(createNull.includes('company_field_null_forbidden')).toBe(true);

  let createEmpty = '';
  try {
    applyRepositoryDefaultCompanyIdOnCreate(deps, { Name: 'n1', CompanyId: '  ' } as any);
  } catch (error) {
    createEmpty = String((error as Error)?.message || error);
  }
  expect(createEmpty.includes('company_field_null_forbidden')).toBe(true);

  let updateNull = '';
  try {
    applyRepositoryDefaultCompanyIdOnUpdate(deps, { CompanyId: null } as any);
  } catch (error) {
    updateNull = String((error as Error)?.message || error);
  }
  expect(updateNull.includes('company_field_null_forbidden')).toBe(true);

  // Shareable models (no column.notNull) still allow explicit null.
  const shareable = createCompanyDeps();
  expect(applyRepositoryDefaultCompanyIdOnCreate(shareable.deps, { Name: 'shared', CompanyId: null } as any)).toEqual({
    Name: 'shared',
    CompanyId: null,
  });
  expect(applyRepositoryDefaultCompanyIdOnUpdate(shareable.deps, { CompanyId: null } as any)).toEqual({ CompanyId: null });
});

test('company scope default company on create and write access guard enforce in-scope company ids', async () => {
  const { deps } = createCompanyDeps();
  expect(applyRepositoryDefaultCompanyIdOnCreate(deps, { Name: 'n1' } as any)).toEqual({
    Name: 'n1',
    CompanyId: 'company_a',
  });

  let createViolation = '';
  try {
    applyRepositoryDefaultCompanyIdOnCreate(deps, { CompanyId: 'company_x' } as any);
  } catch (error) {
    createViolation = String((error as Error)?.message || error);
  }
  expect(createViolation.includes('company_scope_violation')).toBe(true);

  const query = {
    whereFn: undefined as unknown,
    select() {
      return this;
    },
    where(fn: unknown) {
      this.whereFn = fn;
      return this;
    },
  };

  const ids = await assertRepositoryCompanyWriteAccessForCondition(
    {
      ...deps,
      db: {
        selectFrom() {
          return query;
        },
      },
      table: 'demo_table',
      applySoftLayer: (condition: unknown) => ({ And: [condition, ['DeletedAt', 'is', null]] }),
      isEmptyCondition: () => false,
      convertCondition: () => ({ kind: 'compiled' }),
      execute: async () => [{ Id: '1', CompanyId: 'company_a' }],
    } as any,
    ['Name', '=', 'n1'] as any
  );

  expect(ids).toEqual(['1']);

  let violation = '';
  try {
    await assertRepositoryCompanyWriteAccessForCondition(
      {
        ...deps,
        db: {
          selectFrom() {
            return query;
          },
        },
        table: 'demo_table',
        applySoftLayer: (condition: unknown) => condition,
        isEmptyCondition: () => true,
        convertCondition: () => ({ kind: 'compiled' }),
        execute: async () => [{ Id: '2', CompanyId: 'company_x' }],
      } as any,
      [] as any
    );
  } catch (error) {
    violation = String((error as Error)?.message || error);
  }
  expect(violation.includes('company_scope_violation')).toBe(true);
});

test('company scope normalizers prefer enabled list and trim/de-duplicate mixed raw values', () => {
  expect(
    normalizeRepositoryCompanyIds({
      enabledCompanyIds: [' company_a ', 'company_b', 'company_a', ''],
      activeCompanyId: 'company_c',
    })
  ).toEqual(['company_a', 'company_b']);

  expect(normalizeRepositoryCompanyIdForWrite({ enabledCompanyIds: [' company_only '], activeCompanyId: '' })).toBe('company_only');
});

test('company scope layer returns unchanged condition when skipped and supports empty-condition bootstrap', () => {
  const skipped = createCompanyDeps({ companyLayerSkipped: () => true });
  expect(applyRepositoryCompanyLayer(skipped.deps, ['Status', '=', 'ready'] as any)).toEqual(['Status', '=', 'ready']);
  expect(skipped.summaries.length).toBe(0);

  const active = createCompanyDeps();
  expect(applyRepositoryCompanyLayer(active.deps, [] as any)).toEqual({
    Or: [
      ['CompanyId', 'in', ['company_a', 'company_b']],
      ['CompanyId', 'is', null],
    ],
  });
  expect(active.summaries.length).toBe(1);
});

test('company scope permissionDenied metadata includes model and company key details by branch', () => {
  const missingField = createCompanyDeps({
    meta: {
      fullModelName: 'demo.Model',
      modelName: 'Model',
      companyField: 'CompanyId',
      fields: new Map<string, unknown>(),
    },
  });

  try {
    repositoryCompanyFieldEnabled(missingField.deps);
  } catch {
    // expected
  }
  expect(missingField.denied.length).toBe(1);
  expect(missingField.denied[0]).toEqual({
    code: 'company_field_missing',
    message: 'companyField model is missing ownership field',
    metadata: { model: 'demo.Model', companyField: 'CompanyId' },
  });

  const missingDefault = createCompanyDeps({
    ctx: { enabledCompanyIds: ['company_a', 'company_b'] },
  });
  let message = '';
  try {
    applyRepositoryDefaultCompanyIdOnCreate(missingDefault.deps, { Name: 'n1' } as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('company_scope_missing_default_company_id')).toBe(true);
  expect(missingDefault.denied.some(item => item.code === 'company_scope_missing_default_company_id')).toBe(true);
  const denied = missingDefault.denied.find(item => item.code === 'company_scope_missing_default_company_id');
  expect(denied?.metadata).toEqual({ model: 'demo.Model' });
});

test('company scope create/update keep in-scope CompanyId and bypass update checks for non-company-scoped model', () => {
  const scoped = createCompanyDeps();

  const createInput = { Name: 'n2', CompanyId: 'company_b' } as any;
  expect(applyRepositoryDefaultCompanyIdOnCreate(scoped.deps, createInput)).toBe(createInput);

  const updateNoCompany = { Name: 'next' } as any;
  expect(applyRepositoryDefaultCompanyIdOnUpdate(scoped.deps, updateNoCompany)).toBe(updateNoCompany);

  const updateWithCompany = { Name: 'next', CompanyId: 'company_a' } as any;
  expect(applyRepositoryDefaultCompanyIdOnUpdate(scoped.deps, updateWithCompany)).toBe(updateWithCompany);

  const nonScoped = createCompanyDeps({
    meta: {
      fullModelName: 'demo.Model',
      modelName: 'Model',
      name: 'Model',
      companyField: undefined,
      fields: new Map<string, unknown>(),
    },
  });
  const nonScopedVals = { Name: 'plain', CompanyId: 'company_outside' } as any;
  expect(applyRepositoryDefaultCompanyIdOnUpdate(nonScoped.deps, nonScopedVals)).toBe(nonScopedVals);
});

test('company scope normalizers ignore blank and nullable raw values across branches', () => {
  expect(normalizeRepositoryCompanyIds({ enabledCompanyIds: [null, '', '  ', 'company_a'] })).toEqual(['company_a']);
  expect(normalizeRepositoryCompanyIds({ enabledCompanyIds: '   ' })).toEqual([]);
  expect(normalizeRepositoryCompanyIds({ ActiveCompanyId: ' company_x ' })).toEqual(['company_x']);

  expect(normalizeRepositoryCompanyIdForWrite({ activeCompanyId: '   ', enabledCompanyIds: ['  '] })).toBe(undefined);
  expect(normalizeRepositoryCompanyIdForWrite({ ActiveCompanyId: 'company_b' })).toBe('company_b');
});

test('company scope validate helper returns early for null or blank company id', () => {
  const { deps } = createCompanyDeps();

  expect(() => validateRepositoryCompanyIdInScope(deps, null, ['company_a'])).not.toThrow();
  expect(() => validateRepositoryCompanyIdInScope(deps, undefined, ['company_a'])).not.toThrow();
  expect(() => validateRepositoryCompanyIdInScope(deps, '   ', ['company_a'])).not.toThrow();
  expect(() => validateRepositoryCompanyIdInScope(deps, 'company_x', ['company_a'])).toThrow('company_scope_violation');
});

test('company scope enabled gate handles grpc/env and companyField short-circuit branches', () => {
  const originalEnv = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
  try {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_GRPC_COMPANY_FILTER_ENABLED: false };
    expect(repositoryCompanyFieldEnabled(createCompanyDeps().deps)).toBe(false);

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_GRPC_COMPANY_FILTER_ENABLED: 'FALSE' };
    expect(repositoryCompanyFieldEnabled(createCompanyDeps().deps)).toBe(false);

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_GRPC_COMPANY_FILTER_ENABLED: 'true' };
    expect(repositoryCompanyFieldEnabled(createCompanyDeps({ companyLayerSkipped: () => true }).deps)).toBe(false);

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_GRPC_COMPANY_FILTER_ENABLED: true };
    expect(
      repositoryCompanyFieldEnabled(
        createCompanyDeps({
          meta: {
            fullModelName: 'demo.Model',
            modelName: 'Model',
            name: 'Model',
            companyField: undefined,
            fields: new Map<string, unknown>([['CompanyId', { type: 'char' }]]),
          },
        }).deps
      )
    ).toBe(false);
  } finally {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = originalEnv;
  }
});

test('company scope emits summary with fallback model and trimmed empty user id', () => {
  const { deps, summaries } = createCompanyDeps({
    meta: {
      fullModelName: '',
      modelName: 'ModelOnly',
      name: '',
      companyField: 'CompanyId',
      fields: new Map<string, unknown>([['CompanyId', { type: 'char' }]]),
    },
    userId: '   ',
  });

  const out = applyRepositoryCompanyLayer(deps, ['Status', '=', 'ready'] as any);
  expect(out).toEqual({
    And: [
      ['Status', '=', 'ready'],
      {
        Or: [
          ['CompanyId', 'in', ['company_a', 'company_b']],
          ['CompanyId', 'is', null],
        ],
      },
    ],
  });

  expect(summaries.length).toBe(1);
  expect(summaries[0]?.model).toBe('ModelOnly');
  expect(summaries[0]?.userId).toBe('');
});

test('company scope write access executes where callback with converted condition when filtered is non-empty', async () => {
  const { deps } = createCompanyDeps();
  let whereCompiled: any;

  const query = {
    select() {
      return this;
    },
    where(factory: any) {
      whereCompiled = factory({ eb: 'EB' });
      return this;
    },
  };

  const ids = await assertRepositoryCompanyWriteAccessForCondition(
    {
      ...deps,
      db: {
        selectFrom() {
          return query;
        },
      },
      table: 'demo_table',
      applySoftLayer: (condition: unknown) => ({ And: [condition, ['DeletedAt', 'is', null]] }),
      isEmptyCondition: () => false,
      convertCondition: (eb: unknown, condition: unknown, selfTable?: string) => ({ eb, condition, selfTable }),
      execute: async () => [],
    } as any,
    ['Name', '=', 'n1'] as any
  );

  expect(ids).toEqual([]);
  expect(whereCompiled).toEqual({
    eb: 'EB',
    condition: {
      And: [
        ['Name', '=', 'n1'],
        ['DeletedAt', 'is', null],
      ],
    },
    selfTable: 'demo_table',
  });
});

test('company scope create/write-access fallback paths cover model-name chain and row filtering', async () => {
  const { deps, denied } = createCompanyDeps({
    meta: {
      fullModelName: '',
      modelName: '',
      name: 'NameOnly',
      companyField: 'CompanyId',
      fields: new Map<string, unknown>([['CompanyId', { type: 'char' }]]),
    },
    ctx: { enabledCompanyIds: ['company_a', 'company_b'] },
  });

  let message = '';
  try {
    applyRepositoryDefaultCompanyIdOnCreate(deps, { Name: 'n3' } as any);
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message.includes('company_scope_missing_default_company_id')).toBe(true);
  expect(denied.some(item => item.metadata?.model === 'NameOnly')).toBe(true);

  const ids = await assertRepositoryCompanyWriteAccessForCondition(
    {
      ...deps,
      db: {
        selectFrom() {
          return {
            select() {
              return this;
            },
            where() {
              return this;
            },
          };
        },
      },
      table: 'demo_table',
      applySoftLayer: (condition: unknown) => condition,
      isEmptyCondition: () => true,
      convertCondition: () => ({ kind: 'compiled' }),
      execute: async () => [{ Id: '' }, { Id: '2', CompanyId: 'company_a' }, { CompanyId: null }],
    } as any,
    [] as any
  );

  expect(ids).toEqual(['2']);
});

test('company scope normalizer fallback keys and write-id selection cover remaining normalize branches', () => {
  expect(normalizeRepositoryCompanyIds({ EnabledCompanyIds: 'company_upper' })).toEqual(['company_upper']);
  expect(normalizeRepositoryCompanyIdForWrite({ ActiveCompanyId: null, enabledCompanyIds: ['a', 'b'] })).toBe(undefined);
  expect(normalizeRepositoryCompanyIdForWrite({ ActiveCompanyId: null, enabledCompanyIds: 'company_x' })).toBe(undefined);
  expect(normalizeRepositoryCompanyIdForWrite({ ActiveCompanyId: null, enabledCompanyIds: new Array(1) as any })).toBe(undefined);
});

test('company scope layer and enabled checks cover env default path and fallback metadata chains', () => {
  const originalEnv = (globalThis as any).__CHOYSUM_RUNTIME_ENV__;
  try {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_GRPC_COMPANY_FILTER_ENABLED: 1 };

    const missingFieldModelName = createCompanyDeps({
      meta: {
        fullModelName: '',
        modelName: 'ModelOnly',
        name: '',
        companyField: 'CompanyId',
        fields: new Map<string, unknown>(),
      },
    });
    expect(() => repositoryCompanyFieldEnabled(missingFieldModelName.deps)).toThrow('company_field_missing');

    const missingCtxNameOnly = createCompanyDeps({
      meta: {
        fullModelName: '',
        modelName: '',
        name: 'NameOnly',
        companyField: 'CompanyId',
        fields: new Map<string, unknown>([['CompanyId', { type: 'char' }]]),
      },
      ctx: {},
    });
    expect(() => repositoryCompanyFieldEnabled(missingCtxNameOnly.deps)).toThrow('company_scope_missing_ctx_company');

    const layerDisabled = createCompanyDeps();
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_GRPC_COMPANY_FILTER_ENABLED: false };
    expect(applyRepositoryCompanyLayer(layerDisabled.deps, ['Status', '=', 'ready'] as any)).toEqual(['Status', '=', 'ready']);

    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = { CHOYSUM_GRPC_COMPANY_FILTER_ENABLED: true };
    const nameFallback = createCompanyDeps({
      meta: {
        fullModelName: '',
        modelName: '',
        name: 'NameOnly',
        companyField: 'CompanyId',
        fields: new Map<string, unknown>([['CompanyId', { type: 'char' }]]),
      },
      userId: undefined,
    });
    applyRepositoryCompanyLayer(nameFallback.deps, ['Status', '=', 'ready'] as any);
    expect(nameFallback.summaries[0]?.model).toBe('NameOnly');
    expect(nameFallback.summaries[0]?.userId).toBe('');
  } finally {
    (globalThis as any).__CHOYSUM_RUNTIME_ENV__ = originalEnv;
  }
});

test('company scope create/update cover non-isolated create and in-scope branch with modelName fallback', () => {
  const nonScoped = createCompanyDeps({
    meta: {
      fullModelName: 'demo.Model',
      modelName: 'Model',
      name: 'Model',
      companyField: undefined,
      fields: new Map<string, unknown>(),
    },
  });
  const entity = { Name: 'raw' } as any;
  expect(applyRepositoryDefaultCompanyIdOnCreate(nonScoped.deps, entity)).toBe(entity);

  const modelNameFallback = createCompanyDeps({
    meta: {
      fullModelName: '',
      modelName: 'ModelOnly',
      name: '',
      companyField: 'CompanyId',
      fields: new Map<string, unknown>([['CompanyId', { type: 'char' }]]),
    },
  });
  const withCompany = { Name: 'n4', CompanyId: 'company_a' } as any;
  expect(applyRepositoryDefaultCompanyIdOnCreate(modelNameFallback.deps, withCompany)).toBe(withCompany);
});

test('company scope write-access covers companyField short-circuit and execute undefined fallback', async () => {
  const nonScoped = createCompanyDeps({
    meta: {
      fullModelName: 'demo.Model',
      modelName: 'Model',
      name: 'Model',
      companyField: undefined,
      fields: new Map<string, unknown>(),
    },
  });

  const ids = await assertRepositoryCompanyWriteAccessForCondition(
    {
      ...nonScoped.deps,
      db: {
        selectFrom() {
          throw new Error('should not query when not company-isolated');
        },
      },
      table: 'demo_table',
      applySoftLayer: (condition: unknown) => condition,
      isEmptyCondition: () => true,
      convertCondition: () => ({ kind: 'compiled' }),
      execute: async () => [],
    } as any,
    [] as any
  );
  expect(ids).toEqual([]);

  const scoped = createCompanyDeps();
  const idsWhenUndefined = await assertRepositoryCompanyWriteAccessForCondition(
    {
      ...scoped.deps,
      db: {
        selectFrom() {
          return {
            select() {
              return this;
            },
            where() {
              return this;
            },
          };
        },
      },
      table: 'demo_table',
      applySoftLayer: (condition: unknown) => condition,
      isEmptyCondition: () => false,
      convertCondition: () => ({ kind: 'compiled' }),
      execute: async () => undefined as any,
    } as any,
    ['Id', '=', '1'] as any
  );
  expect(idsWhenUndefined).toEqual([]);
});

test('company scope tail branches: normalize blank paths, metadata fallback chain, and create/update null payload guards', () => {
  expect(normalizeRepositoryCompanyIds({ activeCompanyId: '   ' })).toEqual([]);
  expect(normalizeRepositoryCompanyIdForWrite({ activeCompanyId: '   ', enabledCompanyIds: [' company_only '] })).toBe('company_only');

  const modelOnly = createCompanyDeps({
    meta: {
      fullModelName: '',
      modelName: 'ModelOnly',
      name: '',
      companyField: 'CompanyId',
      fields: new Map<string, unknown>([['CompanyId', { type: 'char' }]]),
    },
  });
  expect(() => validateRepositoryCompanyIdInScope(modelOnly.deps, 'company_x', ['company_a'])).toThrow('company_scope_violation');

  const nameOnly = createCompanyDeps({
    meta: {
      fullModelName: '',
      modelName: '',
      name: 'NameOnly',
      companyField: 'CompanyId',
      fields: new Map<string, unknown>([['CompanyId', { type: 'char' }]]),
    },
  });
  expect(() => validateRepositoryCompanyIdInScope(nameOnly.deps, 'company_x', ['company_a'])).toThrow('company_scope_violation');

  const missingFieldNameOnly = createCompanyDeps({
    meta: {
      fullModelName: '',
      modelName: '',
      name: 'NameOnly',
      companyField: 'CompanyId',
      fields: new Map<string, unknown>(),
    },
  });
  expect(() => repositoryCompanyFieldEnabled(missingFieldNameOnly.deps)).toThrow('company_field_missing');

  const createWithNullEntity = createCompanyDeps();
  expect(applyRepositoryDefaultCompanyIdOnCreate(createWithNullEntity.deps, null as any)).toEqual({ CompanyId: 'company_a' });

  const updateWithNullVals = createCompanyDeps();
  expect(applyRepositoryDefaultCompanyIdOnUpdate(updateWithNullVals.deps, null as any) as any).toBeNull();
});

test('company scope ownership helpers cover not-isolated nullability and name fallbacks', () => {
  const denied: DeniedCall[] = [];
  const permissionDenied = (code: string, message: string, metadata?: Record<string, string>) => {
    denied.push({ code, message, metadata });
    return new Error(`${code}:${message}`);
  };

  expect(() =>
    requireRepositoryOwnershipField(
      {
        name: 'OnlyName',
        companyField: undefined,
        fields: new Map([['CompanyId', {}]]),
      } as any,
      permissionDenied
    )
  ).toThrow('company_field_not_isolated');
  expect(denied[0]?.metadata).toEqual({ model: 'OnlyName' });

  expect(
    isRepositoryOwnershipFieldNotNull(
      { fields: { CompanyId: { column: { notNull: true } } } } as any,
      'CompanyId'
    )
  ).toBe(false);
  expect(
    isRepositoryOwnershipFieldNotNull(
      { fields: new Map([['CompanyId', { column: { notNull: true } }]]) } as any,
      'CompanyId'
    )
  ).toBe(true);
  expect(
    isRepositoryOwnershipFieldNotNull({ fields: new Map([['CompanyId', {}]]) } as any, 'CompanyId')
  ).toBe(false);

  const { deps } = createCompanyDeps({
    meta: {
      name: 'PrivateOnlyName',
      companyField: 'CompanyId',
      fields: new Map([['CompanyId', { type: 'char', column: { notNull: true } }]]),
    },
  });
  expect(() => validateRepositoryOwnershipNullability(deps, 'CompanyId', undefined)).toThrow(
    'company_field_null_forbidden'
  );
  expect(() => validateRepositoryOwnershipNullability(deps, 'CompanyId', '   ')).toThrow(
    'company_field_null_forbidden'
  );
  expect(() => validateRepositoryOwnershipNullability(deps, 'CompanyId', 'company_a')).not.toThrow();

  const shareable = createCompanyDeps();
  expect(() => validateRepositoryOwnershipNullability(shareable.deps, 'CompanyId', null)).not.toThrow();

  // Update with explicit undefined ownership on private model.
  let updateUndef = '';
  try {
    applyRepositoryDefaultCompanyIdOnUpdate(deps, { CompanyId: undefined } as any);
  } catch (error) {
    updateUndef = String((error as Error)?.message || error);
  }
  expect(updateUndef.includes('company_field_null_forbidden')).toBe(true);

  // Non-Map fields on requireRepositoryOwnershipField missing path.
  expect(() =>
    requireRepositoryOwnershipField(
      {
        modelName: 'M',
        companyField: 'CompanyId',
        fields: { CompanyId: {} } as any,
      } as any,
      permissionDenied
    )
  ).toThrow('company_field_missing');
});
