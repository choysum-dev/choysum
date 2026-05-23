// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Interceptor } from '@connectrpc/connect';
import { getCurrentRequestContext } from '../../../rpc/context';

function buildBaggage(items: Record<string, string>): string {
  const pairs: string[] = [];
  for (const [key, value] of Object.entries(items || {})) {
    let normalizedKey = key.trim().toLowerCase();
    if (normalizedKey && !normalizedKey.startsWith('ctx.')) {
      normalizedKey = `ctx.${normalizedKey}`;
    }

    const normalizedValue = String(value ?? '').trim();
    if (!normalizedKey || normalizedValue.length === 0) {
      continue;
    }

    pairs.push(`${encodeURIComponent(normalizedKey)}=${encodeURIComponent(normalizedValue)}`);
  }

  return pairs.join(',');
}

export function createContextInterceptor(): Interceptor | undefined {
  return next => async req => {
    try {
      const baggage = buildBaggage(getCurrentRequestContext());
      if (baggage) {
        req.header.set('baggage', baggage);
      }
    } catch (error) {
      console.warn('[ContextInterceptor] Failed to build baggage:', error);
    }

    return next(req);
  };
}
