// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createAuthInterceptor } from './auth';
import { setTokenProvider } from '../providers';

function makeHeaders() {
  const map = new Map<string, string>();
  return {
    get(name: string) {
      return map.get(String(name).toLowerCase()) ?? null;
    },
    has(name: string) {
      return map.has(String(name).toLowerCase());
    },
    set(name: string, value: string) {
      map.set(String(name).toLowerCase(), value);
    },
  };
}

test('createAuthInterceptor: attaches Authorization after setTokenProvider even if interceptor was created earlier', async () => {
  setTokenProvider(null);
  const interceptor = createAuthInterceptor();
  const headers = makeHeaders();
  const req = { header: headers } as any;

  setTokenProvider({
    getToken: async () => 'late-token',
    refreshToken: async () => false,
  });

  let nextCalls = 0;
  const next = async () => {
    nextCalls += 1;
    return { ok: true } as any;
  };
  await interceptor(next as any)(req);

  expect(headers.get('Authorization')).toBe('Bearer late-token');
  expect(nextCalls).toBe(1);
  setTokenProvider(null);
});

test('createAuthInterceptor: passes through when no token provider is registered', async () => {
  setTokenProvider(null);
  const interceptor = createAuthInterceptor();
  const headers = makeHeaders();
  let nextCalls = 0;
  const next = async () => {
    nextCalls += 1;
    return { ok: true } as any;
  };
  await interceptor(next as any)({ header: headers } as any);
  expect(headers.has('Authorization')).toBe(false);
  expect(nextCalls).toBe(1);
  setTokenProvider(null);
});
