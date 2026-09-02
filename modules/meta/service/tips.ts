// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Frozen tip topic for Meta module-op progress (matches pkg/bus). */
export const TOPIC_META_MODULE_OP_CHANGED = 'meta.module_op.changed';

/** Tip source stamped on MetaModule enqueue / status writes. */
export const META_MODULE_OP_TIP_SOURCE = 'meta.MetaModule';

/** Tip model locator for task jobs. */
export const META_MODULE_OP_TIP_MODEL = 'task.Job';

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
  const bus = (globalThis as { $choysum?: { bus?: { publish?: PublishTipFn } } }).$choysum?.bus;
  return typeof bus?.publish === 'function' ? bus.publish.bind(bus) : null;
}

/**
 * Best-effort tip after a Meta module-op job status is visible. Never throws;
 * tip is not authoritative — clients re-read via GetOpStatus.
 */
export async function publishModuleOpChangedTip(input: {
  jobId?: string | null;
  userId?: string | null;
  source?: string | null;
}): Promise<void> {
  const publish = resolvePublishTip();
  const jobId = String(input.jobId || '').trim();
  const userId = String(input.userId || '').trim();
  if (!publish || !jobId) return;
  const source = String(input.source || META_MODULE_OP_TIP_SOURCE).trim() || META_MODULE_OP_TIP_SOURCE;
  try {
    await publish({
      topic: TOPIC_META_MODULE_OP_CHANGED,
      source,
      payload: {
        model: META_MODULE_OP_TIP_MODEL,
        resId: jobId,
        jobId,
        ...(userId ? { userId } : {}),
      },
    });
  } catch {
    // Tip is best-effort; authoritative state remains task.Job + GetOpStatus.
  }
}
