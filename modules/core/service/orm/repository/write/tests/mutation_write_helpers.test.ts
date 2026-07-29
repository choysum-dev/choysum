// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { applyRepositoryMutationWriteCondition, resolveRepositoryMutationWriteTargetIds } from '../mutation_write_helpers';

test('repository mutation write helpers resolve target ids via company access for company scoped models', async () => {
  const calls: Array<Record<string, any>> = [];
  const ids = await resolveRepositoryMutationWriteTargetIds(
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
    'delete',
    ['Id', '=', '1'] as any
  );

  expect(ids).toEqual(['company_1']);
  expect(calls).toEqual([
    { method: 'company', condition: ['Id', '=', '1'] },
    { method: 'recordRule', op: 'delete', targetIds: ['company_1'] },
  ]);
});

test('repository mutation write helpers resolve target ids via locate for non-company scoped models', async () => {
  const calls: Array<Record<string, any>> = [];
  const ids = await resolveRepositoryMutationWriteTargetIds(
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
    'write',
    ['Id', '=', '2'] as any
  );

  expect(ids).toEqual(['locate_1']);
  expect(calls).toEqual([
    { method: 'locate', condition: ['Id', '=', '2'] },
    { method: 'recordRule', op: 'write', targetIds: ['locate_1'] },
  ]);
});

test('repository mutation write helpers skip record rule assertion when no targets resolve', async () => {
  const calls: Array<Record<string, any>> = [];
  const ids = await resolveRepositoryMutationWriteTargetIds(
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
    'delete',
    ['Id', '=', '3'] as any
  );

  expect(ids).toEqual([]);
  expect(calls).toEqual([{ method: 'locate', condition: ['Id', '=', '3'] }]);
});

test('repository mutation write helpers apply record rule and default layers to delete query', async () => {
  const calls: Array<Record<string, any>> = [];
  const query = {
    where(callback: ({ eb }: any) => unknown) {
      calls.push({ method: 'where', result: callback({ eb: 'EB' }) });
      return { tagged: 'where-query' };
    },
  };

  const result = await applyRepositoryMutationWriteCondition(
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
    'delete',
    ['Id', '=', '4'] as any
  );

  expect(result).toEqual({ tagged: 'where-query' });
  expect(calls).toEqual([
    { method: 'recordRule', condition: ['Id', '=', '4'], op: 'delete' },
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

test('repository mutation write helpers leave query unchanged when filtered condition is empty', async () => {
  const calls: Array<Record<string, any>> = [];
  const query = {
    tagged: 'original-query',
    where() {
      calls.push({ method: 'where' });
      return { tagged: 'where-query' };
    },
  };

  const result = await applyRepositoryMutationWriteCondition(
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
    'write',
    ['Id', '=', '5'] as any
  );

  expect(result).toBe(query);
  expect(calls).toEqual([
    { method: 'recordRule', condition: ['Id', '=', '5'], op: 'write' },
    { method: 'defaultLayers', condition: ['Id', '=', '5'] },
    { method: 'isEmpty', condition: ['Id', '=', '5'] },
  ]);
});
