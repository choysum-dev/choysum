// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { REL_ALIAS_PREFIX } from '../../../relation/relation_alias';
import { buildHiddenScaleAlias } from '../../hidden_scale_alias';
import { executeRepositorySearch } from '..';

function createSearchHarness(overrides: Partial<Record<string, any>> = {}) {
  const calls: Record<string, any[]> = {
    aliases: [],
    relation: [],
    executed: [],
  };

  const query: any = {
    selected: undefined,
    whereArg: undefined,
    forUpdateCalled: false,
    orderArg: undefined,
    select(factory: any) {
      const qb = {
        ref(path: string) {
          return {
            path,
            as(alias: string) {
              return { kind: 'ref', path, alias };
            },
          };
        },
      };
      this.selected = factory(qb);
      return this;
    },
    where(factory: any) {
      this.whereArg = factory({ eb: 'EB' });
      return this;
    },
    limit() {
      return this;
    },
    offset() {
      return this;
    },
    forUpdate() {
      this.forUpdateCalled = true;
      return this;
    },
  };

  const deps: any = {
    db: {
      selectFrom() {
        return query;
      },
    },
    table: 'demo_table',
    meta: {
      type: class DemoModel {},
      fields: new Map([
        ['Id', { type: 'char', column: { name: 'Id' } }],
        ['AmountScale', { type: 'integer', column: { name: 'AmountScale' } }],
        ['Amount', { type: 'decimal', column: { name: 'Amount', scaleField: 'AmountScale' } }],
      ]),
      orderBy: undefined,
    },
    getDialect() {
      return 'postgres';
    },
    isTopLevelGrpcCall() {
      return true;
    },
    buildSelectionTree() {
      return {
        columns: new Set(['Amount']),
        relations: new Map([['Owner', { node: true }]]),
      };
    },
    getScalarFields() {
      return ['Id', 'Amount'];
    },
    async pruneSelectionTreeForFieldRule() {
      return;
    },
    makeSelectCtx() {
      return {
        field(_model: any, field: string) {
          return {
            field,
            as(alias: string) {
              return { kind: 'ctx-field', field, alias };
            },
          };
        },
      };
    },
    aliasSelection(selection: any, alias: string) {
      calls.aliases.push({ selection, alias });
      return { selection, alias };
    },
    buildRelationJsonSelect(_qb: any, _parentMeta: any, relKey: string) {
      calls.relation.push(relKey);
      return { kind: 'relation-json', relKey };
    },
    async applyRecordRuleToCondition(condition: any) {
      return condition;
    },
    applyDefaultLayers(condition: any) {
      return condition;
    },
    isEmptyCondition() {
      return false;
    },
    convertCondition(_eb: any, condition: any, table: string) {
      return { kind: 'compiled', condition, table };
    },
    normalizeOrderBy() {
      return [];
    },
    resolveEffectiveOrder() {
      return [];
    },
    applyOrderByToQuery(q: any) {
      return q;
    },
    async execute(input: any) {
      calls.executed.push(input);
      return [{ Id: 'id_1', Amount: '10.5' }];
    },
    decodeRowWithTree(_meta: any, _node: any, row: any) {
      return row;
    },
    ...overrides,
  };

  return { deps, query, calls };
}

test('repository read search runtime builds hidden decimal scale and relation aliases', async () => {
  const { deps, query, calls } = createSearchHarness();
  const rows = await executeRepositorySearch(deps as any, ['Id', '=', 'id_1'] as any, { fields: ['Amount'] as any, forUpdate: true });

  expect(rows).toEqual([{ Id: 'id_1', Amount: '10.5' }]);
  expect(calls.executed.length).toBe(1);
  expect(calls.relation.length).toBe(1);
  expect(Array.isArray(query.selected)).toBe(true);
  expect(JSON.stringify(query.selected || []).includes(REL_ALIAS_PREFIX)).toBe(true);
  expect(JSON.stringify(query.selected || []).includes(buildHiddenScaleAlias('Amount')) || JSON.stringify(query.selected || []).includes('$dec$')).toBe(true);
});

test('repository read search runtime skips forUpdate when dialect is sqlite and tolerates prune failures', async () => {
  let pruneCalls = 0;
  const { deps, query } = createSearchHarness({
    getDialect() {
      return 'sqlite';
    },
    async pruneSelectionTreeForFieldRule() {
      pruneCalls += 1;
      throw new Error('prune failed');
    },
  });

  const rows = await executeRepositorySearch(deps as any, ['Id', '=', 'id_1'] as any, { fields: ['Amount'] as any, forUpdate: true });
  expect(rows).toEqual([{ Id: 'id_1', Amount: '10.5' }]);
  expect(pruneCalls).toBe(1);
  expect(query.forUpdateCalled).toBe(false);
});

