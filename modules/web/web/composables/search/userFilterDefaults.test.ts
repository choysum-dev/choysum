// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import {
  isNewerUserFilter,
  isSharedUserFilterUserId,
  mergeUserFilterDefaults,
  pickLatestIsDefault,
  resolveUserFilterUserId,
  userFilterToNamedFilter,
} from './userFilterDefaults';

describe('resolveUserFilterUserId', () => {
  it('treats null, undefined, and empty string as missing', () => {
    expect(resolveUserFilterUserId(null)).toBe('');
    expect(resolveUserFilterUserId(undefined)).toBe('');
    expect(resolveUserFilterUserId('')).toBe('');
  });

  it('unwraps ManyToOneRef { Id } including null/empty nested Id', () => {
    expect(resolveUserFilterUserId({ Id: 'u1' })).toBe('u1');
    expect(resolveUserFilterUserId({ Id: '  u2  ' })).toBe('u2');
    expect(resolveUserFilterUserId({ Id: null })).toBe('');
    expect(resolveUserFilterUserId({ Id: '' })).toBe('');
    expect(resolveUserFilterUserId({ Id: undefined })).toBe('');
  });

  it('stringifies bare ids; non-Id objects/arrays count as missing', () => {
    expect(resolveUserFilterUserId('  me  ')).toBe('me');
    expect(resolveUserFilterUserId({ Name: 'x' } as any)).toBe('');
    expect(resolveUserFilterUserId(['u1'] as any)).toBe('');
  });
});

describe('isSharedUserFilterUserId', () => {
  it('is true only when resolved owner id is empty', () => {
    expect(isSharedUserFilterUserId(null)).toBe(true);
    expect(isSharedUserFilterUserId({ Id: null })).toBe(true);
    expect(isSharedUserFilterUserId('me')).toBe(false);
    expect(isSharedUserFilterUserId({ Id: 'me' })).toBe(false);
  });
});

describe('userFilterToNamedFilter', () => {
  it('maps Condition to query and optional selected', () => {
    const nf = userFilterToNamedFilter(
      { Name: 'Active', Condition: { And: [['IsActive', '=', true]] }, IsDefault: true },
      true
    );
    expect(nf.name).toBe('Active');
    expect(nf.selected).toBe(true);
    expect(nf.query).toEqual({ And: [['IsActive', '=', true]] });
  });

  it('trims Name and defaults missing Condition to {}', () => {
    const nf = userFilterToNamedFilter({ Name: '  Trimmed  ' });
    expect(nf.name).toBe('Trimmed');
    expect(nf.query).toEqual({});
    expect(nf.selected).toBe(false);
  });

  it('maps falsy Name to empty string', () => {
    expect(userFilterToNamedFilter({ Name: null as any }).name).toBe('');
    expect(userFilterToNamedFilter({ Name: undefined }).name).toBe('');
    expect(userFilterToNamedFilter({}).name).toBe('');
  });
});

describe('isNewerUserFilter', () => {
  it('compares UpdatedAt via Date instances and invalid dates as 0', () => {
    const newer = { Id: 'n', UpdatedAt: new Date('2026-06-01T00:00:00.000Z') };
    const older = { Id: 'o', UpdatedAt: new Date('2026-01-01T00:00:00.000Z') };
    expect(isNewerUserFilter(newer, older)).toBe(true);
    expect(isNewerUserFilter(older, newer)).toBe(false);

    const invalid = { Id: 'i', UpdatedAt: new Date('not-a-date') };
    const missing = { Id: 'm', UpdatedAt: null };
    // Both resolve to 0 → fall through to Id tie-break.
    expect(isNewerUserFilter(invalid, missing)).toBe(false);
    expect(isNewerUserFilter({ Id: 'z', UpdatedAt: 'nope' }, { Id: 'a', UpdatedAt: '' })).toBe(true);
  });

  it('uses CreatedAt when UpdatedAt ties, then Id (including missing Id)', () => {
    const a = {
      Id: 'a',
      UpdatedAt: '2026-01-01T00:00:00.000Z',
      CreatedAt: '2026-06-01T00:00:00.000Z',
    };
    const b = {
      Id: 'b',
      UpdatedAt: '2026-01-01T00:00:00.000Z',
      CreatedAt: '2026-03-01T00:00:00.000Z',
    };
    expect(isNewerUserFilter(a, b)).toBe(true);
    expect(isNewerUserFilter(b, a)).toBe(false);

    const noId = { UpdatedAt: '2026-01-01T00:00:00.000Z', CreatedAt: '2026-01-01T00:00:00.000Z' };
    const withId = { Id: 'z', UpdatedAt: '2026-01-01T00:00:00.000Z', CreatedAt: '2026-01-01T00:00:00.000Z' };
    expect(isNewerUserFilter(withId, noId)).toBe(true);
    expect(isNewerUserFilter(noId, withId)).toBe(false);
  });
});

