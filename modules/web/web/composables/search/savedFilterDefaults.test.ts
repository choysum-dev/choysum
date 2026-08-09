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

  it('trims Name and defaults missing Condition to {}', () => {
    const nf = savedFilterToNamedFilter({ Name: '  Trimmed  ' });
    expect(nf.name).toBe('Trimmed');
    expect(nf.query).toEqual({});
    expect(nf.selected).toBe(false);
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

  it('ignores private/shared rows that are not IsDefault and accepts a single code default', () => {
    const merged = mergeSavedFilterDefaults({
      privateDefault: { Name: 'NotDefault', IsDefault: false, UserId: 'u1' },
      sharedDefault: { Name: 'AlsoNot', IsDefault: false, UserId: null },
      codeDefaults: { name: 'Solo', query: ['A', '=', 1], selected: true },
    });
    expect(merged).toEqual([{ name: 'Solo', query: ['A', '=', 1], selected: true }]);
  });

  it('drops empty-name code presets when a server default wins', () => {
    const merged = mergeSavedFilterDefaults({
      privateDefault: { Name: 'Mine', Condition: {}, IsDefault: true, UserId: 'u1' },
      codeDefaults: [
        { name: '', query: ['X', '=', 1] } as any,
        { name: 'Mine', query: ['Dup', '=', 1], selected: true },
        { name: 'Keep', query: ['Y', '=', 2], selected: true },
      ],
    });
    expect(merged.map(n => n.name)).toEqual(['Mine', 'Keep']);
    expect(merged[1].selected).toBe(false);
  });

  it('returns [] when codeDefaults is null/undefined', () => {
    expect(mergeSavedFilterDefaults({ codeDefaults: null })).toEqual([]);
    expect(mergeSavedFilterDefaults({})).toEqual([]);
  });

  it('drops invalid code presets when shared IsDefault wins over non-default private', () => {
    const merged = mergeSavedFilterDefaults({
      privateDefault: { Name: 'PrivOff', Condition: { And: [['P', '=', 1]] }, IsDefault: false, UserId: 'u1' },
      sharedDefault: { Name: 'Team', Condition: { And: [['T', '=', 1]] }, IsDefault: true, UserId: null },
      codeDefaults: [
        null as any,
        { name: 1 as any, query: ['Bad', '=', 1] },
        { name: 'Keep', query: ['K', '=', 1], selected: true },
      ],
    });
    expect(merged[0]).toMatchObject({ name: 'Team', selected: true, query: { And: [['T', '=', 1]] } });
    expect(merged.slice(1)).toEqual([{ name: 'Keep', query: ['K', '=', 1], selected: false }]);
  });
});
