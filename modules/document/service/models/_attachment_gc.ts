// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeOptionalString, normalizeOptionalNonNegativeInt, asRecord } from '@/core/service/utils/normalization';
import { parseISODate, toDate } from '@/core/service/utils/date';
import { getBackendEnvPositiveInt } from '@/core/service/runtime/env/backend_env';
import { computeRetryBackoffSeconds } from '@/core/service/utils/backoff';
import { resolveGcBatchSize } from './_gc_config';

const DEFAULT_UNBOUND_OBJECT_GRACE_SECONDS = 24 * 60 * 60;
const DEFAULT_CLEANUP_MAX_ATTEMPTS = 8;
const DEFAULT_CLEANUP_RETRY_BASE_SECONDS = 30;

type CleanupStateValue = 'retrying' | 'failed' | 'deleted';

type CleanupState = {
  state?: CleanupStateValue;
  attempts?: number;
  nextRetryAt?: string;
  lastError?: string;
  at?: string;
};

export type GcModelOps = {
  Search(condition: unknown, options?: unknown): Promise<unknown[]>;
  UpdateById(id: string, values: unknown, fields?: unknown): Promise<unknown>;
};

function readCleanupState(metadata: Record<string, unknown> | undefined): CleanupState {
  const cleanup = asRecord(metadata?.cleanup);
  if (!cleanup) {
    return {};
  }
  const state = normalizeOptionalString(cleanup.state) as CleanupStateValue | undefined;
  const attempts = normalizeOptionalNonNegativeInt(cleanup.attempts);
  const nextRetryAt = normalizeOptionalString(cleanup.nextRetryAt);
  const lastError = normalizeOptionalString(cleanup.lastError);
  const at = normalizeOptionalString(cleanup.at);
  return { state, attempts, nextRetryAt, lastError, at };
}

function writeCleanupState(metadata: Record<string, unknown> | undefined, state: CleanupState): Record<string, unknown> {
  const nextMetadata: Record<string, unknown> = {
    ...(metadata || {}),
    cleanup: {
      state: state.state,
      attempts: state.attempts,
      nextRetryAt: state.nextRetryAt,
      lastError: state.lastError,
      at: state.at,
    },
  };
  return nextMetadata;
}

