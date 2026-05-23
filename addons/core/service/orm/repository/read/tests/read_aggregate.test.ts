// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { executeRepositoryReadGroup, executeRepositoryReadGroupCount, executeRepositoryReadTotals } from '..';

type QueryLog = Array<Record<string, any>>;

function createFieldExpr(field: string) {
  return {
    kind: 'field',
    field,
    as(alias: string) {
      return { kind: 'aliased-field', field, alias };
    },
  };
}

function createQuery(tableOrAlias: any, log: QueryLog) {
  return {
    tableOrAlias,
    selectArg: undefined as any,
    whereArg: undefined as any,
    groupByArg: undefined as any,
    havingArg: undefined as any,
    limitArg: undefined as any,
    offsetArg: undefined as any,
    select(arg: any) {
      this.selectArg = typeof arg === 'function' ? arg({ builder: 'SEL' }) : arg;
      log.push({ method: 'select', tableOrAlias: this.tableOrAlias });
      return this;
    },
    where(callback: ({ eb }: any) => unknown) {
      this.whereArg = callback({ eb: 'EB' });
      log.push({ method: 'where', tableOrAlias: this.tableOrAlias, whereArg: this.whereArg });
      return this;
    },
    groupBy(callback: (gb: any) => unknown) {
      this.groupByArg = callback({ gb: 'GB' });
      log.push({ method: 'groupBy', tableOrAlias: this.tableOrAlias });
      return this;
    },
    having(callback: ({ eb }: any) => unknown) {
      this.havingArg = callback({ eb: 'EB' });
      log.push({ method: 'having', tableOrAlias: this.tableOrAlias, havingArg: this.havingArg });
      return this;
    },
    limit(value: number) {
      this.limitArg = value;
      log.push({ method: 'limit', value });
      return this;
    },
    offset(value: number) {
      this.offsetArg = value;
      log.push({ method: 'offset', value });
      return this;
    },
    as(alias: string) {
      return { kind: 'aliased-subquery', alias, source: this };
    },
  };
}

function createAggregateRuntime(overrides?: Partial<any>) {
  const log: QueryLog = [];
  const db = {
    fn: {
      countAll() {
        return {
          as(alias: string) {
            return { kind: 'count-all', alias };
          },
        };
      },
    },
    selectFrom(tableOrAlias: any) {
      log.push({ method: 'selectFrom', tableOrAlias });
      return createQuery(tableOrAlias, log);
    },
  };

  const params = {
    db,
    table: 'demo_table',
    meta: {
      type: class DemoModel {},
      fields: new Map([
        ['Amount', { type: 'decimal', column: { scale: 2 } }],
        ['CreatedAt', { type: 'datetime' }],
        ['Status', { type: 'char' }],
      ]),
    } as any,
    ctx: { tz: 'Asia/Shanghai' },
    getDialect() {
      return 'postgres' as const;
    },
    makeSelectCtx() {
      return {
        field(_type: any, field: string) {
          return createFieldExpr(field);
        },
      };
    },
    convertCondition(_eb: any, condition: any, selfTable?: string) {
      return { kind: 'condition', condition, selfTable };
    },
    convertHaving(_eb: any, condition: any, knownAliases: Set<string>) {
      return { kind: 'having', condition, knownAliases: Array.from(knownAliases).sort() };
    },
    async applyRecordRuleToCondition(condition: any) {
      return condition;
    },
    applyDefaultLayers(condition: any) {
      return condition;
    },
    isEmptyCondition(condition: any) {
      return Array.isArray(condition) && condition.length === 0;
    },
    normalizeOrderBy(orderBy: any) {
      return orderBy;
    },
    applyOrderByToQuery(query: any, _meta: any, _table: string, orderBy: any) {
      (query as any).orderByApplied = orderBy;
      log.push({ method: 'applyOrderByToQuery', orderBy });
      return query;
    },
    async execute(_query: any) {
      return [];
    },
    ...overrides,
  };

  return { params, log };
}

test('repository read aggregate runtime readGroup requires groupby option', async () => {
  const { params } = createAggregateRuntime();
  let actual = '';
  try {
    await executeRepositoryReadGroup(params as any, {} as any);
  } catch (error) {
    actual = String((error as Error)?.message || error);
  }

  expect(actual).toBe('readGroup requires options.groupby');
});

