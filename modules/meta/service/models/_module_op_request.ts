// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Job from '@/task/service/models/job';
import { ensureCurrentUserId } from './_module_management_runtime';
import { getBackendEnv } from '@/core/service/runtime/env/backend_env';

type ModuleAction = 'install' | 'uninstall' | 'upgrade';

/**
 * Validate and normalize a module name input.
 */
export function ensureModuleName(name?: string): string {
  const trimmed = String(name || '').trim();
  if (!trimmed) throw new Error('moduleName cannot be empty');
  return trimmed;
}

/**
 * Enqueue a module operation job and optionally force a lease-conflict
 * failure for E2E testing.
 *
 * Returns the enqueued job Id.
 */
export async function requestModuleOp(action: ModuleAction, moduleName: string, extraPayload?: Record<string, unknown>): Promise<string> {
  const name = ensureModuleName(moduleName);
  const userId = ensureCurrentUserId();
  const method = `meta.IrModule/Execute${action.charAt(0).toUpperCase() + action.slice(1)}`;
  const env = getBackendEnv();
  const forceLockConflict = Boolean((env as any).CHOYSUM_E2E_FORCE_LOCK_CONFLICT || (env as any).choysum_e2e_force_lock_conflict);

  const payload: Record<string, unknown> = { moduleName: name, operatorUserId: userId, ...(extraPayload || {}) };
  const job = await Job.EnqueueJob('meta', method, payload, userId, userId, undefined, 0, 0);

  if (forceLockConflict && (job as any)?.Id) {
    const retryAfterMs = 2500;
    await (Job as any).UpdateById((job as any).Id, {
      Status: 'failed',
      RunAfter: new Date(Date.now() + retryAfterMs),
      LastErrorJson: {
        domain: 'meta.lock',
        code: 'LEASE_CONFLICT',
        message: 'lease conflict',
        details: { retry_after_ms: retryAfterMs },
      },
    });
  }

  return String((job as any)?.Id || '').trim();
}
