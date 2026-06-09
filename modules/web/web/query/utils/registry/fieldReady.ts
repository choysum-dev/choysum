// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick } from 'vue';
import { exportFieldSelection } from '@/web/web/query/utils/registry/field';

export interface AwaitFieldSelectionOptions {
  maxTries?: number; // default 5
  requireNonEmpty?: boolean; // default true; if false, returns even if still empty after maxTries
  intervalMs?: number; // optional fixed delay between tries (fallback to nextTick)
}

export async function awaitFieldSelection(store: { storeId: string } | string, opts: AwaitFieldSelectionOptions = {}): Promise<string[]> {
  const { maxTries = 5, requireNonEmpty = true, intervalMs } = opts;
  const storeId = typeof store === 'string' ? store : store.storeId;
  let last: string[] = [];
  for (let i = 0; i < maxTries; i++) {
    last = exportFieldSelection(storeId) || [];
    if (!requireNonEmpty || (Array.isArray(last) && last.length > 0)) return last;
    if (intervalMs && intervalMs > 0) {
      await new Promise(res => setTimeout(res, intervalMs));
    } else {
      await nextTick();
    }
  }
  return last; // may be empty if requireNonEmpty and registration not finished
}

// Convenience predicate
export function hasFieldSelection(store: { storeId: string } | string): boolean {
  const storeId = typeof store === 'string' ? store : store.storeId;
  const arr = exportFieldSelection(storeId);
  return Array.isArray(arr) && arr.length > 0;
}