test('repository read aggregate runtime readGroup applies having/order/limit/offset and normalizes __count', async () => {
  const { params, log } = createAggregateRuntime({
    normalizeOrderBy() {
      return [{ field: 'Status', order: 'asc' }];
    },
    async execute(query: any) {
      log.push({
        method: 'execute',
        hasWhere: Boolean(query.whereArg),
        hasGroupBy: Boolean(query.groupByArg),
        hasHaving: Boolean(query.havingArg),
        limit: query.limitArg,
        offset: query.offsetArg,
      });
      return [
        {
          Status: 'draft',
          Amount__sum: '7.129',
          __count: '3',
        },
      ] as any;
    },
  });

  const rows = await executeRepositoryReadGroup(
    params as any,
    {
      groupby: 'Status',
      fields: [{ field: 'Amount', agg: 'sum' }],
      condition: ['Id', '=', '1'],
      having: ['Amount__sum', '>', 0],
      orderBy: 'Status asc',
      limit: 10,
      offset: 5,
    } as any
  );

  expect(rows).toEqual([
    {
      Status: 'draft',
      Amount__sum: 7.13,
      __count: 3,
    },
  ]);

  const havingCall = log.find(item => item.method === 'having') as any;
  expect(havingCall.havingArg).toEqual({
    kind: 'having',
    condition: ['Amount__sum', '>', 0],
    knownAliases: ['Amount__sum', 'Status', '__count'],
  });

  expect(log.some(item => item.method === 'applyOrderByToQuery')).toBe(true);
  expect(log.some(item => item.method === 'limit' && item.value === 10)).toBe(true);
  expect(log.some(item => item.method === 'offset' && item.value === 5)).toBe(true);
});

test('repository read aggregate runtime readTotals returns empty object row with numeric __count when query is empty', async () => {
  const { params, log } = createAggregateRuntime({
    async execute(query: any) {
      log.push({ method: 'execute', hasWhere: Boolean(query.whereArg) });
      return undefined as any;
    },
  });

  const row = await executeRepositoryReadTotals(
    params as any,
    {
      fields: [{ field: 'Amount', agg: 'sum' }],
      condition: [],
    } as any
  );

  expect(row).toEqual({ __count: 0 });
});

test('repository read aggregate runtime readGroupCount requires groupby option', async () => {
  const { params } = createAggregateRuntime();
  let actual = '';
  try {
    await executeRepositoryReadGroupCount(params as any, {} as any);
  } catch (error) {
    actual = String((error as Error)?.message || error);
  }

  expect(actual).toBe('readGroupCount requires options.groupby');
});

test('repository read aggregate runtime readGroupCount fast path for single group returns 0 on NaN result', async () => {
  const { params, log } = createAggregateRuntime({
    async execute(query: any) {
      log.push({ method: 'execute', hasWhere: Boolean(query.whereArg), hasHaving: Boolean(query.havingArg) });
      return [{ Total: 'NaN' }] as any;
    },
  });

  const total = await executeRepositoryReadGroupCount(
    params as any,
    {
      groupby: 'Status',
      condition: ['Id', '!=', null],
    } as any
  );

  expect(total).toBe(0);
  expect(log.some(item => item.method === 'having')).toBe(false);
});

test('repository read aggregate runtime readGroupCount subquery path uses having aliases and outer count', async () => {
  const { params, log } = createAggregateRuntime({
    async execute(query: any) {
      const isOuter = query.tableOrAlias && query.tableOrAlias.kind === 'aliased-subquery' && query.tableOrAlias.alias === 't';
      log.push({ method: 'execute', isOuter });
      if (isOuter) return [{ Total: '4' }] as any;
      return [{ Status: 'done', Amount__sum: '10', __count: 1 }] as any;
    },
  });

  const total = await executeRepositoryReadGroupCount(
    params as any,
    {
      groupby: ['CreatedAt:month', 'Status'],
      fields: [{ field: 'Amount', agg: 'sum' }],
      condition: ['Id', '!=', null],
      having: ['Amount__sum', '>', 1],
    } as any
  );

  expect(total).toBe(4);

  const havingCall = log.find(item => item.method === 'having') as any;
  expect(havingCall.havingArg).toEqual({
    kind: 'having',
    condition: ['Amount__sum', '>', 1],
    knownAliases: ['Amount__sum', 'CreatedAt__month', 'Status', '__count'],
  });

  expect(log.some(item => item.method === 'selectFrom' && item.tableOrAlias && item.tableOrAlias.alias === 't')).toBe(true);
  expect(log.filter(item => item.method === 'execute').length).toBe(1);
});

test('repository read aggregate runtime readGroupCount fast path returns 0 when aggregate query yields no rows', async () => {
  const { params, log } = createAggregateRuntime({
    async execute(query: any) {
      log.push({ method: 'execute', hasWhere: Boolean(query.whereArg), hasHaving: Boolean(query.havingArg) });
      return [] as any;
    },
  });

  const total = await executeRepositoryReadGroupCount(
    params as any,
    {
      groupby: 'Status',
      condition: ['Id', '!=', null],
    } as any
  );

  expect(total).toBe(0);
  expect(log.some(item => item.method === 'having')).toBe(false);
});

