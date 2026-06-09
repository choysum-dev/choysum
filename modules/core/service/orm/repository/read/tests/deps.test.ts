// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  createRepositoryConditionQueryDeps,
  createRepositoryReadAggregateDeps,
  createRepositoryReadAggregateFacadeDeps,
  createRepositoryReadConditionDeps,
  createRepositoryReadOrderDeps,
  createRepositoryReadQueryFacadeDeps,
  createRepositorySearchFacadeDeps,
} from '..';

function createExpressionBuilder() {
  const eb: any = (lhs: any, op: any, rhs: any) => ({ kind: 'cmp', lhs, op, rhs });
  eb.ref = (alias: string) => ({ kind: 'ref', alias });
  eb.and = (parts: any[]) => ({ kind: 'and', parts });
  eb.or = (parts: any[]) => ({ kind: 'or', parts });
  return eb;
}

function createAggregateDeps(overrides?: Partial<Parameters<typeof createRepositoryReadAggregateDeps>[0]>) {
  return createRepositoryReadAggregateDeps({
    db: {},
    table: 'demo_table',
    meta: {} as any,
    ctx: {},
    getDialect() {
      return 'postgres';
    },
    makeSelectCtx() {
      return {};
    },
    convertCondition() {
      return { kind: 'fallback' };
    },
    async applyRecordRuleToCondition(condition) {
      return condition;
    },
    applyDefaultLayers(condition) {
      return condition;
    },
    isEmptyCondition() {
      return false;
    },
    normalizeOrderBy(orderBy) {
      return orderBy;
    },
    applyOrderByToQuery(query) {
      return query;
    },
    async execute() {
      return [];
    },
    ...overrides,
  });
}

