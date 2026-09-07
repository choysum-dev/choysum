// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as coreWeb from './index';

test('core/web root export surface: keeps the root facade limited to component primitives', () => {
  expect(Object.keys(coreWeb).sort()).toEqual(['Xpath']);
});
