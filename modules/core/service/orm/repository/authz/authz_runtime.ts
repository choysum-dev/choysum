// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getRuntimeEnvBoolean, getRuntimeEnvValue } from '@/core/utils/env';
import type { RepositoryAuthzDecisionSummary } from './types';
import { asObjectRecord } from '../../../../utils/object';
import type { ObjectRecord } from '../../../../utils/types';

export type RepositoryAuthzDecisionLogMode = 'off' | 'deny' | 'all';

export type RepositoryReqMethodMeta = {
  fullMethod: string;
  method: string;
  companyMode: string;
  recordRuleMode: string;
  fieldRuleMode: string;
};

export type RepositoryCompanyScopeFacts = {
  activeCompanyId: string;
  enabledCompanyIds: string[];
};

type RepositoryReqServiceState = ObjectRecord & {
  depth?: unknown;
  recordRuleBypassDepth?: unknown;
  fieldRuleBypassDepth?: unknown;
  validationBypassDepth?: unknown;
};

function resolveRepositoryRoot(): ObjectRecord | undefined {
  const runtimeRoot = (globalThis as { $choysum?: unknown }).$choysum;
  return asObjectRecord(runtimeRoot);
}

function resolveRepositoryJsCtxRoot(): ObjectRecord | undefined {
  const root = resolveRepositoryRoot();
  if (!root) return undefined;
  const request = asObjectRecord(root.request);
  return asObjectRecord(request?.context) ?? asObjectRecord(root.context) ?? root;
}

function asReqServiceState(value: unknown): RepositoryReqServiceState | undefined {
  const record = asObjectRecord(value);
  return record as RepositoryReqServiceState | undefined;
}

export function getRepositoryAuthzDecisionLogMode(): RepositoryAuthzDecisionLogMode {
  const raw = getRuntimeEnvValue('CHOYSUM_AUTHZ_DECISION_LOG');
  const value = String(raw ?? '')
    .trim()
    .toLowerCase();
  if (value === 'deny') return 'deny';
  if (value === 'all') return 'all';
  return 'off';
}

export function repositoryAuthzDecisionAuditEnabled(): boolean {
  return getRuntimeEnvBoolean('CHOYSUM_AUTHZ_DECISION_AUDIT_ENABLED') ?? false;
}

export function emitRepositoryAuthzDecisionSummary(summary: RepositoryAuthzDecisionSummary): void {
  const mode = getRepositoryAuthzDecisionLogMode();
  const decision = String(summary?.decision ?? '').trim();
  if (mode === 'off') return;
  if (mode === 'deny' && decision !== 'deny') return;

  const payload: ObjectRecord = {
    event: 'authz.decision_summary',
    ...summary,
  };

  try {
    console.error(`[AUTHZ] ${JSON.stringify(payload)}`);
  } catch {
    // ignore
  }
  if (repositoryAuthzDecisionAuditEnabled()) {
    try {
      console.error(`[AUDIT] ${JSON.stringify(payload)}`);
    } catch {
      // ignore
    }
  }
}

export function getRepositoryCurrentReq(): ObjectRecord | undefined {
  const jsCtx = resolveRepositoryJsCtxRoot();
  if (!jsCtx) return undefined;
  const request = asObjectRecord(jsCtx.request);
  const requestContext = asObjectRecord(request?.context);
  const nestedContext = asObjectRecord(jsCtx.context);
  return asObjectRecord(jsCtx.req) ?? asObjectRecord(requestContext?.req) ?? asObjectRecord(nestedContext?.req);
}

export function getRepositoryCurrentReqWrapper(): ObjectRecord | undefined {
  const root = resolveRepositoryRoot();
  return asObjectRecord(root?.request);
}

export function isRepositoryTopLevelGrpcCall(): boolean {
  try {
    const wrapper = getRepositoryCurrentReqWrapper();
    const context = asObjectRecord(wrapper?.context);
    const reqRecord = asObjectRecord(context?.req);
    const kind = reqRecord?.kind;
    const isGrpc = kind === 'grpc' || kind === 'grpc-web';
    if (!isGrpc) return false;

    const callDepth = asReqServiceState(wrapper?.__choysumServiceState)?.depth;
    if (typeof callDepth === 'number') {
      return callDepth === 1;
    }
    const entryDepth = reqRecord?.depth;
    return entryDepth === 0;
  } catch {
    return false;
  }
}

export function getOrInitRepositoryReqServiceState(req: unknown): RepositoryReqServiceState | undefined {
  const reqRecord = asObjectRecord(req);
  if (!reqRecord) return undefined;
  const existing = asReqServiceState(reqRecord.__choysumServiceState);
  if (existing) return existing;
  const created: RepositoryReqServiceState = {};
  reqRecord.__choysumServiceState = created;
  return created;
}

export function getRepositoryReqMethodMeta(): RepositoryReqMethodMeta {
  const req = getRepositoryCurrentReq();
  return {
    fullMethod: typeof req?.fullMethod === 'string' ? req.fullMethod : '',
    method: typeof req?.method === 'string' ? req.method : '',
    companyMode: typeof req?.companyMode === 'string' ? req.companyMode : '',
    recordRuleMode: typeof req?.recordRuleMode === 'string' ? req.recordRuleMode : '',
    fieldRuleMode: typeof req?.fieldRuleMode === 'string' ? req.fieldRuleMode : '',
  };
}

