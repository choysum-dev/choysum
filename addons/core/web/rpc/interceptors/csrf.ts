// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Interceptor } from '@connectrpc/connect';
import { getCSRFProvider } from '../providers';

export function createCSRFInterceptor(): Interceptor | undefined {
  const csrfProvider = getCSRFProvider();
  if (!csrfProvider) {
    return undefined;
  }

  return next => async req => {
    const token = await csrfProvider.getCSRFToken();
    if (token) {
      req.header.set('X-XSRF-TOKEN', token);
    }

    return next(req);
  };
}