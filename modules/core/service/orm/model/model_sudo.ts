// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getRuntimeComputeAuditBucketValue, getRuntimeEnvBoolean, setRuntimeComputeAuditBucketValue } from '@/core/utils/env';
import { asObjectRecord } from '@/core/utils/object';
import type { ObjectRecord } from '../../../utils/types';
import { getCurrentReq, getOrInitReqServiceState } from '../../runtime/context';
import { withRecordRuleAndFieldRuleBypass } from '../repository/authz';

type SudoAuditEntry = {
  version: 1;
  source: 'sudo';
  at: string;
  hint?: string;
};

type SudoAuditBucket = { sudoHits: SudoAuditEntry[] };

function ensureSudoHits(carrier: ObjectRecord): SudoAuditBucket {
  if (!Array.isArray(carrier.sudoHits)) carrier.sudoHits = [];
  return carrier as SudoAuditBucket;
}

/**
 * Prefer request-scoped service state so sudoHits do not accumulate across requests.
 * Fall back to the process global bucket when no req is available (scripts / D10).
 */
function resolveSudoAuditBucket(): SudoAuditBucket {
  const state = asObjectRecord(getOrInitReqServiceState(getCurrentReq()));
  if (state) return ensureSudoHits(state);

  const bucketRecord = asObjectRecord(getRuntimeComputeAuditBucketValue());
  const bucket: ObjectRecord = bucketRecord ?? {};
  const resolved = ensureSudoHits(bucket);
  setRuntimeComputeAuditBucketValue(bucket);
  return resolved;
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
  return withRecordRuleAndFieldRuleBypass(fn);
}
