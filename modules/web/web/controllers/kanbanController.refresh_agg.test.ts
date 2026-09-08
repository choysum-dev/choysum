// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { combineLaneAggregateConditions } from './kanbanController';

test('combineLaneAggregateConditions: uses the single-lane present condition', () => {
  expect(combineLaneAggregateConditions([{ Stage: 'todo' }])).toEqual({ Stage: 'todo' });
  expect(combineLaneAggregateConditions([])).toBeUndefined();
});

test('combineLaneAggregateConditions: drops combined Or when any selected lane is unconditioned', () => {
  expect(combineLaneAggregateConditions([{ Stage: 'todo' }, undefined])).toBeUndefined();
  expect(combineLaneAggregateConditions([{ Stage: 'todo' }, {}])).toBeUndefined();
});

test('combineLaneAggregateConditions: Or-combines present conditions when every lane is conditioned', () => {
  expect(combineLaneAggregateConditions([{ Stage: 'todo' }, { Stage: 'done' }])).toEqual({
    Or: [{ Stage: 'todo' }, { Stage: 'done' }],
  });
});
