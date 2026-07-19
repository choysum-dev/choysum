// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeDateString } from '@/core/service/utils/normalization';
import { createTranslate } from '@/core/service/i18n';
import { mapNormalizationToBase } from './_normalizers';
import type { SequenceCleanupIdempotencyParams, SequenceCleanupIdempotencyResult } from './sequence';

const { _t } = createTranslate('base');

export async function cleanupSequenceIdempotency(params?: SequenceCleanupIdempotencyParams): Promise<SequenceCleanupIdempotencyResult> {
  const { default: SequenceIdempotency } = await import('./sequence_idempotency');
  const olderThan = String(params?.OlderThan ?? '').trim();
  let cutoff: Date;
  if (!olderThan) {
    cutoff = new Date();
  } else {
    const normalizedOlderThan = mapNormalizationToBase(
      () => normalizeDateString(olderThan),
      err => {
        if (err.code === 'required') return _t('OlderThan is required', { scope: 'service/models/_sequence_cleanup' });
        if (err.code === 'invalid_date_value') return _t('OlderThan is invalid', { scope: 'service/models/_sequence_cleanup' });
        return _t('OlderThan must be YYYY-MM-DD', { scope: 'service/models/_sequence_cleanup' });
      }
    );
    cutoff = new Date(`${normalizedOlderThan}T00:00:00.000Z`);
  }
  const deleted = await SequenceIdempotency.Delete(['ExpiresAt', '<', cutoff] as any);
  return { Deleted: Number(deleted) || 0 };
}
