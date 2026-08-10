// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  applyRepositoryUpdatePostWrite,
  executeRepositoryUpdate,
  executeRepositoryUpdateRuntime,
  prepareRepositoryUpdatePayload,
  prepareRepositoryUpdateQuery,
  prepareRepositoryUpdateSanitizedPayload,
  resolveRepositoryUpdatePayloadTargets,
} from '../update';
import { withUser } from '../../../../runtime/context';

test('repository update payload target resolver returns undefined when no targets match', async () => {
  const calls: Array<Record<string, any>> = [];
  const targetIds = await resolveRepositoryUpdatePayloadTargets(
    {
      meta: { companyField: undefined } as any,
      async locateIdsForCondition(condition) {
        calls.push({ method: 'locate', condition });
        return [];
      },
      async assertCompanyWriteAccessForCondition(condition) {
        calls.push({ method: 'company', condition });
        return ['company_row'];
      },
      async assertRecordRuleAllTargetsAllowed(op, targetIdsValue) {
        calls.push({ method: 'recordRule', op, targetIds: targetIdsValue });
      },
    },
    ['Id', '=', '1'] as any
  );

  expect(targetIds).toBe(undefined);
  expect(calls).toEqual([{ method: 'locate', condition: ['Id', '=', '1'] }]);
});

test('repository update sanitized payload prepare validates current rows and returns encoded payload', async () => {
  const calls: Array<Record<string, any>> = [];
  const currentRowsQuery = { kind: 'current-rows-query' };

  const sanitized = await prepareRepositoryUpdateSanitizedPayload(
    {
      meta: { fields: new Map() } as any,
      table: 'demo_table',
      db: {
        selectFrom(table: string) {
          calls.push({ method: 'selectFrom', table });
          return {
            select(callback: (builder: any) => unknown) {
              const selections = callback({
                ref(ref: string) {
                  return {
                    as(alias: string) {
                      return { ref, alias };
                    },
                  };
                },
              });
              calls.push({ method: 'select', selections });
              return {
                where(whereCallback: ({ eb }: any) => unknown) {
                  calls.push({ method: 'where', result: whereCallback({ eb: 'EB' }) });
                  return currentRowsQuery;
                },
              };
            },
          };
        },
      },
      getScalarFields(meta) {
        calls.push({ method: 'scalarFields', meta });
        return [];
      },
      makeSelectCtx(builder, selfTable, curMeta) {
        calls.push({ method: 'selectCtx', builder, selfTable, curMeta });
        return { builder, selfTable, curMeta };
      },
      aliasSelection(selection, alias) {
        calls.push({ method: 'alias', selection, alias });
        return { selection, alias };
      },
      applySoftLayer(condition) {
        calls.push({ method: 'softLayer', condition });
        return condition;
      },
      isEmptyCondition(condition) {
        calls.push({ method: 'isEmpty', condition });
        return false;
      },
      convertCondition(eb, condition, selfTable) {
        calls.push({ method: 'convert', eb, condition, selfTable });
        return { eb, condition, selfTable };
      },
      async execute(query) {
        calls.push({ method: 'execute', query });
        return [{ Id: 'row_1', Name: 'current' }] as any;
      },
      decodeFromDb(row) {
        calls.push({ method: 'decode', row });
        return row;
      },
      async assertFieldRuleWriteAllowed(payload) {
        calls.push({ method: 'fieldRule', payload });
      },
      applyDefaultCompanyIdOnUpdate(vals) {
        calls.push({ method: 'defaultCompany', vals });
        return { ...vals, CompanyId: 'company_a' } as any;
      },
      async validateFields(input, mode, current) {
        calls.push({ method: 'validate', input, mode, current });
      },
      encodeForDb(input) {
        calls.push({ method: 'encode', input });
        return { ...input, Encoded: true } as any;
      },
    },
    { Name: 'demo' } as any,
    ['row_1']
  );

  expect((sanitized as any).Name).toBe('demo');
  expect((sanitized as any).CompanyId).toBe('company_a');
  expect((sanitized as any).Encoded).toBe(true);
  expect((sanitized as any).UpdatedAt instanceof Date).toBe(true);
  expect(calls.filter(c => c.method === 'fieldRule')).toEqual([{ method: 'fieldRule', payload: { Name: 'demo' } }]);
  expect(calls.filter(c => c.method === 'defaultCompany')).toEqual([
    { method: 'defaultCompany', vals: { Name: 'demo' } },
  ]);
  const validate = calls.find(c => c.method === 'validate');
  expect(validate?.mode).toBe('update');
  expect(validate?.current).toEqual({ Id: 'row_1', Name: 'current' });
  expect(validate?.input.Name).toBe('demo');
  expect(validate?.input.CompanyId).toBe('company_a');
  expect(validate?.input.UpdatedAt instanceof Date).toBe(true);
  const encode = calls.find(c => c.method === 'encode');
  expect(encode?.input.Name).toBe('demo');
  expect(encode?.input.CompanyId).toBe('company_a');
  expect(encode?.input.UpdatedAt instanceof Date).toBe(true);
});

