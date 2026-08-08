// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { mergeSavedFilterDefaults, savedFilterToNamedFilter } from './savedFilterDefaults';

describe('savedFilterToNamedFilter', () => {
  it('maps Condition to query and optional selected', () => {
    const nf = savedFilterToNamedFilter(
      { Name: 'Active', Condition: { And: [['IsActive', '=', true]] }, IsDefault: true },
      true
    );
    expect(nf.name).toBe('Active');
    expect(nf.selected).toBe(true);
    expect(nf.query).toEqual({ And: [['IsActive', '=', true]] });
  });
});

describe('mergeSavedFilterDefaults', () => {
  const code = [
    { name: 'CodeA', query: { And: [['A', '=', 1]] }, selected: true },
    { name: 'CodeB', query: { And: [['B', '=', 2]] }, selected: false },
  ];

  it('prefers private IsDefault over shared and code selected', () => {
    const merged = mergeSavedFilterDefaults({
      privateDefault: { Name: 'Mine', Condition: { And: [['X', '=', 1]] }, IsDefault: true, UserId: 'u1' },
      sharedDefault: { Name: 'Team', Condition: { And: [['Y', '=', 1]] }, IsDefault: true, UserId: null },
      codeDefaults: code,
    });
    expect(merged[0]).toMatchObject({ name: 'Mine', selected: true });
    expect(merged.every((n, i) => i === 0 || n.selected !== true)).toBe(true);
  });

  it('uses shared IsDefault when no private default', () => {
    const merged = mergeSavedFilterDefaults({
      privateDefault: null,
      sharedDefault: { Name: 'Team', Condition: {}, IsDefault: true, UserId: null },
      codeDefaults: code,
    });
    expect(merged[0]).toMatchObject({ name: 'Team', selected: true });
    expect(merged.find(n => n.name === 'CodeA')?.selected).toBe(false);
  });

  it('falls back to code defaults when no server default', () => {
    const merged = mergeSavedFilterDefaults({
      privateDefault: null,
      sharedDefault: null,
      codeDefaults: code,
    });
    expect(merged).toEqual(code);
  });
});
