// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { valueToPreview } from './like';
import { filtersToQuery } from './builder';
import type { ConditionGroup } from '../../types';

describe('valueToPreview datetime wall-clock', () => {
  it('formats UTC ISO as user wall clock for datetime fields', () => {
    expect(
      valueToPreview('>=', '2024-06-30T16:00:00.000Z', { fieldType: 'datetime', timeZone: 'Asia/Shanghai' })
    ).toBe('2024-07-01 00:00:00');
    expect(
      valueToPreview('<=', '2024-06-30T16:00:00.000Z', { fieldType: 'datetime', timeZone: 'America/New_York' })
    ).toBe('2024-06-30 12:00:00');
  });

  it('keeps calendar date literals unchanged', () => {
    expect(valueToPreview('=', '2024-07-01', { fieldType: 'date', timeZone: 'America/New_York' })).toBe('2024-07-01');
  });

  it('formats Date date-field values from local calendar components', () => {
    // East-of-UTC: toISOString would shift to previous UTC day.
    const localMorning = new Date(2024, 6, 1, 8, 0, 0); // 2024-07-01 local
    expect(valueToPreview('=', localMorning, { fieldType: 'date', timeZone: 'Asia/Shanghai' })).toBe('2024-07-01');
  });

  it('infers datetime from ISO strings when fieldType omitted', () => {
    expect(valueToPreview('=', '2024-06-30T16:00:00.000Z', { timeZone: 'Asia/Shanghai' })).toBe('2024-07-01 00:00:00');
  });
});

describe('filtersToQuery datetime wire stays UTC', () => {
  it('passes datetime ISO values through without re-zoning', () => {
    const utc = '2024-06-30T16:00:00.000Z';
    const root: ConditionGroup = {
      id: 'root',
      logic: 'And',
      children: [{ id: 'c1', field: 'CreatedAt', operator: '>=', value: utc } as any],
    } as any;
    const query = filtersToQuery([root]);
    expect(query).toEqual(['CreatedAt', '>=', utc]);
  });
});
