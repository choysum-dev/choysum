// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolveDefaultLandPath } from './resolveDefaultLandPath';

describe('resolveDefaultLandPath', () => {
  test('stub returns root until W4 wires menu + canNavigate', () => {
    expect(resolveDefaultLandPath()).toBe('/');
    expect(
      resolveDefaultLandPath({
        menuLeafPaths: ['/partner'],
        canNavigate: () => true,
      }),
    ).toBe('/');
  });
});