test('repository read search runtime supports dotted and select fields together with relation and decimal scale aliases', async () => {
  const dottedCalls: string[] = [];
  class DemoModel {
    sqlDisplayName() {
      return { kind: 'display-expr' };
    }
  }

  const { deps, query, calls } = createSearchHarness({
    meta: {
      type: DemoModel,
      fields: new Map([
        ['Id', { type: 'char', column: { name: 'Id' } }],
        ['AmountScale', { type: 'integer', column: { name: 'AmountScale' } }],
        ['Amount', { type: 'decimal', column: { name: 'Amount', scaleField: 'AmountScale' } }],
        ['DisplayName', {}],
      ]),
      sqlComputeHandlers: new Map([['DisplayName', { field: 'DisplayName', method: 'sqlDisplayName' }]]),
      orderBy: undefined,
    },
    buildSelectionTree() {
      return {
        columns: new Set(['Amount', 'Owner.Name', 'DisplayName']),
        relations: new Map([['Owner', { node: true }]]),
      };
    },
    makeSelectCtx() {
      return {
        field(_model: any, field: string) {
          dottedCalls.push(String(field));
          return {
            kind: 'ctx-field',
            field,
            as(alias: string) {
              return { kind: 'ctx-field', field, alias };
            },
          };
        },
      };
    },
  });

  const rows = await executeRepositorySearch(deps as any, ['Id', '=', 'id_1'] as any, { fields: ['Amount', 'Owner.Name', 'DisplayName'] as any });
  expect(rows).toEqual([{ Id: 'id_1', Amount: '10.5' }]);
  expect(dottedCalls.includes('Owner.Name')).toBe(true);
  expect(calls.relation).toEqual(['Owner']);
  const serialized = JSON.stringify(query.selected || []);
  expect(serialized.includes('DisplayName')).toBe(true);
  expect(serialized.includes(buildHiddenScaleAlias('Amount')) || serialized.includes('$dec$')).toBe(true);
});

test('repository read search runtime keeps explicit Id field, supports decimal select scale, and returns empty result directly', async () => {
  class DemoModel {
    sqlAmountScale() {
      return { kind: 'amount-scale-select' };
    }
  }

  const { deps, query, calls } = createSearchHarness({
    meta: {
      type: DemoModel,
      fields: new Map([
        ['Id', { type: 'char', column: { name: 'Id' } }],
        ['AmountScale', { type: 'integer' }],
        [
          'Amount',
          {
            type: 'decimal',
            column: {
              scaleField: 'AmountScale',
            },
          },
        ],
      ]),
      sqlComputeHandlers: new Map([['AmountScale', { field: 'AmountScale', method: 'sqlAmountScale' }]]),
      orderBy: undefined,
    },
    buildSelectionTree(_meta: any, fields: any[]) {
      expect(fields).toEqual(['Id', 'Amount']);
      return {
        columns: new Set(['Id', 'Amount']),
        relations: new Map(),
      };
    },
    async execute(input: any) {
      calls.executed.push(input);
      return [];
    },
  });

  const rows = await executeRepositorySearch(deps as any, ['Id', '=', 'id_1'] as any, { fields: ['Id', 'Amount'] as any });
  expect(rows).toEqual([]);
  expect(calls.executed.length).toBe(1);
  const serialized = JSON.stringify(query.selected || []);
  expect(serialized.includes(buildHiddenScaleAlias('Amount')) || serialized.includes('$dec$')).toBe(true);
});

test('repository read search runtime tolerates decimal field without column or select spec', async () => {
  const { deps, query } = createSearchHarness({
    meta: {
      type: class DemoModel {},
      fields: new Map([
        ['Id', { type: 'char', column: { name: 'Id' } }],
        ['Amount', { type: 'decimal' }],
      ]),
      orderBy: undefined,
    },
    buildSelectionTree() {
      return {
        columns: new Set(['Amount']),
        relations: new Map(),
      };
    },
    async execute() {
      return [];
    },
  });

  const rows = await executeRepositorySearch(deps as any, ['Id', '=', 'id_1'] as any, { fields: ['Amount'] as any });
  expect(rows).toEqual([]);
  const serialized = JSON.stringify(query.selected || []);
  expect(serialized.includes(buildHiddenScaleAlias('Amount'))).toBe(false);
});

test('repository read search runtime includes sql-compute fields in default projection when fields are omitted', async () => {
  class DemoModel {
    sqlDisplayName() {
      return { kind: 'display-select-expr' };
    }
  }

  const { deps, query, calls } = createSearchHarness({
    meta: {
      type: DemoModel,
      fields: new Map([
        ['Id', { type: 'char', column: { name: 'Id' } }],
        ['Name', { type: 'varchar', column: { name: 'Name' } }],
        ['DisplayName', { type: 'varchar' }],
      ]),
      sqlComputeHandlers: new Map([['DisplayName', { field: 'DisplayName', method: 'sqlDisplayName' }]]),
      orderBy: undefined,
    },
    getScalarFields() {
      return ['Id', 'Name', 'DisplayName'];
    },
    async execute(input: any) {
      calls.executed.push(input);
      return [];
    },
  });

  const rows = await executeRepositorySearch(deps as any, ['Id', '=', 'id_1'] as any);
  expect(rows).toEqual([]);
  expect(calls.executed.length).toBe(1);
  expect(calls.aliases.some(item => item.alias === 'DisplayName')).toBe(true);
  expect(Array.isArray(query.selected)).toBe(true);
  expect((query.selected || []).length).toBe(3);
  const serialized = JSON.stringify(query.selected || []);
  expect(serialized.includes('demo_table.Id')).toBe(true);
  expect(serialized.includes('demo_table.Name')).toBe(true);
  expect(serialized.includes('DisplayName')).toBe(true);
  expect(serialized.includes('display-select-expr')).toBe(true);
});
