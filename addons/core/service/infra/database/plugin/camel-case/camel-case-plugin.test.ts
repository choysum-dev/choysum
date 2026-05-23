// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumCamelCasePlugin } from './camel-case-plugin';

test('camel-case plugin returns input when rows are empty', async () => {
  const plugin = new ChoysumCamelCasePlugin();
  const result = { rows: [] as any[] } as any;
  const out = await plugin.transformResult({ result } as any);
  expect(out).toBe(result);
});

test('camel-case plugin fast-skips rows already in pascal-case', async () => {
  const plugin = new ChoysumCamelCasePlugin();
  const result = {
    rows: [
      {
        Id: '1',
        Name: 'A',
        $rel$Owner: { Id: 'u1' },
      },
    ],
  } as any;

  const out = await plugin.transformResult({ result } as any);
  expect(out).toBe(result);
});

test('camel-case plugin transforms snake keys and keeps special key formats', async () => {
  const plugin = new ChoysumCamelCasePlugin();
  const result = {
    rows: [
      {
        id: '1',
        total_amount: 10,
        total_amount__sum: 20,
        __count: 2,
        $rel$Owner: { Id: 'u1' },
      },
    ],
  } as any;

  const out = await plugin.transformResult({ result } as any);
  const row = out.rows[0] as any;

  expect(row.Id).toBe('1');
  expect(row.TotalAmount).toBe(10);
  expect(row.TotalAmount__sum).toBe(20);
  expect(row.__count).toBe(2);
  expect(row.$rel$Owner).toEqual({ Id: 'u1' });
});

test('camel-case plugin covers empty key, cache hit and upper-first with underscore boundary', async () => {
  const plugin = new ChoysumCamelCasePlugin();
  const result = {
    rows: [
      {
        '': 'empty',
        name: 'n1',
        Ab_c: 1,
      },
      {
        name: 'n2',
      },
    ],
  } as any;

  const out = await plugin.transformResult({ result } as any);
  const r0 = out.rows[0] as any;
  const r1 = out.rows[1] as any;

  expect(r0['']).toBe('empty');
  expect(r0.Name).toBe('n1');
  expect(r0.AbC).toBe(1);
  expect(r1.Name).toBe('n2');
});

test('camel-case plugin processes mixed pascal and snake keys in one row', async () => {
  const plugin = new ChoysumCamelCasePlugin();
  const result = {
    rows: [
      {
        Id: '1',
        snake_key: 'x',
      },
    ],
  } as any;

  const out = await plugin.transformResult({ result } as any);
  expect((out.rows[0] as any).Id).toBe('1');
  expect((out.rows[0] as any).SnakeKey).toBe('x');
});
