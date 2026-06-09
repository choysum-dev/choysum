// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  applyRepositoryReadAggregateCondition,
  buildRepositoryReadAggregateGroupExprs,
  buildRepositoryReadAggregateSelections,
  buildRepositoryReadAggregateTotalSelections,
  normalizeRepositoryAggregateDecimals,
  resolveRepositoryReadAggregateKnownAliases,
} from '..';
import { MetadataStorage } from '../../../metadata/storage';

function withFakeMetadata<T>(metas: Map<Function, any>, fn: () => T): T {
  const storage = MetadataStorage.instance as any;
  const original = storage.getModelMetadata;
  storage.getModelMetadata = function (model: Function) {
    if (metas.has(model)) return metas.get(model);
    return original.call(this, model);
  };

  try {
    return fn();
  } finally {
    storage.getModelMetadata = original;
  }
}

function createFieldExpr(field: string) {
  return {
    kind: 'field-expr',
    field,
    as(alias: string) {
      return { kind: 'aliased-field', field, alias };
    },
  };
}

function createAggregateParams(overrides?: Partial<any>) {
  const calls: Array<Record<string, any>> = [];
  const params = {
    table: 'demo_table',
    meta: { type: class DemoModel {}, fields: new Map() } as any,
    getDialect() {
      calls.push({ method: 'getDialect' });
      return 'postgres' as const;
    },
    makeSelectCtx() {
      calls.push({ method: 'makeSelectCtx' });
      return {
        field(_type: any, field: string) {
          calls.push({ method: 'field', field });
          return createFieldExpr(field);
        },
      };
    },
    ...overrides,
  };

  return { params, calls };
}

test('repository read aggregate helpers build group expressions for composite time and non-time parts', () => {
  const { params, calls } = createAggregateParams();

  const exprs = buildRepositoryReadAggregateGroupExprs(
    params as any,
    { kind: 'builder' },
    {
      composite: true,
      parts: [
        { field: 'CreatedAt', alias: 'CreatedAt__month', granularity: 'month', isTime: true },
        { field: 'Status', alias: 'Status', isTime: false },
      ],
    },
    'Asia/Shanghai'
  );

  expect(exprs.length).toBe(2);
  expect(calls.filter(call => call.method === 'makeSelectCtx').length).toBe(2);
  expect(calls.filter(call => call.method === 'getDialect').length).toBe(1);
  expect(calls.filter(call => call.method === 'field').map(call => call.field)).toEqual(['CreatedAt', 'Status']);
});

test('repository read aggregate helpers build selections for grouped and total queries', () => {
  const { params } = createAggregateParams();

  const countAll = () => ({
    as(alias: string) {
      return { kind: 'count-all', alias };
    },
  });

  const groupedSelections = buildRepositoryReadAggregateSelections(
    params as any,
    { kind: 'builder' },
    { field: 'Status', alias: 'Status', isTime: false },
    [{ field: 'Amount', agg: 'count', alias: 'Amount__count', distinct: true } as any],
    countAll
  );

  expect(groupedSelections.length).toBe(3);
  expect(groupedSelections[0]).toEqual({ kind: 'aliased-field', field: 'Status', alias: 'Status' });
  expect(groupedSelections[2]).toEqual({ kind: 'count-all', alias: '__count' });

  const totalSelections = buildRepositoryReadAggregateTotalSelections(
    params as any,
    { kind: 'builder' },
    [{ field: 'Amount', agg: 'sum', alias: 'Amount__sum' } as any],
    countAll
  );

  expect(totalSelections.length).toBe(2);
  expect(totalSelections[1]).toEqual({ kind: 'count-all', alias: '__count' });
});

test('repository read aggregate helpers apply condition short-circuits on empty condition', () => {
  const calls: Array<Record<string, any>> = [];
  const query = {
    where() {
      calls.push({ method: 'where' });
      return { kind: 'should-not-be-used' };
    },
  };

  const result = applyRepositoryReadAggregateCondition(
    query as any,
    {
      table: 'demo_table',
      isEmptyCondition(condition: any) {
        calls.push({ method: 'isEmpty', condition });
        return true;
      },
      convertCondition() {
        calls.push({ method: 'convert' });
        return { kind: 'cond' };
      },
    },
    [] as any
  );

  expect(result).toBe(query);
  expect(calls).toEqual([{ method: 'isEmpty', condition: [] }]);
});

