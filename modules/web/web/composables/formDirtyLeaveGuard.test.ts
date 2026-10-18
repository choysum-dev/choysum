// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isAuthExitNavigation, shouldSkipFormDirtyLeavePrompt } from './formDirtyLeaveGuard';

describe('isAuthExitNavigation', () => {
  test('matches login / logout / register by name', () => {
    expect(isAuthExitNavigation({ name: 'login' })).toBe(true);
    expect(isAuthExitNavigation({ name: 'logout' })).toBe(true);
    expect(isAuthExitNavigation({ name: 'register' })).toBe(true);
    expect(isAuthExitNavigation({ name: 'UserDetail' })).toBe(false);
  });

  test('name-only auth match does not skip domain paths like /pos/register', () => {
    expect(isAuthExitNavigation({ name: 'register', path: '/pos/register' })).toBe(false);
    expect(isAuthExitNavigation({ name: 'register', path: '/register' })).toBe(true);
    expect(isAuthExitNavigation({ name: 'login', path: '/web/login' })).toBe(true);
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
    // Domain routes that merely end in /register must keep the dirty-leave prompt.
    expect(isAuthExitNavigation({ path: '/pos/register' })).toBe(false);
    expect(isAuthExitNavigation({ path: '/events/42/register' })).toBe(false);
  });
});

describe('shouldSkipFormDirtyLeavePrompt', () => {
  test('skips when navigating to an auth exit', () => {
    expect(shouldSkipFormDirtyLeavePrompt({ name: 'login' }, { name: 'UserDetail' })).toBe(true);
  });

  test('skips when leaving login / register (credential shell)', () => {
    expect(shouldSkipFormDirtyLeavePrompt({ path: '/' }, { name: 'login', path: '/login' })).toBe(
      true
    );
    expect(
      shouldSkipFormDirtyLeavePrompt({ path: '/home' }, { name: 'register', path: '/register' })
    ).toBe(true);
  });

  test('does not skip ordinary app-to-app navigation', () => {
    expect(
      shouldSkipFormDirtyLeavePrompt({ name: 'UserList' }, { name: 'UserDetail', path: '/users/1' })
    ).toBe(false);
  });
});
