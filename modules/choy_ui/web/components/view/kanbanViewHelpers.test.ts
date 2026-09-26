// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  applyChoyKanbanMove,
  groupRowsIntoChoyKanbanLanes,
  normalizeChoyKanbanLaneKey,
  resolveChoyKanbanCardId,
} from './kanbanViewHelpers';

describe('kanbanViewHelpers', () => {
  test('normalizeChoyKanbanLaneKey trims and falls back', () => {
    expect(normalizeChoyKanbanLaneKey('  done  ')).toBe('done');
    expect(normalizeChoyKanbanLaneKey('')).toBe('__unset__');
    expect(normalizeChoyKanbanLaneKey(null)).toBe('__unset__');
  });

  test('resolveChoyKanbanCardId prefers Id then id', () => {
    expect(resolveChoyKanbanCardId({ Id: 'a1' }, 0)).toBe('a1');
    expect(resolveChoyKanbanCardId({ id: 'b2' }, 0)).toBe('b2');
    expect(resolveChoyKanbanCardId({}, 3)).toBe('__row_3');
  });

  test('groupRowsIntoChoyKanbanLanes respects laneDefs order', () => {
    const lanes = groupRowsIntoChoyKanbanLanes(
      [
        { Id: '1', Title: 'A', State: 'done' },
        { Id: '2', Title: 'B', State: 'draft' },
        { Id: '3', Title: 'C', State: 'mystery' },
      ],
      {
        laneField: 'State',
        laneDefs: [
          { key: 'draft', label: 'Draft' },
          { key: 'done', label: 'Done' },
        ],
        titleField: 'Title',
      },
    );
    expect(lanes.map(l => l.key)).toEqual(['draft', 'done', 'mystery']);
    expect(lanes[0]!.cards.map(c => c.id)).toEqual(['2']);
    expect(lanes[1]!.cards.map(c => c.id)).toEqual(['1']);
    expect(lanes[2]!.label).toBe('mystery');
  });

  test('applyChoyKanbanMove moves across lanes', () => {
    const lanes = groupRowsIntoChoyKanbanLanes(
      [
        { Id: '1', Title: 'A', State: 'draft' },
        { Id: '2', Title: 'B', State: 'done' },
      ],
      {
        laneField: 'State',
        laneDefs: [
          { key: 'draft', label: 'Draft' },
          { key: 'done', label: 'Done' },
        ],
      },
    );
    const next = applyChoyKanbanMove(lanes, {
      cardId: '1',
      fromLaneKey: 'draft',
      toLaneKey: 'done',
      toIndex: 0,
    });
    expect(next).not.toBeNull();
    expect(next![0]!.cards).toHaveLength(0);
    expect(next![1]!.cards.map(c => c.id)).toEqual(['1', '2']);
    expect(next![1]!.cards[0]!.laneKey).toBe('done');
  });

  test('applyChoyKanbanMove returns null for no-op same index', () => {
    const lanes = groupRowsIntoChoyKanbanLanes(
      [{ Id: '1', Title: 'A', State: 'draft' }],
      { laneField: 'State', laneDefs: [{ key: 'draft', label: 'Draft' }] },
    );
    expect(
      applyChoyKanbanMove(lanes, {
        cardId: '1',
        fromLaneKey: 'draft',
        toLaneKey: 'draft',
        toIndex: 0,
      }),
    ).toBeNull();
  });
});
