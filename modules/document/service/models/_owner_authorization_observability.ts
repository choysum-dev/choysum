// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeOptionalString } from '@/core/service/utils/normalization';

const STORAGE_PERMISSION_DENIED_COUNTER_KEY = Symbol.for('choysum.document.permission_denied_total');

type OwnerPermissionStage =
  | 'prepare'
  | 'finalize'
  | 'bind'
  | 'unbind'
  | 'descriptor'
  | 'download'
  | 'authorize_upload_put'
  | 'commit_upload_put'
  | 'resolve_download_content';

function normalizeReason(value: string): string {
  const normalized = value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '');
  return normalized || 'unknown';
}

/**
 * Emit a structured warning and increment a best-effort in-memory counter
 * whenever a document permission-denied event occurs.
 *
 * This function is intentionally fire-and-forget; it must never throw.
 */
export function observePermissionDenied(stage: OwnerPermissionStage, message: string, metadata: Record<string, unknown>): void {
  const ownerModel = normalizeOptionalString(metadata.ownerModel) ?? 'unknown';
  const fieldName = normalizeOptionalString(metadata.fieldName) ?? 'unknown';
  const reason = normalizeReason(normalizeOptionalString(metadata.reason) ?? message);

  try {
    console.warn(
      `[DOCUMENT][permission_denied] ${JSON.stringify({
        stage,
        ownerModel,
        fieldName,
        reason,
      })}`
    );
  } catch {
    // Observability should never block business errors.
  }

  try {
    const root = globalThis as any;
    const store: Record<string, number> = root[STORAGE_PERMISSION_DENIED_COUNTER_KEY] ?? {};
    root[STORAGE_PERMISSION_DENIED_COUNTER_KEY] = store;

    const key = `${stage}|${reason}`;
    store[key] = (store[key] ?? 0) + 1;

    console.info(
      `[METRIC] ${JSON.stringify({
        name: 'document.permission_denied_total',
        stage,
        reason,
        value: store[key],
        delta: 1,
      })}`
    );
  } catch {
    // Metrics emission is best-effort.
  }
}