test('repository read deps delegate condition query deps unchanged', async () => {
  const calls: Array<Record<string, any>> = [];
  const query = { kind: 'query' };
  const deps = createRepositoryConditionQueryDeps({
    db: { name: 'db' },
    table: 'demo_table',
    applyConditionLayers(condition) {
      calls.push({ method: 'applyLayers', condition });
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
    async execute(input) {
      calls.push({ method: 'execute', input });
      return [{ Id: 'row_1' }] as any;
    },
  });

  expect(deps.db).toEqual({ name: 'db' });
  expect(deps.table).toBe('demo_table');
  expect(deps.applyConditionLayers(['Id', '=', '1'] as any)).toEqual({
    And: [
      ['Id', '=', '1'],
      ['DeletedAt', 'is', null],
    ],
  });
  expect(deps.isEmptyCondition(['Id', '=', '1'] as any)).toBe(false);
  expect(deps.convertCondition('EB', ['Id', '=', '1'] as any, 'demo_table')).toEqual({
    eb: 'EB',
    condition: ['Id', '=', '1'],
    selfTable: 'demo_table',
  });
  expect(await deps.execute(query)).toEqual([{ Id: 'row_1' }]);
  expect(calls).toEqual([
    { method: 'applyLayers', condition: ['Id', '=', '1'] },
    { method: 'isEmpty', condition: ['Id', '=', '1'] },
    { method: 'convert', eb: 'EB', condition: ['Id', '=', '1'], selfTable: 'demo_table' },
    { method: 'execute', input: query },
  ]);
});

test('repository read deps delegate read condition deps unchanged', async () => {
  const calls: Array<Record<string, any>> = [];
  const query = { kind: 'query' };
  const deps = createRepositoryReadConditionDeps({
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
    async execute(input) {
      calls.push({ method: 'execute', input });
      return [{ Id: 'row_1' }] as any;
    },
  });

  expect(await deps.applyRecordRuleToCondition(['Id', '=', '1'] as any, 'read')).toEqual({
    And: [
      ['Id', '=', '1'],
      ['CompanyId', '=', 'company_a'],
    ],
  });
  expect(deps.applyDefaultLayers(['Id', '=', '1'] as any)).toEqual({
    And: [
      ['Id', '=', '1'],
      ['DeletedAt', 'is', null],
    ],
  });
  expect(deps.isEmptyCondition(['Id', '=', '1'] as any)).toBe(false);
  expect(deps.convertCondition('EB', ['Id', '=', '1'] as any, 'demo_table')).toEqual({
    eb: 'EB',
    condition: ['Id', '=', '1'],
    selfTable: 'demo_table',
  });
  expect(await deps.execute(query)).toEqual([{ Id: 'row_1' }]);
  expect(calls).toEqual([
    { method: 'recordRule', condition: ['Id', '=', '1'], op: 'read' },
    { method: 'defaultLayers', condition: ['Id', '=', '1'] },
    { method: 'isEmpty', condition: ['Id', '=', '1'] },
    { method: 'convert', eb: 'EB', condition: ['Id', '=', '1'], selfTable: 'demo_table' },
    { method: 'execute', input: query },
  ]);
});

test('repository read deps delegate read order deps unchanged', () => {
  const calls: Array<Record<string, any>> = [];
  const deps = createRepositoryReadOrderDeps({
    normalizeOrderBy(input) {
      calls.push({ method: 'normalize', input });
      return [{ field: 'Name', order: 'asc' }];
    },
    applyOrderByToQuery(query, meta, table, orderBy) {
      calls.push({ method: 'applyOrder', query, meta, table, orderBy });
      return { tagged: 'ordered-query' };
    },
  });

  expect(deps.normalizeOrderBy('Name asc')).toEqual([{ field: 'Name', order: 'asc' }]);
  expect(deps.applyOrderByToQuery({ tagged: 'query' }, { modelName: 'Demo' }, 'demo_table', [{ field: 'Name', order: 'asc' }])).toEqual({
    tagged: 'ordered-query',
  });
  expect(calls).toEqual([
    { method: 'normalize', input: 'Name asc' },
    {
      method: 'applyOrder',
      query: { tagged: 'query' },
      meta: { modelName: 'Demo' },
      table: 'demo_table',
      orderBy: [{ field: 'Name', order: 'asc' }],
    },
  ]);
});

test('repository read deps facade merges read condition and order deps unchanged', async () => {
  const calls: Array<Record<string, any>> = [];
  const deps = createRepositoryReadQueryFacadeDeps({
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
      return false;
    },
    convertCondition(eb, condition, selfTable) {
      calls.push({ method: 'convert', eb, condition, selfTable });
      return { eb, condition, selfTable };
    },
    async execute(input) {
      calls.push({ method: 'execute', input });
      return [{ Id: 'row_1' }] as any;
    },
    normalizeOrderBy(input) {
      calls.push({ method: 'normalize', input });
      return [{ field: 'Name', order: 'asc' }];
    },
    applyOrderByToQuery(query, meta, table, orderBy) {
      calls.push({ method: 'applyOrder', query, meta, table, orderBy });
      return { tagged: 'ordered-query' };
    },
  });

  expect(await deps.applyRecordRuleToCondition(['Id', '=', '1'] as any, 'read')).toEqual(['Id', '=', '1']);
  expect(deps.applyDefaultLayers(['Id', '=', '1'] as any)).toEqual(['Id', '=', '1']);
  expect(deps.isEmptyCondition(['Id', '=', '1'] as any)).toBe(false);
  expect(deps.convertCondition('EB', ['Id', '=', '1'] as any, 'demo_table')).toEqual({
    eb: 'EB',
    condition: ['Id', '=', '1'],
    selfTable: 'demo_table',
  });
  expect(await deps.execute({ kind: 'query' })).toEqual([{ Id: 'row_1' }]);
  expect(deps.normalizeOrderBy('Name asc')).toEqual([{ field: 'Name', order: 'asc' }]);
  expect(deps.applyOrderByToQuery({ tagged: 'query' }, { modelName: 'Demo' }, 'demo_table', [{ field: 'Name', order: 'asc' }])).toEqual({
    tagged: 'ordered-query',
  });
  expect(calls).toEqual([
    { method: 'recordRule', condition: ['Id', '=', '1'], op: 'read' },
    { method: 'defaultLayers', condition: ['Id', '=', '1'] },
    { method: 'isEmpty', condition: ['Id', '=', '1'] },
    { method: 'convert', eb: 'EB', condition: ['Id', '=', '1'], selfTable: 'demo_table' },
    { method: 'execute', input: { kind: 'query' } },
    { method: 'normalize', input: 'Name asc' },
    {
      method: 'applyOrder',
      query: { tagged: 'query' },
      meta: { modelName: 'Demo' },
      table: 'demo_table',
      orderBy: [{ field: 'Name', order: 'asc' }],
    },
  ]);
});

