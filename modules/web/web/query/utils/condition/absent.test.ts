// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { asPresentCondition } from './absent';

describe('asPresentCondition', () => {
  it('returns undefined for nullish, empty array, and empty object', () => {
    expect(asPresentCondition(null)).toBeUndefined();
    expect(asPresentCondition(undefined)).toBeUndefined();
    expect(asPresentCondition([])).toBeUndefined();
    expect(asPresentCondition({})).toBeUndefined();
  });

  it('preserves non-empty values', () => {
    expect(asPresentCondition('x')).toBe('x');
    expect(asPresentCondition([1])).toEqual([1]);
    expect(asPresentCondition({ a: 1 })).toEqual({ a: 1 });
    expect(asPresentCondition(false)).toBe(false);
    expect(asPresentCondition(0)).toBe(0);
  });
});
