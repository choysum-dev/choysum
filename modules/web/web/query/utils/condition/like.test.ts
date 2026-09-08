// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { valueToPreview } from './like';
import { filtersToQuery } from './builder';
import type { ConditionGroup } from '../../types';

// density backfill from main before merge
// Wall-clock datetime formatting needs dayjs timezone + real ICU under QJS.

test('valueToPreview datetime wall-clock: keeps calendar date literals unchanged', () => {
  expect(valueToPreview('=', '2024-07-01', { fieldType: 'date', timeZone: 'America/New_York' })).toBe('2024-07-01');
});

test('valueToPreview datetime wall-clock: stringifies time Date values and arrays', () => {
  const t = new Date(2024, 0, 1, 12, 30, 0);
  expect(valueToPreview('=', t, { fieldType: 'time' })).toBe(String(t));
  expect(valueToPreview('in', ['a', 'b'], { fieldType: 'date' })).toBe('(a, b)');
});

test('valueToPreview datetime wall-clock: keeps non-ISO calendar strings literal without fieldType', () => {
  expect(valueToPreview('=', '2024-07-01', { timeZone: 'America/New_York' })).toBe('2024-07-01');
});

test('valueToPreview datetime wall-clock: stringifies time string values and wraps like operators', () => {
  expect(valueToPreview('=', '12:30:00', { fieldType: 'time' })).toBe('12:30:00');
  expect(valueToPreview('like', '2024-06-30T16:00:00.000Z', { fieldType: 'datetime' })).toBe(
    '%2024-06-30T16:00:00.000Z%'
  );
});

test('valueToPreview datetime wall-clock: falls back to literal when datetime wall format yields empty', () => {
  expect(valueToPreview('=', '2024-01-01T99:99:99Z', { fieldType: 'datetime', timeZone: 'UTC' })).toBe(
    '2024-01-01T99:99:99Z'
  );
});

test('filtersToQuery datetime wire stays UTC: passes datetime ISO values through without re-zoning', () => {
  const utc = '2024-06-30T16:00:00.000Z';
  const root: ConditionGroup = {
    id: 'root',
    logic: 'And',
    children: [{ id: 'c1', field: 'CreatedAt', operator: '>=', value: utc } as any],
  } as any;
  const query = filtersToQuery([root]);
  expect(query).toEqual(['CreatedAt', '>=', utc]);
});