test('repository update sanitized payload stamps monetary digits and rejects mismatched multi-row scales', async () => {
  const meta = {
    fields: new Map([
      ['CurrencyId', { type: 'ManyToOneRef', column: { name: 'CurrencyId' } }],
      ['Amount', { type: 'monetary', name: 'Amount', column: { name: 'Amount', currencyField: 'CurrencyId' } }],
    ]),
  } as any;

  const makeDeps = (rows: any[]) => ({
    meta,
    table: 'demo_table',
    db: {
      selectFrom() {
        return {
          select() {
            return {
              where() {
                return { kind: 'q' };
              },
            };
          },
        };
      },
    },
    getScalarFields() {
      return [];
    },
    makeSelectCtx() {
      return {};
    },
    aliasSelection(selection: unknown, alias: string) {
      return { selection, alias };
    },
    applySoftLayer(condition: any) {
      return condition;
    },
    isEmptyCondition() {
      return false;
    },
    convertCondition() {
      return {};
    },
    async execute() {
      return rows;
    },
    decodeFromDb(row: any) {
      return row;
    },
    async assertFieldRuleWriteAllowed() {},
    applyDefaultCompanyIdOnUpdate(vals: any) {
      return vals;
    },
    async validateFields() {},
    encodeForDb(input: any) {
      return input;
    },
  });

  const single = await prepareRepositoryUpdateSanitizedPayload(
    makeDeps([{ Id: 'r1', CurrencyId: { Id: 'C1', DecimalDigits: 0 } }]) as any,
    { Amount: '12.6' } as any,
    ['r1']
  );
  expect((single as any).$dec$Amount__scale).toBe(0);

  let err: unknown;
  try {
    await prepareRepositoryUpdateSanitizedPayload(
      makeDeps([
        { Id: 'r1', CurrencyId: { Id: 'C1', DecimalDigits: 0 } },
        { Id: 'r2', CurrencyId: { Id: 'C2', DecimalDigits: 2 } },
      ]) as any,
      { Amount: '12.6' } as any,
      ['r1', 'r2']
    );
  } catch (e) {
    err = e;
  }
  expect(String((err as Error)?.message || err)).toMatch(/same currency decimal digits/);

  // Missing current row → current: null via ?? while inline currency still stamps.
  const missingCurrent = await prepareRepositoryUpdateSanitizedPayload(
    makeDeps([{ Id: 'r1', CurrencyId: { Id: 'C1', DecimalDigits: 2 } }]) as any,
    { Amount: '1.20', CurrencyId: { Id: 'C1', DecimalDigits: 2 } } as any,
    ['r1', 'ghost']
  );
  expect((missingCurrent as any).$dec$Amount__scale).toBe(2);
});

test('repository update sanitized payload rejects bulk translated field writes', async () => {
  let err: unknown;
  try {
    await prepareRepositoryUpdateSanitizedPayload(
      {
        meta: {
          fields: new Map([['Name', { translate: true, column: { name: 'Name' } }]]),
        } as any,
        table: 'demo_table',
        db: {
          selectFrom() {
            return {
              select() {
                return {
                  where() {
                    return { kind: 'q' };
                  },
                };
              },
            };
          },
        },
        getScalarFields() {
          return [];
        },
        makeSelectCtx() {
          return {};
        },
        aliasSelection(selection, alias) {
          return { selection, alias };
        },
        applySoftLayer(condition) {
          return condition;
        },
        isEmptyCondition() {
          return false;
        },
        convertCondition() {
          return true;
        },
        async execute() {
          return [
            { Id: 'row_1', Name: '{"en_US":"A"}' },
            { Id: 'row_2', Name: '{"en_US":"B"}' },
          ] as any;
        },
        decodeFromDb(row) {
          return row;
        },
        async assertFieldRuleWriteAllowed() {},
        applyDefaultCompanyIdOnUpdate(vals) {
          return vals as any;
        },
        async validateFields() {},
        encodeForDb(input) {
          return input as any;
        },
      },
      { Name: 'bulk' } as any,
      ['row_1', 'row_2']
    );
  } catch (e) {
    err = e;
  }
  expect(String((err as Error)?.message || err)).toMatch(/one record at a time/);
});

