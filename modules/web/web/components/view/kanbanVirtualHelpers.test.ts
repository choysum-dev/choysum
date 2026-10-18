// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  CHOY_KANBAN_CARD_GAP,
  CHOY_KANBAN_DEFAULT_CARD_SIZE,
  CHOY_KANBAN_VIRTUAL_THRESHOLD,
  choyKanbanShouldVirtualize,
  decodeChoyKanbanLaneDragPayload,
  encodeChoyKanbanLaneDragPayload,
  resolveChoyKanbanEstimateCardSize,
  resolveChoyKanbanLaneDropMove,
} from './kanbanVirtualHelpers';

describe('kanbanVirtualHelpers', () => {
  test('choyKanbanShouldVirtualize respects auto threshold and overrides', () => {
    expect(CHOY_KANBAN_VIRTUAL_THRESHOLD).toBe(40);
    expect(choyKanbanShouldVirtualize('auto', 39)).toBe(false);
    expect(choyKanbanShouldVirtualize('auto', 40)).toBe(true);
    expect(choyKanbanShouldVirtualize(true, 1)).toBe(true);
    expect(choyKanbanShouldVirtualize(true, 0)).toBe(false);
    expect(choyKanbanShouldVirtualize(false, 100)).toBe(false);
    expect(choyKanbanShouldVirtualize(undefined, 40)).toBe(true);
  });

  test('virtual card gap matches non-virtual gap-2', () => {
    expect(CHOY_KANBAN_CARD_GAP).toBe(8);
    expect(CHOY_KANBAN_DEFAULT_CARD_SIZE).toBe(88);
  });

  test('resolveChoyKanbanEstimateCardSize falls back to default', () => {
    expect(resolveChoyKanbanEstimateCardSize(undefined)).toBe(CHOY_KANBAN_DEFAULT_CARD_SIZE);
    expect(resolveChoyKanbanEstimateCardSize(0)).toBe(CHOY_KANBAN_DEFAULT_CARD_SIZE);
    expect(resolveChoyKanbanEstimateCardSize(-1)).toBe(CHOY_KANBAN_DEFAULT_CARD_SIZE);
    expect(resolveChoyKanbanEstimateCardSize(88)).toBe(88);
    expect(resolveChoyKanbanEstimateCardSize(88.9)).toBe(88);
  });

  test('encode/decode lane drag payload', () => {
    expect(encodeChoyKanbanLaneDragPayload(null)).toBe('');
    expect(encodeChoyKanbanLaneDragPayload({ cardId: '', fromLaneKey: 'a' })).toBe('');
    const encoded = encodeChoyKanbanLaneDragPayload({ cardId: 'c1', fromLaneKey: 'todo' });
    expect(decodeChoyKanbanLaneDragPayload(encoded)).toEqual({
      cardId: 'c1',
      fromLaneKey: 'todo',
    });
    expect(decodeChoyKanbanLaneDragPayload('not-json')).toBeNull();
    expect(decodeChoyKanbanLaneDragPayload('{"cardId":"c1"}')).toBeNull();
  });

  test('resolveChoyKanbanLaneDropMove appends to target lane', () => {
    expect(
      resolveChoyKanbanLaneDropMove({
        payload: { cardId: 'c1', fromLaneKey: 'todo' },
        toLaneKey: 'done',
        toLaneCardCount: 3,
      }),
    ).toEqual({
      cardId: 'c1',
      fromLaneKey: 'todo',
      toLaneKey: 'done',
      toIndex: 3,
    });
    expect(
      resolveChoyKanbanLaneDropMove({
        payload: { cardId: 'c1', fromLaneKey: 'todo' },
        toLaneKey: 'todo',
        toLaneCardCount: 1,
      }),
    ).toBeNull();
    expect(
      resolveChoyKanbanLaneDropMove({
        payload: null,
        toLaneKey: 'done',
        toLaneCardCount: 0,
      }),
    ).toBeNull();
  });
});
