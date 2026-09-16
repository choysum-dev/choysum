// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getUserId } from '../../api/context';
import { invalidateAllAuthzCaches, invalidateAuthzCachesForUsers } from '../../api/authz_request_cache';
import { raiseDomainError } from '../../error';
import { normalizeRefId, uniqStrings } from '../../utils/normalization';
import type { ModelMetadata } from '../metadata/model';
import type BaseModel from './model';
import type { ModelCtor } from './types';
import type { ObjectRecord } from '../../../utils/types';

/** Append-only policy: boolean or domain/code/message override. */
export type AppendOnlyPolicy =
  | boolean
  | {
      domain?: string;
      code?: string;
      message?: string;
    };

/** Post-write authz cache invalidation policy. */
export type AfterMutationPolicy = {
  invalidateAuthz: 'all' | 'usersFromPayload';
};

/** Sync prepare hook: method name on the model ctor, or a function. */
export type PrepareWritePolicy = string | ((value: ObjectRecord) => ObjectRecord | void);

/** Write-policy fields stored on model metadata / @Model options. */
export type ModelWritePolicyFields = {
  appendOnly?: AppendOnlyPolicy;
  stampActor?: string;
  prepareCreate?: PrepareWritePolicy;
  prepareUpdate?: PrepareWritePolicy;
  afterMutation?: AfterMutationPolicy;
};

export type WriteMutationOperation = 'create' | 'update' | 'delete';

export type AfterMutationContext = {
  operation: WriteMutationOperation;
  /** Create/update payloads (single row or many). */
  payloads?: ObjectRecord | ObjectRecord[] | null;
  /** Pre-mutation row snapshots (update/delete). */
  beforeEntities?: ObjectRecord[] | null;
};

type UserRoleUserIdPayload = {
  UserId?: string | { Id: string } | null;
};

type AuthzInvalidators = {
  invalidateAll: () => void;
  invalidateUsers: (userIds: string[]) => void;
};

let authzInvalidators: AuthzInvalidators = {
  invalidateAll: invalidateAllAuthzCaches,
  invalidateUsers: invalidateAuthzCachesForUsers,
};

/**
 * Test-only override for afterMutation authz invalidators.
 */
export function __setWritePolicyAuthzInvalidatorsForTest(next?: Partial<AuthzInvalidators> | null): void {
  authzInvalidators = {
    invalidateAll: next?.invalidateAll ?? invalidateAllAuthzCaches,
    invalidateUsers: next?.invalidateUsers ?? invalidateAuthzCachesForUsers,
  };
}

/**
 * Collect UserId refs from UserRole-shaped create/update payloads.
 */
export function userIdsFromUserRolePayloads(
  values: Partial<UserRoleUserIdPayload> | Array<Partial<UserRoleUserIdPayload>> | null | undefined
): string[] {
  const rows = Array.isArray(values) ? values : values != null ? [values] : [];
  return uniqStrings(rows.map(v => normalizeRefId(v?.UserId)));
}

function userIdsFromEntities(rows: ObjectRecord[] | null | undefined): string[] {
  if (!rows?.length) return [];
  return uniqStrings(rows.map(row => normalizeRefId((row as UserRoleUserIdPayload)?.UserId)));
}

function resolvePrepareFn(
  ModelCtor: ModelCtor<BaseModel>,
  policy: PrepareWritePolicy | undefined
): ((value: ObjectRecord) => ObjectRecord | void) | undefined {
  if (policy == null) return undefined;
  if (typeof policy === 'function') return policy;
  const name = String(policy || '').trim();
  if (!name) return undefined;
  const fn = (ModelCtor as unknown as ObjectRecord)[name];
  if (typeof fn !== 'function') {
    throw new Error(`prepare write policy '${name}' is not a function on ${ModelCtor.name || 'model'}`);
  }
  return (fn as (value: ObjectRecord) => ObjectRecord | void).bind(ModelCtor);
}

/**
 * Run sync prepareCreate when configured; returns the (possibly replaced) payload.
 */
export function applyPrepareCreate<T extends BaseModel>(
  ModelCtor: ModelCtor<T>,
  meta: Pick<ModelMetadata, 'prepareCreate'>,
  value: ObjectRecord
): ObjectRecord {
  const fn = resolvePrepareFn(ModelCtor as ModelCtor<BaseModel>, meta.prepareCreate);
  if (!fn) return value;
  const out = fn(value);
  return out && typeof out === 'object' ? (out as ObjectRecord) : value;
}

/**
 * Run sync prepareUpdate when configured; returns the (possibly replaced) payload.
 */
export function applyPrepareUpdate<T extends BaseModel>(
  ModelCtor: ModelCtor<T>,
  meta: Pick<ModelMetadata, 'prepareUpdate'>,
  value: ObjectRecord
): ObjectRecord {
  const fn = resolvePrepareFn(ModelCtor as ModelCtor<BaseModel>, meta.prepareUpdate);
  if (!fn) return value;
  const out = fn(value);
  return out && typeof out === 'object' ? (out as ObjectRecord) : value;
}

/**
 * Overwrite stampActor field from trusted request identity (`getUserId`).
 * Ignores any caller-supplied value for that field.
 */
export function applyStampActor(meta: Pick<ModelMetadata, 'stampActor'>, value: ObjectRecord): ObjectRecord {
  const field = String(meta.stampActor || '').trim();
  if (!field) return value;
  const uid = getUserId();
  const stamped = uid == null || String(uid).trim() === '' ? null : String(uid).trim();
  return { ...value, [field]: stamped };
}

/**
 * Throw when the model is append-only (Update/Delete paths).
 */
export function assertNotAppendOnly(
  meta: Pick<ModelMetadata, 'appendOnly' | 'modelName' | 'application' | 'fullModelName'>,
  operation: 'Update' | 'UpdateById' | 'Delete' | 'DeleteById'
): void {
  const policy = meta.appendOnly;
  if (!policy) return;

  const modelLabel =
    String(meta.modelName || '').trim() ||
    String(meta.fullModelName || '').trim() ||
    'Model';

  if (policy === true) {
    const domain = String(meta.application || '').trim() || 'core';
    raiseDomainError(domain, 'APPEND_ONLY', `${modelLabel} does not support ${operation}`);
  }

  const domain = String(policy.domain || meta.application || '').trim() || 'core';
  const code = String(policy.code || 'APPEND_ONLY').trim() || 'APPEND_ONLY';
  const message = String(policy.message || `${modelLabel} does not support ${operation}`).trim();
  raiseDomainError(domain, code, message);
}

/**
 * Run afterMutation invalidateAuthz after a successful write.
 *
 * `usersFromPayload` collects UserId from payloads and beforeEntities; when empty,
 * falls back to invalidating every request-scoped authz cache (safe for partial updates).
 */
export function applyAfterMutation(meta: Pick<ModelMetadata, 'afterMutation'>, ctx: AfterMutationContext): void {
  const policy = meta.afterMutation;
  if (!policy?.invalidateAuthz) return;

  if (policy.invalidateAuthz === 'all') {
    authzInvalidators.invalidateAll();
    return;
  }

  const fromPayload = userIdsFromUserRolePayloads(
    ctx.payloads as Partial<UserRoleUserIdPayload> | Array<Partial<UserRoleUserIdPayload>> | null
  );
  const fromBefore = userIdsFromEntities(ctx.beforeEntities);
  const userIds = uniqStrings([...fromPayload, ...fromBefore]);
  if (userIds.length === 0) {
    authzInvalidators.invalidateAll();
    return;
  }
  authzInvalidators.invalidateUsers(userIds);
}
