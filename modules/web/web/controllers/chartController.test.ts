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

  it('does not set the clear flag when appliedFilters is omitted', async () => {
    const store = {
      state: {
        queryState: {
          appliedFilters: [{ id: 'f1', logic: 'And', children: [] }],
        },
      },
    } as any;
    const ctrl = createChartController(store);
    await ctrl.apply({});
    expect(store.state.queryState.userClearedDefaultFilters).toBeUndefined();
    expect(store.state.queryState.appliedFilters).toEqual([{ id: 'f1', logic: 'And', children: [] }]);
  });

  it('treats non-array prior appliedFilters as empty when clearing', async () => {
    const store = {
      state: {
        queryState: {
          appliedFilters: null,
        },
      },
    } as any;
    const ctrl = createChartController(store);
    await ctrl.apply({ appliedFilters: [] });
    expect(store.state.queryState.appliedFilters).toEqual([]);
    expect(store.state.queryState.userClearedDefaultFilters).toBeUndefined();
  });

  it('does not set the clear flag when replacing with a non-empty filter set', async () => {
    const store = {
      state: {
        queryState: {
          appliedFilters: [{ id: 'f1', logic: 'And', children: [] }],
        },
      },
    } as any;
    const ctrl = createChartController(store);
    const next = [{ id: 'f2', logic: 'Or', children: [] }];
    await ctrl.apply({ appliedFilters: next });
    expect(store.state.queryState.appliedFilters).toEqual(next);
    expect(store.state.queryState.userClearedDefaultFilters).toBeUndefined();
  });

  it('does not set the clear flag when appliedFilters override is non-array', async () => {
    const store = {
      state: {
        queryState: {
          appliedFilters: [{ id: 'f1', logic: 'And', children: [] }],
        },
      },
    } as any;
    const ctrl = createChartController(store);
    await ctrl.apply({ appliedFilters: null as any });
    expect(store.state.queryState.appliedFilters).toBeNull();
    expect(store.state.queryState.userClearedDefaultFilters).toBeUndefined();
  });
});