export async function garbageCollectUnboundObjects(
  modelOps: GcModelOps,
  nowISO?: string
): Promise<{ scannedCount: number; deletedCount: number; retriedCount: number; failedCount: number; skippedCount: number }> {
  const now = parseISODate(nowISO);
  const nowAt = now.toISOString();
  const batch = resolveGcBatchSize();
  const unboundGraceSeconds = getBackendEnvPositiveInt(
    ['CHOYSUM_DOCUMENT_ATTACHMENT_UNBOUND_OBJECT_GRACE_SECONDS', 'CHOYSUM_DOCUMENT_UNBOUND_OBJECT_GRACE_SECONDS'],
    DEFAULT_UNBOUND_OBJECT_GRACE_SECONDS
  );
  const maxAttempts = getBackendEnvPositiveInt(['CHOYSUM_DOCUMENT_CLEANUP_MAX_ATTEMPTS'], DEFAULT_CLEANUP_MAX_ATTEMPTS);
  const retryBaseSeconds = getBackendEnvPositiveInt(['CHOYSUM_DOCUMENT_CLEANUP_RETRY_BASE_SECONDS'], DEFAULT_CLEANUP_RETRY_BASE_SECONDS);
  const graceCutoff = new Date(now.getTime() - unboundGraceSeconds * 1000);

  const AttachmentBinding = (await import('./attachment_binding')).default;

  let scannedCount = 0;
  let deletedCount = 0;
  let retriedCount = 0;
  let failedCount = 0;
  let skippedCount = 0;
  let lastId: string | null = null;

  for (;;) {
    const baseConditions: any[] = [
      ['Status', '=', 'active'],
      ['UpdatedAt', '<', graceCutoff],
    ];
    if (lastId) {
      baseConditions.push(['Id', '>', lastId]);
    }

    const candidates = await modelOps.Search({ And: baseConditions } as any, { limit: batch, orderBy: { field: 'Id', order: 'asc' } as any } as any);
    if (!candidates.length) break;

    const candidateIds = candidates.map(c => normalizeOptionalString((c as any)?.Id)).filter((id): id is string => Boolean(id));
    const activeBindings =
      candidateIds.length > 0
        ? await AttachmentBinding.Search(
            {
              And: [
                ['AttachmentContentId', 'in', candidateIds],
                ['Status', '=', 'active'],
              ],
            } as any,
            { fields: ['AttachmentContentId'] as any } as any
          )
        : [];
    const activeContentIds = new Set(activeBindings.map(b => normalizeOptionalString((b as any)?.AttachmentContentId)).filter(Boolean));

    for (const candidate of candidates) {
      scannedCount += 1;
      const contentId = normalizeOptionalString((candidate as any)?.Id);
      if (!contentId) {
        skippedCount += 1;
        continue;
      }

      if (activeContentIds.has(contentId)) {
        skippedCount += 1;
        continue;
      }

      const metadata = asRecord((candidate as any)?.MetadataJson) ?? undefined;
      const cleanup = readCleanupState(metadata);
      const attempts = Math.max(0, Math.trunc(Number(cleanup.attempts || 0)));
      const state = normalizeOptionalString(cleanup.state)?.toLowerCase();
      if (state === 'failed' && attempts >= maxAttempts) {
        skippedCount += 1;
        continue;
      }

      const nextRetryAt = toDate(cleanup.nextRetryAt);
      if (nextRetryAt && nextRetryAt.getTime() > now.getTime()) {
        skippedCount += 1;
        continue;
      }

      const nextAttempt = attempts + 1;
      try {
        const storedContentId = normalizeOptionalString((candidate as any)?.StoredContentId);
        if (!storedContentId) throw new Error('attachment content missing storedContentId');

        const documentBridge = (globalThis as any)?.$choysum?.document;
        const deleteStoredContent =
          typeof documentBridge?.deleteStoredContent === 'function' ? documentBridge.deleteStoredContent.bind(documentBridge) : undefined;
        if (!deleteStoredContent) throw new Error('document.deleteStoredContent bridge is unavailable');
        try {
          await deleteStoredContent({ storedContentId });
        } catch (err: any) {
          const errMsg = String(err?.message || err).toLowerCase();
          const isNotFound =
            errMsg.includes('not found') || errMsg.includes('nosuchkey') || err?.code === 'NoSuchKey' || err?.status === 404;
          if (!isNotFound) throw err;
        }

        await modelOps.UpdateById(
          contentId,
          {
            Status: 'deleted',
            MetadataJson: writeCleanupState(metadata, { state: 'deleted', attempts: nextAttempt, at: nowAt }),
          } as any,
          ['Id', 'Status', 'MetadataJson'] as any
        );
        deletedCount += 1;
      } catch (error) {
        const message = String((error as any)?.message || error || 'attachment cleanup failed');
        const terminal = nextAttempt >= maxAttempts;
        const nextRetryAtISO = terminal ? undefined : new Date(now.getTime() + computeRetryBackoffSeconds(nextAttempt, retryBaseSeconds) * 1000).toISOString();
        try {
          const errorStateValues: Record<string, unknown> = {
            MetadataJson: writeCleanupState(metadata, {
              state: terminal ? 'failed' : 'retrying',
              attempts: nextAttempt,
              nextRetryAt: nextRetryAtISO,
              lastError: message.slice(0, 1024),
              at: nowAt,
            }),
          };
          const errorStateFields: string[] = ['Id', 'MetadataJson'];
          if (terminal) {
            errorStateValues.UpdatedAt = now;
            errorStateFields.push('UpdatedAt');
          }
          await modelOps.UpdateById(contentId, errorStateValues as any, errorStateFields as any);
        } catch {
          // Metadata update is best-effort; don't abort the entire GC run.
        }
        if (terminal) failedCount += 1;
        else retriedCount += 1;
      }
    }
    const lastCandidate = candidates[candidates.length - 1];
    const nextLastId = normalizeOptionalString((lastCandidate as any)?.Id);
    if (!nextLastId) {
      throw new Error('Garbage collection aborted: candidate is missing a valid Id');
    }
    lastId = nextLastId;
    if (candidates.length < batch) break;
  }

  return { scannedCount, deletedCount, retriedCount, failedCount, skippedCount };
}
