// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { authRoutes } from './routes';

test('auth guest routes use navLoading route for chrome settle', () => {
  const byName = new Map(authRoutes.map(route => [String(route.name), route]));
  for (const name of ['login', 'setup', 'logout', 'register'] as const) {
    const route = byName.get(name);
    expect(route, name).toBeTruthy();
    expect(route?.meta?.requiresAuth).toBe(false);
    expect(route?.meta?.navLoading).toBe('route');
  }
});
