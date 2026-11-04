// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isGuestSetupRoute } from './guestSetupRoute';

test('isGuestSetupRoute: name setup', () => {
  expect(isGuestSetupRoute({ name: 'setup', path: '/other' })).toBe(true);
});

test('isGuestSetupRoute: /web/setup and /setup paths', () => {
  expect(isGuestSetupRoute({ path: '/web/setup' })).toBe(true);
  expect(isGuestSetupRoute({ path: '/web/setup/' })).toBe(true);
  expect(isGuestSetupRoute({ path: '/setup' })).toBe(true);
  expect(isGuestSetupRoute({ path: '/login' })).toBe(false);
  expect(isGuestSetupRoute({ path: '/' })).toBe(false);
  expect(isGuestSetupRoute(null)).toBe(false);
});
