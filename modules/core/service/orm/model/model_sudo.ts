// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getRuntimeComputeAuditBucketValue, getRuntimeEnvBoolean, setRuntimeComputeAuditBucketValue } from '@/core/utils/env';
import { asObjectRecord } from '@/core/utils/object';
import type { ObjectRecord } from '../../../utils/types';
import { withRepositoryAuthzRuleBypass } from '../repository/authz';

type SudoAuditEntry = {
  version: 1;
  source: 'sudo';
  at: string;
  hint?: string;
};

function resolveSudoAuditBucket(): { sudoHits: SudoAuditEntry[] } {
  const bucketRecord = asObjectRecord(getRuntimeComputeAuditBucketValue());
  const bucket: ObjectRecord = bucketRecord ?? {};
  if (!Array.isArray(bucket.sudoHits)) bucket.sudoHits = [];
  setRuntimeComputeAuditBucketValue(bucket);
  return bucket as { sudoHits: SudoAuditEntry[] };
}

function sudoAuditEnabled(): boolean {
  // Default on; set CHOYSUM_SUDO_AUDIT_ENABLED=false to disable.
  return getRuntimeEnvBoolean('CHOYSUM_SUDO_AUDIT_ENABLED') ?? true;
}

/**
 * Records a sudo enter audit hit (once per Model.sudo call, including nested).
 */
export function recordSudoEnterAudit(hint?: string): void {
  if (!sudoAuditEnabled()) return;
  const bucket = resolveSudoAuditBucket();
  const entry: SudoAuditEntry = {
    version: 1,
    source: 'sudo',
    at: new Date().toISOString(),
  };
  const normalizedHint = String(hint || '').trim();
  if (normalizedHint) entry.hint = normalizedHint;
  bucket.sudoHits.push(entry);
}

/**
 * Runs `fn` with RecordRule + FieldRule bypass (company scope retained).
 * Sync and async `fn` are both supported for compute read paths.
 */
export function withModelSudo<R>(fn: () => R, opts?: { hint?: string }): R {
  recordSudoEnterAudit(opts?.hint);
  return withRepositoryAuthzRuleBypass(fn);
}