export function getRepositoryCompanyScopeFacts(requestContext: unknown, enabledCompanyIds: string[]): RepositoryCompanyScopeFacts {
  const ctx = asObjectRecord(requestContext);
  return {
    activeCompanyId: String(ctx?.activeCompanyId ?? ctx?.ActiveCompanyId ?? '').trim(),
    enabledCompanyIds,
  };
}

export function getRepositoryRecordRuleBypassDepth(): number {
  const req = getRepositoryCurrentReq();
  const state = getOrInitRepositoryReqServiceState(req);
  const value = state?.recordRuleBypassDepth;
  return typeof value === 'number' && Number.isFinite(value) ? value : 0;
}

function restoreBypassDepth(state: RepositoryReqServiceState, key: 'recordRuleBypassDepth' | 'fieldRuleBypassDepth'): void {
  // Decrement rather than write back previousDepth so concurrent sibling bypasses
  // that share request service state survive out-of-order completion (aligned with withBypassDepths).
  const current = typeof state[key] === 'number' && Number.isFinite(state[key]) ? (state[key] as number) : 0;
  if (current > 1) state[key] = current - 1;
  else delete state[key];
}

function isPromiseLike<T = unknown>(value: unknown): value is Promise<T> {
  return !!value && typeof (value as { then?: unknown }).then === 'function';
}

function runWithBypassRestore<T>(fn: () => T, restore: () => void): T {
  try {
    const result = fn();
    if (isPromiseLike(result)) {
      return Promise.resolve(result).finally(restore) as unknown as T;
    }
    restore();
    return result;
  } catch (error) {
    restore();
    throw error;
  }
}

/**
 * Increments RecordRule bypass depth for the duration of `fn`.
 * Sync and async `fn` are both supported (aligned with withContext / Model.sudo).
 */
export function withRepositoryRecordRuleBypass<T>(fn: () => T): T {
  const req = getRepositoryCurrentReq();
  const state = getOrInitRepositoryReqServiceState(req);
  if (!state) return fn();

  const previousDepth = getRepositoryRecordRuleBypassDepth();
  state.recordRuleBypassDepth = previousDepth + 1;
  return runWithBypassRestore(fn, () => restoreBypassDepth(state, 'recordRuleBypassDepth'));
}

export function getRepositoryFieldRuleBypassDepth(): number {
  const req = getRepositoryCurrentReq();
  const state = getOrInitRepositoryReqServiceState(req);
  const value = state?.fieldRuleBypassDepth;
  return typeof value === 'number' && Number.isFinite(value) ? value : 0;
}

/**
 * Increments FieldRule bypass depth for the duration of `fn`.
 * Sync and async `fn` are both supported (aligned with withContext / Model.sudo).
 */
export function withRepositoryFieldRuleBypass<T>(fn: () => T): T {
  const req = getRepositoryCurrentReq();
  const state = getOrInitRepositoryReqServiceState(req);
  if (!state) return fn();

  const previousDepth = getRepositoryFieldRuleBypassDepth();
  state.fieldRuleBypassDepth = previousDepth + 1;
  return runWithBypassRestore(fn, () => restoreBypassDepth(state, 'fieldRuleBypassDepth'));
}

/**
 * Elevates for the duration of `fn` by bypassing both RecordRule and FieldRule.
 * Company scope remains in effect. Sync and async `fn` are both supported.
 */
export function withRepositoryAuthzRuleBypass<T>(fn: () => T): T {
  const req = getRepositoryCurrentReq();
  const state = getOrInitRepositoryReqServiceState(req);
  if (!state) return fn();

  const previousRecordDepth = getRepositoryRecordRuleBypassDepth();
  const previousFieldDepth = getRepositoryFieldRuleBypassDepth();
  state.recordRuleBypassDepth = previousRecordDepth + 1;
  state.fieldRuleBypassDepth = previousFieldDepth + 1;

  return runWithBypassRestore(fn, () => {
    restoreBypassDepth(state, 'recordRuleBypassDepth');
    restoreBypassDepth(state, 'fieldRuleBypassDepth');
  });
}

export function getRepositoryValidationBypassState(): RepositoryReqServiceState {
  const req = getRepositoryCurrentReq();
  const reqState = getOrInitRepositoryReqServiceState(req);
  if (reqState) return reqState;

  const key = '__choysumRepositoryServiceState';
  const root = globalThis as unknown as ObjectRecord;
  const existing = asReqServiceState(root[key]);
  if (existing) return existing;

  const created: RepositoryReqServiceState = {};
  root[key] = created;
  return created;
}

export function getRepositoryValidationBypassDepth(): number {
  const state = getRepositoryValidationBypassState();
  const value = state?.validationBypassDepth;
  return typeof value === 'number' && Number.isFinite(value) ? value : 0;
}

export async function withRepositoryValidationBypass<T>(fn: () => Promise<T>): Promise<T> {
  const state = getRepositoryValidationBypassState();
  const previousDepth = getRepositoryValidationBypassDepth();
  state.validationBypassDepth = previousDepth + 1;
  try {
    return await fn();
  } finally {
    const nextDepth = previousDepth;
    if (nextDepth > 0) state.validationBypassDepth = nextDepth;
    else delete state.validationBypassDepth;
  }
}