describe('pickLatestIsDefault', () => {
  it('picks newest private by UpdatedAt then Id', () => {
    const rows = [
      { Id: 'a', Name: 'Old', IsDefault: true, UserId: 'me', UpdatedAt: '2026-01-01T00:00:00.000Z' },
      { Id: 'b', Name: 'New', IsDefault: true, UserId: 'me', UpdatedAt: '2026-06-01T00:00:00.000Z' },
      { Id: 'c', Name: 'Shared', IsDefault: true, UserId: null, UpdatedAt: '2026-12-01T00:00:00.000Z' },
      { Id: 'd', Name: 'Off', IsDefault: false, UserId: 'me', UpdatedAt: '2026-12-01T00:00:00.000Z' },
    ];
    expect(pickLatestIsDefault(rows, 'private')?.Id).toBe('b');
    expect(pickLatestIsDefault(rows, 'shared')?.Id).toBe('c');
  });

  it('falls back to Id when timestamps tie or missing', () => {
    const rows = [
      { Id: 'id1', Name: 'A', IsDefault: true, UserId: 'me' },
      { Id: 'id9', Name: 'B', IsDefault: true, UserId: 'me' },
    ];
    expect(pickLatestIsDefault(rows, 'private')?.Id).toBe('id9');
  });

  it('keeps an already-newer first row when later rows are older', () => {
    const rows = [
      { Id: 'new', IsDefault: true, UserId: 'me', UpdatedAt: '2026-06-01T00:00:00.000Z' },
      { Id: 'old', IsDefault: true, UserId: 'me', UpdatedAt: '2026-01-01T00:00:00.000Z' },
    ];
    expect(pickLatestIsDefault(rows, 'private')?.Id).toBe('new');
  });

  it('returns null when no matching IsDefault or rows are nullish', () => {
    expect(pickLatestIsDefault([], 'private')).toBeNull();
    expect(pickLatestIsDefault(null, 'shared')).toBeNull();
    expect(pickLatestIsDefault(undefined, 'private')).toBeNull();
    expect(pickLatestIsDefault([{ Id: 'x', IsDefault: false, UserId: 'me' }], 'private')).toBeNull();
    expect(pickLatestIsDefault([{ Id: 'x', IsDefault: true, UserId: '' }], 'private')).toBeNull();
  });
});

describe('mergeUserFilterDefaults', () => {
  const code = [
    { name: 'CodeA', query: { And: [['A', '=', 1]] }, selected: true },
    { name: 'CodeB', query: { And: [['B', '=', 2]] }, selected: false },
  ];

  it('prefers private IsDefault over shared and code selected', () => {
    const merged = mergeUserFilterDefaults({
      privateDefault: { Name: 'Mine', Condition: { And: [['X', '=', 1]] }, IsDefault: true, UserId: 'u1' },
      sharedDefault: { Name: 'Team', Condition: { And: [['Y', '=', 1]] }, IsDefault: true, UserId: null },
      codeDefaults: code,
    });
    expect(merged[0]).toMatchObject({ name: 'Mine', selected: true });
    expect(merged.every((n, i) => i === 0 || n.selected !== true)).toBe(true);
  });

  it('uses shared IsDefault when no private default', () => {
    const merged = mergeUserFilterDefaults({
      privateDefault: null,
      sharedDefault: { Name: 'Team', Condition: {}, IsDefault: true, UserId: null },
      codeDefaults: code,
    });
    expect(merged[0]).toMatchObject({ name: 'Team', selected: true });
    expect(merged.find(n => n.name === 'CodeA')?.selected).toBe(false);
  });

  it('falls back to code defaults when no server default', () => {
    const merged = mergeUserFilterDefaults({
      privateDefault: null,
      sharedDefault: null,
      codeDefaults: code,
    });
    expect(merged).toEqual(code);
  });

  it('ignores private/shared rows that are not IsDefault and accepts a single code default', () => {
    const merged = mergeUserFilterDefaults({
      privateDefault: { Name: 'NotDefault', IsDefault: false, UserId: 'u1' },
      sharedDefault: { Name: 'AlsoNot', IsDefault: false, UserId: null },
      codeDefaults: { name: 'Solo', query: ['A', '=', 1], selected: true },
    });
    expect(merged).toEqual([{ name: 'Solo', query: ['A', '=', 1], selected: true }]);
  });

  it('drops empty-name code presets when a server default wins', () => {
    const merged = mergeUserFilterDefaults({
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
    expect(mergeUserFilterDefaults({ codeDefaults: null })).toEqual([]);
    expect(mergeUserFilterDefaults({})).toEqual([]);
  });

  it('drops invalid code presets when shared IsDefault wins over non-default private', () => {
    const merged = mergeUserFilterDefaults({
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
