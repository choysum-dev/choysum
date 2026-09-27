// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { VIEW_CONTAINER_KEY, FIELD_PREFIX_KEY, VIEW_MODE_KEY } from './viewScopeKeys';

describe('viewScopeKeys', () => {
  test('stable string keys match legacy inject sites', () => {
    expect(VIEW_MODE_KEY).toBe('view-mode');
    expect(VIEW_CONTAINER_KEY).toBe('view-container');
    expect(FIELD_PREFIX_KEY).toBe('field-prefix');
  });
});
