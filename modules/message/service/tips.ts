// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Frozen tip topic for Form / Chatter thread refresh (matches pkg/bus). */
export const TOPIC_MESSAGE_THREAD_CHANGED = 'message.thread.changed';

/** Frozen tip topic for one user's inbox refresh (matches pkg/bus). */
export const TOPIC_MESSAGE_NOTIFICATION_USER = 'message.notification.user';

/** Tip source stamped on Message.Post publishes. */
export const MESSAGE_POST_TIP_SOURCE = 'message.Post';

/** Tip source stamped on notification inbox publishes. */
export const MESSAGE_NOTIFICATION_TIP_SOURCE = 'message.Notification';

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
export function __setMessagePublishTipForTest(fn: PublishTipFn | null | undefined): void {
  publishTipOverride = fn;
}

export function resolvePublishTip(): PublishTipFn | null {
  if (publishTipOverride !== undefined) return publishTipOverride;
  const publish = (globalThis as { $choysum?: { bus?: { publish?: PublishTipFn } } }).$choysum?.bus?.publish;
  return typeof publish === 'function' ? publish.bind((globalThis as any).$choysum.bus) : null;
}

function parseCreatedAtMillis(createdAt: Date | string | number | null | undefined): number | undefined {
  if (createdAt instanceof Date && !Number.isNaN(createdAt.getTime())) {
    return createdAt.getTime();
  }
  if (typeof createdAt === 'number' && Number.isFinite(createdAt)) {
    return createdAt;
  }
  if (typeof createdAt === 'string' && createdAt.trim()) {
    const parsed = Date.parse(createdAt);
    if (!Number.isNaN(parsed)) return parsed;
  }
  return undefined;
}

/**
 * Best-effort thread tip after a successful write. Never throws; tip is not authoritative.
 */
export async function publishThreadChangedTip(created: {
  Id?: string | null;
  Model?: string | null;
  ResId?: string | null;
  CreatedAt?: Date | string | number | null;
}): Promise<void> {
  const publish = resolvePublishTip();
  if (!publish) return;
  const messageId = String(created.Id || '').trim();
  const model = String(created.Model || '').trim();
  const resId = String(created.ResId || '').trim();
  if (!messageId || !model || !resId) return;
  const at = parseCreatedAtMillis(created.CreatedAt);
  try {
    await publish({
      topic: TOPIC_MESSAGE_THREAD_CHANGED,
      source: MESSAGE_POST_TIP_SOURCE,
      ...(at != null ? { at } : {}),
      payload: { model, resId, messageId },
    });
  } catch {
    // Tip is best-effort; authoritative state remains the Message row.
  }
}

/**
 * Best-effort inbox tip for one recipient. Never throws.
 */
export async function publishNotificationUserTip(
  userId: string,
  createdAt?: Date | string | number | null
): Promise<void> {
  const publish = resolvePublishTip();
  const uid = String(userId || '').trim();
  if (!publish || !uid) return;
  const at = parseCreatedAtMillis(createdAt);
  try {
    await publish({
      topic: TOPIC_MESSAGE_NOTIFICATION_USER,
      source: MESSAGE_NOTIFICATION_TIP_SOURCE,
      ...(at != null ? { at } : {}),
      payload: { userId: uid },
    });
  } catch {
    // Tip is best-effort; authoritative state remains Notification rows.
  }
}

/**
 * Best-effort inbox tips for many recipients. Never throws.
 */
export async function publishNotificationUserTips(
  userIds: Iterable<string>,
  createdAt?: Date | string | number | null
): Promise<void> {
  const seen = new Set<string>();
  for (const raw of userIds) {
    const uid = String(raw || '').trim();
    if (!uid || seen.has(uid)) continue;
    seen.add(uid);
    await publishNotificationUserTip(uid, createdAt);
  }
}
