// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { createChartController } from './chartController';

describe('createChartController appliedFilters clear flag', () => {
  it('sets userClearedDefaultFilters when appliedFilters clears a non-empty prior set', async () => {
    const store = {
      state: {
        queryState: {
          appliedFilters: [{ id: 'f1', logic: 'And', children: [] }],
        },
      },
    } as any;
    const ctrl = createChartController(store);
    await ctrl.apply({ appliedFilters: [] });
    expect(store.state.queryState.appliedFilters).toEqual([]);
    expect(store.state.queryState.userClearedDefaultFilters).toBe(true);
  });

  it('does not set the clear flag when previous filters were already empty', async () => {
    const store = {
      state: {
        queryState: {
          appliedFilters: [],
        },
      },
    } as any;
    const ctrl = createChartController(store);
    await ctrl.apply({ appliedFilters: [] });
    expect(store.state.queryState.userClearedDefaultFilters).toBeUndefined();
  });
});
