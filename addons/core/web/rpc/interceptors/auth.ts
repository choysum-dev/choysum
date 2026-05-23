// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Code, type Interceptor } from '@connectrpc/connect';
import { getTokenProvider } from '../providers';

export function createAuthInterceptor(): Interceptor | undefined {
  const tokenProvider = getTokenProvider();
  if (!tokenProvider) {
    return undefined;
  }

  return next => async req => {
    try {
      const need = await tokenProvider.shouldRefreshToken?.();
      if (need) {
        await tokenProvider.refreshToken();
      }
    } catch {
      // Ignore pre-refresh failures and let the server handle auth fallback.
    }

    const token = await tokenProvider.getToken();
    if (token) {
      req.header.set('Authorization', `Bearer ${token}`);
    }

    let hasRetried = false;

    try {
      return await next(req);
    } catch (error: unknown) {
      const errorCode = (error as { code?: Code } | null | undefined)?.code;
      if (!hasRetried && errorCode === Code.Unauthenticated) {
        hasRetried = true;
        const ok = await tokenProvider.refreshToken();
        if (ok) {
          const newToken = await tokenProvider.getToken();
          if (newToken) {
            req.header.set('Authorization', `Bearer ${newToken}`);
            return await next(req);
          }
        }
      }

      throw error;
    }
  };
}
