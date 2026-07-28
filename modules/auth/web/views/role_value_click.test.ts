// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { roleIdFromValueClick } from './role_value_click';

describe('roleIdFromValueClick', () => {
  it('returns empty for missing or blank payloads', () => {
    expect(roleIdFromValueClick(undefined)).toBe('');
    expect(roleIdFromValueClick(null)).toBe('');
    expect(roleIdFromValueClick({})).toBe('');
    expect(roleIdFromValueClick({ id: null })).toBe('');
    expect(roleIdFromValueClick({ id: undefined })).toBe('');
    expect(roleIdFromValueClick({ id: '' })).toBe('');
    expect(roleIdFromValueClick({ id: '   ' })).toBe('');
  });

  it('returns trimmed id when present', () => {
    expect(roleIdFromValueClick({ id: 'role-9' })).toBe('role-9');
    expect(roleIdFromValueClick({ id: '  role-9  ' })).toBe('role-9');
  });
});