test('repository update sanitized payload rejects bulk companyDependent field writes', async () => {
  let err: unknown;
  try {
    await prepareRepositoryUpdateSanitizedPayload(
      {
        meta: {
          fields: new Map([['Cost', { companyDependent: true, column: { name: 'Cost' } }]]),
        } as any,
        table: 'demo_table',
        db: {
          selectFrom() {
            return {
              select() {
                return {
                  where() {
                    return { kind: 'q' };
                  },
                };
              },
            };
          },
        },
        getScalarFields() {
          return [];
        },
        makeSelectCtx() {
          return {};
        },
        aliasSelection(selection, alias) {
          return { selection, alias };
        },
        applySoftLayer(condition) {
          return condition;
        },
        isEmptyCondition() {
          return false;
        },
        convertCondition() {
          return true;
        },
        async execute() {
          return [
            { Id: 'row_1', Cost: '{"C1":1}' },
            { Id: 'row_2', Cost: '{"C1":2}' },
          ] as any;
        },
        decodeFromDb(row) {
          return row;
        },
        async assertFieldRuleWriteAllowed() {},
        applyDefaultCompanyIdOnUpdate(vals) {
          return vals as any;
        },
        async validateFields() {},
        encodeForDb(input) {
          return input as any;
        },
      },
      { Cost: 9 } as any,
      ['row_1', 'row_2']
    );
  } catch (e) {
    err = e;
  }
  expect(String((err as Error)?.message || err)).toMatch(/company-dependent fields on multiple rows/);
});

