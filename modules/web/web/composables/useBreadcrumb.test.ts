// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { tryUseMenu } from './useBreadcrumb';

test('useBreadcrumb tryUseMenu returns null when menu/router context is unavailable', () => {
  expect(tryUseMenu()).toBeNull();
});
