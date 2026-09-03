// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { dial } from '@/core/service';
import { assertRecordReadable } from '@/core/service/orm/model';
import { normalizeConditionEnvelope, normalizeFieldRuleSpec, replaceConditionExprTokens } from '@/core/service/api/authz';
import type { ConditionEnvelope, ConditionExpr, FieldRuleSpec, RecordRuleOp } from '@/core/service/api/authz';
import { normalizeOptionalString } from '@/core/service/utils/normalization';
import { createTranslate } from '@/core/service/i18n';
import { GrpcCode } from '../error';
import { newDocumentError, DocumentErrCode } from '../error';
import { observePermissionDenied } from './_owner_authorization_observability';
import { normalizeLooseOptionalText, normalizeCompanyIdList } from './_document_bridge';

const { _t } = createTranslate('document');

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

const AUTH_USER_MODEL = 'auth.User';

/**
 * Verifies write access to the owner model and field used by a document mutation.
 */
export async function assertOwnerWriteAuthorization(input: OwnerWriteAuthorizationInput): Promise<void> {
  const stage = input.stage;
  const ownerModel = requireText(input.ownerModel, 'ownerModel', stage);
  const fieldName = requireText(input.fieldName, 'fieldName', stage);
  const companyId = requireText(input.companyId, 'companyId', stage);
  const companyIds = normalizeCompanyIdList(input.companyIds, companyId);
  const userId = requireText(input.userId, 'userId', stage);
  const operation = input.operation;

  const op: RecordRuleOp = operation === 'create' ? 'create' : 'write';
  const envelope = await fetchRecordRuleEnvelope(ownerModel, op, stage);

  if (envelope.kind === 'false') {
    throw permissionDenied(stage, _t('owner write is denied by record rule', { scope: 'service/models/_owner_authorization' }), {
      ownerModel,
      fieldName,
      op,
      reason: envelope.reason ?? 'record_rule_false',
      companyId,
    });
  }

  const fieldRuleSpec = await fetchFieldRuleSpec(ownerModel, stage);
  if (isFieldDenied(fieldRuleSpec.denyWriteFields, fieldName)) {
    throw permissionDenied(stage, _t('owner field write is denied by field rule', { scope: 'service/models/_owner_authorization' }), {
      ownerModel,
      fieldName,
      access: 'write',
      companyId,
    });
  }

  const ownerRecordId = normalizeLooseOptionalText(input.ownerRecordId);
  if (operation === 'update' && !ownerRecordId) {
    throw permissionDenied(stage, _t('ownerRecordId is required for owner write check', { scope: 'service/models/_owner_authorization' }), {
      ownerModel,
      fieldName,
      op,
      companyId,
    });
  }

  if (ownerRecordId && envelope.kind === 'expr') {
    const recordRuleExpr = replaceTokensForOwnerRecordRule(envelope.expr, stage, userId, companyId, companyIds);
    const ok = await probeOwnerRecord(stage, ownerModel, ownerRecordId, recordRuleExpr);
    if (!ok) {
      throw permissionDenied(stage, _t('owner write target is not allowed by record rule scope', { scope: 'service/models/_owner_authorization' }), {
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
  const companyIds = normalizeCompanyIdList(input.companyIds, companyId);
  const userId = requireText(input.userId, 'userId', stage);

  const envelope = await fetchRecordRuleEnvelope(ownerModel, 'read', stage);

  if (envelope.kind === 'false') {
    throw permissionDenied(stage, _t('owner read is denied by record rule', { scope: 'service/models/_owner_authorization' }), {
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
    throw permissionDenied(stage, _t('owner field read is denied by field rule', { scope: 'service/models/_owner_authorization' }), {
      ownerModel,
      fieldName,
      access: 'read',
      companyId,
    });
  }

  if (envelope.kind === 'expr') {
    const recordRuleExpr = replaceTokensForOwnerRecordRule(envelope.expr, stage, userId, companyId, companyIds);
    const ok = await probeOwnerRecord(stage, ownerModel, ownerRecordId, recordRuleExpr);
    if (!ok) {
      throw permissionDenied(stage, _t('owner read target is not allowed by record rule scope', { scope: 'service/models/_owner_authorization' }), {
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
    throw permissionDenied(stage, _t('failed to fetch owner record rule condition', { scope: 'service/models/_owner_authorization' }), {
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
    throw permissionDenied(stage, _t('failed to fetch owner field rule spec', { scope: 'service/models/_owner_authorization' }), {
      ownerModel,
      detail: errorMessage(err),
    });
  }
}

function getAuthUserService(stage: OwnerPermissionStage): AuthUserServiceLike {
  try {
    return dial<AuthUserServiceLike>(AUTH_USER_MODEL);
  } catch (err) {
    throw permissionDenied(stage, _t('auth service is unavailable for owner authorization check', { scope: 'service/models/_owner_authorization' }), {
      model: AUTH_USER_MODEL,
      detail: errorMessage(err),
    });
  }
}

async function probeOwnerRecord(
  stage: OwnerPermissionStage,
  ownerModel: string,
  ownerRecordId: string,
  recordRuleExpr?: ConditionExpr
): Promise<boolean> {
  if (!recordRuleExpr) {
    try {
      await assertRecordReadable(ownerModel, ownerRecordId, {
        message: _t('owner target is not readable', { scope: 'service/models/_owner_authorization' }),
      });
      return true;
    } catch {
      return false;
    }
  }

  let ownerService: OwnerModelServiceLike;
  try {
    ownerService = dial<OwnerModelServiceLike>(ownerModel);
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

function replaceTokensForOwnerRecordRule(
  expr: ConditionExpr,
  stage: OwnerPermissionStage,
  userId: string,
  companyId: string,
  companyIds: string[]
): ConditionExpr {
  try {
    return replaceConditionExprTokens(expr, {
      userId,
      companyId,
      companyIds,
      strictUnknownToken: true,
    });
  } catch (err) {
    throw permissionDenied(stage, _t('owner record rule contains invalid token mapping', { scope: 'service/models/_owner_authorization' }), {
      detail: errorMessage(err),
    });
  }
}

function isFieldDenied(deniedFields: string[], fieldName: string): boolean {
  const target = fieldName.trim().toLowerCase();
  return deniedFields.some(field => field.trim().toLowerCase() === target);
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
    const text = normalizeLooseOptionalText(v);
    if (text) out[k] = text;
  }
  return out;
}

function requireText(value: unknown, fieldName: string, stage: OwnerPermissionStage): string {
  const text = normalizeLooseOptionalText(value);
  if (!text) {
    throw permissionDenied(stage, _t('%s is required', { scope: 'service/models/_owner_authorization' }, fieldName), { field: fieldName });
  }
  return text;
}

function errorMessage(err: unknown): string {
  if (err instanceof Error) {
    return normalizeLooseOptionalText(err.message) ?? 'unknown_error';
  }
  return normalizeLooseOptionalText(err) ?? 'unknown_error';
}

/** Test seam for owner record probe branches (Id-only and expr-augmented). */
export async function documentProbeOwnerRecordForTest(
  stage: OwnerPermissionStage,
  ownerModel: string,
  ownerRecordId: string,
  recordRuleExpr?: ConditionExpr
): Promise<boolean> {
  return probeOwnerRecord(stage, ownerModel, ownerRecordId, recordRuleExpr);
}