test('repository read aggregate helpers apply condition attaches where for non-empty condition', () => {
  const calls: Array<Record<string, any>> = [];
  const query = {
    where(callback: ({ eb }: any) => unknown) {
      const result = callback({ eb: 'EB' });
      calls.push({ method: 'where', result });
      return { kind: 'query-with-where' };
    },
  };

  const result = applyRepositoryReadAggregateCondition(
    query as any,
    {
      table: 'demo_table',
      isEmptyCondition(condition: any) {
        calls.push({ method: 'isEmpty', condition });
        return false;
      },
      convertCondition(eb: any, condition: any, selfTable?: string) {
        calls.push({ method: 'convert', eb, condition, selfTable });
        return { kind: 'converted', eb, condition, selfTable };
      },
    },
    ['Id', '=', '1'] as any
  );

  expect(result).toEqual({ kind: 'query-with-where' });
  expect(calls).toEqual([
    { method: 'isEmpty', condition: ['Id', '=', '1'] },
    {
      method: 'convert',
      eb: 'EB',
      condition: ['Id', '=', '1'],
      selfTable: 'demo_table',
    },
    {
      method: 'where',
      result: {
        kind: 'converted',
        eb: 'EB',
        condition: ['Id', '=', '1'],
        selfTable: 'demo_table',
      },
    },
  ]);
});

test('repository read aggregate helpers resolve known aliases for simple and composite group specs', () => {
  const simple = resolveRepositoryReadAggregateKnownAliases({ field: 'Status', alias: 'Status', isTime: false } as any, [
    { alias: 'Amount__sum' },
    { alias: 'Amount__avg' },
  ]);

  expect(simple).toEqual(new Set(['Status', '__count', 'Amount__sum', 'Amount__avg']));

  const composite = resolveRepositoryReadAggregateKnownAliases(
    {
      composite: true,
      parts: [{ alias: 'CreatedAt__month' }, { alias: 'Status' }],
    } as any,
    [{ alias: 'Amount__sum' }]
  );

  expect(composite).toEqual(new Set(['CreatedAt__month', 'Status', '__count', 'Amount__sum']));
});

test('repository read aggregate helpers normalize decimal aggregations for direct and relation paths', () => {
  class PartnerModel {}

  const partnerMeta = {
    fields: new Map([
      ['Credit', { type: 'decimal', column: { scale: 1 } }],
      ['Label', { type: 'char' }],
    ]),
  } as any;

  const meta = {
    fields: new Map([
      ['Amount', { type: 'decimal', column: { scale: 2 } }],
      [
        'Partner',
        {
          type: 'ManyToOne',
          relation: {
            targetModel: () => PartnerModel,
          },
        },
      ],
    ]),
  } as any;

  const rows = [
    {
      Amount__sum: '10.126',
      Amount__count: '7',
      PartnerCredit__avg: '3.1415',
      PartnerLabel__max: 'x',
    },
    {
      Amount__sum: 2n,
      PartnerCredit__avg: 'not-a-number',
    },
  ] as any[];

  withFakeMetadata(new Map([[PartnerModel, partnerMeta]]), () => {
    const normalized = normalizeRepositoryAggregateDecimals(meta, rows, [
      { field: 'Amount', agg: 'sum', alias: 'Amount__sum' },
      { field: 'Amount', agg: 'count', alias: 'Amount__count' },
      { field: 'Partner.Credit', agg: 'avg', alias: 'PartnerCredit__avg' },
      { field: 'Partner.Label', agg: 'max', alias: 'PartnerLabel__max' },
    ] as any);

    expect(normalized).toBe(rows);
    expect(rows[0]).toEqual({
      Amount__sum: 10.13,
      Amount__count: '7',
      PartnerCredit__avg: 3.1,
      PartnerLabel__max: 'x',
    });
    expect(rows[1]).toEqual({
      Amount__sum: 2,
      PartnerCredit__avg: 'not-a-number',
    });
  });
});

