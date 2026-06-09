// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { LRUCache } from 'lru-cache';

// Global singleton cache for passing full objects between views/actions
// to avoid immediate re-fetching.
// TTL is short (5s) because this is only for immediate handoffs.
export const handoffCache = new LRUCache<string, any>({
  max: 100,
  ttl: 1000 * 5,
  allowStale: false,
});

/**
 * Retrieves an item from the cache and immediately removes it.
 * This ensures we don't serve stale data if the user comes back much later
 * (though TTL handles that too, this is safer for "one-time pass" semantics).
 */
export function flashRead(id: string): any | undefined {
  const val = handoffCache.get(id);
  if (val) {
    handoffCache.delete(id);
  }
  return val;
}
