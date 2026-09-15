// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getServiceFactory, registerServiceFactory, unregisterServiceFactory } from '@/core/service/rpc';
import {
  OWNER_AUTHZ_SERVICE,
  type OwnerAuthzService,
} from '@/core/service/api/owner_authz';
import type { ConditionEnvelope, FieldRuleSpec, RecordRuleOp } from '@/core/service/api/authz';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');

/** Owner record ids commonly used by document unit suites (probe Search stub). */
const KNOWN_OWNER_RECORD_IDS = new Set(['usr_document_test', 'usr_scope_a', 'usr_scope_b']);

/** Opaque owner model string used by document attachment fixtures (not a product model). */
export const DOCUMENT_TEST_OWNER_MODEL = 'auth.User';

let previousOwnerAuthzFactory: ReturnType<typeof getServiceFactory> | undefined;
let previousOwnerProbeFactory: ReturnType<typeof getServiceFactory> | undefined;
let ownerAuthzStubInstalled = false;
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
 * {@link OWNER_AUTHZ_SERVICE} directly (unaffected by this flag).
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
 * Owner authorization dials {@link OWNER_AUTHZ_SERVICE} directly (unaffected by this flag).
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

function isUnknownOwnerModel(model: string): boolean {
  const text = String(model || '').trim();
  if (!text.includes('.')) return true;
  return text.toLowerCase().startsWith('unknown.');
}

function createAllowAllOwnerAuthz(): OwnerAuthzService {
  return {
    GetRecordRuleCondition: async (model: string, _op: RecordRuleOp): Promise<ConditionEnvelope> => {
      if (isUnknownOwnerModel(model)) {
        return { kind: 'false', reason: 'unknown_model' };
      }
      return { kind: 'true', reason: 'document_test_allow' };
    },
    GetFieldRuleSpec: async (_model: string): Promise<FieldRuleSpec> => ({
      denyReadFields: [],
      denyWriteFields: [],
      reason: 'document_test_allow',
    }),
  };
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

function createOwnerProbeSearchStub(): { Search: (condition: unknown, options?: unknown) => Promise<unknown[]> } {
  return {
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
 * Install allow-all {@link OWNER_AUTHZ_SERVICE} plus a minimal owner-model Search stub
 * so document unit suites do not need auth soft-installed.
 */
export function ensureDocumentOwnerAuthzStub(): void {
  if (!ownerAuthzStubInstalled) {
    previousOwnerAuthzFactory = getServiceFactory(OWNER_AUTHZ_SERVICE);
    previousOwnerProbeFactory = getServiceFactory(DOCUMENT_TEST_OWNER_MODEL);
    ownerAuthzStubInstalled = true;
  }
  registerServiceFactory(OWNER_AUTHZ_SERVICE, () => createAllowAllOwnerAuthz());
  registerServiceFactory(DOCUMENT_TEST_OWNER_MODEL, () => createOwnerProbeSearchStub());
  clearRequestAuthzCaches();
}

/**
 * Temporarily override the owner-authz stub (e.g. field deny / expr deny cases).
 */
export async function withDocumentOwnerAuthzOverride<T>(
  override: Partial<OwnerAuthzService>,
  fn: () => Promise<T>
): Promise<T> {
  ensureDocumentOwnerAuthzStub();
  const base = createAllowAllOwnerAuthz();
  const merged: OwnerAuthzService = {
    GetRecordRuleCondition: override.GetRecordRuleCondition
      ? (model, op) => override.GetRecordRuleCondition!(model, op)
      : base.GetRecordRuleCondition,
    GetFieldRuleSpec: override.GetFieldRuleSpec
      ? model => override.GetFieldRuleSpec!(model)
      : base.GetFieldRuleSpec,
  };
  registerServiceFactory(OWNER_AUTHZ_SERVICE, () => merged);
  clearRequestAuthzCaches();
  try {
    return await fn();
  } finally {
    registerServiceFactory(OWNER_AUTHZ_SERVICE, () => createAllowAllOwnerAuthz());
    clearRequestAuthzCaches();
  }
}

/** Alias for {@link ensureDocumentOwnerAuthzStub} (call-site compatibility). */
export async function ensureAuthUserOwnerRecordRuleGrants(): Promise<void> {
  ensureDocumentOwnerAuthzStub();
}

/** Alias for {@link ensureDocumentOwnerAuthzStub} (call-site compatibility). */
export async function ensureAuthUserOwnerFieldRuleGrants(): Promise<void> {
  ensureDocumentOwnerAuthzStub();
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

  if (ownerAuthzStubInstalled) {
    restoreFactory(OWNER_AUTHZ_SERVICE, previousOwnerAuthzFactory);
    restoreFactory(DOCUMENT_TEST_OWNER_MODEL, previousOwnerProbeFactory);
    previousOwnerAuthzFactory = undefined;
    previousOwnerProbeFactory = undefined;
    ownerAuthzStubInstalled = false;
  }

  clearRequestAuthzCaches();
}
