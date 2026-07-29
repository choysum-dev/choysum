// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { applyRepositoryUpdateCondition, loadRepositoryUpdateValidationCurrentRows, resolveRepositoryUpdateTargetIds } from '../update_helpers';

test('repository update write helpers resolve target ids via company access for company scoped models', async () => {
  const calls: Array<Record<string, any>> = [];
  const ids = await resolveRepositoryUpdateTargetIds(
    {
      meta: { companyField: 'CompanyId', fields: new Map([['CompanyId', {}]]) } as any,
      async locateIdsForCondition(condition) {
        calls.push({ method: 'locate', condition });
        return ['locate_1'];
      },
      async assertCompanyWriteAccessForCondition(condition) {
        calls.push({ method: 'company', condition });
        return ['company_1'];
      },
      async assertRecordRuleAllTargetsAllowed(op, targetIds) {
        calls.push({ method: 'recordRule', op, targetIds });
      },
    },
    ['Id', '=', '1'] as any
  );

  expect(ids).toEqual(['company_1']);
  expect(calls).toEqual([
    { method: 'company', condition: ['Id', '=', '1'] },
    { method: 'recordRule', op: 'write', targetIds: ['company_1'] },
  ]);
});

test('repository update write helpers resolve target ids via locate for non-company scoped models', async () => {
  const calls: Array<Record<string, any>> = [];
  const ids = await resolveRepositoryUpdateTargetIds(
    {
      meta: { companyField: undefined } as any,
      async locateIdsForCondition(condition) {
        calls.push({ method: 'locate', condition });
        return ['locate_1'];
      },
      async assertCompanyWriteAccessForCondition(condition) {
        calls.push({ method: 'company', condition });
        return ['company_1'];
      },
      async assertRecordRuleAllTargetsAllowed(op, targetIds) {
        calls.push({ method: 'recordRule', op, targetIds });
      },
    },
    ['Id', '=', '2'] as any
  );

  expect(ids).toEqual(['locate_1']);
  expect(calls).toEqual([
    { method: 'locate', condition: ['Id', '=', '2'] },
    { method: 'recordRule', op: 'write', targetIds: ['locate_1'] },
  ]);
});

test('repository update write helpers skip record rule assertion when no targets resolve', async () => {
  const calls: Array<Record<string, any>> = [];
  const ids = await resolveRepositoryUpdateTargetIds(
    {
      meta: { companyField: undefined } as any,
      async locateIdsForCondition(condition) {
        calls.push({ method: 'locate', condition });
        return [];
      },
      async assertCompanyWriteAccessForCondition(condition) {
        calls.push({ method: 'company', condition });
        return ['company_1'];
      },
      async assertRecordRuleAllTargetsAllowed(op, targetIds) {
        calls.push({ method: 'recordRule', op, targetIds });
      },
    },
    ['Id', '=', '3'] as any
  );

  expect(ids).toEqual([]);
  expect(calls).toEqual([{ method: 'locate', condition: ['Id', '=', '3'] }]);
});

