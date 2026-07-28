// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { applyOrderByToQuery, computeFallbackOrder, normalizeOrderBy, resolveEffectiveOrder } from '..';

test('repository ordering normalizes inputs and falls back to Id desc', () => {
  expect(normalizeOrderBy(undefined as any)).toBe(undefined);
  expect(normalizeOrderBy([{ field: 'Name', order: 'DESC' }, { field: 'CreatedAt' }])).toEqual([
    { field: 'Name', order: 'desc' },
    { field: 'CreatedAt', order: 'asc' },
  ]);

  const meta = { fields: new Map(), type: { name: 'Demo' } } as any;
  expect(computeFallbackOrder(meta)).toEqual([{ field: 'Id', order: 'desc' }]);
  expect(resolveEffectiveOrder(undefined, undefined, meta)).toEqual([{ field: 'Id', order: 'desc' }]);
  expect(resolveEffectiveOrder([{ field: 'Name', order: 'asc' }], [{ field: 'Id', order: 'desc' }], meta)).toEqual([{ field: 'Name', order: 'asc' }]);
});

test('repository ordering applies alias, path, select and scalar ordering boundaries', () => {
  const calls: Array<Record<string, any>> = [];
  const query = {
    orderBy(arg: any, direction: string) {
      calls.push({ arg, direction });
      return this;
    },
  };

  const meta = {
    fields: new Map([
      ['DisplayName', {}],
      ['CreatedAt', { column: { name: 'CreatedAt' } }],
    ]),
    sqlComputeHandlers: new Map([['DisplayName', { field: 'DisplayName', method: 'sqlDisplayName' }]]),
  } as any;

  applyOrderByToQuery(
    query as any,
    meta,
    'demo_table',
    [
      { field: '__count', order: 'desc' },
      { field: 'Profile.Name', order: 'asc' },
      { field: 'DisplayName', order: 'asc' },
      { field: 'CreatedAt', order: 'desc' },
    ],
    {
      resolvePathField(builder, field) {
        return `path:${field}:${builder.kind}`;
      },
      resolveSelectField(builder, field) {
        return `select:${field}:${builder.kind}`;
      },
    }
  );

  expect(calls.length).toBe(4);
  expect(calls[0]).toEqual({ arg: '__count', direction: 'desc' });
  expect(typeof calls[1].arg).toBe('function');
  expect(typeof calls[2].arg).toBe('function');
  expect(calls[3]).toEqual({ arg: 'demo_table.CreatedAt', direction: 'desc' });
  expect(calls[1].arg({ kind: 'path' })).toBe('path:Profile.Name:path');
  expect(calls[2].arg({ kind: 'select' })).toBe('select:DisplayName:select');
});

test('repository ordering ignores invalid items and uses meta order when override is empty', () => {
  expect(normalizeOrderBy('bad-input' as any)).toBe(undefined);
  expect(normalizeOrderBy([{ order: 'desc' } as any, null as any, { field: '', order: 'desc' } as any])).toBe(undefined);

  const meta = { fields: new Map(), type: { name: 'Demo' } } as any;
  expect(resolveEffectiveOrder(undefined, [{ field: 'CreatedAt', order: 'desc' }], meta)).toEqual([{ field: 'CreatedAt', order: 'desc' }]);
});

test('repository ordering returns original query when order list is empty', () => {
  const calls: Array<Record<string, any>> = [];
  const query = {
    orderBy(arg: any, direction: string) {
      calls.push({ arg, direction });
      return this;
    },
  };

  const returned = applyOrderByToQuery(query as any, { fields: new Map(), type: { name: 'Demo' } } as any, 'demo_table', [], {
    resolvePathField() {
      throw new Error('should not resolve path for empty order list');
    },
    resolveSelectField() {
      throw new Error('should not resolve select for empty order list');
    },
  });

  expect(returned).toBe(query);
  expect(calls).toEqual([]);
});

test('repository ordering keeps raw field when field metadata is missing', () => {
  const calls: Array<Record<string, any>> = [];
  const query = {
    orderBy(arg: any, direction: string) {
      calls.push({ arg, direction });
      return this;
    },
  };

  applyOrderByToQuery(
    query as any,
    {
      fields: new Map([['Known', { column: { name: 'Known' } }]]),
      type: { name: 'Demo' },
    } as any,
    'demo_table',
    [{ field: 'Unknown', order: 'asc' }],
    {
      resolvePathField() {
        throw new Error('path resolution should not be used for unknown scalar field');
      },
      resolveSelectField() {
        throw new Error('select resolution should not be used for unknown scalar field');
      },
    }
  );

  expect(calls).toEqual([{ arg: 'Unknown', direction: 'asc' }]);
});

test('repository ordering unwraps translated scalar fields', () => {
  const calls: Array<Record<string, any>> = [];
  const query = {
    orderBy(arg: any, direction: string) {
      calls.push({ arg, direction });
      return this;
    },
  };

  applyOrderByToQuery(
    query as any,
    {
      fields: new Map([['Name', { translate: true, column: { name: 'Name' } }]]),
      type: { name: 'Demo' },
    } as any,
    'demo_table',
    [{ field: 'Name', order: 'asc' }],
    {
      resolvePathField() {
        throw new Error('path resolution should not be used for translated scalar');
      },
      resolveSelectField() {
        throw new Error('select resolution should not be used for translated scalar');
      },
      getDialect: () => 'postgres',
    }
  );

  expect(calls.length).toBe(1);
  expect(calls[0].direction).toBe('asc');
  expect(typeof calls[0].arg).toBe('function');
  const eb: any = (lhs: any, op: any, rhs: any) => ({ lhs, op, rhs });
  eb.ref = (path: string) => ({ kind: 'ref', path });
  const expr = calls[0].arg(eb);
  expect(typeof (expr as any).toOperationNode).toBe('function');
});

test('repository ordering unwraps companyDependent scalar fields', () => {
  const calls: Array<Record<string, any>> = [];
  const query = {
    orderBy(arg: any, direction: string) {
      calls.push({ arg, direction });
      return this;
    },
  };

  applyOrderByToQuery(
    query as any,
    {
      fields: new Map([['Cost', { companyDependent: true, column: { name: 'Cost' } }]]),
      type: { name: 'Demo' },
    } as any,
    'demo_table',
    [{ field: 'Cost', order: 'desc' }],
    {
      resolvePathField() {
        throw new Error('path resolution should not be used for companyDependent scalar');
      },
      resolveSelectField() {
        throw new Error('select resolution should not be used for companyDependent scalar');
      },
      getDialect: () => 'postgres',
    }
  );

  expect(calls.length).toBe(1);
  expect(calls[0].direction).toBe('desc');
  expect(typeof calls[0].arg).toBe('function');
  const eb: any = (lhs: any, op: any, rhs: any) => ({ lhs, op, rhs });
  eb.ref = (path: string) => ({ kind: 'ref', path });
  const expr = calls[0].arg(eb);
  expect(typeof (expr as any).toOperationNode).toBe('function');
});
