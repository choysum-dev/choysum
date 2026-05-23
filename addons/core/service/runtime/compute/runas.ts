// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ComputeRunAs } from '../../orm/metadata/compute';
import type { ModelMetadata } from '../../orm/metadata/model';
import { getRuntimeComputeAuditBucketValue, setRuntimeComputeAuditBucketValue } from '@/core/utils/env';
import { asObjectRecord } from '@/core/utils/object';
import { withContext } from '../context';
import type { ObjectRecord } from '../../../utils/types';

type ComputeAuditPhase = 'expr' | 'search';

type ComputeAuditEntry = {
  version: 1;
  model: string;
  field: string;
  runAs: ComputeRunAs;
  mode?: string;
  phase: ComputeAuditPhase;
  at: string;
};

function resolveComputeAuditBucket(): { runAsHits: ComputeAuditEntry[] } {
  const bucketRecord = asObjectRecord(getRuntimeComputeAuditBucketValue());
  const bucket: ObjectRecord = bucketRecord ?? {};
  if (!Array.isArray(bucket.runAsHits)) bucket.runAsHits = [];
  setRuntimeComputeAuditBucketValue(bucket);
  return bucket as { runAsHits: ComputeAuditEntry[] };
}

function buildAuditEntry(meta: ModelMetadata, field: string, runAs: ComputeRunAs, phase: ComputeAuditPhase, mode?: string): ComputeAuditEntry {
  const model = String(meta.fullModelName || meta.modelName || meta.className || 'Unknown').trim() || 'Unknown';
  return {
    version: 1,
    model,
    field: String(field || '').trim() || 'unknown',
    runAs,
    mode: mode ? String(mode || '').trim() : undefined,
    phase,
    at: new Date().toISOString(),
  };
}

export function recordComputeRunAsAudit(meta: ModelMetadata, field: string, runAs: ComputeRunAs, phase: ComputeAuditPhase, mode?: string): void {
  if (runAs !== 'sudo') return;
  const bucket = resolveComputeAuditBucket();
  bucket.runAsHits.push(buildAuditEntry(meta, field, runAs, phase, mode));
}

export function withComputeRunAsExecution<T>(meta: ModelMetadata, field: string, runAs: ComputeRunAs, phase: ComputeAuditPhase, fn: () => T, mode?: string): T {
  if (runAs !== 'sudo') return fn();

  recordComputeRunAsAudit(meta, field, runAs, phase, mode);
  const marker = buildAuditEntry(meta, field, runAs, phase, mode);
  return withContext(
    {
      __computeRunAs: 'sudo',
      __computeAuditMarker: marker,
    },
    fn,
    { merge: true }
  );
}