test('repository read aggregate helpers leave rows unchanged when input is empty', () => {
  const rows: any[] = [];
  const normalized = normalizeRepositoryAggregateDecimals({ fields: new Map([['Amount', { type: 'decimal', column: { scale: 2 } }]]) } as any, rows, [
    { field: 'Amount', agg: 'sum', alias: 'Amount__sum' } as any,
  ]);

  expect(normalized).toBe(rows);
  expect(rows).toEqual([]);
});

test('repository read aggregate helpers build count distinct expression from agg kind', () => {
  const { params } = createAggregateParams();

  const countAll = () => ({
    as(alias: string) {
      return { kind: 'count-all', alias };
    },
  });

  const groupedSelections = buildRepositoryReadAggregateSelections(
    params as any,
    { kind: 'builder' },
    { field: 'Status', alias: 'Status', isTime: false },
    [{ field: 'Amount', agg: 'count_distinct', alias: 'Amount__count_distinct' } as any],
    countAll
  );

  expect(groupedSelections.length).toBe(3);
  expect(groupedSelections[1]).toBeTruthy();
  expect(groupedSelections[2]).toEqual({ kind: 'count-all', alias: '__count' });
});

test('repository read aggregate helpers normalize decimal numeric strings without scale to number', () => {
  const meta = {
    fields: new Map([
      ['Amount', { type: 'decimal', column: {} }],
      ['Label', { type: 'char' }],
    ]),
  } as any;

  const rows = [
    {
      Amount__avg: '9.125',
      Label__max: 'n1',
    },
  ] as any[];

  const normalized = normalizeRepositoryAggregateDecimals(meta, rows, [
    { field: 'Amount', agg: 'avg', alias: 'Amount__avg' },
    { field: 'Label', agg: 'max', alias: 'Label__max' },
  ] as any);

  expect(normalized).toBe(rows);
  expect(rows[0]).toEqual({ Amount__avg: 9.125, Label__max: 'n1' });
});

test('repository read aggregate helpers build total selections for min and max aggregates', () => {
  const { params } = createAggregateParams();

  const countAll = () => ({
    as(alias: string) {
      return { kind: 'count-all', alias };
    },
  });

  const totalSelections = buildRepositoryReadAggregateTotalSelections(
    params as any,
    { kind: 'builder' },
    [{ field: 'Amount', agg: 'min', alias: 'Amount__min' } as any, { field: 'Amount', agg: 'max', alias: 'Amount__max' } as any],
    countAll
  );

  expect(totalSelections.length).toBe(3);
  expect(totalSelections[2]).toEqual({ kind: 'count-all', alias: '__count' });
});

test('repository read aggregate helpers keep rows unchanged when dotted decimal paths cannot resolve leaf meta', () => {
  const meta = {
    fields: new Map([
      ['Amount', { type: 'decimal', column: { scale: 2 } }],
      ['Owner', { type: 'ManyToOne', relation: {} }],
      ['Label', { type: 'char' }],
    ]),
  } as any;

  const rows = [{ Amount__sum: '10.11', Unknown__avg: '5.5' }] as any[];

  const normalized = normalizeRepositoryAggregateDecimals(meta, rows, [
    { field: '.', agg: 'avg', alias: 'Dot__avg' },
    { field: 'Missing.Path', agg: 'avg', alias: 'Missing__avg' },
    { field: 'Label.Path', agg: 'avg', alias: 'LabelPath__avg' },
    { field: 'Owner.Credit', agg: 'avg', alias: 'OwnerCredit__avg' },
    { field: 'Amount', agg: 'count', alias: 'Amount__count' },
  ] as any);

  expect(normalized).toBe(rows);
  expect(rows).toEqual([{ Amount__sum: '10.11', Unknown__avg: '5.5' }]);
});

