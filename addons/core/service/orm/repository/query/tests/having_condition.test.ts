// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { convertRepositoryHavingCondition } from '..';

function createExpressionBuilder() {
  const eb: any = (lhs: any, op: any, rhs: any) => ({ kind: 'cmp', lhs, op, rhs });
  eb.ref = (alias: string) => ({ kind: 'ref', alias });
  eb.and = (parts: any[]) => ({ kind: 'and', parts });
  eb.or = (parts: any[]) => ({ kind: 'or', parts });
  return eb;
}

test('repository having condition resolves known aliases and normalizes null operators', () => {
  const eb = createExpressionBuilder();

  expect(
    convertRepositoryHavingCondition(
      {
        convertCondition() {
          throw new Error('should not fall back when alias is known');
        },
        selfTable: 'demo_table',
      },
      eb,
      ['__count', '=', null] as any,
      new Set(['__count'])
    )
  ).toEqual({ kind: 'cmp', lhs: { kind: 'ref', alias: '__count' }, op: 'is', rhs: null });

  expect(
    convertRepositoryHavingCondition(
      {
        convertCondition() {
          throw new Error('should not fall back when alias is known');
        },
        selfTable: 'demo_table',
      },
      eb,
      ['totalAmount', '!=', null] as any,
      new Set(['totalAmount'])
    )
  ).toEqual({ kind: 'cmp', lhs: { kind: 'ref', alias: 'totalAmount' }, op: 'is not', rhs: null });
});

test('repository having condition falls back to condition compiler for non-alias predicates and preserves logical groups', () => {
  const eb = createExpressionBuilder();
  const calls: Array<Record<string, any>> = [];

  const result = convertRepositoryHavingCondition(
    {
      convertCondition(builder, condition, selfTable) {
        calls.push({ builder, condition, selfTable });
        return { kind: 'fallback', condition, selfTable };
      },
      selfTable: 'demo_table',
    },
    eb,
    {
      And: [
        ['sumAmount', '>', 10],
        {
          Or: [
            ['Status', '=', 'ready'],
            ['__count', '>=', 2],
          ],
        },
      ],
    } as any,
    new Set(['sumAmount', '__count'])
  );

  expect(result).toEqual({
    kind: 'and',
    parts: [
      { kind: 'cmp', lhs: { kind: 'ref', alias: 'sumAmount' }, op: '>', rhs: 10 },
      {
        kind: 'or',
        parts: [
          { kind: 'fallback', condition: ['Status', '=', 'ready'], selfTable: 'demo_table' },
          { kind: 'cmp', lhs: { kind: 'ref', alias: '__count' }, op: '>=', rhs: 2 },
        ],
      },
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

test('repository having condition throws on invalid tuple length and falls back unknown envelope to empty-and', () => {
  const eb = createExpressionBuilder();

  let message = '';
  try {
    convertRepositoryHavingCondition(
      {
        convertCondition() {
          throw new Error('should not reach convertCondition');
        },
        selfTable: 'demo_table',
      },
      eb,
      ['__count', '='] as any,
      new Set(['__count'])
    );
  } catch (error) {
    message = String((error as Error)?.message || error);
  }
  expect(message).toBe('invalid condition tuple length in HAVING: 2');

  expect(
    convertRepositoryHavingCondition(
      {
        convertCondition() {
          throw new Error('should not reach convertCondition');
        },
        selfTable: 'demo_table',
      },
      eb,
      { Unknown: true } as any,
      new Set(['__count'])
    )
  ).toEqual({ kind: 'and', parts: [] });
});

test('repository having condition treats empty tuple as empty-and and normalizes == / <> with null', () => {
  const eb = createExpressionBuilder();

  expect(
    convertRepositoryHavingCondition(
      {
        convertCondition() {
          throw new Error('should not be called for empty tuple');
        },
      },
      eb,
      [] as any,
      new Set(['__count'])
    )
  ).toEqual({ kind: 'and', parts: [] });

  expect(
    convertRepositoryHavingCondition(
      {
        convertCondition() {
          throw new Error('should not fall back when alias is known');
        },
      },
      eb,
      ['__count', '==', null] as any,
      new Set(['__count'])
    )
  ).toEqual({ kind: 'cmp', lhs: { kind: 'ref', alias: '__count' }, op: 'is', rhs: null });

  expect(
    convertRepositoryHavingCondition(
      {
        convertCondition() {
          throw new Error('should not fall back when alias is known');
        },
      },
      eb,
      ['__count', '<>', null] as any,
      new Set(['__count'])
    )
  ).toEqual({ kind: 'cmp', lhs: { kind: 'ref', alias: '__count' }, op: 'is not', rhs: null });
});

test('repository having condition keeps falsy operator input without normalization when alias is known', () => {
  const eb = createExpressionBuilder();

  expect(
    convertRepositoryHavingCondition(
      {
        convertCondition() {
          throw new Error('should not fall back when alias is known');
        },
      },
      eb,
      ['__count', undefined, null] as any,
      new Set(['__count'])
    )
  ).toEqual({ kind: 'cmp', lhs: { kind: 'ref', alias: '__count' }, op: undefined, rhs: null });
});
