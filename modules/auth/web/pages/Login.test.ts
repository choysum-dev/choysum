// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { runLoginAuthReady } from './login_auth_ready';

type Call = { args: unknown[] };

function makeFn() {
  const calls: Call[] = [];
  const fn = (...args: any[]) => {
    calls.push({ args });
  };
  return { fn, calls };
}

test('runLoginAuthReady: redirects when still on login and authenticated', async () => {
  const redirect = makeFn();
  let readyCalls = 0;
  await runLoginAuthReady({
    ensureAuthReady: async () => {
      readyCalls += 1;
    },
    getRoutePath: () => '/login',
    isAuthenticated: () => true,
    redirect: redirect.fn,
  });
  expect(readyCalls).toBe(1);
  expect(redirect.calls.length).toBe(1);
});

test('runLoginAuthReady: skips redirect when route changed during init', async () => {
  let path = '/login';
  const redirect = makeFn();
  await runLoginAuthReady({
    ensureAuthReady: async () => {
      path = '/elsewhere';
    },
    getRoutePath: () => path,
    isAuthenticated: () => true,
    redirect: redirect.fn,
  });
  expect(redirect.calls.length).toBe(0);
});

test('runLoginAuthReady: stays on form when unauthenticated', async () => {
  const redirect = makeFn();
  await runLoginAuthReady({
    ensureAuthReady: async () => undefined,
    getRoutePath: () => '/login',
    isAuthenticated: () => false,
    redirect: redirect.fn,
  });
  expect(redirect.calls.length).toBe(0);
});
