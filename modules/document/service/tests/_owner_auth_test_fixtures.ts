// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getServiceFactory, registerServiceFactory, unregisterServiceFactory } from '@/core/service/rpc';
import type { ConditionEnvelope, FieldRuleSpec, RecordRuleOp } from '@/core/service/api/authz';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');

/** Owner record ids commonly used by document unit suites (probe Search stub). */
const KNOWN_OWNER_RECORD_IDS = new Set(['usr_document_test', 'usr_scope_a', 'usr_scope_b']);

/**
 * Dial target for owner RR/FR (and probe Search when ownerModel is auth.User).
 * Matches the auto-exposed auth.User gRPC surface; unit suites stub this factory
 * so document does not soft-install auth.
 */
export const DOCUMENT_TEST_AUTH_USER_MODEL = 'auth.User';

type AuthUserOwnerAuthzStub = {
  GetRecordRuleCondition(model: string, op: RecordRuleOp): Promise<ConditionEnvelope>;
  GetFieldRuleSpec(model: string): Promise<FieldRuleSpec>;
  Search(condition: unknown, options?: unknown): Promise<unknown[]>;
};

let previousAuthUserFactory: ReturnType<typeof getServiceFactory> | undefined;
let authUserStubInstalled = false;
let previousRecordRuleEnabled: unknown = undefined;
let capturedRecordRuleEnv = false;
let previousFieldRuleEnabled: unknown = undefined;
let capturedFieldRuleEnv = false;

function clearRequestAuthzCaches(): void {
  const root: any = (globalThis as any).$choysum ?? {};
  const jsCtx = root?.request?.context;
  if (jsCtx) {
    delete jsCtx[RR_CACHE_KEY];
    delete jsCtx[FR_CACHE_KEY];
  }
}

/**
 * Document unit tests historically relied on repository RecordRule allow-by-default
 * for UploadSession / Binding / Content CRUD. Under deny-default those writes need
 * either grant packs or a repository-layer disable. Owner authorization dials
 * auth.User GetRecordRuleCondition directly (unaffected by this flag).
 */
export function disableRepositoryRecordRuleForDocumentTests(): void {
  const root = globalThis as any;
  const prev = root.__CHOYSUM_RUNTIME_ENV__ && typeof root.__CHOYSUM_RUNTIME_ENV__ === 'object' ? { ...root.__CHOYSUM_RUNTIME_ENV__ } : {};
  if (!capturedRecordRuleEnv) {
    previousRecordRuleEnabled = prev.CHOYSUM_GRPC_RECORD_RULE_ENABLED;
    capturedRecordRuleEnv = true;
  }
  root.__CHOYSUM_RUNTIME_ENV__ = { ...prev, CHOYSUM_GRPC_RECORD_RULE_ENABLED: false };
}

/**
 * Mirror RecordRule disable for repository FieldRule: nested Create/Update at depth>0
 * ignores top-level fieldRuleMode=skip, so deny-default would block UploadSession rows.
 * Owner authorization dials auth.User GetFieldRuleSpec directly (unaffected by this flag).
 */
export function disableRepositoryFieldRuleForDocumentTests(): void {
  const root = globalThis as any;
  const prev = root.__CHOYSUM_RUNTIME_ENV__ && typeof root.__CHOYSUM_RUNTIME_ENV__ === 'object' ? { ...root.__CHOYSUM_RUNTIME_ENV__ } : {};
  if (!capturedFieldRuleEnv) {
    previousFieldRuleEnabled = prev.CHOYSUM_GRPC_FIELD_RULE_ENABLED;
    capturedFieldRuleEnv = true;
  }
  root.__CHOYSUM_RUNTIME_ENV__ = { ...prev, CHOYSUM_GRPC_FIELD_RULE_ENABLED: false };
}

function isGrantedOwnerModel(model: string): boolean {
  return String(model || '').trim().toLowerCase() === DOCUMENT_TEST_AUTH_USER_MODEL.toLowerCase();
}

function collectIdEquals(condition: unknown): string[] {
  if (Array.isArray(condition) && condition.length >= 3) {
    const field = String(condition[0] ?? '').trim();
    const op = String(condition[1] ?? '').trim();
    if (field === 'Id' && op === '=') {
      const id = String(condition[2] ?? '').trim();
      return id ? [id] : [];
    }
    return [];
  }
  if (condition && typeof condition === 'object' && !Array.isArray(condition)) {
    const and = (condition as { And?: unknown }).And;
    if (Array.isArray(and)) {
      return and.flatMap(item => collectIdEquals(item));
    }
  }
  return [];
}

/**
 * Default stub mirrors the old everyone grant on auth.User (read+write only).
 * Other owner models and create/delete stay denied; suites that need create
 * must use {@link withDocumentAuthUserStubOverride}.
 */
