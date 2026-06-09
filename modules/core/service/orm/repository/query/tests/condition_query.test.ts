// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { countRepositoryConditionMatches, locateRepositoryIdsForCondition } from '..';

function createDbHarness() {
  const builders: any[] = [];

  const db = {
    fn: {
      countAll() {
        return {
          as(alias: string) {
            return { kind: 'countAll', alias };
          },
        };
      },
    },
    selectFrom(table: string) {
      const builder = {
        table,
        trace: [{ type: 'from', table }] as any[],
        select(selection: any) {
          this.trace.push({ type: 'select', selection });
          return this;
        },
        where(factory: any) {
          this.trace.push({ type: 'where', value: factory({ eb: 'EB' }) });
          return this;
        },
      };
      builders.push(builder);
      return builder;
    },
  };

  return { db, builders };
}

test('repository condition query locates ids with layered condition and compiled where clause', async () => {
  const { db, builders } = createDbHarness();
  const executeCalls: any[] = [];

  const ids = await locateRepositoryIdsForCondition(
    {
      db,
      table: 'demo_table',
      applyConditionLayers(condition) {
        return { And: [condition, ['DeletedAt', 'is', null] as any] } as any;
      },
      isEmptyCondition() {
        return false;
      },
      convertCondition(eb, condition, selfTable) {
        return { eb, condition, selfTable };
      },
      async execute(query) {
        executeCalls.push(query);
        return [{ Id: 'row_1' }, { Id: 2 }, { Id: '' }] as any;
      },
    },
    ['Status', '=', 'ready'] as any
  );

  expect(ids).toEqual(['row_1', '2']);
  expect(executeCalls).toEqual([builders[0]]);
  expect(builders[0].trace).toEqual([
    { type: 'from', table: 'demo_table' },
    { type: 'select', selection: 'Id' },
    {
      type: 'where',
      value: {
        eb: 'EB',
        condition: {
          And: [
            ['Status', '=', 'ready'],
            ['DeletedAt', 'is', null],
          ],
        },
        selfTable: 'demo_table',
      },
    },
  ]);
});

test('repository condition query counts rows and skips where when filtered condition is empty', async () => {
  const { db, builders } = createDbHarness();
  const convertCalls: any[] = [];

  const total = await countRepositoryConditionMatches(
    {
      db,
      table: 'demo_table',
      applyConditionLayers() {
        return [] as any;
      },
      isEmptyCondition(condition) {
        return Array.isArray(condition) && condition.length === 0;
      },
      convertCondition(eb, condition, selfTable) {
        convertCalls.push({ eb, condition, selfTable });
        return { eb, condition, selfTable };
      },
      async execute() {
        return [{ Total: '3' }] as any;
      },
    },
    ['Ignored', '=', true] as any
  );

  expect(total).toBe(3);
  expect(convertCalls).toEqual([]);
  expect(builders[0].trace).toEqual([
    { type: 'from', table: 'demo_table' },
    { type: 'select', selection: { kind: 'countAll', alias: 'Total' } },
  ]);
});

test('repository condition query normalizes empty execute results for locate and count', async () => {
  const { db, builders } = createDbHarness();
  let executeCount = 0;

  const deps = {
    db,
    table: 'demo_table',
    applyConditionLayers(condition: any) {
      return condition;
    },
    isEmptyCondition() {
      return true;
    },
    convertCondition() {
      return { shouldNotReach: true };
    },
    async execute() {
      executeCount += 1;
      if (executeCount === 1) return undefined as any;
      return [] as any;
    },
  };

  const ids = await locateRepositoryIdsForCondition(deps as any, [] as any);
  const total = await countRepositoryConditionMatches(deps as any, [] as any);

  expect(ids).toEqual([]);
  expect(total).toBe(0);
  expect(builders[0].trace).toEqual([
    { type: 'from', table: 'demo_table' },
    { type: 'select', selection: 'Id' },
  ]);
  expect(builders[1].trace).toEqual([
    { type: 'from', table: 'demo_table' },
    { type: 'select', selection: { kind: 'countAll', alias: 'Total' } },
  ]);
});
