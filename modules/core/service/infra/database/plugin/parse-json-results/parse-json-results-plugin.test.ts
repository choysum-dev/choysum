// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumParseJSONResultsPlugin } from './parse-json-results-plugin';

test('parse-json-results plugin returns input when rows are empty', async () => {
  const plugin = new ChoysumParseJSONResultsPlugin();
  const result = { rows: [] as any[] } as any;
  const out = await plugin.transformResult({ result } as any);
  expect(out).toBe(result);
});

test('parse-json-results plugin fast-skips already normalized relation payload', async () => {
  const plugin = new ChoysumParseJSONResultsPlugin();
  const result = {
    rows: [
      {
        $rel$Owner: { Id: 'u1', Name: 'Owner' },
        Name: 'row-1',
      },
    ],
  } as any;

  const out = await plugin.transformResult({ result } as any);
  expect(out).toBe(result);
  expect(out.rows[0].$rel$Owner).toEqual({ Id: 'u1', Name: 'Owner' });
});

test('parse-json-results plugin parses string relation json and normalizes keys/alias', async () => {
  const plugin = new ChoysumParseJSONResultsPlugin();
  const result = {
    rows: [
      {
        $rel$_owner_profile: '{"id":"u1","display_name":"U","tags":[{"tag_id":1}]}',
      },
    ],
  } as any;

  const out = await plugin.transformResult({ result } as any);
  expect(out.rows[0].$rel$OwnerProfile).toEqual({
    Id: 'u1',
    DisplayName: 'U',
    Tags: [{ TagId: 1 }],
  });
  expect('$rel$_owner_profile' in out.rows[0]).toBe(false);
});

test('parse-json-results plugin keeps non-json relation string as-is', async () => {
  const plugin = new ChoysumParseJSONResultsPlugin();
  const result = {
    rows: [
      {
        $rel$Owner: 'raw-text',
      },
    ],
  } as any;

  const out = await plugin.transformResult({ result } as any);
  expect(out.rows[0].$rel$Owner).toBe('raw-text');
});

test('parse-json-results plugin covers sparse rows, empty json hint, key migration collision and nested-array recursion', async () => {
  const plugin = new ChoysumParseJSONResultsPlugin();
  const result = {
    rows: [
      undefined,
      { Name: 'row-no-rel' },
      {
        $rel$OwnerProfile: { Id: 'keep-existing' },
        $rel$_owner_profile: '[1,[{"tag_id":1}],{"display_name":"U"}]',
        $rel$___: '',
      },
      {
        $rel$_owner_profile: '{"id":"u2","display_name":"U2"}',
      },
    ] as any,
  } as any;

  const out = await plugin.transformResult({ result } as any);

  expect(out.rows[2].$rel$OwnerProfile).toEqual({ Id: 'keep-existing' });
  expect('$rel$_owner_profile' in out.rows[2]).toBe(false);
  expect(out.rows[2].$rel$).toBe('');
  expect(out.rows[3].$rel$OwnerProfile).toEqual({ Id: 'u2', DisplayName: 'U2' });
});
