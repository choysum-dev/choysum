// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolveChoysum } from '@/core/service/runtime/choysum_runtime';

/** Frozen tip topic for Meta module-op progress (matches pkg/bus). */
export const TOPIC_META_MODULE_OP_CHANGED = 'meta.module_op.changed';

/** Tip source stamped on Meta module-op enqueue / status writes. */
export const META_MODULE_OP_TIP_SOURCE = 'meta.MetaModuleOperation';

/** Tip model locator for module operations (authoritative row). */
export const META_MODULE_OP_TIP_MODEL = 'meta.MetaModuleOperation';

export type PublishTipFn = (event: {
  topic: string;
  source: string;
  at?: number;
  payload?: Record<string, string>;
}) => void | Promise<void>;

let publishTipOverride: PublishTipFn | null | undefined;

/**
 * Test-only override for tip Publish.
 * undefined = live $choysum.bus.publish; null = force missing bus; function = stub.
 */
export function __setMetaPublishTipForTest(fn: PublishTipFn | null | undefined): void {
  publishTipOverride = fn;
}

export function resolvePublishTip(): PublishTipFn | null {
  if (publishTipOverride !== undefined) return publishTipOverride;
  const bus = resolveChoysum()?.bus;
  const publish = bus?.publish;
  return typeof publish === 'function' ? publish.bind(bus) : null;
}

/**
 * Best-effort tip after a Meta module operation status is visible. Never throws;
 * tip is not authoritative — clients re-read the operation row.
 */
export async function publishModuleOpChangedTip(input: {
  operationId?: string | null;
  jobId?: string | null;
  userId?: string | null;
  source?: string | null;
}): Promise<void> {
  const publish = resolvePublishTip();
  const operationId = String(input.operationId || '').trim();
  const jobId = String(input.jobId || '').trim();
  const userId = String(input.userId || '').trim();
  if (!publish || !operationId) return;
  const source = String(input.source || META_MODULE_OP_TIP_SOURCE).trim() || META_MODULE_OP_TIP_SOURCE;
  try {
    await publish({
      topic: TOPIC_META_MODULE_OP_CHANGED,
      source,
      payload: {
        model: META_MODULE_OP_TIP_MODEL,
        resId: operationId,
        operationId,
        ...(jobId ? { jobId } : {}),
        ...(userId ? { userId } : {}),
      },
    });
  } catch {
    // Tip is best-effort; authoritative state remains meta_module_operation.
  }
}