test('repository update payload prepare resolves targets validates current rows and returns sanitized payload', async () => {
  const calls: Array<Record<string, any>> = [];
  const currentRowsQuery = { kind: 'current-rows-query' };

  const prepared = await prepareRepositoryUpdatePayload(
    {
      meta: { companyField: undefined, fields: new Map() } as any,
      async locateIdsForCondition(condition) {
        calls.push({ method: 'locate', condition });
        return ['row_1'];
      },
      async assertCompanyWriteAccessForCondition(condition) {
        calls.push({ method: 'company', condition });
        return ['company_row'];
      },
      async assertRecordRuleAllTargetsAllowed(op, targetIds) {
        calls.push({ method: 'recordRule', op, targetIds });
      },
      table: 'demo_table',
      db: {
        selectFrom(table: string) {
          calls.push({ method: 'selectFrom', table });
          return {
            select(callback: (builder: any) => unknown) {
              const selections = callback({
                ref(ref: string) {
                  return {
                    as(alias: string) {
                      return { ref, alias };
                    },
                  };
                },
              });
              calls.push({ method: 'select', selections });
              return {
                where(whereCallback: ({ eb }: any) => unknown) {
                  calls.push({ method: 'where', result: whereCallback({ eb: 'EB' }) });
                  return currentRowsQuery;
                },
              };
            },
          };
        },
        updateTable(table: string) {
          calls.push({ method: 'updateTable', table });
          return {
            set(input: unknown) {
              calls.push({ method: 'set', input });
              return { kind: 'update-query' };
            },
          };
        },
      },
      getScalarFields(meta) {
        calls.push({ method: 'scalarFields', meta });
        return [];
      },
      makeSelectCtx(builder, selfTable, curMeta) {
        calls.push({ method: 'selectCtx', builder, selfTable, curMeta });
        return { builder, selfTable, curMeta };
      },
      aliasSelection(selection, alias) {
        calls.push({ method: 'alias', selection, alias });
        return { selection, alias };
      },
      applySoftLayer(condition) {
        calls.push({ method: 'softLayer', condition });
        return condition;
      },
      isEmptyCondition(condition) {
        calls.push({ method: 'isEmpty', condition });
        return false;
      },
      convertCondition(eb, condition, selfTable) {
        calls.push({ method: 'convert', eb, condition, selfTable });
        return { eb, condition, selfTable };
      },
      async execute(query) {
        calls.push({ method: 'execute', query });
        return [{ Id: 'row_1', Name: 'current' }] as any;
      },
      decodeFromDb(row) {
        calls.push({ method: 'decode', row });
        return row;
      },
      async assertFieldRuleWriteAllowed(payload) {
        calls.push({ method: 'fieldRule', payload });
      },
      applyDefaultCompanyIdOnUpdate(vals) {
        calls.push({ method: 'defaultCompany', vals });
        return { ...vals, CompanyId: 'company_a' } as any;
      },
      async validateFields(input, mode, current) {
        calls.push({ method: 'validate', input, mode, current });
      },
      encodeForDb(input) {
        calls.push({ method: 'encode', input });
        return { ...input, Encoded: true } as any;
      },
      async applyRecordRuleToCondition(condition, op) {
        calls.push({ method: 'applyRecordRule', condition, op });
        return condition;
      },
      applyDefaultLayers(condition) {
        calls.push({ method: 'defaultLayers', condition });
        return condition;
      },
    },
    { Name: 'demo' } as any,
    ['Id', '=', '1'] as any
  );

  expect(prepared?.targetIds).toEqual(['row_1']);
  expect((prepared?.sanitized as any).Name).toBe('demo');
  expect((prepared?.sanitized as any).CompanyId).toBe('company_a');
  expect((prepared?.sanitized as any).Encoded).toBe(true);
  expect((prepared?.sanitized as any).UpdatedAt instanceof Date).toBe(true);
  expect(calls.filter(c => c.method === 'locate')).toEqual([{ method: 'locate', condition: ['Id', '=', '1'] }]);
  expect(calls.filter(c => c.method === 'recordRule')).toEqual([
    { method: 'recordRule', op: 'write', targetIds: ['row_1'] },
  ]);
  expect(calls.filter(c => c.method === 'fieldRule')).toEqual([{ method: 'fieldRule', payload: { Name: 'demo' } }]);
  expect(calls.filter(c => c.method === 'defaultCompany')).toEqual([
    { method: 'defaultCompany', vals: { Name: 'demo' } },
  ]);
  const validate = calls.find(c => c.method === 'validate');
  expect(validate?.mode).toBe('update');
  expect(validate?.input.Name).toBe('demo');
  expect(validate?.input.CompanyId).toBe('company_a');
  expect(validate?.input.UpdatedAt instanceof Date).toBe(true);
  const encode = calls.find(c => c.method === 'encode');
  expect(encode?.input.Name).toBe('demo');
  expect(encode?.input.CompanyId).toBe('company_a');
  expect(encode?.input.UpdatedAt instanceof Date).toBe(true);
});

test('repository update query prepare builds conditioned query from sanitized payload', async () => {
  const calls: Array<Record<string, any>> = [];

  const prepared = await prepareRepositoryUpdateQuery(
    {
      db: {
        updateTable(table: string) {
          calls.push({ method: 'updateTable', table });
          return {
            set(input: unknown) {
              calls.push({ method: 'set', input });
              return {
                where(callback: ({ eb }: any) => unknown) {
                  calls.push({ method: 'where', result: callback({ eb: 'EB' }) });
                  return { kind: 'conditioned-query' };
                },
              };
            },
          };
        },
      },
      table: 'demo_table',
      async applyRecordRuleToCondition(condition, op) {
        calls.push({ method: 'applyRecordRule', condition, op });
        return condition;
      },
      applyDefaultLayers(condition) {
        calls.push({ method: 'defaultLayers', condition });
        return condition;
      },
      isEmptyCondition(condition) {
        calls.push({ method: 'isEmpty', condition });
        return false;
      },
      convertCondition(eb, condition, selfTable) {
        calls.push({ method: 'convert', eb, condition, selfTable });
        return { eb, condition, selfTable };
      },
    },
    { Name: 'demo', Encoded: true } as any,
    ['Id', '=', '1'] as any
  );

  expect(prepared).toEqual({ query: { kind: 'conditioned-query' } });
  expect(calls).toEqual([
    { method: 'updateTable', table: 'demo_table' },
    { method: 'set', input: { Name: 'demo', Encoded: true } },
    { method: 'applyRecordRule', condition: ['Id', '=', '1'], op: 'write' },
    { method: 'defaultLayers', condition: ['Id', '=', '1'] },
    { method: 'isEmpty', condition: ['Id', '=', '1'] },
    { method: 'convert', eb: 'EB', condition: ['Id', '=', '1'], selfTable: 'demo_table' },
    { method: 'where', result: { eb: 'EB', condition: ['Id', '=', '1'], selfTable: 'demo_table' } },
  ]);
});

