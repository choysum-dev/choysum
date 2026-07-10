// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createServiceByModel } from '@/core/service/rpc/service_factory';
import type { ConditionEnvelope, ConditionExpr, RecordRuleOp } from '@/core/service/api/authz';
import { GrpcCode } from '../error';
import { newDocumentError, DocumentErrCode } from '../error';
import { observePermissionDenied } from './_owner_authorization_observability';

type OwnerWriteOperation = 'create' | 'update';
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

type OwnerWriteAuthorizationInput = {
  stage: OwnerPermissionStage;
  ownerModel: string;
  ownerRecordId?: string;
  fieldName: string;
  operation: OwnerWriteOperation;
  companyId: string;
  companyIds?: string[];
  userId: string;
};

type OwnerReadAuthorizationInput = {
  stage: OwnerPermissionStage;
  ownerModel: string;
  ownerRecordId: string;
  fieldName: string;
  companyId: string;
  companyIds?: string[];
  userId: string;
};

type AuthUserServiceLike = {
  GetRecordRuleCondition(model: string, op: RecordRuleOp): Promise<unknown>;
  GetFieldRuleSpec(model: string): Promise<unknown>;
};

type OwnerModelServiceLike = {
  Search(condition: unknown, options?: unknown): Promise<unknown>;
};

type FieldRuleSpec = {
  denyReadFields: string[];
  denyWriteFields: string[];
};

const AUTH_USER_MODEL = 'auth.User';

/**
 * Verifies write access to the owner model and field used by a document mutation.
 */
export async function assertOwnerWriteAuthorization(input: OwnerWriteAuthorizationInput): Promise<void> {
  const stage = input.stage;
  const ownerModel = requireText(input.ownerModel, 'ownerModel', stage);
  const fieldName = requireText(input.fieldName, 'fieldName', stage);
  const companyId = requireText(input.companyId, 'companyId', stage);
  const companyIds = normalizeCompanyIds(input.companyIds, companyId);
  const userId = requireText(input.userId, 'userId', stage);
  const operation = input.operation;

  const op: RecordRuleOp = operation === 'create' ? 'create' : 'write';
  const envelope = await fetchRecordRuleEnvelope(ownerModel, op, stage);

  if (envelope.kind === 'false') {
    throw permissionDenied(stage, 'owner write is denied by record rule', {
      ownerModel,
      fieldName,
      op,
      reason: envelope.reason ?? 'record_rule_false',
      companyId,
    });
  }

  const fieldRuleSpec = await fetchFieldRuleSpec(ownerModel, stage);
  if (isFieldDenied(fieldRuleSpec.denyWriteFields, fieldName)) {
    throw permissionDenied(stage, 'owner field write is denied by field rule', {
      ownerModel,
      fieldName,
      access: 'write',
      companyId,
    });
  }

  const ownerRecordId = normalizeOptionalText(input.ownerRecordId);
  if (operation === 'update' && !ownerRecordId) {
    throw permissionDenied(stage, 'ownerRecordId is required for owner write check', {
      ownerModel,
      fieldName,
      op,
      companyId,
    });
  }

  if (ownerRecordId && envelope.kind === 'expr') {
    const recordRuleExpr = replaceConditionTokens(envelope.expr, userId, companyId, companyIds);
    const ok = await probeOwnerRecord(ownerModel, ownerRecordId, recordRuleExpr);
    if (!ok) {
      throw permissionDenied(stage, 'owner write target is not allowed by record rule scope', {
        ownerModel,
        ownerRecordId,
        fieldName,
        op,
        companyId,
      });
    }
  }
}

/**
 * Verifies read access to the owner model and field used by a document download flow.
 */
export async function assertOwnerReadAuthorization(input: OwnerReadAuthorizationInput): Promise<void> {
  const stage = input.stage;
  const ownerModel = requireText(input.ownerModel, 'ownerModel', stage);
  const ownerRecordId = requireText(input.ownerRecordId, 'ownerRecordId', stage);
  const fieldName = requireText(input.fieldName, 'fieldName', stage);
  const companyId = requireText(input.companyId, 'companyId', stage);
  const companyIds = normalizeCompanyIds(input.companyIds, companyId);
  const userId = requireText(input.userId, 'userId', stage);

  const envelope = await fetchRecordRuleEnvelope(ownerModel, 'read', stage);

  if (envelope.kind === 'false') {
    throw permissionDenied(stage, 'owner read is denied by record rule', {
      ownerModel,
      ownerRecordId,
      fieldName,
      op: 'read',
      reason: envelope.reason ?? 'record_rule_false',
      companyId,
    });
  }

  const fieldRuleSpec = await fetchFieldRuleSpec(ownerModel, stage);
  if (isFieldDenied(fieldRuleSpec.denyReadFields, fieldName)) {
    throw permissionDenied(stage, 'owner field read is denied by field rule', {
      ownerModel,
      fieldName,
      access: 'read',
      companyId,
    });
  }

  if (envelope.kind === 'expr') {
    const recordRuleExpr = replaceConditionTokens(envelope.expr, userId, companyId, companyIds);
    const ok = await probeOwnerRecord(ownerModel, ownerRecordId, recordRuleExpr);
    if (!ok) {
      throw permissionDenied(stage, 'owner read target is not allowed by record rule scope', {
        ownerModel,
        ownerRecordId,
        fieldName,
        op: 'read',
        companyId,
      });
    }
  }
}

async function fetchRecordRuleEnvelope(ownerModel: string, op: RecordRuleOp, stage: OwnerPermissionStage): Promise<ConditionEnvelope> {
  const authService = getAuthUserService(stage);
  try {
    const raw = await authService.GetRecordRuleCondition(ownerModel, op);
    return normalizeConditionEnvelope(raw);
  } catch (err) {
    throw permissionDenied(stage, 'failed to fetch owner record rule condition', {
      ownerModel,
      op,
      detail: errorMessage(err),
    });
  }
}

