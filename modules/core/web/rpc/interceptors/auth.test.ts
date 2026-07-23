// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, afterEach } from 'vitest';
import { createAuthInterceptor } from './auth';
import { setTokenProvider } from '../providers';

describe('createAuthInterceptor', () => {
  afterEach(() => {
    setTokenProvider(null);
  });

  it('attaches Authorization after setTokenProvider even if interceptor was created earlier', async () => {
    const interceptor = createAuthInterceptor();
    const headers = new Headers();
    const req = { header: headers } as any;

    setTokenProvider({
      getToken: async () => 'late-token',
      refreshToken: async () => false,
    });

    const next = vi.fn(async () => ({ ok: true } as any));
    await interceptor(next as any)(req);

    expect(headers.get('Authorization')).toBe('Bearer late-token');
    expect(next).toHaveBeenCalledTimes(1);
  });

  it('passes through when no token provider is registered', async () => {
    setTokenProvider(null);
    const interceptor = createAuthInterceptor();
    const headers = new Headers();
    const next = vi.fn(async () => ({ ok: true } as any));
    await interceptor(next as any)({ header: headers } as any);
    expect(headers.has('Authorization')).toBe(false);
    expect(next).toHaveBeenCalledTimes(1);
  });
});
