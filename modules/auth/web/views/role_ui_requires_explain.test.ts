// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { normalizeUiResourceRequires } from './role_ui_requires_explain';

describe('normalizeUiResourceRequires', () => {
  it('parses arrays and JSON strings; dedupes empties', () => {
    expect(normalizeUiResourceRequires(['rpc:/auth.User/Browse', 'rpc:/auth.User/Browse', ''])).toEqual([
      'rpc:/auth.User/Browse',
    ]);
    expect(normalizeUiResourceRequires('["rpc:/auth.User/Update"]')).toEqual(['rpc:/auth.User/Update']);
    expect(normalizeUiResourceRequires(null)).toEqual([]);
    expect(normalizeUiResourceRequires('rpc:/auth.User/Create')).toEqual(['rpc:/auth.User/Create']);
  });
});
