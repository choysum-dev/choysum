// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useConnectivityStore } from '@/web/web/stores/connectivityStore';

/**
 * Host-side offline gate for Pager / Pagination.
 * Missing Pinia (no connectivity store) is treated as reachable.
 */
export function readSessionUnreachable(): boolean {
  try {
    return useConnectivityStore().status === 'unreachable';
  } catch {
    return false;
  }
}