test('repository read aggregate helpers build group expression for non-composite time group', () => {
  const { params, calls } = createAggregateParams();

  const exprs = buildRepositoryReadAggregateGroupExprs(
    params as any,
    { kind: 'builder' },
    { field: 'CreatedAt', alias: 'CreatedAt__day', granularity: 'day', isTime: true },
    'Asia/Shanghai'
  );

  expect(exprs.length).toBe(1);
  expect(calls.filter(call => call.method === 'getDialect').length).toBe(1);
  expect(calls.filter(call => call.method === 'field').map(call => call.field)).toEqual(['CreatedAt']);
});

test('repository read aggregate helpers normalize decimal when aggregate value is number', () => {
  const meta = {
    fields: new Map([['Amount', { type: 'decimal', column: { scale: 2 } }]]),
  } as any;

  const rows = [{ Amount__sum: 12.3456 }] as any[];
  const normalized = normalizeRepositoryAggregateDecimals(meta, rows, [{ field: 'Amount', agg: 'sum', alias: 'Amount__sum' }] as any);

  expect(normalized).toBe(rows);
  expect(rows).toEqual([{ Amount__sum: 12.35 }]);
});

test('repository read aggregate helpers fallback unknown aggregate kind to count expression', () => {
  const { params } = createAggregateParams();

  const countAll = () => ({
    as(alias: string) {
      return { kind: 'count-all', alias };
    },
  });

  const selections = buildRepositoryReadAggregateSelections(
    params as any,
    { kind: 'builder' },
    { field: 'Status', alias: 'Status', isTime: false },
    [{ field: 'Amount', agg: 'unknown_agg' as any, alias: 'Amount__unknown' } as any],
    countAll
  );

  expect(selections.length).toBe(3);
  expect(selections[0]).toEqual({ kind: 'aliased-field', field: 'Status', alias: 'Status' });
  expect(selections[2]).toEqual({ kind: 'count-all', alias: '__count' });
});

test('repository read aggregate helpers handle dotted agg path getter that resolves to undefined on second read', () => {
  const meta = {
    fields: new Map([['Amount', { type: 'decimal', column: { scale: 2 } }]]),
  } as any;

  let access = 0;
  const trickyAgg = {
    get field() {
      access += 1;
      return access === 1 ? 'Owner.Credit' : undefined;
    },
    agg: 'avg',
    alias: 'OwnerCredit__avg',
  } as any;

  const rows = [{ OwnerCredit__avg: '3.14' }] as any[];
  const normalized = normalizeRepositoryAggregateDecimals(meta, rows, [trickyAgg]);

  expect(normalized).toBe(rows);
  expect(rows).toEqual([{ OwnerCredit__avg: '3.14' }]);
});

test('repository read aggregate helpers keep null aggregate value unchanged', () => {
  const meta = {
    fields: new Map([['Amount', { type: 'decimal', column: { scale: 2 } }]]),
  } as any;

  const rows = [{ Amount__sum: null }] as any[];
  const normalized = normalizeRepositoryAggregateDecimals(meta, rows, [{ field: 'Amount', agg: 'sum', alias: 'Amount__sum' }] as any);

  expect(normalized).toBe(rows);
  expect(rows).toEqual([{ Amount__sum: null }]);
});

test('repository read aggregate helpers build grouped selections with avg aggregate', () => {
  const { params } = createAggregateParams();

  const countAll = () => ({
    as(alias: string) {
      return { kind: 'count-all', alias };
    },
  });

  const groupedSelections = buildRepositoryReadAggregateSelections(
    params as any,
    { kind: 'builder' },
    { field: 'Status', alias: 'Status', isTime: false },
    [{ field: 'Amount', agg: 'avg', alias: 'Amount__avg' } as any],
    countAll
  );

  expect(groupedSelections.length).toBe(3);
  expect(groupedSelections[0]).toEqual({ kind: 'aliased-field', field: 'Status', alias: 'Status' });
  expect(groupedSelections[2]).toEqual({ kind: 'count-all', alias: '__count' });
});
