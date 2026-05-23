// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { asObjectRecord } from '@/core/utils/object';

export function shouldSilenceServerRpcError(serviceName: string, error: unknown): boolean {
  const message = String(asObjectRecord(error)?.message ?? '');
  return serviceName.startsWith('auth.') && (/no registered proto files for app\s+auth/i.test(message) || /failed to load method descriptor/i.test(message));
}

export function logServerRpcError(serviceName: string, methodName: string, error: unknown): void {
  if (shouldSilenceServerRpcError(serviceName, error)) {
    return;
  }
  // eslint-disable-next-line no-console
  console.error(`[ServerClient] Error invoking ${serviceName}.${methodName}:`, error);
}
