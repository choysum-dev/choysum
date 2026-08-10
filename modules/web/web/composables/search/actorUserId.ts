// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useAuthStore } from '@/auth/web/stores/auth';

/**
 * Resolve the current actor user id for UserFilter queries/creates.
 * Browser SPA identity lives on the auth store; unit harness may use request context.
 */
export function actorUserId(): string {
  try {
    const auth = useAuthStore();
    const fromStore = String((auth.currentUser as any)?.Id || (auth.identity as any)?.userId || '').trim();
    if (fromStore) return fromStore;
  } catch {
    // Auth store unavailable (unit harness / early boot).
  }
  try {
    const id = (globalThis as any)?.$choysum?.request?.context?.identity?.userId;
    return String(id || '').trim();
  } catch {
    return '';
  }
}
