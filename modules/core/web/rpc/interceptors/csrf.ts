// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Interceptor } from '@connectrpc/connect';
import { getCSRFProvider } from '../providers';

/**
 * Attach CSRF header when a provider is registered.
 * Resolve the provider per request so early client creation still works later.
 */
export function createCSRFInterceptor(): Interceptor {
  return next => async req => {
    const csrfProvider = getCSRFProvider();
    if (!csrfProvider) {
      return next(req);
    }

    const token = await csrfProvider.getCSRFToken();
    if (token) {
      req.header.set('X-XSRF-TOKEN', token);
    }

    return next(req);
  };
}
