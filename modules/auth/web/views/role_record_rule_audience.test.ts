// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { isGrantEveryoneWarning } from './role_record_rule_audience';

describe('isGrantEveryoneWarning', () => {
  it('is false without a draft', () => {
    expect(isGrantEveryoneWarning(null)).toBe(false);
    expect(isGrantEveryoneWarning(undefined)).toBe(false);
  });

  it('is false when Kind is not grant', () => {
    expect(isGrantEveryoneWarning({ Kind: 'restrict', RoleId: null })).toBe(false);
    expect(isGrantEveryoneWarning({ Kind: 'RESTRICT', RoleId: '' })).toBe(false);
  });

  it('treats missing Kind as grant and empty Role as everyone', () => {
    expect(isGrantEveryoneWarning({})).toBe(true);
    expect(isGrantEveryoneWarning({ Kind: 'grant', RoleId: null })).toBe(true);
    expect(isGrantEveryoneWarning({ Kind: 'Grant', RoleId: '' })).toBe(true);
  });

  it('handles RoleId object and string forms', () => {
    expect(isGrantEveryoneWarning({ Kind: 'grant', RoleId: { Id: '' } })).toBe(true);
    expect(isGrantEveryoneWarning({ Kind: 'grant', RoleId: {} })).toBe(true);
    expect(isGrantEveryoneWarning({ Kind: 'grant', RoleId: { Id: 'role-1' } })).toBe(false);
    expect(isGrantEveryoneWarning({ Kind: 'grant', RoleId: 'role-1' })).toBe(false);
    expect(isGrantEveryoneWarning({ Kind: 'grant', RoleId: '   ' })).toBe(true);
  });
});