test('repository read search facade merges search-specific deps with read query facade unchanged', async () => {
  const calls: Array<Record<string, any>> = [];
  const deps = createRepositorySearchFacadeDeps({
    db: { name: 'db' },
    table: 'demo_table',
    meta: { modelName: 'Demo' } as any,
    getDialect() {
      calls.push({ method: 'dialect' });
      return 'postgres';
    },
    isTopLevelGrpcCall() {
      calls.push({ method: 'topLevel' });
      return true;
    },
    buildSelectionTree(meta, fields) {
      calls.push({ method: 'buildSelectionTree', meta, fields });
      return { meta, fields };
    },
    getScalarFields(meta) {
      calls.push({ method: 'getScalarFields', meta });
      return ['Id', 'Name'];
    },
    async pruneSelectionTreeForFieldRule(meta, node, denyCache) {
      calls.push({ method: 'pruneSelectionTree', meta, node, denyCache });
    },
    makeSelectCtx(builder, table, meta) {
      calls.push({ method: 'makeSelectCtx', builder, table, meta });
      return { builder, table, meta };
    },
    aliasSelection(selection, alias) {
      calls.push({ method: 'alias', selection, alias });
      return { selection, alias };
    },
    buildRelationJsonSelect(qb, parentMeta, relKey, entry) {
      calls.push({ method: 'buildRelationJsonSelect', qb, parentMeta, relKey, entry });
      return { qb, parentMeta, relKey, entry };
    },
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
      return false;
    },
    convertCondition(eb, condition, selfTable) {
      calls.push({ method: 'convert', eb, condition, selfTable });
      return { eb, condition, selfTable };
    },
    async execute(input) {
      calls.push({ method: 'execute', input });
      return [{ Id: 'row_1' }] as any;
    },
    normalizeOrderBy(input) {
      calls.push({ method: 'normalize', input });
      return [{ field: 'Name', order: 'asc' }];
    },
    applyOrderByToQuery(query, meta, table, orderBy) {
      calls.push({ method: 'applyOrder', query, meta, table, orderBy });
      return { tagged: 'ordered-query' };
    },
    resolveEffectiveOrder(overrideOrder, metaOrder, meta) {
      calls.push({ method: 'resolveOrder', overrideOrder, metaOrder, meta });
      return overrideOrder || metaOrder;
    },
    decodeRowWithTree(meta, node, row) {
      calls.push({ method: 'decodeRowWithTree', meta, node, row });
      return { ...row, meta, node };
    },
  });

  expect(deps.db).toEqual({ name: 'db' });
  expect(deps.table).toBe('demo_table');
  expect(deps.meta).toEqual({ modelName: 'Demo' });
  expect(deps.getDialect()).toBe('postgres');
  expect(deps.isTopLevelGrpcCall()).toBe(true);
  expect(deps.buildSelectionTree({ modelName: 'Demo' } as any, ['Id'])).toEqual({ meta: { modelName: 'Demo' }, fields: ['Id'] });
  expect(deps.getScalarFields({ modelName: 'Demo' } as any)).toEqual(['Id', 'Name']);
  await deps.pruneSelectionTreeForFieldRule({ modelName: 'Demo' } as any, { columns: new Set(['Id']) } as any, new Map());
  expect(deps.makeSelectCtx('QB', 'demo_table', { modelName: 'Demo' } as any)).toEqual({ builder: 'QB', table: 'demo_table', meta: { modelName: 'Demo' } });
  expect(deps.aliasSelection('SEL', 'Name')).toEqual({ selection: 'SEL', alias: 'Name' });
  expect(deps.buildRelationJsonSelect('QB', { modelName: 'Demo' } as any, 'Owner', { fieldType: 'ManyToOne' } as any)).toEqual({
    qb: 'QB',
    parentMeta: { modelName: 'Demo' },
    relKey: 'Owner',
    entry: { fieldType: 'ManyToOne' },
  });
  expect(await deps.applyRecordRuleToCondition(['Id', '=', '1'] as any, 'read')).toEqual(['Id', '=', '1']);
  expect(deps.applyDefaultLayers(['Id', '=', '1'] as any)).toEqual(['Id', '=', '1']);
  expect(deps.isEmptyCondition(['Id', '=', '1'] as any)).toBe(false);
  expect(deps.convertCondition('EB', ['Id', '=', '1'] as any, 'demo_table')).toEqual({ eb: 'EB', condition: ['Id', '=', '1'], selfTable: 'demo_table' });
  expect(await deps.execute({ kind: 'query' })).toEqual([{ Id: 'row_1' }]);
  expect(deps.normalizeOrderBy('Name asc')).toEqual([{ field: 'Name', order: 'asc' }]);
  expect(deps.applyOrderByToQuery({ tagged: 'query' }, { modelName: 'Demo' }, 'demo_table', [{ field: 'Name', order: 'asc' }])).toEqual({
    tagged: 'ordered-query',
  });
  expect(deps.resolveEffectiveOrder([{ field: 'Name', order: 'asc' }], undefined, { modelName: 'Demo' } as any)).toEqual([{ field: 'Name', order: 'asc' }]);
  expect(deps.decodeRowWithTree({ modelName: 'Demo' } as any, { columns: new Set(['Id']) } as any, { Id: 'row_1' })).toEqual({
    Id: 'row_1',
    meta: { modelName: 'Demo' },
    node: { columns: new Set(['Id']) },
  });
});