test('repository read aggregate runtime readGroupCount composite without having still uses subquery and returns 0 on outer NaN', async () => {
  const { params, log } = createAggregateRuntime({
    async execute(query: any) {
      const isOuter = query.tableOrAlias && query.tableOrAlias.kind === 'aliased-subquery' && query.tableOrAlias.alias === 't';
      log.push({ method: 'execute', isOuter });
      if (isOuter) return [{ Total: 'NaN' }] as any;
      return [{ Status: 'done', Amount__sum: '10', __count: 1 }] as any;
    },
  });

  const total = await executeRepositoryReadGroupCount(
    params as any,
    {
      groupby: ['CreatedAt:month', 'Status'],
      fields: [{ field: 'Amount', agg: 'sum' }],
      condition: ['Id', '!=', null],
    } as any
  );

  expect(total).toBe(0);
  const havingCall = log.find(item => item.method === 'having') as any;
  expect(havingCall.havingArg).toEqual({
    kind: 'having',
    condition: undefined,
    knownAliases: ['Amount__sum', 'CreatedAt__month', 'Status', '__count'],
  });
});

test('repository read aggregate runtime readGroup uses nullish fallbacks for fields/condition and empty execute result', async () => {
  const { params, log } = createAggregateRuntime({
    ctx: {},
    async execute(query: any) {
      log.push({ method: 'execute', hasWhere: Boolean(query.whereArg), hasGroupBy: Boolean(query.groupByArg) });
      return undefined as any;
    },
  });

  const rows = await executeRepositoryReadGroup(
    params as any,
    {
      groupby: 'Status',
      timezone: 'UTC',
    } as any
  );

  expect(rows).toEqual([]);
  expect(log.some(item => item.method === 'where')).toBe(false);
});

test('repository read aggregate runtime readTotals supports undefined fields/condition and defaults __count on sparse row', async () => {
  const { params } = createAggregateRuntime({
    async execute() {
      return [{}] as any;
    },
  });

  const row = await executeRepositoryReadTotals(
    params as any,
    {
      fields: undefined,
      condition: undefined,
    } as any
  );

  expect(row).toEqual({ __count: 0 });
});

test('repository read aggregate runtime readGroupCount normalizes missing Total to zero for both fast and subquery paths', async () => {
  const fast = createAggregateRuntime({
    ctx: {},
    async execute() {
      return [{}] as any;
    },
  });

  const fastTotal = await executeRepositoryReadGroupCount(
    fast.params as any,
    {
      groupby: 'Status',
      condition: undefined,
    } as any
  );
  expect(fastTotal).toBe(0);

  const subquery = createAggregateRuntime({
    async execute(query: any) {
      const isOuter = query.tableOrAlias && query.tableOrAlias.kind === 'aliased-subquery' && query.tableOrAlias.alias === 't';
      if (isOuter) return [{}] as any;
      return [{ Status: 'done', __count: 1 }] as any;
    },
  });

  const subTotal = await executeRepositoryReadGroupCount(
    subquery.params as any,
    {
      groupby: ['CreatedAt:month', 'Status'],
      condition: undefined,
      having: ['__count', '>', 0],
    } as any
  );
  expect(subTotal).toBe(0);
});

test('repository read aggregate runtime readGroup supports composite groupby and row-level __count fallback', async () => {
  const { params } = createAggregateRuntime({
    async execute() {
      return [
        {
          CreatedAt__month: '2026-04',
          Status: 'draft',
          Amount__sum: '9.995',
        },
      ] as any;
    },
  });

  const rows = await executeRepositoryReadGroup(
    params as any,
    {
      groupby: ['CreatedAt:month', 'Status'],
      fields: [{ field: 'Amount', agg: 'sum' }],
      condition: ['Id', '!=', null],
    } as any
  );

  expect(rows).toEqual([
    {
      CreatedAt__month: '2026-04',
      Status: 'draft',
      Amount__sum: 9.99,
      __count: 0,
    },
  ]);
});

test('repository read aggregate runtime readGroupCount subquery path returns 0 when outer rows are empty', async () => {
  const { params } = createAggregateRuntime({
    async execute(query: any) {
      const isOuter = query.tableOrAlias && query.tableOrAlias.kind === 'aliased-subquery' && query.tableOrAlias.alias === 't';
      if (isOuter) return [] as any;
      return [{ Status: 'done', __count: 1 }] as any;
    },
  });

  const total = await executeRepositoryReadGroupCount(
    params as any,
    {
      groupby: ['CreatedAt:month', 'Status'],
      having: ['__count', '>', 0],
      condition: ['Id', '!=', null],
    } as any
  );

  expect(total).toBe(0);
});