test('repository update runtime executes query and returns rows', async () => {
  const calls: Array<Record<string, any>> = [];
  const query = { kind: 'update-query' };

  const rows = await executeRepositoryUpdateRuntime(
    {
      async execute(input) {
        calls.push({ method: 'execute', input });
        return [{ numUpdatedRows: 1 }] as any;
      },
    },
    query
  );

  expect(rows).toEqual([{ numUpdatedRows: 1 }]);
  expect(calls).toEqual([{ method: 'execute', input: query }]);
});

test('repository update post-write invalidates cache when rows are returned', () => {
  const calls: Array<Record<string, any>> = [];

  const rows = applyRepositoryUpdatePostWrite(
    {
      invalidateCache() {
        calls.push({ method: 'invalidate' });
      },
    },
    [{ numUpdatedRows: 1 }] as any
  );

  expect(rows).toEqual([{ numUpdatedRows: 1 }]);
  expect(calls).toEqual([{ method: 'invalidate' }]);
});

test('repository update post-write skips cache invalidation when no rows are returned', () => {
  const calls: Array<Record<string, any>> = [];

  const rows = applyRepositoryUpdatePostWrite(
    {
      invalidateCache() {
        calls.push({ method: 'invalidate' });
      },
    },
    [] as any
  );

  expect(rows).toEqual([]);
  expect(calls).toEqual([]);
});

test('repository update runtime keeps empty result unchanged', async () => {
  const calls: Array<Record<string, any>> = [];

  const rows = await executeRepositoryUpdateRuntime(
    {
      async execute(input) {
        calls.push({ method: 'execute', input });
        return [] as any;
      },
    },
    { kind: 'update-query' }
  );

  expect(rows).toEqual([]);
  expect(calls).toEqual([{ method: 'execute', input: { kind: 'update-query' } }]);
});

test('repository update runtime normalizes undefined execute result to empty list', async () => {
  const calls: Array<Record<string, any>> = [];

  const rows = await executeRepositoryUpdateRuntime(
    {
      async execute(input) {
        calls.push({ method: 'execute', input });
        return undefined as any;
      },
    },
    { kind: 'update-query-undefined' }
  );

  expect(rows).toEqual([]);
  expect(calls).toEqual([{ method: 'execute', input: { kind: 'update-query-undefined' } }]);
});

test('repository update post-write normalizes undefined rows and skips invalidation', () => {
  const calls: Array<Record<string, any>> = [];

  const rows = applyRepositoryUpdatePostWrite(
    {
      invalidateCache() {
        calls.push({ method: 'invalidate' });
      },
    },
    undefined as any
  );

  expect(rows).toEqual([]);
  expect(calls).toEqual([]);
});

