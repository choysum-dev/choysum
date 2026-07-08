// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError, GrpcCode } from '@/core/service/error';
import type { SequenceCleanupIdempotencyParams, SequenceCleanupIdempotencyResult } from './sequence';

export async function cleanupSequenceIdempotency(params?: SequenceCleanupIdempotencyParams): Promise<SequenceCleanupIdempotencyResult> {
  const { default: SequenceIdempotency } = await import('./sequence_idempotency');
  const olderThan = String(params?.OlderThan ?? '').trim();
  let cutoff: Date;
  if (!olderThan) {
    cutoff = new Date();
  } else {
    if (!/^\d{4}-\d{2}-\d{2}$/.test(olderThan)) {
      throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'OlderThan must be YYYY-MM-DD' }).withGrpcCode(GrpcCode.InvalidArgument);
    }
    cutoff = new Date(`${olderThan}T00:00:00.000Z`);
  }
  const deleted = await SequenceIdempotency.Delete(['ExpiresAt', '<', cutoff] as any);
  return { Deleted: Number(deleted) || 0 };
}
