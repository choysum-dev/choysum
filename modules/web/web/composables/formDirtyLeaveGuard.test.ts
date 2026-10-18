// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isAuthExitNavigation } from './formDirtyLeaveGuard';

describe('isAuthExitNavigation', () => {
  test('matches login / logout / register by name', () => {
    expect(isAuthExitNavigation({ name: 'login' })).toBe(true);
    expect(isAuthExitNavigation({ name: 'logout' })).toBe(true);
    expect(isAuthExitNavigation({ name: 'register' })).toBe(true);
    expect(isAuthExitNavigation({ name: 'UserDetail' })).toBe(false);
  });

  test('matches auth route resource ids', () => {
    expect(isAuthExitNavigation({ meta: { resourceId: 'auth.route.login' } })).toBe(true);
    expect(isAuthExitNavigation({ meta: { resourceId: 'auth.route.logout' } })).toBe(true);
    expect(isAuthExitNavigation({ meta: { resourceId: 'auth.route.register' } })).toBe(true);
    expect(isAuthExitNavigation({ meta: { resourceId: 'auth.route.user_list' } })).toBe(false);
  });

  test('matches login paths including /web prefix', () => {
    expect(isAuthExitNavigation({ path: '/login' })).toBe(true);
    expect(isAuthExitNavigation({ path: '/web/login' })).toBe(true);
    expect(isAuthExitNavigation({ path: '/web/logout/' })).toBe(true);
    expect(isAuthExitNavigation({ path: '/auth/users' })).toBe(false);
  });
});