test('repository update executor invokes persist recompute hook with prepared payload', async () => {
  const calls: Array<Record<string, any>> = [];
  const currentRowsQuery = { kind: 'current-rows-query' };

  const rows = await executeRepositoryUpdate(
    {
      meta: { fields: new Map() } as any,
      table: 'demo_table',
      db: {
        selectFrom() {
          return {
            select(callback: (builder: any) => unknown) {
              callback({
                ref(ref: string) {
                  return {
                    as(alias: string) {
                      return { ref, alias };
                    },
                  };
                },
              });
              return {
                where(whereCallback: ({ eb }: any) => unknown) {
                  whereCallback({ eb: 'EB' });
                  return currentRowsQuery;
                },
              };
            },
          };
        },
        updateTable() {
          return {
            set() {
              return {
                where(whereCallback: ({ eb }: any) => unknown) {
                  whereCallback({ eb: 'EB' });
                  return { kind: 'update-query' };
                },
              };
            },
          };
        },
      },
      getScalarFields() {
        return [];
      },
      makeSelectCtx() {
        return {};
      },
      aliasSelection(selection: any) {
        return selection;
      },
      applySoftLayer(condition) {
        return condition;
      },
      isEmptyCondition() {
        return false;
      },
      convertCondition(_eb, condition) {
        return condition;
      },
      async execute(query) {
        if (query === currentRowsQuery) {
          return [{ Id: 'row_1', Name: 'old' }] as any;
        }
        return [{ numUpdatedRows: 1 }] as any;
      },
      decodeFromDb(row) {
        return row;
      },
      async locateIdsForCondition() {
        return ['row_1'];
      },
      async assertCompanyWriteAccessForCondition() {
        return ['row_1'];
      },
      async assertRecordRuleAllTargetsAllowed() {},
      async assertFieldRuleWriteAllowed() {},
      applyDefaultCompanyIdOnUpdate(vals) {
        return vals;
      },
      async validateFields() {},
      encodeForDb(input) {
        return input;
      },
      async applyRecordRuleToCondition(condition) {
        return condition;
      },
      applyDefaultLayers(condition) {
        return condition;
      },
      invalidateCache() {
        calls.push({ method: 'invalidate' });
      },
      async recomputePersistForUpdate(payload) {
        calls.push({ method: 'recompute', payload });
      },
    },
    { Name: 'new' } as any,
    ['Id', '=', 'row_1'] as any
  );

  expect(rows).toEqual([{ numUpdatedRows: 1 }]);
  expect(calls).toHaveLength(2);
  expect(calls[0].method).toBe('recompute');
  expect(calls[0].payload.targetIds).toEqual(['row_1']);
  expect(calls[0].payload.condition).toEqual(['Id', '=', 'row_1']);
  expect(calls[0].payload.rows).toEqual([{ numUpdatedRows: 1 }]);
  expect(calls[0].payload.sanitized.Name).toBe('new');
  expect(calls[0].payload.sanitized.UpdatedAt instanceof Date).toBe(true);
  expect(calls[1]).toEqual({ method: 'invalidate' });
});
function updateSanitizeDeps(overrides?: Record<string, unknown>) {
  return {
    meta: { fields: new Map() } as any,
    table: 'demo_table',
    db: {
      selectFrom() {
        return {
          select() {
            return {
              where() {
                return { kind: 'current-rows-query' };
              },
            };
          },
        };
      },
    },
    getScalarFields() {
      return [];
    },
    makeSelectCtx() {
      return {};
    },
    aliasSelection(selection: unknown, alias: string) {
      return { selection, alias };
    },
    applySoftLayer(condition: any) {
      return condition;
    },
    isEmptyCondition() {
      return false;
    },
    convertCondition() {
      return {};
    },
    async execute() {
      return [{ Id: 'row_1', Name: 'current' }] as any;
    },
    decodeFromDb(row: any) {
      return row;
    },
    async assertFieldRuleWriteAllowed() {},
    applyDefaultCompanyIdOnUpdate(vals: any) {
      return vals;
    },
    async validateFields() {},
    encodeForDb(input: any) {
      return input;
    },
    ...overrides,
  };
}

test('repository update prepare stamps UpdatedAt/UpdatedUid and strips CreatedUid', async () => {
  let validated: any;
  const sanitized = await withUser('U-UPD', async () =>
    prepareRepositoryUpdateSanitizedPayload(
      updateSanitizeDeps({
        async validateFields(input: any) {
          validated = input;
        },
      }) as any,
      { Name: 'demo', CreatedUid: 'hijack', DeletedUid: 'hijack-del' } as any,
      ['row_1']
    )
  );

  expect((sanitized as any).UpdatedAt instanceof Date).toBe(true);
  expect((sanitized as any).UpdatedUid).toBe('U-UPD');
  expect((sanitized as any).CreatedUid).toBeUndefined();
  expect((sanitized as any).DeletedUid).toBeUndefined();
  expect(validated.UpdatedUid).toBe('U-UPD');
  expect(validated.CreatedUid).toBeUndefined();
});

test('repository update prepare clears DeletedUid on restore and leaves uid empty without actor', async () => {
  const restored = await withUser('U-REST', async () =>
    prepareRepositoryUpdateSanitizedPayload(
      updateSanitizeDeps() as any,
      { DeletedAt: null, DeletedUid: 'old-del' } as any,
      ['row_1']
    )
  );
  expect((restored as any).DeletedUid).toBeNull();
  expect((restored as any).UpdatedUid).toBe('U-REST');

  const noActor = await prepareRepositoryUpdateSanitizedPayload(
    updateSanitizeDeps() as any,
    { Name: 'x', CreatedUid: 'hijack' } as any,
    ['row_1']
  );
  expect((noActor as any).CreatedUid).toBeUndefined();
  expect((noActor as any).UpdatedUid).toBeUndefined();
  expect((noActor as any).UpdatedAt instanceof Date).toBe(true);
});