test('repository read deps convertHaving resolves alias predicates with current table binding', () => {
  const eb = createExpressionBuilder();
  const deps = createAggregateDeps({
    convertCondition() {
      throw new Error('should not fall back for known alias');
    },
  });

  expect(deps.convertHaving(eb, ['__count', '=', null] as any, new Set(['__count']))).toEqual({
    kind: 'cmp',
    lhs: { kind: 'ref', alias: '__count' },
    op: 'is',
    rhs: null,
  });
});

test('repository read deps convertHaving falls back to condition compiler with repository table', () => {
  const eb = createExpressionBuilder();
  const calls: Array<Record<string, any>> = [];
  const deps = createAggregateDeps({
    convertCondition(builder, condition, selfTable) {
      calls.push({ builder, condition, selfTable });
      return { kind: 'fallback', condition, selfTable };
    },
  });

  const result = deps.convertHaving(
    eb,
    {
      Or: [
        ['Status', '=', 'ready'],
        ['totalAmount', '>', 10],
      ],
    } as any,
    new Set(['totalAmount'])
  );

  expect(result).toEqual({
    kind: 'or',
    parts: [
      { kind: 'fallback', condition: ['Status', '=', 'ready'], selfTable: 'demo_table' },
      { kind: 'cmp', lhs: { kind: 'ref', alias: 'totalAmount' }, op: '>', rhs: 10 },
    ],
  });

  expect(calls).toEqual([
    {
      builder: eb,
      condition: ['Status', '=', 'ready'],
      selfTable: 'demo_table',
    },
  ]);
});

test('repository read aggregate facade delegates aggregate deps unchanged', () => {
  const deps = createRepositoryReadAggregateFacadeDeps({
    db: { name: 'db' },
    table: 'demo_table',
    meta: { modelName: 'Demo' } as any,
    ctx: { lang: 'zh-CN' },
    getDialect() {
      return 'postgres';
    },
    makeSelectCtx() {
      return { kind: 'ctx' };
    },
    async applyRecordRuleToCondition(condition) {
      return condition;
    },
    applyDefaultLayers(condition) {
      return condition;
    },
    isEmptyCondition() {
      return false;
    },
    convertCondition() {
      return { kind: 'fallback' };
    },
    normalizeOrderBy(orderBy) {
      return orderBy;
    },
    applyOrderByToQuery(query) {
      return query;
    },
    async execute() {
      return [];
    },
  });

  expect(deps.db).toEqual({ name: 'db' });
  expect(deps.table).toBe('demo_table');
  expect(deps.meta).toEqual({ modelName: 'Demo' });
  expect(deps.ctx).toEqual({ lang: 'zh-CN' });
  expect(deps.getDialect()).toBe('postgres');
  expect(deps.makeSelectCtx('QB', 'demo_table', { modelName: 'Demo' } as any)).toEqual({ kind: 'ctx' });
});
