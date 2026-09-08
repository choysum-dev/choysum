// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/** FE unit stubs for page-context / list-view composables. */
export function resolvePageStore() {
  return {
    $id: 'fe-stub-page-store',
    records: {},
  };
}

export function useListViewExpose() {
  return {};
}

export function provideOPageContext() {}

export default {
  resolvePageStore: resolvePageStore,
  useListViewExpose: useListViewExpose,
  provideOPageContext: provideOPageContext,
};