test('repository update write helpers apply record rule and default layers to update query', async () => {
  const calls: Array<Record<string, any>> = [];
  const query = {
    where(callback: ({ eb }: any) => unknown) {
      calls.push({ method: 'where', result: callback({ eb: 'EB' }) });
      return { tagged: 'where-query' };
    },
  };

  const result = await applyRepositoryUpdateCondition(
    query as any,
    {
      table: 'demo_table',
      async applyRecordRuleToCondition(condition, op) {
        calls.push({ method: 'recordRule', condition, op });
        return { And: [condition, ['CompanyId', '=', 'company_a'] as any] } as any;
      },
      applyDefaultLayers(condition) {
        calls.push({ method: 'defaultLayers', condition });
        return { And: [condition, ['DeletedAt', 'is', null] as any] } as any;
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
    ['Id', '=', '4'] as any
  );

  expect(result).toEqual({ tagged: 'where-query' });
  expect(calls).toEqual([
    { method: 'recordRule', condition: ['Id', '=', '4'], op: 'write' },
    {
      method: 'defaultLayers',
      condition: {
        And: [
          ['Id', '=', '4'],
          ['CompanyId', '=', 'company_a'],
        ],
      },
    },
    {
      method: 'isEmpty',
      condition: {
        And: [
          {
            And: [
              ['Id', '=', '4'],
              ['CompanyId', '=', 'company_a'],
            ],
          },
          ['DeletedAt', 'is', null],
        ],
      },
    },
    {
      method: 'convert',
      eb: 'EB',
      condition: {
        And: [
          {
            And: [
              ['Id', '=', '4'],
              ['CompanyId', '=', 'company_a'],
            ],
          },
          ['DeletedAt', 'is', null],
        ],
      },
      selfTable: 'demo_table',
    },
    {
      method: 'where',
      result: {
        eb: 'EB',
        condition: {
          And: [
            {
              And: [
                ['Id', '=', '4'],
                ['CompanyId', '=', 'company_a'],
              ],
            },
            ['DeletedAt', 'is', null],
          ],
        },
        selfTable: 'demo_table',
      },
    },
  ]);
});

test('repository update write helpers leave query unchanged when filtered condition is empty', async () => {
  const calls: Array<Record<string, any>> = [];
  const query = {
    tagged: 'original-query',
    where() {
      calls.push({ method: 'where' });
      return { tagged: 'where-query' };
    },
  };

  const result = await applyRepositoryUpdateCondition(
    query as any,
    {
      table: 'demo_table',
      async applyRecordRuleToCondition(condition, op) {
        calls.push({ method: 'recordRule', condition, op });
        return condition;
      },
      applyDefaultLayers(condition) {
        calls.push({ method: 'defaultLayers', condition });
        return condition;
      },
      isEmptyCondition(condition) {
        calls.push({ method: 'isEmpty', condition });
        return true;
      },
      convertCondition(eb, condition, selfTable) {
        calls.push({ method: 'convert', eb, condition, selfTable });
        return { eb, condition, selfTable };
      },
    },
    ['Id', '=', '5'] as any
  );

  expect(result).toBe(query);
  expect(calls).toEqual([
    { method: 'recordRule', condition: ['Id', '=', '5'], op: 'write' },
    { method: 'defaultLayers', condition: ['Id', '=', '5'] },
    { method: 'isEmpty', condition: ['Id', '=', '5'] },
  ]);
});

test('repository update write helpers load validation current rows with decoded map and selection aliases', async () => {
  class DemoModel {
    sqlName() {
      return { kind: 'expr', ctx: (this as any).$sql };
    }
  }

  const calls: Array<Record<string, any>> = [];
  const query = {
    where(callback: ({ eb }: any) => unknown) {
      calls.push({ method: 'where', result: callback({ eb: 'EB' }) });
      return query;
    },
  };

  const selections: any[] = [];
  const result = await loadRepositoryUpdateValidationCurrentRows(
    {
      db: {
        selectFrom(table: string) {
          calls.push({ method: 'selectFrom', table });
          return {
            select(callback: (builder: any) => any[]) {
              const builder = {
                ref(value: string) {
                  return {
                    as(alias: string) {
                      return { kind: 'ref', value, alias };
                    },
                  };
                },
              };
              const builtSelections = callback(builder);
              selections.push(...builtSelections);
              calls.push({ method: 'select', selections: builtSelections });
              return query;
            },
          };
        },
      },
      table: 'demo_table',
      meta: {
        type: DemoModel,
        fields: new Map([
          ['Id', {}],
          ['Name', {}],
        ]),
        sqlComputeHandlers: new Map([['Name', { field: 'Name', method: 'sqlName' }]]),
      } as any,
      getScalarFields() {
        calls.push({ method: 'scalarFields' });
        return ['Name'];
      },
      makeSelectCtx(builder, selfTable, curMeta) {
        calls.push({ method: 'selectCtx', builder: !!builder, selfTable, curMeta: !!curMeta });
        return { tag: 'ctx', selfTable };
      },
      aliasSelection(selection, alias) {
        calls.push({ method: 'alias', selection, alias });
        return { kind: 'alias', selection, alias };
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
      async execute(input) {
        calls.push({ method: 'execute', input });
        return [
          { Id: ' row_1 ', Name: 'demo' },
          { Id: '', Name: 'skip' },
        ] as any;
      },
      decodeFromDb(row) {
        calls.push({ method: 'decode', row });
        return row;
      },
    },
    [' row_1 ', '', 'row_1']
  );

  expect(Array.from(result.entries())).toEqual([['row_1', { Id: ' row_1 ', Name: 'demo' }]]);
  expect(selections).toEqual([
    { kind: 'ref', value: 'demo_table.Id', alias: 'Id' },
    { kind: 'alias', selection: { kind: 'expr', ctx: { tag: 'ctx', selfTable: 'demo_table' } }, alias: 'Name' },
  ]);
});

test('repository update write helpers skip validation row load when normalized ids are empty', async () => {
  const calls: Array<Record<string, any>> = [];
  const result = await loadRepositoryUpdateValidationCurrentRows(
    {
      db: {
        selectFrom(table: string) {
          calls.push({ method: 'selectFrom', table });
          return {
            select() {
              calls.push({ method: 'select' });
              return {};
            },
          };
        },
      },
      table: 'demo_table',
      meta: { fields: new Map() } as any,
      getScalarFields() {
        calls.push({ method: 'scalarFields' });
        return ['Name'];
      },
      makeSelectCtx() {
        calls.push({ method: 'selectCtx' });
        return {};
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
        return true;
      },
      convertCondition(eb, condition, selfTable) {
        calls.push({ method: 'convert', eb, condition, selfTable });
        return { eb, condition, selfTable };
      },
      async execute(input) {
        calls.push({ method: 'execute', input });
        return [];
      },
      decodeFromDb(row) {
        calls.push({ method: 'decode', row });
        return row;
      },
    },
    [' ', '']
  );

  expect(result.size).toBe(0);
  expect(calls).toEqual([]);
});

test('repository update write helpers treat undefined ids as empty validation set', async () => {
  const calls: Array<Record<string, any>> = [];
  const result = await loadRepositoryUpdateValidationCurrentRows(
    {
      db: {
        selectFrom(table: string) {
          calls.push({ method: 'selectFrom', table });
          return {
            select() {
              calls.push({ method: 'select' });
              return {};
            },
          };
        },
      },
      table: 'demo_table',
      meta: { fields: new Map() } as any,
      getScalarFields() {
        calls.push({ method: 'scalarFields' });
        return ['Name'];
      },
      makeSelectCtx() {
        calls.push({ method: 'selectCtx' });
        return {};
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
        return true;
      },
      convertCondition(eb, condition, selfTable) {
        calls.push({ method: 'convert', eb, condition, selfTable });
        return { eb, condition, selfTable };
      },
      async execute(input) {
        calls.push({ method: 'execute', input });
        return [];
      },
      decodeFromDb(row) {
        calls.push({ method: 'decode', row });
        return row;
      },
    },
    undefined as any
  );

  expect(result.size).toBe(0);
  expect(calls).toEqual([]);
});

test('repository update write helpers drop decoded rows when Id is missing after trim normalization', async () => {
  const calls: Array<Record<string, any>> = [];
  const query = {
    where(callback: ({ eb }: any) => unknown) {
      calls.push({ method: 'where', result: callback({ eb: 'EB' }) });
      return query;
    },
  };

  const result = await loadRepositoryUpdateValidationCurrentRows(
    {
      db: {
        selectFrom(table: string) {
          calls.push({ method: 'selectFrom', table });
          return {
            select(callback: (builder: any) => any[]) {
              callback({
                ref(value: string) {
                  return {
                    as(alias: string) {
                      return { kind: 'ref', value, alias };
                    },
                  };
                },
              });
              return query;
            },
          };
        },
      },
      table: 'demo_table',
      meta: { fields: new Map([['Id', {}]]) } as any,
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
      convertCondition(eb, condition, selfTable) {
        return { eb, condition, selfTable };
      },
      async execute() {
        return [{ Name: 'missing-id' }] as any;
      },
      decodeFromDb(row) {
        return row;
      },
    },
    ['row_1']
  );

  expect(Array.from(result.entries())).toEqual([]);
});

test('repository update write helpers drop decoded rows when Id is numeric zero', async () => {
  const query = {
    where(callback: ({ eb }: any) => unknown) {
      callback({ eb: 'EB' });
      return query;
    },
  };

  const result = await loadRepositoryUpdateValidationCurrentRows(
    {
      db: {
        selectFrom() {
          return {
            select(callback: (builder: any) => any[]) {
              callback({
                ref(value: string) {
                  return {
                    as(alias: string) {
                      return { kind: 'ref', value, alias };
                    },
                  };
                },
              });
              return query;
            },
          };
        },
      },
      table: 'demo_table',
      meta: { fields: new Map([['Id', {}]]) } as any,
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
      convertCondition(eb, condition, selfTable) {
        return { eb, condition, selfTable };
      },
      async execute() {
        return [{ Id: 0, Name: 'zero-id' }] as any;
      },
      decodeFromDb(row) {
        return row;
      },
    },
    ['row_1']
  );

  expect(Array.from(result.entries())).toEqual([]);
});

test('repository update write helpers keep decoded rows when Id is present and non-empty after trim', async () => {
  const query = {
    where(callback: ({ eb }: any) => unknown) {
      callback({ eb: 'EB' });
      return query;
    },
  };

  const result = await loadRepositoryUpdateValidationCurrentRows(
    {
      db: {
        selectFrom() {
          return {
            select(callback: (builder: any) => any[]) {
              callback({
                ref(value: string) {
                  return {
                    as(alias: string) {
                      return { kind: 'ref', value, alias };
                    },
                  };
                },
              });
              return query;
            },
          };
        },
      },
      table: 'demo_table',
      meta: { fields: new Map([['Id', {}]]) } as any,
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
      convertCondition(eb, condition, selfTable) {
        return { eb, condition, selfTable };
      },
      async execute() {
        return [{ Id: ' id_1 ', Name: 'keep' }] as any;
      },
      decodeFromDb(row) {
        return row;
      },
    },
    ['id_1']
  );

  expect(Array.from(result.entries())).toEqual([['id_1', { Id: ' id_1 ', Name: 'keep' }]]);
});

test('repository update write helpers handle undefined execute result as empty current rows', async () => {
  const query = {
    where(callback: ({ eb }: any) => unknown) {
      callback({ eb: 'EB' });
      return query;
    },
  };

  const result = await loadRepositoryUpdateValidationCurrentRows(
    {
      db: {
        selectFrom() {
          return {
            select(callback: (builder: any) => any[]) {
              callback({
                ref(value: string) {
                  return {
                    as(alias: string) {
                      return { kind: 'ref', value, alias };
                    },
                  };
                },
              });
              return query;
            },
          };
        },
      },
      table: 'demo_table',
      meta: { fields: new Map([['Id', {}]]) } as any,
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
      convertCondition(eb, condition, selfTable) {
        return { eb, condition, selfTable };
      },
      async execute() {
        return undefined as any;
      },
      decodeFromDb(row) {
        return row;
      },
    },
    ['id_1']
  );

  expect(Array.from(result.entries())).toEqual([]);
});
