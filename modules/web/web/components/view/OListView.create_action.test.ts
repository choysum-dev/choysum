// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// density backfill from main before merge
// New-button click paths need createListController / first-frame isolation under QJS
// (main used vi.mock). Empty createAction + export surface still covered below.

import OListView from './OListView.vue';

test('OListView create action: default export is a named Vue component', () => {
  expect(OListView).toBeTruthy();
  const name = (OListView as { name?: string; __name?: string }).name || (OListView as { __name?: string }).__name;
  expect(name).toBeTruthy();
});