function createDefaultAuthUserStub(): AuthUserOwnerAuthzStub {
  return {
    GetRecordRuleCondition: async (model: string, op: RecordRuleOp): Promise<ConditionEnvelope> => {
      if (!isGrantedOwnerModel(model)) {
        return { kind: 'false', reason: 'unknown_model' };
      }
      if (op !== 'read' && op !== 'write') {
        return { kind: 'false', reason: 'document_test_op_denied' };
      }
      return { kind: 'true', reason: 'document_test_allow' };
    },
    GetFieldRuleSpec: async (_model: string): Promise<FieldRuleSpec> => ({
      denyReadFields: [],
      denyWriteFields: [],
      reason: 'document_test_allow',
    }),
    Search: async (condition: unknown) => {
      const ids = collectIdEquals(condition);
      if (ids.length === 0) return [];
      const first = ids[0];
      if (!ids.every(id => id === first)) return [];
      if (KNOWN_OWNER_RECORD_IDS.has(first)) return [{ Id: first }];
      return [];
    },
  };
}

function restoreFactory(modelName: string, previous: ReturnType<typeof getServiceFactory> | undefined): void {
  unregisterServiceFactory(modelName);
  if (previous) registerServiceFactory(modelName, previous);
}

/**
 * Install the default auth.User stub once per suite process so nested
 * {@link withDocumentAuthUserStubOverride} calls are not clobbered by re-ensure.
 */
export function ensureDocumentAuthUserStub(): void {
  if (authUserStubInstalled) {
    if (!getServiceFactory(DOCUMENT_TEST_AUTH_USER_MODEL)) {
      registerServiceFactory(DOCUMENT_TEST_AUTH_USER_MODEL, () => createDefaultAuthUserStub());
    }
    clearRequestAuthzCaches();
    return;
  }
  previousAuthUserFactory = getServiceFactory(DOCUMENT_TEST_AUTH_USER_MODEL);
  registerServiceFactory(DOCUMENT_TEST_AUTH_USER_MODEL, () => createDefaultAuthUserStub());
  authUserStubInstalled = true;
  clearRequestAuthzCaches();
}

/**
 * Temporarily override auth.User stub methods (e.g. field deny / expr deny / create grant).
 * Merges onto the currently registered factory so nested overrides compose; restores
 * that prior factory when the callback finishes.
 */
export async function withDocumentAuthUserStubOverride<T>(
  override: Partial<Pick<AuthUserOwnerAuthzStub, 'GetRecordRuleCondition' | 'GetFieldRuleSpec' | 'Search'>>,
  fn: () => Promise<T>
): Promise<T> {
  ensureDocumentAuthUserStub();
  const priorFactory = getServiceFactory(DOCUMENT_TEST_AUTH_USER_MODEL);
  const base = (priorFactory ? priorFactory() : createDefaultAuthUserStub()) as AuthUserOwnerAuthzStub;
  const merged: AuthUserOwnerAuthzStub = {
    GetRecordRuleCondition: override.GetRecordRuleCondition
      ? (model, op) => override.GetRecordRuleCondition!(model, op)
      : base.GetRecordRuleCondition,
    GetFieldRuleSpec: override.GetFieldRuleSpec
      ? model => override.GetFieldRuleSpec!(model)
      : base.GetFieldRuleSpec,
    Search: override.Search
      ? (condition, options) => override.Search!(condition, options)
      : base.Search,
  };
  registerServiceFactory(DOCUMENT_TEST_AUTH_USER_MODEL, () => merged);
  clearRequestAuthzCaches();
  try {
    return await fn();
  } finally {
    restoreFactory(DOCUMENT_TEST_AUTH_USER_MODEL, priorFactory);
    clearRequestAuthzCaches();
  }
}

/**
 * Restore process env mutated by document fixtures and drop suite-owned stubs.
 */
export async function restoreDocumentOwnerAuthFixtures(): Promise<void> {
  if (capturedRecordRuleEnv) {
    const root = globalThis as any;
    const prev = root.__CHOYSUM_RUNTIME_ENV__ && typeof root.__CHOYSUM_RUNTIME_ENV__ === 'object' ? { ...root.__CHOYSUM_RUNTIME_ENV__ } : {};
    if (previousRecordRuleEnabled === undefined) {
      delete prev.CHOYSUM_GRPC_RECORD_RULE_ENABLED;
    } else {
      prev.CHOYSUM_GRPC_RECORD_RULE_ENABLED = previousRecordRuleEnabled;
    }
    root.__CHOYSUM_RUNTIME_ENV__ = prev;
    capturedRecordRuleEnv = false;
    previousRecordRuleEnabled = undefined;
  }

  if (capturedFieldRuleEnv) {
    const root = globalThis as any;
    const prev = root.__CHOYSUM_RUNTIME_ENV__ && typeof root.__CHOYSUM_RUNTIME_ENV__ === 'object' ? { ...root.__CHOYSUM_RUNTIME_ENV__ } : {};
    if (previousFieldRuleEnabled === undefined) {
      delete prev.CHOYSUM_GRPC_FIELD_RULE_ENABLED;
    } else {
      prev.CHOYSUM_GRPC_FIELD_RULE_ENABLED = previousFieldRuleEnabled;
    }
    root.__CHOYSUM_RUNTIME_ENV__ = prev;
    capturedFieldRuleEnv = false;
    previousFieldRuleEnabled = undefined;
  }

  if (authUserStubInstalled) {
    restoreFactory(DOCUMENT_TEST_AUTH_USER_MODEL, previousAuthUserFactory);
    previousAuthUserFactory = undefined;
    authUserStubInstalled = false;
  }

  clearRequestAuthzCaches();
}
