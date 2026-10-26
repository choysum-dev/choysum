// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  O2M_KANBAN_VIRTUAL_THRESHOLD,
  o2mKanbanColumnCount,
  o2mKanbanRowCardIndices,
  o2mKanbanRowCount,
  o2mKanbanShouldVirtualize,
  resolveO2MKanbanEstimateCardSize,
} from './o2mKanbanVirtualHelpers';

describe('o2mKanbanVirtualHelpers', () => {
  test('column count follows auto-fill math', () => {
    expect(o2mKanbanColumnCount({ boardWidth: 0, minCardWidth: 220, gap: 10 })).toBe(1);
    expect(o2mKanbanColumnCount({ boardWidth: 220, minCardWidth: 220, gap: 10 })).toBe(1);
    // 220 + 10 + 220 = 450 → two tracks fit exactly.
    expect(o2mKanbanColumnCount({ boardWidth: 450, minCardWidth: 220, gap: 10 })).toBe(2);
    expect(o2mKanbanColumnCount({ boardWidth: 449, minCardWidth: 220, gap: 10 })).toBe(1);
    expect(o2mKanbanColumnCount({ boardWidth: 700, minCardWidth: 220, gap: 10 })).toBe(3);
  });

  test('row count and card indices cover wrap grid', () => {
    expect(o2mKanbanRowCount(0, 3)).toBe(0);
    expect(o2mKanbanRowCount(5, 3)).toBe(2);
    expect(o2mKanbanRowCardIndices(0, 3, 5)).toEqual([0, 1, 2]);
    expect(o2mKanbanRowCardIndices(1, 3, 5)).toEqual([3, 4]);
    expect(o2mKanbanRowCardIndices(2, 3, 5)).toEqual([]);
  });

  test('auto virtualize uses the shared threshold', () => {
    expect(O2M_KANBAN_VIRTUAL_THRESHOLD).toBe(40);
    expect(o2mKanbanShouldVirtualize('auto', 39)).toBe(false);
    expect(o2mKanbanShouldVirtualize('auto', 40)).toBe(true);
    expect(o2mKanbanShouldVirtualize(true, 1)).toBe(true);
    expect(o2mKanbanShouldVirtualize(false, 100)).toBe(false);
    expect(resolveO2MKanbanEstimateCardSize(undefined)).toBe(88);
    expect(resolveO2MKanbanEstimateCardSize(120)).toBe(120);
  });
});
