// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeDateString } from './_normalizers';
import type { SequenceCleanupIdempotencyParams, SequenceCleanupIdempotencyResult } from './sequence';

export async function cleanupSequenceIdempotency(params?: SequenceCleanupIdempotencyParams): Promise<SequenceCleanupIdempotencyResult> {
  const { default: SequenceIdempotency } = await import('./sequence_idempotency');
  const olderThan = String(params?.OlderThan ?? '').trim();
  let cutoff: Date;
  if (!olderThan) {
    cutoff = new Date();
  } else {
    normalizeDateString(olderThan, 'OlderThan');
    cutoff = new Date(`${olderThan}T00:00:00.000Z`);
  }
  const deleted = await SequenceIdempotency.Delete(['ExpiresAt', '<', cutoff] as any);
  return { Deleted: Number(deleted) || 0 };
}
