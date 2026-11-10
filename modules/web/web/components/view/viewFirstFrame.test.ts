// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  finishInitialSearchBoundLoad,
  shouldDeferToSearchFirstFrame,
  shouldDeferViewFirstFrame,
} from './viewFirstFrame';

test('shouldDeferToSearchFirstFrame: search slot defers', () => {
  expect(shouldDeferToSearchFirstFrame({ hasSearchSlot: true })).toBe(true);
  expect(shouldDeferToSearchFirstFrame({ hasSearchSlot: false })).toBe(false);
});

test('shouldDeferToSearchFirstFrame: List identity match', () => {
  const oSearchView = { name: 'ChoySearchView' };
  expect(
    shouldDeferToSearchFirstFrame({ searchView: oSearchView, canonicalSearchView: oSearchView }),
  ).toBe(true);
  expect(
    shouldDeferToSearchFirstFrame({
      searchView: { name: 'ChoySearchView' },
      canonicalSearchView: oSearchView,
    }),
  ).toBe(false);
  expect(shouldDeferToSearchFirstFrame({ searchView: undefined, canonicalSearchView: oSearchView })).toBe(
    false,
  );
});

test('shouldDeferViewFirstFrame aliases identity check', () => {
  const oSearchView = { name: 'ChoySearchView' };
  expect(shouldDeferViewFirstFrame(oSearchView, oSearchView)).toBe(true);
  expect(shouldDeferViewFirstFrame({ name: 'ChoySearchView' }, oSearchView)).toBe(false);
});

test('finishInitialSearchBoundLoad: skips empty when search already landed', async () => {
  let applied = false;
  await finishInitialSearchBoundLoad({
    getLastSearchQuery: () => ({ keyword: 'x' }),
    applyEmpty: async () => {
      applied = true;
    },
  });
  expect(applied).toBe(false);
});

test('finishInitialSearchBoundLoad: deferToSearch skips empty apply', async () => {
  const calls: string[] = [];
  await finishInitialSearchBoundLoad({
    getLastSearchQuery: () => null,
    deferToSearch: true,
    applyEmpty: async () => {
      calls.push('empty');
    },
    afterEmpty: async () => {
      calls.push('after');
    },
  });
  expect(calls).toEqual([]);
});

test('finishInitialSearchBoundLoad: without defer applies empty then afterEmpty', async () => {
  const calls: string[] = [];
  await finishInitialSearchBoundLoad({
    getLastSearchQuery: () => null,
    applyEmpty: async () => {
      calls.push('empty');
    },
    afterEmpty: async () => {
      calls.push('after');
    },
  });
  expect(calls).toEqual(['empty', 'after']);
});
