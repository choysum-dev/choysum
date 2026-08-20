// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Frozen tip topic for Form / Chatter field-change refresh (matches pkg/bus). */
export const TOPIC_AUDIT_FIELD_CHANGE_APPENDED = 'audit.field_change.appended';

/** Tip source stamped on FieldChange.Append publishes. */
export const AUDIT_FIELD_CHANGE_TIP_SOURCE = 'audit.FieldChange';

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
export function __setAuditPublishTipForTest(fn: PublishTipFn | null | undefined): void {
  publishTipOverride = fn;
}

export function resolvePublishTip(): PublishTipFn | null {
  if (publishTipOverride !== undefined) return publishTipOverride;
  const publish = (globalThis as { $choysum?: { bus?: { publish?: PublishTipFn } } }).$choysum?.bus?.publish;
  return typeof publish === 'function' ? publish.bind((globalThis as any).$choysum.bus) : null;
}

function parseAtMillis(at: Date | string | number | null | undefined): number | undefined {
  if (at instanceof Date && !Number.isNaN(at.getTime())) {
    return at.getTime();
  }
  if (typeof at === 'number' && Number.isFinite(at)) {
    return at;
  }
  if (typeof at === 'string' && at.trim()) {
    const parsed = Date.parse(at);
    if (!Number.isNaN(parsed)) return parsed;
  }
  return undefined;
}

/**
 * Best-effort thread tip after a successful FieldChange append. Never throws; tip is not authoritative.
 */
export async function publishFieldChangeAppendedTip(created: {
  Id?: string | null;
  Model?: string | null;
  ResId?: string | null;
  At?: Date | string | number | null;
}): Promise<void> {
  const publish = resolvePublishTip();
  if (!publish) return;
  const fieldChangeId = String(created.Id || '').trim();
  const model = String(created.Model || '').trim();
  const resId = String(created.ResId || '').trim();
  if (!fieldChangeId || !model || !resId) return;
  const at = parseAtMillis(created.At);
  try {
    await publish({
      topic: TOPIC_AUDIT_FIELD_CHANGE_APPENDED,
      source: AUDIT_FIELD_CHANGE_TIP_SOURCE,
      ...(at != null ? { at } : {}),
      payload: { model, resId, fieldChangeId },
    });
  } catch {
    // Tip is best-effort; authoritative state remains the FieldChange row.
  }
}
