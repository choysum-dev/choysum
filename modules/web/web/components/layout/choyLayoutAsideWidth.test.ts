// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { choyLayoutAsideWidth } from './choyLayoutAsideWidth';

describe('choyLayoutAsideWidth', () => {
  test('uses expanded and collapsed CSS variables', () => {
    expect(choyLayoutAsideWidth(false)).toContain('--choy-layout-sidebar-width');
    expect(choyLayoutAsideWidth(true)).toContain('--choy-layout-sidebar-collapsed-width');
  });
});
