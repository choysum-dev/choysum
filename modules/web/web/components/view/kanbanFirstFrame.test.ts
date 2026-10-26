// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { shouldDeferViewFirstFrame } from './kanbanFirstFrame';

test('shouldDeferViewFirstFrame > is true only when searchView is the SearchView component reference', () => {
  const oSearchView = { name: 'ChoySearchView' };
  expect(shouldDeferViewFirstFrame(oSearchView, oSearchView)).toBe(true);
  expect(shouldDeferViewFirstFrame({ name: 'ChoySearchView' }, oSearchView)).toBe(false);
  expect(shouldDeferViewFirstFrame(undefined, oSearchView)).toBe(false);
});

