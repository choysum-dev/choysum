// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { authRoutes } from './routes';

test('auth guest routes use navLoading route for chrome settle', () => {
  const byName = new Map(authRoutes.map(route => [String(route.name), route]));
  const names: Array<'login' | 'setup' | 'logout' | 'register'> = ['login', 'setup', 'logout'];
  if (import.meta.env.CHOYSUM_ENABLE_REGISTRATION !== false) {
    names.push('register');
  }
  for (const name of names) {
    const route = byName.get(name);
    expect(route).toBeTruthy();
    expect(route?.meta?.requiresAuth).toBe(false);
    expect(route?.meta?.navLoading).toBe('route');
  }
});