async function fetchFieldRuleSpec(ownerModel: string, stage: OwnerPermissionStage): Promise<FieldRuleSpec> {
  const authService = getAuthUserService(stage);
  try {
    const raw = await authService.GetFieldRuleSpec(ownerModel);
    return normalizeFieldRuleSpec(raw);
  } catch (err) {
    throw permissionDenied(stage, 'failed to fetch owner field rule spec', {
      ownerModel,
      detail: errorMessage(err),
    });
  }
}

function getAuthUserService(stage: OwnerPermissionStage): AuthUserServiceLike {
  try {
    return createServiceByModel(AUTH_USER_MODEL) as AuthUserServiceLike;
  } catch (err) {
    throw permissionDenied(stage, 'auth service is unavailable for owner authorization check', {
      model: AUTH_USER_MODEL,
      detail: errorMessage(err),
    });
  }
}

async function probeOwnerRecord(ownerModel: string, ownerRecordId: string, recordRuleExpr?: ConditionExpr): Promise<boolean> {
  let ownerService: OwnerModelServiceLike;
  try {
    ownerService = createServiceByModel(ownerModel) as OwnerModelServiceLike;
  } catch {
    return false;
  }

  const query = recordRuleExpr
    ? ({
        And: [['Id', '=', ownerRecordId], recordRuleExpr],
      } as any)
    : (['Id', '=', ownerRecordId] as any);

  try {
    const rows = await ownerService.Search(query, { limit: 1, fields: ['Id'] } as any);
    return Array.isArray(rows) && rows.length > 0;
  } catch {
    return false;
  }
}

function normalizeConditionEnvelope(value: unknown): ConditionEnvelope {
  const record = asRecord(value);
  if (!record) return { kind: 'false', reason: 'invalid_record_rule_envelope' };

  const kind = normalizeOptionalText(record.kind);
  if (kind === 'true') return { kind: 'true', reason: normalizeOptionalText(record.reason) };
  if (kind === 'false') return { kind: 'false', reason: normalizeOptionalText(record.reason) };
  if (kind === 'expr' && (Array.isArray(record.expr) || asRecord(record.expr))) {
    return {
      kind: 'expr',
      expr: record.expr as ConditionExpr,
      reason: normalizeOptionalText(record.reason),
    };
  }
  return { kind: 'false', reason: 'invalid_record_rule_envelope' };
}

function normalizeFieldRuleSpec(value: unknown): FieldRuleSpec {
  const record = asRecord(value);
  if (!record) {
    return { denyReadFields: [], denyWriteFields: [] };
  }

  return {
    denyReadFields: normalizeTextArray(record.denyReadFields),
    denyWriteFields: normalizeTextArray(record.denyWriteFields),
  };
}

function replaceConditionTokens(expr: ConditionExpr, userId: string, companyId: string, companyIds: string[]): ConditionExpr {
  const replace = (value: unknown): unknown => {
    if (value === null || value === undefined) return value;
    if (typeof value === 'string') {
      const token = value.trim();
      if (token === '$userId') return userId;
      if (token === '$companyId') return companyId;
      if (token === '$companyIds') return companyIds;
      return value;
    }
    if (Array.isArray(value)) return value.map(item => replace(item));
    const record = asRecord(value);
    if (!record) return value;

    const out: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(record)) {
      out[k] = replace(v);
    }
    return out;
  };

  return replace(expr) as ConditionExpr;
}

function normalizeCompanyIds(value: unknown, activeCompanyId: string): string[] {
  const out: string[] = [];
  if (Array.isArray(value)) {
    for (const item of value) {
      const text = normalizeOptionalText(item);
      if (text) out.push(text);
    }
  }
  if (out.length === 0 && activeCompanyId) out.push(activeCompanyId);
  if (activeCompanyId && !out.includes(activeCompanyId)) out.unshift(activeCompanyId);
  return Array.from(new Set(out));
}

function isFieldDenied(deniedFields: string[], fieldName: string): boolean {
  const target = fieldName.trim().toLowerCase();
  return deniedFields.some(field => field.trim().toLowerCase() === target);
}

function normalizeTextArray(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  const out: string[] = [];
  for (const item of value) {
    const text = normalizeOptionalText(item);
    if (text) out.push(text);
  }
  return Array.from(new Set(out));
}

function permissionDenied(stage: OwnerPermissionStage, message: string, metadata: Record<string, unknown>): Error {
  observePermissionDenied(stage, message, metadata);
  return newDocumentError({
    code: DocumentErrCode.PERMISSION_DENIED,
    message,
  })
    .withGrpcCode(GrpcCode.PermissionDenied)
    .withMetadata(stringifyMetadata({ stage, ...metadata }));
}

function stringifyMetadata(metadata: Record<string, unknown>): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [k, v] of Object.entries(metadata)) {
    const text = normalizeOptionalText(v);
    if (text) out[k] = text;
  }
  return out;
}

function requireText(value: unknown, fieldName: string, stage: OwnerPermissionStage): string {
  const text = normalizeOptionalText(value);
  if (!text) {
    throw permissionDenied(stage, `${fieldName} is required`, { field: fieldName });
  }
  return text;
}

function normalizeOptionalText(value: unknown): string | undefined {
  const text = String(value ?? '').trim();
  return text === '' ? undefined : text;
}

function asRecord(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null;
  return value as Record<string, unknown>;
}

function errorMessage(err: unknown): string {
  if (err instanceof Error) {
    return normalizeOptionalText(err.message) ?? 'unknown_error';
  }
  return normalizeOptionalText(err) ?? 'unknown_error';
}
