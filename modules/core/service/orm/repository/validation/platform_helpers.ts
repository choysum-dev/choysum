// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../metadata';
import type { ConstraintMode } from '../../metadata';
import { asObjectRecord } from '@/core/utils/object';
import type { ObjectRecord } from '../../../../utils/types';

type ValidationAuditBucket = {
  version?: number;
  platformCreateWhitelistHits?: Array<ObjectRecord>;
};

type MutableRecord = ObjectRecord;

function toMutableRecord(value: unknown): MutableRecord | undefined {
  return value && typeof value === 'object' ? (value as MutableRecord) : undefined;
}

function getContextRecord(value: unknown): MutableRecord {
  return toMutableRecord(value) || {};
}

function hasOwn(record: unknown, key: string): boolean {
  const objectRecord = asObjectRecord(record);
  if (!objectRecord) return false;
  return Object.prototype.hasOwnProperty.call(objectRecord, key);
}

function normalizeBucket(raw: unknown): ValidationAuditBucket {
  const record = asObjectRecord(raw) || {};
  const hits = Array.isArray(record.platformCreateWhitelistHits)
    ? record.platformCreateWhitelistHits.filter(item => !!item && typeof item === 'object').map(item => ({ ...(item as ObjectRecord) }))
    : [];
  return {
    version: typeof record.version === 'number' ? record.version : 1,
    platformCreateWhitelistHits: hits,
  };
}

export function resolveRepositoryPlatformRejectUnknownFields(requestContext: unknown): boolean {
  const ctx = getContextRecord(requestContext);
  const validation = asObjectRecord(ctx.validation);
  return Boolean(validation?.platformRejectUnknownFields ?? ctx.platformRejectUnknownFields ?? false);
}

function resolveValidationAuditBucket(ctx: unknown): {
  bucket: {
    version?: number;
    platformCreateWhitelistHits?: Array<ObjectRecord>;
  };
  source: 'request_context' | 'global_fallback';
} {
  const contextRecord = toMutableRecord(ctx);
  if (contextRecord && Object.isExtensible(contextRecord)) {
    const key = '__validationAudit';
    const bucket = normalizeBucket(contextRecord[key]);
    contextRecord[key] = bucket;
    return { bucket, source: 'request_context' };
  }

  const root = globalThis as unknown as MutableRecord;
  const globalKey = '__choysumValidationAudit';
  const bucket = normalizeBucket(root[globalKey]);
  root[globalKey] = bucket;
  return { bucket, source: 'global_fallback' };
}

export function recordRepositoryPlatformCreateWhitelistAudit(meta: ModelMetadata, requestContext: unknown, mode: ConstraintMode, fields: string[]): void {
  if (mode !== 'create' || !Array.isArray(fields) || fields.length === 0) {
    return;
  }

  const ctx = getContextRecord(requestContext);
  const ctxRequestId = typeof ctx.requestId === 'string' || typeof ctx.requestId === 'number' ? String(ctx.requestId).trim() : '';
  const ctxRequestIdLegacy = typeof ctx.RequestId === 'string' || typeof ctx.RequestId === 'number' ? String(ctx.RequestId).trim() : '';
  const { bucket, source } = resolveValidationAuditBucket(ctx);

  const model = String(meta.fullModelName || meta.modelName || meta.name || '').trim() || 'unknown';
  const requestId = ctxRequestId || ctxRequestIdLegacy || undefined;
  const entry = {
    version: 1,
    source,
    model,
    mode,
    fields: Array.from(new Set(fields.map(f => String(f || '').trim()).filter(Boolean))).sort(),
    at: new Date().toISOString(),
    requestId,
  };

  const entries = (bucket.platformCreateWhitelistHits ||= []) as Array<ObjectRecord>;
  entries.push(entry);
}

export function resolveRepositoryPlatformCreateWriteWhitelist(meta: ModelMetadata, requestContext: unknown): string[] {
  const ctx = getContextRecord(requestContext);
  const validation = asObjectRecord(ctx.validation);
  const modelName = String(meta.fullModelName || meta.modelName || meta.name || '').trim();

  const normalizeList = (input: unknown): string[] => {
    if (!Array.isArray(input)) return [];
    return input.map(item => String(item || '').trim()).filter(Boolean);
  };

  const byModelInValidation = asObjectRecord(validation?.platformCreateWriteWhitelistByModel);
  if (hasOwn(byModelInValidation, modelName)) {
    return Array.from(new Set(normalizeList(byModelInValidation?.[modelName])));
  }

  const byModelAtRoot = asObjectRecord(ctx.platformCreateWriteWhitelistByModel);
  if (hasOwn(byModelAtRoot, modelName)) {
    return Array.from(new Set(normalizeList(byModelAtRoot?.[modelName])));
  }

  const globalInValidation = normalizeList(validation?.platformCreateWriteWhitelist);
  if (globalInValidation.length > 0) {
    return Array.from(new Set(globalInValidation));
  }

  return Array.from(new Set(normalizeList(ctx.platformCreateWriteWhitelist)));
}
