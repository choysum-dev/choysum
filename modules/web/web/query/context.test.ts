// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { buildUnifiedQuery } from './context';

function makeStore(queryState: Record<string, unknown> = {}) {
  return {
    storeId: 'demo.Task',
    fieldsMetadata: {},
    state: {
      queryState: {
        appliedFilters: [],
        pagination: { limit: 40, offset: 0 },
        ...queryState,
      },
    },
  } as any;
}

describe('buildUnifiedQuery condition merge', () => {
  it('merges ui, forced, and parent conditions via combinePresentConditions', () => {
    const store = makeStore({
      forcedCondition: { Status: 'open' },
    });
    const ctx = buildUnifiedQuery(store, { parentCondition: { AssigneeId: 'u1' } });
    expect(ctx.filters).toEqual({
      And: [{ Status: 'open' }, { AssigneeId: 'u1' }],
    });
  });

  it('preserves present false/0 operands when merging', () => {
    const store = makeStore({
      forcedCondition: false as any,
    });
    const ctx = buildUnifiedQuery(store, { parentCondition: { A: 1 } });
    expect(ctx.filters).toEqual({ And: [false, { A: 1 }] });
  });
});
