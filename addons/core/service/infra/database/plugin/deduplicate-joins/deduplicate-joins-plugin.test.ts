// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumDeduplicateJoinsPlugin } from './deduplicate-joins-plugin';

test('deduplicate joins plugin returns same result when rows exist', async () => {
  const plugin = new ChoysumDeduplicateJoinsPlugin();
  const result = {
    rows: [{ Id: '1' }],
  } as any;

  const out = await plugin.transformResult({ result } as any);
  expect(out).toBe(result);
});

test('deduplicate joins plugin returns same result when rows are missing', async () => {
  const plugin = new ChoysumDeduplicateJoinsPlugin();
  const result = {} as any;

  const out = await plugin.transformResult({ result } as any);
  expect(out).toBe(result);
});
